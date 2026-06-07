package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/shah-dhwanil/tasker/internal/handler"
	"github.com/shah-dhwanil/tasker/internal/middleware"
	"github.com/shah-dhwanil/tasker/internal/schema"
)

func RegisterTodoRoutes(server *echo.Echo, handlers *handler.Handler) {
	group := server.Group("/api/v1/todos", middleware.ClerkAuth())

	group.POST("", handler.Handle(handlers.TodoHandler.Create, &schema.CreateTodoRequest{}))
	group.GET("/:todoId", handler.Handle(handlers.TodoHandler.GetByID, &handler.TodoIDRequest{}))
	group.GET("", handler.Handle(handlers.TodoHandler.GetAll, &schema.GetTodosQuery{}))
	group.PATCH("/:todoId", handler.Handle(handlers.TodoHandler.Update, &handler.UpdateTodoRequest{}))
	group.DELETE("/:todoId", handler.Handle(handlers.TodoHandler.Delete, &handler.TodoIDRequest{}))
}
