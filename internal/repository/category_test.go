package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shah-dhwanil/tasker/internal/errors"
	"github.com/shah-dhwanil/tasker/internal/repository"
	"github.com/shah-dhwanil/tasker/internal/schema"
	pkgTesting "github.com/shah-dhwanil/tasker/internal/testing"
)

func createTestCategory(t *testing.T, ctx context.Context, tx pgx.Tx, userID uuid.UUID, name string) *schema.CreateCategoryResponse {
	t.Helper()
	repo := repository.New(pkgTesting.Services().DB()).CategoryRepository.WithExecutor(tx)
	resp, err := repo.CreateCategory(ctx, userID, &schema.CreateCategoryRequest{Name: name})
	require.NoError(t, err)
	require.NotNil(t, resp)
	return resp
}

func assertAppErrorType(t *testing.T, err error, expectedType errors.ErrorType) {
	t.Helper()
	require.Error(t, err)
	var appErr *errors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, expectedType, appErr.Type)
}

func TestCategoryRepository_CreateCategory_Success(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		desc := "test description"
		meta := map[string]any{"env": "test"}
		req := &schema.CreateCategoryRequest{
			Name: "My Category", Description: &desc, Metadata: meta,
		}
		resp, err := repo.CreateCategory(ctx, userID, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotZero(t, resp.ID)
		assert.Equal(t, "My Category", resp.Name)
		require.NotNil(t, resp.Description)
		assert.Equal(t, "test description", *resp.Description)
		assert.Equal(t, "test", resp.Metadata["env"])
		assert.False(t, resp.CreatedAt.IsZero())
		assert.False(t, resp.UpdatedAt.IsZero())
		return nil
	})
}

func TestCategoryRepository_CreateCategory_RequiredOnly(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		resp, err := repo.CreateCategory(ctx, uuid.New(), &schema.CreateCategoryRequest{Name: "Minimal"})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Minimal", resp.Name)
		assert.Nil(t, resp.Description)
		assert.Nil(t, resp.Metadata)
		return nil
	})
}

func TestCategoryRepository_CreateCategory_DuplicateName(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		_, err := repo.CreateCategory(ctx, userID, &schema.CreateCategoryRequest{Name: "Dup"})
		require.NoError(t, err)

		_, err = repo.CreateCategory(ctx, userID, &schema.CreateCategoryRequest{Name: "Dup"})
		assertAppErrorType(t, err, errors.ResourceAlreadyExists)
		return nil
	})
}

func TestCategoryRepository_CreateCategory_SameNameDifferentUser(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		_, err := repo.CreateCategory(ctx, uuid.New(), &schema.CreateCategoryRequest{Name: "Common"})
		require.NoError(t, err)

		_, err = repo.CreateCategory(ctx, uuid.New(), &schema.CreateCategoryRequest{Name: "Common"})
		require.NoError(t, err)
		return nil
	})
}

func TestCategoryRepository_CreateCategory_DuplicateNameAfterSoftDelete(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		created := createTestCategory(t, ctx, tx, userID, "Reuse")
		require.NoError(t, repo.DeleteCategory(ctx, created.ID, nil))

		_, err := repo.CreateCategory(ctx, userID, &schema.CreateCategoryRequest{Name: "Reuse"})
		assertAppErrorType(t, err, errors.ResourceAlreadyExists)
		return nil
	})
}

func TestCategoryRepository_CreateCategory_WithoutDescription(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		resp, err := repo.CreateCategory(ctx, uuid.New(), &schema.CreateCategoryRequest{Name: "NoDesc"})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Description)
		return nil
	})
}

func TestCategoryRepository_GetCategoryByID_Success(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		created := createTestCategory(t, ctx, tx, userID, "Fetch Me")

		fetched, err := repo.GetCategoryByID(ctx, created.ID, false)
		require.NoError(t, err)
		require.NotNil(t, fetched)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, "Fetch Me", fetched.Name)
		assert.Equal(t, userID, fetched.UserID)
		return nil
	})
}

func TestCategoryRepository_GetCategoryByID_NotFound(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		_, err := repo.GetCategoryByID(ctx, uuid.New(), false)
		assertAppErrorType(t, err, errors.ResourceNotFound)
		return nil
	})
}

func TestCategoryRepository_GetCategoryByID_DeletedNotIncluded(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		created := createTestCategory(t, ctx, tx, userID, "To Delete")
		require.NoError(t, repo.DeleteCategory(ctx, created.ID, nil))

		_, err := repo.GetCategoryByID(ctx, created.ID, false)
		assertAppErrorType(t, err, errors.ResourceNotFound)
		return nil
	})
}

