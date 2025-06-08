// Package postgres provides PostgreSQL management tools for the Assistant.
// It includes query optimization, migration management, schema analysis,
// and performance insights for PostgreSQL databases.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/koopa0/assistant-go/internal/tool"
)

// PostgresTool implements the Tool interface for PostgreSQL operations
type PostgresTool struct {
	logger *slog.Logger
	pool   *pgxpool.Pool // Optional: can work without connection for some operations
}

// NewPostgresTool creates a new PostgreSQL tool instance
func NewPostgresTool(logger *slog.Logger) *PostgresTool {
	return &PostgresTool{
		logger: logger,
	}
}

// Name returns the tool name
func (t *PostgresTool) Name() string {
	return "postgres"
}

// Description returns the tool description
func (t *PostgresTool) Description() string {
	return "PostgreSQL query optimization, migration management, schema analysis, and performance insights"
}

// Parameters returns the tool parameter schema
func (t *PostgresTool) Parameters() *tool.ToolParametersSchema {
	return &tool.ToolParametersSchema{
		Type: "object",
		Properties: map[string]tool.ParameterProperty{
			"action": {
				Type:        tool.ParameterTypeString,
				Description: "The PostgreSQL action to perform",
				Enum: []string{
					"analyze_query",
					"optimize_query",
					"explain_query",
					"generate_migration",
					"analyze_schema",
					"suggest_indexes",
					"check_performance",
					"validate_migration",
				},
			},
			"query": {
				Type:        tool.ParameterTypeString,
				Description: "SQL query to analyze or optimize",
			},
			"schema": {
				Type:        tool.ParameterTypeString,
				Description: "Schema name to analyze",
			},
			"table": {
				Type:        tool.ParameterTypeString,
				Description: "Table name for operations",
			},
			"migration_type": {
				Type:        tool.ParameterTypeString,
				Description: "Type of migration to generate",
				Enum:        []string{"create_table", "add_column", "add_index", "alter_table"},
			},
			"connection_string": {
				Type:        tool.ParameterTypeString,
				Description: "PostgreSQL connection string (optional for analysis)",
			},
		},
		Required: []string{"action"},
	}
}

// Health checks the health of the tool
func (t *PostgresTool) Health(ctx context.Context) error {
	// PostgresTool is always healthy as it doesn't maintain persistent connections
	return nil
}

// Close closes any resources
func (t *PostgresTool) Close(ctx context.Context) error {
	// PostgresTool doesn't maintain persistent connections
	return nil
}

// Execute runs the PostgreSQL tool with the given parameters
func (t *PostgresTool) Execute(ctx context.Context, input *tool.ToolInput) (*tool.ToolResult, error) {
	startTime := time.Now()

	// Extract parameters from input
	params := input.Parameters
	if params == nil {
		params = make(map[string]interface{})
	}

	action, ok := params["action"].(string)
	if !ok {
		return &tool.ToolResult{
			Success: false,
			Error:   "action parameter is required",
		}, nil
	}

	t.logger.Info("Executing PostgreSQL action",
		slog.String("action", action))

	// Connect to database if connection string provided
	if connStr, ok := params["connection_string"].(string); ok && connStr != "" {
		pool, err := pgxpool.New(ctx, connStr)
		if err != nil {
			t.logger.Warn("Failed to connect to database, continuing with limited functionality",
				slog.String("error", err.Error()))
		} else {
			t.pool = pool
			defer pool.Close()
		}
	}

	var result interface{}
	var err error

	switch action {
	case "analyze_query":
		result, err = t.analyzeQuery(ctx, params)
	case "optimize_query":
		result, err = t.optimizeQuery(ctx, params)
	case "explain_query":
		result, err = t.explainQuery(ctx, params)
	case "generate_migration":
		result, err = t.generateMigration(ctx, params)
	case "analyze_schema":
		result, err = t.analyzeSchema(ctx, params)
	case "suggest_indexes":
		result, err = t.suggestIndexes(ctx, params)
	case "check_performance":
		result, err = t.checkPerformance(ctx, params)
	case "validate_migration":
		result, err = t.validateMigration(ctx, params)
	default:
		return &tool.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown action: %s", action),
		}, nil
	}

	if err != nil {
		return &tool.ToolResult{
			Success:       false,
			Error:         err.Error(),
			ExecutionTime: time.Since(startTime),
		}, nil
	}

	// Convert result to map[string]interface{} for output
	var outputMap map[string]interface{}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		outputMap = map[string]interface{}{
			"result": result,
		}
	} else {
		err = json.Unmarshal(resultJSON, &outputMap)
		if err != nil {
			outputMap = map[string]interface{}{
				"result": result,
			}
		}
	}

	return &tool.ToolResult{
		Success: true,
		Data: &tool.ToolResultData{
			Result: result,
			Output: outputMap,
		},
		ExecutionTime: time.Since(startTime),
	}, nil
}

// analyzeQuery analyzes a SQL query
func (t *PostgresTool) analyzeQuery(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	// If no database connection, perform static analysis
	if t.pool == nil {
		return t.performStaticAnalysis(query), nil
	}

	// With database connection, can do EXPLAIN ANALYZE
	analyzer := NewQueryAnalyzer(t.logger)
	return analyzer.Analyze(query)
}

// optimizeQuery optimizes a SQL query
func (t *PostgresTool) optimizeQuery(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	optimizer := NewQueryOptimizer(t.logger)
	suggestions, err := optimizer.Optimize(query)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"original_query":           query,
		"optimization_suggestions": suggestions,
	}, nil
}

