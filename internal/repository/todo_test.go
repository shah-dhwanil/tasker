package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shah-dhwanil/tasker/internal/database"
	"github.com/shah-dhwanil/tasker/internal/errors"
	"github.com/shah-dhwanil/tasker/internal/repository"
	"github.com/shah-dhwanil/tasker/internal/schema"
	"github.com/shah-dhwanil/tasker/internal/schema/dto"
	pkgTesting "github.com/shah-dhwanil/tasker/internal/testing"
)

func getTodoRepository(t *testing.T, repo *repository.Repository, tx database.Transaction) repository.Todo {
	t.Helper()
	return repo.TodoRepository.WithExecutor(tx)
}



func createTestTodo(t *testing.T, ctx context.Context, todoRepo repository.Todo, categoryRepo *repository.CategoryRepository, userID string, payload *dto.CreateTodoRequest) *dto.Todo {
	t.Helper()
	cat := createTestCategory(t, ctx, categoryRepo, userID, &dto.CreateCategoryRequest{
		Name: "Test-Cat-" + uuid.New().String()[:8],
	})
	payload.CategoryID = &cat.ID
	payload.UserID = userID
	resp, err := todoRepo.Create(ctx, payload)
	require.NoError(t, err)
	require.NotNil(t, resp)
	return resp
}

func TestTodoRepository_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := pkgTesting.Services()

	tests := []struct {
		name string
		run  func(t *testing.T, tx database.Transaction, repo *repository.Repository)
	}{
		{
			name: "Success",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)
				desc := "test description"
				dueDate := time.Now().Add(24 * time.Hour).Truncate(time.Microsecond)
				meta := map[string]any{"env": "test"}
				parent := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Parent", Status: "pending", Priority: 1,
				})

				resp, err := todoRepo.Create(ctx, &dto.CreateTodoRequest{
					UserID:      userID,
					CategoryID:  parent.CategoryID,
					Title:       "My Todo",
					Description: &desc,
					Status:      "pending",
					Priority:    3,
					DueDate:     &dueDate,
					ParentID:    &parent.ID,
					Metadata:    meta,
				})

				require.NoError(t, err)
				require.NotNil(t, resp)

				assert.NotZero(t, resp.ID)
				assert.Equal(t, userID, resp.UserID)
				assert.Equal(t, "My Todo", resp.Title)
				require.NotNil(t, resp.Description)
				assert.Equal(t, "test description", *resp.Description)
				assert.Equal(t, "pending", resp.Status)
				assert.Equal(t, 3, resp.Priority)
				require.NotNil(t, resp.DueDate)
				assert.WithinDuration(t, dueDate, *resp.DueDate, time.Millisecond)
				require.NotNil(t, resp.ParentID)
				assert.Equal(t, parent.ID, *resp.ParentID)
				require.NotNil(t, resp.Metadata)
				assert.Equal(t, "test", resp.Metadata["env"])
				assert.Nil(t, resp.CompletedAt)
				assert.False(t, resp.IsDeleted)
				assert.False(t, resp.CreatedAt.IsZero())
				assert.False(t, resp.UpdatedAt.IsZero())
			},
		},
		{
			name: "RequiredOnly",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				todoRepo := getTodoRepository(t, repo, tx)

				resp, err := todoRepo.Create(ctx, &dto.CreateTodoRequest{
					UserID:   "test-user-id",
					Title:    "Minimal",
					Status:   "pending",
					Priority: 1,
				})

				require.NoError(t, err)
				require.NotNil(t, resp)

				assert.Equal(t, "Minimal", resp.Title)
				assert.Equal(t, "pending", resp.Status)
				assert.Equal(t, 1, resp.Priority)
				assert.Nil(t, resp.Description)
				assert.Nil(t, resp.DueDate)
				assert.Nil(t, resp.ParentID)
				assert.Nil(t, resp.CompletedAt)
				assert.Nil(t, resp.CategoryID)
				assert.Nil(t, resp.Metadata)
			},
		},
		{
			name: "WithMetadata",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				todoRepo := getTodoRepository(t, repo, tx)
				meta := map[string]any{"key": "value", "count": 42}

				resp, err := todoRepo.Create(ctx, &dto.CreateTodoRequest{
					UserID:   "test-user-id",
					Title:    "Metadata Test",
					Status:   "in_progress",
					Priority: 2,
					Metadata: meta,
				})

				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotNil(t, resp.Metadata)
				assert.Equal(t, "value", resp.Metadata["key"])
				assert.Equal(t, float64(42), resp.Metadata["count"])
			},
		},
		{
			name: "WithDueDate",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				todoRepo := getTodoRepository(t, repo, tx)
				dueDate := time.Now().Add(48 * time.Hour).Truncate(time.Microsecond)

				resp, err := todoRepo.Create(ctx, &dto.CreateTodoRequest{
					UserID:   "test-user-id",
					Title:    "Due Date Test",
					Status:   "pending",
					Priority: 1,
					DueDate:  &dueDate,
				})

				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotNil(t, resp.DueDate)
				assert.WithinDuration(t, dueDate, *resp.DueDate, time.Millisecond)
			},
		},
		{
			name: "InvalidCategory",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				invalidCatID := uuid.New()

				_, err := todoRepo.Create(ctx, &dto.CreateTodoRequest{
					UserID:     userID,
					CategoryID: &invalidCatID,
					Title:      "Bad Category",
					Status:     "pending",
					Priority:   1,
				})

				require.Error(t, err)
				var appErr *errors.AppError
				require.ErrorAs(t, err, &appErr)
				assert.Equal(t, errors.Validation, appErr.Type)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx database.Transaction) {
				repo := repository.New(svc.DB())
				tt.run(t, tx, repo)
			})
		})
	}
}

