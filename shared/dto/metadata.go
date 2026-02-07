package dto

import (
	"oil/shared/model"
	"oil/shared/timezone"
)

type Metadata struct {
	CreatedAt  string `json:"created_at"`
	ModifiedAt string `json:"modified_at"`
	CreatedBy  string `json:"created_by"`
	ModifiedBy string `json:"modified_by"`
}

func (m *Metadata) FromModel(model model.Metadata) {
	// Ensure times are converted to UTC before formatting
	m.CreatedAt = timezone.FormatRFC3339(model.CreatedAt)
	m.ModifiedAt = timezone.FormatRFC3339(model.ModifiedAt)
	m.CreatedBy = model.CreatedBy
	m.ModifiedBy = model.ModifiedBy
}
