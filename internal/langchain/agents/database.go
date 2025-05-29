package agents

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"

	"github.com/koopa0/assistant/internal/config"
	"github.com/koopa0/assistant/internal/storage/postgres"
)

// DatabaseAgent specializes in database operations and SQL assistance
type DatabaseAgent struct {
	*BaseAgent
	sqlGenerator   *SQLGenerator
	queryOptimizer *QueryOptimizer
	schemaExplorer *SchemaExplorer
	dbClient       *postgres.SQLCClient
}

// NewDatabaseAgent creates a new database expert agent
func NewDatabaseAgent(llm llms.Model, config config.LangChain, dbClient *postgres.SQLCClient, logger *slog.Logger) *DatabaseAgent {
	base := NewBaseAgent(AgentTypeDatabase, llm, config, logger)

	agent := &DatabaseAgent{
		BaseAgent:      base,
		sqlGenerator:   NewSQLGenerator(llm, logger),
		queryOptimizer: NewQueryOptimizer(llm, logger),
		schemaExplorer: NewSchemaExplorer(dbClient, logger),
		dbClient:       dbClient,
	}

	// Add database-specific capabilities
	agent.initializeCapabilities()

	return agent
}

// initializeCapabilities sets up the database agent's capabilities
func (d *DatabaseAgent) initializeCapabilities() {
	capabilities := []AgentCapability{
		{
			Name:        "sql_generation",
			Description: "Generate SQL queries from natural language descriptions",
			Parameters: map[string]interface{}{
				"description": "string",
				"table_names": "[]string",
				"query_type":  "string (SELECT|INSERT|UPDATE|DELETE)",
				"complexity":  "string (simple|medium|complex)",
			},
		},
		{
			Name:        "query_optimization",
			Description: "Analyze and optimize SQL queries for better performance",
			Parameters: map[string]interface{}{
				"query":             "string",
				"optimization_type": "string (performance|readability|maintainability)",
				"include_explain":   "bool",
			},
		},
		{
			Name:        "schema_exploration",
			Description: "Explore database schema and relationships",
			Parameters: map[string]interface{}{
				"exploration_type": "string (tables|relationships|indexes|constraints)",
				"table_pattern":    "string",
				"include_data":     "bool",
			},
		},
		{
			Name:        "data_analysis",
			Description: "Perform data analysis and generate insights",
			Parameters: map[string]interface{}{
				"table_name":    "string",
				"analysis_type": "string (summary|distribution|trends|anomalies)",
				"date_range":    "string",
			},
		},
		{
			Name:        "migration_assistance",
			Description: "Help with database migrations and schema changes",
			Parameters: map[string]interface{}{
				"migration_type": "string (create|alter|drop)",
				"target_schema":  "string",
				"safety_checks":  "bool",
			},
		},
	}

	for _, capability := range capabilities {
		d.AddCapability(capability)
	}
}

// executeSteps implements specialized database agent logic
func (d *DatabaseAgent) executeSteps(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Analyze the request to determine the database task
	taskType, err := d.analyzeRequestType(request.Query)
	if err != nil {
		return "", nil, fmt.Errorf("failed to analyze request type: %w", err)
	}

	d.logger.Debug("Database task identified",
		slog.String("task_type", taskType),
		slog.String("query", request.Query))

	// Execute based on task type
	switch taskType {
	case "sql_generation":
		return d.executeSQLGeneration(ctx, request, maxSteps)
	case "query_optimization":
		return d.executeQueryOptimization(ctx, request, maxSteps)
	case "schema_exploration":
		return d.executeSchemaExploration(ctx, request, maxSteps)
	case "data_analysis":
		return d.executeDataAnalysis(ctx, request, maxSteps)
	case "migration_assistance":
		return d.executeMigrationAssistance(ctx, request, maxSteps)
	default:
		return d.executeGeneralDatabase(ctx, request, maxSteps)
	}
}

// analyzeRequestType determines what type of database task is being requested
func (d *DatabaseAgent) analyzeRequestType(query string) (string, error) {
	query = strings.ToLower(query)

	// SQL generation patterns
	if strings.Contains(query, "generate") || strings.Contains(query, "create query") || strings.Contains(query, "write sql") {
		return "sql_generation", nil
	}

	// Query optimization patterns
	if strings.Contains(query, "optimize") || strings.Contains(query, "improve") || strings.Contains(query, "performance") {
		return "query_optimization", nil
	}

	// Schema exploration patterns
	if strings.Contains(query, "schema") || strings.Contains(query, "tables") || strings.Contains(query, "structure") {
		return "schema_exploration", nil
	}

	// Data analysis patterns
	if strings.Contains(query, "analyze") || strings.Contains(query, "insights") || strings.Contains(query, "trends") {
		return "data_analysis", nil
	}

	// Migration patterns
	if strings.Contains(query, "migration") || strings.Contains(query, "migrate") || strings.Contains(query, "alter table") {
		return "migration_assistance", nil
	}

	return "general", nil
}

