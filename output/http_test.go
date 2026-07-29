/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package output

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steadybit/steadybit-debug/config"
)

// TestStreamHttpKeepsReadingWhileTheResponseMakesProgress pins the semantics of stallTimeout: it is not a limit
// on the total duration, so a response that keeps arriving in chunks has to survive even when it takes longer
// than the timeout in total. The body crosses maxBytesForJsonFormatting to cover the streaming part after the
// pre-read as well.
func TestStreamHttpKeepsReadingWhileTheResponseMakesProgress(t *testing.T) {
	withStallTimeout(t, 150*time.Millisecond)

	chunk := bytes.Repeat([]byte("a"), maxBytesForJsonFormatting/2)
	const chunks = 4
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for i := 0; i < chunks; i++ {
			_, _ = w.Write(chunk)
			w.(http.Flusher).Flush()
			time.Sleep(50 * time.Millisecond)
		}
	}))
	defer server.Close()

	var written bytes.Buffer
	start := time.Now()
	err := streamHttpOnce(HttpOptions{
		Config: &config.Config{},
		Method: "GET",
		URL:    *mustParse(t, server.URL),
	}, &written)

	if err != nil {
		t.Fatalf("expected the response to be read completely, got: %s", err)
	}
	if written.Len() != chunks*len(chunk) {
		t.Errorf("expected %d bytes, got %d", chunks*len(chunk), written.Len())
	}
	if elapsed := time.Since(start); elapsed < stallTimeout {
		t.Errorf("the response arrived within %s, which is faster than the stall timeout - the test proves nothing", elapsed)
	}
}

func TestStreamHttpCancelsAStalledResponse(t *testing.T) {
	withStallTimeout(t, 100*time.Millisecond)

	// the handler stalls until the test is over, so it outlives the timeout no matter how slow the machine is
	stalling := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("everything up to here arrived"))
		w.(http.Flusher).Flush()
		<-stalling
	}))
	// releasing the handler has to happen before Close, which waits for outstanding requests
	defer server.Close()
	defer close(stalling)

	var written bytes.Buffer
	start := time.Now()
	err := streamHttpOnce(HttpOptions{
		Config: &config.Config{},
		Method: "GET",
		URL:    *mustParse(t, server.URL),
	}, &written)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected the stalled response to be reported as an error")
	}
	if elapsed > time.Second {
		t.Errorf("expected the stalled response to be given up on after %s, took %s", stallTimeout, elapsed)
	}
}

func withStallTimeout(t *testing.T, timeout time.Duration) {
	t.Helper()
	previous := stallTimeout
	stallTimeout = timeout
	t.Cleanup(func() {
		stallTimeout = previous
	})
}

func TestAddHttpOutputFormatsJson(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"a":1}`))
	}))
	defer server.Close()
	outputPath := filepath.Join(t.TempDir(), "out.yml")

	AddHttpOutput(AddHttpOutputOptions{
		Config:     &config.Config{},
		Method:     "GET",
		URL:        *mustParse(t, server.URL),
		OutputPath: outputPath,
		FormatJson: true,
	})

	content := readFile(t, outputPath)
	if !strings.Contains(content, "{\n\t\"a\": 1\n}") {
		t.Errorf("expected formatted json, got:\n%s", content)
	}
	if !strings.Contains(content, "# Total execution time: ") {
		t.Errorf("expected the footer to be written, got:\n%s", content)
	}
}

func TestAddHttpOutputReportsFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	outputPath := filepath.Join(t.TempDir(), "out.yml")

	AddHttpOutput(AddHttpOutputOptions{
		Config:     &config.Config{},
		Method:     "GET",
		URL:        *mustParse(t, server.URL),
		OutputPath: outputPath,
	})

	content := readFile(t, outputPath)
	if !strings.Contains(content, "# Resulted in error: request failed with status code 500") {
		t.Errorf("expected the status code to be reported, got:\n%s", content)
	}
}

// TestAddHttpOutputStreamsLargeResponses guards the fix for the tool exhausting the machines memory: a large
// discovery response must be written to disk instead of being formatted and copied in memory.
func TestAddHttpOutputStreamsLargeResponses(t *testing.T) {
	body := bytes.Repeat([]byte("a"), maxBytesForJsonFormatting+1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()
	outputPath := filepath.Join(t.TempDir(), "out.yml")

	AddHttpOutput(AddHttpOutputOptions{
		Config:     &config.Config{},
		Method:     "GET",
		URL:        *mustParse(t, server.URL),
		OutputPath: outputPath,
		FormatJson: true,
	})

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("expected output file: %s", err)
	}
	if info.Size() < int64(len(body)) {
		t.Errorf("expected the complete response of %d bytes to be written, got %d", len(body), info.Size())
	}
	content := readFile(t, outputPath)
	if !strings.Contains(content, fmt.Sprintf("# Response is larger than %d bytes and therefore not JSON formatted", maxBytesForJsonFormatting)) {
		t.Errorf("expected a hint about the skipped formatting, got:\n%s", content[:200])
	}
}

// TestStreamHttpRetriesViaHttpsWhenTheServerExpectsTls talks plain HTTP to a TLS server, which answers with the
// '400 Bad Request' plus indicator that the retry has to recognize - a 200 would not exercise the status handling.
func TestStreamHttpRetriesViaHttpsWhenTheServerExpectsTls(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"served":"via tls"}`))
	}))
	defer server.Close()

	plaintextUrl := *mustParse(t, server.URL)
	plaintextUrl.Scheme = "http"

	var written bytes.Buffer
	err := streamHttpOnce(HttpOptions{
		Config: &config.Config{},
		Method: "GET",
		URL:    plaintextUrl,
	}, &written)

	if !errors.Is(err, errHttpsRequired) {
		t.Fatalf("expected a retry via https to be requested, got: %v", err)
	}
	if written.Len() > 0 {
		t.Errorf("expected nothing to be written before the retry, got: %s", written.String())
	}

	written.Reset()
	if err := streamHttp(HttpOptions{
		Config: &config.Config{},
		Method: "GET",
		URL:    plaintextUrl,
	}, &written); err != nil {
		t.Fatalf("expected the retry to succeed, got: %s", err)
	}
	if !strings.Contains(written.String(), "via tls") {
		t.Errorf("expected the response of the https retry, got: %s", written.String())
	}
}

func mustParse(t *testing.T, rawUrl string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(rawUrl)
	if err != nil {
		t.Fatalf("failed to parse '%s': %s", rawUrl, err)
	}
	return parsed
}
