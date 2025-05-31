package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MockClient is a mock implementation of the Client interface for testing
type MockClient struct {
	logger *slog.Logger
}

// NewMockClient creates a new mock database client
func NewMockClient(logger *slog.Logger) *MockClient {
	return &MockClient{
		logger: logger,
	}
}

// Query implements the database query method
func (m *MockClient) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	m.logger.Debug("Mock query executed", slog.String("sql", sql))
	return &mockRows{}, nil
}

// QueryRow implements the database query row method
func (m *MockClient) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	m.logger.Debug("Mock query row executed", slog.String("sql", sql))
	return &mockRow{}
}

// Exec implements the database exec method
func (m *MockClient) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	m.logger.Debug("Mock exec executed", slog.String("sql", sql))
	return pgconn.NewCommandTag(""), nil
}

// Begin implements the database transaction begin method
func (m *MockClient) Begin(ctx context.Context) (pgx.Tx, error) {
	m.logger.Debug("Mock transaction begun")
	return &mockTx{client: m}, nil
}

// WithTransaction implements transaction handling
func (m *MockClient) WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := m.Begin(ctx)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

// Health checks the health of the mock database
func (m *MockClient) Health(ctx context.Context) error {
	m.logger.Debug("Mock health check")
	return nil
}

// Stats returns mock database statistics
func (m *MockClient) Stats() *pgxpool.Stat {
	// Return nil for mock implementation to indicate demo mode
	return nil
}

// GetPoolStats returns typed pool statistics
func (m *MockClient) GetPoolStats() *PoolStats {
	// Return mock statistics for demo mode
	return &PoolStats{
		AcquireCount:            100,
		AcquireDuration:         5 * time.Second,
		AcquiredConns:           2,
		CanceledAcquireCount:    0,
		ConstructingConns:       0,
		EmptyAcquireCount:       0,
		EmptyAcquireWaitTime:    0,
		IdleConns:               3,
		MaxConns:                10,
		MaxIdleDestroyCount:     0,
		MaxLifetimeDestroyCount: 0,
		NewConnsCount:           5,
		TotalConns:              5,
	}
}

// Close closes the mock database connection
func (m *MockClient) Close() error {
	m.logger.Debug("Mock database connection closed")
	return nil
}

// Migrate runs mock database migrations
func (m *MockClient) Migrate(ctx context.Context) error {
	m.logger.Info("Mock database migrations completed")
	return nil
}

// mockRows implements pgx.Rows interface
type mockRows struct {
	pgx.Rows
	closed bool
	index  int
}

func (r *mockRows) Close() {
	r.closed = true
}

func (r *mockRows) Next() bool {
	if r.index > 2 { // Return 3 mock rows
		return false
	}
	r.index++
	return true
}

func (r *mockRows) Scan(dest ...interface{}) error {
	// Mock scan implementation
	for i, d := range dest {
		switch v := d.(type) {
		case *string:
			*v = fmt.Sprintf("mock-value-%d", i)
		case *int:
			*v = i
		case *time.Time:
			*v = time.Now()
		case *bool:
			*v = true
		}
	}
	return nil
}

func (r *mockRows) Err() error {
	return nil
}

// mockRow implements pgx.Row interface
type mockRow struct{}

func (r *mockRow) Scan(dest ...interface{}) error {
	// Mock scan implementation
	for i, d := range dest {
		switch v := d.(type) {
		case *string:
			*v = fmt.Sprintf("mock-value-%d", i)
		case *int:
			*v = i
		case *time.Time:
			*v = time.Now()
		case *bool:
			*v = true
		}
	}
	return nil
}

// mockTx implements pgx.Tx interface
type mockTx struct {
	pgx.Tx
	client *MockClient
}

func (t *mockTx) Commit(ctx context.Context) error {
	t.client.logger.Debug("Mock transaction committed")
	return nil
}

func (t *mockTx) Rollback(ctx context.Context) error {
	t.client.logger.Debug("Mock transaction rolled back")
	return nil
}

func (t *mockTx) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	return t.client.Exec(ctx, sql, arguments...)
}

func (t *mockTx) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return t.client.Query(ctx, sql, args...)
}

func (t *mockTx) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return t.client.QueryRow(ctx, sql, args...)
}
