package testing

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/shah-dhwanil/tasker/internal/database"
)

// TxFn represents a function that executes within a transaction
type TxFn func(tx pgx.Tx) 

// WithTransaction runs a function within a transaction and rolls it back afterward
func WithTransaction(ctx context.Context, db database.PgPool, fn TxFn) error {
	// Begin transaction
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure rollback happens if commit doesn't occur
	defer tx.Rollback(ctx)

	// Run the function within the transaction
	fn(tx)

	// Transaction was successful, commit it
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithRollbackTransaction runs a function within a transaction and always rolls it back
// Useful for tests where you want to execute operations but never persist them
func WithRollbackTransaction(ctx context.Context, db database.PgPool, fn TxFn) error {
	// Begin transaction
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Always rollback at the end
	defer tx.Rollback(ctx)

	// Run the function within the transaction
	fn(tx)
	return nil
}