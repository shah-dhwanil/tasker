package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/shah-dhwanil/tasker/internal/database"
	"github.com/shah-dhwanil/tasker/internal/repository"
	"github.com/shah-dhwanil/tasker/internal/schema"
	"github.com/shah-dhwanil/tasker/internal/schema/dto"
	"github.com/shah-dhwanil/tasker/internal/service"
)

type MockTodoRepository struct {
	mock.Mock
}

func (m *MockTodoRepository) Create(ctx context.Context, req *dto.CreateTodoRequest) (*dto.Todo, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.Todo), args.Error(1)
}

func (m *MockTodoRepository) GetByID(ctx context.Context, todoID uuid.UUID, userID string, includeDeleted bool) (*dto.Todo, error) {
	args := m.Called(ctx, todoID, userID, includeDeleted)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.Todo), args.Error(1)
}

func (m *MockTodoRepository) GetAll(ctx context.Context, userID *string, q *dto.GetTodosQuery, includeDeleted bool) ([]dto.TodoListItems, error) {
	args := m.Called(ctx, userID, q, includeDeleted)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.TodoListItems), args.Error(1)
}

func (m *MockTodoRepository) Count(ctx context.Context, userID *string, q *dto.GetTodosQuery, includeDeleted bool) (int, error) {
	args := m.Called(ctx, userID, q, includeDeleted)
	return args.Int(0), args.Error(1)
}

func (m *MockTodoRepository) Update(ctx context.Context, todoID uuid.UUID, userID string, payload *dto.UpdateTodoRequest) (*dto.Todo, error) {
	args := m.Called(ctx, todoID, userID, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.Todo), args.Error(1)
}

func (m *MockTodoRepository) Delete(ctx context.Context, todoID uuid.UUID, userID string) error {
	args := m.Called(ctx, todoID, userID)
	return args.Error(0)
}

func (m *MockTodoRepository) WithExecutor(executor database.DBTX) repository.Todo{
	return m
}