func TestTodoRepository_GetByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := pkgTesting.Services()

	tests := []struct {
		name string
		run  func(t *testing.T, tx database.Transaction, repo *repository.Repository)
	}{
		{
			name: "Success",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)
				desc := "fetch description"
				dueDate := time.Now().Add(24 * time.Hour).Truncate(time.Microsecond)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title:       "Fetch Me",
					Description: &desc,
					Status:      "completed",
					Priority:    5,
					DueDate:     &dueDate,
				})

				fetched, err := todoRepo.GetByID(ctx, created.ID, userID, false)

				require.NoError(t, err)
				require.NotNil(t, fetched)

				assert.Equal(t, created.ID, fetched.ID)
				assert.Equal(t, userID, fetched.UserID)
				assert.Equal(t, "Fetch Me", fetched.Title)
				assert.Equal(t, "completed", fetched.Status)
				assert.Equal(t, 5, fetched.Priority)
				require.NotNil(t, fetched.Description)
				assert.Equal(t, "fetch description", *fetched.Description)
				require.NotNil(t, fetched.DueDate)
				assert.WithinDuration(t, dueDate, *fetched.DueDate, time.Millisecond)
			},
		},
		{
			name: "NotFound",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				todoRepo := getTodoRepository(t, repo, tx)

				_, err := todoRepo.GetByID(ctx, uuid.New(), "non-existent-user", false)

				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
		{
			name: "WrongUser",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				ownerID := "owner-id"
				otherID := "other-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, ownerID, &dto.CreateTodoRequest{
					Title: "Owned Todo", Status: "pending", Priority: 1,
				})

				_, err := todoRepo.GetByID(ctx, created.ID, otherID, false)

				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
		{
			name: "DeletedExcluded",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "To Delete", Status: "pending", Priority: 1,
				})

				require.NoError(t, todoRepo.Delete(ctx, created.ID, userID))

				_, err := todoRepo.GetByID(ctx, created.ID, userID, false)

				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
		{
			name: "DeletedIncluded",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "To Fetch Deleted", Status: "pending", Priority: 1,
				})

				require.NoError(t, todoRepo.Delete(ctx, created.ID, userID))

				fetched, err := todoRepo.GetByID(ctx, created.ID, userID, true)

				require.NoError(t, err)
				require.NotNil(t, fetched)
				assert.Equal(t, created.ID, fetched.ID)
				assert.True(t, fetched.IsDeleted)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx database.Transaction) {
				repo := repository.New(svc.DB())
				tt.run(t, tx, repo)
			})
		})
	}
}

