package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"avitohvk/config"
	chownrepo "avitohvk/internal/repository/chown"
	dealrepo "avitohvk/internal/repository/deal"
	itemrepo "avitohvk/internal/repository/item"
	proposalrepo "avitohvk/internal/repository/proposal"
	searchrepo "avitohvk/internal/repository/search"
	userrepo "avitohvk/internal/repository/user"
	wishrepo "avitohvk/internal/repository/wish"
	"avitohvk/internal/server"
	chownservice "avitohvk/internal/service/chown"
	itemservice "avitohvk/internal/service/item"
	proposalservice "avitohvk/internal/service/proposal"
	searchservice "avitohvk/internal/service/search"
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

	searchRepository := searchrepo.NewRepository(pool)
	userRepository := userrepo.NewRepository(pool)
	itemRepository := itemrepo.NewRepository(pool)
	wishRepository := wishrepo.NewRepository(pool)
	chownRepo := chownrepo.NewRepository(pool)
	dealRepo := dealrepo.NewRepository(pool)
	proposalRepo := proposalrepo.NewRepository(pool)

	searchSvc := searchservice.NewService(searchRepository)
	userSvc := userservice.NewService(userRepository, []byte(cfg.JWT.Secret), jwtTTL)
	itemSvc := itemservice.NewService(itemRepository)
	wishSvc := wishservice.NewService(wishRepository)
	chownSvc := chownservice.NewService(chownRepo)
	proposalSvc := proposalservice.NewService(dealRepo, proposalRepo, chownRepo)

	searchHandler := search.New(searchSvc)
	authHandler := auth.New(userSvc, jwtTTL)
	userHandler := user.New(userSvc, itemSvc)
	itemHandler := item.New(itemSvc)
	wishHandler := wish.New(wishSvc)
	usersHandler := users.New(itemSvc)
	chownHandler := chown.New(chownSvc)
	dealHandler := deal.New(proposalSvc)
	propsHandler := props.New(proposalSvc)

	handler := router.New(
		router.WithGroup([]router.RouteRegistrator{
			authHandler,
			router.RegistratorFunc(searchHandler.RegisterRoutes),
			router.RegistratorFunc(itemHandler.RegisterPublicRoutes),
			router.RegistratorFunc(wishHandler.RegisterPublicRoutes),
			router.RegistratorFunc(usersHandler.RegisterPublicRoutes),
			router.RegistratorFunc(userHandler.RegisterPublicRoutes),
		}),
		router.WithGroup([]router.RouteRegistrator{
			chownHandler,
			dealHandler,
			propsHandler,
			router.RegistratorFunc(itemHandler.RegisterProtectedRoutes),
			router.RegistratorFunc(wishHandler.RegisterProtectedRoutes),
			router.RegistratorFunc(usersHandler.RegisterProtectedRoutes),
			router.RegistratorFunc(userHandler.RegisterProtectedRoutes),
		}, middleware.NewJWTAuth([]byte(cfg.JWT.Secret), logger)),
	)

	srv := server.StartServer(cfg, handler, logger)
	srv.GracefulShutdown(logger)
}
