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

func (m *MockCategoryRepository) CreateCategory(ctx context.Context, userID string, req *schema.CreateCategoryRequest) (*schema.CreateCategoryResponse, error) {
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

func (m *MockCategoryRepository) GetAllCategories(ctx context.Context, userID *string, payload *schema.GetCategoriesQuery, includeDeletedRecords bool) ([]schema.GetCategoriesResponse, error) {
	args := m.Called(ctx, userID, payload, includeDeletedRecords)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]schema.GetCategoriesResponse), args.Error(1)
}

func (m *MockCategoryRepository) CountCategories(ctx context.Context, userID *string, payload *schema.GetCategoriesQuery, includeDeletedRecords bool) (int, error) {
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

var (
	ctx        = context.Background()
	userID     = "test-user-id"
	errNotFound = "RESOURCE_NOT_FOUND [Category Not Found]: No category found with the specified data"
)

func TestCategoryService_CreateCategory(t *testing.T) {
	req := &schema.CreateCategoryRequest{Name: "Work"}
	expected := &schema.CreateCategoryResponse{ID: uuid.New(), Name: "Work"}

	tests := []struct {
		name    string
		setup   func(*MockCategoryRepository)
		wantErr error
		check   func(*testing.T, *schema.CreateCategoryResponse)
	}{
		{
			name: "success",
			setup: func(m *MockCategoryRepository) {
				m.On("CreateCategory", ctx, userID, req).Return(expected, nil)
			},
			check: func(t *testing.T, result *schema.CreateCategoryResponse) {
				assert.Equal(t, expected.ID, result.ID)
				assert.Equal(t, expected.Name, result.Name)
			},
		},
		{
			name: "repo error",
			setup: func(m *MockCategoryRepository) {
				m.On("CreateCategory", ctx, userID, req).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockCategoryRepository)
			svc := service.NewCategoryService(mockRepo)
			tt.setup(mockRepo)
			result, err := svc.CreateCategory(ctx, userID, req)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.check != nil {
					tt.check(t, result)
				}
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCategoryService_GetCategoryByID(t *testing.T) {
	categoryID := uuid.New()

	tests := []struct {
		name       string
		setup      func(*MockCategoryRepository)
		wantErr    error
		wantErrMsg string
		check      func(*testing.T, *schema.Category)
	}{
		{
			name: "success",
			setup: func(m *MockCategoryRepository) {
				m.On("GetCategoryByID", ctx, categoryID, false).Return(&schema.Category{
					ID: categoryID, UserID: userID, Name: "Work",
				}, nil)
			},
			check: func(t *testing.T, result *schema.Category) {
				assert.Equal(t, categoryID, result.ID)
				assert.Equal(t, "Work", result.Name)
				assert.Equal(t, userID, result.UserID)
			},
		},
		{
			name: "ownership violation",
			setup: func(m *MockCategoryRepository) {
				m.On("GetCategoryByID", ctx, categoryID, false).Return(&schema.Category{
					ID: categoryID, UserID: "other-user-id", Name: "Work",
				}, nil)
			},
			wantErrMsg: errNotFound,
		},
		{
			name: "repo error",
			setup: func(m *MockCategoryRepository) {
				m.On("GetCategoryByID", ctx, categoryID, false).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockCategoryRepository)
			svc := service.NewCategoryService(mockRepo)
			tt.setup(mockRepo)
			result, err := svc.GetCategoryByID(ctx, userID, categoryID)
			if tt.wantErr != nil || tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.Nil(t, result)
				if tt.wantErr != nil {
					assert.ErrorIs(t, err, tt.wantErr)
				}
				if tt.wantErrMsg != "" {
					assert.Equal(t, tt.wantErrMsg, err.Error())
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.check != nil {
					tt.check(t, result)
				}
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCategoryService_GetAllCategories(t *testing.T) {
	page := 1
	limit := 10
	query := &schema.GetCategoriesQuery{Page: &page, Limit: &limit}
	query.Normalize()

	tests := []struct {
		name    string
		setup   func(*MockCategoryRepository)
		wantErr error
		check   func(*testing.T, *schema.PaginatedResponse[schema.GetCategoriesResponse])
	}{
		{
			name: "success",
			setup: func(m *MockCategoryRepository) {
				categories := []schema.GetCategoriesResponse{
					{ID: uuid.New(), Name: "Work"},
					{ID: uuid.New(), Name: "Personal"},
				}
				m.On("GetAllCategories", ctx, mock.MatchedBy(func(u *string) bool {
					return u != nil && *u == userID
				}), query, false).Return(categories, nil)
				m.On("CountCategories", ctx, mock.MatchedBy(func(u *string) bool {
					return u != nil && *u == userID
				}), query, false).Return(2, nil)
			},
			check: func(t *testing.T, result *schema.PaginatedResponse[schema.GetCategoriesResponse]) {
				assert.Len(t, result.Data, 2)
				assert.Equal(t, 1, result.Page)
				assert.Equal(t, 10, result.Limit)
				assert.Equal(t, 2, result.Total)
				assert.Equal(t, 1, result.TotalPages)
			},
		},
		{
			name: "total pages ceiling",
			setup: func(m *MockCategoryRepository) {
				categories := make([]schema.GetCategoriesResponse, 15)
				for i := 0; i < 15; i++ {
					categories[i] = schema.GetCategoriesResponse{ID: uuid.New(), Name: "Category"}
				}
				m.On("GetAllCategories", ctx, mock.MatchedBy(func(u *string) bool {
					return u != nil && *u == userID
				}), query, false).Return(categories, nil)
				m.On("CountCategories", ctx, mock.MatchedBy(func(u *string) bool {
					return u != nil && *u == userID
				}), query, false).Return(15, nil)
			},
			check: func(t *testing.T, result *schema.PaginatedResponse[schema.GetCategoriesResponse]) {
				assert.Equal(t, 2, result.TotalPages)
			},
		},
		{
			name: "zero total",
			setup: func(m *MockCategoryRepository) {
				m.On("GetAllCategories", ctx, mock.MatchedBy(func(u *string) bool {
					return u != nil && *u == userID
				}), query, false).Return([]schema.GetCategoriesResponse{}, nil)
				m.On("CountCategories", ctx, mock.MatchedBy(func(u *string) bool {
					return u != nil && *u == userID
				}), query, false).Return(0, nil)
			},
			check: func(t *testing.T, result *schema.PaginatedResponse[schema.GetCategoriesResponse]) {
				assert.Equal(t, 0, result.Total)
				assert.Equal(t, 1, result.TotalPages)
				assert.Empty(t, result.Data)
			},
		},
		{
			name: "repo error on get all",
			setup: func(m *MockCategoryRepository) {
				m.On("GetAllCategories", ctx, mock.MatchedBy(func(u *string) bool {
					return u != nil && *u == userID
				}), query, false).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name: "repo error on count",
			setup: func(m *MockCategoryRepository) {
				m.On("GetAllCategories", ctx, mock.MatchedBy(func(u *string) bool {
					return u != nil && *u == userID
				}), query, false).Return([]schema.GetCategoriesResponse{
					{ID: uuid.New(), Name: "Work"},
				}, nil)
				m.On("CountCategories", ctx, mock.MatchedBy(func(u *string) bool {
					return u != nil && *u == userID
				}), query, false).Return(0, assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockCategoryRepository)
			svc := service.NewCategoryService(mockRepo)
			tt.setup(mockRepo)
			result, err := svc.GetAllCategories(ctx, userID, query)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.check != nil {
					tt.check(t, result)
				}
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCategoryService_UpdateCategory(t *testing.T) {
	categoryID := uuid.New()
	newName := "Updated"
	req := &schema.UpdateCategoryRequest{Name: &newName}

	tests := []struct {
		name       string
		setup      func(*MockCategoryRepository)
		wantErr    error
		wantErrMsg string
		check      func(*testing.T, *schema.UpdateCategoryResponse)
	}{
		{
			name: "success",
			setup: func(m *MockCategoryRepository) {
				fetched := &schema.Category{
					ID: categoryID, UserID: userID, Name: "Original",
				}
				expected := &schema.UpdateCategoryResponse{
					ID: categoryID, Name: newName,
				}
				m.On("GetCategoryByID", ctx, categoryID, false).Return(fetched, nil)
				m.On("UpdateCategory", ctx, categoryID, req, false).Return(expected, nil)
			},
			check: func(t *testing.T, result *schema.UpdateCategoryResponse) {
				assert.Equal(t, categoryID, result.ID)
				assert.Equal(t, newName, result.Name)
			},
		},
		{
			name: "ownership violation",
			setup: func(m *MockCategoryRepository) {
				fetched := &schema.Category{
					ID: categoryID, UserID: "other-user-id", Name: "Original",
				}
				m.On("GetCategoryByID", ctx, categoryID, false).Return(fetched, nil)
			},
			wantErrMsg: errNotFound,
		},
		{
			name: "get by id error",
			setup: func(m *MockCategoryRepository) {
				m.On("GetCategoryByID", ctx, categoryID, false).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name: "update error",
			setup: func(m *MockCategoryRepository) {
				fetched := &schema.Category{
					ID: categoryID, UserID: userID, Name: "Original",
				}
				m.On("GetCategoryByID", ctx, categoryID, false).Return(fetched, nil)
				m.On("UpdateCategory", ctx, categoryID, req, false).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockCategoryRepository)
			svc := service.NewCategoryService(mockRepo)
			tt.setup(mockRepo)
			result, err := svc.UpdateCategory(ctx, userID, categoryID, req)
			if tt.wantErr != nil || tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.Nil(t, result)
				if tt.wantErr != nil {
					assert.ErrorIs(t, err, tt.wantErr)
				}
				if tt.wantErrMsg != "" {
					assert.Equal(t, tt.wantErrMsg, err.Error())
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.check != nil {
					tt.check(t, result)
				}
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCategoryService_DeleteCategory(t *testing.T) {
	categoryID := uuid.New()

	tests := []struct {
		name       string
		setup      func(*MockCategoryRepository)
		wantErr    error
		wantErrMsg string
	}{
		{
			name: "success",
			setup: func(m *MockCategoryRepository) {
				fetched := &schema.Category{
					ID: categoryID, UserID: userID, Name: "Work",
				}
				m.On("GetCategoryByID", ctx, categoryID, false).Return(fetched, nil)
				m.On("DeleteCategory", ctx, categoryID, (*bool)(nil)).Return(nil)
			},
		},
		{
			name: "ownership violation",
			setup: func(m *MockCategoryRepository) {
				fetched := &schema.Category{
					ID: categoryID, UserID: "other-user-id", Name: "Work",
				}
				m.On("GetCategoryByID", ctx, categoryID, false).Return(fetched, nil)
			},
			wantErrMsg: errNotFound,
		},
		{
			name: "get by id error",
			setup: func(m *MockCategoryRepository) {
				m.On("GetCategoryByID", ctx, categoryID, false).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name: "delete error",
			setup: func(m *MockCategoryRepository) {
				fetched := &schema.Category{
					ID: categoryID, UserID: userID, Name: "Work",
				}
				m.On("GetCategoryByID", ctx, categoryID, false).Return(fetched, nil)
				m.On("DeleteCategory", ctx, categoryID, (*bool)(nil)).Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockCategoryRepository)
			svc := service.NewCategoryService(mockRepo)
			tt.setup(mockRepo)
			err := svc.DeleteCategory(ctx, userID, categoryID)
			if tt.wantErr != nil || tt.wantErrMsg != "" {
				require.Error(t, err)
				if tt.wantErr != nil {
					assert.ErrorIs(t, err, tt.wantErr)
				}
				if tt.wantErrMsg != "" {
					assert.Equal(t, tt.wantErrMsg, err.Error())
				}
			} else {
				require.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}
