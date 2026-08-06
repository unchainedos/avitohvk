package server

import (
	"avitohvk/config"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultReadTimeout     = 10 * time.Second
	defaultWriteTimeout    = 10 * time.Second
	defaultIdleTimeout     = 60 * time.Second
	defaultShutdownTimeout = 10 * time.Second
)

type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
	logger          *slog.Logger
}

func New(cfg config.Config, handler http.Handler, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}

	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = defaultReadTimeout
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = defaultWriteTimeout
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = defaultIdleTimeout
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = defaultShutdownTimeout
	}

	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.Addr,
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		shutdownTimeout: cfg.ShutdownTimeout,
		logger:          log,
	}
}

func (s *Server) Run() error {
	serverErr := make(chan error, 1)

	go func() {
		s.logger.Info("server starting", "addr", s.httpServer.Addr)
		err := s.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server failed to start: %w", err)

	case sig := <-stop:
		s.logger.Info("shutdown signal received", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("graceful shutdown failed, forcing close", "error", err)
		if closeErr := s.httpServer.Close(); closeErr != nil {
			s.logger.Error("force close failed", "error", closeErr)
		}
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	s.logger.Info("server stopped gracefully")
	return nil
}
