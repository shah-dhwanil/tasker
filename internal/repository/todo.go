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
type Todo interface {
	Create(ctx context.Context, req *dto.CreateTodoRequest) (*dto.Todo, error)
	GetByID(ctx context.Context, todoID uuid.UUID, userID string, includeDeleted bool) (*dto.Todo, error)
	GetAll(ctx context.Context, userID *string, q *dto.GetTodosQuery, includeDeleted bool) ([]dto.TodoListItems, error)
	Count(ctx context.Context, userID *string, q *dto.GetTodosQuery, includeDeleted bool) (int, error)
	Update(ctx context.Context, todoID uuid.UUID, userID string, payload *dto.UpdateTodoRequest) (*dto.Todo, error)
	Delete(ctx context.Context, todoID uuid.UUID, userID string) error
	WithExecutor(executor database.DBTX) Todo
}


type TodoRepository struct {
	executor database.DBTX
}

func newTodoRepository(executor database.DBTX) *TodoRepository {
	return &TodoRepository{
		executor: executor,
	}
}

func (r *TodoRepository) WithExecutor(executor database.DBTX) Todo {
	return &TodoRepository{
		executor: executor,
	}
}

const createTodoQuery = `
INSERT INTO tasker.todos (id, user_id, category_id, title, description, status, priority, due_date, parent_id, metadata)
VALUES (@id, @user_id, @category_id, @title, @description, @status, @priority, @due_date, @parent_id, @metadata)
RETURNING id, user_id, category_id, title, description, status, priority, due_date, completed_at, parent_id, metadata, is_deleted, created_at, updated_at
`

func (r *TodoRepository) Create(ctx context.Context, req *dto.CreateTodoRequest) (*dto.Todo, error) {
	logger := observability.FromContext(ctx)

	id, err := uuid.NewV7()
	if err != nil {
		id = uuid.New()
		logger.Warn("Failed to generate UUIDv7, falling back to UUIDv4", zap.Error(err))
	}

	args, err := database.StructToNamedArgs(req)
	if err != nil {
		return nil, pkgErrors.NewStructToPayloadConversionError(err, "Todo.Create")
	}

	args["id"] = id.String()

	rows, err := database.QueryInTransaction(ctx, r.executor,
		func(executor database.Transaction) (dto.Todo, error) {
			rows, _ := executor.Query(ctx, createTodoQuery, args)
			return pgx.CollectOneRow(rows, pgx.RowToStructByName[dto.Todo])
		},
	)
	if err != nil {
		return nil, mapErrorToTodoRepositoryError(err, args)
	}
	return &rows, nil
}

const getTodoByIDQuery = `
SELECT id, user_id, category_id, title, description, status, priority, due_date, completed_at, parent_id, metadata, is_deleted, created_at, updated_at
FROM tasker.todos
WHERE id = @id AND user_id = @user_id
`

func (r *TodoRepository) GetByID(ctx context.Context, todoID uuid.UUID, userID string, includeDeleted bool) (*dto.Todo, error) {
	args := pgx.NamedArgs{
		"id":      todoID.String(),
		"user_id": userID,
	}
	getByIDQueryWithDeleted := getTodoByIDQuery
	if !includeDeleted {
		getByIDQueryWithDeleted = getTodoByIDQuery + " AND is_deleted = false"
	}

	todoRes, err := database.QueryInTransaction(
		ctx,
		r.executor,
		func(d database.Transaction) (dto.Todo, error) {
			rows, _ := d.Query(ctx, getByIDQueryWithDeleted, args)
			return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[dto.Todo])
		},
	)

	if err != nil {
		return nil, mapErrorToTodoRepositoryError(err, args)
	}
	return &todoRes, nil
}

const deleteTodoQuery = `
UPDATE tasker.todos
SET is_deleted = true
WHERE id = @id AND user_id = @user_id AND is_deleted = false
`

const getAllTodosQuery = `
SELECT id, title, status, priority, due_date, category_id
FROM tasker.todos
WHERE %s
ORDER BY %s
LIMIT @limit OFFSET @offset
`

