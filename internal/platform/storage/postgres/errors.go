// Package postgres provides domain-specific error handling for PostgreSQL operations
// following CLAUDE.md best practices with proper error hierarchy and context.
package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/koopa0/assistant-go/internal/errors"
)

// PostgreSQL-specific error codes
const (
	// Connection errors
	CodeConnectionFailed        = "DB_CONNECTION_FAILED"
	CodeConnectionTimeout       = "DB_CONNECTION_TIMEOUT"
	CodeConnectionPoolExhausted = "DB_CONNECTION_POOL_EXHAUSTED"
	CodeConnectionLost          = "DB_CONNECTION_LOST"

	// Transaction errors
	CodeTransactionBegin    = "DB_TRANSACTION_BEGIN"
	CodeTransactionCommit   = "DB_TRANSACTION_COMMIT"
	CodeTransactionRollback = "DB_TRANSACTION_ROLLBACK"
	CodeTransactionDeadlock = "DB_TRANSACTION_DEADLOCK"
	CodeTransactionConflict = "DB_TRANSACTION_CONFLICT"

	// Query errors
	CodeQueryExecution = "DB_QUERY_EXECUTION"
	CodeQueryTimeout   = "DB_QUERY_TIMEOUT"
	CodeQuerySyntax    = "DB_QUERY_SYNTAX"
	CodeQueryParameter = "DB_QUERY_PARAMETER"
	CodeQueryResult    = "DB_QUERY_RESULT"

	// Schema errors
	CodeTableNotFound       = "DB_TABLE_NOT_FOUND"
	CodeColumnNotFound      = "DB_COLUMN_NOT_FOUND"
	CodeConstraintViolation = "DB_CONSTRAINT_VIOLATION"
	CodeUniqueViolation     = "DB_UNIQUE_VIOLATION"
	CodeForeignKeyViolation = "DB_FOREIGN_KEY_VIOLATION"
	CodeCheckViolation      = "DB_CHECK_VIOLATION"

	// Migration errors
	CodeMigrationFailed  = "DB_MIGRATION_FAILED"
	CodeMigrationVersion = "DB_MIGRATION_VERSION"
	CodeMigrationLock    = "DB_MIGRATION_LOCK"

	// Data errors
	CodeDataTruncation = "DB_DATA_TRUNCATION"
	CodeDataType       = "DB_DATA_TYPE"
	CodeDataEncoding   = "DB_DATA_ENCODING"
	CodeDataIntegrity  = "DB_DATA_INTEGRITY"
)

// PostgreSQL error class constants (first two characters of SQLSTATE)
const (
	SQLStateClassConnectionException          = "08"
	SQLStateClassTriggeredDataChangeViolation = "27"
	SQLStateClassInvalidTransactionState      = "25"
	SQLStateClassIntegrityConstraintViolation = "23"
	SQLStateClassTransactionRollback          = "40"
	SQLStateClassSyntaxError                  = "42"
	SQLStateClassInsufficientPrivilege        = "42501"
)

// Connection Error Constructors

// NewConnectionFailedError creates a connection failed error
func NewConnectionFailedError(host, database string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeConnectionFailed, "database connection failed", cause).
		WithComponent("database").
		WithOperation("connect").
		WithContext("host", host).
		WithContext("database", database).
		WithUserMessage("Database connection failed. Please try again.").
		WithActions("Check database server status", "Verify connection parameters", "Check network connectivity").
		WithRetryAfter(time.Second * 5)
}

// NewConnectionTimeoutError creates a connection timeout error
func NewConnectionTimeoutError(host string, timeout time.Duration, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeConnectionTimeout, "database connection timed out", cause).
		WithComponent("database").
		WithOperation("connect").
		WithContext("host", host).
		WithContext("timeout", timeout.String()).
		WithDuration(timeout).
		WithUserMessage("Database connection timed out. Please try again.").
		WithActions("Check network latency", "Increase connection timeout", "Verify database load").
		WithRetryAfter(time.Second * 10)
}

