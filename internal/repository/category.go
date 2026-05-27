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
	"github.com/shah-dhwanil/tasker/internal/schema"
	"go.uber.org/zap"
)

type CategoryRepository struct {
	executor database.DBTX
}

func newCategoryRepository(executor database.DBTX) *CategoryRepository {
	return &CategoryRepository{
		executor: executor,
	}
}

func(r *CategoryRepository) WithExecutor(executor database.DBTX) *CategoryRepository {
	return &CategoryRepository{
		executor: executor,
	}
}

const createQuery = `
INSERT INTO tasker.todo_categories (id, name, user_id, description, metadata)
VALUES (@id, @name, @user_id, @description, @metadata)
RETURNING id, name, description, metadata, created_at, updated_at
`

func (r *CategoryRepository) CreateCategory(ctx context.Context, user_id uuid.UUID, category *schema.CreateCategoryRequest) (*schema.CreateCategoryResponse, error) {
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
	args["user_id"] = user_id.String()
	
	rows, err := database.QueryInTransaction(ctx,r.executor,
		func(executor database.Transaction) (schema.CreateCategoryResponse, error) {
			rows, _ := executor.Query(ctx, createQuery, args)
			return pgx.CollectOneRow(rows, pgx.RowToStructByName[schema.CreateCategoryResponse])
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

func (r *CategoryRepository) GetCategoryByID(ctx context.Context, categoryID uuid.UUID, includeDeletedRecord bool) (*schema.Category, error) {
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
		func(d database.Transaction) (schema.Category,error) {
			rows, _ := r.executor.Query(ctx, getByIDQueryWithDeleted, args)
			return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[schema.Category])
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

func (r *CategoryRepository) GetAllCategories(ctx context.Context, userID *uuid.UUID, payload *schema.GetCategoriesQuery,includeDeletedRecords bool) ([]schema.GetCategoriesResponse, error){
	whereClause := make([]string, 0)
	args, err := database.StructToNamedArgs(payload)
	if err != nil {
		return nil, pkgErrors.NewStructToPayloadConversionError(err, "Category.GetAll")
	}
	if userID != nil {
		whereClause = append(whereClause, "user_id = @user_id")
		args["user_id"] = userID.String()
	}
	if payload.Search != nil {
		whereClause = append(whereClause, "name ILIKE @search")
		args["search"] = fmt.Sprintf("%%%s%%", *payload.Search)
	}
	if !includeDeletedRecords {
		whereClause = append(whereClause, "is_deleted = false")
	}
	if payload.Limit != nil && payload.Page != nil {
		args["offset"] = (*payload.Page - 1) * *payload.Limit
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
		func(executor database.Transaction) ([]schema.GetCategoriesResponse,error) {
			 rows, _ := executor.Query(ctx, query, args)
			 return pgx.CollectRows(rows, pgx.RowToStructByName[schema.GetCategoriesResponse])
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
RETURNING id, name, description, metadata, created_at, updated_at
`
func (r *CategoryRepository) UpdateCategory(ctx context.Context, categoryID uuid.UUID, payload *schema.UpdateCategoryRequest,considerDeletedRecords bool) (*schema.UpdateCategoryResponse, error) {
	setClause := make([]string, 0)
	args, err := database.StructToNamedArgs(payload)
	if err != nil {
		return nil, pkgErrors.NewStructToPayloadConversionError(err, "Category.Update")
	}
	args["id"] = categoryID.String()
	if payload.Name != nil {
		setClause = append(setClause, "name = @name")
	}
	if payload.Description != nil {
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
		return &schema.UpdateCategoryResponse{
			ID:          res.ID,
			Name:        res.Name,
			Description: res.Description,
			Metadata:    res.Metadata,
			CreatedAt:   res.CreatedAt,
			UpdatedAt:   res.UpdatedAt,
		}, nil
	}
	isDeleteClause := ""
	if !considerDeletedRecords {
		isDeleteClause = "AND is_deleted = false"
	}
	query := fmt.Sprintf(updateCategoryQuery, database.ConstructSetClause(setClause),isDeleteClause)
	categoryRes, err := database.QueryInTransaction(
		ctx,
		r.executor,
		func(executor database.Transaction) (schema.UpdateCategoryResponse,error) {
			rows, _ := executor.Query(ctx, query, args)
			return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[schema.UpdateCategoryResponse])
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