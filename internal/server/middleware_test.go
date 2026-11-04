package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cecobask/fibonacci/internal/api"
)

func TestRecoverer(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	h := recoverer(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	var p api.Problem
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("decoding problem: %v", err)
	}
	if p.Status != http.StatusInternalServerError || p.Detail != nil {
		t.Errorf("problem = %+v, want status 500 without detail", p)
	}
	if !strings.Contains(logs.String(), "panic recovered") {
		t.Errorf("logs = %q, want a panic record", logs.String())
	}
}

func TestValidRequestID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   string
		want bool
	}{
		{id: "abc-123", want: true},
		{id: "", want: false},
		{id: "has space", want: false},
		{id: "new\nline", want: false},
		{id: strings.Repeat("a", maxRequestIDLength), want: true},
		{id: strings.Repeat("a", maxRequestIDLength+1), want: false},
	}

	for _, tt := range tests {
		if got := validRequestID(tt.id); got != tt.want {
			t.Errorf("validRequestID(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

func TestAccessLog(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	h := requestID(accessLog(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})))

	req := httptest.NewRequest(http.MethodGet, "/v1/fibonacci?n=1", nil)
	req.Header.Set(requestIDHeader, "req-1")
	h.ServeHTTP(httptest.NewRecorder(), req)

	var record map[string]any
	if err := json.Unmarshal(logs.Bytes(), &record); err != nil {
		t.Fatalf("decoding log record %q: %v", logs.String(), err)
	}
	if record["status"] != float64(http.StatusTeapot) {
		t.Errorf("status = %v, want %d", record["status"], http.StatusTeapot)
	}
	if record["request_id"] != "req-1" {
		t.Errorf("request_id = %v, want req-1", record["request_id"])
	}
	if record["path"] != "/v1/fibonacci" {
		t.Errorf("path = %v, want /v1/fibonacci", record["path"])
	}
}
