package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shah-dhwanil/tasker/internal/database"
	pkgErrors "github.com/shah-dhwanil/tasker/internal/errors"
	"github.com/shah-dhwanil/tasker/internal/observability"
	"github.com/shah-dhwanil/tasker/internal/schema/dto"
	"go.uber.org/zap"
)

type Category interface {
	CreateCategory(ctx context.Context, userID string, req *dto.CreateCategoryRequest) (*dto.Category, error)
	GetCategoryByID(ctx context.Context, categoryID uuid.UUID, includeDeletedRecord bool) (*dto.Category, error)
	GetAllCategories(ctx context.Context, userID *string, payload *dto.GetCategoriesQuery, includeDeletedRecords bool) ([]dto.CategoriesListItems, error)
	CountCategories(ctx context.Context, userID *string, payload *dto.GetCategoriesQuery, includeDeletedRecords bool) (int, error)
	UpdateCategory(ctx context.Context, categoryID uuid.UUID, payload *dto.UpdateCategoryRequest, considerDeletedRecords bool) (*dto.Category, error)
	DeleteCategory(ctx context.Context, categoryID uuid.UUID, isHardDelete *bool) error
}


type CategoryRepository struct {
	executor database.DBTX
}

func newCategoryRepository(executor database.DBTX) *CategoryRepository {
	return &CategoryRepository{
		executor: executor,
	}
}

func(r *CategoryRepository) WithExecutor(executor database.DBTX) Category {
	return &CategoryRepository{
		executor: executor,
	}
}

const createQuery = `
INSERT INTO tasker.todo_categories (id, name, user_id, description, metadata)
VALUES (@id, @name, @user_id, @description, @metadata)
RETURNING id, user_id, name, description, metadata, created_at, updated_at
`

func (r *CategoryRepository) CreateCategory(ctx context.Context, user_id string, category *dto.CreateCategoryRequest) (*dto.Category, error) {
	logger := observability.FromContext(ctx)
	id, err := uuid.NewV7()
	if err != nil {
		id = uuid.New()
		logger.Warn("Failed to generate UUIDv7, falling back to UUIDv4",zap.Error(err))
	}
	args, err := database.StructToNamedArgs(category)
	if err != nil {
		return nil, pkgErrors.NewStructToPayloadConversionError(err, "Category.Create")
	}
	args["id"] = id.String()
	args["user_id"] = user_id
	
	rows, err := database.QueryInTransaction(ctx,r.executor,
		func(executor database.Transaction) (dto.Category, error) {
			rows, _ := executor.Query(ctx, createQuery, args)
			return pgx.CollectOneRow(rows, pgx.RowToStructByName[dto.Category])
		},
	)
	if err != nil {
		return nil, mapErrorToCategoryRepositoryError(err,args)
	}
	return &rows, nil
}

const getByIDQuery = `
SELECT id, name, user_id, description, metadata, created_at, updated_at
FROM tasker.todo_categories
WHERE id = @id
`

func (r *CategoryRepository) GetCategoryByID(ctx context.Context, categoryID uuid.UUID, includeDeletedRecord bool) (*dto.Category, error) {
	args := pgx.NamedArgs{
		"id": categoryID.String(),
	}
	getByIDQueryWithDeleted := getByIDQuery
	if !includeDeletedRecord {
		getByIDQueryWithDeleted = getByIDQuery + " AND is_deleted = false"
	}
	categoryRes,err:= database.QueryInTransaction(
		ctx,
		r.executor,
		func(d database.Transaction) (dto.Category,error) {
			rows, _ := d.Query(ctx, getByIDQueryWithDeleted, args)
			return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[dto.Category])
		},
	)
	
	if err != nil {
		return nil, mapErrorToCategoryRepositoryError(err,args)
	}
	return &categoryRes, nil
}

const getAllCategoriesQuery = `
SELECT id, name
FROM tasker.todo_categories
WHERE %s
ORDER BY %s
LIMIT @limit OFFSET @offset
`

func (r *CategoryRepository) GetAllCategories(ctx context.Context, userID *string, payload *dto.GetCategoriesQuery,includeDeletedRecords bool) ([]dto.CategoriesListItems, error){
	whereClause := make([]string, 0)
	args, err := database.StructToNamedArgs(payload)
	if err != nil {
		return nil, pkgErrors.NewStructToPayloadConversionError(err, "Category.GetAll")
	}
	if userID != nil {
		whereClause = append(whereClause, "user_id = @user_id")
		args["user_id"] = userID
	}
	if payload.Search != nil {
		whereClause = append(whereClause, "name ILIKE @search")
		args["search"] = fmt.Sprintf("%%%s%%", *payload.Search)
	}
	if !includeDeletedRecords {
		whereClause = append(whereClause, "is_deleted = false")
	}
	orderByClause := make([]string, 0)
	for _, orderBy := range payload.OrderBy {
		col, dir := database.ExtractOrderParam(orderBy)
		orderByClause = append(orderByClause, fmt.Sprintf("%s %s", col, dir))
	}
	query := fmt.Sprintf(getAllCategoriesQuery, database.ConstructWhereClause(whereClause), database.ConstructOrderByClause(orderByClause))
	categories, err := database.QueryInTransaction(
		ctx,
		r.executor,
		func(executor database.Transaction) ([]dto.CategoriesListItems,error) {
			 rows, _ := executor.Query(ctx, query, args)
			 return pgx.CollectRows(rows, pgx.RowToStructByName[dto.CategoriesListItems])
		},
	)
	if err != nil {
		return nil, mapErrorToCategoryRepositoryError(err,args)
	}
	return categories, nil
}

