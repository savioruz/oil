package todo_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"oil/config"
	"oil/infras/otel/mocks"
	postgresMocks "oil/infras/postgres/mocks"
	"oil/infras/unleash"
	"oil/internal/domains/todo/model/dto"
	"oil/internal/domains/todo/repository"
	"oil/internal/domains/todo/service"
	"oil/internal/handlers/todo"
	cacheMocks "oil/shared/cache/mocks"
	"oil/shared/constant"
	"oil/transport/http/response"
)

func setup(t *testing.T, ctrl *gomock.Controller) (*httptest.Server, sqlmock.Sqlmock, *cacheMocks.MockRedisCache) {
	mux := chi.NewRouter()
	otel := mocks.NewOtel()
	cfg := &config.Config{}
	cfg.Cache.TTL = 300

	_, sqlMock, sqlConn := postgresMocks.SetupPostgresConnection(t)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)

	repo := repository.New(sqlConn, otel)
	ff, _ := unleash.New(cfg)
	svc := service.New(repo, cfg, mockCache, otel, ff)
	handler := todo.New(svc, otel)

	mux.Route("/api", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.WithValue(r.Context(), constant.ContextKeyUserID, "test-user-id")
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})
		handler.Router(r)
	})

	ts := httptest.NewServer(mux)

	return ts, sqlMock, mockCache
}

func TestCreateTodo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, sqlMock, mockCache := setup(t, ctrl)
	defer ts.Close()

	var responseMessage response.Message
	var responseErr response.Error
	var responseValidationErr response.ValidationErrors

	resetResponse := func() {
		responseMessage = response.Message{}
		responseErr = response.Error{}
		responseValidationErr = response.ValidationErrors{}
	}

	getClient := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseMessage).
			SetError(&responseErr)
	}

	getClientForValidation := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseMessage).
			SetError(&responseValidationErr)
	}

	validBody := dto.CreateTodoRequest{
		Title:       "Test Todo",
		Description: "This is a test todo item",
	}

	t.Run("Error: invalid request - missing title", func(t *testing.T) {
		defer resetResponse()

		invalidBody := map[string]interface{}{
			"description": "This is a test todo item",
		}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Post("/todos")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: invalid request - missing description", func(t *testing.T) {
		defer resetResponse()

		invalidBody := map[string]interface{}{
			"title": "Test Todo",
		}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Post("/todos")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: invalid request - title too long", func(t *testing.T) {
		defer resetResponse()

		longTitle := ""
		for i := 0; i < 256; i++ {
			longTitle += "a"
		}

		invalidBody := map[string]interface{}{
			"title":       longTitle,
			"description": "Valid description",
		}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Post("/todos")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: failed to insert todo", func(t *testing.T) {
		defer resetResponse()

		sqlMock.ExpectExec(regexp.QuoteMeta("INSERT INTO todos (id, title, description, completed, created_at, modified_at, created_by, modified_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")).
			WillReturnError(fmt.Errorf("insert error"))

		resp, err := getClient().
			SetBody(validBody).
			Post("/todos")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Success: create todo", func(t *testing.T) {
		defer resetResponse()

		sqlMock.ExpectExec(regexp.QuoteMeta("INSERT INTO todos (id, title, description, completed, created_at, modified_at, created_by, modified_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().
			SetBody(validBody).
			Post("/todos")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusCreated, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseMessage.Message)
		assert.Equal(t, "Todo created successfully", *responseMessage.Message)
	})
}

