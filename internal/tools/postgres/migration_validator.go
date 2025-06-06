package postgres

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// MigrationValidator validates PostgreSQL migration files
type MigrationValidator struct {
	logger *slog.Logger
}

// NewMigrationValidator creates a new migration validator
func NewMigrationValidator(logger *slog.Logger) *MigrationValidator {
	return &MigrationValidator{
		logger: logger,
	}
}

// ValidationResult represents the result of migration validation
type ValidationResult struct {
	IsValid       bool              `json:"is_valid"`
	Syntax        SyntaxValidation  `json:"syntax"`
	Safety        SafetyValidation  `json:"safety"`
	BestPractices BestPractices     `json:"best_practices"`
	Reversibility Reversibility     `json:"reversibility"`
	Issues        []ValidationIssue `json:"issues"`
	Warnings      []ValidationIssue `json:"warnings"`
	Suggestions   []string          `json:"suggestions"`
}

// SyntaxValidation contains syntax validation results
type SyntaxValidation struct {
	HasSyntaxErrors bool     `json:"has_syntax_errors"`
	Errors          []string `json:"errors,omitempty"`
	SQLStatements   int      `json:"sql_statements"`
	StatementTypes  []string `json:"statement_types"`
}

// SafetyValidation contains safety check results
type SafetyValidation struct {
	IsSafe            bool     `json:"is_safe"`
	HasDestructiveOps bool     `json:"has_destructive_operations"`
	RequiresDowntime  bool     `json:"requires_downtime"`
	LocksTable        bool     `json:"locks_table"`
	DataLossRisk      bool     `json:"data_loss_risk"`
	SafetyIssues      []string `json:"safety_issues,omitempty"`
}

// BestPractices contains best practice check results
type BestPractices struct {
	FollowsNaming     bool     `json:"follows_naming_conventions"`
	HasTransaction    bool     `json:"has_transaction"`
	HasComments       bool     `json:"has_comments"`
	UsesIfExists      bool     `json:"uses_if_exists_checks"`
	ViolatedPractices []string `json:"violated_practices,omitempty"`
}

// Reversibility contains reversibility check results
type Reversibility struct {
	IsReversible       bool     `json:"is_reversible"`
	HasDownMigration   bool     `json:"has_down_migration"`
	IrreversibleOps    []string `json:"irreversible_operations,omitempty"`
	DownMigrationHints []string `json:"down_migration_hints,omitempty"`
}