const updateCategoryQuery = `
UPDATE tasker.todo_categories
SET %s
WHERE id = @id %s
RETURNING id, user_id, name, description, metadata, created_at, updated_at
`
func (r *CategoryRepository) UpdateCategory(ctx context.Context, categoryID uuid.UUID, payload *dto.UpdateCategoryRequest,considerDeletedRecords bool) (*dto.Category, error) {
	setClause := make([]string, 0)
	args, err := database.StructToNamedArgs(payload)
	if err != nil {
		return nil, pkgErrors.NewStructToPayloadConversionError(err, "Category.Update")
	}
	args["id"] = categoryID.String()
	if payload.Name != nil {
		setClause = append(setClause, "name = @name")
	}
	if payload.Description.IsSet() {
		setClause = append(setClause, "description = @description")
	}
	if payload.Metadata != nil {
		setClause = append(setClause, "metadata = COALESCE(metadata, '{}'::jsonb) || @metadata::jsonb")
		args["metadata"] = *payload.Metadata
	}
	if len(setClause) == 0 {
		res, err:=r.GetCategoryByID(ctx, categoryID,considerDeletedRecords)
		if err != nil {
			return nil, err
		}
		return res,nil
	}
	isDeleteClause := ""
	if !considerDeletedRecords {
		isDeleteClause = "AND is_deleted = false"
	}
	query := fmt.Sprintf(updateCategoryQuery, database.ConstructSetClause(setClause),isDeleteClause)
	categoryRes, err := database.QueryInTransaction(
		ctx,
		r.executor,
		func(executor database.Transaction) (dto.Category,error) {
			rows, _ := executor.Query(ctx, query, args)
			return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[dto.Category])
		},
	)
	if err != nil {
		return nil, mapErrorToCategoryRepositoryError(err,args)
	}
	return &categoryRes, nil
}

const deleteCategoryQuery = `
UPDATE tasker.todo_categories
SET is_deleted = true
WHERE id = @id and is_deleted = false
`

const hardDeleteCategoryQuery = `
DELETE FROM tasker.todo_categories
WHERE id = @id
`

func (r *CategoryRepository) DeleteCategory(ctx context.Context, categoryID uuid.UUID, isHardDelete *bool) error {
	args := pgx.NamedArgs{
		"id": categoryID.String(),
	}
	query := ""
	if isHardDelete != nil && *isHardDelete {
		query = hardDeleteCategoryQuery
	}else {
		query = deleteCategoryQuery
	}
	_,err:= database.ExecuteInTransaction(
		ctx,
		r.executor,
		func(executor database.Transaction) (pgconn.CommandTag,error) {
			 return executor.Exec(ctx, query, args)
		},
	)
	if err != nil {
		return mapErrorToCategoryRepositoryError(err,args)
	}
	return nil
}


const countCategoriesQuery = `
SELECT COUNT(*)
FROM tasker.todo_categories
WHERE %s
`

func (r *CategoryRepository) CountCategories(ctx context.Context, userID *string, payload *dto.GetCategoriesQuery, includeDeletedRecords bool) (int, error) {
	whereClause := make([]string, 0)
	args := pgx.NamedArgs{}
	if userID != nil {
		whereClause = append(whereClause, "user_id = @user_id")
		args["user_id"] = userID
	}
	if payload.Search != nil {
		whereClause = append(whereClause, "name ILIKE @search")
		args["search"] = fmt.Sprintf("%%%s%%", *payload.Search)
	}
	if !includeDeletedRecords {
		whereClause = append(whereClause, "is_deleted = false")
	}
	query := fmt.Sprintf(countCategoriesQuery, database.ConstructWhereClause(whereClause))
	
	count, err := database.QueryInTransaction(
		ctx,
		r.executor,
		func(executor database.Transaction) (int, error) {
			row := executor.QueryRow(ctx, query, args)
			var total int
			err := row.Scan(&total)
			return total, err
		},
	)
	if err != nil {
		return 0, mapErrorToCategoryRepositoryError(err, args)
	}
	return count, nil
}


func mapErrorToCategoryRepositoryError(err error, payload pgx.NamedArgs) error {
	err, ok := pkgErrors.ConvertPgError(err)
	if !ok {
		return pkgErrors.NewUnknownError(err,"Database Error","Unknown Error while fetching record from postgres",nil)
	}
	dbError,ok := err.(*pkgErrors.DatabaseError)
	if !ok {
		return pkgErrors.NewUnknownError(err,"Database Error","Unknown Error while fetching record from postgres",nil)
	}
	switch dbError.Code {
	case pkgErrors.UniqueViolation:
		switch dbError.ConstraintName {
		case "uniq_category_user_id_name":
			msg:= fmt.Sprintf("Category with %s already exists for the user", "(user_id,name)")
			var nameVal string
			switch v := payload["name"].(type) {
			case string:
				nameVal = v
			case *string:
				if v != nil {
					nameVal = *v
				}
			}
			return pkgErrors.NewCategoryAlreadyExistsError(dbError, "(name,user_id)", nameVal, &msg)
		default:
			return pkgErrors.NewUnknownError(dbError,"Database Constraint Error","Error while fetching record from postgres due to constraint failure",nil)
		}
	case pkgErrors.NoRecordsFound:
		return pkgErrors.NewCategoryNotFoundError(dbError, nil)
	default:
		return pkgErrors.NewUnknownError(err,"Database Error","Unknown Error while fetching record from postgres",nil)
	}
}