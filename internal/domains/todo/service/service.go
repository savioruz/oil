package service

import (
	"context"
	"fmt"
	"oil/config"
	"oil/infras/otel"
	"oil/infras/unleash"
	"oil/internal/domains/todo/model"
	"oil/internal/domains/todo/model/dto"
	"oil/internal/domains/todo/repository"
	"oil/shared"
	"oil/shared/cache"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/errkey"
	"oil/shared/failure"
	"oil/shared/singleflight"

	"github.com/rs/zerolog/log"
)

const (
	cacheGetTodo    = "todo:get"
	cacheGetAllTodo = "todo:gets"
	cacheCountTodo  = "todo:count"
)

type Todo interface {
	Create(ctx context.Context, req dto.CreateTodoRequest) error
	GetAll(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (dto.GetTodosResponse, error)
	Count(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (int, error)
	Get(ctx context.Context, id string) (dto.TodoResponse, error)
	Update(ctx context.Context, req dto.UpdateTodoRequest, id string) error
	Delete(ctx context.Context, id string) error
}

type serviceImpl struct {
	repo  repository.Todo
	cfg   *config.Config
	cache cache.RedisCache
	otel  otel.Otel
	ff    unleash.FeatureFlag
	sf    *singleflight.Group
}

func New(repo repository.Todo, cfg *config.Config, cache cache.RedisCache, otel otel.Otel, ff unleash.FeatureFlag, sf *singleflight.Group) Todo {
	return &serviceImpl{
		repo:  repo,
		cfg:   cfg,
		cache: cache,
		otel:  otel,
		ff:    ff,
		sf:    sf,
	}
}

func (s *serviceImpl) Create(ctx context.Context, req dto.CreateTodoRequest) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Create")
	defer scope.End()
	defer scope.TraceIfError(err)

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)

	if s.ff.IsEnabled(ctx, "todo-create-v2") {
		return s.createV2(ctx, req, user)
	}

	return s.createV1(ctx, req, user)
}

func (s *serviceImpl) createV1(ctx context.Context, req dto.CreateTodoRequest, user string) error {
	if err := s.repo.Insert(ctx, req.ToModel(user)); err != nil {
		log.Error().Err(err).Msg("failed to create todo")

		return failure.InternalErrorWithKey(errkey.ErrTodoCreateFailed, fmt.Sprintf("failed to create todo: %v", err))
	}

	go func() {
		c := context.WithoutCancel(ctx)

		shared.InvalidateCaches(c, s.cache, cacheGetAllTodo)
		shared.InvalidateCaches(c, s.cache, cacheCountTodo)
	}()

	return nil
}

func (s *serviceImpl) createV2(ctx context.Context, req dto.CreateTodoRequest, user string) error {
	if err := s.repo.Insert(ctx, req.ToModelWithFullFields(user)); err != nil {
		log.Error().Err(err).Msg("failed to create todo")

		return failure.InternalErrorWithKey(errkey.ErrTodoCreateFailed, fmt.Sprintf("failed to create todo: %v", err))
	}

	go func() {
		c := context.WithoutCancel(ctx)

		shared.InvalidateCaches(c, s.cache, cacheGetAllTodo)
		shared.InvalidateCaches(c, s.cache, cacheCountTodo)
	}()

	return nil
}

func (s *serviceImpl) GetAll(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (res dto.GetTodosResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".GetAll")
	defer scope.End()
	defer scope.TraceIfError(err)

	cacheKey := shared.BuildCacheKeyWithQuery(cacheGetAllTodo, req, filter)

	return cache.Remember(ctx, s.cache, s.sf, cacheKey, s.cfg.Cache.TTL, func(ctx context.Context) (dto.GetTodosResponse, error) {
		var out dto.GetTodosResponse

		total, err := s.Count(ctx, req, filter)
		if err != nil {
			log.Error().Err(err).Msg("failed to count todos")

			return out, failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to count todos: %v", err))
		}

		models, err := s.repo.GetAll(ctx, req, filter)
		if err != nil {
			log.Error().Err(err).Msg("failed to get todos")

			return out, failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to get todos: %v", err))
		}

		out.FromModels(models, total, req.Limit)

		return out, nil
	})
}

