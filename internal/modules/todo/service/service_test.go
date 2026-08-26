package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/savioruz/oil/config"
	"github.com/savioruz/oil/infras/otel"
	otelMocks "github.com/savioruz/oil/infras/otel/mocks"
	"github.com/savioruz/oil/infras/unleash"
	todoMocks "github.com/savioruz/oil/internal/modules/todo/mocks"
	"github.com/savioruz/oil/internal/modules/todo/model"
	"github.com/savioruz/oil/internal/modules/todo/model/dto"
	"github.com/savioruz/oil/internal/modules/todo/service"
	cacheMocks "github.com/savioruz/oil/shared/cache/mocks"
	"github.com/savioruz/oil/shared/constant"
	gDto "github.com/savioruz/oil/shared/dto"
	gModel "github.com/savioruz/oil/shared/model"
	"github.com/savioruz/oil/shared/singleflight"
)

func setup(t *testing.T) (*gomock.Controller, *todoMocks.MockTodo, *cacheMocks.MockRedisCache, *config.Config, otel.Otel) {
	ctrl := gomock.NewController(t)
	mockRepo := todoMocks.NewMockTodo(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := otelMocks.NewOtel()
	cfg := &config.Config{}
	cfg.Cache.TTL = 3600
	return ctrl, mockRepo, mockCache, cfg, mockOtel
}

func TestTodoService_Create(t *testing.T) {
	ctrl, mockRepo, mockCache, cfg, mockOtel := setup(t)
	defer ctrl.Finish()

	ff, _ := unleash.New(cfg)
	svc := service.New(mockRepo, cfg, mockCache, mockOtel, ff, singleflight.New())

	tests := []struct {
		name      string
		req       dto.CreateTodoRequest
		setupMock func()
		wantErr   bool
	}{
		{
			name: "success",
			req: dto.CreateTodoRequest{
				Title:       "Test Todo",
				Description: "Test Description",
			},
			setupMock: func() {
				mockRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)
				mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "error - repository error",
			req: dto.CreateTodoRequest{
				Title:       "Test Todo",
				Description: "Test Description",
			},
			setupMock: func() {
				mockRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.WithValue(context.Background(), constant.ContextKeyUserID, "test-user-id")
			err := svc.Create(ctx, tt.req)

			time.Sleep(10 * time.Millisecond)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTodoService_GetAll(t *testing.T) {
	ctrl, mockRepo, mockCache, cfg, mockOtel := setup(t)
	defer ctrl.Finish()

	ff, _ := unleash.New(cfg)
	svc := service.New(mockRepo, cfg, mockCache, mockOtel, ff, singleflight.New())

	tests := []struct {
		name       string
		params     gDto.QueryParams
		filter     gDto.FilterGroup
		setupMock  func()
		wantErr    bool
		wantResult dto.GetTodosResponse
	}{
		{
			name:   "success",
			params: gDto.QueryParams{Limit: 10, Page: 1},
			filter: gDto.FilterGroup{},
			setupMock: func() {
				mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("cache miss"))
				mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("cache miss"))
				mockRepo.EXPECT().Count(gomock.Any(), gomock.Any()).Return(1, nil)
				now := time.Now()
				todos := []model.Todo{{
					ID:          "test-id",
					Title:       "Test Todo",
					Description: "Test Description",
					Completed:   false,
					Metadata: gModel.Metadata{
						CreatedAt:  now,
						ModifiedAt: now,
						CreatedBy:  "test-user",
						ModifiedBy: "test-user",
					},
				}}
				mockRepo.EXPECT().GetAll(gomock.Any(), gomock.Any(), gomock.Any()).Return(todos, nil)
				mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
			wantResult: dto.GetTodosResponse{
				TotalData: 1,
				TotalPage: 1,
			},
		},
		{
			name:   "error - count error",
			params: gDto.QueryParams{Limit: 10, Page: 1},
			filter: gDto.FilterGroup{},
			setupMock: func() {
				mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("cache miss"))
				mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("cache miss"))
				mockRepo.EXPECT().Count(gomock.Any(), gomock.Any()).Return(0, errors.New("count error"))
			},
			wantErr: true,
		},
		{
			name:   "error - get all error",
			params: gDto.QueryParams{Limit: 10, Page: 1},
			filter: gDto.FilterGroup{},
			setupMock: func() {
				mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("cache miss"))
				mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("cache miss"))
				mockRepo.EXPECT().Count(gomock.Any(), gomock.Any()).Return(1, nil)
				mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), 1, gomock.Any()).Return(nil).AnyTimes()
				mockRepo.EXPECT().GetAll(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("get all error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.Background()
			result, err := svc.GetAll(ctx, tt.params, tt.filter)

			time.Sleep(10 * time.Millisecond)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult.TotalData, result.TotalData)
				assert.Equal(t, tt.wantResult.TotalPage, result.TotalPage)
			}
		})
	}
}

