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
	"oil/infras/unleash"
	todoHandler "oil/internal/handlers/todo"
	userprofileHandler "oil/internal/handlers/userprofile"
	"oil/permissions"
	"oil/shared/cache"
	"oil/shared/singleflight"
	"oil/transport/http"
	"oil/transport/http/middleware"
	"oil/transport/http/router"

	"github.com/google/wire"

	todoRepository "oil/internal/domains/todo/repository"
	todoService "oil/internal/domains/todo/service"

	userprofileRepository "oil/internal/domains/userprofile/repository"
	userprofileService "oil/internal/domains/userprofile/service"

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
	unleash.New,
)

var middlewares = wire.NewSet(
	middleware.NewAppMiddleware,
	middleware.NewAuthRoleMiddleware,
)

var sharedHelpers = wire.NewSet(
	cache.New,
	singleflight.New,
)

var todoDomain = wire.NewSet(
	todoRepository.New,
	todoService.New,
)

var userprofileDomain = wire.NewSet(
	userprofileRepository.New,
	userprofileService.New,
)

var galleryDomain = wire.NewSet(
	galleryRepository.New,
	galleryService.New,
)

var domains = wire.NewSet(
	todoDomain,
	userprofileDomain,
	galleryDomain,
)

var routing = wire.NewSet(
	wire.Struct(new(router.DomainHandlers), "*"),
	todoHandler.New,
	galleryHandler.New,
	userprofileHandler.New,
	router.New,
)

func InitializeService() (*http.HTTP, error) {
	wire.Build(
		configurations,
		infrastructures,
		middlewares,
		sharedHelpers,
		domains,
		routing,
		http.New,
	)

	return &http.HTTP{}, nil
}