// NewConnectionPoolExhaustedError creates a connection pool exhausted error
func NewConnectionPoolExhaustedError(maxConnections, activeConnections int) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeConnectionPoolExhausted, "database connection pool exhausted", nil).
		WithComponent("database").
		WithOperation("acquire_connection").
		WithContext("max_connections", maxConnections).
		WithContext("active_connections", activeConnections).
		WithUserMessage("Database is temporarily busy. Please try again.").
		WithActions("Wait and retry", "Increase connection pool size", "Check for connection leaks").
		WithRetryAfter(time.Second * 2)
}

// Transaction Error Constructors

// NewTransactionBeginError creates a transaction begin error
func NewTransactionBeginError(cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeTransactionBegin, "failed to begin transaction", cause).
		WithComponent("database").
		WithOperation("begin_transaction").
		WithUserMessage("Database transaction failed to start. Please try again.").
		WithActions("Check database health", "Retry operation", "Check transaction isolation").
		WithRetryable(true)
}

// NewTransactionDeadlockError creates a transaction deadlock error
func NewTransactionDeadlockError(query string, cause error) *errors.AssistantError {
	// Truncate query for logging
	truncatedQuery := query
	if len(query) > 200 {
		truncatedQuery = query[:200] + "..."
	}

	return errors.NewInfrastructureError(CodeTransactionDeadlock, "transaction deadlock detected", cause).
		WithComponent("database").
		WithOperation("execute_transaction").
		WithContext("query_preview", truncatedQuery).
		WithUserMessage("Database operation conflicted with another request. Please try again.").
		WithActions("Retry operation", "Reduce transaction scope", "Check for long-running transactions").
		WithRetryAfter(time.Millisecond * 100)
}

// Query Error Constructors

// NewQueryExecutionError creates a query execution error
func NewQueryExecutionError(query string, cause error) *errors.AssistantError {
	// Truncate query for logging
	truncatedQuery := query
	if len(query) > 200 {
		truncatedQuery = query[:200] + "..."
	}

	return errors.NewBusinessError(CodeQueryExecution, "query execution failed", cause).
		WithComponent("database").
		WithOperation("execute_query").
		WithContext("query_preview", truncatedQuery).
		WithUserMessage("Database operation failed. Please try again.").
		WithActions("Check query syntax", "Verify parameters", "Check database constraints")
}

// NewQueryTimeoutError creates a query timeout error
func NewQueryTimeoutError(query string, timeout time.Duration, cause error) *errors.AssistantError {
	truncatedQuery := query
	if len(query) > 200 {
		truncatedQuery = query[:200] + "..."
	}

	return errors.NewInfrastructureError(CodeQueryTimeout, "query execution timed out", cause).
		WithComponent("database").
		WithOperation("execute_query").
		WithContext("query_preview", truncatedQuery).
		WithContext("timeout", timeout.String()).
		WithDuration(timeout).
		WithUserMessage("Database operation timed out. Please try again.").
		WithActions("Optimize query", "Increase timeout", "Check database performance").
		WithRetryAfter(time.Second * 5)
}

// NewQuerySyntaxError creates a query syntax error
func NewQuerySyntaxError(query string, position int, cause error) *errors.AssistantError {
	truncatedQuery := query
	if len(query) > 200 {
		truncatedQuery = query[:200] + "..."
	}

	return errors.NewValidationError(CodeQuerySyntax, "SQL syntax error", cause).
		WithComponent("database").
		WithOperation("parse_query").
		WithContext("query_preview", truncatedQuery).
		WithContext("error_position", position).
		WithUserMessage("Database query syntax error.").
		WithActions("Check SQL syntax", "Verify column names", "Review query structure")
}

// Constraint Error Constructors

// NewUniqueViolationError creates a unique constraint violation error
func NewUniqueViolationError(table, constraint string, value interface{}, cause error) *errors.AssistantError {
	return errors.NewValidationError(CodeUniqueViolation, "unique constraint violation", cause).
		WithComponent("database").
		WithOperation("insert_or_update").
		WithContext("table", table).
		WithContext("constraint", constraint).
		WithContext("value", value).
		WithUserMessage("The provided value already exists and must be unique.").
		WithActions("Use different value", "Update existing record", "Check for duplicates")
}

