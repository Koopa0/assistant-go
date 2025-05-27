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
)

// Client represents a PostgreSQL client
type Client struct {
	pool   *pgxpool.Pool
	config config.DatabaseConfig
	logger *slog.Logger
}

// NewClient creates a new PostgreSQL client
func NewClient(ctx context.Context, cfg config.DatabaseConfig) (*Client, error) {
	// Parse connection configuration
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = int32(cfg.MaxConnections)
	poolConfig.MinConns = int32(cfg.MinConnections)
	poolConfig.MaxConnIdleTime = cfg.MaxIdleTime
	poolConfig.MaxConnLifetime = cfg.MaxLifetime
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Configure connection settings
	poolConfig.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	poolConfig.ConnConfig.RuntimeParams = map[string]string{
		"application_name": "goassistant",
		"timezone":         "UTC",
	}

	// Enable logging if configured
	if cfg.EnableLogging {
		poolConfig.ConnConfig.Tracer = &QueryTracer{
			logger: slog.Default(),
		}
	}

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	client := &Client{
		pool:   pool,
		config: cfg,
		logger: slog.Default(),
	}

	return client, nil
}

// Close closes the database connection pool
func (c *Client) Close() {
	if c.pool != nil {
		c.pool.Close()
	}
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

// Begin starts a new transaction
func (c *Client) Begin(ctx context.Context) (pgx.Tx, error) {
	return c.pool.Begin(ctx)
}

// BeginTx starts a new transaction with options
func (c *Client) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return c.pool.BeginTx(ctx, txOptions)
}

// Query executes a query that returns rows
func (c *Client) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return c.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns at most one row
func (c *Client) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return c.pool.QueryRow(ctx, sql, args...)
}

// Exec executes a query that doesn't return rows
func (c *Client) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
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
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			c.logger.Error("Failed to rollback transaction",
				slog.Any("rollback_error", rbErr),
				slog.Any("original_error", err))
		}
		return err
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
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			c.logger.Error("Failed to rollback transaction",
				slog.Any("rollback_error", rbErr),
				slog.Any("original_error", err))
		}
		return err
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
