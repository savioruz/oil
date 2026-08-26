package service_test

import (
	"context"
	"errors"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/savioruz/oil/config"
	"github.com/savioruz/oil/infras/otel"
	otelMocks "github.com/savioruz/oil/infras/otel/mocks"
	s3mocks "github.com/savioruz/oil/infras/s3/mocks"
	galleryMocks "github.com/savioruz/oil/internal/modules/gallery/mocks"
	"github.com/savioruz/oil/internal/modules/gallery/model"
	"github.com/savioruz/oil/internal/modules/gallery/model/dto"
	"github.com/savioruz/oil/internal/modules/gallery/service"
	"github.com/savioruz/oil/shared"
	cacheMocks "github.com/savioruz/oil/shared/cache/mocks"
	gDto "github.com/savioruz/oil/shared/dto"
	"github.com/savioruz/oil/shared/errkey"
	"github.com/savioruz/oil/shared/failure"
	"github.com/savioruz/oil/shared/singleflight"
)

func setup(t *testing.T) (*gomock.Controller, *galleryMocks.MockGallery, *cacheMocks.MockRedisCache, *config.Config, *s3mocks.MockS3, otel.Otel) {
	ctrl := gomock.NewController(t)
	mockRepo := galleryMocks.NewMockGallery(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := otelMocks.NewOtel()
	mockS3 := s3mocks.NewMockS3(ctrl)
	cfg := &config.Config{}
	cfg.Cache.TTL = 3600
	return ctrl, mockRepo, mockCache, cfg, mockS3, mockOtel
}

func TestCreate(t *testing.T) {
	ctrl, mockRepo, mockCache, cfg, mockS3, mockOtel := setup(t)
	defer ctrl.Finish()

	svc := service.New(mockRepo, cfg, mockCache, mockOtel, mockS3, singleflight.New())

	tests := []struct {
		name      string
		req       dto.CreateGalleryRequest
		setupMock func()
		wantErr   bool
		errKey    errkey.ErrorKey
	}{
		{
			name: "success",
			req: dto.CreateGalleryRequest{
				Title:       "Test Gallery",
				Description: "Test Description",
				Images:      []string{"https://example.com/image1.jpg"},
			},
			setupMock: func() {
				mockRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)
				mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "error - insert failed",
			req: dto.CreateGalleryRequest{
				Title:       "Test Gallery",
				Description: "Test Description",
				Images:      []string{"https://example.com/image1.jpg"},
			},
			setupMock: func() {
				mockRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
			},
			wantErr: true,
			errKey:  errkey.ErrGalleryCreateFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := svc.Create(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				f, ok := err.(*failure.Failure)
				assert.True(t, ok)
				assert.Equal(t, tt.errKey, f.Key)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetAll(t *testing.T) {
	ctrl, mockRepo, mockCache, cfg, mockS3, mockOtel := setup(t)
	defer ctrl.Finish()

	svc := service.New(mockRepo, cfg, mockCache, mockOtel, mockS3, singleflight.New())

	mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("cache miss")).AnyTimes()
	mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	tests := []struct {
		name       string
		params     gDto.QueryParams
		filter     gDto.FilterGroup
		setupMock  func()
		wantErr    bool
		errKey     errkey.ErrorKey
		wantResult dto.GetGalleriesResponse
	}{
		{
			name:   "success",
			params: gDto.QueryParams{Limit: 10, Page: 1},
			filter: gDto.FilterGroup{},
			setupMock: func() {
				mockRepo.EXPECT().Count(gomock.Any(), gomock.Any()).Return(1, nil)
				mockRepo.EXPECT().GetAll(gomock.Any(), gomock.Any(), gomock.Any()).Return([]model.Gallery{
					{ID: "gallery-1", Title: "Test Gallery"},
				}, nil)
			},
			wantErr: false,
			wantResult: dto.GetGalleriesResponse{
				TotalData: 1,
				TotalPage: 1,
			},
		},
		{
			name:   "error - count failed",
			params: gDto.QueryParams{Limit: 10, Page: 1},
			filter: gDto.FilterGroup{},
			setupMock: func() {
				mockRepo.EXPECT().Count(gomock.Any(), gomock.Any()).Return(0, errors.New("db error"))
			},
			wantErr: true,
			errKey:  errkey.ErrDatabaseQuery,
		},
		{
			name:   "error - get all failed",
			params: gDto.QueryParams{Limit: 10, Page: 1},
			filter: gDto.FilterGroup{},
			setupMock: func() {
				mockRepo.EXPECT().Count(gomock.Any(), gomock.Any()).Return(1, nil)
				mockRepo.EXPECT().GetAll(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			wantErr: true,
			errKey:  errkey.ErrDatabaseQuery,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := svc.GetAll(context.Background(), tt.params, tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
				f, ok := err.(*failure.Failure)
				assert.True(t, ok)
				assert.Equal(t, tt.errKey, f.Key)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult.TotalData, result.TotalData)
				assert.Equal(t, tt.wantResult.TotalPage, result.TotalPage)
			}
		})
	}
}

func TestGet(t *testing.T) {
	ctrl, mockRepo, mockCache, cfg, mockS3, mockOtel := setup(t)
	defer ctrl.Finish()

	svc := service.New(mockRepo, cfg, mockCache, mockOtel, mockS3, singleflight.New())

	mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("cache miss")).AnyTimes()
	mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	tests := []struct {
		name      string
		id        string
		setupMock func()
		wantErr   bool
		errKey    errkey.ErrorKey
		wantID    string
	}{
		{
			name: "success",
			id:   "gallery-1",
			setupMock: func() {
				filter := shared.SingleFilter("gallery-1", model.FieldID, model.TableName)
				mockRepo.EXPECT().Get(gomock.Any(), filter).Return(model.Gallery{
					ID:    "gallery-1",
					Title: "Test Gallery",
				}, nil)
			},
			wantErr: false,
			wantID:  "gallery-1",
		},
		{
			name: "error - not found",
			id:   "nonexistent",
			setupMock: func() {
				filter := shared.SingleFilter("nonexistent", model.FieldID, model.TableName)
				mockRepo.EXPECT().Get(gomock.Any(), filter).Return(model.Gallery{}, nil)
			},
			wantErr: true,
			errKey:  errkey.ErrGalleryNotFound,
		},
		{
			name: "error - database query failed",
			id:   "gallery-1",
			setupMock: func() {
				filter := shared.SingleFilter("gallery-1", model.FieldID, model.TableName)
				mockRepo.EXPECT().Get(gomock.Any(), filter).Return(model.Gallery{}, errors.New("db error"))
			},
			wantErr: true,
			errKey:  errkey.ErrDatabaseQuery,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := svc.Get(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				f, ok := err.(*failure.Failure)
				assert.True(t, ok)
				assert.Equal(t, tt.errKey, f.Key)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantID, result.ID)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	ctrl, mockRepo, mockCache, cfg, mockS3, mockOtel := setup(t)
	defer ctrl.Finish()

	svc := service.New(mockRepo, cfg, mockCache, mockOtel, mockS3, singleflight.New())

	mockCache.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	tests := []struct {
		name      string
		id        string
		req       dto.UpdateGalleryRequest
		setupMock func()
		wantErr   bool
		errKey    errkey.ErrorKey
	}{
		{
			name: "success",
			id:   "gallery-1",
			req: dto.UpdateGalleryRequest{
				Title:       "Updated Title",
				Description: "Updated Description",
			},
			setupMock: func() {
				filter := shared.SingleFilter("gallery-1", model.FieldID, model.TableName)
				mockRepo.EXPECT().Exist(gomock.Any(), filter).Return(true, nil)
				mockRepo.EXPECT().Update(gomock.Any(), gomock.Any(), filter).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - not found",
			id:   "nonexistent",
			req: dto.UpdateGalleryRequest{
				Title: "Updated Title",
			},
			setupMock: func() {
				filter := shared.SingleFilter("nonexistent", model.FieldID, model.TableName)
				mockRepo.EXPECT().Exist(gomock.Any(), filter).Return(false, nil)
			},
			wantErr: true,
			errKey:  errkey.ErrGalleryNotFound,
		},
		{
			name: "error - exist check failed",
			id:   "gallery-1",
			req: dto.UpdateGalleryRequest{
				Title: "Updated Title",
			},
			setupMock: func() {
				filter := shared.SingleFilter("gallery-1", model.FieldID, model.TableName)
				mockRepo.EXPECT().Exist(gomock.Any(), filter).Return(false, errors.New("db error"))
			},
			wantErr: true,
			errKey:  errkey.ErrDatabaseQuery,
		},
		{
			name: "error - update failed",
			id:   "gallery-1",
			req: dto.UpdateGalleryRequest{
				Title: "Updated Title",
			},
			setupMock: func() {
				filter := shared.SingleFilter("gallery-1", model.FieldID, model.TableName)
				mockRepo.EXPECT().Exist(gomock.Any(), filter).Return(true, nil)
				mockRepo.EXPECT().Update(gomock.Any(), gomock.Any(), filter).Return(errors.New("db error"))
			},
			wantErr: true,
			errKey:  errkey.ErrGalleryUpdateFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := svc.Update(context.Background(), tt.req, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				f, ok := err.(*failure.Failure)
				assert.True(t, ok)
				assert.Equal(t, tt.errKey, f.Key)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	ctrl, mockRepo, mockCache, cfg, mockS3, mockOtel := setup(t)
	defer ctrl.Finish()

	cfg.External.S3.BucketName = "test-bucket"

	svc := service.New(mockRepo, cfg, mockCache, mockOtel, mockS3, singleflight.New())

	tests := []struct {
		name      string
		id        string
		setupMock func()
		wantErr   bool
		errKey    errkey.ErrorKey
	}{
		{
			name: "success",
			id:   "gallery-1",
			setupMock: func() {
				filter := shared.SingleFilter("gallery-1", model.FieldID, model.TableName)
				mockRepo.EXPECT().Get(gomock.Any(), filter).Return(model.Gallery{
					ID:     "gallery-1",
					Images: []string{"https://example.com/bucket/image1.jpg"},
				}, nil)
				mockRepo.EXPECT().Delete(gomock.Any(), filter).Return(nil)
				mockCache.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				mockS3.EXPECT().GetObjectNameFromURL(gomock.Any(), gomock.Any()).Return("image1.jpg").AnyTimes()
				mockS3.EXPECT().DeleteFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "error - not found",
			id:   "nonexistent",
			setupMock: func() {
				filter := shared.SingleFilter("nonexistent", model.FieldID, model.TableName)
				mockRepo.EXPECT().Get(gomock.Any(), filter).Return(model.Gallery{}, nil)
			},
			wantErr: true,
			errKey:  errkey.ErrGalleryNotFound,
		},
		{
			name: "error - get failed",
			id:   "gallery-1",
			setupMock: func() {
				filter := shared.SingleFilter("gallery-1", model.FieldID, model.TableName)
				mockRepo.EXPECT().Get(gomock.Any(), filter).Return(model.Gallery{}, errors.New("db error"))
			},
			wantErr: true,
			errKey:  errkey.ErrDatabaseQuery,
		},
		{
			name: "error - delete failed",
			id:   "gallery-1",
			setupMock: func() {
				filter := shared.SingleFilter("gallery-1", model.FieldID, model.TableName)
				mockRepo.EXPECT().Get(gomock.Any(), filter).Return(model.Gallery{
					ID:     "gallery-1",
					Images: []string{},
				}, nil)
				mockRepo.EXPECT().Delete(gomock.Any(), filter).Return(errors.New("db error"))
				mockCache.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: true,
			errKey:  errkey.ErrGalleryDeleteFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := svc.Delete(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				f, ok := err.(*failure.Failure)
				assert.True(t, ok)
				assert.Equal(t, tt.errKey, f.Key)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUploadImage(t *testing.T) {
	ctrl, mockRepo, mockCache, cfg, mockS3, mockOtel := setup(t)
	defer ctrl.Finish()

	cfg.External.S3.BucketName = "test-bucket"

	svc := service.New(mockRepo, cfg, mockCache, mockOtel, mockS3, singleflight.New())

	tests := []struct {
		name      string
		req       dto.UploadImageRequest
		setupMock func()
		wantErr   bool
		errKey    errkey.ErrorKey
	}{
		{
			name: "success",
			req: dto.UploadImageRequest{
				Image:     &multipart.FileHeader{Filename: "test-image.jpg"},
				ImageFile: nil,
			},
			setupMock: func() {
				mockS3.EXPECT().UploadFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("https://example.com/bucket/test-image.jpg", nil)
			},
			wantErr: false,
		},
		{
			name: "error - upload failed",
			req: dto.UploadImageRequest{
				Image:     &multipart.FileHeader{Filename: "test-image.jpg"},
				ImageFile: nil,
			},
			setupMock: func() {
				mockS3.EXPECT().UploadFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", errors.New("s3 error"))
			},
			wantErr: true,
			errKey:  errkey.ErrS3Upload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := svc.UploadImage(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				f, ok := err.(*failure.Failure)
				assert.True(t, ok)
				assert.Equal(t, tt.errKey, f.Key)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result.URL)
			}
		})
	}
}

func TestDeleteImagesFromS3(t *testing.T) {
	ctrl, mockRepo, mockCache, cfg, mockS3, mockOtel := setup(t)
	defer ctrl.Finish()

	cfg.External.S3.BucketName = "test-bucket"

	svc := service.New(mockRepo, cfg, mockCache, mockOtel, mockS3, singleflight.New())

	tests := []struct {
		name      string
		req       dto.DeleteImagesRequest
		setupMock func()
		wantErr   bool
		errKey    errkey.ErrorKey
	}{
		{
			name: "success",
			req: dto.DeleteImagesRequest{
				ImageURLs: []string{"https://example.com/bucket/image1.jpg"},
			},
			setupMock: func() {
				mockS3.EXPECT().GetObjectNameFromURL(gomock.Any(), "https://example.com/bucket/image1.jpg").Return("image1.jpg")
				mockS3.EXPECT().DeleteFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - delete failed",
			req: dto.DeleteImagesRequest{
				ImageURLs: []string{"https://example.com/bucket/image1.jpg"},
			},
			setupMock: func() {
				mockS3.EXPECT().GetObjectNameFromURL(gomock.Any(), "https://example.com/bucket/image1.jpg").Return("image1.jpg")
				mockS3.EXPECT().DeleteFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("s3 error"))
			},
			wantErr: true,
			errKey:  errkey.ErrS3Delete,
		},
		{
			name: "success - invalid URL (empty object name)",
			req: dto.DeleteImagesRequest{
				ImageURLs: []string{"https://invalid.com/image.jpg"},
			},
			setupMock: func() {
				mockS3.EXPECT().GetObjectNameFromURL(gomock.Any(), "https://invalid.com/image.jpg").Return("")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := svc.DeleteImagesFromS3(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				f, ok := err.(*failure.Failure)
				assert.True(t, ok)
				assert.Equal(t, tt.errKey, f.Key)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
