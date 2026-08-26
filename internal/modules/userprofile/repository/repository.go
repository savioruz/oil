package repository

//go:generate go run go.uber.org/mock/mockgen -source=./repository.go -destination=../mocks/repository_mock.go -package=mocks

import (
	"context"
	"github.com/savioruz/oil/infras/otel"
	"github.com/savioruz/oil/infras/postgres"
	"github.com/savioruz/oil/internal/modules/userprofile/model"
	gDto "github.com/savioruz/oil/shared/dto"
	gRepo "github.com/savioruz/oil/shared/repository"
)

type Userprofile interface {
	Insert(ctx context.Context, model model.Userprofile) error
	Get(ctx context.Context, filter gDto.FilterGroup, columns ...string) (model.Userprofile, error)
	GetAll(ctx context.Context, params gDto.QueryParams, filter gDto.FilterGroup, columns ...string) ([]model.Userprofile, error)
	Exist(ctx context.Context, filter gDto.FilterGroup) (bool, error)
	Count(ctx context.Context, filter gDto.FilterGroup) (int, error)
	Update(ctx context.Context, req map[string]any, filter gDto.FilterGroup) error
	Delete(ctx context.Context, filter gDto.FilterGroup) error
}

type repositoryImpl struct {
	gRepo.Repository[model.Userprofile]
	db   *postgres.Connection
	otel otel.Otel
}

func New(db *postgres.Connection, otel otel.Otel) Userprofile {
	return &repositoryImpl{
		Repository: gRepo.NewRepository[model.Userprofile](model.EntityName, model.TableName, model.FieldID, db, otel),
		db:         db,
		otel:       otel,
	}
}
