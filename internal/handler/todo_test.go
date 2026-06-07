package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	errorhandler "github.com/shah-dhwanil/tasker/internal/error_handler"
	"github.com/shah-dhwanil/tasker/internal/schema"
)

type MockTodoService struct {
	mock.Mock
}

func (m *MockTodoService) Create(ctx context.Context, userID string, req *schema.CreateTodoRequest) (*schema.Todo, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Todo), args.Error(1)
}

func (m *MockTodoService) GetByID(ctx context.Context, todoID uuid.UUID, userID string, includeDeleted bool) (*schema.Todo, error) {
	args := m.Called(ctx, todoID, userID, includeDeleted)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Todo), args.Error(1)
}

func (m *MockTodoService) GetAll(ctx context.Context, userID string, query *schema.GetTodosQuery) (*schema.PaginatedResponse[schema.TodoListItems], error) {
	args := m.Called(ctx, userID, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.PaginatedResponse[schema.TodoListItems]), args.Error(1)
}

func (m *MockTodoService) Update(ctx context.Context, todoID uuid.UUID, userID string, payload *schema.UpdateTodoRequest) (*schema.Todo, error) {
	args := m.Called(ctx, todoID, userID, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Todo), args.Error(1)
}

func (m *MockTodoService) Delete(ctx context.Context, todoID uuid.UUID, userID string) error {
	args := m.Called(ctx, todoID, userID)
	return args.Error(0)
}

func setupTodoServer(t *testing.T, mockSvc *MockTodoService, userID *uuid.UUID) *echo.Echo {
	t.Helper()

	e := echo.New()
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if !c.Response().Committed {
			res := errorhandler.HandleError(err)
			c.JSON(res.StatusCode, res)
		}
	}

	h := &TodoHandler{TodoService: mockSvc}
	group := e.Group("/api/v1/todos")
	if userID != nil {
		group.Use(authMiddleware(*userID))
	}

	group.POST("", Handle(h.Create, &schema.CreateTodoRequest{}))
	group.GET("/:todoId", Handle(h.GetByID, &TodoIDRequest{}))
	group.GET("", Handle(h.GetAll, &schema.GetTodosQuery{}))
	group.PATCH("/:todoId", Handle(h.Update, &UpdateTodoRequest{}))
	group.DELETE("/:todoId", Handle(h.Delete, &TodoIDRequest{}))

	return e
}

type todoTestCase struct {
	name           string
	method         string
	url            string
	body           string
	userID         *uuid.UUID
	setupMock      func(*MockTodoService)
	expectedStatus int
	assertResponse func(*testing.T, *httptest.ResponseRecorder)
}

func runTodoTest(t *testing.T, tc todoTestCase) {
	t.Helper()

	mockSvc := new(MockTodoService)
	e := setupTodoServer(t, mockSvc, tc.userID)

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

func TestCreateTodo(t *testing.T) {
	t.Parallel()

	todoID := uuid.New()

	tests := []todoTestCase{
		{
			name:           "Success",
			method:         http.MethodPost,
			url:            "/api/v1/todos",
			body:           `{"title":"Test Todo","status":"pending","priority":3}`,
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusCreated,
			setupMock: func(m *MockTodoService) {
				m.On("Create", mock.Anything, mock.Anything, mock.Anything).
					Return(&schema.Todo{ID: todoID, Title: "Test Todo"}, nil)
			},
			assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp schema.Response[schema.Todo]
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, http.StatusCreated, rec.Code)
				assert.Equal(t, todoID, resp.Data.ID)
				assert.Equal(t, "Test Todo", resp.Data.Title)
			},
		},
		{
			name:           "ServiceError",
			method:         http.MethodPost,
			url:            "/api/v1/todos",
			body:           `{"title":"Test Todo","status":"pending","priority":3}`,
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusInternalServerError,
			setupMock: func(m *MockTodoService) {
				m.On("Create", mock.Anything, mock.Anything, mock.Anything).
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
			url:            "/api/v1/todos",
			body:           `{"title":"Test Todo","status":"pending","priority":3}`,
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
			url:            "/api/v1/todos",
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
			runTodoTest(t, tc)
		})
	}
}

func TestGetTodoByID(t *testing.T) {
	t.Parallel()

	todoID := uuid.New()

	tests := []todoTestCase{
		{
			name:           "Success",
			method:         http.MethodGet,
			url:            "/api/v1/todos/" + todoID.String(),
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusOK,
			setupMock: func(m *MockTodoService) {
				m.On("GetByID", mock.Anything, todoID, mock.Anything, false).
					Return(&schema.Todo{ID: todoID, Title: "Test Todo"}, nil)
			},
			assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp schema.Response[schema.Todo]
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, todoID, resp.Data.ID)
				assert.Equal(t, "Test Todo", resp.Data.Title)
			},
		},
		{
			name:           "ServiceError",
			method:         http.MethodGet,
			url:            "/api/v1/todos/" + todoID.String(),
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusInternalServerError,
			setupMock: func(m *MockTodoService) {
				m.On("GetByID", mock.Anything, todoID, mock.Anything, false).
					Return(nil, assert.AnError)
			},
		},
		{
			name:           "Unauthenticated",
			method:         http.MethodGet,
			url:            "/api/v1/todos/" + todoID.String(),
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runTodoTest(t, tc)
		})
	}
}

