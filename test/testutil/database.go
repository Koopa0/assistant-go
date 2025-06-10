package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// DatabaseContainer wraps a PostgreSQL test container
type DatabaseContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
	Database  string
	Username  string
	Password  string
	URL       string
}

// NewPostgreSQLContainer creates a new PostgreSQL test container with pgvector extension
func NewPostgreSQLContainer(ctx context.Context, t *testing.T) (*DatabaseContainer, error) {
	t.Helper()

	// Create PostgreSQL container with pgvector
	container, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("pgvector/pgvector:pg15"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.WithInitScripts("../fixtures/init.sql"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start PostgreSQL container: %w", err)
	}

	// Get connection details
	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	dbContainer := &DatabaseContainer{
		Container: container,
		Host:      host,
		Port:      mappedPort.Port(),
		Database:  "testdb",
		Username:  "testuser",
		Password:  "testpass",
	}

	dbContainer.URL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbContainer.Username,
		dbContainer.Password,
		dbContainer.Host,
		dbContainer.Port,
		dbContainer.Database,
	)

	return dbContainer, nil
}

// GetConnectionPool returns a pgx connection pool for testing
func (dc *DatabaseContainer) GetConnectionPool(ctx context.Context) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dc.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Test-specific pool configuration
	config.MaxConns = 5
	config.MinConns = 1
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = time.Minute * 30

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// GetStandardConnection returns a standard database/sql connection for testing
func (dc *DatabaseContainer) GetStandardConnection() (*sql.DB, error) {
	db, err := sql.Open("postgres", dc.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// Close terminates the container
func (dc *DatabaseContainer) Close(ctx context.Context) error {
	if dc.Container != nil {
		return dc.Container.Terminate(ctx)
	}
	return nil
}

// SetupTestDatabase creates a test database container and returns cleanup function
func SetupTestDatabase(t *testing.T) (*DatabaseContainer, func()) {
	t.Helper()

	ctx := context.Background()
	container, err := NewPostgreSQLContainer(ctx, t)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	cleanup := func() {
		if err := container.Close(ctx); err != nil {
			t.Logf("Failed to cleanup test database: %v", err)
		}
	}

	return container, cleanup
}

// RunMigrations runs database migrations for testing
func (dc *DatabaseContainer) RunMigrations(ctx context.Context, migrationsPath string) error {
	// This would integrate with your migration system
	// For now, we'll assume migrations are handled by init scripts
	return nil
}

// TruncateAllTables truncates all tables for test cleanup
func (dc *DatabaseContainer) TruncateAllTables(ctx context.Context) error {
	pool, err := dc.GetConnectionPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Get all table names
	rows, err := pool.Query(ctx, `
		SELECT tablename 
		FROM pg_tables 
		WHERE schemaname = 'public' 
		AND tablename NOT LIKE 'pg_%'
		AND tablename != 'schema_migrations'
	`)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	// Truncate all tables
	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	return nil
}

// WaitForHealthy waits for the database to be healthy
func (dc *DatabaseContainer) WaitForHealthy(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for database to be healthy: %w", ctx.Err())
		case <-ticker.C:
			pool, err := dc.GetConnectionPool(ctx)
			if err != nil {
				continue
			}

			if err := pool.Ping(ctx); err != nil {
				pool.Close()
				continue
			}

			pool.Close()
			return nil
		}
	}
}

// CreateTestLogger creates a logger suitable for testing
func CreateTestLogger(level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewTextHandler(testWriter{}, opts)
	return slog.New(handler)
}

// testWriter implements io.Writer for test logging
type testWriter struct{}

func (tw testWriter) Write(p []byte) (n int, err error) {
	// In tests, we might want to capture or suppress logs
	// For now, just return success without writing
	return len(p), nil
}
