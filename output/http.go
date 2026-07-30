/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package output

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/limit"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

const httpsRequiredIndicator = "Client sent an HTTP request to an HTTPS server"

// maxBytesForJsonFormatting bounds the memory needed to pretty print a response. Larger responses - a discovery
// on a big cluster easily returns hundreds of megabytes - are streamed to disk unformatted.
const maxBytesForJsonFormatting = 1024 * 1024

// maxBytesForInMemoryResponse bounds responses that callers want to parse instead of writing them to a file.
const maxBytesForInMemoryResponse = 64 * 1024 * 1024

// stallTimeout aborts a request that makes no progress, so that a dead port-forward cannot occupy an execution
// slot for the rest of the run. It is not a limit on the total duration: every chunk of the response that
// arrives grants the next stallTimeout, which leaves a slow but progressing discovery alone. A variable so that
// the tests do not have to wait for it.
var stallTimeout = 5 * time.Minute

// errHttpsRequired is returned when the server answered that it expects the request via HTTPS.
var errHttpsRequired = errors.New("server sent an HTTPS response to an HTTP request")

type AddHttpOutputOptions struct {
	Config           *config.Config
	Method           string
	URL              url.URL
	OutputPath       string
	UseHttps         bool
	FormatJson       bool
	ExecutionContext string
	LogError         bool
}

type HttpOptions struct {
	Config     *config.Config
	Method     string
	URL        url.URL
	UseHttps   bool
	FormatJson bool
}

func AddHttpOutput(opts AddHttpOutputOptions) {
	release := limit.Commands.Acquire()
	defer release()

	command := fmt.Sprintf("%s %s", opts.Method, opts.URL.String())

	addOutputFile(opts.OutputPath, command, nil, func(out *os.File) error {
		err := streamHttp(HttpOptions{
			Config:     opts.Config,
			Method:     opts.Method,
			URL:        opts.URL,
			UseHttps:   opts.UseHttps,
			FormatJson: opts.FormatJson,
		}, out)
		if err != nil {
			event := log.Debug()
			if opts.LogError {
				event = log.Error()
			}
			event.Str("context", opts.ExecutionContext).Str("cmd", command).Msgf("Error executing command")
		}
		return err
	})
}

// streamHttp copies the response body to dst without keeping all of it in memory, retrying via HTTPS when the
// server answers that it expects TLS.
func streamHttp(options HttpOptions, dst io.Writer) error {
	err := streamHttpOnce(options, dst)
	if errors.Is(err, errHttpsRequired) {
		options.UseHttps = true
		err = streamHttpOnce(options, dst)
	}
	return err
}

func streamHttpOnce(options HttpOptions, dst io.Writer) error {
	return doHttpRequest(options, func(responseBody io.Reader) error {
		// only the beginning of the response is read into memory, just enough to recognize the HTTPS hint and to
		// pretty print responses of a reasonable size
		body, err := io.ReadAll(io.LimitReader(responseBody, maxBytesForJsonFormatting))
		if err != nil {
			return err
		}

		if indicatesHttpsRequired(options, body) {
			return errHttpsRequired
		}

		if options.FormatJson {
			if len(body) < maxBytesForJsonFormatting {
				var formatted bytes.Buffer
				if json.Indent(&formatted, body, "", "\t") == nil {
					body = formatted.Bytes()
				}
			} else {
				_, _ = fmt.Fprintf(dst, "# Response is larger than %d bytes and therefore not JSON formatted\n\n", maxBytesForJsonFormatting)
			}
		}

		if _, err = dst.Write(body); err != nil {
			return err
		}
		_, err = io.Copy(dst, responseBody)
		return err
	})
}

// DoHttp reads the response into memory for callers that need to parse it. Use AddHttpOutput for responses that
// only end up in a file.
//
// Unlike AddHttpOutput this takes no limit.Commands slot, and it must not: its callers crawl an extension while
// holding the port-forward the request goes through, and AddHttpOutput already holds a slot when it reaches
// doHttpRequest - acquiring one further down would nest the semaphore under itself and deadlock as soon as every
// slot is held by a caller waiting for the inner one. Concurrency here is bounded one level up, by limit.Items.
func DoHttp(options HttpOptions) ([]byte, error) {
	body, err := doHttpBody(options)
	if errors.Is(err, errHttpsRequired) {
		options.UseHttps = true
		body, err = doHttpBody(options)
	}
	if err != nil {
		return nil, err
	}
	if options.FormatJson {
		var prettyJSON bytes.Buffer
		err := json.Indent(&prettyJSON, body, "", "\t")
		if err != nil {
			return nil, err
		}
		return prettyJSON.Bytes(), nil
	}
	return body, nil
}