// NewForeignKeyViolationError creates a foreign key violation error
func NewForeignKeyViolationError(table, constraint, referencedTable string, cause error) *errors.AssistantError {
	return errors.NewValidationError(CodeForeignKeyViolation, "foreign key constraint violation", cause).
		WithComponent("database").
		WithOperation("insert_or_update").
		WithContext("table", table).
		WithContext("constraint", constraint).
		WithContext("referenced_table", referencedTable).
		WithUserMessage("Referenced record does not exist.").
		WithActions("Create referenced record first", "Use existing reference", "Check foreign key values")
}

// NewCheckViolationError creates a check constraint violation error
func NewCheckViolationError(table, constraint string, value interface{}, cause error) *errors.AssistantError {
	return errors.NewValidationError(CodeCheckViolation, "check constraint violation", cause).
		WithComponent("database").
		WithOperation("insert_or_update").
		WithContext("table", table).
		WithContext("constraint", constraint).
		WithContext("value", value).
		WithUserMessage("Data violates database constraints.").
		WithActions("Adjust data values", "Check constraint rules", "Validate input data")
}

// Migration Error Constructors

// NewMigrationFailedError creates a migration failed error
func NewMigrationFailedError(version string, direction string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeMigrationFailed, "database migration failed", cause).
		WithComponent("database").
		WithOperation("migrate").
		WithContext("version", version).
		WithContext("direction", direction).
		WithUserMessage("Database migration failed.").
		WithActions("Check migration scripts", "Verify database state", "Review migration logs").
		WithSeverity(errors.SeverityCritical)
}

// PostgreSQL Error Analysis

// AnalyzePgError analyzes a PostgreSQL error and creates an appropriate AssistantError
func AnalyzePgError(err error, operation string, query string) *errors.AssistantError {
	if err == nil {
		return nil
	}

	// Handle pgx specific errors
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return analyzePgConnError(pgErr, operation, query)
	}

	// Handle pgx context errors
	if err == context.DeadlineExceeded {
		return NewQueryTimeoutError(query, 0, err)
	}

	// Handle connection errors
	if err == pgx.ErrNoRows {
		return errors.NewBusinessError("DB_NO_ROWS", "no rows found", err).
			WithComponent("database").
			WithOperation(operation).
			WithUserMessage("No matching records found.")
	}

	// Handle pool errors
	if strings.Contains(err.Error(), "failed to connect") {
		return NewConnectionFailedError("unknown", "unknown", err)
	}

	// Generic database error
	return errors.NewInfrastructureError("DB_GENERIC_ERROR", "database operation failed", err).
		WithComponent("database").
		WithOperation(operation).
		WithUserMessage("Database operation failed. Please try again.")
}

// analyzePgConnError analyzes PostgreSQL connection errors
func analyzePgConnError(pgErr *pgconn.PgError, operation string, query string) *errors.AssistantError {
	sqlState := pgErr.Code

	// Determine error category and create appropriate error
	switch {
	case strings.HasPrefix(sqlState, SQLStateClassConnectionException):
		return handleConnectionError(pgErr, operation)
	case strings.HasPrefix(sqlState, SQLStateClassIntegrityConstraintViolation):
		return handleConstraintError(pgErr, operation)
	case strings.HasPrefix(sqlState, SQLStateClassTransactionRollback):
		return handleTransactionError(pgErr, operation, query)
	case strings.HasPrefix(sqlState, SQLStateClassSyntaxError):
		return handleSyntaxError(pgErr, operation, query)
	default:
		return handleGenericError(pgErr, operation, query)
	}
}

// handleConnectionError handles connection-related PostgreSQL errors
func handleConnectionError(pgErr *pgconn.PgError, operation string) *errors.AssistantError {
	return NewConnectionFailedError("unknown", "unknown", pgErr).
		WithContext("sql_state", pgErr.Code).
		WithContext("pg_message", pgErr.Message).
		WithContext("pg_detail", pgErr.Detail)
}