func (r *TodoRepository) GetAll(ctx context.Context, userID *string, q *dto.GetTodosQuery, includeDeleted bool) ([]dto.TodoListItems, error) {
	whereClause := make([]string, 0)
	args,err := database.StructToNamedArgs(q)
	if err != nil {
		return nil, pkgErrors.NewStructToPayloadConversionError(err, "Todo.GetAll")
	}
	// args := pgx.NamedArgs{
	// 	"user_id": userID.String(),
	// }
	if userID != nil {
		whereClause = append(whereClause, "user_id = @user_id")
		args["user_id"] = *userID
	}
	if !includeDeleted {
		whereClause = append(whereClause, "is_deleted = false")
	}
	if q.Status != nil {
		whereClause = append(whereClause, "status = @status")
	}
	if q.Priority != nil {
		whereClause = append(whereClause, "priority = @priority")
		args["priority"] = q.Priority
	}
	if q.CategoryID != nil {
		whereClause = append(whereClause, "category_id = @category_id")
		args["category_id"] = q.CategoryID.String()
	}
	if q.Search != nil {
		whereClause = append(whereClause, "search_vector @@ plainto_tsquery('english', @search)")
		args["search"] = q.Search
	}
	if q.ParentID != nil {
		whereClause = append(whereClause, "parent_id = @parent_id")
		args["parent_id"] = q.ParentID.String()
	}
	orderByClause := make([]string, 0)
	if len(q.OrderBy) > 0 {
		for _, orderBy := range q.OrderBy {
			col, dir := database.ExtractOrderParam(orderBy)
			orderByClause = append(orderByClause, fmt.Sprintf("%s %s", col, dir))
		}
	} else {
		orderByClause = append(orderByClause, "created_at DESC")
	}

	query := fmt.Sprintf(getAllTodosQuery, database.ConstructWhereClause(whereClause), database.ConstructOrderByClause(orderByClause))

	todos, err := database.QueryInTransaction(
		ctx,
		r.executor,
		func(executor database.Transaction) ([]dto.TodoListItems, error) {
			rows, _ := executor.Query(ctx, query, args)
			return pgx.CollectRows(rows, pgx.RowToStructByName[dto.TodoListItems])
		},
	)
	if err != nil {
		return nil, mapErrorToTodoRepositoryError(err, args)
	}
	return todos, nil
}

const countTodosQuery = `
SELECT COUNT(*)
FROM tasker.todos
WHERE %s
`

func (r *TodoRepository) Count(ctx context.Context, userID *string, q *dto.GetTodosQuery, includeDeleted bool) (int, error) {
	whereClause := make([]string, 0)
	args,err := database.StructToNamedArgs(q)
	if err != nil {
		return 0, pkgErrors.NewStructToPayloadConversionError(err, "Todo.Count")
	}
	// args := pgx.NamedArgs{
	// 	"user_id": userID.String(),
	// }
	if userID != nil {
		whereClause = append(whereClause, "user_id = @user_id")
		args["user_id"] = *userID
	}
	if !includeDeleted {
		whereClause = append(whereClause, "is_deleted = false")
	}
	if q.Status != nil {
		whereClause = append(whereClause, "status = @status")
	}
	if q.Priority != nil{
		whereClause = append(whereClause, "priority = @priority")
	}
	if q.CategoryID != nil {
		whereClause = append(whereClause, "category_id = @category_id")
		args["category_id"] = q.CategoryID.String()
	}
	if q.Search != nil {
		whereClause = append(whereClause, "search_vector @@ plainto_tsquery('english', @search)")
	}
	if q.ParentID != nil {
		whereClause = append(whereClause, "parent_id = @parent_id")
		args["parent_id"] = q.ParentID.String()
	}

	query := fmt.Sprintf(countTodosQuery, database.ConstructWhereClause(whereClause))

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
		return 0, mapErrorToTodoRepositoryError(err, args)
	}
	return count, nil
}

const updateTodoQuery = `
UPDATE tasker.todos
SET %s
WHERE id = @id AND user_id = @user_id AND is_deleted = false
RETURNING id, user_id, category_id, title, description, status, priority, due_date, completed_at, parent_id, metadata, is_deleted, created_at, updated_at
`

