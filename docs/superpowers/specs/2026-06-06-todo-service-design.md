# Todo Service Design

## Status

Approved.

## Overview

Add a TodoService layer with parent-child status synchronization to the existing TodoRepository. The service uses `database.QueryInTransaction` for atomic multi-step operations and runs GetAll+Count in parallel via errgroup.

## New Files

### `internal/schema/todo.go`

Domain types for the Todo domain — separate from `dto` package (which is DB-mapping only).

**Types:**

| Type | Description |
|---|---|
| `TodoStatus` | String enum: `draft`, `pending`, `in_progress`, `completed`, `archived` |
| `Todo` | Full todo with `Children []TodoListItems` |
| `TodoListItems` | Lightweight: id, title, status, priority, due_date, category_id |
| `CreateTodoRequest` | Validable — validates title, status, priority |
| `UpdateTodoRequest` | Validable — uses `schema.Nullable` for partial updates |
| `GetTodosQuery` | Validable + Normalizable — page, limit, search, status, priority, category_id, parent_id, order_by |

All request/query types implement `validation.Validable`. `GetTodosQuery` also implements `normalization.Normalizable`.

### `internal/service/todo.go`

```go
type TodoRepository interface {
    Create(ctx, *dto.CreateTodoRequest) (*dto.Todo, error)
    GetByID(ctx, todoID, userID uuid.UUID, includeDeleted) (*dto.Todo, error)
    GetAll(ctx, *string, *dto.GetTodosQuery, bool) ([]dto.TodoListItems, error)
    Count(ctx, *string, *dto.GetTodosQuery, bool) (int, error)
    Update(ctx, todoID, userID uuid.UUID, *dto.UpdateTodoRequest) (*dto.Todo, error)
    Delete(ctx, todoID, userID uuid.UUID) error
    WithExecutor(database.DBTX) *TodoRepository
}

type TodoService struct {
    repo TodoRepository
    db   database.DBTX
}
```

### Service Methods

#### `Create(ctx, userID, *schema.CreateTodoRequest) (*schema.Todo, error)`

Flow:
1. Convert to `dto.CreateTodoRequest`
2. `database.QueryInTransaction(ctx, s.db, func(tx) {
     todo = repo.WithExecutor(tx).Create(req)
     if todo.ParentID != nil {
         parent = repo.WithExecutor(tx).GetByID(*todo.ParentID, userID)
         if parent.Status == "completed" {
             repo.WithExecutor(tx).Update(parent.ID, userID, set status=in_progress, completed_at=null)
         }
     }
     return todo
   })`
3. Convert `dto.Todo` → `schema.Todo`

#### `GetByID(ctx, todoID, userID uuid.UUID, includeDeleted bool) (*schema.Todo, error)`

Flow:
1. `todo = repo.GetByID(ctx, todoID, userID, includeDeleted)`
2. `children = repo.GetAll(ctx, &userID, &dto.GetTodosQuery{ParentID: &todoID}, includeDeleted)` — no transaction needed for reads
3. Convert + merge into `schema.Todo` with `Children`

#### `Update(ctx, todoID, userID uuid.UUID, payload *schema.UpdateTodoRequest) (*schema.Todo, error)`

Flow:
1. Convert to `dto.UpdateTodoRequest`
2. `database.QueryInTransaction(ctx, s.db, func(tx) {
     r = repo.WithExecutor(tx)
     todo = r.GetByID(todoID, userID)
     oldParentID = todo.ParentID
     updatedTodo = r.Update(todoID, userID, payload)
     if payload.Status changed && updatedTodo.ParentID != nil {
         recalculateParentStatus(tx, r, *updatedTodo.ParentID, userID)
     }
     if payload.ParentID changed {
         oldParent = repo oldParentID
         if old parent now has all children completed/archived
             mark old parent as completed
         else if old parent was completed and still has incomplete children
             mark old parent as in_progress
         recalculateParentStatus(tx, r, newParentID, userID)
     }
     return updatedTodo
   })`

`recalculateParentStatus`: fetch all children of parent via `GetAll(parent_id=X)` with no status filter. If all are `completed` or `archived`, update parent to `completed` + `completed_at=now`. If parent was `completed` but not all children are done, set to `in_progress` + clear `completed_at`.

#### `GetAll(ctx, userID, *schema.GetTodosQuery) (*schema.PaginatedResponse[schema.TodoListItems], error)`

Flow:
1. Convert schema query to dto query (compute offset from page/limit)
2. Use `errgroup` to run `repo.GetAll` + `repo.Count` in parallel
3. Return paginated response

#### `Delete(ctx, todoID, userID uuid.UUID) error`

Simple soft-delete delegation to repo. No parent status recalculation on delete (unless explicitly added later).

## Modified Files

### `internal/schema/dto/todo.go`

- Add `ParentID *uuid.UUID` with `db:"parent_id,omitempty"` to `GetTodosQuery`

### `internal/repository/todo.go`

- `GetAll`: add `parent_id = @parent_id` to `whereClause` when `q.ParentID != nil`
- `Count`: same `parent_id` filter addition
- No new repository methods needed

### `internal/service/service.go`

- Add `TodoService *TodoService` field
- Add `ToService := NewTodoService(repo.TodoRepository, db)` in `New()`

## Parallel Execution

- `GetAll` + `Count`: run with `errgroup.Group` — two goroutines, each using the pool directly (reads only, no shared transaction)
- `GetByID` + children: sequential queries in the same call (not true parallel, but the DB handles this fast enough)

## Error Handling

- Repo errors already typed as `*pkgErrors.AppError`
- Service adds context: wraps with observability logging
- Validation errors from schema types caught before any DB call
- Ownership check: verify `todo.UserID == userID.String()` before returning

## Testing

- `TodoService` uses repository interface — full mock testing in `internal/service/todo_test.go`
- Tests: create sub-todo with completed parent, update sub-todo to trigger parent completion, reparenting scenarios, parallel GetAll+Count