func TestTodoService_Create(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	todoID := uuid.New()
	parentID := uuid.New()

	tests := []struct {
		name    string
		req     *schema.CreateTodoRequest
		setup   func(*MockTodoRepository)
		wantErr error
		check   func(*testing.T, *schema.Todo)
	}{
		{
			name: "success",
			req: &schema.CreateTodoRequest{
				Title:    "Test Todo",
				Status:   "pending",
				Priority: 1,
			},
			setup: func(m *MockTodoRepository) {
				dtoTodo := &dto.Todo{
					ID:        todoID,
					UserID:    userID,
					Title:     "Test Todo",
					Status:    "pending",
					Priority:  1,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("Create", ctx, mock.MatchedBy(func(r *dto.CreateTodoRequest) bool {
					return r.UserID == userID && r.Title == "Test Todo"
				})).Return(dtoTodo, nil)
			},
			check: func(t *testing.T, result *schema.Todo) {
				assert.Equal(t, todoID, result.ID)
				assert.Equal(t, userID, result.UserID)
				assert.Equal(t, "Test Todo", result.Title)
				assert.Equal(t, schema.TodoStatus("pending"), result.Status)
			},
		},
		{
			name: "parent completed reactivates",
			req: &schema.CreateTodoRequest{
				Title:    "Child Todo",
				Status:   "pending",
				Priority: 1,
				ParentID: &parentID,
			},
			setup: func(m *MockTodoRepository) {
				dtoTodo := &dto.Todo{
					ID:        todoID,
					UserID:    userID,
					Title:     "Child Todo",
					Status:    "pending",
					Priority:  1,
					ParentID:  &parentID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				dtoParent := &dto.Todo{
					ID:          parentID,
					UserID:      userID,
					Title:       "Parent Todo",
					Status:      "completed",
					CompletedAt: ptr(time.Now()),
				}
				dtoUpdatedParent := &dto.Todo{
					ID:     parentID,
					UserID: userID,
					Title:  "Parent Todo",
					Status: "in_progress",
				}
				m.On("Create", ctx, mock.Anything).Return(dtoTodo, nil)
				m.On("GetByID", ctx, parentID, userID, false).Return(dtoParent, nil)
				m.On("Update", ctx, parentID, userID, mock.MatchedBy(func(r *dto.UpdateTodoRequest) bool {
					return r.Status != nil && *r.Status == "in_progress" &&
						r.CompletedAt.IsSet() && r.CompletedAt.Data == nil
				})).Return(dtoUpdatedParent, nil)
			},
			check: func(t *testing.T, result *schema.Todo) {
				assert.Equal(t, todoID, result.ID)
			},
		},
		{
			name: "repo error",
			req: &schema.CreateTodoRequest{
				Title:    "Test Todo",
				Status:   "pending",
				Priority: 1,
			},
			setup: func(m *MockTodoRepository) {
				m.On("Create", ctx, mock.Anything).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTodoRepository)
			svc := service.NewTodoService(mockRepo, &MockDBTX{})

			tt.setup(mockRepo)

			result, err := svc.Create(ctx, userID, tt.req)

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

func TestTodoService_GetByID(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	todoID := uuid.New()
	childID := uuid.New()

	tests := []struct {
		name    string
		setup   func(*MockTodoRepository)
		wantErr error
		check   func(*testing.T, *schema.Todo)
	}{
		{
			name: "success with children",
			setup: func(m *MockTodoRepository) {
				dtoTodo := &dto.Todo{
					ID:        todoID,
					UserID:    userID,
					Title:     "Parent Todo",
					Status:    "pending",
					Priority:  1,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				children := []dto.TodoListItems{
					{ID: childID, Title: "Child 1", Status: "pending", Priority: 1},
				}
				userIDStr := userID
				m.On("GetByID", mock.Anything, todoID, userID, false).Return(dtoTodo, nil)
				m.On("GetAll", mock.Anything, mock.MatchedBy(func(u *string) bool {
					return *u == userIDStr
				}), mock.MatchedBy(func(q *dto.GetTodosQuery) bool {
					return q.ParentID != nil && *q.ParentID == todoID
				}), false).Return(children, nil)
			},
			check: func(t *testing.T, result *schema.Todo) {
				assert.Equal(t, todoID, result.ID)
				require.Len(t, result.Children, 1)
				assert.Equal(t, childID, result.Children[0].ID)
			},
		},
		{
			name: "repo error",
			setup: func(m *MockTodoRepository) {
				m.On("GetByID", mock.Anything, todoID, userID, false).Return(nil, assert.AnError)
				m.On("GetAll", mock.Anything, &userID, mock.Anything, false).Return(nil, assert.AnError).Maybe()
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTodoRepository)
			svc := service.NewTodoService(mockRepo, &MockDBTX{})

			tt.setup(mockRepo)

			result, err := svc.GetByID(ctx, todoID, userID, false)

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

func TestTodoService_Update(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	todoID := uuid.New()
	parentID := uuid.New()
	oldParentID := uuid.New()
	newParentID := uuid.New()
	newTitle := "Updated Title"
	newStatus := "completed"

	tests := []struct {
		name    string
		req     *schema.UpdateTodoRequest
		setup   func(*MockTodoRepository)
		wantErr error
		check   func(*testing.T, *schema.Todo)
	}{
		{
			name: "success",
			req: &schema.UpdateTodoRequest{
				Title: &newTitle,
			},
			setup: func(m *MockTodoRepository) {
				oldTodo := &dto.Todo{
					ID:        todoID,
					UserID:    userID,
					Title:     "Original",
					Status:    "pending",
					Priority:  1,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				updatedDTO := &dto.Todo{
					ID:        todoID,
					UserID:    userID,
					Title:     newTitle,
					Status:    "pending",
					Priority:  1,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("GetByID", ctx, todoID, userID, false).Return(oldTodo, nil)
				m.On("Update", ctx, todoID, userID, mock.MatchedBy(func(r *dto.UpdateTodoRequest) bool {
					return r.Title != nil && *r.Title == newTitle
				})).Return(updatedDTO, nil)
			},
			check: func(t *testing.T, result *schema.Todo) {
				assert.Equal(t, newTitle, result.Title)
			},
		},
		{
			name: "status completed all children done parent completes",
			req: &schema.UpdateTodoRequest{
				Status: &newStatus,
			},
			setup: func(m *MockTodoRepository) {
				oldTodo := &dto.Todo{
					ID:        todoID,
					UserID:    userID,
					Title:     "Child",
					Status:    "pending",
					Priority:  1,
					ParentID:  &parentID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				updatedDTO := &dto.Todo{
					ID:        todoID,
					UserID:    userID,
					Title:     "Child",
					Status:    "completed",
					Priority:  1,
					ParentID:  &parentID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				children := []dto.TodoListItems{
					{ID: todoID, Title: "Child", Status: "completed", Priority: 1},
				}
				parentUpdated := &dto.Todo{
					ID:     parentID,
					UserID: userID,
					Title:  "Parent",
					Status: "completed",
				}
				m.On("GetByID", ctx, todoID, userID, false).Return(oldTodo, nil)
				m.On("Update", ctx, todoID, userID, mock.MatchedBy(func(r *dto.UpdateTodoRequest) bool {
					return r.Status != nil && *r.Status == "completed"
				})).Return(updatedDTO, nil)
				m.On("GetAll", mock.Anything, mock.Anything, mock.MatchedBy(func(q *dto.GetTodosQuery) bool {
					return q.ParentID != nil && *q.ParentID == parentID
				}), false).Return(children, nil)
				m.On("Update", ctx, parentID, userID, mock.MatchedBy(func(r *dto.UpdateTodoRequest) bool {
					return r.Status != nil && *r.Status == "completed" &&
						r.CompletedAt.IsSet() && r.CompletedAt.Data != nil
				})).Return(parentUpdated, nil)
			},
			check: func(t *testing.T, result *schema.Todo) {
				assert.Equal(t, "completed", string(result.Status))
			},
		},
		{
			name: "parent changed recalculates both",
			req: &schema.UpdateTodoRequest{
				Status:   &newStatus,
				ParentID: schema.Nullable[*uuid.UUID]{IsExplicitlySet: true, Data: &newParentID},
			},
			setup: func(m *MockTodoRepository) {
				oldTodo := &dto.Todo{
					ID:        todoID,
					UserID:    userID,
					Title:     "Child",
					Status:    "pending",
					Priority:  1,
					ParentID:  &oldParentID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				updatedDTO := &dto.Todo{
					ID:        todoID,
					UserID:    userID,
					Title:     "Child",
					Status:    "completed",
					Priority:  1,
					ParentID:  &newParentID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				completedParent := &dto.Todo{
					ID:     oldParentID,
					UserID: userID,
					Title:  "Old Parent",
					Status: "completed",
				}
				m.On("GetByID", ctx, todoID, userID, false).Return(oldTodo, nil)
				m.On("Update", ctx, todoID, userID, mock.Anything).Return(updatedDTO, nil)
				m.On("GetAll", ctx, mock.Anything, mock.MatchedBy(func(q *dto.GetTodosQuery) bool {
					return q.ParentID != nil && *q.ParentID == oldParentID
				}), false).Return([]dto.TodoListItems{}, nil)
				m.On("GetByID", ctx, oldParentID, userID, false).Return(completedParent, nil)
				m.On("GetAll", ctx, mock.Anything, mock.MatchedBy(func(q *dto.GetTodosQuery) bool {
					return q.ParentID != nil && *q.ParentID == newParentID
				}), false).Return([]dto.TodoListItems{
					{ID: todoID, Title: "Child", Status: "completed", Priority: 1},
				}, nil)
				m.On("Update", ctx, oldParentID, userID, mock.MatchedBy(func(r *dto.UpdateTodoRequest) bool {
					return r.Status != nil && *r.Status == "in_progress" &&
						r.CompletedAt.IsSet() && r.CompletedAt.Data == nil
				})).Return(&dto.Todo{ID: oldParentID, UserID: userID, Status: "in_progress"}, nil)
				m.On("Update", ctx, newParentID, userID, mock.MatchedBy(func(r *dto.UpdateTodoRequest) bool {
					return r.Status != nil && *r.Status == "completed" &&
						r.CompletedAt.IsSet() && r.CompletedAt.Data != nil
				})).Return(&dto.Todo{ID: newParentID, UserID: userID, Status: "completed"}, nil)
			},
			check: func(t *testing.T, result *schema.Todo) {
				assert.Equal(t, "completed", string(result.Status))
			},
		},
		{
			name: "repo error",
			req: &schema.UpdateTodoRequest{
				Title: ptr("Updated"),
			},
			setup: func(m *MockTodoRepository) {
				oldTodo := &dto.Todo{
					ID:     todoID,
					UserID: userID,
					Title:  "Original",
					Status: "pending",
				}
				m.On("GetByID", ctx, todoID, userID, false).Return(oldTodo, nil)
				m.On("Update", ctx, todoID, userID, mock.Anything).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTodoRepository)
			svc := service.NewTodoService(mockRepo, &MockDBTX{})

			tt.setup(mockRepo)

			result, err := svc.Update(ctx, todoID, userID, tt.req)

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

func TestTodoService_GetAll(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	page := 1
	limit := 10

	tests := []struct {
		name    string
		query   *schema.GetTodosQuery
		setup   func(*MockTodoRepository)
		wantErr error
		check   func(*testing.T, *schema.PaginatedResponse[schema.TodoListItems])
	}{
		{
			name:  "success",
			query: &schema.GetTodosQuery{Page: &page, Limit: &limit},
			setup: func(m *MockTodoRepository) {
				todos := []schema.TodoListItems{
					{ID: uuid.New(), Title: "Todo 1", Status: "pending", Priority: 1},
					{ID: uuid.New(), Title: "Todo 2", Status: "completed", Priority: 2},
				}
				userIDStr := userID
				m.On("GetAll", mock.Anything, mock.MatchedBy(func(u *string) bool {
					return *u == userIDStr
				}), mock.MatchedBy(func(q *dto.GetTodosQuery) bool {
					return q.Limit == limit && q.Offset == 0
				}), false).Return([]dto.TodoListItems{
					{ID: todos[0].ID, Title: todos[0].Title, Status: todos[0].Status, Priority: todos[0].Priority},
					{ID: todos[1].ID, Title: todos[1].Title, Status: todos[1].Status, Priority: todos[1].Priority},
				}, nil)
				m.On("Count", mock.Anything, mock.MatchedBy(func(u *string) bool {
					return *u == userIDStr
				}), mock.MatchedBy(func(q *dto.GetTodosQuery) bool {
					return q.Limit == limit && q.Offset == 0
				}), false).Return(2, nil)
			},
			check: func(t *testing.T, result *schema.PaginatedResponse[schema.TodoListItems]) {
				assert.Len(t, result.Data, 2)
				assert.Equal(t, 1, result.Page)
				assert.Equal(t, 10, result.Limit)
				assert.Equal(t, 2, result.Total)
				assert.Equal(t, 1, result.TotalPages)
			},
		},
		{
			name:  "repo error on get all",
			query: &schema.GetTodosQuery{Page: &page, Limit: &limit},
			setup: func(m *MockTodoRepository) {
				userIDStr := userID
				m.On("GetAll", mock.Anything, mock.MatchedBy(func(u *string) bool {
					return *u == userIDStr
				}), mock.Anything, false).Return(nil, assert.AnError)
				m.On("Count", mock.Anything, mock.MatchedBy(func(u *string) bool {
					return *u == userIDStr
				}), mock.Anything, false).Return(0, nil)
			},
			wantErr: assert.AnError,
		},
		{
			name:  "repo error on count",
			query: &schema.GetTodosQuery{Page: &page, Limit: &limit},
			setup: func(m *MockTodoRepository) {
				userIDStr := userID
				m.On("GetAll", mock.Anything, mock.MatchedBy(func(u *string) bool {
					return *u == userIDStr
				}), mock.Anything, false).Return([]dto.TodoListItems{}, nil)
				m.On("Count", mock.Anything, mock.MatchedBy(func(u *string) bool {
					return *u == userIDStr
				}), mock.Anything, false).Return(0, assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTodoRepository)
			svc := service.NewTodoService(mockRepo, &MockDBTX{})

			tt.setup(mockRepo)

			result, err := svc.GetAll(ctx, userID, tt.query)

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

func TestTodoService_Delete(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	todoID := uuid.New()

	tests := []struct {
		name    string
		setup   func(*MockTodoRepository)
		wantErr error
	}{
		{
			name: "success",
			setup: func(m *MockTodoRepository) {
				m.On("Delete", ctx, todoID, userID).Return(nil)
			},
		},
		{
			name: "repo error",
			setup: func(m *MockTodoRepository) {
				m.On("Delete", ctx, todoID, userID).Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTodoRepository)
			svc := service.NewTodoService(mockRepo, &MockDBTX{})

			tt.setup(mockRepo)

			err := svc.Delete(ctx, todoID, userID)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}