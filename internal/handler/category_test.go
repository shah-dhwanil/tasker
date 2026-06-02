package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	errorhandler "github.com/shah-dhwanil/tasker/internal/error_handler"
	"github.com/shah-dhwanil/tasker/internal/schema"
)

type MockCategoryService struct {
	mock.Mock
}

func (m *MockCategoryService) CreateCategory(ctx context.Context, userID uuid.UUID, req *schema.CreateCategoryRequest) (*schema.CreateCategoryResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.CreateCategoryResponse), args.Error(1)
}

func (m *MockCategoryService) GetCategoryByID(ctx context.Context, userID, categoryID uuid.UUID) (*schema.Category, error) {
	args := m.Called(ctx, userID, categoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Category), args.Error(1)
}

func (m *MockCategoryService) GetAllCategories(ctx context.Context, userID uuid.UUID, query *schema.GetCategoriesQuery) (*schema.PaginatedResponse[schema.GetCategoriesResponse], error) {
	args := m.Called(ctx, userID, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.PaginatedResponse[schema.GetCategoriesResponse]), args.Error(1)
}

func (m *MockCategoryService) UpdateCategory(ctx context.Context, userID, categoryID uuid.UUID, req *schema.UpdateCategoryRequest) (*schema.UpdateCategoryResponse, error) {
	args := m.Called(ctx, userID, categoryID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.UpdateCategoryResponse), args.Error(1)
}

func (m *MockCategoryService) DeleteCategory(ctx context.Context, userID, categoryID uuid.UUID) error {
	args := m.Called(ctx, userID, categoryID)
	return args.Error(0)
}

func authMiddleware(userID uuid.UUID) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user", &clerk.SessionClaims{
				RegisteredClaims: clerk.RegisteredClaims{
					Subject: userID.String(),
				},
			})
			return next(c)
		}
	}
}

func setupServer(t *testing.T, mockSvc *MockCategoryService, userID *uuid.UUID) *echo.Echo {
	t.Helper()

	e := echo.New()
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if !c.Response().Committed {
			res := errorhandler.HandleError(err)
			c.JSON(res.StatusCode, res)
		}
	}

	h := &CategoryHandler{CategoryService: mockSvc}
	group := e.Group("/api/v1/categories")
	if userID != nil {
		group.Use(authMiddleware(*userID))
	}

	group.POST("", Handle(h.CreateCategory, &schema.CreateCategoryRequest{}))
	group.GET("/:categoryId", Handle(h.GetCategoryByID, &CategoryIDRequest{}))
	group.GET("", Handle(h.GetAllCategories, &schema.GetCategoriesQuery{}))
	group.PATCH("/:categoryId", Handle(h.UpdateCategory, &UpdateCategoryRequest{}))
	group.DELETE("/:categoryId", Handle(h.DeleteCategory, &CategoryIDRequest{}))

	return e
}

type categoryTestCase struct {
	name           string
	method         string
	url            string
	body           string
	userID         *uuid.UUID
	setupMock      func(*MockCategoryService)
	expectedStatus int
	assertResponse func(*testing.T, *httptest.ResponseRecorder)
}

func runCategoryTest(t *testing.T, tc categoryTestCase) {
	t.Helper()

	mockSvc := new(MockCategoryService)
	e := setupServer(t, mockSvc, tc.userID)

	if tc.setupMock != nil {
		tc.setupMock(mockSvc)
	}

	var reqBody io.Reader
	if tc.body != "" {
		reqBody = strings.NewReader(tc.body)
	}
	req := httptest.NewRequest(tc.method, tc.url, reqBody)
	if tc.body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	require.Equal(t, tc.expectedStatus, rec.Code)

	if tc.assertResponse != nil {
		tc.assertResponse(t, rec)
	}

	mockSvc.AssertExpectations(t)
}

func ptr(u uuid.UUID) *uuid.UUID {
	return &u
}

func TestCreateCategory(t *testing.T) {
	t.Parallel()

	successID := uuid.New()

	tests := []categoryTestCase{
		{
			name:           "Success",
			method:         http.MethodPost,
			url:            "/api/v1/categories",
			body:           `{"name":"Work"}`,
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusCreated,
			setupMock: func(m *MockCategoryService) {
				m.On("CreateCategory", mock.Anything, mock.Anything, mock.Anything).
					Return(&schema.CreateCategoryResponse{ID: successID, Name: "Work"}, nil)
			},
			assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp schema.Response[schema.CreateCategoryResponse]
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, successID, resp.Data.ID)
				assert.Equal(t, "Work", resp.Data.Name)
			},
		},
		{
			name:           "ServiceError",
			method:         http.MethodPost,
			url:            "/api/v1/categories",
			body:           `{"name":"Work"}`,
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusInternalServerError,
			setupMock: func(m *MockCategoryService) {
				m.On("CreateCategory", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, assert.AnError)
			},
			assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var errResp schema.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &errResp)
				require.NoError(t, err)
				assert.Equal(t, http.StatusInternalServerError, errResp.StatusCode)
			},
		},
		{
			name:           "Unauthenticated",
			method:         http.MethodPost,
			url:            "/api/v1/categories",
			body:           `{"name":"Work"}`,
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
			assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var errResp schema.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &errResp)
				require.NoError(t, err)
				assert.Equal(t, "UNAUTHORIZED", errResp.Type)
			},
		},
		{
			name:           "InvalidBody",
			method:         http.MethodPost,
			url:            "/api/v1/categories",
			body:           `invalid json`,
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusBadRequest,
			assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var errResp schema.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &errResp)
				require.NoError(t, err)
				assert.Equal(t, "VALIDATION_ERROR", errResp.Type)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCategoryTest(t, tc)
		})
	}
}