func TestTodoService_Count(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := todoMocks.NewMockTodo(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := otelMocks.NewOtel()

	cfg := &config.Config{}
	cfg.Cache.TTL = 3600

	ff, _ := unleash.New(cfg)
	svc := service.New(mockRepo, cfg, mockCache, mockOtel, ff, singleflight.New())

	tests := []struct {
		name       string
		params     gDto.QueryParams
		filter     gDto.FilterGroup
		setupMock  func()
		wantResult int
		wantErr    bool
	}{
		{
			name:   "successful count with cache hit",
			params: gDto.QueryParams{},
			filter: gDto.FilterGroup{},
			setupMock: func() {
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, key string, dest *int) error {
						*dest = 5
						return nil
					})
			},
			wantResult: 5,
			wantErr:    false,
		},
		{
			name:   "successful count with cache miss",
			params: gDto.QueryParams{},
			filter: gDto.FilterGroup{},
			setupMock: func() {
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("cache miss"))

				mockRepo.EXPECT().
					Count(gomock.Any(), gomock.Any()).
					Return(10, nil)

				mockCache.EXPECT().
					Save(gomock.Any(), gomock.Any(), 10, gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			wantResult: 10,
			wantErr:    false,
		},
		{
			name:   "repository error",
			params: gDto.QueryParams{},
			filter: gDto.FilterGroup{},
			setupMock: func() {
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("cache miss"))

				mockRepo.EXPECT().
					Count(gomock.Any(), gomock.Any()).
					Return(0, errors.New("database error"))
			},
			wantResult: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.Background()
			result, err := svc.Count(ctx, tt.params, tt.filter)

			// Allow time for goroutines to complete
			time.Sleep(10 * time.Millisecond)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}
		})
	}
}

func TestTodoService_Get(t *testing.T) {
	ctrl, mockRepo, mockCache, cfg, mockOtel := setup(t)
	defer ctrl.Finish()

	ff, _ := unleash.New(cfg)
	svc := service.New(mockRepo, cfg, mockCache, mockOtel, ff, singleflight.New())

	now := time.Now()
	todo := model.Todo{
		ID:          "test-id",
		Title:       "Test Todo",
		Description: "Test Description",
		Completed:   false,
		Metadata: gModel.Metadata{
			CreatedAt:  now,
			ModifiedAt: now,
			CreatedBy:  "test-user",
			ModifiedBy: "test-user",
		},
	}

	tests := []struct {
		name      string
		id        string
		setupMock func()
		wantErr   bool
		wantID    string
	}{
		{
			name: "success - cache hit",
			id:   "test-id",
			setupMock: func() {
				mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
			wantID:  "",
		},
		{
			name: "success - cache miss",
			id:   "test-id",
			setupMock: func() {
				mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("cache miss"))
				mockRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(todo, nil)
				mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
			wantID:  "test-id",
		},
		{
			name: "error - not found",
			id:   "nonexistent-id",
			setupMock: func() {
				mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("cache miss"))
				mockRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(model.Todo{}, nil)
			},
			wantErr: true,
		},
		{
			name: "error - repository error",
			id:   "test-id",
			setupMock: func() {
				mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("cache miss"))
				mockRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(model.Todo{}, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.Background()
			result, err := svc.Get(ctx, tt.id)

			time.Sleep(10 * time.Millisecond)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantID != "" {
					assert.Equal(t, tt.wantID, result.ID)
				}
			}
		})
	}
}

