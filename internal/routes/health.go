package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/shah-dhwanil/tasker/internal/handler"
)


func RegisterHealthRoutes(server *echo.Echo, handlers *handler.Handler){
	server.GET("/health", handler.Handle(handlers.HealthHandler.HealthCheck,&handler.User{}))
}