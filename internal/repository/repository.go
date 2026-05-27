package repository

import "github.com/shah-dhwanil/tasker/internal/database"


type Repository struct {
	CategoryRepository *CategoryRepository
}

func New(pool database.PgPool) *Repository{
	return &Repository{
		CategoryRepository: newCategoryRepository(pool),
	}
}