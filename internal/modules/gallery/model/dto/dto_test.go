package dto_test

import (
	"testing"
	"time"

	"github.com/savioruz/oil/internal/modules/gallery/model"
	"github.com/savioruz/oil/internal/modules/gallery/model/dto"
	gModel "github.com/savioruz/oil/shared/model"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestCreateGalleryRequest_ToModel(t *testing.T) {
	tests := []struct {
		name     string
		req      dto.CreateGalleryRequest
		userID   string
		expected model.Gallery
	}{
		{
			name: "with all fields",
			req: dto.CreateGalleryRequest{
				Title:       "Test Gallery",
				Description: "Test Description",
				Images:      []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg"},
			},
			userID: "test-user-id",
			expected: model.Gallery{
				Title:       "Test Gallery",
				Description: "Test Description",
				Images:      []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg"},
			},
		},
		{
			name: "with empty description",
			req: dto.CreateGalleryRequest{
				Title:  "Test Gallery",
				Images: []string{"https://example.com/image1.jpg"},
			},
			userID: "test-user-id",
			expected: model.Gallery{
				Title:  "Test Gallery",
				Images: []string{"https://example.com/image1.jpg"},
			},
		},
		{
			name: "with empty title",
			req: dto.CreateGalleryRequest{
				Description: "Test Description",
				Images:      []string{"https://example.com/image1.jpg"},
			},
			userID: "test-user-id",
			expected: model.Gallery{
				Description: "Test Description",
				Images:      []string{"https://example.com/image1.jpg"},
			},
		},
		{
			name: "with multiple images",
			req: dto.CreateGalleryRequest{
				Title:       "Photo Gallery",
				Description: "Vacation photos",
				Images:      []string{"img1.jpg", "img2.jpg", "img3.jpg", "img4.jpg"},
			},
			userID: "user-123",
			expected: model.Gallery{
				Title:       "Photo Gallery",
				Description: "Vacation photos",
				Images:      []string{"img1.jpg", "img2.jpg", "img3.jpg", "img4.jpg"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.req.ToModel(tt.userID)

			assert.NotEmpty(t, result.ID)
			assert.Equal(t, tt.expected.Title, result.Title)
			assert.Equal(t, tt.expected.Description, result.Description)
			assert.Equal(t, tt.expected.Images, result.Images)
			assert.Equal(t, tt.userID, result.Metadata.CreatedBy)
			assert.Equal(t, tt.userID, result.Metadata.ModifiedBy)
			assert.False(t, result.Metadata.CreatedAt.IsZero())
			assert.False(t, result.Metadata.ModifiedAt.IsZero())
		})
	}
}

func TestGalleryResponse_FromModel(t *testing.T) {
	tests := []struct {
		name     string
		model    model.Gallery
		expected dto.GalleryResponse
	}{
		{
			name: "with all fields",
			model: model.Gallery{
				ID:          "gallery-123",
				Title:       "Test Gallery",
				Description: "Test Description",
				Images:      pq.StringArray{"https://example.com/image1.jpg"},
				Metadata: gModel.Metadata{
					CreatedAt:  testTime(2024, 1, 1),
					ModifiedAt: testTime(2024, 1, 2),
					CreatedBy:  "user-123",
					ModifiedBy: "user-123",
				},
			},
			expected: dto.GalleryResponse{
				ID:          "gallery-123",
				Title:       "Test Gallery",
				Description: "Test Description",
				Images:      []string{"https://example.com/image1.jpg"},
			},
		},
		{
			name: "with empty optional fields",
			model: model.Gallery{
				ID:     "gallery-456",
				Title:  "Simple Gallery",
				Images: pq.StringArray{},
				Metadata: gModel.Metadata{
					CreatedAt:  testTime(2024, 1, 1),
					ModifiedAt: testTime(2024, 1, 1),
					CreatedBy:  "user-123",
					ModifiedBy: "user-123",
				},
			},
			expected: dto.GalleryResponse{
				ID:     "gallery-456",
				Title:  "Simple Gallery",
				Images: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result dto.GalleryResponse
			result.FromModel(tt.model)

			assert.Equal(t, tt.expected.ID, result.ID)
			assert.Equal(t, tt.expected.Title, result.Title)
			assert.Equal(t, tt.expected.Description, result.Description)
			assert.Equal(t, tt.expected.Images, result.Images)
		})
	}
}

func TestGetGalleriesResponse_FromModels(t *testing.T) {
	tests := []struct {
		name      string
		models    []model.Gallery
		totalData int
		limit     int
		expected  dto.GetGalleriesResponse
	}{
		{
			name: "with multiple models",
			models: []model.Gallery{
				{ID: "1", Title: "Gallery 1", Images: pq.StringArray{"img1.jpg"}},
				{ID: "2", Title: "Gallery 2", Images: pq.StringArray{"img2.jpg"}},
			},
			totalData: 2,
			limit:     10,
			expected: dto.GetGalleriesResponse{
				TotalPage: 1,
				TotalData: 2,
			},
		},
		{
			name:      "with empty models",
			models:    []model.Gallery{},
			totalData: 0,
			limit:     10,
			expected: dto.GetGalleriesResponse{
				TotalPage: 0,
				TotalData: 0,
				Galleries: []dto.GalleryResponse{},
			},
		},
		{
			name: "with pagination",
			models: []model.Gallery{
				{ID: "1", Title: "Gallery 1", Images: pq.StringArray{"img1.jpg"}},
			},
			totalData: 25,
			limit:     10,
			expected: dto.GetGalleriesResponse{
				TotalPage: 3,
				TotalData: 25,
			},
		},
		{
			name: "with exact division",
			models: []model.Gallery{
				{ID: "1", Title: "Gallery 1"},
				{ID: "2", Title: "Gallery 2"},
			},
			totalData: 20,
			limit:     10,
			expected: dto.GetGalleriesResponse{
				TotalPage: 2,
				TotalData: 20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result dto.GetGalleriesResponse
			result.FromModels(tt.models, tt.totalData, tt.limit)

			assert.Equal(t, tt.expected.TotalData, result.TotalData)
			assert.Equal(t, tt.expected.TotalPage, result.TotalPage)
			assert.Len(t, result.Galleries, len(tt.models))
		})
	}
}

func TestUploadImageResponse_FromModel(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		fileName string
		expected dto.UploadImageResponse
	}{
		{
			name:     "with valid URL",
			url:      "https://example.com/bucket/test-image.jpg",
			fileName: "test-image.jpg",
			expected: dto.UploadImageResponse{
				URL:      "https://example.com/bucket/test-image.jpg",
				FileName: "test-image.jpg",
			},
		},
		{
			name:     "with different file extension",
			url:      "https://example.com/bucket/photo.png",
			fileName: "photo.png",
			expected: dto.UploadImageResponse{
				URL:      "https://example.com/bucket/photo.png",
				FileName: "photo.png",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result dto.UploadImageResponse
			result.FromModel(tt.url, tt.fileName)

			assert.Equal(t, tt.expected.URL, result.URL)
			assert.Equal(t, tt.expected.FileName, result.FileName)
		})
	}
}

func testTime(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}
