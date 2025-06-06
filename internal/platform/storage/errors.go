package storage

import (
	"errors"
	"fmt"
	"time"
)

// Base database error types
var (
	ErrDatabase     = errors.New("database error")
	ErrConnection   = errors.New("database connection error")
	ErrTimeout      = errors.New("database timeout")
	ErrConstraint   = errors.New("database constraint violation")
	ErrMigration    = errors.New("database migration error")
	ErrTransaction  = errors.New("transaction error")
	ErrNotFound     = errors.New("record not found")
	ErrDuplicateKey = errors.New("duplicate key")
	ErrInvalidQuery = errors.New("invalid query")
)

// DatabaseError represents a database-specific error
type DatabaseError struct {
	Code      string
	Message   string
	Operation string
	Table     string
	Cause     error
	Retryable bool
	Timestamp time.Time
}

// Error implements the error interface
func (e *DatabaseError) Error() string {
	return fmt.Sprintf("database error [%s] %s: %s (table: %s, retryable: %v)",
		e.Code, e.Operation, e.Message, e.Table, e.Retryable)
}

// Unwrap returns the underlying error
func (e *DatabaseError) Unwrap() error {
	return e.Cause
}

// NewConnectionError creates a database connection error
func NewConnectionError(host string, cause error) *DatabaseError {
	return &DatabaseError{
		Code:      "DB_CONNECTION",
		Message:   fmt.Sprintf("failed to connect to database at %s", host),
		Operation: "connect",
		Cause:     cause,
		Retryable: true,
		Timestamp: time.Now(),
	}
}

// NewTimeoutError creates a database timeout error
func NewTimeoutError(operation string, timeout time.Duration, cause error) *DatabaseError {
	return &DatabaseError{
		Code:      "DB_TIMEOUT",
		Message:   fmt.Sprintf("operation timed out after %v", timeout),
		Operation: operation,
		Cause:     cause,
		Retryable: true,
		Timestamp: time.Now(),
	}
}

// NewConstraintError creates a database constraint violation error
func NewConstraintError(constraint string, table string, cause error) *DatabaseError {
	return &DatabaseError{
		Code:      "DB_CONSTRAINT",
		Message:   fmt.Sprintf("constraint violation: %s", constraint),
		Operation: "constraint_check",
		Table:     table,
		Cause:     cause,
		Retryable: false,
		Timestamp: time.Now(),
	}
}

// NewNotFoundError creates a record not found error
func NewNotFoundError(table string, id interface{}) *DatabaseError {
	return &DatabaseError{
		Code:      "DB_NOT_FOUND",
		Message:   fmt.Sprintf("record not found: %v", id),
		Operation: "select",
		Table:     table,
		Cause:     ErrNotFound,
		Retryable: false,
		Timestamp: time.Now(),
	}
}

// NewDuplicateKeyError creates a duplicate key error
func NewDuplicateKeyError(table string, key string, cause error) *DatabaseError {
	return &DatabaseError{
		Code:      "DB_DUPLICATE",
		Message:   fmt.Sprintf("duplicate key: %s", key),
		Operation: "insert",
		Table:     table,
		Cause:     cause,
		Retryable: false,
		Timestamp: time.Now(),
	}
}

// NewTransactionError creates a transaction error
func NewTransactionError(operation string, cause error) *DatabaseError {
	return &DatabaseError{
		Code:      "DB_TRANSACTION",
		Message:   fmt.Sprintf("transaction failed during %s", operation),
		Operation: operation,
		Cause:     cause,
		Retryable: true,
		Timestamp: time.Now(),
	}
}

// NewMigrationError creates a migration error
func NewMigrationError(version string, direction string, cause error) *DatabaseError {
	return &DatabaseError{
		Code:      "DB_MIGRATION",
		Message:   fmt.Sprintf("migration %s failed (%s)", version, direction),
		Operation: "migrate",
		Cause:     cause,
		Retryable: false,
		Timestamp: time.Now(),
	}
}
