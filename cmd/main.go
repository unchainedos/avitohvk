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
	"avitohvk/internal/transport/handler/props"
	"avitohvk/internal/transport/handler/search"
	"avitohvk/internal/transport/handler/user"
	"avitohvk/internal/transport/handler/users"
	"avitohvk/internal/transport/handler/wish"
	"avitohvk/internal/transport/middleware"
	"avitohvk/internal/transport/router"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.NewConfig()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	itemHandler := item.New()
	wishHandler := wish.New()
	usersHandler := users.New()

	handler := router.New(
		router.WithGroup([]router.RouteRegistrator{
			auth.New(),
			search.New(),
			router.RegistratorFunc(itemHandler.RegisterPublicRoutes),
			router.RegistratorFunc(wishHandler.RegisterPublicRoutes),
			router.RegistratorFunc(usersHandler.RegisterPublicRoutes),
		}),
		router.WithGroup([]router.RouteRegistrator{
			chown.New(),
			deal.New(),
			user.New(),
			props.New(),
			router.RegistratorFunc(itemHandler.RegisterProtectedRoutes),
			router.RegistratorFunc(wishHandler.RegisterProtectedRoutes),
			router.RegistratorFunc(usersHandler.RegisterProtectedRoutes),
		}, middleware.NewJWTAuth([]byte(cfg.JWT.Secret), logger)),
	)

	srv := server.StartServer(cfg, handler, logger)
	srv.GracefulShutdown(logger)
}
