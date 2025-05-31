// Package postgres provides PostgreSQL database client implementation with connection pooling,
// transaction management, and migration support using pgx driver.
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ClientInterface defines the interface for database operations
type ClientInterface interface {
	// Query executes a query that returns rows
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)

	// QueryRow executes a query that returns at most one row
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row

	// Exec executes a query without returning any rows
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)

	// Begin starts a transaction
	Begin(ctx context.Context) (pgx.Tx, error)

	// WithTransaction executes a function within a transaction
	WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error

	// Health checks the health of the database connection
	Health(ctx context.Context) error

	// Stats returns database pool statistics
	Stats() *pgxpool.Stat

	// GetPoolStats returns typed pool statistics
	GetPoolStats() *PoolStats

	// Close closes the database connection
	Close() error
}
