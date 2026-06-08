package service

import (
	"context"
	"math"

	"github.com/google/uuid"
	"github.com/shah-dhwanil/tasker/internal/errors"
	"github.com/shah-dhwanil/tasker/internal/observability"
	"github.com/shah-dhwanil/tasker/internal/schema"
	"github.com/shah-dhwanil/tasker/internal/schema/dto"
	"go.uber.org/zap"
)

type CategoryRepository interface {
	CreateCategory(ctx context.Context, userID string, req *dto.CreateCategoryRequest) (*dto.Category, error)
	GetCategoryByID(ctx context.Context, categoryID uuid.UUID, includeDeletedRecord bool) (*dto.Category, error)
	GetAllCategories(ctx context.Context, userID *string, payload *dto.GetCategoriesQuery, includeDeletedRecords bool) ([]dto.CategoriesListItems, error)
	CountCategories(ctx context.Context, userID *string, payload *dto.GetCategoriesQuery, includeDeletedRecords bool) (int, error)
	UpdateCategory(ctx context.Context, categoryID uuid.UUID, payload *dto.UpdateCategoryRequest, considerDeletedRecords bool) (*dto.Category, error)
	DeleteCategory(ctx context.Context, categoryID uuid.UUID, isHardDelete *bool) error
}

type CategoryService struct {
	repo CategoryRepository
}

func NewCategoryService(repo CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) CreateCategory(ctx context.Context, userID string, req *schema.CreateCategoryRequest) (*schema.CreateCategoryResponse, error) {
	logger := observability.FromContext(ctx)
	dtoReq := &dto.CreateCategoryRequest{
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
	}
	result, err := s.repo.CreateCategory(ctx, userID, dtoReq)
	if err != nil {
		logger.Error("failed to create category", zap.String("user_id", userID), zap.String("name", req.Name), zap.Error(err))
		return nil, err
	}
	logger.Info("category created", zap.String("category_id", result.ID.String()), zap.String("name", result.Name))
	return &schema.CreateCategoryResponse{
		ID:          result.ID,
		Name:        result.Name,
		Description: result.Description,
		Metadata:    result.Metadata,
		CreatedAt:   result.CreatedAt,
		UpdatedAt:   result.UpdatedAt,
	}, nil
}

func (s *CategoryService) GetCategoryByID(ctx context.Context, userID string, categoryID uuid.UUID) (*schema.Category, error) {
	logger := observability.FromContext(ctx)
	dtoCategory, err := s.repo.GetCategoryByID(ctx, categoryID, false)
	if err != nil {
		return nil, err
	}
	if dtoCategory.UserID != userID {
		logger.Debug("unauthorized category access",
			zap.String("requesting_user", userID),
			zap.String("category_id", categoryID.String()),
			zap.String("owner_user", dtoCategory.UserID),
		)
		return nil, errors.NewCategoryNotFoundError(nil, nil)
	}
	return &schema.Category{
		ID:          dtoCategory.ID,
		Name:        dtoCategory.Name,
		UserID:      dtoCategory.UserID,
		Description: dtoCategory.Description,
		Metadata:    dtoCategory.Metadata,
		CreatedAt:   dtoCategory.CreatedAt,
		UpdatedAt:   dtoCategory.UpdatedAt,
	}, nil
}

func (s *CategoryService) GetAllCategories(ctx context.Context, userID string, query *schema.GetCategoriesQuery) (*schema.PaginatedResponse[schema.GetCategoriesResponse], error) {
	logger := observability.FromContext(ctx)

	page := 1
	limit := 10
	if query.Page != nil {
		page = *query.Page
	}
	if query.Limit != nil {
		limit = *query.Limit
	}
	offset := (page - 1) * limit

	dtoQuery := &dto.GetCategoriesQuery{
		Offset:  offset,
		Limit:   limit,
		Search:  query.Search,
		OrderBy: query.OrderBy,
	}

	categories, err := s.repo.GetAllCategories(ctx, &userID, dtoQuery, false)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.CountCategories(ctx, &userID, dtoQuery, false)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	logger.Info("categories fetched",
		zap.Int("count", len(categories)),
		zap.Int("total", total),
		zap.Int("page", page),
	)

	data := make([]schema.GetCategoriesResponse, len(categories))
	for i, c := range categories {
		data[i] = schema.GetCategoriesResponse{ID: c.ID, Name: c.Name}
	}

	return &schema.PaginatedResponse[schema.GetCategoriesResponse]{
		Data:       data,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *CategoryService) UpdateCategory(ctx context.Context, userID string, categoryID uuid.UUID, req *schema.UpdateCategoryRequest) (*schema.UpdateCategoryResponse, error) {
	logger := observability.FromContext(ctx)

	dtoCategory, err := s.repo.GetCategoryByID(ctx, categoryID, false)
	if err != nil {
		return nil, err
	}
	if dtoCategory.UserID != userID {
		logger.Warn("unauthorized category update",
			zap.String("requesting_user", userID),
			zap.String("category_id", categoryID.String()),
		)
		return nil, errors.NewCategoryNotFoundError(nil, nil)
	}


	dtoReq := &dto.UpdateCategoryRequest{
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
	}

	result, err := s.repo.UpdateCategory(ctx, categoryID, dtoReq, false)
	if err != nil {
		logger.Error("failed to update category", zap.String("category_id", categoryID.String()), zap.Error(err))
		return nil, err
	}
	logger.Info("category updated", zap.String("category_id", result.ID.String()), zap.String("name", result.Name))
	return &schema.UpdateCategoryResponse{
		ID:          result.ID,
		Name:        result.Name,
		Description: result.Description,
		Metadata:    result.Metadata,
		CreatedAt:   result.CreatedAt,
		UpdatedAt:   result.UpdatedAt,
	}, nil
}

func (s *CategoryService) DeleteCategory(ctx context.Context, userID string, categoryID uuid.UUID) error {
	logger := observability.FromContext(ctx)

	category, err := s.repo.GetCategoryByID(ctx, categoryID, false)
	if err != nil {
		return err
	}
	if category.UserID != userID {
		logger.Warn("unauthorized category delete",
			zap.String("requesting_user", userID),
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