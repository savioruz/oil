package todo

import (
	"net/http"
	"oil/infras/otel"
	"oil/internal/modules/todo/model"
	"oil/internal/modules/todo/model/dto"
	"oil/internal/modules/todo/service"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/validator"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"

	"github.com/rs/zerolog/log"
)

type Handler struct {
	service service.Todo
	otel    otel.Otel
}

func New(service service.Todo, otel otel.Otel) Handler {
	return Handler{
		service: service,
		otel:    otel,
	}
}

func (handler *Handler) Router(router chi.Router) {
	router.Route("/todos", func(routerGroup chi.Router) {
		routerGroup.Post("/", handler.CreateTodo)
		routerGroup.Get("/", handler.GetTodos)
		routerGroup.Get("/{id}", handler.GetTodoByID)
		routerGroup.Patch("/{id}", handler.UpdateTodo)
		routerGroup.Delete("/{id}", handler.DeleteTodo)
	})
}

// CreateTodo handles the creation of a new todo item.
// @Summary Create a new todo item
// @Description Create a new todo item with the provided details.
// @Tags Todo
// @Accept json
// @Produce json
// @Param request body dto.CreateTodoRequest true "Create Todo Request"
// @Success 201 {object} response.Message "Todo created successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /api/todos [post]
// @Security BearerAuth
func (handler *Handler) CreateTodo(writer http.ResponseWriter, request *http.Request) {
	ctx, scope := handler.otel.NewScope(request.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".CreateTodo")
	defer scope.End()

	req := dto.CreateTodoRequest{}

	if err := validator.Validate(request.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(writer, err)

		return
	}

	if err := handler.service.Create(ctx, req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to create todo")

		response.WithError(writer, err)

		return
	}

	user := shared.GetUserID(ctx)
	scope.AddEvent("Todo created successfully by user " + user)

	response.WithMessage(writer, http.StatusCreated, "Todo created successfully")
}

// GetTodos retrieves all todo items based on query parameters.
// @Summary Get all todo items
// @Description Retrieve all todo items with optional filtering and pagination.
// @Tags Todo
// @Accept json
// @Produce json
// @Param title query string false "Filter by title"
// @Param completed query boolean false "Filter by completion status"
// @Success 200 {array} model.Todo "List of todo items"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /api/todos [get]
func (handler *Handler) GetTodos(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GetTodos")
	defer scope.End()

	queryParams := gDto.QueryParams{}
	queryParams.FromRequest(r, true)

	title := r.URL.Query().Get(model.FieldTitle)

	const maxFilters = 3

	filterGroup := gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters:  make([]any, 0, maxFilters),
	}

	filterGroup.Filters = append(filterGroup.Filters, shared.UserFilter(ctx, constant.FieldCreatedBy, model.TableName))

	filterGroup.Filters = append(filterGroup.Filters, gDto.Filter{
		Field:    model.FieldTitle,
		Operator: gDto.FilterOperatorLike,
		Value:    title,
		Table:    model.TableName,
	})

	if complete, err := shared.ConvertStringToBool(r.URL.Query().Get(model.FieldCompleted)); err == nil {
		filterGroup.Filters = append(filterGroup.Filters, gDto.Filter{
			Field:    model.FieldCompleted,
			Operator: gDto.FilterOperatorEq,
			Value:    complete,
			Table:    model.TableName,
		})
	}

	todos, err := handler.service.GetAll(ctx, queryParams, filterGroup)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get todos")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("Todos retrieved successfully")

	response.WithJSON(w, http.StatusOK, todos)
}

// GetTodoByID retrieves a todo item by its ID.
// @Summary Get a todo item by ID
// @Description Retrieve a todo item by its unique identifier.
// @Tags Todo
// @Accept json
// @Produce json
// @Param id path string true "Todo ID"
// @Success 200 {object} dto.TodoResponse "Todo item details"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /api/todos/{id} [get]
func (handler *Handler) GetTodoByID(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GetTodoByID")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	todo, err := handler.service.Get(ctx, id)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get todo by ID")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("Todo retrieved successfully")

	response.WithJSON(w, http.StatusOK, todo)
}

// UpdateTodo updates an existing todo item by its ID.
// @Summary Update a todo item by ID
// @Description Update the details of an existing todo item.
// @Tags Todo
// @Accept json
// @Produce json
// @Param id path string true "Todo ID"
// @Param request body dto.UpdateTodoRequest true "Update Todo Request"
// @Success 200 {object} response.Message "Todo updated successfully"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /api/todos/{id} [patch]
// @Security BearerAuth
func (handler *Handler) UpdateTodo(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".UpdateTodo")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	req := dto.UpdateTodoRequest{}
	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(w, err)

		return
	}

	if err := handler.service.Update(ctx, req, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to update todo")

		response.WithError(w, err)

		return
	}

	user := shared.GetUserID(ctx)
	scope.AddEvent("Todo updated successfully by user " + user)

	response.WithMessage(w, http.StatusOK, "Todo updated successfully")
}

// DeleteTodo deletes a todo item by its ID.
// @Summary Delete a todo item by ID @SuperAdmin
// @Description Delete a todo item using its unique identifier.
// @Tags Todo
// @Accept json
// @Produce json
// @Param id path string true "Todo ID"
// @Success 200 {object} response.Message "Todo deleted successfully"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /api/todos/{id} [delete]
// @Security BearerAuth
func (handler *Handler) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".DeleteTodo")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	if err := handler.service.Delete(ctx, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to delete todo")

		response.WithError(w, err)

		return
	}

	user := shared.GetUserID(ctx)
	scope.AddEvent("Todo deleted successfully by user " + user)

	response.WithMessage(w, http.StatusOK, "Todo deleted successfully")
}
