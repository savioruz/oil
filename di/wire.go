//go:build wireinject
// +build wireinject

package di

import (
	"oil/config"
	"oil/infras/jwt"
	"oil/infras/otel"
	"oil/infras/postgres"
	"oil/infras/redis"
	"oil/infras/s3"
	todoHandler "oil/internal/handlers/todo"
	"oil/permissions"
	"oil/shared/cache"
	"oil/transport/http"
	"oil/transport/http/middleware"
	"oil/transport/http/router"

	todoRepository "oil/internal/domains/todo/repository"
	todoService "oil/internal/domains/todo/service"

	"github.com/google/wire"

	authService "oil/internal/domains/auth/service"
	userRepository "oil/internal/domains/user/repository"
	authHandler "oil/internal/handlers/auth"

	galleryRepository "oil/internal/domains/gallery/repository"
	galleryService "oil/internal/domains/gallery/service"
	galleryHandler "oil/internal/handlers/gallery"
)

var configurations = wire.NewSet(
	config.Get,
	permissions.Get,
)

var infrastructures = wire.NewSet(
	postgres.New,
	otel.New,
	redis.New,
	s3.New,
	jwt.New,
	// kafka.New,
)

var middlewares = wire.NewSet(
	middleware.NewAppMiddleware,
	middleware.NewAuthRoleMiddleware,
)

var sharedHelpers = wire.NewSet(
	cache.NewRedisCache,
)

var todoDomain = wire.NewSet(
	todoRepository.New,
	todoService.New,
)

var authDomain = wire.NewSet(
	userRepository.New,
	authService.New,
)

var galleryDomain = wire.NewSet(
	galleryRepository.New,
	galleryService.New,
)

var domains = wire.NewSet(
	todoDomain,
	authDomain,
	galleryDomain,
)

var routing = wire.NewSet(
	wire.Struct(new(router.DomainHandlers), "*"),
	todoHandler.New,
	authHandler.New,
	galleryHandler.New,
	router.New,
)

func InitializeService() *http.HTTP {
	wire.Build(
		configurations,
		infrastructures,
		middlewares,
		sharedHelpers,
		domains,
		routing,
		http.New,
	)

	return &http.HTTP{}
}
