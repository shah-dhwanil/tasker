package repository_test

import (
	"context"
	"testing"

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

func getCategoryRepository(t *testing.T, repository *repository.Repository, tx database.Transaction) repository.Category {
	t.Helper()
	return repository.CategoryRepository.WithExecutor(tx)
}

func createTestCategory(t *testing.T, ctx context.Context, repo repository.Category, userID string, payload *dto.CreateCategoryRequest) *dto.Category {
	t.Helper()
	resp, err := repo.CreateCategory(ctx, userID, payload)
	require.NoError(t, err)
	require.NotNil(t, resp)
	return resp
}

func TestCategoryRepository_CreateCategory(t *testing.T) {
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
				desc := "test description"
				meta := map[string]any{"env": "test"}

				categoryRepo := getCategoryRepository(t, repo, tx)

				resp, err := categoryRepo.CreateCategory(ctx, userID, &dto.CreateCategoryRequest{
					Name:        "My Category",
					Description: &desc,
					Metadata:    meta,
				})

				require.NoError(t, err)
				require.NotNil(t, resp)

				assert.NotZero(t, resp.ID)
				assert.Equal(t, "My Category", resp.Name)

				require.NotNil(t, resp.Description)
				assert.Equal(t, "test description", *resp.Description)

				assert.Equal(t, "test", resp.Metadata["env"])
				assert.False(t, resp.CreatedAt.IsZero())
				assert.False(t, resp.UpdatedAt.IsZero())
			},
		},
		{
			name: "RequiredOnly",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				resp, err := categoryRepo.CreateCategory(
					ctx,
					"test-user-id",
					&dto.CreateCategoryRequest{Name: "Minimal"},
				)

				require.NoError(t, err)
				require.NotNil(t, resp)

				assert.Equal(t, "Minimal", resp.Name)
				assert.Nil(t, resp.Description)
				assert.Nil(t, resp.Metadata)
			},
		},
		{
			name: "DuplicateName",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"

				categoryRepo := getCategoryRepository(t, repo, tx)

				_, err := categoryRepo.CreateCategory(
					ctx,
					userID,
					&dto.CreateCategoryRequest{Name: "Dup"},
				)
				require.NoError(t, err)

				_, err = categoryRepo.CreateCategory(
					ctx,
					userID,
					&dto.CreateCategoryRequest{Name: "Dup"},
				)

				assertAppErrorType(t, err, errors.ResourceAlreadyExists)
			},
		},
		{
			name: "SameNameDifferentUser",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				_, err := categoryRepo.CreateCategory(
					ctx,
					"user-a",
					&dto.CreateCategoryRequest{Name: "Common"},
				)
				require.NoError(t, err)

				_, err = categoryRepo.CreateCategory(
					ctx,
					"user-b",
					&dto.CreateCategoryRequest{Name: "Common"},
				)
				require.NoError(t, err)
			},
		},
		{
			name: "DuplicateNameAfterSoftDelete",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"

				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{Name: "Reuse"},
				)

				require.NoError(t, categoryRepo.DeleteCategory(ctx, created.ID, nil))

				_, err := categoryRepo.CreateCategory(
					ctx,
					userID,
					&dto.CreateCategoryRequest{Name: "Reuse"},
				)

				assertAppErrorType(t, err, errors.ResourceAlreadyExists)
			},
		},
		{
			name: "WithoutDescription",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				resp, err := categoryRepo.CreateCategory(
					ctx,
					"test-user-id",
					&dto.CreateCategoryRequest{Name: "NoDesc"},
				)

				require.NoError(t, err)
				require.NotNil(t, resp)

				assert.Nil(t, resp.Description)
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

func TestCategoryRepository_GetCategoryByID(t *testing.T) {
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
	
				categoryRepo := getCategoryRepository(t, repo, tx)
	
				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{
						Name: "Fetch Me",
					},
				)
	
				fetched, err := categoryRepo.GetCategoryByID(
					ctx,
					created.ID,
					false,
				)
	
				require.NoError(t, err)
				require.NotNil(t, fetched)
	
				assert.Equal(t, created.ID, fetched.ID)
				assert.Equal(t, "Fetch Me", fetched.Name)
				assert.Equal(t, userID, fetched.UserID)
			},
		},
		{
			name: "NotFound",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)
	
				_, err := categoryRepo.GetCategoryByID(
					ctx,
					uuid.New(),
					false,
				)
	
				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
		{
			name: "DeletedNotIncluded",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
	
				categoryRepo := getCategoryRepository(t, repo, tx)
	
				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{
						Name: "To Delete",
					},
				)
	
				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, created.ID, nil),
				)
	
				_, err := categoryRepo.GetCategoryByID(
					ctx,
					created.ID,
					false,
				)
	
				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
		{
			name: "DeletedIncluded",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"
	
				categoryRepo := getCategoryRepository(t, repo, tx)
	
				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{
						Name: "To Fetch Deleted",
					},
				)
	
				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, created.ID, nil),
				)
	
				fetched, err := categoryRepo.GetCategoryByID(
					ctx,
					created.ID,
					true,
				)
	
				require.NoError(t, err)
				require.NotNil(t, fetched)
	
				assert.Equal(t, created.ID, fetched.ID)
			},
		},
		{
			name: "HardDeletedNotFound",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)
	
				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id",
					&dto.CreateCategoryRequest{
						Name: "Hard Delete",
					},
				)
	
				hard := true
	
				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, created.ID, &hard),
				)
	
				_, err := categoryRepo.GetCategoryByID(
					ctx,
					created.ID,
					true,
				)
	
				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
	}
	
	for _, tt := range tests {
	
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
	
			pkgTesting.WithRollbackTransaction(
				ctx,
				svc.DB(),
				func(tx database.Transaction) {
					repo := repository.New(svc.DB())
					tt.run(t, tx, repo)
				},
			)
		})
	}
}

