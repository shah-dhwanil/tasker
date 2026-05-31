package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	pkgErrors "github.com/shah-dhwanil/tasker/internal/errors"
	"github.com/shah-dhwanil/tasker/internal/middleware"
	"github.com/shah-dhwanil/tasker/internal/schema"
	"github.com/shah-dhwanil/tasker/internal/service"
	"github.com/shah-dhwanil/tasker/internal/validation"
)

type categoryHandler struct {
	CategoryService *service.CategoryService
}

func newCategoryHandler(service *service.Service) *categoryHandler {
	return &categoryHandler{
		CategoryService: service.CategoryService,
	}
}

type CategoryIDRequest struct {
	CategoryID uuid.UUID `param:"categoryId" validate:"required"`
}

func (r *CategoryIDRequest) Validate(client validation.ValidatorClient) error {
	return client.Struct(r)
}

type UpdateCategoryRequest struct {
	CategoryID uuid.UUID `param:"categoryId" validate:"required"`
	*schema.UpdateCategoryRequest
}

func (r *UpdateCategoryRequest) Validate(client validation.ValidatorClient) error {
	return client.Struct(r)
}

// @Summary      Create a new category
// @Description  Create a new category for the authenticated user
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        category  body  schema.CreateCategoryRequest  true  "Category details"
// @Success      201  {object}  schema.Response[schema.CreateCategoryResponse]
// @Failure      400  {object}  schema.ErrorResponse
// @Failure      401  {object}  schema.ErrorResponse
// @Failure      409  {object}  schema.ErrorResponse
// @Router       /categories [post]
// @Security     BearerAuth
func (h *categoryHandler) CreateCategory(c echo.Context, req *schema.CreateCategoryRequest) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	result, err := h.CategoryService.CreateCategory(c.Request().Context(), userID, req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, schema.Response[schema.CreateCategoryResponse]{
		StatusCode: http.StatusCreated,
		Data:       *result,
	})
}

// @Summary      Get a category by ID
// @Description  Get a single category for the authenticated user by its ID
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        categoryId  path  string  true  "Category ID"
// @Success      200  {object}  schema.Response[schema.Category]
// @Failure      400  {object}  schema.ErrorResponse
// @Failure      401  {object}  schema.ErrorResponse
// @Failure      404  {object}  schema.ErrorResponse
// @Router       /categories/{categoryId} [get]
// @Security     BearerAuth
func (h *categoryHandler) GetCategoryByID(c echo.Context, req *CategoryIDRequest) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	result, err := h.CategoryService.GetCategoryByID(c.Request().Context(), userID, req.CategoryID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, schema.Response[schema.Category]{
		StatusCode: http.StatusOK,
		Data:       *result,
	})
}

// @Summary      List all categories
// @Description  Get all categories for the authenticated user with pagination and search
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        page      query  int     false  "Page number"         minimum(1)
// @Param        limit     query  int     false  "Items per page"      minimum(1)  maximum(100)
// @Param        search    query  string  false  "Search term"         maxlength(32)
// @Param        order_by  query  []string  false  "Order by fields"
// @Success      200  {object}  schema.PaginatedResponse[schema.GetCategoriesResponse]
// @Failure      400  {object}  schema.ErrorResponse
// @Failure      401  {object}  schema.ErrorResponse
// @Router       /categories [get]
// @Security     BearerAuth
func (h *categoryHandler) GetAllCategories(c echo.Context, req *schema.GetCategoriesQuery) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	result, err := h.CategoryService.GetAllCategories(c.Request().Context(), userID, req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// @Summary      Update a category
// @Description  Update an existing category for the authenticated user
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        categoryId  path   string                        true  "Category ID"
// @Param        category    body   schema.UpdateCategoryRequest  true  "Category update details"
// @Success      200  {object}  schema.Response[schema.UpdateCategoryResponse]
// @Failure      400  {object}  schema.ErrorResponse
// @Failure      401  {object}  schema.ErrorResponse
// @Failure      404  {object}  schema.ErrorResponse
// @Router       /categories/{categoryId} [patch]
// @Security     BearerAuth
func (h *categoryHandler) UpdateCategory(c echo.Context, req *UpdateCategoryRequest) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	result, err := h.CategoryService.UpdateCategory(c.Request().Context(), userID, req.CategoryID, req.UpdateCategoryRequest)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, schema.Response[schema.UpdateCategoryResponse]{
		StatusCode: http.StatusOK,
		Data:       *result,
	})
}

// @Summary      Delete a category
// @Description  Delete an existing category for the authenticated user
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        categoryId  path  string  true  "Category ID"
// @Success      204  {string}  no content
// @Failure      400  {object}  schema.ErrorResponse
// @Failure      401  {object}  schema.ErrorResponse
// @Failure      404  {object}  schema.ErrorResponse
// @Router       /categories/{categoryId} [delete]
// @Security     BearerAuth
func (h *categoryHandler) DeleteCategory(c echo.Context, req *CategoryIDRequest) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	if err := h.CategoryService.DeleteCategory(c.Request().Context(), userID, req.CategoryID); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func getUserID(c echo.Context) (uuid.UUID, error) {
	claims, ok := middleware.GetUserFromContext(c)
	if !ok {
		return uuid.Nil, pkgErrors.NewUnauthorizedError(nil, "user not authenticated", nil, nil)
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, pkgErrors.NewUnauthorizedError(err, "invalid user identity", nil, nil)
	}
	return userID, nil
}
