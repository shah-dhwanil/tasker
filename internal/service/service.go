package service

import "github.com/shah-dhwanil/tasker/internal/repository"

type Service struct {
}

func New(repo *repository.Repository) *Service {
	return &Service{}
}