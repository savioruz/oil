package dto_test

import (
	"testing"
	"time"

	"oil/internal/domains/gallery/model"
	"oil/internal/domains/gallery/model/dto"
	gModel "oil/shared/model"

	"github.com/stretchr/testify/assert"
)

func TestCreateGalleryRequest_ToModel(t *testing.T) {
	req := dto.CreateGalleryRequest{
		Title:       "Test Gallery",
		Description: "Test Description",
		Images:      []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg"},
	}

	userID := "test-user-id"
	model := req.ToModel(userID)

	assert.NotEmpty(t, model.ID, "expected ID to be generated")
	assert.Equal(t, req.Title, model.Title)
	assert.Equal(t, req.Description, model.Description)
	assert.Equal(t, req.Images, model.Images)
	assert.Equal(t, userID, model.CreatedBy)
	assert.Equal(t, userID, model.ModifiedBy)
	assert.False(t, model.CreatedAt.IsZero(), "expected CreatedAt to be set")
	assert.False(t, model.ModifiedAt.IsZero(), "expected ModifiedAt to be set")
}

func TestGalleryResponse_FromModel(t *testing.T) {
	now := time.Now()
	galleryModel := model.Gallery{
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

	var response dto.GalleryResponse
	response.FromModel(galleryModel)

	assert.Equal(t, galleryModel.ID, response.ID)
	assert.Equal(t, galleryModel.Title, response.Title)
	assert.Equal(t, galleryModel.Description, response.Description)
	assert.Equal(t, galleryModel.Images, response.Images)
	assert.Equal(t, galleryModel.CreatedBy, response.CreatedBy)
	assert.Equal(t, galleryModel.ModifiedBy, response.ModifiedBy)
}

func TestGetGalleriesResponse_FromModels(t *testing.T) {
	now := time.Now()
	galleries := []model.Gallery{
		{
			ID:          "test-id-1",
			Title:       "Test Gallery 1",
			Description: "Test Description 1",
			Images:      []string{"https://example.com/image1.jpg"},
			Metadata: gModel.Metadata{
				CreatedAt:  now,
				ModifiedAt: now,
				CreatedBy:  "test-user",
				ModifiedBy: "test-user",
			},
		},
		{
			ID:          "test-id-2",
			Title:       "Test Gallery 2",
			Description: "Test Description 2",
			Images:      []string{"https://example.com/image2.jpg", "https://example.com/image3.jpg"},
			Metadata: gModel.Metadata{
				CreatedAt:  now,
				ModifiedAt: now,
				CreatedBy:  "test-user",
				ModifiedBy: "test-user",
			},
		},
	}

	totalData := 15
	limit := 10

	var response dto.GetGalleriesResponse
	response.FromModels(galleries, totalData, limit)

	assert.Equal(t, totalData, response.TotalData)
	assert.Equal(t, 2, response.TotalPage) // 15 items with limit 10 should give 2 pages
	assert.Len(t, response.Galleries, len(galleries))

	// Test individual gallery mapping
	for i, gallery := range response.Galleries {
		assert.Equal(t, galleries[i].ID, gallery.ID)
		assert.Equal(t, galleries[i].Title, gallery.Title)
		assert.Equal(t, galleries[i].Images, gallery.Images)
	}
}

func TestGetGalleriesResponse_FromModels_EmptyList(t *testing.T) {
	var galleries []model.Gallery
	totalData := 0
	limit := 10

	var response dto.GetGalleriesResponse
	response.FromModels(galleries, totalData, limit)

	assert.Equal(t, totalData, response.TotalData)
	assert.Equal(t, 1, response.TotalPage)
	assert.Len(t, response.Galleries, 0)
}

func TestUploadImageResponse_FromModel(t *testing.T) {
	url := "https://example.com/bucket/test-image.jpg"
	fileName := "test-image.jpg"

	var response dto.UploadImageResponse
	response.FromModel(url, fileName)

	assert.Equal(t, url, response.URL)
	assert.Equal(t, fileName, response.FileName)
}
