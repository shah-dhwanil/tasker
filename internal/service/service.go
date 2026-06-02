package service

import "github.com/shah-dhwanil/tasker/internal/repository"

type Service struct {
	CategoryService *CategoryService
}

func New(repo *repository.Repository) *Service {
	return &Service{
		CategoryService: NewCategoryService(repo.CategoryRepository),
	}
}