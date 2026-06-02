package handler

import (
	"fmt"

	"github.com/labstack/echo/v4"
	pkgErrors "github.com/shah-dhwanil/tasker/internal/errors"
	"github.com/shah-dhwanil/tasker/internal/normalization"
	"github.com/shah-dhwanil/tasker/internal/service"
	"github.com/shah-dhwanil/tasker/internal/validation"
)

type Handler struct {
	CategoryHandler *categoryHandler
	HealthHandler   *HealthHandler
}

func New(service *service.Service) *Handler {
	return &Handler{
		CategoryHandler: newCategoryHandler(service),
		HealthHandler:   &HealthHandler{},
	}
}

func Handle[T validation.Validable](
	handler func(c echo.Context, request T) error,
	payload T,
) func(c echo.Context) error {

	return func(c echo.Context) error {

		if err := c.Bind(payload); err != nil {
			fmt.Println("Error in binding request body:", err)
			return pkgErrors.NewBindingError(err)
		}

		if err := validation.Validate(payload); err != nil {
			return err
		}

		if v, ok := any(payload).(normalization.Normalizable[T]); ok {
			request, err := v.Normalize()
			if err != nil {
				return fmt.Errorf("Normalization Failed: %w", err)
			}

			payload = request
		}

		return handler(c, payload)
	}
}