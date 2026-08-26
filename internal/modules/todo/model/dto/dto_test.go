package dto_test

import (
	"testing"
	"time"

	"github.com/savioruz/oil/internal/modules/todo/model"
	"github.com/savioruz/oil/internal/modules/todo/model/dto"
	gModel "github.com/savioruz/oil/shared/model"

	"github.com/stretchr/testify/assert"
)

func TestCreateTodoRequest_ToModel(t *testing.T) {
	tests := []struct {
		name     string
		req      dto.CreateTodoRequest
		userID   string
		expected model.Todo
	}{
		{
			name: "with all fields",
			req: dto.CreateTodoRequest{
				Title:       "Test Todo",
				Description: "Test Description",
			},
			userID: "test-user-id",
			expected: model.Todo{
				Title:       "Test Todo",
				Description: "Test Description",
				Completed:   false,
			},
		},
		{
			name: "with empty title",
			req: dto.CreateTodoRequest{
				Title:       "",
				Description: "Test Description",
			},
			userID: "test-user-id",
			expected: model.Todo{
				Title:       "",
				Description: "Test Description",
				Completed:   false,
			},
		},
		{
			name: "with empty description",
			req: dto.CreateTodoRequest{
				Title:       "Test Todo",
				Description: "",
			},
			userID: "test-user-id",
			expected: model.Todo{
				Title:       "Test Todo",
				Description: "",
				Completed:   false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.req.ToModel(tt.userID)

			assert.NotEmpty(t, result.ID)
			assert.Equal(t, tt.expected.Title, result.Title)
			assert.Equal(t, tt.expected.Description, result.Description)
			assert.Equal(t, tt.expected.Completed, result.Completed)
			assert.Equal(t, tt.userID, result.CreatedBy)
			assert.Equal(t, tt.userID, result.ModifiedBy)
			assert.False(t, result.CreatedAt.IsZero())
			assert.False(t, result.ModifiedAt.IsZero())
		})
	}
}

func TestCreateTodoRequest_ToModelWithFullFields(t *testing.T) {
	tests := []struct {
		name     string
		req      dto.CreateTodoRequest
		userID   string
		expected model.Todo
	}{
		{
			name: "with completed true",
			req: dto.CreateTodoRequest{
				Title:       "Test Todo",
				Description: "Test Description",
				Completed:   boolPtr(true),
			},
			userID: "test-user-id",
			expected: model.Todo{
				Title:       "Test Todo",
				Description: "Test Description",
				Completed:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.req.ToModelWithFullFields(tt.userID)

			assert.NotEmpty(t, result.ID)
			assert.Equal(t, tt.expected.Title, result.Title)
			assert.Equal(t, tt.expected.Description, result.Description)
			assert.Equal(t, tt.expected.Completed, result.Completed)
		})
	}
}

func TestTodoResponse_FromModel(t *testing.T) {
	tests := []struct {
		name     string
		model    model.Todo
		expected dto.TodoResponse
	}{
		{
			name: "with all fields",
			model: model.Todo{
				ID:          "test-id",
				Title:       "Test Todo",
				Description: "Test Description",
				Completed:   true,
				Metadata: gModel.Metadata{
					CreatedAt:  time.Now(),
					ModifiedAt: time.Now(),
					CreatedBy:  "test-user",
					ModifiedBy: "test-user",
				},
			},
			expected: dto.TodoResponse{
				ID:          "test-id",
				Title:       "Test Todo",
				Description: "Test Description",
				Completed:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result dto.TodoResponse
			result.FromModel(tt.model)

			assert.Equal(t, tt.expected.ID, result.ID)
			assert.Equal(t, tt.expected.Title, result.Title)
			assert.Equal(t, tt.expected.Description, result.Description)
			assert.Equal(t, tt.expected.Completed, result.Completed)
		})
	}
}

func TestGetTodosResponse_FromModels(t *testing.T) {
	tests := []struct {
		name      string
		models    []model.Todo
		totalData int
		limit     int
		expected  dto.GetTodosResponse
	}{
		{
			name: "with multiple models",
			models: []model.Todo{
				{ID: "1", Title: "Todo 1", Completed: false},
				{ID: "2", Title: "Todo 2", Completed: true},
			},
			totalData: 15,
			limit:     10,
			expected: dto.GetTodosResponse{
				TotalPage: 2,
				TotalData: 15,
			},
		},
		{
			name:      "with empty models",
			models:    []model.Todo{},
			totalData: 0,
			limit:     10,
			expected: dto.GetTodosResponse{
				TotalPage: 0,
				TotalData: 0,
				Todos:     []dto.TodoResponse{},
			},
		},
		{
			name: "with exact division",
			models: []model.Todo{
				{ID: "1", Title: "Todo 1"},
				{ID: "2", Title: "Todo 2"},
			},
			totalData: 20,
			limit:     10,
			expected: dto.GetTodosResponse{
				TotalPage: 2,
				TotalData: 20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result dto.GetTodosResponse
			result.FromModels(tt.models, tt.totalData, tt.limit)

			assert.Equal(t, tt.expected.TotalData, result.TotalData)
			assert.Equal(t, tt.expected.TotalPage, result.TotalPage)
			assert.Len(t, result.Todos, len(tt.models))
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
