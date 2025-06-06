// Package postgres provides PostgreSQL database client implementation with connection pooling,
// transaction management, and migration support using pgx driver.
package postgres

import (
	"context"
	"io"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// DB is the main database interface combining all database operations
// This replaces the old ClientInterface with proper Go naming
type DB interface {
	Querier
	Executor
	Transactor
	HealthChecker
	StatsProvider
	QueriesProvider
	io.Closer
}

// Querier provides read-only database operations
// Small interface following interface segregation principle
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Executor provides write database operations
type Executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Transactor provides transaction operations
type Transactor interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error
}

// HealthChecker provides health monitoring
type HealthChecker interface {
	Health(ctx context.Context) error
}

// StatsProvider provides connection pool statistics
type StatsProvider interface {
	Stats() *pgxpool.Stat
	GetPoolStats() *PoolStats
	DatabaseInfo(ctx context.Context) (*DatabaseInfo, error)
}

// QueriesProvider provides access to SQLC generated queries
type QueriesProvider interface {
	GetQueries() *sqlc.Queries
}

// DatabaseInfo provides information about the database
type DatabaseInfo struct {
	Version     string `json:"version"`
	Connections struct {
		Active int32 `json:"active"`
		Idle   int32 `json:"idle"`
		Max    int32 `json:"max"`
	} `json:"connections"`
}

// io.Closer is already defined in standard library
// Using it directly follows Go idioms
