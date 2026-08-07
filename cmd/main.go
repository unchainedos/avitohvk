package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"avitohvk/config"
	itemrepo "avitohvk/internal/repository/item"
	userrepo "avitohvk/internal/repository/user"
	wishrepo "avitohvk/internal/repository/wish"
	"avitohvk/internal/server"
	itemservice "avitohvk/internal/service/item"
	userservice "avitohvk/internal/service/user"
	wishservice "avitohvk/internal/service/wish"
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

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.NewConfig()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		logger.Error("failed to connect db", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	jwtTTL := 24 * time.Hour
	if cfg.JWT.TTL != nil {
		jwtTTL = *cfg.JWT.TTL
	}

	userRepository := userrepo.NewRepository(pool)
	itemRepository := itemrepo.NewRepository(pool)
	wishRepository := wishrepo.NewRepository(pool)

	userSvc := userservice.NewService(userRepository, []byte(cfg.JWT.Secret), jwtTTL)
	itemSvc := itemservice.NewService(itemRepository)
	wishSvc := wishservice.NewService(wishRepository)

	authHandler := auth.New(userSvc)
	userHandler := user.New(userSvc, itemSvc)
	itemHandler := item.New(itemSvc)
	wishHandler := wish.New(wishSvc)
	usersHandler := users.New(itemSvc)

	handler := router.New(
		router.WithGroup([]router.RouteRegistrator{
			authHandler,
			search.New(),
			router.RegistratorFunc(itemHandler.RegisterPublicRoutes),
			router.RegistratorFunc(wishHandler.RegisterPublicRoutes),
			router.RegistratorFunc(usersHandler.RegisterPublicRoutes),
		}),
		router.WithGroup([]router.RouteRegistrator{
			chown.New(),
			deal.New(),
			userHandler,
			props.New(),
			router.RegistratorFunc(itemHandler.RegisterProtectedRoutes),
			router.RegistratorFunc(wishHandler.RegisterProtectedRoutes),
			router.RegistratorFunc(usersHandler.RegisterProtectedRoutes),
		}, middleware.NewJWTAuth([]byte(cfg.JWT.Secret), logger)),
	)

	srv := server.StartServer(cfg, handler, logger)
	srv.GracefulShutdown(logger)
}