// executeSQLGeneration generates SQL queries from natural language
func (d *DatabaseAgent) executeSQLGeneration(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	steps := make([]AgentStep, 0)

	// Step 1: Analyze requirements
	stepStart := time.Now()
	requirements := d.extractSQLRequirements(request.Query)

	step1 := AgentStep{
		StepNumber: 1,
		Action:     "analyze_sql_requirements",
		Input:      request.Query,
		Output:     fmt.Sprintf("Identified %d requirements", len(requirements)),
		Reasoning:  "Extract SQL generation requirements from natural language",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"requirements": requirements},
	}
	steps = append(steps, step1)

	// Step 2: Explore relevant schema
	stepStart = time.Now()
	schema, err := d.schemaExplorer.GetRelevantSchema(ctx, requirements)
	if err != nil {
		d.logger.Warn("Failed to get schema information", slog.Any("error", err))
		schema = &SchemaInfo{Tables: []TableInfo{}}
	}

	step2 := AgentStep{
		StepNumber: 2,
		Action:     "explore_schema",
		Tool:       "schema_explorer",
		Input:      fmt.Sprintf("Requirements: %v", requirements),
		Output:     fmt.Sprintf("Found %d relevant tables", len(schema.Tables)),
		Reasoning:  "Get database schema information for SQL generation",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"table_count": len(schema.Tables)},
	}
	steps = append(steps, step2)

	// Step 3: Generate SQL using LLM
	stepStart = time.Now()
	prompt := d.buildSQLGenerationPrompt(request.Query, requirements, schema)
	sql, err := d.llm.Call(ctx, prompt, llms.WithMaxTokens(2000))
	if err != nil {
		return "", nil, fmt.Errorf("SQL generation failed: %w", err)
	}

	step3 := AgentStep{
		StepNumber: 3,
		Action:     "generate_sql",
		Tool:       "llm",
		Input:      prompt,
		Output:     sql,
		Reasoning:  "Generate SQL query using LLM with schema context",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"sql_length": len(sql)},
	}
	steps = append(steps, step3)

	// Step 4: Validate and explain SQL
	stepStart = time.Now()
	validation := d.validateSQL(sql)
	explanation := d.explainSQL(sql)

	step4 := AgentStep{
		StepNumber: 4,
		Action:     "validate_and_explain",
		Tool:       "sql_validator",
		Input:      sql,
		Output:     fmt.Sprintf("Valid: %v, Explanation: %s", validation.IsValid, explanation),
		Reasoning:  "Validate generated SQL and provide explanation",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"is_valid": validation.IsValid},
	}
	steps = append(steps, step4)

	// Build final response
	response := fmt.Sprintf("Generated SQL Query:\n\n```sql\n%s\n```\n\nExplanation:\n%s", sql, explanation)
	if !validation.IsValid {
		response += fmt.Sprintf("\n\nValidation Issues:\n%s", strings.Join(validation.Issues, "\n"))
	}

	return response, steps, nil
}

// executeQueryOptimization optimizes existing SQL queries
func (d *DatabaseAgent) executeQueryOptimization(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	steps := make([]AgentStep, 0)

	// Step 1: Extract SQL from request
	stepStart := time.Now()
	sql, err := d.extractSQLFromRequest(request)
	if err != nil {
		return "", nil, fmt.Errorf("failed to extract SQL: %w", err)
	}

	step1 := AgentStep{
		StepNumber: 1,
		Action:     "extract_sql",
		Input:      request.Query,
		Output:     fmt.Sprintf("Extracted SQL (%d characters)", len(sql)),
		Reasoning:  "Extract SQL query for optimization",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"sql_length": len(sql)},
	}
	steps = append(steps, step1)

	// Step 2: Analyze query performance
	stepStart = time.Now()
	analysis := d.queryOptimizer.AnalyzeQuery(sql)

	step2 := AgentStep{
		StepNumber: 2,
		Action:     "analyze_performance",
		Tool:       "query_optimizer",
		Input:      sql,
		Output:     analysis.Summary,
		Reasoning:  "Analyze query performance characteristics",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"issues_found": len(analysis.Issues)},
	}
	steps = append(steps, step2)

	// Step 3: Generate optimized query
	stepStart = time.Now()
	prompt := d.buildOptimizationPrompt(sql, analysis)
	optimizedSQL, err := d.llm.Call(ctx, prompt, llms.WithMaxTokens(2000))
	if err != nil {
		return "", nil, fmt.Errorf("query optimization failed: %w", err)
	}

	step3 := AgentStep{
		StepNumber: 3,
		Action:     "optimize_query",
		Tool:       "llm",
		Input:      prompt,
		Output:     optimizedSQL,
		Reasoning:  "Generate optimized version of the query",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"optimized_length": len(optimizedSQL)},
	}
	steps = append(steps, step3)

	// Build response with before/after comparison
	response := fmt.Sprintf("Query Optimization Results:\n\nOriginal Query:\n```sql\n%s\n```\n\nOptimized Query:\n```sql\n%s\n```\n\nOptimization Summary:\n%s",
		sql, optimizedSQL, analysis.Summary)

	return response, steps, nil
}

