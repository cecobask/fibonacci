package server_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/cecobask/fibonacci/internal/api"
	"github.com/cecobask/fibonacci/internal/server"
)

type testEnv struct {
	srv *server.Server
	url string
}

func newTestEnv(t *testing.T) testEnv {
	t.Helper()

	srv := server.New("", slog.New(slog.DiscardHandler))
	srv.SetReady(true)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return testEnv{srv: srv, url: ts.URL}
}

type response struct {
	status int
	header http.Header
	body   []byte
}

func do(t *testing.T, method, url string, header http.Header) response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), method, url, http.NoBody)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	for k, v := range header {
		req.Header[k] = v
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	return response{status: resp.StatusCode, header: resp.Header, body: body}
}

func get(t *testing.T, url string) response {
	t.Helper()
	return do(t, http.MethodGet, url, nil)
}

func TestGetFibonacci(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)

	tests := []struct {
		name  string
		query string
		want  []int64
	}{
		{name: "zero", query: "n=0", want: []int64{}},
		{name: "one", query: "n=1", want: []int64{0}},
		{name: "five", query: "n=5", want: []int64{0, 1, 1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := get(t, env.url+"/v1/fibonacci?"+tt.query)
			if resp.status != http.StatusOK {
				t.Fatalf("status = %d, want %d", resp.status, http.StatusOK)
			}
			if ct := resp.header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			var got []int64
			if err := json.Unmarshal(resp.body, &got); err != nil {
				t.Fatalf("decoding body %q: %v", resp.body, err)
			}
			if got == nil {
				t.Fatalf("body = %s, want a JSON array", resp.body)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("body = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFibonacciMaxN(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)

	resp := get(t, env.url+"/v1/fibonacci?n=93")
	if resp.status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.status, http.StatusOK)
	}
	// Check the raw JSON to make sure large values are not rounded.
	if !strings.HasSuffix(strings.TrimSpace(string(resp.body)), "7540113804746346429]") {
		t.Errorf("body does not end with F(92): %s", resp.body)
	}
}

func TestGetFibonacciBadRequest(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)

	tests := []struct {
		name       string
		query      string
		wantDetail string
	}{
		{name: "negative", query: "n=-1", wantDetail: "n must not be negative"},
		{name: "too large", query: "n=94", wantDetail: "n must not be greater than 93"},
		{name: "not a number", query: "n=abc", wantDetail: `query parameter "n" has an invalid format`},
		{name: "overflow", query: "n=99999999999999999999", wantDetail: `query parameter "n" has an invalid format`},
		{name: "missing", query: "", wantDetail: `query parameter "n" is required`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := get(t, env.url+"/v1/fibonacci?"+tt.query)
			if resp.status != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", resp.status, http.StatusBadRequest)
			}
			if ct := resp.header.Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("Content-Type = %q, want application/problem+json", ct)
			}

			var p api.Problem
			if err := json.Unmarshal(resp.body, &p); err != nil {
				t.Fatalf("decoding problem: %v", err)
			}
			if p.Status != http.StatusBadRequest || p.Title != "Bad Request" || p.Type != "about:blank" {
				t.Errorf("problem = %+v, want status 400, title Bad Request, type about:blank", p)
			}
			if p.Detail == nil || *p.Detail != tt.wantDetail {
				t.Errorf("detail = %v, want %q", p.Detail, tt.wantDetail)
			}
		})
	}
}

func TestMethodNotAllowed(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)

	resp := do(t, http.MethodPost, env.url+"/v1/fibonacci?n=5", nil)
	if resp.status != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", resp.status, http.StatusMethodNotAllowed)
	}
}

func TestProbes(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)

	if resp := get(t, env.url+"/healthz"); resp.status != http.StatusOK {
		t.Errorf("healthz status = %d, want %d", resp.status, http.StatusOK)
	}
	if resp := get(t, env.url+"/readyz"); resp.status != http.StatusOK {
		t.Errorf("readyz status = %d, want %d", resp.status, http.StatusOK)
	}

	env.srv.SetReady(false)

	if resp := get(t, env.url+"/readyz"); resp.status != http.StatusServiceUnavailable {
		t.Errorf("readyz status after SetReady(false) = %d, want %d", resp.status, http.StatusServiceUnavailable)
	}
	if resp := get(t, env.url+"/healthz"); resp.status != http.StatusOK {
		t.Errorf("healthz status after SetReady(false) = %d, want %d", resp.status, http.StatusOK)
	}
}

func TestRequestID(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)

	t.Run("generated", func(t *testing.T) {
		t.Parallel()

		resp := get(t, env.url+"/healthz")
		if id := resp.header.Get("X-Request-ID"); len(id) != 32 {
			t.Errorf("X-Request-ID = %q, want a 32 character ID", id)
		}
	})

	t.Run("propagated", func(t *testing.T) {
		t.Parallel()

		resp := do(t, http.MethodGet, env.url+"/healthz", http.Header{"X-Request-Id": {"abc-123"}})
		if id := resp.header.Get("X-Request-ID"); id != "abc-123" {
			t.Errorf("X-Request-ID = %q, want abc-123", id)
		}
	})
}
