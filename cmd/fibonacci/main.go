// Command fibonacci runs the Fibonacci HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/cecobask/fibonacci/internal/config"
	"github.com/cecobask/fibonacci/internal/server"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(context.Background(), os.Getenv, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "fibonacci: %v\n", err)
		os.Exit(1)
	}
}

// run starts the server and blocks until ctx is cancelled or an interrupt
// signal is received, then shuts the server down gracefully.
func run(ctx context.Context, getenv func(string) string, stdout io.Writer) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(getenv)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	logger := newLogger(stdout, cfg)

	addr := net.JoinHostPort("", strconv.Itoa(cfg.Port))
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}

	srv := server.New(addr, logger)
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ln)
	}()
	srv.SetReady(true)
	logger.InfoContext(ctx, "server started", slog.String("addr", ln.Addr().String()), slog.String("version", version))

	select {
	case err := <-serveErr:
		return fmt.Errorf("serving: %w", err)
	case <-ctx.Done():
	}
	stop() // a second signal terminates the process immediately

	logger.InfoContext(ctx, "shutdown started", slog.Duration("delay", cfg.ShutdownDelay), slog.Duration("timeout", cfg.ShutdownTimeout))
	srv.SetReady(false)
	time.Sleep(cfg.ShutdownDelay)

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down: %w", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serving: %w", err)
	}

	logger.InfoContext(ctx, "shutdown complete")
	return nil
}

func newLogger(w io.Writer, cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	var h slog.Handler
	switch cfg.LogFormat {
	case config.LogFormatText:
		h = slog.NewTextHandler(w, opts)
	default:
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(h).With(slog.String("service", "fibonacci"))
}
