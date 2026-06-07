package service_test

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)
type MockTx struct{}

func (m *MockTx) Begin(ctx context.Context) (pgx.Tx, error) { return nil, nil }
func (m *MockTx) Commit(ctx context.Context) error           { return nil }
func (m *MockTx) Rollback(ctx context.Context) error         { return nil }
func (m *MockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *MockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (m *MockTx) LargeObjects() pgx.LargeObjects                                { return pgx.LargeObjects{} }
func (m *MockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (m *MockTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *MockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (m *MockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row { return nil }
func (m *MockTx) Conn() *pgx.Conn                                                { return nil }

type MockDBTX struct{}

func (m *MockDBTX) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *MockDBTX) Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error) {
	return nil, nil
}
func (m *MockDBTX) QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row {
	return nil
}
func (m *MockDBTX) Begin(ctx context.Context) (pgx.Tx, error) {
	return &MockTx{}, nil
}

func ptr[T any](v T) *T {
	return &v
}