func TestCategoryRepository_GetAllCategories(t *testing.T) {
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
				userID := "test-user-id1"

				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(t, ctx, categoryRepo, userID,
					&dto.CreateCategoryRequest{Name: "A"},
				)
				createTestCategory(t, ctx, categoryRepo, userID,
					&dto.CreateCategoryRequest{Name: "B"},
				)

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10}

				cats, err := categoryRepo.GetAllCategories(
					ctx,
					&userID,
					query,
					true,
				)

				require.NoError(t, err)
				assert.Len(t, cats, 2)
			},
		},
		{
			name: "FilterByUserID",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userA, userB := "user-a", "user-b"

				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(t, ctx, categoryRepo, userA,
					&dto.CreateCategoryRequest{Name: "A's"},
				)

				createTestCategory(t, ctx, categoryRepo, userB,
					&dto.CreateCategoryRequest{Name: "B's"},
				)

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10}

				cats, err := categoryRepo.GetAllCategories(
					ctx,
					&userA,
					query,
					true,
				)

				require.NoError(t, err)
				require.Len(t, cats, 1)

				assert.Equal(t, "A's", cats[0].Name)
			},
		},
		{
			name: "Search",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id2"

				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(t, ctx, categoryRepo, userID,
					&dto.CreateCategoryRequest{Name: "Shopping List"},
				)

				createTestCategory(t, ctx, categoryRepo, userID,
					&dto.CreateCategoryRequest{Name: "Work Tasks"},
				)

				search := "shop"

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10, Search: &search}

				cats, err := categoryRepo.GetAllCategories(
					ctx,
					&userID,
					query,
					true,
				)

				require.NoError(t, err)
				require.Len(t, cats, 1)

				assert.Equal(t, "Shopping List", cats[0].Name)
			},
		},
		{
			name: "OrderByName",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id3"

				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(t, ctx, categoryRepo, userID,
					&dto.CreateCategoryRequest{Name: "Gamma"},
				)

				createTestCategory(t, ctx, categoryRepo, userID,
					&dto.CreateCategoryRequest{Name: "Alpha"},
				)

				createTestCategory(t, ctx, categoryRepo, userID,
					&dto.CreateCategoryRequest{Name: "Beta"},
				)

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10, OrderBy: []string{"name"}}

				cats, err := categoryRepo.GetAllCategories(
					ctx,
					&userID,
					query,
					true,
				)

				require.NoError(t, err)
				require.Len(t, cats, 3)

				assert.Equal(t, "Alpha", cats[0].Name)
				assert.Equal(t, "Beta", cats[1].Name)
				assert.Equal(t, "Gamma", cats[2].Name)
			},
		},
		{
			name: "ExcludeSoftDeleted",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id4"

				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(t, ctx, categoryRepo, userID,
					&dto.CreateCategoryRequest{Name: "Active"},
				)

				deleted := createTestCategory(t, ctx, categoryRepo, userID,
					&dto.CreateCategoryRequest{Name: "Deleted"},
				)

				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, deleted.ID, nil),
				)

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10}

				cats, err := categoryRepo.GetAllCategories(
					ctx,
					&userID,
					query,
					false,
				)

				require.NoError(t, err)
				require.Len(t, cats, 1)

				assert.Equal(t, "Active", cats[0].Name)
			},
		},
		{
			name: "NoResults",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10}

				nonExistentUser := "non-existent-user"

				cats, err := categoryRepo.GetAllCategories(
					ctx,
					&nonExistentUser,
					query,
					true,
				)

				require.NoError(t, err)
				assert.Empty(t, cats)
			},
		},
		{
			name: "CustomPagination",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id5"

				categoryRepo := getCategoryRepository(t, repo, tx)

				for _, name := range []string{
					"Alpha",
					"Beta",
					"Gamma",
					"Delta",
					"Epsilon",
				} {
					createTestCategory(
						t,
						ctx,
						categoryRepo,
						userID,
						&dto.CreateCategoryRequest{Name: name},
					)
				}

				query := &dto.GetCategoriesQuery{Offset: 2, Limit: 2, OrderBy: []string{"name"}}

				cats, err := categoryRepo.GetAllCategories(
					ctx,
					&userID,
					query,
					true,
				)

				require.NoError(t, err)
				require.Len(t, cats, 2)

				assert.Equal(t, "Delta", cats[0].Name)
				assert.Equal(t, "Epsilon", cats[1].Name)
			},
		},
		{
			name: "OrderByDescendingName",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id6"

				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(t, ctx, categoryRepo, userID,
					&dto.CreateCategoryRequest{Name: "Alpha"},
				)

				createTestCategory(t, ctx, categoryRepo, userID,
					&dto.CreateCategoryRequest{Name: "Beta"},
				)

				createTestCategory(t, ctx, categoryRepo, userID,
					&dto.CreateCategoryRequest{Name: "Gamma"},
				)

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10, OrderBy: []string{"-name"}}

				cats, err := categoryRepo.GetAllCategories(
					ctx,
					&userID,
					query,
					true,
				)

				require.NoError(t, err)
				require.Len(t, cats, 3)

				assert.Equal(t, "Gamma", cats[0].Name)
				assert.Equal(t, "Beta", cats[1].Name)
				assert.Equal(t, "Alpha", cats[2].Name)
			},
		},
		{
			name: "IncludeDeletedRecords",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id7"

				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(t, ctx, categoryRepo, userID,
					&dto.CreateCategoryRequest{Name: "Active"},
				)

				deleted := createTestCategory(t, ctx, categoryRepo, userID,
					&dto.CreateCategoryRequest{Name: "Deleted"},
				)

				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, deleted.ID, nil),
				)

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10}

				cats, err := categoryRepo.GetAllCategories(
					ctx,
					&userID,
					query,
					true,
				)

				require.NoError(t, err)
				require.Len(t, cats, 2)
			},
		},
		{
			name: "NilUserID",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(t, ctx, categoryRepo, "user-a",
					&dto.CreateCategoryRequest{Name: "UserA"},
				)

				createTestCategory(t, ctx, categoryRepo, "user-b",
					&dto.CreateCategoryRequest{Name: "UserB"},
				)

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10}

				cats, err := categoryRepo.GetAllCategories(
					ctx,
					nil,
					query,
					true,
				)

				require.NoError(t, err)
				require.GreaterOrEqual(t, len(cats), 2)
			},
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pkgTesting.WithRollbackTransaction(
				ctx,
				svc.DB(),
				func(tx database.Transaction) {
					repo := repository.New(svc.DB())
					tt.run(t, tx, repo)
				},
			)
		})
	}
}