func doHttpBody(options HttpOptions) ([]byte, error) {
	var body []byte

	err := doHttpRequest(options, func(responseBody io.Reader) error {
		// one byte beyond the limit tells a response that just fits apart from one that has to be rejected -
		// handing a truncated body to a caller that parses it would look like an empty extension instead of an
		// error
		read, err := io.ReadAll(io.LimitReader(responseBody, maxBytesForInMemoryResponse+1))
		if err != nil {
			return err
		}
		if indicatesHttpsRequired(options, read) {
			return errHttpsRequired
		}
		if len(read) > maxBytesForInMemoryResponse {
			return fmt.Errorf("response exceeds %d bytes", maxBytesForInMemoryResponse)
		}
		body = read
		return nil
	})
	if err != nil {
		return nil, err
	}
	return body, nil
}

// indicatesHttpsRequired reports whether the server answered a plaintext request by saying that it expects TLS.
func indicatesHttpsRequired(options HttpOptions, body []byte) bool {
	return !options.UseHttps && bytes.Contains(body, []byte(httpsRequiredIndicator))
}

// doHttpRequest executes the request and lets consume read the response body. The body is only valid until
// consume returns, which is what allows the request to be cancelled as soon as it stops making progress.
func doHttpRequest(options HttpOptions, consume func(responseBody io.Reader) error) error {
	// a transport per request, because the port-forward it talks to is closed right afterwards - keep-alives
	// would only leave connections behind that are already dead when the next request looks for them
	tr := &http.Transport{DisableKeepAlives: true}
	if options.UseHttps {
		options.URL.Scheme = "https"
		tlsConfig, err := tlsConfig(options.Config)
		if err != nil {
			return err
		}
		tr.TLSClientConfig = tlsConfig
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stall := time.AfterFunc(stallTimeout, cancel)
	defer stall.Stop()

	client := &http.Client{Transport: tr}
	var req = &http.Request{
		Method: options.Method,
		URL:    &options.URL,
	}
	response, err := client.Do(req.WithContext(ctx))
	if err != nil {
		log.Debug().Err(err).Msgf("Failed to execute request")
		return err
	}
	defer closeResponse(response)

	body := &progressReader{reader: response.Body, stall: stall}

	if response.StatusCode != http.StatusOK {
		// a Go server answers a plaintext request to its TLS port with '400 Bad Request' and the indicator in the
		// body, so the body has to be looked at before the status is turned into an error
		if hint, _ := io.ReadAll(io.LimitReader(body, 4*1024)); indicatesHttpsRequired(options, hint) {
			return errHttpsRequired
		}
		return fmt.Errorf("request failed with status code %d", response.StatusCode)
	}

	return consume(body)
}

// progressReader keeps the stall timer of a request alive as long as the response keeps arriving.
type progressReader struct {
	reader io.Reader
	stall  *time.Timer
}

func (r *progressReader) Read(p []byte) (int, error) {
	read, err := r.reader.Read(p)
	if read > 0 {
		r.stall.Reset(stallTimeout)
	}
	return read, err
}

var (
	tlsConfigMutex  sync.Mutex
	cachedTlsConfig *tls.Config
)

// tlsConfig returns the shared client configuration. It is built once: the certificates do not change while the
// tool runs, so reading and parsing them per request would only waste work and prevent TLS session reuse.
func tlsConfig(cfg *config.Config) (*tls.Config, error) {
	tlsConfigMutex.Lock()
	defer tlsConfigMutex.Unlock()

	if cachedTlsConfig != nil {
		return cachedTlsConfig, nil
	}

	if cfg.Tls.CertChainFile == "" || cfg.Tls.CertKeyFile == "" {
		cachedTlsConfig = &tls.Config{InsecureSkipVerify: true}
		return cachedTlsConfig, nil
	}

	cert, err := os.ReadFile(cfg.Tls.CertChainFile)
	if err != nil {
		log.Err(err).Msgf("Failed to read certificate")
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(cert)

	certificate, err := tls.LoadX509KeyPair(cfg.Tls.CertChainFile, cfg.Tls.CertKeyFile)
	if err != nil {
		log.Err(err).Msgf("Failed to load certificate")
		return nil, err
	}

	cachedTlsConfig = &tls.Config{
		RootCAs:            caCertPool,
		Certificates:       []tls.Certificate{certificate},
		InsecureSkipVerify: true,
	}
	return cachedTlsConfig, nil
}

func closeResponse(response *http.Response) {
	if response == nil {
		return
	}
	err := response.Body.Close()
	if err != nil {
		log.Debug().Msgf("Failed to close response body")
	}
}
