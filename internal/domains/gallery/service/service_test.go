package service_test

import (
	"context"
	"errors"
	"mime/multipart"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"oil/config"
	"oil/infras/otel/mocks"
	s3Mocks "oil/infras/s3/mocks"
	galleryMocks "oil/internal/domains/gallery/mocks"
	"oil/internal/domains/gallery/model"
	"oil/internal/domains/gallery/model/dto"
	"oil/internal/domains/gallery/service"
	cacheMocks "oil/shared/cache/mocks"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	gModel "oil/shared/model"
)

func TestGalleryService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := galleryMocks.NewMockGallery(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := mocks.NewOtel()
	mockS3 := s3Mocks.NewMockS3(ctrl)

	cfg := &config.Config{}
	cfg.Cache.TTL = 3600

	svc := service.New(mockRepo, cfg, mockCache, mockOtel, mockS3)

	tests := []struct {
		name      string
		req       dto.CreateGalleryRequest
		setupMock func()
		wantErr   bool
	}{
		{
			name: "successful creation",
			req: dto.CreateGalleryRequest{
				Title:       "Test Gallery",
				Description: "Test Description",
				Images:      []string{"https://example.com/image1.jpg"},
			},
			setupMock: func() {
				mockRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					Return(nil)

				mockCache.EXPECT().
					Clear(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "repository error",
			req: dto.CreateGalleryRequest{
				Title:       "Test Gallery",
				Description: "Test Description",
				Images:      []string{"https://example.com/image1.jpg"},
			},
			setupMock: func() {
				mockRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
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

func TestGalleryService_GetAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := galleryMocks.NewMockGallery(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := mocks.NewOtel()
	mockS3 := s3Mocks.NewMockS3(ctrl)

	cfg := &config.Config{}
	cfg.Cache.TTL = 3600

	svc := service.New(mockRepo, cfg, mockCache, mockOtel, mockS3)

	tests := []struct {
		name       string
		params     gDto.QueryParams
		filter     gDto.FilterGroup
		setupMock  func()
		wantErr    bool
		wantResult dto.GetGalleriesResponse
	}{
		{
			name: "successful get all",
			params: gDto.QueryParams{
				Limit: 10,
				Page:  1,
			},
			filter: gDto.FilterGroup{},
			setupMock: func() {
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("cache miss"))

				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("cache miss"))

				mockRepo.EXPECT().
					Count(gomock.Any(), gomock.Any()).
					Return(1, nil)

				now := time.Now()

				galleries := []model.Gallery{
					{
						ID:          "test-id",
						Title:       "Test Gallery",
						Description: "Test Description",
						Images:      []string{"https://example.com/image1.jpg"},
						Metadata: gModel.Metadata{
							CreatedAt:  now,
							ModifiedAt: now,
							CreatedBy:  "test-user",
							ModifiedBy: "test-user",
						},
					},
				}

				mockRepo.EXPECT().
					GetAll(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(galleries, nil)

				mockCache.EXPECT().
					Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			wantErr: false,
			wantResult: dto.GetGalleriesResponse{
				TotalData: 1,
				TotalPage: 1,
			},
		},
		{
			name: "count error",
			params: gDto.QueryParams{
				Limit: 10,
				Page:  1,
			},
			filter: gDto.FilterGroup{},
			setupMock: func() {
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("cache miss"))

				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("cache miss"))

				mockRepo.EXPECT().
					Count(gomock.Any(), gomock.Any()).
					Return(0, errors.New("count error"))
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

func TestGalleryService_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := galleryMocks.NewMockGallery(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := mocks.NewOtel()
	mockS3 := s3Mocks.NewMockS3(ctrl)

	cfg := &config.Config{}
	cfg.Cache.TTL = 3600

	svc := service.New(mockRepo, cfg, mockCache, mockOtel, mockS3)

	now := time.Now()

	gallery := model.Gallery{
		ID:          "test-id",
		Title:       "Test Gallery",
		Description: "Test Description",
		Images:      []string{"https://example.com/image1.jpg"},
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
			name: "cache hit",
			id:   "test-id",
			setupMock: func() {
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
			wantID:  "",
		},
		{
			name: "cache miss, successful get from db",
			id:   "test-id",
			setupMock: func() {
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("cache miss"))

				mockRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(gallery, nil)

				mockCache.EXPECT().
					Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			wantErr: false,
			wantID:  "test-id",
		},
		{
			name: "gallery not found",
			id:   "nonexistent-id",
			setupMock: func() {
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("cache miss"))

				mockRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(model.Gallery{}, nil)
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

func TestGalleryService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := galleryMocks.NewMockGallery(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := mocks.NewOtel()
	mockS3 := s3Mocks.NewMockS3(ctrl)

	cfg := &config.Config{}
	cfg.Cache.TTL = 3600

	svc := service.New(mockRepo, cfg, mockCache, mockOtel, mockS3)

	tests := []struct {
		name      string
		req       dto.UpdateGalleryRequest
		id        string
		setupMock func()
		wantErr   bool
	}{
		{
			name: "successful update",
			req: dto.UpdateGalleryRequest{
				Title:       "Updated Title",
				Description: "Updated Description",
			},
			id: "test-id",
			setupMock: func() {
				mockRepo.EXPECT().
					Exist(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockRepo.EXPECT().
					Update(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)

				mockCache.EXPECT().
					Delete(gomock.Any(), gomock.Any()).
					Return(nil)

				mockCache.EXPECT().
					Clear(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "gallery not found",
			req: dto.UpdateGalleryRequest{
				Title: "Updated Title",
			},
			id: "nonexistent-id",
			setupMock: func() {
				mockRepo.EXPECT().
					Exist(gomock.Any(), gomock.Any()).
					Return(false, nil)
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

func TestGalleryService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := galleryMocks.NewMockGallery(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := mocks.NewOtel()
	mockS3 := s3Mocks.NewMockS3(ctrl)

	cfg := &config.Config{}
	cfg.Cache.TTL = 3600
	cfg.External.S3.BucketName = "test-bucket"

	svc := service.New(mockRepo, cfg, mockCache, mockOtel, mockS3)

	tests := []struct {
		name      string
		id        string
		setupMock func()
		wantErr   bool
	}{
		{
			name: "successful deletion with images",
			id:   "test-id",
			setupMock: func() {
				gallery := model.Gallery{
					ID:     "test-id",
					Images: []string{"https://example.com/bucket/image1.jpg"},
				}

				mockRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(gallery, nil)

				mockRepo.EXPECT().
					Delete(gomock.Any(), gomock.Any()).
					Return(nil)

				mockCache.EXPECT().
					Delete(gomock.Any(), gomock.Any()).
					Return(nil)

				mockCache.EXPECT().
					Clear(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()

				mockS3.EXPECT().
					GetObjectNameFromURL(gomock.Any(), gomock.Any()).
					Return("image1.jpg")

				mockS3.EXPECT().
					DeleteFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "gallery not found",
			id:   "nonexistent-id",
			setupMock: func() {
				mockRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(model.Gallery{}, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.Background()
			err := svc.Delete(ctx, tt.id)

			time.Sleep(50 * time.Millisecond)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGalleryService_UploadImage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := galleryMocks.NewMockGallery(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := mocks.NewOtel()
	mockS3 := s3Mocks.NewMockS3(ctrl)

	cfg := &config.Config{}
	cfg.Cache.TTL = 3600
	cfg.External.S3.BucketName = "test-bucket"

	svc := service.New(mockRepo, cfg, mockCache, mockOtel, mockS3)

	tests := []struct {
		name      string
		req       dto.UploadImageRequest
		setupMock func()
		wantErr   bool
	}{
		{
			name: "successful upload",
			req: dto.UploadImageRequest{
				Image: &multipart.FileHeader{
					Filename: "test-image.jpg",
				},
				ImageFile: nil,
			},
			setupMock: func() {
				mockS3.EXPECT().
					UploadFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return("https://example.com/bucket/test-image.jpg", nil)
			},
			wantErr: false,
		},
		{
			name: "upload error",
			req: dto.UploadImageRequest{
				Image: &multipart.FileHeader{
					Filename: "test-image.jpg",
				},
				ImageFile: nil,
			},
			setupMock: func() {
				mockS3.EXPECT().
					UploadFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return("", errors.New("s3 upload error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.Background()
			result, err := svc.UploadImage(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result.URL)
			}
		})
	}
}

func TestGalleryService_DeleteImagesFromS3(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := galleryMocks.NewMockGallery(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := mocks.NewOtel()
	mockS3 := s3Mocks.NewMockS3(ctrl)

	cfg := &config.Config{}
	cfg.Cache.TTL = 3600
	cfg.External.S3.BucketName = "test-bucket"

	svc := service.New(mockRepo, cfg, mockCache, mockOtel, mockS3)

	tests := []struct {
		name      string
		req       dto.DeleteImagesRequest
		setupMock func()
		wantErr   bool
	}{
		{
			name: "successful deletion",
			req: dto.DeleteImagesRequest{
				ImageURLs: []string{"https://example.com/bucket/image1.jpg"},
			},
			setupMock: func() {
				mockS3.EXPECT().
					GetObjectNameFromURL(gomock.Any(), gomock.Any()).
					Return("image1.jpg")

				mockS3.EXPECT().
					DeleteFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "delete error",
			req: dto.DeleteImagesRequest{
				ImageURLs: []string{"https://example.com/bucket/image1.jpg"},
			},
			setupMock: func() {
				mockS3.EXPECT().
					GetObjectNameFromURL(gomock.Any(), gomock.Any()).
					Return("image1.jpg")

				mockS3.EXPECT().
					DeleteFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("s3 delete error"))
			},
			wantErr: true,
		},
		{
			name: "invalid URL - empty object name",
			req: dto.DeleteImagesRequest{
				ImageURLs: []string{"https://invalid.com/image.jpg"},
			},
			setupMock: func() {
				mockS3.EXPECT().
					GetObjectNameFromURL(gomock.Any(), gomock.Any()).
					Return("")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.Background()
			err := svc.DeleteImagesFromS3(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