// explainQuery explains a SQL query execution plan
func (t *PostgresTool) explainQuery(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	if t.pool == nil {
		return map[string]interface{}{
			"error": "Database connection required for EXPLAIN",
		}, fmt.Errorf("database connection required")
	}

	// Execute EXPLAIN
	rows, err := t.pool.Query(ctx, fmt.Sprintf("EXPLAIN (ANALYZE, BUFFERS) %s", query))
	if err != nil {
		return nil, fmt.Errorf("failed to explain query: %w", err)
	}
	defer rows.Close()

	var plan []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			continue
		}
		plan = append(plan, line)
	}

	return map[string]interface{}{
		"query": query,
		"plan":  plan,
	}, nil
}

// generateMigration generates a database migration
func (t *PostgresTool) generateMigration(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	migrationType, _ := params["migration_type"].(string)
	if migrationType == "" {
		return nil, fmt.Errorf("migration_type parameter is required")
	}

	generator := NewMigrationGenerator(t.logger)

	// Prepare migration parameters
	tableName, _ := params["table"].(string)
	schema, _ := params["schema"].(string)

	migrationParams := MigrationParams{
		Type:      migrationType,
		TableName: tableName,
		Schema:    schema,
	}

	if migrationParams.Schema == "" {
		migrationParams.Schema = "public"
	}

	switch migrationType {
	case "create_table":
		if migrationParams.TableName == "" {
			return nil, fmt.Errorf("table parameter is required for create_table")
		}
		// Parse columns from params
		if columns, ok := params["columns"].([]interface{}); ok {
			migrationParams.Columns = columns
		}
		return generator.Generate(migrationParams)

	case "add_column":
		if migrationParams.TableName == "" {
			return nil, fmt.Errorf("table parameter is required")
		}
		// Build column definition from params
		columnName, _ := params["column"].(string)
		columnType, _ := params["column_type"].(string)
		if columnName == "" || columnType == "" {
			return nil, fmt.Errorf("column and column_type parameters are required")
		}
		migrationParams.Columns = []interface{}{
			map[string]interface{}{
				"name": columnName,
				"type": columnType,
			},
		}
		return generator.Generate(migrationParams)

	case "add_index":
		if migrationParams.TableName == "" {
			return nil, fmt.Errorf("table parameter is required")
		}
		columns, _ := params["columns"].([]string)
		if len(columns) == 0 {
			return nil, fmt.Errorf("columns parameter is required")
		}
		// Build index definition
		migrationParams.Indexes = []IndexDef{
			{
				Columns: columns,
			},
		}
		return generator.Generate(migrationParams)

	default:
		return nil, fmt.Errorf("unsupported migration type: %s", migrationType)
	}
}

// analyzeSchema analyzes database schema
func (t *PostgresTool) analyzeSchema(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	schema, _ := params["schema"].(string)
	if schema == "" {
		schema = "public"
	}

	if t.pool == nil {
		return map[string]interface{}{
			"error": "Database connection required for schema analysis",
		}, fmt.Errorf("database connection required")
	}

	analyzer := NewSchemaAnalyzer(t.logger, t.pool)
	return analyzer.Analyze(ctx, schema)
}

// suggestIndexes suggests indexes for a table
func (t *PostgresTool) suggestIndexes(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	table, ok := params["table"].(string)
	if !ok || table == "" {
		return nil, fmt.Errorf("table parameter is required")
	}

	schema, _ := params["schema"].(string)
	if schema == "" {
		schema = "public"
	}

	if t.pool == nil {
		return map[string]interface{}{
			"error": "Database connection required for index suggestions",
		}, fmt.Errorf("database connection required")
	}

	advisor := NewIndexAdvisor(t.logger, t.pool)
	return advisor.SuggestIndexes(ctx, table)
}

// checkPerformance checks database performance
func (t *PostgresTool) checkPerformance(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if t.pool == nil {
		return map[string]interface{}{
			"error": "Database connection required for performance check",
		}, fmt.Errorf("database connection required")
	}

	checker := NewPerformanceChecker(t.logger, t.pool)
	return checker.Check(ctx)
}

// validateMigration validates a migration
func (t *PostgresTool) validateMigration(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	migration, ok := params["migration"].(string)
	if !ok || migration == "" {
		return nil, fmt.Errorf("migration parameter is required")
	}

	validator := NewMigrationValidator(t.logger)
	return validator.Validate(migration)
}

// performStaticAnalysis performs static SQL analysis without database connection
func (t *PostgresTool) performStaticAnalysis(query string) map[string]interface{} {
	// Basic static analysis
	result := map[string]interface{}{
		"query":       query,
		"warnings":    []string{},
		"suggestions": []string{},
	}

	// Check for common issues
	warnings := []string{}
	suggestions := []string{}

	if len(query) > 1000 {
		warnings = append(warnings, "Query is very long, consider breaking it down")
	}

	// Check for SELECT *
	if contains(query, "SELECT *") || contains(query, "select *") {
		suggestions = append(suggestions, "Avoid SELECT *, specify columns explicitly")
	}

	// Check for missing WHERE in UPDATE/DELETE
	upperQuery := strings.ToUpper(query)
	if (strings.Contains(upperQuery, "UPDATE") || strings.Contains(upperQuery, "DELETE")) &&
		!strings.Contains(upperQuery, "WHERE") {
		warnings = append(warnings, "UPDATE/DELETE without WHERE clause affects all rows")
	}

	result["warnings"] = warnings
	result["suggestions"] = suggestions

	return result
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToUpper(s), strings.ToUpper(substr))
}
