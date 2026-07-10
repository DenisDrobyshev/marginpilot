// Package httpx is a thin wrapper over net/http that every service uses to get
// a consistent health endpoint, timeouts and graceful shutdown.
package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server wraps http.Server with a shared mux, sane timeouts and graceful
// shutdown driven by SIGINT/SIGTERM.
type Server struct {
	addr string
	log  *slog.Logger
	mux  *http.ServeMux
}

// New creates a Server bound to addr (e.g. ":8080").
func New(addr string, log *slog.Logger) *Server {
	return &Server{addr: addr, log: log, mux: http.NewServeMux()}
}

// Mux exposes the router so inbound adapters can register their routes.
func (s *Server) Mux() *http.ServeMux { return s.mux }

// Run registers /healthz, starts listening and blocks until the process is
// signalled, then drains in-flight requests within a 10s deadline.
func (s *Server) Run(ctx context.Context) error {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:              s.addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() {
		s.log.Info("http server listening", "addr", s.addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		s.log.Info("shutdown signal received, draining")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