func TestGetAllTodos(t *testing.T) {
	t.Parallel()

	parentID := uuid.New()

	tests := []todoTestCase{
		{
			name:           "Success",
			method:         http.MethodGet,
			url:            "/api/v1/todos?page=1&limit=10",
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusOK,
			setupMock: func(m *MockTodoService) {
				m.On("GetAll", mock.Anything, mock.Anything, mock.Anything).
					Return(&schema.PaginatedResponse[schema.TodoListItems]{
						Page: 1, Limit: 10, Total: 2, TotalPages: 1,
						Data: []schema.TodoListItems{
							{ID: uuid.New(), Title: "Todo 1"},
							{ID: uuid.New(), Title: "Todo 2"},
						},
					}, nil)
			},
			assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp schema.PaginatedResponse[schema.TodoListItems]
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, 2, resp.Total)
				assert.Len(t, resp.Data, 2)
			},
		},
		{
			name:           "WithParentID",
			method:         http.MethodGet,
			url:            "/api/v1/todos?parent_id=" + parentID.String(),
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusOK,
			setupMock: func(m *MockTodoService) {
				m.On("GetAll", mock.Anything, mock.Anything, mock.MatchedBy(func(q *schema.GetTodosQuery) bool {
					return q.ParentID != nil && *q.ParentID == parentID
				})).
					Return(&schema.PaginatedResponse[schema.TodoListItems]{
						Page: 1, Limit: 10, Total: 1, TotalPages: 1,
						Data: []schema.TodoListItems{
							{ID: uuid.New(), Title: "Child Todo"},
						},
					}, nil)
			},
			assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp schema.PaginatedResponse[schema.TodoListItems]
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, 1, resp.Total)
			},
		},
		{
			name:           "ServiceError",
			method:         http.MethodGet,
			url:            "/api/v1/todos?page=1&limit=10",
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusInternalServerError,
			setupMock: func(m *MockTodoService) {
				m.On("GetAll", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, assert.AnError)
			},
		},
		{
			name:           "Unauthenticated",
			method:         http.MethodGet,
			url:            "/api/v1/todos?page=1&limit=10",
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runTodoTest(t, tc)
		})
	}
}

func TestUpdateTodo(t *testing.T) {
	t.Parallel()

	todoID := uuid.New()

	tests := []todoTestCase{
		{
			name:           "Success",
			method:         http.MethodPatch,
			url:            "/api/v1/todos/" + todoID.String(),
			body:           `{"title":"Updated Todo"}`,
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusOK,
			setupMock: func(m *MockTodoService) {
				m.On("Update", mock.Anything, todoID, mock.Anything, mock.Anything).
					Return(&schema.Todo{ID: todoID, Title: "Updated Todo"}, nil)
			},
			assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp schema.Response[schema.Todo]
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, todoID, resp.Data.ID)
				assert.Equal(t, "Updated Todo", resp.Data.Title)
			},
		},
		{
			name:           "ServiceError",
			method:         http.MethodPatch,
			url:            "/api/v1/todos/" + todoID.String(),
			body:           `{"title":"Updated Todo"}`,
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusInternalServerError,
			setupMock: func(m *MockTodoService) {
				m.On("Update", mock.Anything, todoID, mock.Anything, mock.Anything).
					Return(nil, assert.AnError)
			},
		},
		{
			name:           "Unauthenticated",
			method:         http.MethodPatch,
			url:            "/api/v1/todos/" + todoID.String(),
			body:           `{"title":"Updated Todo"}`,
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runTodoTest(t, tc)
		})
	}
}

func TestDeleteTodo(t *testing.T) {
	t.Parallel()

	todoID := uuid.New()

	tests := []todoTestCase{
		{
			name:           "Success",
			method:         http.MethodDelete,
			url:            "/api/v1/todos/" + todoID.String(),
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusNoContent,
			setupMock: func(m *MockTodoService) {
				m.On("Delete", mock.Anything, todoID, mock.Anything).Return(nil)
			},
			assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Empty(t, rec.Body.String())
			},
		},
		{
			name:           "ServiceError",
			method:         http.MethodDelete,
			url:            "/api/v1/todos/" + todoID.String(),
			userID:         ptr(uuid.New()),
			expectedStatus: http.StatusInternalServerError,
			setupMock: func(m *MockTodoService) {
				m.On("Delete", mock.Anything, todoID, mock.Anything).
					Return(assert.AnError)
			},
		},
		{
			name:           "Unauthenticated",
			method:         http.MethodDelete,
			url:            "/api/v1/todos/" + todoID.String(),
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runTodoTest(t, tc)
		})
	}
}
