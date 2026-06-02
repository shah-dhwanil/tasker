package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/shah-dhwanil/tasker/internal/schema"
	"github.com/shah-dhwanil/tasker/internal/service"
)

type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) CreateCategory(ctx context.Context, userID uuid.UUID, req *schema.CreateCategoryRequest) (*schema.CreateCategoryResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.CreateCategoryResponse), args.Error(1)
}

func (m *MockCategoryRepository) GetCategoryByID(ctx context.Context, categoryID uuid.UUID, includeDeletedRecord bool) (*schema.Category, error) {
	args := m.Called(ctx, categoryID, includeDeletedRecord)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetAllCategories(ctx context.Context, userID *uuid.UUID, payload *schema.GetCategoriesQuery, includeDeletedRecords bool) ([]schema.GetCategoriesResponse, error) {
	args := m.Called(ctx, userID, payload, includeDeletedRecords)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]schema.GetCategoriesResponse), args.Error(1)
}

func (m *MockCategoryRepository) CountCategories(ctx context.Context, userID *uuid.UUID, payload *schema.GetCategoriesQuery, includeDeletedRecords bool) (int, error) {
	args := m.Called(ctx, userID, payload, includeDeletedRecords)
	return args.Int(0), args.Error(1)
}

func (m *MockCategoryRepository) UpdateCategory(ctx context.Context, categoryID uuid.UUID, payload *schema.UpdateCategoryRequest, considerDeletedRecords bool) (*schema.UpdateCategoryResponse, error) {
	args := m.Called(ctx, categoryID, payload, considerDeletedRecords)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.UpdateCategoryResponse), args.Error(1)
}

func (m *MockCategoryRepository) DeleteCategory(ctx context.Context, categoryID uuid.UUID, isHardDelete *bool) error {
	args := m.Called(ctx, categoryID, isHardDelete)
	return args.Error(0)
}

