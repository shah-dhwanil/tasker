package service

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/shah-dhwanil/tasker/internal/database"
	"github.com/shah-dhwanil/tasker/internal/observability"
	"github.com/shah-dhwanil/tasker/internal/repository"
	"github.com/shah-dhwanil/tasker/internal/schema"
	"github.com/shah-dhwanil/tasker/internal/schema/dto"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)


type TodoService struct {
	repo repository.Todo
	db   database.DBTX
}

func NewTodoService(repo repository.Todo, db database.DBTX) *TodoService {
	return &TodoService{repo: repo, db: db}
}

func (s *TodoService) Create(ctx context.Context, userID string, req *schema.CreateTodoRequest) (*schema.Todo, error) {
	logger := observability.FromContext(ctx)
	dtoReq := createTodoReqToDTO(req, userID)

	dtoTodo, err := database.QueryInTransaction(ctx, s.db, func(tx database.Transaction) (*dto.Todo, error) {
		txRepo := s.repo.WithExecutor(tx)

		todo, err := txRepo.Create(ctx, dtoReq)
		if err != nil {
			return nil, err
		}

		if todo.ParentID != nil {
			parent, err := txRepo.GetByID(ctx, *todo.ParentID, userID, false)
			if err != nil {
				return nil, err
			}
			if parent.Status == "completed" {
				_, err = txRepo.Update(ctx, parent.ID, userID, &dto.UpdateTodoRequest{
					UserID:      userID,
					Status:      strPtr("in_progress"),
					CompletedAt: schema.Nullable[*time.Time]{IsExplicitlySet: true, Data: nil},
				})
				if err != nil {
					return nil, err
				}
				logger.Info("parent todo reactivated due to new child", zap.String("parent_id", parent.ID.String()), zap.String("child_id", todo.ID.String()))
			}
		}

		return todo, nil
	})
	if err != nil {
		logger.Error("failed to create todo", zap.String("user_id", userID), zap.String("title", req.Title), zap.Error(err))
		return nil, err
	}

	logger.Info("todo created", zap.String("todo_id", dtoTodo.ID.String()), zap.String("title", dtoTodo.Title))
	return dtoTodoToSchema(dtoTodo), nil
}


