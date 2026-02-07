package dto_test

import (
	"testing"
	"time"

	"oil/internal/domains/todo/model"
	"oil/internal/domains/todo/model/dto"
	gModel "oil/shared/model"

	"github.com/stretchr/testify/assert"
)

func TestCreateTodoRequest_ToModel(t *testing.T) {
	req := dto.CreateTodoRequest{
		Title:       "Test Todo",
		Description: "Test Description",
	}

	userID := "test-user-id"
	model := req.ToModel(userID)

	assert.NotEmpty(t, model.ID, "expected ID to be generated")
	assert.Equal(t, req.Title, model.Title)
	assert.Equal(t, req.Description, model.Description)
	assert.False(t, model.Completed)
	assert.Equal(t, userID, model.CreatedBy)
	assert.Equal(t, userID, model.ModifiedBy)
	assert.False(t, model.CreatedAt.IsZero(), "expected CreatedAt to be set")
	assert.False(t, model.ModifiedAt.IsZero(), "expected ModifiedAt to be set")
}

func TestTodoResponse_FromModel(t *testing.T) {
	now := time.Now()
	todoModel := model.Todo{
		ID:          "test-id",
		Title:       "Test Todo",
		Description: "Test Description",
		Completed:   true,
		Metadata: gModel.Metadata{
			CreatedAt:  now,
			ModifiedAt: now,
			CreatedBy:  "test-user",
			ModifiedBy: "test-user",
		},
	}

	var response dto.TodoResponse
	response.FromModel(todoModel)

	assert.Equal(t, todoModel.ID, response.ID)
	assert.Equal(t, todoModel.Title, response.Title)
	assert.Equal(t, todoModel.Description, response.Description)
	assert.Equal(t, todoModel.Completed, response.Completed)
	assert.Equal(t, todoModel.CreatedBy, response.CreatedBy)
	assert.Equal(t, todoModel.ModifiedBy, response.ModifiedBy)
}

func TestGetTodosResponse_FromModels(t *testing.T) {
	now := time.Now()
	todos := []model.Todo{
		{
			ID:          "test-id-1",
			Title:       "Test Todo 1",
			Description: "Test Description 1",
			Completed:   false,
			Metadata: gModel.Metadata{
				CreatedAt:  now,
				ModifiedAt: now,
				CreatedBy:  "test-user",
				ModifiedBy: "test-user",
			},
		},
		{
			ID:          "test-id-2",
			Title:       "Test Todo 2",
			Description: "Test Description 2",
			Completed:   true,
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

	var response dto.GetTodosResponse
	response.FromModels(todos, totalData, limit)

	assert.Equal(t, totalData, response.TotalData)
	assert.Equal(t, 2, response.TotalPage)
	assert.Len(t, response.Todos, len(todos))

	for i, todo := range response.Todos {
		assert.Equal(t, todos[i].ID, todo.ID)
		assert.Equal(t, todos[i].Title, todo.Title)
	}
}

func TestGetTodosResponse_FromModels_EmptyList(t *testing.T) {
	var todos []model.Todo
	totalData := 0
	limit := 10

	var response dto.GetTodosResponse
	response.FromModels(todos, totalData, limit)

	assert.Equal(t, totalData, response.TotalData)
	assert.Equal(t, 1, response.TotalPage) // Function returns 1 when total is 0
	assert.Len(t, response.Todos, 0)
}