func TestCategoryRepository_UpdateCategory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := pkgTesting.Services()

	tests := []struct {
		name string
		run  func(t *testing.T, tx database.Transaction, repo *repository.Repository)
	}{
		{
			name: "Name",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id",
					&dto.CreateCategoryRequest{Name: "Old"},
				)

				newName := "New"

				updated, err := categoryRepo.UpdateCategory(
					ctx,
					created.ID,
					&dto.UpdateCategoryRequest{
						Name: &newName,
					},
					false,
				)

				require.NoError(t, err)
				require.NotNil(t, updated)

				assert.Equal(t, "New", updated.Name)
				assert.Equal(t, created.ID, updated.ID)
			},
		},
		{
			name: "Description",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id",
					&dto.CreateCategoryRequest{Name: "Desc"},
				)

				newDesc := "Updated"

				updated, err := categoryRepo.UpdateCategory(
					ctx,
					created.ID,
					&dto.UpdateCategoryRequest{
						Description: schema.Nullable[*string]{Data: &newDesc, IsExplicitlySet: true},
					},
					false,
				)

				require.NoError(t, err)
				require.NotNil(t, updated)
				require.NotNil(t, updated.Description)

				assert.Equal(t, "Updated", *updated.Description)
			},
		},
		{
			name: "MetadataMerge",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id",
					&dto.CreateCategoryRequest{Name: "Meta"},
				)

				meta := map[string]any{"key": "val"}

				updated, err := categoryRepo.UpdateCategory(
					ctx,
					created.ID,
					&dto.UpdateCategoryRequest{
						Metadata: &meta,
					},
					false,
				)

				require.NoError(t, err)
				require.NotNil(t, updated)

				assert.Equal(t, "val", updated.Metadata["key"])
			},
		},
		{
			name: "AllFields",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id",
					&dto.CreateCategoryRequest{Name: "Orig"},
				)

				n := "Updated"
				d := "New desc"
				m := map[string]any{"p": "h"}

				updated, err := categoryRepo.UpdateCategory(
					ctx,
					created.ID,
					&dto.UpdateCategoryRequest{
						Name:        &n,
						Description: schema.Nullable[*string]{Data: &d, IsExplicitlySet: true},
						Metadata:    &m,
					},
					false,
				)

				require.NoError(t, err)
				require.NotNil(t, updated)

				assert.Equal(t, "Updated", updated.Name)
				assert.Equal(t, "New desc", *updated.Description)
				assert.Equal(t, "h", updated.Metadata["p"])
			},
		},
		{
			name: "NoChanges",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id",
					&dto.CreateCategoryRequest{Name: "No Change"},
				)

				updated, err := categoryRepo.UpdateCategory(
					ctx,
					created.ID,
					&dto.UpdateCategoryRequest{},
					false,
				)

				require.NoError(t, err)
				require.NotNil(t, updated)

				assert.Equal(t, created.ID, updated.ID)
				assert.Equal(t, "No Change", updated.Name)
			},
		},
		{
			name: "NotFound",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				n := "Should Fail"

				_, err := categoryRepo.UpdateCategory(
					ctx,
					uuid.New(),
					&dto.UpdateCategoryRequest{
						Name: &n,
					},
					false,
				)

				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
		{
			name: "SoftDeletedNotConsidered",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id",
					&dto.CreateCategoryRequest{
						Name: "To Update Deleted",
					},
				)

				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, created.ID, nil),
				)

				n := "New Name"

				_, err := categoryRepo.UpdateCategory(
					ctx,
					created.ID,
					&dto.UpdateCategoryRequest{
						Name: &n,
					},
					false,
				)

				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
		{
			name: "DuplicateName",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id"

				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{
						Name: "Existing",
					},
				)

				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{
						Name: "To Rename",
					},
				)

				n := "Existing"

				_, err := categoryRepo.UpdateCategory(
					ctx,
					created.ID,
					&dto.UpdateCategoryRequest{
						Name: &n,
					},
					false,
				)

				assertAppErrorType(t, err, errors.ResourceAlreadyExists)
			},
		},
		{
			name: "ConsiderDeletedRecords",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id",
					&dto.CreateCategoryRequest{
						Name: "Soft Deleted Update",
					},
				)

				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, created.ID, nil),
				)

				n := "Updated After Deletion"

				updated, err := categoryRepo.UpdateCategory(
					ctx,
					created.ID,
					&dto.UpdateCategoryRequest{
						Name: &n,
					},
					true,
				)

				require.NoError(t, err)
				require.NotNil(t, updated)

				assert.Equal(t, "Updated After Deletion", updated.Name)
			},
		},
		{
			name: "EmptyDescription",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				desc := "initial"

				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id",
					&dto.CreateCategoryRequest{
						Name: "Clear Desc",
					},
				)

				_, err := categoryRepo.UpdateCategory(
					ctx,
					created.ID,
					&dto.UpdateCategoryRequest{
						Description: schema.Nullable[*string]{Data: &desc, IsExplicitlySet: true},
					},
					false,
				)

				require.NoError(t, err)

				empty := ""

				updated, err := categoryRepo.UpdateCategory(
					ctx,
					created.ID,
					&dto.UpdateCategoryRequest{
						Description: schema.Nullable[*string]{Data: &empty, IsExplicitlySet: true},
					},
					false,
				)

				require.NoError(t, err)
				require.NotNil(t, updated.Description)

				assert.Equal(t, "", *updated.Description)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pkgTesting.WithRollbackTransaction(
				ctx,
				svc.DB(),
				func(tx database.Transaction) {
					repo := repository.New(svc.DB())
					tt.run(t, tx, repo)
				},
			)
		})
	}
}