// executeSchemaExploration explores database schema
func (d *DatabaseAgent) executeSchemaExploration(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	steps := make([]AgentStep, 0)

	// Step 1: Determine exploration scope
	stepStart := time.Now()
	scope := d.determineExplorationScope(request.Query)

	step1 := AgentStep{
		StepNumber: 1,
		Action:     "determine_scope",
		Input:      request.Query,
		Output:     fmt.Sprintf("Exploration scope: %s", scope),
		Reasoning:  "Determine what aspects of schema to explore",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"scope": scope},
	}
	steps = append(steps, step1)

	// Step 2: Explore schema
	stepStart = time.Now()
	schema, err := d.schemaExplorer.ExploreSchema(ctx, scope)
	if err != nil {
		return "", nil, fmt.Errorf("schema exploration failed: %w", err)
	}

	step2 := AgentStep{
		StepNumber: 2,
		Action:     "explore_schema",
		Tool:       "schema_explorer",
		Input:      scope,
		Output:     fmt.Sprintf("Found %d tables, %d relationships", len(schema.Tables), len(schema.Relationships)),
		Reasoning:  "Explore database schema based on scope",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"tables": len(schema.Tables), "relationships": len(schema.Relationships)},
	}
	steps = append(steps, step2)

	// Step 3: Generate schema report
	stepStart = time.Now()
	prompt := d.buildSchemaReportPrompt(schema, scope)
	report, err := d.llm.Call(ctx, prompt, llms.WithMaxTokens(3000))
	if err != nil {
		return "", nil, fmt.Errorf("schema report generation failed: %w", err)
	}

	step3 := AgentStep{
		StepNumber: 3,
		Action:     "generate_report",
		Tool:       "llm",
		Input:      prompt,
		Output:     report,
		Reasoning:  "Generate comprehensive schema exploration report",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"report_length": len(report)},
	}
	steps = append(steps, step3)

	return report, steps, nil
}

// executeDataAnalysis performs data analysis tasks
func (d *DatabaseAgent) executeDataAnalysis(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Simplified implementation - delegate to general database handling
	return d.executeGeneralDatabase(ctx, request, maxSteps)
}

// executeMigrationAssistance helps with database migrations
func (d *DatabaseAgent) executeMigrationAssistance(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Simplified implementation - delegate to general database handling
	return d.executeGeneralDatabase(ctx, request, maxSteps)
}

// executeGeneralDatabase handles general database queries
func (d *DatabaseAgent) executeGeneralDatabase(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Fallback to base agent implementation with database context
	prompt := fmt.Sprintf("As a database expert, please help with: %s", request.Query)
	result, err := d.llm.Call(ctx, prompt, llms.WithMaxTokens(2000))
	if err != nil {
		return "", nil, fmt.Errorf("general database query failed: %w", err)
	}

	step := AgentStep{
		StepNumber: 1,
		Action:     "general_database_assistance",
		Tool:       "llm",
		Input:      prompt,
		Output:     result,
		Reasoning:  "Provide general database assistance",
		Duration:   time.Millisecond * 100, // Placeholder
		Metadata:   map[string]interface{}{"response_length": len(result)},
	}

	return result, []AgentStep{step}, nil
}

// Helper methods