func TestTodoRepository_GetAll(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := pkgTesting.Services()

	tests := []struct {
		name string
		run  func(t *testing.T, tx database.Transaction, repo *repository.Repository)
	}{
		{
			name: "DefaultPagination",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "A", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "B", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "C", Status: "pending", Priority: 1,
				})

				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10}

				todos, err := todoRepo.GetAll(ctx, &userIDStr, query, true)

				require.NoError(t, err)
				assert.Len(t, todos, 3)
			},
		},
		{
			name: "FilterByUserID",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userA, userB := "user-a", "user-b"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userA, &dto.CreateTodoRequest{
					Title: "A's", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userB, &dto.CreateTodoRequest{
					Title: "B's", Status: "pending", Priority: 1,
				})

				query := &dto.GetTodosQuery{Offset: 0, Limit: 10}

				todos, err := todoRepo.GetAll(ctx, &userA, query, true)

				require.NoError(t, err)
				require.Len(t, todos, 1)
				assert.Equal(t, "A's", todos[0].Title)
			},
		},
		{
			name: "FilterByStatus",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Pending", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Completed", Status: "completed", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "In Progress", Status: "in_progress", Priority: 1,
				})

				status := "completed"
				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10, Status: &status}

				todos, err := todoRepo.GetAll(ctx, &userIDStr, query, true)

				require.NoError(t, err)
				require.Len(t, todos, 1)
				assert.Equal(t, "Completed", todos[0].Title)
				assert.Equal(t, "completed", todos[0].Status)
			},
		},
		{
			name: "FilterByPriority",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "High", Status: "pending", Priority: 5,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Low", Status: "pending", Priority: 1,
				})

				priority := 5
				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10, Priority: &priority}

				todos, err := todoRepo.GetAll(ctx, &userIDStr, query, true)

				require.NoError(t, err)
				require.Len(t, todos, 1)
				assert.Equal(t, "High", todos[0].Title)
				assert.Equal(t, 5, todos[0].Priority)
			},
		},
		{
			name: "FilterByCategory",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				cat1 := createTestCategory(t, ctx, categoryRepo, userID, &dto.CreateCategoryRequest{
					Name: "Category-One",
				})
				cat2 := createTestCategory(t, ctx, categoryRepo, userID, &dto.CreateCategoryRequest{
					Name: "Category-Two",
				})

				_, err := todoRepo.Create(ctx, &dto.CreateTodoRequest{
					Title: "In Cat1", Status: "pending", Priority: 1,
					UserID: userID, CategoryID: &cat1.ID,
				})
				require.NoError(t, err)

				_, err = todoRepo.Create(ctx, &dto.CreateTodoRequest{
					Title: "In Cat2", Status: "pending", Priority: 1,
					UserID: userID, CategoryID: &cat2.ID,
				})
				require.NoError(t, err)

				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10, CategoryID: &cat1.ID}

				todos, err := todoRepo.GetAll(ctx, &userIDStr, query, true)

				require.NoError(t, err)
				require.Len(t, todos, 1)
				assert.Equal(t, "In Cat1", todos[0].Title)
				require.NotNil(t, todos[0].CategoryID)
				assert.Equal(t, cat1.ID, *todos[0].CategoryID)
			},
		},
		{
			name: "Search",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Shopping List", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Work Tasks", Status: "pending", Priority: 1,
				})

				search := "Shopping"
				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10, Search: &search}

				todos, err := todoRepo.GetAll(ctx, &userIDStr, query, true)

				require.NoError(t, err)
				require.Len(t, todos, 1)
				assert.Equal(t, "Shopping List", todos[0].Title)
			},
		},
		{
			name: "OrderByTitle",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Gamma", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Alpha", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Beta", Status: "pending", Priority: 1,
				})

				userIDStr := userID
				query := &dto.GetTodosQuery{
					Offset:  0,
					Limit:   10,
					OrderBy: []string{"title"},
				}

				todos, err := todoRepo.GetAll(ctx, &userIDStr, query, true)

				require.NoError(t, err)
				require.Len(t, todos, 3)
				assert.Equal(t, "Alpha", todos[0].Title)
				assert.Equal(t, "Beta", todos[1].Title)
				assert.Equal(t, "Gamma", todos[2].Title)
			},
		},
		{
			name: "ExcludeDeleted",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Active", Status: "pending", Priority: 1,
				})
				deleted := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Deleted", Status: "pending", Priority: 1,
				})

				require.NoError(t, todoRepo.Delete(ctx, deleted.ID, userID))

				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10}

				todos, err := todoRepo.GetAll(ctx, &userIDStr, query, false)

				require.NoError(t, err)
				require.Len(t, todos, 1)
				assert.Equal(t, "Active", todos[0].Title)
			},
		},
		{
			name: "IncludeDeleted",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Active", Status: "pending", Priority: 1,
				})
				deleted := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Deleted", Status: "pending", Priority: 1,
				})

				require.NoError(t, todoRepo.Delete(ctx, deleted.ID, userID))

				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10}

				todos, err := todoRepo.GetAll(ctx, &userIDStr, query, true)

				require.NoError(t, err)
				require.Len(t, todos, 2)
			},
		},
		{
			name: "NoResults",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				todoRepo := getTodoRepository(t, repo, tx)
				nonExistentUser := "non-existent-user"
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10}

				todos, err := todoRepo.GetAll(ctx, &nonExistentUser, query, true)

				require.NoError(t, err)
				assert.Empty(t, todos)
			},
		},
		{
			name: "CustomPagination",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				names := []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"}
				for _, name := range names {
					createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
						Title: name, Status: "pending", Priority: 1,
					})
				}

				userIDStr := userID
				query := &dto.GetTodosQuery{
					Offset:  2,
					Limit:   2,
					OrderBy: []string{"title"},
				}

				todos, err := todoRepo.GetAll(ctx, &userIDStr, query, true)

				require.NoError(t, err)
				require.Len(t, todos, 2)
				assert.Equal(t, "Delta", todos[0].Title)
				assert.Equal(t, "Epsilon", todos[1].Title)
			},
		},
		{
			name: "NilUserID",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, "user-a", &dto.CreateTodoRequest{
					Title: "UserA", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, "user-b", &dto.CreateTodoRequest{
					Title: "UserB", Status: "pending", Priority: 1,
				})

				query := &dto.GetTodosQuery{Offset: 0, Limit: 10}

				todos, err := todoRepo.GetAll(ctx, nil, query, true)

				require.NoError(t, err)
				require.GreaterOrEqual(t, len(todos), 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx database.Transaction) {
				repo := repository.New(svc.DB())
				tt.run(t, tx, repo)
			})
		})
	}
}

