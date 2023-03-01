/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package output

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AddHttpOutputOptions struct {
	Config     *config.Config
	Method     string
	URL        url.URL
	OutputPath string
	UseHttps   bool
	FormatJson bool
}

type HttpOptions struct {
	Config     *config.Config
	Method     string
	URL        url.URL
	UseHttps   bool
	FormatJson bool
}

func AddHttpOutput(opts AddHttpOutputOptions) {
	start := time.Now()
	outputPath := opts.OutputPath

	content := fmt.Sprintf("# Executed command: %s %s", opts.Method, opts.URL.String())
	content = fmt.Sprintf("%s\n# Started at: %s", content, time.Now().Format(time.RFC3339))

	out, err := DoHttp(HttpOptions{
		Config:     opts.Config,
		Method:     opts.Method,
		URL:        opts.URL,
		UseHttps:   opts.UseHttps,
		FormatJson: opts.FormatJson,
	})
	if err != nil {
		content = fmt.Sprintf("%s\n# Resulted in error: %s", content, err)
	}
	if strings.Contains(string(out), "Client sent an HTTP request to an HTTPS server") {
		opts.UseHttps = true
		out, err = DoHttp(HttpOptions{
			Config:     opts.Config,
			Method:     opts.Method,
			URL:        opts.URL,
			UseHttps:   opts.UseHttps,
			FormatJson: opts.FormatJson,
		})
		if err != nil {
			content = fmt.Sprintf("%s\n# Resulted in error: %s", content, err)
		}
	}
	content = fmt.Sprintf("%s\n\n%s", content, out)

	totalTime := time.Now().Sub(start)
	content = fmt.Sprintf("%s\n\n# Total execution time: %d millis", content, totalTime.Milliseconds())

	WriteToFile(outputPath, []byte(strings.TrimSpace(content)))
}

func DoHttp(options HttpOptions) ([]byte, error) {
	body, err := doHttp(options)
	if err != nil {
		return nil, err
	}
	if strings.Contains(string(body), "Client sent an HTTP request to an HTTPS server") {
		options.UseHttps = true
		body, err = doHttp(options)
		if err != nil {
			return nil, err
		}
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

func doHttp(options HttpOptions) ([]byte, error) {
	var tr *http.Transport
	if options.UseHttps {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		options.URL.Scheme = "https"
	} else {
		tr = &http.Transport{}
	}
	client := &http.Client{Transport: tr}
	var req = &http.Request{
		Method: options.Method,
		URL:    &options.URL,
	}
	response, err := client.Do(req)
	defer closeResponse(response)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func closeResponse(response *http.Response) {
	func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error().Msgf("Failed to close response body")
			return
		}
	}(response.Body)
}
