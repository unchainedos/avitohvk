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

	cfg := config.Config{Addr: "0.0.0.0:8080"}

	handler := router.New()

	srv := server.New(cfg, handler, logger)
	if err := srv.Run(); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
