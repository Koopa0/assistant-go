package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Migration represents a database migration
type Migration struct {
	Version   int
	Name      string
	UpSQL     string
	DownSQL   string
	AppliedAt *time.Time
	Checksum  string
}

// Migrator handles database migrations
type Migrator struct {
	client *Client
	logger *slog.Logger
	path   string
}

// NewMigrator creates a new migrator
func NewMigrator(client *Client, migrationsPath string) *Migrator {
	return &Migrator{
		client: client,
		logger: slog.Default(),
		path:   migrationsPath,
	}
}

// Migrate runs all pending migrations
func (c *Client) Migrate(ctx context.Context) error {
	migrator := NewMigrator(c, c.config.MigrationsPath)
	return migrator.Up(ctx)
}

// Up runs all pending migrations
func (m *Migrator) Up(ctx context.Context) error {
	// Ensure migrations table exists
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	// Load migrations from filesystem
	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Find pending migrations
	pending := m.findPendingMigrations(migrations, applied)
	if len(pending) == 0 {
		m.logger.Info("No pending migrations")
		return nil
	}

	m.logger.Info("Running migrations", slog.Int("count", len(pending)))

	// Apply pending migrations
	for _, migration := range pending {
		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}
		m.logger.Info("Applied migration",
			slog.Int("version", migration.Version),
			slog.String("name", migration.Name))
	}

	m.logger.Info("All migrations completed successfully")
	return nil
}

// Down rolls back the last migration
func (m *Migrator) Down(ctx context.Context) error {
	// Get the last applied migration
	lastMigration, err := m.getLastAppliedMigration(ctx)
	if err != nil {
		return fmt.Errorf("failed to get last applied migration: %w", err)
	}

	if lastMigration == nil {
		m.logger.Info("No migrations to rollback")
		return nil
	}

	// Load the migration file
	migration, err := m.loadMigration(lastMigration.Version, lastMigration.Name)
	if err != nil {
		return fmt.Errorf("failed to load migration %d: %w", lastMigration.Version, err)
	}

	// Apply the down migration
	if err := m.rollbackMigration(ctx, *migration); err != nil {
		return fmt.Errorf("failed to rollback migration %d: %w", migration.Version, err)
	}

	m.logger.Info("Rolled back migration",
		slog.Int("version", migration.Version),
		slog.String("name", migration.Name))

	return nil
}

// Status returns the current migration status
func (m *Migrator) Status(ctx context.Context) ([]Migration, error) {
	// Ensure migrations table exists
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	// Load migrations from filesystem
	migrations, err := m.loadMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Merge filesystem and database information
	status := make([]Migration, 0, len(migrations))
	appliedMap := make(map[int]*Migration)
	for _, a := range applied {
		appliedMap[a.Version] = &a
	}

	for _, migration := range migrations {
		if applied, exists := appliedMap[migration.Version]; exists {
			migration.AppliedAt = applied.AppliedAt
		}
		status = append(status, migration)
	}

	return status, nil
}

// ensureMigrationsTable creates the migrations table if it doesn't exist
func (m *Migrator) ensureMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			checksum TEXT NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)
	`
	_, err := m.client.Exec(ctx, query)
	return err
}

// loadMigrations loads all migrations from the filesystem
func (m *Migrator) loadMigrations() ([]Migration, error) {
	if m.path == "" {
		return []Migration{}, nil
	}

	var migrations []Migration

	err := filepath.WalkDir(m.path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".up.sql") {
			return nil
		}

		migration, err := m.parseMigrationFile(path)
		if err != nil {
			return fmt.Errorf("failed to parse migration file %s: %w", path, err)
		}

		migrations = append(migrations, *migration)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationFile parses a migration file
func (m *Migrator) parseMigrationFile(upPath string) (*Migration, error) {
	// Extract version and name from filename
	filename := filepath.Base(upPath)
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid migration filename format: %s", filename)
	}

	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid version in filename: %s", parts[0])
	}

	name := strings.TrimSuffix(parts[1], ".up.sql")

	// Read up migration
	upSQL, err := os.ReadFile(upPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read up migration: %w", err)
	}

	// Read down migration
	downPath := strings.Replace(upPath, ".up.sql", ".down.sql", 1)
	var downSQL []byte
	if _, err := os.Stat(downPath); err == nil {
		downSQL, err = os.ReadFile(downPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read down migration: %w", err)
		}
	}

	return &Migration{
		Version: version,
		Name:    name,
		UpSQL:   string(upSQL),
		DownSQL: string(downSQL),
	}, nil
}

// loadMigration loads a specific migration by version and name
func (m *Migrator) loadMigration(version int, name string) (*Migration, error) {
	upPath := filepath.Join(m.path, fmt.Sprintf("%d_%s.up.sql", version, name))
	return m.parseMigrationFile(upPath)
}

// getAppliedMigrations returns all applied migrations
func (m *Migrator) getAppliedMigrations(ctx context.Context) ([]Migration, error) {
	query := `
		SELECT version, name, checksum, applied_at
		FROM schema_migrations
		ORDER BY version
	`

	rows, err := m.client.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var migration Migration
		err := rows.Scan(&migration.Version, &migration.Name, &migration.Checksum, &migration.AppliedAt)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, migration)
	}

	return migrations, rows.Err()
}

// getLastAppliedMigration returns the last applied migration
func (m *Migrator) getLastAppliedMigration(ctx context.Context) (*Migration, error) {
	query := `
		SELECT version, name, checksum, applied_at
		FROM schema_migrations
		ORDER BY version DESC
		LIMIT 1
	`

	var migration Migration
	err := m.client.QueryRow(ctx, query).Scan(&migration.Version, &migration.Name, &migration.Checksum, &migration.AppliedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &migration, nil
}

// findPendingMigrations finds migrations that haven't been applied
func (m *Migrator) findPendingMigrations(all []Migration, applied []Migration) []Migration {
	appliedMap := make(map[int]bool)
	for _, a := range applied {
		appliedMap[a.Version] = true
	}

	var pending []Migration
	for _, migration := range all {
		if !appliedMap[migration.Version] {
			pending = append(pending, migration)
		}
	}

	return pending
}

// applyMigration applies a single migration
func (m *Migrator) applyMigration(ctx context.Context, migration Migration) error {
	return m.client.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Execute the migration SQL
		_, err := tx.Exec(ctx, migration.UpSQL)
		if err != nil {
			return fmt.Errorf("failed to execute migration SQL: %w", err)
		}

		// Record the migration
		_, err = tx.Exec(ctx,
			"INSERT INTO schema_migrations (version, name, checksum) VALUES ($1, $2, $3)",
			migration.Version, migration.Name, migration.Checksum)
		if err != nil {
			return fmt.Errorf("failed to record migration: %w", err)
		}

		return nil
	})
}

// rollbackMigration rolls back a single migration
func (m *Migrator) rollbackMigration(ctx context.Context, migration Migration) error {
	if migration.DownSQL == "" {
		return fmt.Errorf("no down migration available for version %d", migration.Version)
	}

	return m.client.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Execute the down migration SQL
		_, err := tx.Exec(ctx, migration.DownSQL)
		if err != nil {
			return fmt.Errorf("failed to execute down migration SQL: %w", err)
		}

		// Remove the migration record
		_, err = tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", migration.Version)
		if err != nil {
			return fmt.Errorf("failed to remove migration record: %w", err)
		}

		return nil
	})
}