// ValidationIssue represents a validation issue
type ValidationIssue struct {
	Type       string `json:"type"`
	Severity   string `json:"severity"` // "error", "warning", "info"
	Message    string `json:"message"`
	Line       int    `json:"line,omitempty"`
	Column     int    `json:"column,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// Validate performs comprehensive validation on a migration SQL
func (v *MigrationValidator) Validate(migration string) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:     true,
		Issues:      []ValidationIssue{},
		Warnings:    []ValidationIssue{},
		Suggestions: []string{},
	}

	// Validate syntax
	v.validateSyntax(migration, result)

	// Check safety
	v.checkSafety(migration, result)

	// Check best practices
	v.checkBestPractices(migration, result)

	// Check reversibility
	v.checkReversibility(migration, result)

	// Add general suggestions
	v.addSuggestions(migration, result)

	// Determine overall validity
	result.IsValid = len(result.Issues) == 0 && !result.Syntax.HasSyntaxErrors

	return result, nil
}

// validateSyntax checks for SQL syntax issues
func (v *MigrationValidator) validateSyntax(migration string, result *ValidationResult) {
	syntax := &SyntaxValidation{
		HasSyntaxErrors: false,
		Errors:          []string{},
		StatementTypes:  []string{},
	}

	// Split into statements (simple split by semicolon)
	statements := v.splitStatements(migration)
	syntax.SQLStatements = len(statements)

	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		// Check for basic syntax patterns
		if !v.isValidStatement(stmt) {
			syntax.HasSyntaxErrors = true
			syntax.Errors = append(syntax.Errors, fmt.Sprintf("Invalid SQL statement at position %d", i+1))
			result.Issues = append(result.Issues, ValidationIssue{
				Type:     "syntax_error",
				Severity: "error",
				Message:  fmt.Sprintf("Invalid SQL statement syntax at statement %d", i+1),
				Line:     i + 1,
			})
		}

		// Identify statement type
		stmtType := v.identifyStatementType(stmt)
		if stmtType != "" {
			syntax.StatementTypes = append(syntax.StatementTypes, stmtType)
		}

		// Check for common syntax issues
		v.checkCommonSyntaxIssues(stmt, i+1, result)
	}

	result.Syntax = *syntax
}

// checkSafety evaluates migration safety
func (v *MigrationValidator) checkSafety(migration string, result *ValidationResult) {
	safety := &SafetyValidation{
		IsSafe:       true,
		SafetyIssues: []string{},
	}

	migrationUpper := strings.ToUpper(migration)

	// Check for destructive operations
	destructiveOps := []string{
		"DROP TABLE",
		"DROP COLUMN",
		"DROP INDEX",
		"DROP CONSTRAINT",
		"TRUNCATE",
		"DROP DATABASE",
		"DROP SCHEMA",
	}

	for _, op := range destructiveOps {
		if strings.Contains(migrationUpper, op) {
			safety.HasDestructiveOps = true
			safety.DataLossRisk = true
			safety.SafetyIssues = append(safety.SafetyIssues, fmt.Sprintf("Contains destructive operation: %s", op))

			if !strings.Contains(migrationUpper, "IF EXISTS") {
				result.Issues = append(result.Issues, ValidationIssue{
					Type:       "safety",
					Severity:   "error",
					Message:    fmt.Sprintf("%s without IF EXISTS can fail if object doesn't exist", op),
					Suggestion: fmt.Sprintf("Use %s IF EXISTS for safer execution", op),
				})
			}
		}
	}

	// Check for operations that require exclusive locks
	lockingOps := []struct {
		pattern string
		message string
	}{
		{"ALTER TABLE.*ADD COLUMN.*NOT NULL", "Adding NOT NULL column without default requires table rewrite"},
		{"ALTER TABLE.*ALTER COLUMN.*TYPE", "Changing column type may require table rewrite"},
		{"CREATE INDEX", "CREATE INDEX without CONCURRENTLY locks table for writes"},
		{"REINDEX", "REINDEX locks table"},
		{"CLUSTER", "CLUSTER locks table exclusively"},
		{"VACUUM FULL", "VACUUM FULL locks table exclusively"},
	}

	for _, op := range lockingOps {
		if matched, _ := regexp.MatchString(op.pattern, migrationUpper); matched {
			// Special case: CREATE INDEX CONCURRENTLY doesn't lock
			if op.pattern == "CREATE INDEX" && strings.Contains(migrationUpper, "CONCURRENTLY") {
				continue
			}

			safety.LocksTable = true
			safety.RequiresDowntime = true
			safety.SafetyIssues = append(safety.SafetyIssues, op.message)

			result.Warnings = append(result.Warnings, ValidationIssue{
				Type:     "locking",
				Severity: "warning",
				Message:  op.message,
			})
		}
	}

	// Check for adding NOT NULL without default
	if regexp.MustCompile(`(?i)ALTER\s+TABLE.*ADD\s+COLUMN.*NOT\s+NULL`).MatchString(migration) &&
		!regexp.MustCompile(`(?i)ALTER\s+TABLE.*ADD\s+COLUMN.*NOT\s+NULL.*DEFAULT`).MatchString(migration) {
		safety.RequiresDowntime = true
		result.Issues = append(result.Issues, ValidationIssue{
			Type:       "safety",
			Severity:   "error",
			Message:    "Adding NOT NULL column without DEFAULT will fail if table has data",
			Suggestion: "Add a DEFAULT value or add column as nullable first, then add constraint",
		})
	}

	// Check for renaming operations
	if regexp.MustCompile(`(?i)ALTER\s+TABLE.*RENAME\s+(TO|COLUMN)`).MatchString(migration) {
		result.Warnings = append(result.Warnings, ValidationIssue{
			Type:       "compatibility",
			Severity:   "warning",
			Message:    "Renaming tables or columns can break application code",
			Suggestion: "Ensure all application code is updated before renaming",
		})
	}

	safety.IsSafe = !safety.DataLossRisk && len(result.Issues) == 0
	result.Safety = *safety
}

// checkBestPractices evaluates adherence to best practices
func (v *MigrationValidator) checkBestPractices(migration string, result *ValidationResult) {
	practices := &BestPractices{
		ViolatedPractices: []string{},
	}

	migrationUpper := strings.ToUpper(migration)

	// Check for transaction usage
	practices.HasTransaction = strings.Contains(migrationUpper, "BEGIN") ||
		strings.Contains(migrationUpper, "START TRANSACTION")

	// Check for comments
	practices.HasComments = strings.Contains(migration, "--") ||
		strings.Contains(migration, "/*") ||
		strings.Contains(migrationUpper, "COMMENT ON")

	// Check for IF EXISTS usage
	practices.UsesIfExists = strings.Contains(migrationUpper, "IF EXISTS") ||
		strings.Contains(migrationUpper, "IF NOT EXISTS")

	// Check naming conventions
	practices.FollowsNaming = v.checkNamingConventions(migration)

	// Evaluate violations
	if !practices.HasTransaction && !v.containsDDLOnly(migration) {
		practices.ViolatedPractices = append(practices.ViolatedPractices,
			"Migration should use transactions for atomicity")
		result.Warnings = append(result.Warnings, ValidationIssue{
			Type:       "best_practice",
			Severity:   "warning",
			Message:    "Migration doesn't use explicit transaction",
			Suggestion: "Wrap migration in BEGIN/COMMIT for atomicity",
		})
	}

	if !practices.HasComments {
		practices.ViolatedPractices = append(practices.ViolatedPractices,
			"Migration should include comments explaining changes")
		result.Warnings = append(result.Warnings, ValidationIssue{
			Type:       "documentation",
			Severity:   "info",
			Message:    "Migration lacks comments",
			Suggestion: "Add comments to explain the purpose and impact of changes",
		})
	}

	// Check for hardcoded values
	if regexp.MustCompile(`(?i)VALUES\s*\([^)]*'[^']+`).MatchString(migration) {
		result.Warnings = append(result.Warnings, ValidationIssue{
			Type:       "best_practice",
			Severity:   "info",
			Message:    "Migration contains hardcoded values",
			Suggestion: "Consider if hardcoded values should be configuration-driven",
		})
	}

	// Check for proper index naming
	if regexp.MustCompile(`(?i)CREATE\s+INDEX\s+ON`).MatchString(migration) &&
		!regexp.MustCompile(`(?i)CREATE\s+INDEX\s+\w+\s+ON`).MatchString(migration) {
		practices.ViolatedPractices = append(practices.ViolatedPractices,
			"Indexes should have explicit names")
		result.Warnings = append(result.Warnings, ValidationIssue{
			Type:       "naming",
			Severity:   "warning",
			Message:    "Index created without explicit name",
			Suggestion: "Name indexes explicitly for easier management",
		})
	}

	result.BestPractices = *practices
}

// checkReversibility evaluates if migration can be reversed
func (v *MigrationValidator) checkReversibility(migration string, result *ValidationResult) {
	reversibility := &Reversibility{
		IsReversible:       true,
		IrreversibleOps:    []string{},
		DownMigrationHints: []string{},
	}

	migrationUpper := strings.ToUpper(migration)

	// Check for inherently irreversible operations
	irreversiblePatterns := []struct {
		pattern string
		op      string
		hint    string
	}{
		{
			pattern: "DROP TABLE",
			op:      "DROP TABLE",
			hint:    "Save table structure and data before dropping",
		},
		{
			pattern: "DROP COLUMN",
			op:      "DROP COLUMN",
			hint:    "Save column data before dropping",
		},
		{
			pattern: "ALTER COLUMN.*TYPE",
			op:      "Column type change",
			hint:    "Some type conversions may lose data precision",
		},
		{
			pattern: "TRUNCATE",
			op:      "TRUNCATE",
			hint:    "Data cannot be recovered after TRUNCATE",
		},
	}

	for _, p := range irreversiblePatterns {
		if strings.Contains(migrationUpper, p.pattern) {
			reversibility.IsReversible = false
			reversibility.IrreversibleOps = append(reversibility.IrreversibleOps, p.op)
			reversibility.DownMigrationHints = append(reversibility.DownMigrationHints, p.hint)
		}
	}

	// Provide down migration hints for common operations
	if strings.Contains(migrationUpper, "CREATE TABLE") {
		reversibility.DownMigrationHints = append(reversibility.DownMigrationHints,
			"Down migration: DROP TABLE IF EXISTS table_name")
	}

	if strings.Contains(migrationUpper, "ADD COLUMN") {
		reversibility.DownMigrationHints = append(reversibility.DownMigrationHints,
			"Down migration: ALTER TABLE table_name DROP COLUMN IF EXISTS column_name")
	}

	if strings.Contains(migrationUpper, "CREATE INDEX") {
		reversibility.DownMigrationHints = append(reversibility.DownMigrationHints,
			"Down migration: DROP INDEX IF EXISTS index_name")
	}

	if strings.Contains(migrationUpper, "ADD CONSTRAINT") {
		reversibility.DownMigrationHints = append(reversibility.DownMigrationHints,
			"Down migration: ALTER TABLE table_name DROP CONSTRAINT IF EXISTS constraint_name")
	}

	// Check if this looks like a down migration
	if strings.Contains(migration, ".down.sql") || strings.Contains(migrationUpper, "-- DOWN") {
		reversibility.HasDownMigration = true
	}

	result.Reversibility = *reversibility
}

// Helper methods

func (v *MigrationValidator) splitStatements(migration string) []string {
	// Simple statement splitter - in production, use a proper SQL parser
	statements := strings.Split(migration, ";")
	var result []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			result = append(result, stmt)
		}
	}
	return result
}

func (v *MigrationValidator) isValidStatement(stmt string) bool {
	// Basic validation - check if statement starts with known SQL keywords
	validStarts := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP",
		"BEGIN", "COMMIT", "ROLLBACK", "START", "END", "GRANT", "REVOKE",
		"TRUNCATE", "COMMENT", "VACUUM", "ANALYZE", "REINDEX", "CLUSTER",
		"COPY", "DO", "EXPLAIN", "LOCK", "NOTIFY", "PREPARE", "EXECUTE",
		"DEALLOCATE", "DECLARE", "FETCH", "MOVE", "CLOSE", "SET", "RESET",
		"SHOW", "--", "/*", "WITH",
	}

	stmtUpper := strings.ToUpper(strings.TrimSpace(stmt))
	for _, start := range validStarts {
		if strings.HasPrefix(stmtUpper, start) {
			return true
		}
	}
	return false
}

func (v *MigrationValidator) identifyStatementType(stmt string) string {
	stmtUpper := strings.ToUpper(strings.TrimSpace(stmt))

	typeMap := map[string]string{
		"CREATE TABLE":    "CREATE_TABLE",
		"ALTER TABLE":     "ALTER_TABLE",
		"DROP TABLE":      "DROP_TABLE",
		"CREATE INDEX":    "CREATE_INDEX",
		"DROP INDEX":      "DROP_INDEX",
		"INSERT":          "INSERT",
		"UPDATE":          "UPDATE",
		"DELETE":          "DELETE",
		"CREATE VIEW":     "CREATE_VIEW",
		"CREATE FUNCTION": "CREATE_FUNCTION",
		"CREATE TRIGGER":  "CREATE_TRIGGER",
		"CREATE SEQUENCE": "CREATE_SEQUENCE",
		"GRANT":           "GRANT",
		"REVOKE":          "REVOKE",
	}

	for pattern, stmtType := range typeMap {
		if strings.HasPrefix(stmtUpper, pattern) {
			return stmtType
		}
	}

	return "OTHER"
}

func (v *MigrationValidator) checkCommonSyntaxIssues(stmt string, lineNum int, result *ValidationResult) {
	// Check for missing semicolons (heuristic)
	if !strings.HasSuffix(strings.TrimSpace(stmt), ";") &&
		!strings.HasPrefix(strings.TrimSpace(stmt), "--") &&
		!strings.HasPrefix(strings.TrimSpace(stmt), "/*") {
		// This might be OK for the last statement, so make it a warning
		result.Warnings = append(result.Warnings, ValidationIssue{
			Type:       "syntax",
			Severity:   "warning",
			Message:    "Statement may be missing semicolon",
			Line:       lineNum,
			Suggestion: "Ensure all SQL statements end with semicolons",
		})
	}

	// Check for common typos
	typos := map[string]string{
		"CRAETE":  "CREATE",
		"TABEL":   "TABLE",
		"FORM":    "FROM",
		"WEHRE":   "WHERE",
		"UDPATE":  "UPDATE",
		"DELEET":  "DELETE",
		"DEFUALT": "DEFAULT",
	}

	stmtUpper := strings.ToUpper(stmt)
	for typo, correct := range typos {
		if strings.Contains(stmtUpper, typo) {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:       "typo",
				Severity:   "error",
				Message:    fmt.Sprintf("Possible typo: '%s' should be '%s'", typo, correct),
				Line:       lineNum,
				Suggestion: fmt.Sprintf("Replace '%s' with '%s'", typo, correct),
			})
		}
	}

	// Check for unmatched quotes
	singleQuotes := strings.Count(stmt, "'") - strings.Count(stmt, "''")
	if singleQuotes%2 != 0 {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:       "syntax",
			Severity:   "error",
			Message:    "Unmatched single quotes",
			Line:       lineNum,
			Suggestion: "Check for missing or extra single quotes",
		})
	}

	// Check for unmatched parentheses
	openParens := strings.Count(stmt, "(")
	closeParens := strings.Count(stmt, ")")
	if openParens != closeParens {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:       "syntax",
			Severity:   "error",
			Message:    fmt.Sprintf("Unmatched parentheses: %d opening, %d closing", openParens, closeParens),
			Line:       lineNum,
			Suggestion: "Ensure all parentheses are properly matched",
		})
	}
}

func (v *MigrationValidator) checkNamingConventions(migration string) bool {
	// Check for consistent naming patterns
	hasIssues := false

	// Check for mixed case in table/column names (PostgreSQL folds to lowercase)
	if regexp.MustCompile(`(?i)CREATE\s+TABLE\s+"?[a-z]*[A-Z]+[a-zA-Z]*"?\s*\(`).MatchString(migration) {
		hasIssues = true
	}

	// Check for spaces in names without quotes
	if regexp.MustCompile(`(?i)CREATE\s+TABLE\s+[a-zA-Z]+\s+[a-zA-Z]+\s*\(`).MatchString(migration) {
		hasIssues = true
	}

	return !hasIssues
}

func (v *MigrationValidator) containsDDLOnly(migration string) bool {
	// Check if migration contains only DDL (no DML)
	dmlPatterns := []string{
		"INSERT", "UPDATE", "DELETE", "MERGE", "COPY",
	}

	migrationUpper := strings.ToUpper(migration)
	for _, pattern := range dmlPatterns {
		if strings.Contains(migrationUpper, pattern) {
			return false
		}
	}

	return true
}

func (v *MigrationValidator) addSuggestions(migration string, result *ValidationResult) {
	// Add general migration suggestions
	result.Suggestions = append(result.Suggestions,
		"Test migration on a copy of production data before applying",
		"Always have a rollback plan for migrations",
		"Monitor application logs during and after migration",
		"Consider using migration tools like golang-migrate or goose",
	)

	// Add specific suggestions based on content
	if strings.Contains(strings.ToUpper(migration), "ALTER TABLE") {
		result.Suggestions = append(result.Suggestions,
			"For large tables, consider running ALTER TABLE operations during low-traffic periods",
			"Use pg_stat_progress_create_index to monitor long-running index operations",
		)
	}

	if strings.Contains(strings.ToUpper(migration), "CREATE INDEX") &&
		!strings.Contains(strings.ToUpper(migration), "CONCURRENTLY") {
		result.Suggestions = append(result.Suggestions,
			"Consider using CREATE INDEX CONCURRENTLY to avoid locking the table",
		)
	}

	if result.Safety.RequiresDowntime {
		result.Suggestions = append(result.Suggestions,
			"This migration requires downtime - plan accordingly",
			"Consider breaking into multiple smaller migrations to minimize downtime",
		)
	}

	if len(result.Reversibility.IrreversibleOps) > 0 {
		result.Suggestions = append(result.Suggestions,
			"Create a backup before running irreversible operations",
			"Document the original state for manual recovery if needed",
		)
	}

	// Performance suggestions
	if strings.Contains(strings.ToUpper(migration), "NOT NULL") {
		result.Suggestions = append(result.Suggestions,
			"Adding NOT NULL constraints to existing columns requires a full table scan",
		)
	}

	if regexp.MustCompile(`(?i)CREATE\s+TABLE.*REFERENCES`).MatchString(migration) {
		result.Suggestions = append(result.Suggestions,
			"Foreign keys are created with indexes on the referencing column by default",
		)
	}
}