func TestTodoRepository_Count(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := pkgTesting.Services()

	tests := []struct {
		name string
		run  func(t *testing.T, tx database.Transaction, repo *repository.Repository)
	}{
		{
			name: "NoFilters",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "A", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "B", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "C", Status: "pending", Priority: 1,
				})

				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10}

				count, err := todoRepo.Count(ctx, &userIDStr, query, false)

				require.NoError(t, err)
				assert.Equal(t, 3, count)
			},
		},
		{
			name: "FilterByUserID",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userA, userB := "user-a", "user-b"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userA, &dto.CreateTodoRequest{
					Title: "A's 1", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userA, &dto.CreateTodoRequest{
					Title: "A's 2", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userB, &dto.CreateTodoRequest{
					Title: "B's", Status: "pending", Priority: 1,
				})

				query := &dto.GetTodosQuery{Offset: 0, Limit: 10}

				count, err := todoRepo.Count(ctx, &userA, query, false)

				require.NoError(t, err)
				assert.Equal(t, 2, count)
			},
		},
		{
			name: "FilterByStatus",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Pending", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Completed", Status: "completed", Priority: 1,
				})

				status := "completed"
				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10, Status: &status}

				count, err := todoRepo.Count(ctx, &userIDStr, query, false)

				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "FilterByPriority",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "High", Status: "pending", Priority: 5,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Low", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Medium", Status: "pending", Priority: 3,
				})

				priority := 3
				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10, Priority: &priority}

				count, err := todoRepo.Count(ctx, &userIDStr, query, false)

				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "FilterByCategory",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				cat1 := createTestCategory(t, ctx, categoryRepo, userID, &dto.CreateCategoryRequest{
					Name: "Count-Cat1",
				})
				cat2 := createTestCategory(t, ctx, categoryRepo, userID, &dto.CreateCategoryRequest{
					Name: "Count-Cat2",
				})

				_, err := todoRepo.Create(ctx, &dto.CreateTodoRequest{
					Title: "In Cat1", Status: "pending", Priority: 1,
					UserID: userID, CategoryID: &cat1.ID,
				})
				require.NoError(t, err)
				_, err = todoRepo.Create(ctx, &dto.CreateTodoRequest{
					Title: "Also Cat1", Status: "pending", Priority: 1,
					UserID: userID, CategoryID: &cat1.ID,
				})
				require.NoError(t, err)
				_, err = todoRepo.Create(ctx, &dto.CreateTodoRequest{
					Title: "In Cat2", Status: "pending", Priority: 1,
					UserID: userID, CategoryID: &cat2.ID,
				})
				require.NoError(t, err)

				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10, CategoryID: &cat1.ID}

				count, err := todoRepo.Count(ctx, &userIDStr, query, false)

				require.NoError(t, err)
				assert.Equal(t, 2, count)
			},
		},
		{
			name: "WithSearch",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Shopping List", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Work Tasks", Status: "pending", Priority: 1,
				})

				search := "Shopping"
				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10, Search: &search}

				count, err := todoRepo.Count(ctx, &userIDStr, query, false)

				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "ExcludeDeleted",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Active", Status: "pending", Priority: 1,
				})
				deleted := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Deleted", Status: "pending", Priority: 1,
				})
				require.NoError(t, todoRepo.Delete(ctx, deleted.ID, userID))

				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10}

				count, err := todoRepo.Count(ctx, &userIDStr, query, false)

				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "IncludeDeleted",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Active", Status: "pending", Priority: 1,
				})
				deleted := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Deleted", Status: "pending", Priority: 1,
				})
				require.NoError(t, todoRepo.Delete(ctx, deleted.ID, userID))

				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10}

				count, err := todoRepo.Count(ctx, &userIDStr, query, true)

				require.NoError(t, err)
				assert.Equal(t, 2, count)
			},
		},
		{
			name: "NilUserID",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, "user-a", &dto.CreateTodoRequest{
					Title: "UserA", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, "user-b", &dto.CreateTodoRequest{
					Title: "UserB", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, "user-c", &dto.CreateTodoRequest{
					Title: "UserC", Status: "pending", Priority: 1,
				})

				query := &dto.GetTodosQuery{Offset: 0, Limit: 10}

				count, err := todoRepo.Count(ctx, nil, query, false)

				require.NoError(t, err)
				require.GreaterOrEqual(t, count, 3)
			},
		},
		{
			name: "NoResults",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				todoRepo := getTodoRepository(t, repo, tx)
				nonExistentUser := "non-existent-user"
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10}

				count, err := todoRepo.Count(ctx, &nonExistentUser, query, false)

				require.NoError(t, err)
				assert.Equal(t, 0, count)
			},
		},
		{
			name: "SearchNoMatch",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Shopping", Status: "pending", Priority: 1,
				})
				createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Work", Status: "pending", Priority: 1,
				})

				search := "nonexistent"
				userIDStr := userID
				query := &dto.GetTodosQuery{Offset: 0, Limit: 10, Search: &search}

				count, err := todoRepo.Count(ctx, &userIDStr, query, false)

				require.NoError(t, err)
				assert.Equal(t, 0, count)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx database.Transaction) {
				repo := repository.New(svc.DB())
				tt.run(t, tx, repo)
			})
		})
	}
}

