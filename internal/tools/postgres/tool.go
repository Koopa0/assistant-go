// Package postgres provides PostgreSQL management tools for the Assistant.
// It includes query optimization, migration management, schema analysis,
// and performance insights for PostgreSQL databases.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/koopa0/assistant-go/internal/tools"
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
func (t *PostgresTool) Parameters() *tools.ToolParametersSchema {
	return &tools.ToolParametersSchema{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"action": {
				Type:        tools.ParameterTypeString,
				Description: "The PostgreSQL action to perform",
				Required:    true,
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
				Type:        tools.ParameterTypeString,
				Description: "SQL query to analyze or optimize",
				Required:    false,
			},
			"schema": {
				Type:        tools.ParameterTypeString,
				Description: "Schema name to analyze",
				Required:    false,
			},
			"table": {
				Type:        tools.ParameterTypeString,
				Description: "Table name for operations",
				Required:    false,
			},
			"migration_type": {
				Type:        tools.ParameterTypeString,
				Description: "Type of migration to generate",
				Required:    false,
				Enum:        []string{"create_table", "add_column", "add_index", "alter_table"},
			},
			"connection_string": {
				Type:        tools.ParameterTypeString,
				Description: "PostgreSQL connection string (optional for analysis)",
				Required:    false,
			},
		},
		Required: []string{"action"},
	}
}

// Execute runs the PostgreSQL tool with the given parameters
func (t *PostgresTool) Execute(ctx context.Context, input *tools.ToolInput) (*tools.ToolResult, error) {
	startTime := time.Now()

	// Extract parameters from input
	params := input.Parameters
	if params == nil {
		params = make(map[string]interface{})
	}

	action, ok := params["action"].(string)
	if !ok {
		return &tools.ToolResult{
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
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown action: %s", action),
		}, nil
	}

	if err != nil {
		return &tools.ToolResult{
			Success:       false,
			Error:         err.Error(),
			ExecutionTime: time.Since(startTime),
		}, nil
	}

	// Convert result to JSON for output
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &tools.ToolResult{
			Success:       false,
			Error:         fmt.Sprintf("failed to marshal result: %v", err),
			ExecutionTime: time.Since(startTime),
		}, nil
	}

	return &tools.ToolResult{
		Success: true,
		Data: &tools.ToolResultData{
			Output: string(resultJSON),
		},
		ExecutionTime: time.Since(startTime),
	}, nil
}

// analyzeQuery analyzes a SQL query for potential issues
func (t *PostgresTool) analyzeQuery(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required for analyze_query")
	}

	analyzer := NewQueryAnalyzer(t.logger)
	return analyzer.Analyze(query)
}

// optimizeQuery suggests optimizations for a SQL query
func (t *PostgresTool) optimizeQuery(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required for optimize_query")
	}

	optimizer := NewQueryOptimizer(t.logger)
	return optimizer.Optimize(query)
}

// explainQuery runs EXPLAIN ANALYZE on a query if connected
func (t *PostgresTool) explainQuery(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required for explain_query")
	}

	if t.pool == nil {
		// Provide static analysis without connection
		analyzer := NewQueryAnalyzer(t.logger)
		return analyzer.StaticExplain(query)
	}

	// Run actual EXPLAIN ANALYZE
	explainQuery := fmt.Sprintf("EXPLAIN (ANALYZE, BUFFERS, TIMING, VERBOSE) %s", query)

	rows, err := t.pool.Query(ctx, explainQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to run EXPLAIN: %w", err)
	}
	defer rows.Close()

	var planLines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return nil, fmt.Errorf("failed to scan EXPLAIN result: %w", err)
		}
		planLines = append(planLines, line)
	}

	analyzer := NewQueryAnalyzer(t.logger)
	return analyzer.AnalyzeExplainOutput(planLines)
}

// generateMigration generates a database migration
func (t *PostgresTool) generateMigration(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	migrationType, _ := params["migration_type"].(string)
	table, _ := params["table"].(string)

	generator := NewMigrationGenerator(t.logger)

	schema, _ := params["schema"].(string)

	migrationParams := MigrationParams{
		Type:      migrationType,
		TableName: table,
		Schema:    schema,
	}

	// Extract additional parameters
	if columns, ok := params["columns"].([]interface{}); ok {
		migrationParams.Columns = columns
	}

	return generator.Generate(migrationParams)
}

// analyzeSchema analyzes database schema for improvements
func (t *PostgresTool) analyzeSchema(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	schemaName, _ := params["schema"].(string)
	if schemaName == "" {
		schemaName = "public"
	}

	analyzer := NewSchemaAnalyzer(t.logger, t.pool)
	return analyzer.Analyze(ctx, schemaName)
}

// suggestIndexes suggests indexes based on query patterns
func (t *PostgresTool) suggestIndexes(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	table, _ := params["table"].(string)

	advisor := NewIndexAdvisor(t.logger, t.pool)
	return advisor.SuggestIndexes(ctx, table)
}

// checkPerformance checks database performance metrics
func (t *PostgresTool) checkPerformance(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if t.pool == nil {
		return map[string]string{
			"error": "Database connection required for performance check",
		}, nil
	}

	checker := NewPerformanceChecker(t.logger, t.pool)
	return checker.Check(ctx)
}

// validateMigration validates a migration file
func (t *PostgresTool) validateMigration(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	migration, ok := params["migration"].(string)
	if !ok || migration == "" {
		return nil, fmt.Errorf("migration parameter is required for validate_migration")
	}

	validator := NewMigrationValidator(t.logger)
	return validator.Validate(migration)
}

// Health checks if the PostgreSQL tool is healthy
func (t *PostgresTool) Health(ctx context.Context) error {
	// Tool can work without database connection
	t.logger.Debug("PostgreSQL tool health check passed")
	return nil
}

// Close closes the PostgreSQL tool and cleans up resources
func (t *PostgresTool) Close(ctx context.Context) error {
	if t.pool != nil {
		t.pool.Close()
	}
	t.logger.Debug("PostgreSQL tool closed")
	return nil
}