func TestCategoryRepository_GetCategoryByID_DeletedIncluded(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		created := createTestCategory(t, ctx, tx, userID, "To Fetch Deleted")
		require.NoError(t, repo.DeleteCategory(ctx, created.ID, nil))

		fetched, err := repo.GetCategoryByID(ctx, created.ID, true)
		require.NoError(t, err)
		require.NotNil(t, fetched)
		assert.Equal(t, created.ID, fetched.ID)
		return nil
	})
}

func TestCategoryRepository_GetCategoryByID_HardDeletedNotFound(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		created := createTestCategory(t, ctx, tx, uuid.New(), "Hard Delete")
		hard := true
		require.NoError(t, repo.DeleteCategory(ctx, created.ID, &hard))

		_, err := repo.GetCategoryByID(ctx, created.ID, true)
		assertAppErrorType(t, err, errors.ResourceNotFound)
		return nil
	})
}

func TestCategoryRepository_GetAllCategories_DefaultPagination(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		createTestCategory(t, ctx, tx, userID, "A")
		createTestCategory(t, ctx, tx, userID, "B")
		query := (&schema.GetCategoriesQuery{}).Normalize()
		cats, err := repo.GetAllCategories(ctx, &userID, query, true)
		require.NoError(t, err)
		assert.Len(t, cats, 2)
		return nil
	})
}

func TestCategoryRepository_GetAllCategories_FilterByUserID(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userA, userB := uuid.New(), uuid.New()
		createTestCategory(t, ctx, tx, userA, "A's")
		createTestCategory(t, ctx, tx, userB, "B's")
		query := (&schema.GetCategoriesQuery{}).Normalize()
		cats, err := repo.GetAllCategories(ctx, &userA, query, true)
		require.NoError(t, err)
		require.Len(t, cats, 1)
		assert.Equal(t, "A's", cats[0].Name)
		return nil
	})
}

func TestCategoryRepository_GetAllCategories_Search(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		createTestCategory(t, ctx, tx, userID, "Shopping List")
		createTestCategory(t, ctx, tx, userID, "Work Tasks")
		search := "shop"
		query := (&schema.GetCategoriesQuery{Search: &search}).Normalize()
		cats, err := repo.GetAllCategories(ctx, &userID, query, true)
		require.NoError(t, err)
		require.Len(t, cats, 1)
		assert.Equal(t, "Shopping List", cats[0].Name)
		return nil
	})
}

func TestCategoryRepository_GetAllCategories_OrderByName(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		createTestCategory(t, ctx, tx, userID, "Gamma")
		createTestCategory(t, ctx, tx, userID, "Alpha")
		createTestCategory(t, ctx, tx, userID, "Beta")
		query := (&schema.GetCategoriesQuery{OrderBy: []string{"name"}}).Normalize()
		cats, err := repo.GetAllCategories(ctx, &userID, query, true)
		require.NoError(t, err)
		require.Len(t, cats, 3)
		assert.Equal(t, "Alpha", cats[0].Name)
		assert.Equal(t, "Beta", cats[1].Name)
		assert.Equal(t, "Gamma", cats[2].Name)
		return nil
	})
}

func TestCategoryRepository_GetAllCategories_ExcludeSoftDeleted(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		createTestCategory(t, ctx, tx, userID, "Active")
		deleted := createTestCategory(t, ctx, tx, userID, "Deleted")
		require.NoError(t, repo.DeleteCategory(ctx, deleted.ID, nil))
		query := (&schema.GetCategoriesQuery{}).Normalize()
		cats, err := repo.GetAllCategories(ctx, &userID, query, false)
		require.NoError(t, err)
		require.Len(t, cats, 1)
		assert.Equal(t, "Active", cats[0].Name)
		return nil
	})
}

func TestCategoryRepository_GetAllCategories_NoResults(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		query := (&schema.GetCategoriesQuery{}).Normalize()
		nonExistentUser := uuid.New()
		cats, err := repo.GetAllCategories(ctx, &nonExistentUser, query, true)
		require.NoError(t, err)
		assert.Empty(t, cats)
		return nil
	})
}

func TestCategoryRepository_GetAllCategories_CustomPagination(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		for _, name := range []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"} {
			createTestCategory(t, ctx, tx, userID, name)
		}
		limit := 2
		page := 2
		query := (&schema.GetCategoriesQuery{Limit: &limit, Page: &page, OrderBy: []string{"name"}}).Normalize()
		cats, err := repo.GetAllCategories(ctx, &userID, query, true)
		require.NoError(t, err)
		require.Len(t, cats, 2)
		assert.Equal(t, "Delta", cats[0].Name)
		assert.Equal(t, "Epsilon", cats[1].Name)
		return nil
	})
}