func (s *serviceImpl) Count(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (res int, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Count")
	defer scope.End()
	defer scope.TraceIfError(err)

	cacheKey := shared.BuildCacheKeyWithQuery(cacheCountTodo, req, filter)

	return cache.Remember(ctx, s.cache, s.sf, cacheKey, s.cfg.Cache.TTL, func(ctx context.Context) (int, error) {
		count, err := s.repo.Count(ctx, filter)
		if err != nil {
			log.Error().Err(err).Msg("failed to count todos")

			return 0, failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to count todos: %v", err))
		}

		return count, nil
	})
}

func (s *serviceImpl) Get(ctx context.Context, id string) (res dto.TodoResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Get")
	defer scope.End()
	defer scope.TraceIfError(nil)

	cacheKey := shared.BuildCacheKey(cacheGetTodo, id)

	return cache.Remember(ctx, s.cache, s.sf, cacheKey, s.cfg.Cache.TTL, func(ctx context.Context) (dto.TodoResponse, error) {
		var out dto.TodoResponse

		todo, err := s.repo.Get(ctx, shared.SingleFilter(id, model.FieldID, model.TableName))
		if err != nil {
			log.Error().Err(err).Msg("failed to get todo")

			return out, failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to get todo: %v", err))
		}

		if todo.ID == constant.EmptyString {
			return out, failure.NotFoundWithKey(errkey.ErrTodoNotFound, "todo not found")
		}

		out.FromModel(todo)

		return out, nil
	})
}

func (s *serviceImpl) Update(ctx context.Context, req dto.UpdateTodoRequest, id string) error {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Update")
	defer scope.End()
	defer scope.TraceIfError(nil)

	if req == (dto.UpdateTodoRequest{}) {
		return failure.BadRequestFromString("update request cannot be empty") // nolint:wrapcheck
	}

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)
	filter := shared.SingleFilter(id, model.FieldID, model.TableName)

	exist, err := s.repo.Exist(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if todo exists")

		return failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to check if todo exists: %v", err))
	}

	if !exist {
		log.Error().Msg("todo not found")

		return failure.NotFoundWithKey(errkey.ErrTodoNotFound, "todo not found")
	}

	updatedFields := shared.TransformFields(req, user)
	if err := s.repo.Update(ctx, updatedFields, filter); err != nil {
		log.Error().Err(err).Msg("failed to update todo")

		return failure.InternalErrorWithKey(errkey.ErrTodoUpdateFailed, fmt.Sprintf("failed to update todo: %v", err))
	}

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Delete(c, shared.BuildCacheKey(cacheGetTodo, id)); err != nil {
			log.Error().Err(err).Msg("failed to delete todo from cache")
		}

		shared.InvalidateCaches(c, s.cache, cacheGetAllTodo)
		shared.InvalidateCaches(c, s.cache, cacheCountTodo)
	}()

	return nil
}

func (s *serviceImpl) Delete(ctx context.Context, id string) error {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Delete")
	defer scope.End()
	defer scope.TraceIfError(nil)

	exist, err := s.repo.Exist(ctx, shared.SingleFilter(id, model.FieldID, model.TableName))
	if err != nil {
		log.Error().Err(err).Msg("failed to check if todo exists")

		return failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to check if todo exists: %v", err))
	}

	if !exist {
		log.Error().Msg("todo not found")

		return failure.NotFoundWithKey(errkey.ErrTodoNotFound, "todo not found")
	}

	if err := s.repo.Delete(ctx, shared.SingleFilter(id, model.FieldID, model.TableName)); err != nil {
		log.Error().Err(err).Msg("failed to delete todo")

		return failure.InternalErrorWithKey(errkey.ErrTodoDeleteFailed, fmt.Sprintf("failed to delete todo: %v", err))
	}

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Delete(c, shared.BuildCacheKey(cacheGetTodo, id)); err != nil {
			log.Error().Err(err).Msg("failed to delete todo from cache")
		}

		shared.InvalidateCaches(c, s.cache, cacheGetAllTodo)
		shared.InvalidateCaches(c, s.cache, cacheCountTodo)
	}()

	return nil
}