func (d *DatabaseAgent) extractSQLRequirements(query string) []string {
	requirements := make([]string, 0)

	// Simple pattern matching for SQL requirements
	if strings.Contains(strings.ToLower(query), "select") || strings.Contains(strings.ToLower(query), "find") {
		requirements = append(requirements, "SELECT")
	}
	if strings.Contains(strings.ToLower(query), "insert") || strings.Contains(strings.ToLower(query), "add") {
		requirements = append(requirements, "INSERT")
	}
	if strings.Contains(strings.ToLower(query), "update") || strings.Contains(strings.ToLower(query), "modify") {
		requirements = append(requirements, "UPDATE")
	}
	if strings.Contains(strings.ToLower(query), "delete") || strings.Contains(strings.ToLower(query), "remove") {
		requirements = append(requirements, "DELETE")
	}

	return requirements
}

func (d *DatabaseAgent) extractSQLFromRequest(request *AgentRequest) (string, error) {
	// Try to extract SQL from context
	if sql, exists := request.Context["sql"]; exists {
		if sqlStr, ok := sql.(string); ok {
			return sqlStr, nil
		}
	}

	// Try to extract SQL from query using regex
	sqlPattern := regexp.MustCompile("```sql\\s*([\\s\\S]*?)\\s*```")
	matches := sqlPattern.FindStringSubmatch(request.Query)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1]), nil
	}

	return "", fmt.Errorf("no SQL found in request")
}

func (d *DatabaseAgent) determineExplorationScope(query string) string {
	query = strings.ToLower(query)

	if strings.Contains(query, "relationship") || strings.Contains(query, "foreign key") {
		return "relationships"
	}
	if strings.Contains(query, "index") {
		return "indexes"
	}
	if strings.Contains(query, "constraint") {
		return "constraints"
	}

	return "tables"
}

func (d *DatabaseAgent) buildSQLGenerationPrompt(query string, requirements []string, schema *SchemaInfo) string {
	return fmt.Sprintf(`Generate a SQL query based on the following requirements:

User Request: %s
Requirements: %v
Available Tables: %s

Please provide:
1. A well-formatted SQL query
2. Brief explanation of the query logic
3. Any assumptions made

SQL Query:`, query, requirements, d.formatSchemaForPrompt(schema))
}

func (d *DatabaseAgent) buildOptimizationPrompt(sql string, analysis *QueryAnalysis) string {
	return fmt.Sprintf(`Optimize the following SQL query:

Original Query:
%s

Performance Analysis:
%s

Please provide:
1. Optimized SQL query
2. Explanation of optimizations made
3. Expected performance improvements

Optimized Query:`, sql, analysis.Summary)
}

func (d *DatabaseAgent) buildSchemaReportPrompt(schema *SchemaInfo, scope string) string {
	return fmt.Sprintf(`Generate a comprehensive database schema report:

Scope: %s
Schema Information: %s

Please provide:
1. Overview of the database structure
2. Key relationships and dependencies
3. Recommendations for improvements
4. Potential issues or concerns

Report:`, scope, d.formatSchemaForPrompt(schema))
}

func (d *DatabaseAgent) formatSchemaForPrompt(schema *SchemaInfo) string {
	if len(schema.Tables) == 0 {
		return "No schema information available"
	}

	result := ""
	for _, table := range schema.Tables {
		result += fmt.Sprintf("Table: %s (Columns: %d)\n", table.Name, len(table.Columns))
	}

	return result
}

func (d *DatabaseAgent) validateSQL(sql string) *SQLValidation {
	// Simple SQL validation (could be enhanced with actual SQL parser)
	validation := &SQLValidation{
		IsValid: true,
		Issues:  make([]string, 0),
	}

	// Basic syntax checks
	if !strings.Contains(strings.ToUpper(sql), "SELECT") &&
		!strings.Contains(strings.ToUpper(sql), "INSERT") &&
		!strings.Contains(strings.ToUpper(sql), "UPDATE") &&
		!strings.Contains(strings.ToUpper(sql), "DELETE") {
		validation.IsValid = false
		validation.Issues = append(validation.Issues, "No valid SQL statement found")
	}

	return validation
}

func (d *DatabaseAgent) explainSQL(sql string) string {
	// Simple SQL explanation (could be enhanced with actual SQL analysis)
	return fmt.Sprintf("This SQL query performs operations on the database. Query length: %d characters.", len(sql))
}

// Supporting types

type SchemaInfo struct {
	Tables        []TableInfo        `json:"tables"`
	Relationships []RelationshipInfo `json:"relationships"`
	Indexes       []IndexInfo        `json:"indexes"`
	Constraints   []ConstraintInfo   `json:"constraints"`
}