func TestTodoRepository_Update(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := pkgTesting.Services()

	tests := []struct {
		name string
		run  func(t *testing.T, tx database.Transaction, repo *repository.Repository)
	}{
		{
			name: "Title",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Old Title", Status: "pending", Priority: 1,
				})

				newTitle := "Updated Title"
				updated, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID: userID,
					Title:  &newTitle,
				})

				require.NoError(t, err)
				require.NotNil(t, updated)
				assert.Equal(t, "Updated Title", updated.Title)
				assert.Equal(t, created.ID, updated.ID)
			},
		},
		{
			name: "Status",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Status Test", Status: "pending", Priority: 1,
				})

				newStatus := "completed"
				updated, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID: userID,
					Status: &newStatus,
				})

				require.NoError(t, err)
				require.NotNil(t, updated)
				assert.Equal(t, "completed", updated.Status)
			},
		},
		{
			name: "Priority",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Priority Test", Status: "pending", Priority: 1,
				})

				newPriority := 5
				updated, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID:   userID,
					Priority: &newPriority,
				})

				require.NoError(t, err)
				require.NotNil(t, updated)
				assert.Equal(t, 5, updated.Priority)
			},
		},
		{
			name: "Description",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Desc Test", Status: "pending", Priority: 1,
				})

				newDesc := "Updated Description"
				updated, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID:      userID,
					Description: &newDesc,
				})

				require.NoError(t, err)
				require.NotNil(t, updated)
				require.NotNil(t, updated.Description)
				assert.Equal(t, "Updated Description", *updated.Description)
			},
		},
		{
			name: "CategoryID",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Category Change", Status: "pending", Priority: 1,
				})

				newCat := createTestCategory(t, ctx, categoryRepo, userID, &dto.CreateCategoryRequest{
					Name: "New-Category",
				})

				updated, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID:     userID,
					CategoryID: &newCat.ID,
				})

				require.NoError(t, err)
				require.NotNil(t, updated)
				require.NotNil(t, updated.CategoryID)
				assert.Equal(t, newCat.ID, *updated.CategoryID)
			},
		},
		{
			name: "DueDate",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "DueDate Test", Status: "pending", Priority: 1,
				})
				require.Nil(t, created.DueDate)

				dueDate := time.Now().Add(24 * time.Hour).Truncate(time.Microsecond)
				updated, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID:  userID,
					DueDate: schema.Nullable[*time.Time]{IsExplicitlySet: true, Data: &dueDate},
				})

				require.NoError(t, err)
				require.NotNil(t, updated)
				require.NotNil(t, updated.DueDate)
				assert.WithinDuration(t, dueDate, *updated.DueDate, time.Millisecond)
			},
		},
		{
			name: "CompletedAt",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Complete Test", Status: "pending", Priority: 1,
				})
				require.Nil(t, created.CompletedAt)

				now := time.Now().Truncate(time.Microsecond)
				updated, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID:      userID,
					CompletedAt: schema.Nullable[*time.Time]{IsExplicitlySet: true, Data: &now},
				})

				require.NoError(t, err)
				require.NotNil(t, updated)
				require.NotNil(t, updated.CompletedAt)
				assert.WithinDuration(t, now, *updated.CompletedAt, time.Millisecond)
			},
		},
		{
			name: "ParentID",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				parent := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Parent", Status: "pending", Priority: 1,
				})
				child := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Child", Status: "pending", Priority: 1,
				})
				require.Nil(t, child.ParentID)

				updated, err := todoRepo.Update(ctx, child.ID, userID, &dto.UpdateTodoRequest{
					UserID:   userID,
					ParentID: schema.Nullable[*uuid.UUID]{IsExplicitlySet: true, Data: &parent.ID},
				})

				require.NoError(t, err)
				require.NotNil(t, updated)
				require.NotNil(t, updated.ParentID)
				assert.Equal(t, parent.ID, *updated.ParentID)
			},
		},
		{
			name: "MetadataMerge",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title:    "Meta Test",
					Status:   "pending",
					Priority: 1,
					Metadata: map[string]any{"original": "value1"},
				})
				require.NotNil(t, created.Metadata)

				meta := map[string]any{"additional": "value2"}
				updated, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID:   userID,
					Metadata: &meta,
				})

				require.NoError(t, err)
				require.NotNil(t, updated.Metadata)
				assert.Equal(t, "value1", updated.Metadata["original"])
				assert.Equal(t, "value2", updated.Metadata["additional"])
			},
		},
		{
			name: "AllFields",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Original", Status: "pending", Priority: 1,
				})

				parent := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "New Parent", Status: "pending", Priority: 1,
				})
				newCat := createTestCategory(t, ctx, categoryRepo, userID, &dto.CreateCategoryRequest{
					Name: "New-Cat",
				})
				newTitle := "All Updated"
				newDesc := "New Description"
				newStatus := "completed"
				newPriority := 5
				dueDate := time.Now().Add(24 * time.Hour).Truncate(time.Microsecond)
				now := time.Now().Truncate(time.Microsecond)
				meta := map[string]any{"key": "val"}

				updated, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID:      userID,
					Title:       &newTitle,
					Description: &newDesc,
					Status:      &newStatus,
					Priority:    &newPriority,
					CategoryID:  &newCat.ID,
					DueDate:     schema.Nullable[*time.Time]{IsExplicitlySet: true, Data: &dueDate},
					CompletedAt: schema.Nullable[*time.Time]{IsExplicitlySet: true, Data: &now},
					ParentID:    schema.Nullable[*uuid.UUID]{IsExplicitlySet: true, Data: &parent.ID},
					Metadata:    &meta,
				})

				require.NoError(t, err)
				require.NotNil(t, updated)
				assert.Equal(t, "All Updated", updated.Title)
				require.NotNil(t, updated.Description)
				assert.Equal(t, "New Description", *updated.Description)
				assert.Equal(t, "completed", updated.Status)
				assert.Equal(t, 5, updated.Priority)
				require.NotNil(t, updated.CategoryID)
				assert.Equal(t, newCat.ID, *updated.CategoryID)
				require.NotNil(t, updated.DueDate)
				assert.WithinDuration(t, dueDate, *updated.DueDate, time.Millisecond)
				require.NotNil(t, updated.CompletedAt)
				assert.WithinDuration(t, now, *updated.CompletedAt, time.Millisecond)
				require.NotNil(t, updated.ParentID)
				assert.Equal(t, parent.ID, *updated.ParentID)
				require.NotNil(t, updated.Metadata)
				assert.Equal(t, "val", updated.Metadata["key"])
			},
		},
		{
			name: "NoChanges",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "No Change", Status: "pending", Priority: 1,
				})

				updated, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID: userID,
				})

				require.NoError(t, err)
				require.NotNil(t, updated)
				assert.Equal(t, created.ID, updated.ID)
				assert.Equal(t, "No Change", updated.Title)
			},
		},
		{
			name: "NotFound",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				todoRepo := getTodoRepository(t, repo, tx)
				newTitle := "Should Fail"

				_, err := todoRepo.Update(ctx, uuid.New(), "non-existent-user", &dto.UpdateTodoRequest{
					UserID: "non-existent-user",
					Title:  &newTitle,
				})

				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
		{
			name: "SoftDeleted",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "To Update Deleted", Status: "pending", Priority: 1,
				})

				require.NoError(t, todoRepo.Delete(ctx, created.ID, userID))

				newTitle := "New Name"
				_, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID: userID,
					Title:  &newTitle,
				})

				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
		{
			name: "SetDueDateToNull",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				dueDate := time.Now().Add(24 * time.Hour).Truncate(time.Microsecond)
				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Due Date Clear", Status: "pending", Priority: 1,
					DueDate: &dueDate,
				})
				require.NotNil(t, created.DueDate)

				updated, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID:  userID,
					DueDate: schema.Nullable[*time.Time]{IsExplicitlySet: true, Data: nil},
				})

				require.NoError(t, err)
				require.Nil(t, updated.DueDate)
			},
		},
		{
			name: "SetCompletedAtToNull",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Completed At Clear", Status: "pending", Priority: 1,
				})
				require.Nil(t, created.CompletedAt)

				now := time.Now().Truncate(time.Microsecond)
				withCompleted, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID:      userID,
					CompletedAt: schema.Nullable[*time.Time]{IsExplicitlySet: true, Data: &now},
				})
				require.NoError(t, err)
				require.NotNil(t, withCompleted.CompletedAt)

				cleared, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID:      userID,
					CompletedAt: schema.Nullable[*time.Time]{IsExplicitlySet: true, Data: nil},
				})

				require.NoError(t, err)
				require.Nil(t, cleared.CompletedAt)
			},
		},
		{
			name: "SetParentIDToNull",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				parent := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Parent", Status: "pending", Priority: 1,
				})
				child := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Child With Parent", Status: "pending", Priority: 1,
				})

				withParent, err := todoRepo.Update(ctx, child.ID, userID, &dto.UpdateTodoRequest{
					UserID:   userID,
					ParentID: schema.Nullable[*uuid.UUID]{IsExplicitlySet: true, Data: &parent.ID},
				})
				require.NoError(t, err)
				require.NotNil(t, withParent.ParentID)

				cleared, err := todoRepo.Update(ctx, child.ID, userID, &dto.UpdateTodoRequest{
					UserID:   userID,
					ParentID: schema.Nullable[*uuid.UUID]{IsExplicitlySet: true, Data: nil},
				})

				require.NoError(t, err)
				require.Nil(t, cleared.ParentID)
			},
		},
		{
			name: "ClearAllNullableToNull",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				parent := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Parent", Status: "pending", Priority: 1,
				})
				dueDate := time.Now().Add(24 * time.Hour).Truncate(time.Microsecond)

				child := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Child With All", Status: "pending", Priority: 1,
				})

				withAll, err := todoRepo.Update(ctx, child.ID, userID, &dto.UpdateTodoRequest{
					UserID:      userID,
					DueDate:     schema.Nullable[*time.Time]{IsExplicitlySet: true, Data: &dueDate},
					CompletedAt: schema.Nullable[*time.Time]{IsExplicitlySet: true, Data: &dueDate},
					ParentID:    schema.Nullable[*uuid.UUID]{IsExplicitlySet: true, Data: &parent.ID},
				})
				require.NoError(t, err)
				require.NotNil(t, withAll.DueDate)
				require.NotNil(t, withAll.CompletedAt)
				require.NotNil(t, withAll.ParentID)

				cleared, err := todoRepo.Update(ctx, child.ID, userID, &dto.UpdateTodoRequest{
					UserID:      userID,
					DueDate:     schema.Nullable[*time.Time]{IsExplicitlySet: true, Data: nil},
					CompletedAt: schema.Nullable[*time.Time]{IsExplicitlySet: true, Data: nil},
					ParentID:    schema.Nullable[*uuid.UUID]{IsExplicitlySet: true, Data: nil},
				})

				require.NoError(t, err)
				require.Nil(t, cleared.DueDate)
				require.Nil(t, cleared.CompletedAt)
				require.Nil(t, cleared.ParentID)
			},
		},
		{
			name: "SelfParentError",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Self Parent", Status: "pending", Priority: 1,
				})

				_, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID:   userID,
					ParentID: schema.Nullable[*uuid.UUID]{IsExplicitlySet: true, Data: &created.ID},
				})

				require.Error(t, err)
				var appErr *errors.AppError
				require.ErrorAs(t, err, &appErr)
				assert.Equal(t, errors.Validation, appErr.Type)
			},
		},
		{
			name: "InvalidParentError",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Bad Parent", Status: "pending", Priority: 1,
				})
				invalidID := uuid.New()

				_, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID:   userID,
					ParentID: schema.Nullable[*uuid.UUID]{IsExplicitlySet: true, Data: &invalidID},
				})

				require.Error(t, err)
				var appErr *errors.AppError
				require.ErrorAs(t, err, &appErr)
				assert.Equal(t, errors.Validation, appErr.Type)
			},
		},
		{
			name: "InvalidCategoryError",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Bad Category", Status: "pending", Priority: 1,
				})
				invalidCatID := uuid.New()

				_, err := todoRepo.Update(ctx, created.ID, userID, &dto.UpdateTodoRequest{
					UserID:     userID,
					CategoryID: &invalidCatID,
				})

				require.Error(t, err)
				var appErr *errors.AppError
				require.ErrorAs(t, err, &appErr)
				assert.Equal(t, errors.Validation, appErr.Type)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx database.Transaction) {
				repo := repository.New(svc.DB())
				tt.run(t, tx, repo)
			})
		})
	}
}

