package server

import (
	"avitohvk/config"
	"context"
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
	defaultShutdownTimeout = 60 * time.Minute
	defaultAddr            = ":8080"
	chanBufferSize         = 1
)

type Server struct {
	internalServer  *http.Server
	channelErr      chan error
	shutdownTimeout time.Duration
}

func (s *Server) Start() {
	go func() {
		s.channelErr <- s.internalServer.ListenAndServe()
		close(s.channelErr)
	}()
}

func NewServer(handler http.Handler, options ...Option) *Server {
	server := &Server{
		internalServer: &http.Server{
			Handler:      handler,
			ReadTimeout:  defaultReadTimeout,
			WriteTimeout: defaultWriteTimeout,
			Addr:         defaultAddr,
		},
		channelErr:      make(chan error, chanBufferSize),
		shutdownTimeout: defaultShutdownTimeout,
	}
	for _, option := range options {
		option(server)
	}

	server.Start()

	return server
}

func (s *Server) FullShutdownTimeout(logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	logger.Info("Shutting down server...\n")

	if err := s.internalServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown filed: %w", err)
	}

	return nil
}

func (s *Server) GracefulShutdown(logger *slog.Logger) {
	osInterruptChan := make(chan os.Signal, chanBufferSize)
	signal.Notify(osInterruptChan, syscall.SIGTERM, syscall.SIGINT)

	timeoutChan := time.After(s.shutdownTimeout)

	select {
	case <-osInterruptChan:
		logger.Info("server interrupted by system or user")
	case <-s.channelErr:
		logger.Error("server error occurred", slog.Any("error", <-s.channelErr))
	case <-timeoutChan:
		logger.Info("shutdown timeout reached")
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	logger.Info("shutting down server...")

	if err := s.internalServer.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", slog.Any("error", err))
	} else {
		logger.Info("server stopped successfully")
	}

	close(osInterruptChan)
}

func StartServer(
	cfg *config.Config,
	handler http.Handler,
	logger *slog.Logger,
) *Server {
	readTimeout := defaultReadTimeout
	if cfg.Server.ReadTimeout != nil {
		readTimeout = *cfg.Server.ReadTimeout
	}

	writeTimeout := defaultWriteTimeout
	if cfg.Server.WriteTimeout != nil {
		writeTimeout = *cfg.Server.WriteTimeout
	}

	shutdownTimeout := cfg.Server.ShutdownTimeout
	if shutdownTimeout == 0 {
		shutdownTimeout = defaultShutdownTimeout
	}

	customServer := NewServer(handler,
		SetReadTimeout(readTimeout),
		SetWriteTimeout(writeTimeout),
		SetAddr(cfg.Server.Addr),
		SetShutdownTimeout(shutdownTimeout),
	)

	logger.Info("successfully created server",
		"addr", customServer.internalServer.Addr,
		"shutdownTimeout", customServer.shutdownTimeout,
	)

	return customServer
}