func TestCategoryRepository_DeleteCategory(t *testing.T) {
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
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id",
					&dto.CreateCategoryRequest{
						Name: "Soft",
					},
				)

				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, created.ID, nil),
				)

				_, err := categoryRepo.GetCategoryByID(
					ctx,
					created.ID,
					false,
				)

				assertAppErrorType(t, err, errors.ResourceNotFound)

				_, err = categoryRepo.GetCategoryByID(
					ctx,
					created.ID,
					true,
				)

				require.NoError(t, err)
			},
		},
		{
			name: "HardDelete",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id",
					&dto.CreateCategoryRequest{
						Name: "Hard",
					},
				)

				hard := true

				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, created.ID, &hard),
				)

				_, err := categoryRepo.GetCategoryByID(
					ctx,
					created.ID,
					true,
				)

				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
		{
			name: "NonExistent",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, uuid.New(), nil),
				)
			},
		},
		{
			name: "DefaultIsSoftDelete",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id",
					&dto.CreateCategoryRequest{
						Name: "Default",
					},
				)

				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, created.ID, nil),
				)

				_, err := categoryRepo.GetCategoryByID(
					ctx,
					created.ID,
					false,
				)

				assertAppErrorType(t, err, errors.ResourceNotFound)
			},
		},
		{
			name: "DoubleSoftDelete",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				created := createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id",
					&dto.CreateCategoryRequest{
						Name: "Double Soft",
					},
				)

				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, created.ID, nil),
				)

				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, created.ID, nil),
				)
			},
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pkgTesting.WithRollbackTransaction(
				ctx,
				svc.DB(),
				func(tx database.Transaction) {
					repo := repository.New(svc.DB())
					tt.run(t, tx, repo)
				},
			)
		})
	}
}