func TestGetTodos(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, sqlMock, mockCache := setup(t, ctrl)
	defer ts.Close()

	var responseData response.Data[dto.GetTodosResponse]
	var responseErr response.Error

	resetResponse := func() {
		responseData = response.Data[dto.GetTodosResponse]{}
		responseErr = response.Error{}
	}

	getClient := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseData).
			SetError(&responseErr)
	}

	t.Run("Error: failed to count todos", func(t *testing.T) {
		defer resetResponse()

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT COUNT(todos.id) FROM todos")).
			ExpectQuery().
			WillReturnError(fmt.Errorf("count error"))

		resp, err := getClient().Get("/todos")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: failed to get todos", func(t *testing.T) {
		defer resetResponse()

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT COUNT(todos.id) FROM todos")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		sqlMock.ExpectPrepare("SELECT .* FROM todos").
			ExpectQuery().
			WillReturnError(fmt.Errorf("query error"))

		resp, err := getClient().Get("/todos")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Success: get all todos", func(t *testing.T) {
		defer resetResponse()

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT COUNT(todos.id) FROM todos")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		now := time.Now()
		sqlMock.ExpectPrepare("SELECT .* FROM todos").
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "completed", "created_at", "modified_at", "created_by", "modified_by"}).
				AddRow("id1", "Todo 1", "Description 1", false, now, now, "user1", "user1").
				AddRow("id2", "Todo 2", "Description 2", true, now, now, "user1", "user1"))

		mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().Get("/todos")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseData.Data)
		assert.Equal(t, 2, len(responseData.Data.Todos))
		assert.Equal(t, "Todo 1", responseData.Data.Todos[0].Title)
		assert.False(t, responseData.Data.Todos[0].Completed)
		assert.Equal(t, "Todo 2", responseData.Data.Todos[1].Title)
		assert.True(t, responseData.Data.Todos[1].Completed)
	})

	t.Run("Success: get todos with pagination", func(t *testing.T) {
		defer resetResponse()

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT COUNT(todos.id) FROM todos")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

		now := time.Now()
		sqlMock.ExpectPrepare("SELECT .* FROM todos").
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "completed", "created_at", "modified_at", "created_by", "modified_by"}).
				AddRow("id1", "Todo 1", "Description 1", false, now, now, "user1", "user1"))

		mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().
			SetQueryParam("page", "2").
			SetQueryParam("limit", "10").
			Get("/todos")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseData.Data)
		assert.Equal(t, 25, responseData.Data.TotalData)
		assert.Equal(t, 3, responseData.Data.TotalPage)
	})

	t.Run("Success: filter by title", func(t *testing.T) {
		defer resetResponse()

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT COUNT(todos.id) FROM todos")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		now := time.Now()
		sqlMock.ExpectPrepare("SELECT .* FROM todos").
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "completed", "created_at", "modified_at", "created_by", "modified_by"}).
				AddRow("id1", "Shopping", "Buy groceries", false, now, now, "user1", "user1"))

		mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().
			SetQueryParam("title", "Shopping").
			Get("/todos")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseData.Data)
		assert.Equal(t, 1, len(responseData.Data.Todos))
		assert.Contains(t, responseData.Data.Todos[0].Title, "Shopping")
	})

	t.Run("Success: filter by completed status", func(t *testing.T) {
		defer resetResponse()

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT COUNT(todos.id) FROM todos")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		now := time.Now()
		sqlMock.ExpectPrepare("SELECT .* FROM todos").
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "completed", "created_at", "modified_at", "created_by", "modified_by"}).
				AddRow("id1", "Completed Todo", "This is done", true, now, now, "user1", "user1"))

		mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().
			SetQueryParam("completed", "true").
			Get("/todos")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseData.Data)
		assert.Equal(t, 1, len(responseData.Data.Todos))
		assert.True(t, responseData.Data.Todos[0].Completed)
	})

	t.Run("Success: cache hit", func(t *testing.T) {
		defer resetResponse()

		cachedResponse := dto.GetTodosResponse{
			Todos: []dto.TodoResponse{
				{
					ID:          "id1",
					Title:       "Cached Todo",
					Description: "From cache",
					Completed:   true,
				},
			},
			TotalPage: 1,
			TotalData: 1,
		}

		// Mock cache hit
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, key string, dest interface{}) error {
				*dest.(*dto.GetTodosResponse) = cachedResponse
				return nil
			})

		resp, err := getClient().Get("/todos")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.Equal(t, 1, len(responseData.Data.Todos))
		assert.Equal(t, "Cached Todo", responseData.Data.Todos[0].Title)
	})
}

func TestGetTodoByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, sqlMock, mockCache := setup(t, ctrl)
	defer ts.Close()

	var responseData response.Data[dto.TodoResponse]
	var responseErr response.Error

	resetResponse := func() {
		responseData = response.Data[dto.TodoResponse]{}
		responseErr = response.Error{}
	}

	getClient := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseData).
			SetError(&responseErr)
	}

	t.Run("Error: todo not found", func(t *testing.T) {
		defer resetResponse()

		todoID := "non-existent-id"

		// Mock cache miss
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		// Mock get query - no rows returned (uses PrepareNamed)
		sqlMock.ExpectPrepare("SELECT .* FROM todos WHERE \\(todos\\.id = \\?\\)").
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "completed", "created_at", "modified_at", "created_by", "modified_by"}))

		resp, err := getClient().Get("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusNotFound, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: database query error", func(t *testing.T) {
		defer resetResponse()

		todoID := "test-id"

		// Mock cache miss
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		// Mock get query failure (uses PrepareNamed)
		sqlMock.ExpectPrepare("SELECT .* FROM todos WHERE \\(todos\\.id = \\?\\)").
			ExpectQuery().
			WillReturnError(fmt.Errorf("database error"))

		resp, err := getClient().Get("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Success: get todo by ID", func(t *testing.T) {
		defer resetResponse()

		todoID := "test-todo-id"

		// Mock cache miss
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		// Mock successful get (uses PrepareNamed)
		now := time.Now()
		sqlMock.ExpectPrepare("SELECT .* FROM todos WHERE \\(todos\\.id = \\?\\)").
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "completed", "created_at", "modified_at", "created_by", "modified_by"}).
				AddRow(todoID, "Test Todo", "Test Description", false, now, now, "user1", "user1"))

		// Mock cache save
		mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().Get("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseData.Data)
		if responseData.Data != nil {
			assert.Equal(t, todoID, responseData.Data.ID)
			assert.Equal(t, "Test Todo", responseData.Data.Title)
			assert.Equal(t, "Test Description", responseData.Data.Description)
			assert.False(t, responseData.Data.Completed)
		}
	})

	t.Run("Success: cache hit", func(t *testing.T) {
		defer resetResponse()

		todoID := "cached-todo-id"

		cachedTodo := dto.TodoResponse{
			ID:          todoID,
			Title:       "Cached Todo",
			Description: "From cache",
			Completed:   true,
		}

		// Mock cache hit
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, key string, dest interface{}) error {
				*dest.(*dto.TodoResponse) = cachedTodo
				return nil
			})

		resp, err := getClient().Get("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseData.Data)
		if responseData.Data != nil {
			assert.Equal(t, "Cached Todo", responseData.Data.Title)
			assert.True(t, responseData.Data.Completed)
		}
	})
}

