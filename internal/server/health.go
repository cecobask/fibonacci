package server

import (
	"net/http"
)

// handleHealthz is the liveness probe. It succeeds as long as the process is
// able to serve HTTP requests.
func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeStatus(w, http.StatusOK, "ok")
}

// handleReadyz is the readiness probe. It fails before the server is marked
// ready and once shutdown has started, so that no new traffic is routed here.
func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	if !s.ready.Load() {
		writeStatus(w, http.StatusServiceUnavailable, "not ready")
		return
	}
	writeStatus(w, http.StatusOK, "ok")
}

func writeStatus(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body + "\n"))
}
