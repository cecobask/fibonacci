package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer is a bytes.Buffer that is safe for concurrent use.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func freePort(t *testing.T) int {
	t.Helper()

	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("finding free port: %v", err)
	}
	defer func() { _ = ln.Close() }()
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected address type %T", ln.Addr())
	}
	return addr.Port
}

// client does not reuse connections, so that no idle keep-alive connection
// is left behind to delay shutdown.
var client = &http.Client{
	Transport: &http.Transport{DisableKeepAlives: true},
	Timeout:   5 * time.Second,
}

func waitReady(t *testing.T, baseURL string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, baseURL+"/readyz", http.NoBody)
		if err != nil {
			t.Fatalf("creating request: %v", err)
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("server did not become ready in time")
}

func TestRun(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	env := map[string]string{
		"PORT":             strconv.Itoa(port),
		"LOG_LEVEL":        "debug",
		"SHUTDOWN_TIMEOUT": "5s",
	}
	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var logs syncBuffer
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, func(k string) string { return env[k] }, &logs)
	}()

	waitReady(t, baseURL)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, baseURL+"/v1/fibonacci?n=5", http.NoBody)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if got := strings.TrimSpace(string(body)); got != "[0,1,1,2,3]" {
		t.Errorf("body = %s, want [0,1,1,2,3]", got)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run() error = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run() did not return after cancellation")
	}

	for _, msg := range []string{"server started", "shutdown started", "shutdown complete"} {
		if !strings.Contains(logs.String(), msg) {
			t.Errorf("logs do not contain %q:\n%s", msg, logs.String())
		}
	}
}

func TestRunInvalidConfig(t *testing.T) {
	t.Parallel()

	env := map[string]string{"PORT": "invalid"}
	err := run(t.Context(), func(k string) string { return env[k] }, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "PORT") {
		t.Errorf("run() error = %v, want a PORT config error", err)
	}
}