func TestUpdateTodo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, sqlMock, mockCache := setup(t, ctrl)
	defer ts.Close()

	var responseMessage response.Message
	var responseErr response.Error
	var responseValidationErr response.ValidationErrors

	resetResponse := func() {
		responseMessage = response.Message{}
		responseErr = response.Error{}
		responseValidationErr = response.ValidationErrors{}
	}

	getClient := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseMessage).
			SetError(&responseErr)
	}

	getClientForValidation := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseMessage).
			SetError(&responseValidationErr)
	}

	completed := true
	validBody := dto.UpdateTodoRequest{
		Title:     "Updated Title",
		Completed: &completed,
	}

	t.Run("Error: invalid request - title too long", func(t *testing.T) {
		defer resetResponse()

		longTitle := ""
		for i := 0; i < 256; i++ {
			longTitle += "a"
		}

		invalidBody := map[string]interface{}{
			"title": longTitle,
		}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Patch("/todos/test-id")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: todo not found", func(t *testing.T) {
		defer resetResponse()

		todoID := "non-existent-id"

		// Mock exists check - doesn't exist (uses PrepareNamed)
		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM todos WHERE (todos.id = ?) )")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		resp, err := getClient().
			SetBody(validBody).
			Patch("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusNotFound, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: failed to check existence", func(t *testing.T) {
		defer resetResponse()

		todoID := "test-id"

		// Mock exists check failure (uses PrepareNamed)
		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM todos WHERE (todos.id = ?) )")).
			ExpectQuery().
			WillReturnError(fmt.Errorf("db error"))

		resp, err := getClient().
			SetBody(validBody).
			Patch("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: failed to update todo", func(t *testing.T) {
		defer resetResponse()

		todoID := "test-id"

		// Mock exists check - exists (uses PrepareNamed)
		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM todos WHERE (todos.id = ?) )")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock update failure (uses NamedExec directly, no Prepare)
		sqlMock.ExpectExec(regexp.QuoteMeta("UPDATE todos SET")).
			WillReturnError(fmt.Errorf("update error"))

		resp, err := getClient().
			SetBody(validBody).
			Patch("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Success: update todo", func(t *testing.T) {
		defer resetResponse()

		todoID := "test-id"

		// Mock exists check - exists (uses PrepareNamed)
		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM todos WHERE (todos.id = ?) )")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock successful update (uses NamedExec directly, no Prepare)
		sqlMock.ExpectExec(regexp.QuoteMeta("UPDATE todos SET")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock cache invalidation (Delete specific key + Clear patterns)
		mockCache.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().
			SetBody(validBody).
			Patch("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseMessage.Message)
		assert.Equal(t, "Todo updated successfully", *responseMessage.Message)
	})

	t.Run("Success: partial update - only title", func(t *testing.T) {
		defer resetResponse()

		todoID := "test-id"
		partialBody := dto.UpdateTodoRequest{
			Title: "Only Title Updated",
		}

		// Mock exists check - exists (uses PrepareNamed)
		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM todos WHERE (todos.id = ?) )")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock successful update (uses NamedExec directly, no Prepare)
		sqlMock.ExpectExec(regexp.QuoteMeta("UPDATE todos SET")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock cache invalidation (Delete specific key + Clear patterns)
		mockCache.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().
			SetBody(partialBody).
			Patch("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
	})

	t.Run("Success: update completed status only", func(t *testing.T) {
		defer resetResponse()

		todoID := "test-id"
		completed := false
		statusBody := dto.UpdateTodoRequest{
			Completed: &completed,
		}

		// Mock exists check - exists (uses PrepareNamed)
		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM todos WHERE (todos.id = ?) )")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock successful update (uses NamedExec directly, no Prepare)
		sqlMock.ExpectExec(regexp.QuoteMeta("UPDATE todos SET")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock cache invalidation (Delete specific key + Clear patterns)
		mockCache.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().
			SetBody(statusBody).
			Patch("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
	})
}

func TestDeleteTodo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, sqlMock, mockCache := setup(t, ctrl)
	defer ts.Close()

	var responseMessage response.Message
	var responseErr response.Error

	resetResponse := func() {
		responseMessage = response.Message{}
		responseErr = response.Error{}
	}

	getClient := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseMessage).
			SetError(&responseErr)
	}

	t.Run("Error: todo not found", func(t *testing.T) {
		defer resetResponse()

		todoID := "non-existent-id"

		// Mock exists check - doesn't exist (uses PrepareNamed)
		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM todos WHERE (todos.id = ?) )")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		resp, err := getClient().Delete("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusNotFound, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: failed to check existence", func(t *testing.T) {
		defer resetResponse()

		todoID := "test-id"

		// Mock exists check failure (uses PrepareNamed)
		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM todos WHERE (todos.id = ?) )")).
			ExpectQuery().
			WillReturnError(fmt.Errorf("db error"))

		resp, err := getClient().Delete("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: failed to delete todo", func(t *testing.T) {
		defer resetResponse()

		todoID := "test-id"

		// Mock exists check - exists (uses PrepareNamed)
		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM todos WHERE (todos.id = ?) )")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock delete failure (uses NamedExec directly, no Prepare)
		sqlMock.ExpectExec(regexp.QuoteMeta("DELETE FROM todos WHERE (todos.id = ?)")).
			WillReturnError(fmt.Errorf("delete error"))

		resp, err := getClient().Delete("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Success: delete todo", func(t *testing.T) {
		defer resetResponse()

		todoID := "test-id"

		// Mock exists check - exists (uses PrepareNamed)
		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM todos WHERE (todos.id = ?) )")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock successful delete (uses NamedExec directly, no Prepare)
		sqlMock.ExpectExec(regexp.QuoteMeta("DELETE FROM todos WHERE (todos.id = ?)")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock cache invalidation (Delete specific key + Clear patterns)
		mockCache.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().Delete("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseMessage.Message)
		assert.Equal(t, "Todo deleted successfully", *responseMessage.Message)
	})

	t.Run("Success: delete todo with UUID format", func(t *testing.T) {
		defer resetResponse()

		todoID := "550e8400-e29b-41d4-a716-446655440000"

		// Mock exists check - exists (uses PrepareNamed)
		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM todos WHERE (todos.id = ?) )")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Mock successful delete (uses NamedExec directly, no Prepare)
		sqlMock.ExpectExec(regexp.QuoteMeta("DELETE FROM todos WHERE (todos.id = ?)")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock cache invalidation (Delete specific key + Clear patterns)
		mockCache.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().Delete("/todos/" + todoID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
	})
}