func (r *TodoRepository) Update(ctx context.Context, todoID uuid.UUID, userID string, payload *dto.UpdateTodoRequest) (*dto.Todo, error) {
	setClause := make([]string, 0)
	args, err := database.StructToNamedArgs(payload)
	if err != nil {
		return nil, pkgErrors.NewStructToPayloadConversionError(err, "Todo.Update")
	}
	args["id"] = todoID.String()
	if payload.Title != nil {
		setClause = append(setClause, "title = @title")
	}
	if payload.Status != nil {
		setClause = append(setClause, "status = @status")
	}
	if payload.Priority != nil {
		setClause = append(setClause, "priority = @priority")
	}
	if payload.Description != nil {
		setClause = append(setClause, "description = @description")
	}
	if payload.DueDate.IsSet() {
		setClause = append(setClause, "due_date = @due_date")
	}
	if payload.CompletedAt.IsSet() {
		setClause = append(setClause, "completed_at = @completed_at")
	}
	if payload.CategoryID != nil {
		setClause = append(setClause, "category_id = @category_id")
	}
	if payload.ParentID.IsSet() {
		setClause = append(setClause, "parent_id = @parent_id")
	}
	if payload.Metadata != nil {
		setClause = append(setClause, "metadata = COALESCE(metadata, '{}'::jsonb) || @metadata::jsonb")
		args["metadata"] = *payload.Metadata
	}

	if len(setClause) == 0 {
		res, err := r.GetByID(ctx, todoID, userID, false)
		if err != nil {
			return nil, err
		}
		return res,nil
	}

	query := fmt.Sprintf(updateTodoQuery, database.ConstructSetClause(setClause))
	todoRes, err := database.QueryInTransaction(
		ctx,
		r.executor,
		func(executor database.Transaction) (dto.Todo, error) {
			rows, _ := executor.Query(ctx, query, args)
			return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[dto.Todo])
		},
	)
	if err != nil {
		return nil, mapErrorToTodoRepositoryError(err, args)
	}
	return &todoRes, nil
}

func (r *TodoRepository) Delete(ctx context.Context, todoID uuid.UUID, userID string) error {
	args := pgx.NamedArgs{
		"id":      todoID.String(),
		"user_id": userID,
	}

	ct, err := database.ExecuteInTransaction(
		ctx,
		r.executor,
		func(executor database.Transaction) (pgconn.CommandTag, error) {
			return executor.Exec(ctx, deleteTodoQuery, args)
		},
	)
	if err != nil {
		return mapErrorToTodoRepositoryError(err, args)
	}
	if ct.RowsAffected() == 0 {
		return pkgErrors.NewTodoNotFoundError(nil, nil)
	}
	return nil
}

func mapErrorToTodoRepositoryError(err error, payload pgx.NamedArgs) error {
	dbErr, ok := pkgErrors.ConvertPgError(err)
	if !ok {
		return pkgErrors.NewUnknownError(err, "Database Error", "Unknown Error while fetching record from postgres", nil)
	}
	pgErr, ok := dbErr.(*pkgErrors.DatabaseError)
	if !ok {
		return pkgErrors.NewUnknownError(err, "Database Error", "Unknown Error while fetching record from postgres", nil)
	}

	switch pgErr.Code {
	case pkgErrors.ForeignKeyViolation:
		switch pgErr.ConstraintName {
		case "fk_todo_category":
			return pkgErrors.NewTodoCategoryNotFoundError(pgErr)
		case "fk_todo_parent":
			return pkgErrors.NewTodoParentNotFoundError(pgErr)
		default:
			return pkgErrors.NewUnknownError(pgErr, "Database Constraint Error", fmt.Sprintf("Foreign key violation: %s", pgErr.ConstraintName), nil)
		}
	case pkgErrors.CheckViolation:
		switch pgErr.ConstraintName {
		case "chk_todo_no_self_parent":
			return pkgErrors.NewTodoSelfParentError(pgErr)
		default:
			return pkgErrors.NewUnknownError(pgErr, "Database Constraint Error", fmt.Sprintf("Check violation: %s", pgErr.ConstraintName), nil)
		}
	case pkgErrors.NoRecordsFound:
		return pkgErrors.NewTodoNotFoundError(pgErr, nil)
	case pkgErrors.UniqueViolation:
		return pkgErrors.NewTodoAlreadyExistsError(pgErr, nil)
	default:
		return pkgErrors.NewUnknownError(pgErr, "Database Error", "Unknown Error while fetching record from postgres", nil)
	}
}