func TestCategoryRepository_GetAllCategories_OrderByDescendingName(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		createTestCategory(t, ctx, tx, userID, "Alpha")
		createTestCategory(t, ctx, tx, userID, "Beta")
		createTestCategory(t, ctx, tx, userID, "Gamma")
		query := (&schema.GetCategoriesQuery{OrderBy: []string{"-name"}}).Normalize()
		cats, err := repo.GetAllCategories(ctx, &userID, query, true)
		require.NoError(t, err)
		require.Len(t, cats, 3)
		assert.Equal(t, "Gamma", cats[0].Name)
		assert.Equal(t, "Beta", cats[1].Name)
		assert.Equal(t, "Alpha", cats[2].Name)
		return nil
	})
}

func TestCategoryRepository_GetAllCategories_OrderByCreatedAt(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		createTestCategory(t, ctx, tx, userID, "First")
		createTestCategory(t, ctx, tx, userID, "Second")
		createTestCategory(t, ctx, tx, userID, "Third")
		query := (&schema.GetCategoriesQuery{OrderBy: []string{"created_at"}}).Normalize()
		cats, err := repo.GetAllCategories(ctx, &userID, query, true)
		require.NoError(t, err)
		require.Len(t, cats, 3)
		return nil
	})
}

func TestCategoryRepository_GetAllCategories_IncludeDeletedRecords(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		createTestCategory(t, ctx, tx, userID, "Active")
		deleted := createTestCategory(t, ctx, tx, userID, "Deleted")
		require.NoError(t, repo.DeleteCategory(ctx, deleted.ID, nil))
		query := (&schema.GetCategoriesQuery{}).Normalize()
		cats, err := repo.GetAllCategories(ctx, &userID, query, true)
		require.NoError(t, err)
		require.Len(t, cats, 2)
		return nil
	})
}

func TestCategoryRepository_GetAllCategories_NilUserID(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		createTestCategory(t, ctx, tx, uuid.New(), "UserA")
		createTestCategory(t, ctx, tx, uuid.New(), "UserB")
		query := (&schema.GetCategoriesQuery{}).Normalize()
		cats, err := repo.GetAllCategories(ctx, nil, query, true)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(cats), 2)
		return nil
	})
}

func TestCategoryRepository_UpdateCategory_Name(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		created := createTestCategory(t, ctx, tx, uuid.New(), "Old")
		newName := "New"
		updated, err := repo.UpdateCategory(ctx, created.ID, &schema.UpdateCategoryRequest{Name: &newName}, false)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, "New", updated.Name)
		assert.Equal(t, created.ID, updated.ID)
		return nil
	})
}

func TestCategoryRepository_UpdateCategory_Description(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		created := createTestCategory(t, ctx, tx, uuid.New(), "Desc")
		newDesc := "Updated"
		updated, err := repo.UpdateCategory(ctx, created.ID, &schema.UpdateCategoryRequest{Description: &newDesc}, false)
		require.NoError(t, err)
		require.NotNil(t, updated)
		require.NotNil(t, updated.Description)
		assert.Equal(t, "Updated", *updated.Description)
		return nil
	})
}

func TestCategoryRepository_UpdateCategory_MetadataMerge(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		created := createTestCategory(t, ctx, tx, uuid.New(), "Meta")
		meta := map[string]any{"key": "val"}
		updated, err := repo.UpdateCategory(ctx, created.ID, &schema.UpdateCategoryRequest{Metadata: &meta}, false)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, "val", updated.Metadata["key"])
		return nil
	})
}

func TestCategoryRepository_UpdateCategory_AllFields(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		created := createTestCategory(t, ctx, tx, uuid.New(), "Orig")
		n, d := "Updated", "New desc"
		m := map[string]any{"p": "h"}
		updated, err := repo.UpdateCategory(ctx, created.ID, &schema.UpdateCategoryRequest{Name: &n, Description: &d, Metadata: &m}, false)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, "Updated", updated.Name)
		require.NotNil(t, updated.Description)
		require.Equal(t, "h", updated.Metadata["p"])
		assert.Equal(t, "New desc", *updated.Description)
		return nil
	})
}

func TestCategoryRepository_UpdateCategory_NoChanges(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		created := createTestCategory(t, ctx, tx, uuid.New(), "No Change")
		updated, err := repo.UpdateCategory(ctx, created.ID, &schema.UpdateCategoryRequest{}, false)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, created.ID, updated.ID)
		assert.Equal(t, "No Change", updated.Name)
		return nil
	})
}

func TestCategoryRepository_UpdateCategory_NotFound(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		n := "Should Fail"
		_, err := repo.UpdateCategory(ctx, uuid.New(), &schema.UpdateCategoryRequest{Name: &n}, false)
		assertAppErrorType(t, err, errors.ResourceNotFound)
		return nil
	})
}

