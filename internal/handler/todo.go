package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shah-dhwanil/tasker/internal/schema"
	"github.com/shah-dhwanil/tasker/internal/service"
	"github.com/shah-dhwanil/tasker/internal/validation"
)

type todoService interface {
	Create(ctx context.Context, userID string, req *schema.CreateTodoRequest) (*schema.Todo, error)
	GetByID(ctx context.Context, todoID uuid.UUID, userID string, includeDeleted bool) (*schema.Todo, error)
	GetAll(ctx context.Context, userID string, query *schema.GetTodosQuery) (*schema.PaginatedResponse[schema.TodoListItems], error)
	Update(ctx context.Context, todoID uuid.UUID, userID string, payload *schema.UpdateTodoRequest) (*schema.Todo, error)
	Delete(ctx context.Context, todoID uuid.UUID, userID string) error
}

type TodoHandler struct {
	TodoService todoService
}

func NewTodoHandler(service *service.Service) *TodoHandler {
	return &TodoHandler{
		TodoService: service.TodoService,
	}
}

type TodoIDRequest struct {
	TodoID uuid.UUID `param:"todoId" validate:"required"`
}

func (r *TodoIDRequest) Validate(client validation.ValidatorClient) error {
	return client.Struct(r)
}

type UpdateTodoRequest struct {
	TodoID uuid.UUID `param:"todoId" validate:"required"`
	*schema.UpdateTodoRequest
}

func (r *UpdateTodoRequest) Validate(client validation.ValidatorClient) error {
	return client.Struct(r)
}

// @Summary      Create a new todo
// @Description  Create a new todo for the authenticated user
// @Tags         todos
// @Accept       json
// @Produce      json
// @Param        todo  body  schema.CreateTodoRequest  true  "Todo details"
// @Success      201  {object}  schema.Response[schema.Todo]
// @Failure      400  {object}  schema.ErrorResponse
// @Failure      401  {object}  schema.ErrorResponse
// @Router       /todos [post]
// @Security     BearerAuth
func (h *TodoHandler) Create(c echo.Context, req *schema.CreateTodoRequest) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	result, err := h.TodoService.Create(c.Request().Context(), *userID, req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, schema.Response[schema.Todo]{
		StatusCode: http.StatusCreated,
		Data:       *result,
	})
}

// @Summary      Get a todo by ID
// @Description  Get a single todo for the authenticated user by its ID
// @Tags         todos
// @Accept       json
// @Produce      json
// @Param        todoId  path  string  true  "Todo ID"
// @Success      200  {object}  schema.Response[schema.Todo]
// @Failure      400  {object}  schema.ErrorResponse
// @Failure      401  {object}  schema.ErrorResponse
// @Failure      404  {object}  schema.ErrorResponse
// @Router       /todos/{todoId} [get]
// @Security     BearerAuth
func (h *TodoHandler) GetByID(c echo.Context, req *TodoIDRequest) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	result, err := h.TodoService.GetByID(c.Request().Context(), req.TodoID, *userID, false)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, schema.Response[schema.Todo]{
		StatusCode: http.StatusOK,
		Data:       *result,
	})
}

// @Summary      List all todos
// @Description  Get all todos for the authenticated user with pagination, search, filtering, and sorting
// @Tags         todos
// @Accept       json
// @Produce      json
// @Param        page        query  int      false  "Page number"                minimum(1)
// @Param        limit       query  int      false  "Items per page"             minimum(1)  maximum(100)
// @Param        search      query  string   false  "Search term"                maxlength(32)
// @Param        status      query  string   false  "Filter by status"           Enums(draft,pending,in_progress,completed,archived)
// @Param        priority    query  int      false  "Filter by priority"         minimum(1)  maximum(5)
// @Param        category_id query  string   false  "Filter by category ID"
// @Param        parent_id   query  string   false  "Filter by parent ID"
// @Param        order_by    query  []string false  "Order by fields"
// @Success      200  {object}  schema.PaginatedResponse[schema.TodoListItems]
// @Failure      400  {object}  schema.ErrorResponse
// @Failure      401  {object}  schema.ErrorResponse
// @Router       /todos [get]
// @Security     BearerAuth
func (h *TodoHandler) GetAll(c echo.Context, req *schema.GetTodosQuery) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	result, err := h.TodoService.GetAll(c.Request().Context(), *userID, req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// @Summary      Update a todo
// @Description  Update an existing todo for the authenticated user
// @Tags         todos
// @Accept       json
// @Produce      json
// @Param        todoId  path   string                        true  "Todo ID"
// @Param        todo    body   schema.UpdateTodoRequest      true  "Todo update details"
// @Success      200  {object}  schema.Response[schema.Todo]
// @Failure      400  {object}  schema.ErrorResponse
// @Failure      401  {object}  schema.ErrorResponse
// @Failure      404  {object}  schema.ErrorResponse
// @Router       /todos/{todoId} [patch]
// @Security     BearerAuth
func (h *TodoHandler) Update(c echo.Context, req *UpdateTodoRequest) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	result, err := h.TodoService.Update(c.Request().Context(), req.TodoID, *userID, req.UpdateTodoRequest)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, schema.Response[schema.Todo]{
		StatusCode: http.StatusOK,
		Data:       *result,
	})
}

// @Summary      Delete a todo
// @Description  Delete an existing todo for the authenticated user
// @Tags         todos
// @Accept       json
// @Produce      json
// @Param        todoId  path  string  true  "Todo ID"
// @Success      204  {string}  no content
// @Failure      400  {object}  schema.ErrorResponse
// @Failure      401  {object}  schema.ErrorResponse
// @Failure      404  {object}  schema.ErrorResponse
// @Router       /todos/{todoId} [delete]
// @Security     BearerAuth
func (h *TodoHandler) Delete(c echo.Context, req *TodoIDRequest) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	if err := h.TodoService.Delete(c.Request().Context(), req.TodoID, *userID); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}
