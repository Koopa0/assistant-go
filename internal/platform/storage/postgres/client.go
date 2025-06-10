// Package postgres provides PostgreSQL database connectivity and operations.
// It uses pgx v5 for high-performance database access, sqlc for type-safe queries,
// and includes support for migrations, connection pooling, and transaction management.
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// Client represents a PostgreSQL client
type Client struct {
	pool    *pgxpool.Pool
	config  config.DatabaseConfig
	logger  *slog.Logger
	queries *sqlc.Queries
}

// Ensure Client implements the new interfaces
var (
	_ DB              = (*Client)(nil)
	_ Querier         = (*Client)(nil)
	_ Executor        = (*Client)(nil)
	_ Transactor      = (*Client)(nil)
	_ HealthChecker   = (*Client)(nil)
	_ StatsProvider   = (*Client)(nil)
	_ QueriesProvider = (*Client)(nil)
)

// NewClient creates a new PostgreSQL client
func NewClient(ctx context.Context, cfg config.DatabaseConfig) (*Client, error) {
	// Parse connection configuration
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure connection pool with PostgreSQL 17 optimizations
	// Based on golang_guide.md PostgreSQL 17 best practices
	poolConfig.MaxConns = int32(cfg.MaxConnections)
	poolConfig.MinConns = int32(cfg.MinConnections)
	poolConfig.MaxConnLifetime = cfg.MaxLifetime // Connection rotation for PostgreSQL 17
	poolConfig.MaxConnIdleTime = cfg.MaxIdleTime // Close idle connections
	poolConfig.HealthCheckPeriod = time.Minute   // Regular health checks

	// Configure connection settings
	poolConfig.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	poolConfig.ConnConfig.RuntimeParams = map[string]string{
		"application_name": "goassistant",
		"timezone":         "UTC",
		// PostgreSQL 17 performance optimizations
		"track_io_timing":            "on",   // Enable I/O timing for EXPLAIN ANALYZE
		"log_statement_stats":        "off",  // Use pg_stat_statements instead
		"log_min_duration_statement": "1000", // Log slow queries (1s+)
		// Note: shared_preload_libraries cannot be set at runtime, must be configured in postgresql.conf
	}

	// Enable logging if configured
	if cfg.EnableLogging {
		poolConfig.ConnConfig.Tracer = &QueryTracer{
			logger: slog.Default(),
		}
	}

	// Create a connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger := slog.Default()

	client := &Client{
		pool:    pool,
		config:  cfg,
		logger:  logger,
		queries: sqlc.New(pool),
	}

	return client, nil
}

// Close closes the database connection pool
func (c *Client) Close() error {
	if c.pool != nil {
		c.pool.Close()
	}
	return nil
}

// GetQueries returns the underlying sqlc.Queries for direct access
func (c *Client) GetQueries() *sqlc.Queries {
	return c.queries
}

// Pool returns the underlying connection pool
func (c *Client) Pool() *pgxpool.Pool {
	return c.pool
}

// Ping tests the database connection
func (c *Client) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

// Stats returns connection pool statistics
func (c *Client) Stats() *pgxpool.Stat {
	return c.pool.Stat()
}

// GetPoolStats returns typed pool statistics
func (c *Client) GetPoolStats() *PoolStats {
	return NewPoolStatsFromPgxStat(c.Stats())
}

// Begin starts a new transaction
func (c *Client) Begin(ctx context.Context) (pgx.Tx, error) {
	return c.pool.Begin(ctx)
}

// BeginTx starts a new transaction with options
func (c *Client) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return c.pool.BeginTx(ctx, txOptions)
}

// Query executes a query that returns rows.
// It directly returns errors from the underlying pgxpool.Pool.
func (c *Client) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return c.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns at most one row.
// It directly returns errors from the underlying pgxpool.Pool.
func (c *Client) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return c.pool.QueryRow(ctx, sql, args...)
}

// Exec executes a query that doesn't return rows.
// It directly returns errors from the underlying pgxpool.Pool.
func (c *Client) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return c.pool.Exec(ctx, sql, args...)
}

// CopyFrom efficiently inserts multiple rows using COPY protocol
func (c *Client) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return c.pool.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

// SendBatch sends a batch of queries
func (c *Client) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults {
	return c.pool.SendBatch(ctx, batch)
}

// WithTransaction executes a function within a transaction
func (c *Client) WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := c.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		// Attempt to rollback the transaction if the provided function fails.
		// Log any error during rollback but return the original function's error.
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			c.logger.Error("Failed to rollback transaction",
				slog.Any("rollback_error", rbErr),
				slog.Any("original_error", err)) // Log original error for context
		}
		return err // Return the original error from fn(tx)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithTransactionTx executes a function within a transaction with options
func (c *Client) WithTransactionTx(ctx context.Context, txOptions pgx.TxOptions, fn func(pgx.Tx) error) error {
	tx, err := c.BeginTx(ctx, txOptions)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		// Attempt to rollback the transaction if the provided function fails.
		// Log any error during rollback but return the original function's error.
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			c.logger.Error("Failed to rollback transaction",
				slog.Any("rollback_error", rbErr),
				slog.Any("original_error", err)) // Log original error for context
		}
		return err // Return the original error from fn(tx)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// QueryTracer implements pgx.QueryTracer for logging
type QueryTracer struct {
	logger *slog.Logger
}

// TraceQueryStart is called at the beginning of Query, QueryRow, and Exec calls
func (t *QueryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	t.logger.Debug("Executing query",
		slog.String("sql", data.SQL),
		slog.Any("args", data.Args))
	return ctx
}

// TraceQueryEnd is called at the end of Query, QueryRow, and Exec calls
func (t *QueryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	if data.Err != nil {
		t.logger.Error("Query failed",
			slog.Any("error", data.Err))
	} else {
		t.logger.Debug("Query completed",
			slog.String("command_tag", data.CommandTag.String()))
	}
}

// Health checks the health of the database connection
func (c *Client) Health(ctx context.Context) error {
	// Check if pool is available
	if c.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	// Test connection with a simple query
	var result int
	err := c.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("health check query failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("health check returned unexpected result: %d", result)
	}

	return nil
}

// DatabaseInfo returns information about the database
func (c *Client) DatabaseInfo(ctx context.Context) (*DatabaseInfo, error) {
	var version string
	row := c.pool.QueryRow(ctx, "SELECT version()")
	if err := row.Scan(&version); err != nil {
		return nil, fmt.Errorf("failed to get database version: %w", err)
	}

	stats := c.pool.Stat()

	info := &DatabaseInfo{
		Version: version,
	}
	info.Connections.Active = stats.AcquiredConns()
	info.Connections.Idle = stats.IdleConns()
	info.Connections.Max = stats.MaxConns()

	return info, nil
}

// NewOptimizedClient creates a PostgreSQL client optimized for PostgreSQL 17
// This is a convenience function that creates a client with recommended settings for PostgreSQL 17
func NewOptimizedClient(ctx context.Context, databaseURL string) (*Client, error) {
	// Use optimized default configuration for PostgreSQL 17
	cfg := config.DatabaseConfig{
		URL:            databaseURL,
		MaxConnections: 30,               // Based on CPU cores (golang_guide.md recommendation)
		MinConnections: 5,                // Keep minimum connections warm
		MaxIdleTime:    15 * time.Minute, // Close idle connections
		MaxLifetime:    time.Hour,        // Connection rotation
		ConnectTimeout: 10 * time.Second,
		EnableLogging:  false,
	}

	return NewClient(ctx, cfg)
}
