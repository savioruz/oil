package dto

import (
	"github.com/savioruz/oil/internal/modules/gallery/model"
	"github.com/savioruz/oil/shared"
	gDto "github.com/savioruz/oil/shared/dto"
	gModel "github.com/savioruz/oil/shared/model"
	"github.com/savioruz/oil/shared/timezone"
	"mime/multipart"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type CreateGalleryRequest struct {
	Title       string   `json:"title"       validate:"required,min=3,max=100"`
	Description string   `json:"description"`
	Images      []string `json:"images"      validate:"required,dive,url"`
}

func (c *CreateGalleryRequest) ToModel(user string) model.Gallery {
	now := timezone.NowUTC()

	return model.Gallery{
		ID:          uuid.NewString(),
		Title:       c.Title,
		Description: c.Description,
		Images:      c.Images,
		Metadata: gModel.Metadata{
			CreatedAt:  now,
			ModifiedAt: now,
			CreatedBy:  user,
			ModifiedBy: user,
		},
	}
}

type UpdateGalleryRequest struct {
	Title       string         `db:"title"       json:"title"       validate:"omitempty,min=3,max=100"`
	Description string         `db:"description" json:"description" validate:"omitempty"`
	Images      pq.StringArray `db:"images"      json:"-"           swaggerignore:"true"               validate:"omitempty,dive,url"`
}

type GalleryResponse struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Images      []string `json:"images"`
	gDto.Metadata
}

func (r *GalleryResponse) FromModel(model model.Gallery) {
	r.ID = model.ID
	r.Title = model.Title
	r.Description = model.Description
	r.Images = model.Images
	r.Metadata.FromModel(model.Metadata)
}

type GetGalleriesResponse struct {
	Galleries []GalleryResponse `json:"galleries"`
	TotalPage int               `json:"total_page"`
	TotalData int               `json:"total_data"`
}

func (r *GetGalleriesResponse) FromModels(models []model.Gallery, totalData, limit int) {
	r.TotalData = totalData
	r.TotalPage = shared.CalculateTotalPage(totalData, limit)

	r.Galleries = make([]GalleryResponse, len(models))
	for i, m := range models {
		r.Galleries[i].FromModel(m)
	}
}

type UploadImageRequest struct {
	Image     *multipart.FileHeader `json:"image" swaggerignore:"true" validate:"required,mimetypes=image/png image/jpg image/jpeg"`
	ImageFile multipart.File        `json:"-"`
}

type UploadImageResponse struct {
	URL      string `json:"url"`
	FileName string `json:"file_name"`
}

func (r *UploadImageResponse) FromModel(url, fileName string) {
	r.URL = url
	r.FileName = fileName
}

type DeleteImagesRequest struct {
	ImageURLs []string `json:"image_urls" validate:"required,min=1,dive,url"`
}
