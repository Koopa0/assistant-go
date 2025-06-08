package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// TxManager provides transaction management for database operations
// Following Go pattern: explicit transaction management
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager creates a new transaction manager
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// WithTx executes a function within a database transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func (tm *TxManager) WithTx(ctx context.Context, fn func(sqlc.Querier) error) error {
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Create queries with transaction
	queries := sqlc.New(tx)

	// Execute the function
	if err := fn(queries); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback failed: %v, original error: %w", rbErr, err)
		}
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// QuerierWithTx creates a Querier that uses the given transaction
// This is useful when you need to pass a transaction-aware Querier to repositories
func QuerierWithTx(tx pgx.Tx) sqlc.Querier {
	return sqlc.New(tx)
}
