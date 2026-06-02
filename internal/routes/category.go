package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/shah-dhwanil/tasker/internal/handler"
	"github.com/shah-dhwanil/tasker/internal/middleware"
	"github.com/shah-dhwanil/tasker/internal/schema"
)

func RegisterCategoryRoutes(server *echo.Echo, handlers *handler.Handler) {
	group := server.Group("/api/v1/categories", middleware.ClerkAuth())

	group.POST("", handler.Handle(handlers.CategoryHandler.CreateCategory, &schema.CreateCategoryRequest{}))
	group.GET("/:categoryId", handler.Handle(handlers.CategoryHandler.GetCategoryByID, &handler.CategoryIDRequest{}))
	group.GET("", handler.Handle(handlers.CategoryHandler.GetAllCategories, &schema.GetCategoriesQuery{}))
	group.PATCH("/:categoryId", handler.Handle(handlers.CategoryHandler.UpdateCategory, &handler.UpdateCategoryRequest{}))
	group.DELETE("/:categoryId", handler.Handle(handlers.CategoryHandler.DeleteCategory, &handler.CategoryIDRequest{}))
}