func TestCategoryRepository_UpdateCategory_SoftDeletedNotConsidered(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		created := createTestCategory(t, ctx, tx, uuid.New(), "To Update Deleted")
		require.NoError(t, repo.DeleteCategory(ctx, created.ID, nil))
		n := "New Name"
		_, err := repo.UpdateCategory(ctx, created.ID, &schema.UpdateCategoryRequest{Name: &n}, false)
		assertAppErrorType(t, err, errors.ResourceNotFound)
		return nil
	})
}

func TestCategoryRepository_UpdateCategory_DuplicateName(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		userID := uuid.New()
		createTestCategory(t, ctx, tx, userID, "Existing")
		created := createTestCategory(t, ctx, tx, userID, "To Rename")
		n := "Existing"
		_, err := repo.UpdateCategory(ctx, created.ID, &schema.UpdateCategoryRequest{Name: &n}, false)
		assertAppErrorType(t, err, errors.ResourceAlreadyExists)
		return nil
	})
}

func TestCategoryRepository_UpdateCategory_ConsiderDeletedRecords(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		created := createTestCategory(t, ctx, tx, uuid.New(), "Soft Deleted Update")
		require.NoError(t, repo.DeleteCategory(ctx, created.ID, nil))
		n := "Updated After Deletion"
		updated, err := repo.UpdateCategory(ctx, created.ID, &schema.UpdateCategoryRequest{Name: &n}, true)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, "Updated After Deletion", updated.Name)
		return nil
	})
}

func TestCategoryRepository_UpdateCategory_EmptyDescription(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		desc := "initial"
		created := createTestCategory(t, ctx, tx, uuid.New(), "Clear Desc")
		_, err := repo.UpdateCategory(ctx, created.ID, &schema.UpdateCategoryRequest{Description: &desc}, false)
		require.NoError(t, err)
		empty := ""
		updated, err := repo.UpdateCategory(ctx, created.ID, &schema.UpdateCategoryRequest{Description: &empty}, false)
		require.NoError(t, err)
		require.NotNil(t, updated.Description)
		assert.Equal(t, "", *updated.Description)
		return nil
	})
}

func TestCategoryRepository_UpdateCategory_HardDeleted(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		created := createTestCategory(t, ctx, tx, uuid.New(), "Hard Delete Update")
		hard := true
		require.NoError(t, repo.DeleteCategory(ctx, created.ID, &hard))
		n := "Should Fail"
		_, err := repo.UpdateCategory(ctx, created.ID, &schema.UpdateCategoryRequest{Name: &n}, true)
		assertAppErrorType(t, err, errors.ResourceNotFound)
		return nil
	})
}

func TestCategoryRepository_DeleteCategory_SoftDelete(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		created := createTestCategory(t, ctx, tx, uuid.New(), "Soft")
		require.NoError(t, repo.DeleteCategory(ctx, created.ID, nil))

		_, err := repo.GetCategoryByID(ctx, created.ID, false)
		assertAppErrorType(t, err, errors.ResourceNotFound)

		_, err = repo.GetCategoryByID(ctx, created.ID, true)
		require.NoError(t, err)
		return nil
	})
}

func TestCategoryRepository_DeleteCategory_HardDelete(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		created := createTestCategory(t, ctx, tx, uuid.New(), "Hard")
		hard := true
		require.NoError(t, repo.DeleteCategory(ctx, created.ID, &hard))

		_, err := repo.GetCategoryByID(ctx, created.ID, true)
		assertAppErrorType(t, err, errors.ResourceNotFound)
		return nil
	})
}

func TestCategoryRepository_DeleteCategory_NonExistent(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		require.NoError(t, repo.DeleteCategory(ctx, uuid.New(), nil))
		return nil
	})
}

func TestCategoryRepository_DeleteCategory_DefaultIsSoftDelete(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		created := createTestCategory(t, ctx, tx, uuid.New(), "Default")
		require.NoError(t, repo.DeleteCategory(ctx, created.ID, nil))

		_, err := repo.GetCategoryByID(ctx, created.ID, false)
		assertAppErrorType(t, err, errors.ResourceNotFound)
		return nil
	})
}

func TestCategoryRepository_DeleteCategory_DoubleSoftDelete(t *testing.T) {
	ctx := context.Background()
	svc := pkgTesting.Services()
	pkgTesting.WithRollbackTransaction(ctx, svc.DB(), func(tx pgx.Tx) error {
		repo := repository.New(svc.DB()).CategoryRepository.WithExecutor(tx)
		created := createTestCategory(t, ctx, tx, uuid.New(), "Double Soft")
		require.NoError(t, repo.DeleteCategory(ctx, created.ID, nil))
		require.NoError(t, repo.DeleteCategory(ctx, created.ID, nil))
		return nil
	})
}