func TestTodoRepository_Delete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := pkgTesting.Services()

	tests := []struct {
		name string
		run  func(t *testing.T, tx database.Transaction, repo *repository.Repository)
	}{
		{
			name: "SoftDelete",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Soft", Status: "pending", Priority: 1,
				})

				require.NoError(t, todoRepo.Delete(ctx, created.ID, userID))

				_, err := todoRepo.GetByID(ctx, created.ID, userID, false)
				assertAppErrorType(t, err, errors.ResourceNotFound)

				fetched, err := todoRepo.GetByID(ctx, created.ID, userID, true)
				require.NoError(t, err)
				assert.True(t, fetched.IsDeleted)
			},
		},
		{
			name: "NonExistent",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				todoRepo := getTodoRepository(t, repo, tx)

				err := todoRepo.Delete(ctx, uuid.New(), "non-existent-user")
				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
		{
			name: "WrongUser",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				ownerID := "owner-id"
				otherID := "other-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, ownerID, &dto.CreateTodoRequest{
					Title: "Wrong User Delete", Status: "pending", Priority: 1,
				})

				err := todoRepo.Delete(ctx, created.ID, otherID)
				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
		{
			name: "DoubleSoftDelete",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
				todoRepo := getTodoRepository(t, repo, tx)
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestTodo(t, ctx, todoRepo, categoryRepo, userID, &dto.CreateTodoRequest{
					Title: "Double Soft", Status: "pending", Priority: 1,
				})

				require.NoError(t, todoRepo.Delete(ctx, created.ID, userID))

				err := todoRepo.Delete(ctx, created.ID, userID)
				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx database.Transaction) {
				repo := repository.New(svc.DB())
				tt.run(t, tx, repo)
			})
		})
	}
}
