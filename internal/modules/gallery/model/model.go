package model

import (
	"github.com/savioruz/oil/shared/model"

	"github.com/lib/pq"
)

const (
	TableName  = "galleries"
	EntityName = "gallery"

	FieldID          = "id"
	FieldTitle       = "title"
	FieldDescription = "description"
	FieldImages      = "images"
)

type Gallery struct {
	ID          string         `db:"id"`
	Title       string         `db:"title"`
	Description string         `db:"description"`
	Images      pq.StringArray `db:"images"`
	model.Metadata
}