// handleConstraintError handles constraint violation errors
func handleConstraintError(pgErr *pgconn.PgError, operation string) *errors.AssistantError {
	constraintName := pgErr.ConstraintName
	tableName := pgErr.TableName

	switch pgErr.Code {
	case "23505": // unique_violation
		return NewUniqueViolationError(tableName, constraintName, nil, pgErr)
	case "23503": // foreign_key_violation
		return NewForeignKeyViolationError(tableName, constraintName, "", pgErr)
	case "23514": // check_violation
		return NewCheckViolationError(tableName, constraintName, nil, pgErr)
	default:
		return errors.NewValidationError(CodeConstraintViolation, "constraint violation", pgErr).
			WithComponent("database").
			WithOperation(operation).
			WithContext("constraint", constraintName).
			WithContext("table", tableName)
	}
}

// handleTransactionError handles transaction-related errors
func handleTransactionError(pgErr *pgconn.PgError, operation string, query string) *errors.AssistantError {
	if pgErr.Code == "40P01" { // deadlock_detected
		return NewTransactionDeadlockError(query, pgErr)
	}

	return errors.NewInfrastructureError(CodeTransactionConflict, "transaction conflict", pgErr).
		WithComponent("database").
		WithOperation(operation).
		WithContext("sql_state", pgErr.Code).
		WithContext("pg_message", pgErr.Message)
}

// handleSyntaxError handles SQL syntax errors
func handleSyntaxError(pgErr *pgconn.PgError, operation string, query string) *errors.AssistantError {
	position := 0
	if pgErr.Position != 0 {
		position = int(pgErr.Position)
	}

	return NewQuerySyntaxError(query, position, pgErr).
		WithContext("pg_hint", pgErr.Hint).
		WithContext("pg_detail", pgErr.Detail)
}

// handleGenericError handles other PostgreSQL errors
func handleGenericError(pgErr *pgconn.PgError, operation string, query string) *errors.AssistantError {
	return NewQueryExecutionError(query, pgErr).
		WithContext("sql_state", pgErr.Code).
		WithContext("severity", pgErr.Severity).
		WithContext("pg_message", pgErr.Message).
		WithContext("pg_detail", pgErr.Detail).
		WithContext("pg_hint", pgErr.Hint)
}

// Database-specific error helpers

// IsPostgresError checks if an error is a PostgreSQL error
func IsPostgresError(err error) bool {
	_, ok := err.(*pgconn.PgError)
	return ok
}

// IsConnectionError checks if an error is connection-related
func IsConnectionError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeConnectionFailed, CodeConnectionTimeout,
			CodeConnectionPoolExhausted, CodeConnectionLost:
			return true
		}
	}

	// Also check for pgx connection errors
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return strings.HasPrefix(pgErr.Code, SQLStateClassConnectionException)
	}

	return false
}

// IsRetryableError checks if a database error is retryable
func IsRetryableError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		return assistantErr.Retryable
	}

	// Check for specific PostgreSQL retryable errors
	if pgErr, ok := err.(*pgconn.PgError); ok {
		switch pgErr.Code {
		case "40P01": // deadlock_detected
			return true
		case "53300": // too_many_connections
			return true
		}
		// Connection errors are generally retryable
		return strings.HasPrefix(pgErr.Code, SQLStateClassConnectionException)
	}

	return false
}

// IsConstraintError checks if an error is a constraint violation
func IsConstraintError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeConstraintViolation, CodeUniqueViolation,
			CodeForeignKeyViolation, CodeCheckViolation:
			return true
		}
	}

	if pgErr, ok := err.(*pgconn.PgError); ok {
		return strings.HasPrefix(pgErr.Code, SQLStateClassIntegrityConstraintViolation)
	}

	return false
}

// GetConstraintInfo extracts constraint information from database errors
func GetConstraintInfo(err error) (table, constraint string) {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		if t, ok := assistantErr.Context["table"].(string); ok {
			table = t
		}
		if c, ok := assistantErr.Context["constraint"].(string); ok {
			constraint = c
		}
	}

	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.TableName, pgErr.ConstraintName
	}

	return "", ""
}
