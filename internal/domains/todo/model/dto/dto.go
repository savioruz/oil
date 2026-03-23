package dto

import (
	"oil/internal/domains/todo/model"
	"oil/shared"
	gDto "oil/shared/dto"
	gModel "oil/shared/model"
	"time"

	"github.com/google/uuid"
)

type CreateTodoRequest struct {
	Title       string `json:"title"       validate:"required,max=255"`
	Description string `json:"description" validate:"required,max=255"`
	Completed   *bool  `json:"completed"   swaggerignore:"true"        validate:"omitempty"`
}

func (c *CreateTodoRequest) ToModel(user string) model.Todo {
	now := time.Now()

	return model.Todo{
		ID:          uuid.NewString(),
		Title:       c.Title,
		Description: c.Description,
		Completed:   false,
		Metadata: gModel.Metadata{
			CreatedAt:  now,
			ModifiedAt: now,
			CreatedBy:  user,
			ModifiedBy: user,
		},
	}
}

func (c *CreateTodoRequest) ToModelWithFullFields(user string) model.Todo {
	now := time.Now()

	completed := false
	if c.Completed != nil {
		completed = *c.Completed
	}

	return model.Todo{
		ID:          uuid.NewString(),
		Title:       c.Title,
		Description: c.Description,
		Completed:   completed,
		Metadata: gModel.Metadata{
			CreatedAt:  now,
			ModifiedAt: now,
			CreatedBy:  user,
			ModifiedBy: user,
		},
	}
}

type UpdateTodoRequest struct {
	Title       string `db:"title"       json:"title"       validate:"omitempty,max=255"`
	Description string `db:"description" json:"description" validate:"omitempty,max=255"`
	Completed   *bool  `db:"completed"   json:"completed"   validate:"omitempty"`
}

type TodoResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
	gDto.Metadata
}

func (r *TodoResponse) FromModel(model model.Todo) {
	r.ID = model.ID
	r.Title = model.Title
	r.Description = model.Description
	r.Completed = model.Completed
	r.Metadata.FromModel(model.Metadata)
}

type GetTodosResponse struct {
	Todos     []TodoResponse `json:"todos"`
	TotalPage int            `json:"total_page"`
	TotalData int            `json:"total_data"`
}

func (r *GetTodosResponse) FromModels(models []model.Todo, totalData, limit int) {
	r.TotalData = totalData
	r.TotalPage = shared.CalculateTotalPage(totalData, limit)

	r.Todos = make([]TodoResponse, len(models))
	for i, mod := range models {
		r.Todos[i].FromModel(mod)
	}
}
