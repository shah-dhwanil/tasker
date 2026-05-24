package handler

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/shah-dhwanil/tasker/internal/validation"
)

type HealthHandler struct {
}

type User struct {
  ID int `query:"id" validate:"required"`
}

func (u *User) Validate(client validation.ValidatorClient) error {
	return client.Struct(u)
}

func (h *HealthHandler) HealthCheck(c echo.Context, payload *User) error {
	fmt.Println(payload)
	return c.JSON(200, map[string]string{"status": "healthy"})
}