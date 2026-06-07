package service

import (
	"github.com/shah-dhwanil/tasker/internal/database"
	"github.com/shah-dhwanil/tasker/internal/repository"
)

type Service struct {
	CategoryService *CategoryService
	TodoService     *TodoService
}

func New(repo *repository.Repository, db database.DBTX) *Service {
	return &Service{
		CategoryService: NewCategoryService(repo.CategoryRepository),
		TodoService:     NewTodoService(repo.TodoRepository, db),
	}
}