type TableInfo struct {
	Name     string       `json:"name"`
	Columns  []ColumnInfo `json:"columns"`
	RowCount int64        `json:"row_count"`
}

type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default"`
}

type RelationshipInfo struct {
	FromTable  string `json:"from_table"`
	FromColumn string `json:"from_column"`
	ToTable    string `json:"to_table"`
	ToColumn   string `json:"to_column"`
	Type       string `json:"type"`
}

type IndexInfo struct {
	Name    string   `json:"name"`
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}

type ConstraintInfo struct {
	Name       string `json:"name"`
	Table      string `json:"table"`
	Type       string `json:"type"`
	Definition string `json:"definition"`
}

type QueryAnalysis struct {
	Summary       string   `json:"summary"`
	Issues        []string `json:"issues"`
	Suggestions   []string `json:"suggestions"`
	EstimatedCost int      `json:"estimated_cost"`
}

type SQLValidation struct {
	IsValid bool     `json:"is_valid"`
	Issues  []string `json:"issues"`
}

// SQLGenerator generates SQL queries using LLM
type SQLGenerator struct {
	llm    llms.Model
	logger *slog.Logger
}

func NewSQLGenerator(llm llms.Model, logger *slog.Logger) *SQLGenerator {
	return &SQLGenerator{llm: llm, logger: logger}
}

// QueryOptimizer optimizes SQL queries
type QueryOptimizer struct {
	llm    llms.Model
	logger *slog.Logger
}

func NewQueryOptimizer(llm llms.Model, logger *slog.Logger) *QueryOptimizer {
	return &QueryOptimizer{llm: llm, logger: logger}
}

func (q *QueryOptimizer) AnalyzeQuery(sql string) *QueryAnalysis {
	// Simple query analysis (could be enhanced with actual SQL parsing)
	analysis := &QueryAnalysis{
		Summary:       fmt.Sprintf("Query analysis for %d character SQL statement", len(sql)),
		Issues:        make([]string, 0),
		Suggestions:   make([]string, 0),
		EstimatedCost: len(sql), // Simplified cost estimation
	}

	// Basic analysis patterns
	if strings.Contains(strings.ToUpper(sql), "SELECT *") {
		analysis.Issues = append(analysis.Issues, "Using SELECT * may impact performance")
		analysis.Suggestions = append(analysis.Suggestions, "Specify only needed columns")
	}

	if !strings.Contains(strings.ToUpper(sql), "WHERE") && strings.Contains(strings.ToUpper(sql), "SELECT") {
		analysis.Issues = append(analysis.Issues, "Query lacks WHERE clause")
		analysis.Suggestions = append(analysis.Suggestions, "Consider adding WHERE clause to limit results")
	}

	return analysis
}

// SchemaExplorer explores database schema
type SchemaExplorer struct {
	dbClient *postgres.SQLCClient
	logger   *slog.Logger
}

func NewSchemaExplorer(dbClient *postgres.SQLCClient, logger *slog.Logger) *SchemaExplorer {
	return &SchemaExplorer{dbClient: dbClient, logger: logger}
}

func (s *SchemaExplorer) GetRelevantSchema(ctx context.Context, requirements []string) (*SchemaInfo, error) {
	// Simplified implementation - return basic schema info
	schema := &SchemaInfo{
		Tables:        make([]TableInfo, 0),
		Relationships: make([]RelationshipInfo, 0),
		Indexes:       make([]IndexInfo, 0),
		Constraints:   make([]ConstraintInfo, 0),
	}

	// Add some basic tables that are likely to exist
	schema.Tables = append(schema.Tables, TableInfo{
		Name: "conversations",
		Columns: []ColumnInfo{
			{Name: "id", Type: "UUID", Nullable: false},
			{Name: "user_id", Type: "VARCHAR", Nullable: false},
			{Name: "title", Type: "VARCHAR", Nullable: false},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
		},
		RowCount: 0,
	})

	schema.Tables = append(schema.Tables, TableInfo{
		Name: "messages",
		Columns: []ColumnInfo{
			{Name: "id", Type: "UUID", Nullable: false},
			{Name: "conversation_id", Type: "UUID", Nullable: false},
			{Name: "role", Type: "VARCHAR", Nullable: false},
			{Name: "content", Type: "TEXT", Nullable: false},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
		},
		RowCount: 0,
	})

	return schema, nil
}

func (s *SchemaExplorer) ExploreSchema(ctx context.Context, scope string) (*SchemaInfo, error) {
	// Simplified implementation based on scope
	return s.GetRelevantSchema(ctx, []string{scope})
}