func TestCategoryService_CreateCategory_Success(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	req := &schema.CreateCategoryRequest{Name: "Work"}
	expected := &schema.CreateCategoryResponse{
		ID:   uuid.New(),
		Name: "Work",
	}

	mockRepo.On("CreateCategory", ctx, userID, req).Return(expected, nil)

	result, err := svc.CreateCategory(ctx, userID, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expected.ID, result.ID)
	assert.Equal(t, expected.Name, result.Name)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_CreateCategory_RepoError(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	req := &schema.CreateCategoryRequest{Name: "Work"}
	expectedErr := assert.AnError

	mockRepo.On("CreateCategory", ctx, userID, req).Return(nil, expectedErr)

	result, err := svc.CreateCategory(ctx, userID, req)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_GetCategoryByID_Success(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	categoryID := uuid.New()
	expected := &schema.Category{
		ID:     categoryID,
		UserID: userID,
		Name:   "Work",
	}

	mockRepo.On("GetCategoryByID", ctx, categoryID, false).Return(expected, nil)

	result, err := svc.GetCategoryByID(ctx, userID, categoryID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expected.ID, result.ID)
	assert.Equal(t, expected.Name, result.Name)
	assert.Equal(t, expected.UserID, result.UserID)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_GetCategoryByID_OwnershipViolation(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	categoryID := uuid.New()
	otherUserID := uuid.New()
	fetched := &schema.Category{
		ID:     categoryID,
		UserID: otherUserID,
		Name:   "Work",
	}

	mockRepo.On("GetCategoryByID", ctx, categoryID, false).Return(fetched, nil)

	result, err := svc.GetCategoryByID(ctx, userID, categoryID)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "RESOURCE_NOT_FOUND [Category Not Found]: No category found with the specified data", err.Error())
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_GetCategoryByID_RepoError(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	categoryID := uuid.New()
	expectedErr := assert.AnError

	mockRepo.On("GetCategoryByID", ctx, categoryID, false).Return(nil, expectedErr)

	result, err := svc.GetCategoryByID(ctx, userID, categoryID)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_GetAllCategories_Success(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	page := 1
	limit := 10
	query := &schema.GetCategoriesQuery{Page: &page, Limit: &limit}
	query.Normalize()
	expectedCategories := []schema.GetCategoriesResponse{
		{ID: uuid.New(), Name: "Work"},
		{ID: uuid.New(), Name: "Personal"},
	}

	mockRepo.On("GetAllCategories", ctx, mock.MatchedBy(func(u *uuid.UUID) bool {
		return u != nil && *u == userID
	}), query, false).Return(expectedCategories, nil)
	mockRepo.On("CountCategories", ctx, mock.MatchedBy(func(u *uuid.UUID) bool {
		return u != nil && *u == userID
	}), query, false).Return(2, nil)

	result, err := svc.GetAllCategories(ctx, userID, query)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expectedCategories, result.Data)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.Limit)
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 1, result.TotalPages)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_GetAllCategories_TotalPagesCeiling(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	page := 1
	limit := 10
	query := &schema.GetCategoriesQuery{Page: &page, Limit: &limit}
	query.Normalize()
	categories := make([]schema.GetCategoriesResponse, 15)
	for i := 0; i < 15; i++ {
		categories[i] = schema.GetCategoriesResponse{ID: uuid.New(), Name: "Category"}
	}

	mockRepo.On("GetAllCategories", ctx, mock.MatchedBy(func(u *uuid.UUID) bool {
		return u != nil && *u == userID
	}), query, false).Return(categories, nil)
	mockRepo.On("CountCategories", ctx, mock.MatchedBy(func(u *uuid.UUID) bool {
		return u != nil && *u == userID
	}), query, false).Return(15, nil)

	result, err := svc.GetAllCategories(ctx, userID, query)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalPages)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_GetAllCategories_ZeroTotal(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	page := 1
	limit := 10
	query := &schema.GetCategoriesQuery{Page: &page, Limit: &limit}
	query.Normalize()

	mockRepo.On("GetAllCategories", ctx, mock.MatchedBy(func(u *uuid.UUID) bool {
		return u != nil && *u == userID
	}), query, false).Return([]schema.GetCategoriesResponse{}, nil)
	mockRepo.On("CountCategories", ctx, mock.MatchedBy(func(u *uuid.UUID) bool {
		return u != nil && *u == userID
	}), query, false).Return(0, nil)

	result, err := svc.GetAllCategories(ctx, userID, query)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Equal(t, 1, result.TotalPages)
	assert.Empty(t, result.Data)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_GetAllCategories_RepoErrorOnGetAll(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	page := 1
	limit := 10
	query := &schema.GetCategoriesQuery{Page: &page, Limit: &limit}
	query.Normalize()
	expectedErr := assert.AnError

	mockRepo.On("GetAllCategories", ctx, mock.MatchedBy(func(u *uuid.UUID) bool {
		return u != nil && *u == userID
	}), query, false).Return(nil, expectedErr)

	result, err := svc.GetAllCategories(ctx, userID, query)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_GetAllCategories_RepoErrorOnCount(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	page := 1
	limit := 10
	query := &schema.GetCategoriesQuery{Page: &page, Limit: &limit}
	query.Normalize()
	expectedErr := assert.AnError

	mockRepo.On("GetAllCategories", ctx, mock.MatchedBy(func(u *uuid.UUID) bool {
		return u != nil && *u == userID
	}), query, false).Return([]schema.GetCategoriesResponse{
		{ID: uuid.New(), Name: "Work"},
	}, nil)
	mockRepo.On("CountCategories", ctx, mock.MatchedBy(func(u *uuid.UUID) bool {
		return u != nil && *u == userID
	}), query, false).Return(0, expectedErr)

	result, err := svc.GetAllCategories(ctx, userID, query)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_UpdateCategory_Success(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	categoryID := uuid.New()
	newName := "Updated"
	req := &schema.UpdateCategoryRequest{Name: &newName}
	fetched := &schema.Category{
		ID:     categoryID,
		UserID: userID,
		Name:   "Original",
	}
	expected := &schema.UpdateCategoryResponse{
		ID:   categoryID,
		Name: newName,
	}

	mockRepo.On("GetCategoryByID", ctx, categoryID, false).Return(fetched, nil)
	mockRepo.On("UpdateCategory", ctx, categoryID, req, false).Return(expected, nil)

	result, err := svc.UpdateCategory(ctx, userID, categoryID, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expected.ID, result.ID)
	assert.Equal(t, expected.Name, result.Name)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_UpdateCategory_OwnershipViolation(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	categoryID := uuid.New()
	otherUserID := uuid.New()
	newName := "Updated"
	req := &schema.UpdateCategoryRequest{Name: &newName}
	fetched := &schema.Category{
		ID:     categoryID,
		UserID: otherUserID,
		Name:   "Original",
	}

	mockRepo.On("GetCategoryByID", ctx, categoryID, false).Return(fetched, nil)

	result, err := svc.UpdateCategory(ctx, userID, categoryID, req)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "RESOURCE_NOT_FOUND [Category Not Found]: No category found with the specified data", err.Error())
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_UpdateCategory_GetByIDError(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	categoryID := uuid.New()
	newName := "Updated"
	req := &schema.UpdateCategoryRequest{Name: &newName}
	expectedErr := assert.AnError

	mockRepo.On("GetCategoryByID", ctx, categoryID, false).Return(nil, expectedErr)

	result, err := svc.UpdateCategory(ctx, userID, categoryID, req)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_UpdateCategory_UpdateError(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	categoryID := uuid.New()
	newName := "Updated"
	req := &schema.UpdateCategoryRequest{Name: &newName}
	fetched := &schema.Category{
		ID:     categoryID,
		UserID: userID,
		Name:   "Original",
	}
	expectedErr := assert.AnError

	mockRepo.On("GetCategoryByID", ctx, categoryID, false).Return(fetched, nil)
	mockRepo.On("UpdateCategory", ctx, categoryID, req, false).Return(nil, expectedErr)

	result, err := svc.UpdateCategory(ctx, userID, categoryID, req)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_DeleteCategory_Success(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	categoryID := uuid.New()
	fetched := &schema.Category{
		ID:     categoryID,
		UserID: userID,
		Name:   "Work",
	}

	mockRepo.On("GetCategoryByID", ctx, categoryID, false).Return(fetched, nil)
	mockRepo.On("DeleteCategory", ctx, categoryID, (*bool)(nil)).Return(nil)

	err := svc.DeleteCategory(ctx, userID, categoryID)
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_DeleteCategory_OwnershipViolation(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	categoryID := uuid.New()
	otherUserID := uuid.New()
	fetched := &schema.Category{
		ID:     categoryID,
		UserID: otherUserID,
		Name:   "Work",
	}

	mockRepo.On("GetCategoryByID", ctx, categoryID, false).Return(fetched, nil)

	err := svc.DeleteCategory(ctx, userID, categoryID)
	require.Error(t, err)
	assert.Equal(t, "RESOURCE_NOT_FOUND [Category Not Found]: No category found with the specified data", err.Error())
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_DeleteCategory_GetByIDError(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	categoryID := uuid.New()
	expectedErr := assert.AnError

	mockRepo.On("GetCategoryByID", ctx, categoryID, false).Return(nil, expectedErr)

	err := svc.DeleteCategory(ctx, userID, categoryID)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	mockRepo.AssertExpectations(t)
}

func TestCategoryService_DeleteCategory_DeleteError(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	svc := service.NewCategoryService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	categoryID := uuid.New()
	fetched := &schema.Category{
		ID:     categoryID,
		UserID: userID,
		Name:   "Work",
	}
	expectedErr := assert.AnError

	mockRepo.On("GetCategoryByID", ctx, categoryID, false).Return(fetched, nil)
	mockRepo.On("DeleteCategory", ctx, categoryID, (*bool)(nil)).Return(expectedErr)

	err := svc.DeleteCategory(ctx, userID, categoryID)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	mockRepo.AssertExpectations(t)
}
