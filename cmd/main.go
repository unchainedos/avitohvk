package main

import (
	"log/slog"
	"os"

	"avitohvk/config"
	"avitohvk/internal/server"
	"avitohvk/internal/transport/router"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.NewConfig()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	handler := router.New()

	srv := server.StartServer(cfg, handler, logger)
	srv.GracefulShutdown(logger)
}
