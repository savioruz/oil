package dto

import (
	"github.com/savioruz/oil/internal/modules/userprofile/model"
	"github.com/savioruz/oil/shared"
	"github.com/savioruz/oil/shared/constant"
	gDto "github.com/savioruz/oil/shared/dto"
	gModel "github.com/savioruz/oil/shared/model"
	"github.com/savioruz/oil/shared/timezone"

	"github.com/google/uuid"
)

type CreateUserprofileRequest struct {
	AuthUserID string `json:"auth_user_id" validate:"required"`
	Email      string `json:"email"        validate:"required,email"`
	Role       string `json:"role"         validate:"omitempty,oneof=user admin"`
	Name       string `json:"name"         validate:"omitempty,max=255"`
	Image      string `json:"image"        validate:"omitempty,max=500"`
}

func (r *CreateUserprofileRequest) ToModel() model.Userprofile {
	role := r.Role
	if role == "" {
		role = constant.RoleUser
	}

	now := timezone.NowUTC()

	return model.Userprofile{
		ID:         uuid.NewString(),
		AuthUserID: r.AuthUserID,
		Email:      r.Email,
		Role:       role,
		Name:       r.Name,
		Image:      r.Image,
		Active:     true,
		Metadata: gModel.Metadata{
			CreatedAt:  now,
			ModifiedAt: now,
			CreatedBy:  r.AuthUserID,
			ModifiedBy: r.AuthUserID,
		},
	}
}

type UserprofileResponse struct {
	ID         string `json:"id"`
	AuthUserID string `json:"auth_user_id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	Name       string `json:"name,omitempty"`
	Image      string `json:"image,omitempty"`
	Active     bool   `json:"active"`
	gDto.Metadata
}

func (r *UserprofileResponse) FromModel(m model.Userprofile) {
	r.ID = m.ID
	r.AuthUserID = m.AuthUserID
	r.Email = m.Email
	r.Role = m.Role
	r.Name = m.Name
	r.Image = m.Image
	r.Active = m.Active
	r.Metadata.FromModel(m.Metadata)
}

type GetUserprofilesResponse struct {
	Userprofiles []UserprofileResponse `json:"userprofiles"`
	TotalPage    int                   `json:"total_page"`
	TotalData    int                   `json:"total_data"`
}

func (r *GetUserprofilesResponse) FromModels(models []model.Userprofile, totalData, limit int) {
	r.TotalData = totalData
	r.TotalPage = shared.CalculateTotalPage(totalData, limit)

	r.Userprofiles = make([]UserprofileResponse, len(models))
	for i, mod := range models {
		r.Userprofiles[i].FromModel(mod)
	}
}

type UpdateUserprofileRequest struct {
	Name       string `db:"name"         json:"name"         validate:"omitempty,max=255"`
	Image      string `db:"image"        json:"image"        validate:"omitempty,max=500"`
	AuthUserID string `db:"auth_user_id" json:"auth_user_id" validate:"omitempty,max=255"`
}

type GeneratePresignedURLRequest struct {
	FileName    string `json:"file_name"    validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
}

type GeneratePresignedURLResponse struct {
	UploadURL string `json:"upload_url"`
	FileKey   string `json:"file_key"`
}
