package main

import (
	"log/slog"
	"os"

	"avitohvk/config"
	"avitohvk/internal/server"
	"avitohvk/internal/transport/handler/auth"
	"avitohvk/internal/transport/handler/chown"
	"avitohvk/internal/transport/handler/deal"
	"avitohvk/internal/transport/handler/item"
	"avitohvk/internal/transport/handler/search"
	"avitohvk/internal/transport/handler/user"
	"avitohvk/internal/transport/handler/wish"
	"avitohvk/internal/transport/router"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.NewConfig()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	handler := router.New(
		router.WithGroup([]router.RouteRegistrator{
			chown.New(),
			deal.New(),
			search.New(),
			auth.New(),
			user.New(),
			item.New(),
			wish.New(),
		}),
	)

	srv := server.StartServer(cfg, handler, logger)
	srv.GracefulShutdown(logger)
}
