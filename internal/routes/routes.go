package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/shah-dhwanil/tasker/internal/handler"
)


func RegisterRoutes(server *echo.Echo, handler *handler.Handler){
	RegisterHealthRoutes(server, handler)
}