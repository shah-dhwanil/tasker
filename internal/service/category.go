package service

import (
	"context"
	"math"

	"github.com/google/uuid"
	"github.com/shah-dhwanil/tasker/internal/errors"
	"github.com/shah-dhwanil/tasker/internal/observability"
	"github.com/shah-dhwanil/tasker/internal/schema"
	"go.uber.org/zap"
)

type CategoryRepository interface {
	CreateCategory(ctx context.Context, userID uuid.UUID, req *schema.CreateCategoryRequest) (*schema.CreateCategoryResponse, error)
	GetCategoryByID(ctx context.Context, categoryID uuid.UUID, includeDeletedRecord bool) (*schema.Category, error)
	GetAllCategories(ctx context.Context, userID *uuid.UUID, payload *schema.GetCategoriesQuery, includeDeletedRecords bool) ([]schema.GetCategoriesResponse, error)
	CountCategories(ctx context.Context, userID *uuid.UUID, payload *schema.GetCategoriesQuery, includeDeletedRecords bool) (int, error)
	UpdateCategory(ctx context.Context, categoryID uuid.UUID, payload *schema.UpdateCategoryRequest, considerDeletedRecords bool) (*schema.UpdateCategoryResponse, error)
	DeleteCategory(ctx context.Context, categoryID uuid.UUID, isHardDelete *bool) error
}

type CategoryService struct {
	repo CategoryRepository
}

func NewCategoryService(repo CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) CreateCategory(ctx context.Context, userID uuid.UUID, req *schema.CreateCategoryRequest) (*schema.CreateCategoryResponse, error) {
	logger := observability.FromContext(ctx)
	result, err := s.repo.CreateCategory(ctx, userID, req)
	if err != nil {
		logger.Error("failed to create category", zap.String("user_id", userID.String()), zap.String("name", req.Name), zap.Error(err))
		return nil, err
	}
	logger.Info("category created", zap.String("category_id", result.ID.String()), zap.String("name", result.Name))
	return result, nil
}

func (s *CategoryService) GetCategoryByID(ctx context.Context, userID, categoryID uuid.UUID) (*schema.Category, error) {
	logger := observability.FromContext(ctx)
	category, err := s.repo.GetCategoryByID(ctx, categoryID, false)
	if err != nil {
		return nil, err
	}
	if category.UserID != userID {
		logger.Debug("unauthorized category access",
			zap.String("requesting_user", userID.String()),
			zap.String("category_id", categoryID.String()),
			zap.String("owner_user", category.UserID.String()),
		)
		return nil, errors.NewCategoryNotFoundError(nil, nil)
	}
	return category, nil
}

func (s *CategoryService) GetAllCategories(ctx context.Context, userID uuid.UUID, query *schema.GetCategoriesQuery) (*schema.PaginatedResponse[schema.GetCategoriesResponse], error) {
	logger := observability.FromContext(ctx)

	categories, err := s.repo.GetAllCategories(ctx, &userID, query, false)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.CountCategories(ctx, &userID, query, false)
	if err != nil {
		return nil, err
	}

	page := *query.Page
	limit := *query.Limit
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	logger.Info("categories fetched",
		zap.Int("count", len(categories)),
		zap.Int("total", total),
		zap.Int("page", page),
	)
	return &schema.PaginatedResponse[schema.GetCategoriesResponse]{
		Data:       categories,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *CategoryService) UpdateCategory(ctx context.Context, userID, categoryID uuid.UUID, req *schema.UpdateCategoryRequest) (*schema.UpdateCategoryResponse, error) {
	logger := observability.FromContext(ctx)

	category, err := s.repo.GetCategoryByID(ctx, categoryID, false)
	if err != nil {
		return nil, err
	}
	if category.UserID != userID {
		logger.Warn("unauthorized category update",
			zap.String("requesting_user", userID.String()),
			zap.String("category_id", categoryID.String()),
		)
		return nil, errors.NewCategoryNotFoundError(nil, nil)
	}

	result, err := s.repo.UpdateCategory(ctx, categoryID, req, false)
	if err != nil {
		logger.Error("failed to update category", zap.String("category_id", categoryID.String()), zap.Error(err))
		return nil, err
	}
	logger.Info("category updated", zap.String("category_id", result.ID.String()), zap.String("name", result.Name))
	return result, nil
}

func (s *CategoryService) DeleteCategory(ctx context.Context, userID, categoryID uuid.UUID) error {
	logger := observability.FromContext(ctx)

	category, err := s.repo.GetCategoryByID(ctx, categoryID, false)
	if err != nil {
		return err
	}
	if category.UserID != userID {
		logger.Warn("unauthorized category delete",
			zap.String("requesting_user", userID.String()),
			zap.String("category_id", categoryID.String()),
		)
		return errors.NewCategoryNotFoundError(nil, nil)
	}

	if err := s.repo.DeleteCategory(ctx, categoryID, nil); err != nil {
		logger.Error("failed to delete category", zap.String("category_id", categoryID.String()), zap.Error(err))
		return err
	}
	logger.Info("category deleted", zap.String("category_id", categoryID.String()))
	return nil
}