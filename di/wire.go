//go:build wireinject
// +build wireinject

package di

import (
	"github.com/savioruz/oil/config"
	"github.com/savioruz/oil/infras/jwt"
	"github.com/savioruz/oil/infras/otel"
	"github.com/savioruz/oil/infras/postgres"
	"github.com/savioruz/oil/infras/redis"
	"github.com/savioruz/oil/infras/s3"
	"github.com/savioruz/oil/infras/unleash"
	todoHandler "github.com/savioruz/oil/internal/handlers/todo"
	userprofileHandler "github.com/savioruz/oil/internal/handlers/userprofile"
	"github.com/savioruz/oil/permissions"
	"github.com/savioruz/oil/shared/cache"
	"github.com/savioruz/oil/shared/singleflight"
	"github.com/savioruz/oil/transport/http"
	"github.com/savioruz/oil/transport/http/middleware"
	"github.com/savioruz/oil/transport/http/router"

	"github.com/google/wire"

	todoRepository "github.com/savioruz/oil/internal/modules/todo/repository"
	todoService "github.com/savioruz/oil/internal/modules/todo/service"

	userprofileRepository "github.com/savioruz/oil/internal/modules/userprofile/repository"
	userprofileService "github.com/savioruz/oil/internal/modules/userprofile/service"

	galleryHandler "github.com/savioruz/oil/internal/handlers/gallery"
	galleryRepository "github.com/savioruz/oil/internal/modules/gallery/repository"
	galleryService "github.com/savioruz/oil/internal/modules/gallery/service"
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

var todoModule = wire.NewSet(
	todoRepository.New,
	todoService.New,
)

var userprofileModule = wire.NewSet(
	userprofileRepository.New,
	userprofileService.New,
)

var galleryModule = wire.NewSet(
	galleryRepository.New,
	galleryService.New,
)

var modules = wire.NewSet(
	todoModule,
	userprofileModule,
	galleryModule,
)

var routing = wire.NewSet(
	wire.Struct(new(router.ModuleHandlers), "*"),
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
		modules,
		routing,
		http.New,
	)

	return &http.HTTP{}, nil
}