func (s *TodoService) GetByID(
	ctx context.Context,
	todoID uuid.UUID,
	userID string,
	includeDeleted bool,
) (*schema.Todo, error) {
	var (
		dtoTodo  *dto.Todo
		children []dto.TodoListItems
	)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		dtoTodo, err = s.repo.GetByID(ctx, todoID, userID, includeDeleted)
		return err
	})

	g.Go(func() error {
		var err error
		children, err = s.repo.GetAll(
			ctx,
			&userID,
			&dto.GetTodosQuery{ParentID: &todoID,Limit: 10},
			includeDeleted,
		)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	result := dtoTodoToSchema(dtoTodo)
	result.Children = dtoTodoListItemsToSchema(children)

	return result, nil
}

func (s *TodoService) Update(ctx context.Context, todoID uuid.UUID, userID string, payload *schema.UpdateTodoRequest) (*schema.Todo, error) {
	logger := observability.FromContext(ctx)

	updatedDTO, err := database.QueryInTransaction(ctx, s.db, func(tx database.Transaction) (*dto.Todo, error) {
		txRepo := s.repo.WithExecutor(tx)

		oldTodo, err := txRepo.GetByID(ctx, todoID, userID, false)
		if err != nil {
			return nil, err
		}

		dtoReq := updateTodoReqToDTO(payload, userID)
		updatedDTO, err := txRepo.Update(ctx, todoID, userID, dtoReq)
		if err != nil {
			return nil, err
		}

		statusChanged := payload.Status != nil && *payload.Status != oldTodo.Status
		if statusChanged && updatedDTO.ParentID != nil {
			if err := s.recalculateParentStatus(ctx, txRepo, *updatedDTO.ParentID, userID); err != nil {
				return nil, err
			}
		}

		parentChanged := payload.ParentID.IsSet() && !parentIDsEqual(payload.ParentID.Data, oldTodo.ParentID)
		if parentChanged {
			if oldTodo.ParentID != nil {
				if err := s.recalculateParentStatus(ctx, txRepo, *oldTodo.ParentID, userID); err != nil {
					return nil, err
				}
			}
			if payload.ParentID.Data != nil {
				if err := s.recalculateParentStatus(ctx, txRepo, *payload.ParentID.Data, userID); err != nil {
					return nil, err
				}
			}
		}

		return updatedDTO, nil
	})
	if err != nil {
		logger.Error("failed to update todo", zap.String("todo_id", todoID.String()), zap.Error(err))
		return nil, err
	}

	logger.Info("todo updated", zap.String("todo_id", todoID.String()))
	return dtoTodoToSchema(updatedDTO), nil
}

func (s *TodoService) GetAll(ctx context.Context, userID string, query *schema.GetTodosQuery) (*schema.PaginatedResponse[schema.TodoListItems], error) {
	logger := observability.FromContext(ctx)
	dtoQuery := getTodosQueryToDTO(query)

	var todos []dto.TodoListItems
	var total int

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		todos, err = s.repo.GetAll(gCtx, &userID, dtoQuery, false)
		return err
	})

	g.Go(func() error {
		var err error
		total, err = s.repo.Count(gCtx, &userID, dtoQuery, false)
		return err
	})

	if err := g.Wait(); err != nil {
		logger.Error("failed to fetch todos", zap.Error(err))
		return nil, err
	}

	page := 1
	if query.Page != nil {
		page = *query.Page
	}
	limit := 10
	if query.Limit != nil {
		limit = *query.Limit
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	logger.Info("todos fetched", zap.Int("count", len(todos)), zap.Int("total", total), zap.Int("page", page))
	return &schema.PaginatedResponse[schema.TodoListItems]{
		Data:       dtoTodoListItemsToSchema(todos),
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *TodoService) Delete(ctx context.Context, todoID uuid.UUID, userID string) error {
	logger := observability.FromContext(ctx)

	if err := s.repo.Delete(ctx, todoID, userID); err != nil {
		logger.Error("failed to delete todo", zap.String("todo_id", todoID.String()), zap.Error(err))
		return err
	}

	logger.Info("todo deleted", zap.String("todo_id", todoID.String()))
	return nil
}

func (s *TodoService) recalculateParentStatus(ctx context.Context, repo repository.Todo, parentID uuid.UUID, userID string) error {
	logger := observability.FromContext(ctx)

	children, err := repo.GetAll(ctx, &userID, &dto.GetTodosQuery{ParentID: &parentID}, false)
	if err != nil {
		logger.Error("failed to fetch children for parent status recalculation", zap.String("parent_id", parentID.String()), zap.Error(err))
		return err
	}

	allDone := len(children) > 0
	for _, child := range children {
		if child.Status != "completed" && child.Status != "archived" {
			allDone = false
			break
		}
	}

	if allDone {
		now := time.Now()
		_, err = repo.Update(ctx, parentID, userID, &dto.UpdateTodoRequest{
			UserID:      userID,
			Status:      strPtr("completed"),
			CompletedAt: schema.Nullable[*time.Time]{IsExplicitlySet: true, Data: &now},
		})
		if err != nil {
			return err
		}
		logger.Info("parent todo completed via status sync", zap.String("todo_id", parentID.String()))
		return nil
	}

	parent, err := repo.GetByID(ctx, parentID, userID, false)
	if err != nil {
		return err
	}

	if parent.Status == "completed" {
		_, err = repo.Update(ctx, parentID, userID, &dto.UpdateTodoRequest{
			UserID:      userID,
			Status:      strPtr("in_progress"),
			CompletedAt: schema.Nullable[*time.Time]{IsExplicitlySet: true, Data: nil},
		})
		if err != nil {
			return err
		}
		logger.Info("parent todo reactivated via status sync", zap.String("todo_id", parentID.String()))
	}

	return nil
}

func createTodoReqToDTO(req *schema.CreateTodoRequest, userID string) *dto.CreateTodoRequest {
	return &dto.CreateTodoRequest{
		UserID:      userID,
		CategoryID:  req.CategoryID,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
		ParentID:    req.ParentID,
		Metadata:    req.Metadata,
	}
}

func updateTodoReqToDTO(req *schema.UpdateTodoRequest, userID string) *dto.UpdateTodoRequest {
	return &dto.UpdateTodoRequest{
		UserID:      userID,
		CategoryID:  req.CategoryID,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
		CompletedAt: req.CompletedAt,
		ParentID:    req.ParentID,
		Metadata:    req.Metadata,
	}
}

func getTodosQueryToDTO(q *schema.GetTodosQuery) *dto.GetTodosQuery {
	page := 1
	limit := 10
	if q.Page != nil {
		page = *q.Page
	}
	if q.Limit != nil {
		limit = *q.Limit
	}
	offset := (page - 1) * limit

	return &dto.GetTodosQuery{
		Offset:     offset,
		Limit:      limit,
		Search:     q.Search,
		Status:     q.Status,
		Priority:   q.Priority,
		CategoryID: q.CategoryID,
		ParentID:   q.ParentID,
		OrderBy:    q.OrderBy,
	}
}

func dtoTodoToSchema(d *dto.Todo) *schema.Todo {
	return &schema.Todo{
		ID:          d.ID,
		UserID:      d.UserID,
		CategoryID:  d.CategoryID,
		Title:       d.Title,
		Description: d.Description,
		Status:      schema.TodoStatus(d.Status),
		Priority:    d.Priority,
		DueDate:     d.DueDate,
		CompletedAt: d.CompletedAt,
		ParentID:    d.ParentID,
		Metadata:    d.Metadata,
		IsDeleted:   d.IsDeleted,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

func dtoTodoListItemsToSchema(items []dto.TodoListItems) []schema.TodoListItems {
	result := make([]schema.TodoListItems, len(items))
	for i, item := range items {
		result[i] = schema.TodoListItems{
			ID:         item.ID,
			Title:      item.Title,
			Status:     item.Status,
			Priority:   item.Priority,
			DueDate:    item.DueDate,
			CategoryID: item.CategoryID,
		}
	}
	return result
}

func strPtr(s string) *string {
	return &s
}

func parentIDsEqual(a, b *uuid.UUID) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