func TestGetCategoryByID(t *testing.T) {
	t.Parallel()

	categoryID := uuid.New()

	tests := []categoryTestCase{
		{
			name:           "Success",
			method:         http.MethodGet,
			url:            "/api/v1/categories/" + categoryID.String(),
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusOK,
			setupMock: func(m *MockCategoryService) {
				m.On("GetCategoryByID", mock.Anything, mock.Anything, mock.Anything).
					Return(&schema.Category{ID: categoryID, Name: "Work"}, nil)
			},
			assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp schema.Response[schema.Category]
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, categoryID, resp.Data.ID)
				assert.Equal(t, "Work", resp.Data.Name)
			},
		},
		{
			name:           "ServiceError",
			method:         http.MethodGet,
			url:            "/api/v1/categories/" + categoryID.String(),
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusInternalServerError,
			setupMock: func(m *MockCategoryService) {
				m.On("GetCategoryByID", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, assert.AnError)
			},
		},
		{
			name:           "Unauthenticated",
			method:         http.MethodGet,
			url:            "/api/v1/categories/" + categoryID.String(),
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCategoryTest(t, tc)
		})
	}
}

func TestGetAllCategories(t *testing.T) {
	t.Parallel()

	tests := []categoryTestCase{
		{
			name:           "Success",
			method:         http.MethodGet,
			url:            "/api/v1/categories?page=1&limit=10",
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusOK,
			setupMock: func(m *MockCategoryService) {
				m.On("GetAllCategories", mock.Anything, mock.Anything, mock.Anything).
					Return(&schema.PaginatedResponse[schema.GetCategoriesResponse]{
						Page: 1, Limit: 10, Total: 2, TotalPages: 1,
						Data: []schema.GetCategoriesResponse{
							{ID: uuid.New(), Name: "Work"},
							{ID: uuid.New(), Name: "Personal"},
						},
					}, nil)
			},
			assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp schema.PaginatedResponse[schema.GetCategoriesResponse]
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, 2, resp.Total)
				assert.Len(t, resp.Data, 2)
			},
		},
		{
			name:           "ServiceError",
			method:         http.MethodGet,
			url:            "/api/v1/categories?page=1&limit=10",
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusInternalServerError,
			setupMock: func(m *MockCategoryService) {
				m.On("GetAllCategories", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, assert.AnError)
			},
		},
		{
			name:           "Unauthenticated",
			method:         http.MethodGet,
			url:            "/api/v1/categories?page=1&limit=10",
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCategoryTest(t, tc)
		})
	}
}

func TestUpdateCategory(t *testing.T) {
	t.Parallel()

	categoryID := uuid.New()
	successID := uuid.New()

	tests := []categoryTestCase{
		{
			name:           "Success",
			method:         http.MethodPatch,
			url:            "/api/v1/categories/" + categoryID.String(),
			body:           `{"name":"Updated"}`,
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusOK,
			setupMock: func(m *MockCategoryService) {
				m.On("UpdateCategory", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(&schema.UpdateCategoryResponse{ID: successID, Name: "Updated"}, nil)
			},
			assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp schema.Response[schema.UpdateCategoryResponse]
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, successID, resp.Data.ID)
				assert.Equal(t, "Updated", resp.Data.Name)
			},
		},
		{
			name:           "ServiceError",
			method:         http.MethodPatch,
			url:            "/api/v1/categories/" + categoryID.String(),
			body:           `{"name":"Updated"}`,
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusInternalServerError,
			setupMock: func(m *MockCategoryService) {
				m.On("UpdateCategory", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, assert.AnError)
			},
		},
		{
			name:           "Unauthenticated",
			method:         http.MethodPatch,
			url:            "/api/v1/categories/" + categoryID.String(),
			body:           `{"name":"Updated"}`,
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCategoryTest(t, tc)
		})
	}
}

func TestDeleteCategory(t *testing.T) {
	t.Parallel()

	categoryID := uuid.New()

	tests := []categoryTestCase{
		{
			name:           "Success",
			method:         http.MethodDelete,
			url:            "/api/v1/categories/" + categoryID.String(),
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusNoContent,
			setupMock: func(m *MockCategoryService) {
				m.On("DeleteCategory", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Empty(t, rec.Body.String())
			},
		},
		{
			name:           "ServiceError",
			method:         http.MethodDelete,
			url:            "/api/v1/categories/" + categoryID.String(),
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusInternalServerError,
			setupMock: func(m *MockCategoryService) {
				m.On("DeleteCategory", mock.Anything, mock.Anything, mock.Anything).
					Return(assert.AnError)
			},
		},
		{
			name:           "Unauthenticated",
			method:         http.MethodDelete,
			url:            "/api/v1/categories/" + categoryID.String(),
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCategoryTest(t, tc)
		})
	}
}
