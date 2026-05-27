package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shah-dhwanil/tasker/internal/config"
)

type PgPool = *pgxpool.Pool
type Transaction = pgx.Tx

type DBTX interface{
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

func NewPgPool(config *config.Config) (PgPool, error) {
	pgConfig, err := pgxpool.ParseConfig(config.Postgres.DSN)
	if err != nil {
		return nil, err
	}
	pgConfig.MaxConnIdleTime = time.Duration(config.Postgres.ConnMaxIdleTime) * time.Second
	pgConfig.MaxConnLifetime = time.Duration(config.Postgres.ConnMaxLifetime) * time.Second
	pgConfig.MaxConns = int32(config.Postgres.MaxOpenConns)
	pgConfig.MinIdleConns = int32(config.Postgres.MinIdleConns)
	pgpool, err := pgxpool.NewWithConfig(context.Background(), pgConfig)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Postgres.PingTimeout) * time.Second)
	defer cancel()
	if err = pgpool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("Timeout Occured: Unable to connect to Database: %w", err)
	}
	return pgpool, nil
}