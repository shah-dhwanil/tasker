package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/shah-dhwanil/tasker/internal/config"
	"github.com/shah-dhwanil/tasker/internal/observability"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5"
	tern "github.com/jackc/tern/v2/migrate"
)

//go:embed pg_migrations/*.sql
var migrations embed.FS

func Migrate(ctx context.Context, config *config.Config, logger observability.Logger) error {

	conn, err := pgx.Connect(ctx, config.Postgres.DSN)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	m, err := tern.NewMigrator(ctx, conn, "schema_version")
	if err != nil {
		return fmt.Errorf("constructing database migrator: %w", err)
	}
	subtree, err := fs.Sub(migrations, "pg_migrations")
	if err != nil {
		return fmt.Errorf("retrieving database migrations subtree: %w", err)
	}
	if err := m.LoadMigrations(subtree); err != nil {
		return fmt.Errorf("loading database migrations: %w", err)
	}
	from, err := m.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("retreiving current database migration version")
	}
	if err := m.Migrate(ctx); err != nil {
		return err
	}
	if from == int32(len(m.Migrations)) {
		logger.Info("Database uptodate.",zap.Int32("version",from))
	} else {
		logger.Info("Database migrated successfully.",zap.Int32("from_version",from),zap.Int32("to_version",int32(len(m.Migrations))))
	}
	return nil
}