func TestTodoService_Update(t *testing.T) {
	ctrl, mockRepo, mockCache, cfg, mockOtel := setup(t)
	defer ctrl.Finish()

	ff, _ := unleash.New(cfg)
	svc := service.New(mockRepo, cfg, mockCache, mockOtel, ff, singleflight.New())

	tests := []struct {
		name      string
		req       dto.UpdateTodoRequest
		id        string
		setupMock func()
		wantErr   bool
	}{
		{
			name: "success",
			req: dto.UpdateTodoRequest{
				Title:       "Updated Title",
				Description: "Updated Description",
			},
			id: "test-id",
			setupMock: func() {
				mockRepo.EXPECT().Exist(gomock.Any(), gomock.Any()).Return(true, nil)
				mockRepo.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockCache.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)
				mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "error - empty request",
			req:  dto.UpdateTodoRequest{},
			id:   "test-id",
			setupMock: func() {
			},
			wantErr: true,
		},
		{
			name: "error - not found",
			req: dto.UpdateTodoRequest{
				Title: "Updated Title",
			},
			id: "nonexistent-id",
			setupMock: func() {
				mockRepo.EXPECT().Exist(gomock.Any(), gomock.Any()).Return(false, nil)
			},
			wantErr: true,
		},
		{
			name: "error - exist check error",
			req: dto.UpdateTodoRequest{
				Title: "Updated Title",
			},
			id: "test-id",
			setupMock: func() {
				mockRepo.EXPECT().Exist(gomock.Any(), gomock.Any()).Return(false, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "error - update error",
			req: dto.UpdateTodoRequest{
				Title: "Updated Title",
			},
			id: "test-id",
			setupMock: func() {
				mockRepo.EXPECT().Exist(gomock.Any(), gomock.Any()).Return(true, nil)
				mockRepo.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("update error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.WithValue(context.Background(), constant.ContextKeyUserID, "test-user-id")
			err := svc.Update(ctx, tt.req, tt.id)

			time.Sleep(10 * time.Millisecond)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTodoService_Delete(t *testing.T) {
	ctrl, mockRepo, mockCache, cfg, mockOtel := setup(t)
	defer ctrl.Finish()

	ff, _ := unleash.New(cfg)
	svc := service.New(mockRepo, cfg, mockCache, mockOtel, ff, singleflight.New())

	tests := []struct {
		name      string
		id        string
		setupMock func()
		wantErr   bool
	}{
		{
			name: "success",
			id:   "test-id",
			setupMock: func() {
				mockRepo.EXPECT().Exist(gomock.Any(), gomock.Any()).Return(true, nil)
				mockRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)
				mockCache.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)
				mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "error - not found",
			id:   "nonexistent-id",
			setupMock: func() {
				mockRepo.EXPECT().Exist(gomock.Any(), gomock.Any()).Return(false, nil)
			},
			wantErr: true,
		},
		{
			name: "error - exist check error",
			id:   "test-id",
			setupMock: func() {
				mockRepo.EXPECT().Exist(gomock.Any(), gomock.Any()).Return(false, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "error - delete error",
			id:   "test-id",
			setupMock: func() {
				mockRepo.EXPECT().Exist(gomock.Any(), gomock.Any()).Return(true, nil)
				mockRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(errors.New("delete error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.Background()
			err := svc.Delete(ctx, tt.id)

			time.Sleep(10 * time.Millisecond)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