func TestCategoryRepository_CountCategories(t *testing.T) {
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
				userID := "test-user-id1"

				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{Name: "A"},
				)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{Name: "B"},
				)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{Name: "C"},
				)

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10}

				count, err := categoryRepo.CountCategories(
					ctx,
					&userID,
					query,
					false,
				)

				require.NoError(t, err)
				assert.Equal(t, 3, count)
			},
		},
		{
			name: "FilterByUserID",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userA, userB := "user-a", "user-b"

				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					userA,
					&dto.CreateCategoryRequest{Name: "A's"},
				)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					userA,
					&dto.CreateCategoryRequest{Name: "A's 2"},
				)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					userB,
					&dto.CreateCategoryRequest{Name: "B's"},
				)

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10}

				count, err := categoryRepo.CountCategories(
					ctx,
					&userA,
					query,
					false,
				)

				require.NoError(t, err)
				assert.Equal(t, 2, count)
			},
		},
		{
			name: "WithSearch",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id2"

				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{
						Name: "Shopping List",
					},
				)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{
						Name: "Work Tasks",
					},
				)

				search := "shop"

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10, Search: &search}

				count, err := categoryRepo.CountCategories(
					ctx,
					&userID,
					query,
					false,
				)

				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "ExcludeSoftDeleted",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id3"

				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{
						Name: "Active",
					},
				)

				deleted := createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{
						Name: "Deleted",
					},
				)

				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, deleted.ID, nil),
				)

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10}

				count, err := categoryRepo.CountCategories(
					ctx,
					&userID,
					query,
					false,
				)

				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "IncludeDeleted",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id4"

				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{
						Name: "Active",
					},
				)

				deleted := createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{
						Name: "Deleted",
					},
				)

				require.NoError(
					t,
					categoryRepo.DeleteCategory(ctx, deleted.ID, nil),
				)

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10}

				count, err := categoryRepo.CountCategories(
					ctx,
					&userID,
					query,
					true,
				)

				require.NoError(t, err)
				assert.Equal(t, 2, count)
			},
		},
		{
			name: "NilUserID",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id5",
					&dto.CreateCategoryRequest{
						Name: "UserA",
					},
				)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id5",
					&dto.CreateCategoryRequest{
						Name: "UserB",
					},
				)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					"test-user-id5",
					&dto.CreateCategoryRequest{
						Name: "UserC",
					},
				)

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10}

				count, err := categoryRepo.CountCategories(
					ctx,
					nil,
					query,
					false,
				)

				require.NoError(t, err)
				require.GreaterOrEqual(t, count, 3)
			},
		},
		{
			name: "NoResults",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				categoryRepo := getCategoryRepository(t, repo, tx)

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10}

				nonExistentUser := "non-existent-user"

				count, err := categoryRepo.CountCategories(
					ctx,
					&nonExistentUser,
					query,
					false,
				)

				require.NoError(t, err)
				assert.Equal(t, 0, count)
			},
		},
		{
			name: "SearchNoMatch",
			run: func(t *testing.T, tx database.Transaction, repo *repository.Repository) {
				userID := "test-user-id6"

				categoryRepo := getCategoryRepository(t, repo, tx)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{
						Name: "Shopping",
					},
				)

				createTestCategory(
					t,
					ctx,
					categoryRepo,
					userID,
					&dto.CreateCategoryRequest{
						Name: "Work",
					},
				)

				search := "nonexistent"

				query := &dto.GetCategoriesQuery{Offset: 0, Limit: 10, Search: &search}

				count, err := categoryRepo.CountCategories(
					ctx,
					&userID,
					query,
					false,
				)

				require.NoError(t, err)
				assert.Equal(t, 0, count)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pkgTesting.WithRollbackTransaction(
				ctx,
				svc.DB(),
				func(tx database.Transaction){
					repo := repository.New(svc.DB())
					tt.run(t, tx, repo)
				},
			)
		})
	}
}