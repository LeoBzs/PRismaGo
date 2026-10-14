// Package server owns the HTTP transport: the mux, global middleware and the
// server lifecycle. Feature packages plug in by implementing RouteRegistrar.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/LeoBzs/prismago/internal/config"
	"github.com/LeoBzs/prismago/internal/httpx"
)

// RouteRegistrar is implemented by each feature's handler.
type RouteRegistrar interface {
	RegisterRoutes(mux *http.ServeMux)
}

type Server struct {
	http            *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

func New(cfg config.Config, logger *slog.Logger, modules ...RouteRegistrar) *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	for _, m := range modules {
		m.RegisterRoutes(mux)
	}

	// Outermost middleware is listed first. cors sits inside the logger so
	// preflight requests still show up in the logs.
	handler := chain(mux,
		recoverer(logger),
		requestLogger(logger),
		cors(cfg.CORSAllowedOrigins),
	)

	return &Server{
		http: &http.Server{
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		logger:          logger,
		shutdownTimeout: cfg.ShutdownTimeout,
	}
}

// Run serves until ctx is cancelled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("http server listening", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	s.logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	return s.http.Shutdown(shutdownCtx)
}
