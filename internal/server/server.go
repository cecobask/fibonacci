// Package server wires the generated API handlers, operational endpoints and
// middleware into an http.Server.
package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/cecobask/fibonacci/internal/api"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 120 * time.Second
	maxHeaderBytes    = 64 << 10
)

// Server is the HTTP server of the service.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
	ready      atomic.Bool
}

// New returns a Server listening on addr. The server starts in the not-ready
// state; call SetReady once it is able to accept traffic.
func New(addr string, logger *slog.Logger) *Server {
	s := &Server{logger: logger}
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           s.routes(),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
	return s
}

// Handler returns the root handler including all middleware.
func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}

// SetReady controls the result of the readiness probe.
func (s *Server) SetReady(ready bool) {
	s.ready.Store(ready)
}

// Serve accepts connections on ln until Shutdown is called. Like
// http.Server.Serve, it always returns a non-nil error; after Shutdown the
// error is http.ErrServerClosed.
func (s *Server) Serve(ln net.Listener) error {
	return s.httpServer.Serve(ln)
}

// Shutdown gracefully stops the server, waiting for in-flight requests to
// complete until ctx is done.
func (s *Server) Shutdown(ctx context.Context) error {
	s.SetReady(false)
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)

	strict := api.NewStrictHandlerWithOptions(newHandler(), nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  s.handleRequestError,
		ResponseErrorHandlerFunc: s.handleResponseError,
	})
	api.HandlerWithOptions(strict, api.StdHTTPServerOptions{
		BaseRouter:       mux,
		ErrorHandlerFunc: s.handleRequestError,
	})

	var h http.Handler = mux
	h = securityHeaders(h)
	h = recoverer(s.logger)(h)
	h = accessLog(s.logger)(h)
	h = requestID(h)
	return h
}
