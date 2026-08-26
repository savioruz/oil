package model

import "oil/shared/model"

const (
	TableName  = "todos"
	EntityName = "todo"

	FieldID          = "id"
	FieldTitle       = "title"
	FieldDescription = "description"
	FieldCompleted   = "completed"
)

type Todo struct {
	ID          string `db:"id"`
	Title       string `db:"title"`
	Description string `db:"description"`
	Completed   bool   `db:"completed"`
	model.Metadata
}
