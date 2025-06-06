package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SchemaAnalyzer analyzes PostgreSQL database schemas
type SchemaAnalyzer struct {
	logger *slog.Logger
	pool   *pgxpool.Pool
}

// NewSchemaAnalyzer creates a new schema analyzer
func NewSchemaAnalyzer(logger *slog.Logger, pool *pgxpool.Pool) *SchemaAnalyzer {
	return &SchemaAnalyzer{
		logger: logger,
		pool:   pool,
	}
}

// SchemaAnalysis represents the analysis of a database schema
type SchemaAnalysis struct {
	SchemaName      string             `json:"schema_name"`
	Tables          []TableInfo        `json:"tables"`
	Indexes         []IndexInfo        `json:"indexes"`
	Constraints     []ConstraintInfo   `json:"constraints"`
	Relationships   []RelationshipInfo `json:"relationships"`
	Issues          []SchemaIssue      `json:"issues"`
	Recommendations []string           `json:"recommendations"`
	Statistics      SchemaStatistics   `json:"statistics"`
}

// TableInfo contains information about a table
type TableInfo struct {
	Name          string       `json:"name"`
	Columns       []ColumnInfo `json:"columns"`
	RowCount      int64        `json:"row_count"`
	SizeBytes     int64        `json:"size_bytes"`
	IndexCount    int          `json:"index_count"`
	HasPrimaryKey bool         `json:"has_primary_key"`
	Description   string       `json:"description,omitempty"`
}

// ColumnInfo contains information about a column
type ColumnInfo struct {
	Name         string `json:"name"`
	DataType     string `json:"data_type"`
	IsNullable   bool   `json:"is_nullable"`
	DefaultValue string `json:"default_value,omitempty"`
	IsPrimaryKey bool   `json:"is_primary_key"`
	IsForeignKey bool   `json:"is_foreign_key"`
	IsUnique     bool   `json:"is_unique"`
	Description  string `json:"description,omitempty"`
}

// IndexInfo contains information about an index
type IndexInfo struct {
	Name       string   `json:"name"`
	TableName  string   `json:"table_name"`
	Columns    []string `json:"columns"`
	IndexType  string   `json:"index_type"`
	IsUnique   bool     `json:"is_unique"`
	IsPrimary  bool     `json:"is_primary"`
	IsPartial  bool     `json:"is_partial"`
	SizeBytes  int64    `json:"size_bytes"`
	UsageCount int64    `json:"usage_count"`
}

// ConstraintInfo contains information about a constraint
type ConstraintInfo struct {
	Name           string `json:"name"`
	TableName      string `json:"table_name"`
	ConstraintType string `json:"constraint_type"`
	Definition     string `json:"definition"`
}

// RelationshipInfo contains information about table relationships
type RelationshipInfo struct {
	ParentTable    string   `json:"parent_table"`
	ParentColumns  []string `json:"parent_columns"`
	ChildTable     string   `json:"child_table"`
	ChildColumns   []string `json:"child_columns"`
	ConstraintName string   `json:"constraint_name"`
	OnDelete       string   `json:"on_delete"`
	OnUpdate       string   `json:"on_update"`
}

// SchemaIssue represents an issue found in the schema
type SchemaIssue struct {
	Severity   string `json:"severity"` // "error", "warning", "info"
	Category   string `json:"category"`
	Table      string `json:"table,omitempty"`
	Column     string `json:"column,omitempty"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

// SchemaStatistics contains overall schema statistics
type SchemaStatistics struct {
	TotalTables      int   `json:"total_tables"`
	TotalIndexes     int   `json:"total_indexes"`
	TotalConstraints int   `json:"total_constraints"`
	TotalSizeBytes   int64 `json:"total_size_bytes"`
	TotalRows        int64 `json:"total_rows"`
	UnusedIndexes    int   `json:"unused_indexes"`
	MissingIndexes   int   `json:"missing_indexes"`
}

// Analyze performs a comprehensive schema analysis
func (a *SchemaAnalyzer) Analyze(ctx context.Context, schemaName string) (*SchemaAnalysis, error) {
	if a.pool == nil {
		return a.staticAnalysis(schemaName), nil
	}

	analysis := &SchemaAnalysis{
		SchemaName:      schemaName,
		Tables:          []TableInfo{},
		Indexes:         []IndexInfo{},
		Constraints:     []ConstraintInfo{},
		Relationships:   []RelationshipInfo{},
		Issues:          []SchemaIssue{},
		Recommendations: []string{},
	}

	// Analyze tables
	if err := a.analyzeTables(ctx, schemaName, analysis); err != nil {
		return nil, fmt.Errorf("failed to analyze tables: %w", err)
	}

	// Analyze indexes
	if err := a.analyzeIndexes(ctx, schemaName, analysis); err != nil {
		return nil, fmt.Errorf("failed to analyze indexes: %w", err)
	}

	// Analyze constraints
	if err := a.analyzeConstraints(ctx, schemaName, analysis); err != nil {
		return nil, fmt.Errorf("failed to analyze constraints: %w", err)
	}

	// Analyze relationships
	if err := a.analyzeRelationships(ctx, schemaName, analysis); err != nil {
		return nil, fmt.Errorf("failed to analyze relationships: %w", err)
	}

	// Check for issues
	a.checkForIssues(analysis)

	// Generate recommendations
	a.generateRecommendations(analysis)

	// Calculate statistics
	a.calculateStatistics(analysis)

	return analysis, nil
}

// staticAnalysis provides analysis without database connection
func (a *SchemaAnalyzer) staticAnalysis(schemaName string) *SchemaAnalysis {
	return &SchemaAnalysis{
		SchemaName: schemaName,
		Issues: []SchemaIssue{
			{
				Severity:   "info",
				Category:   "connection",
				Message:    "Schema analysis running without database connection",
				Suggestion: "Connect to database for comprehensive analysis",
			},
		},
		Recommendations: []string{
			"Use naming conventions: tables (plural), columns (snake_case)",
			"Add comments to tables and columns for documentation",
			"Create indexes on foreign key columns",
			"Use appropriate data types (avoid VARCHAR(255) everywhere)",
			"Consider partitioning for large tables",
			"Implement proper constraints for data integrity",
		},
	}
}

// analyzeTables analyzes all tables in the schema
func (a *SchemaAnalyzer) analyzeTables(ctx context.Context, schemaName string, analysis *SchemaAnalysis) error {
	query := `
		SELECT 
			t.tablename,
			pg_stat_user_tables.n_live_tup as row_count,
			pg_total_relation_size(quote_ident(t.schemaname)||'.'||quote_ident(t.tablename)) as size_bytes,
			obj_description(c.oid, 'pg_class') as description
		FROM pg_tables t
		JOIN pg_class c ON c.relname = t.tablename AND c.relnamespace = (
			SELECT oid FROM pg_namespace WHERE nspname = t.schemaname
		)
		LEFT JOIN pg_stat_user_tables ON 
			pg_stat_user_tables.schemaname = t.schemaname AND 
			pg_stat_user_tables.relname = t.tablename
		WHERE t.schemaname = $1
		ORDER BY t.tablename`

	rows, err := a.pool.Query(ctx, query, schemaName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var table TableInfo
		var rowCount, sizeBytes *int64
		var description *string

		if err := rows.Scan(&table.Name, &rowCount, &sizeBytes, &description); err != nil {
			return err
		}

		if rowCount != nil {
			table.RowCount = *rowCount
		}
		if sizeBytes != nil {
			table.SizeBytes = *sizeBytes
		}
		if description != nil {
			table.Description = *description
		}

		// Analyze columns for this table
		if err := a.analyzeTableColumns(ctx, schemaName, &table); err != nil {
			a.logger.Warn("Failed to analyze columns",
				slog.String("table", table.Name),
				slog.String("error", err.Error()))
		}

		analysis.Tables = append(analysis.Tables, table)
	}

	return rows.Err()
}

// analyzeTableColumns analyzes columns for a specific table
func (a *SchemaAnalyzer) analyzeTableColumns(ctx context.Context, schemaName string, table *TableInfo) error {
	query := `
		SELECT 
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' as is_nullable,
			c.column_default,
			EXISTS(
				SELECT 1 FROM information_schema.key_column_usage k
				WHERE k.table_schema = c.table_schema 
				AND k.table_name = c.table_name 
				AND k.column_name = c.column_name
				AND k.constraint_name LIKE '%_pkey'
			) as is_primary_key,
			EXISTS(
				SELECT 1 FROM information_schema.key_column_usage k
				WHERE k.table_schema = c.table_schema 
				AND k.table_name = c.table_name 
				AND k.column_name = c.column_name
				AND k.constraint_name LIKE '%_fkey'
			) as is_foreign_key,
			pgd.description
		FROM information_schema.columns c
		LEFT JOIN pg_catalog.pg_description pgd ON 
			pgd.objoid = (
				SELECT oid FROM pg_class 
				WHERE relname = c.table_name 
				AND relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = c.table_schema)
			) AND pgd.objsubid = c.ordinal_position
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position`

	rows, err := a.pool.Query(ctx, query, schemaName, table.Name)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnInfo
		var defaultValue, description *string

		if err := rows.Scan(
			&col.Name,
			&col.DataType,
			&col.IsNullable,
			&defaultValue,
			&col.IsPrimaryKey,
			&col.IsForeignKey,
			&description,
		); err != nil {
			return err
		}

		if defaultValue != nil {
			col.DefaultValue = *defaultValue
		}
		if description != nil {
			col.Description = *description
		}

		table.Columns = append(table.Columns, col)

		if col.IsPrimaryKey {
			table.HasPrimaryKey = true
		}
	}

	return rows.Err()
}

// analyzeIndexes analyzes all indexes in the schema
func (a *SchemaAnalyzer) analyzeIndexes(ctx context.Context, schemaName string, analysis *SchemaAnalysis) error {
	query := `
		SELECT 
			i.indexname,
			i.tablename,
			string_to_array(replace(replace(split_part(indexdef, '(', 2), ')', ''), ' ', ''), ',') as columns,
			am.amname as index_type,
			ix.indisunique,
			ix.indisprimary,
			ix.indpred IS NOT NULL as is_partial,
			pg_relation_size(c.oid) as size_bytes,
			COALESCE(pg_stat_user_indexes.idx_scan, 0) as usage_count
		FROM pg_indexes i
		JOIN pg_class c ON c.relname = i.indexname
		JOIN pg_index ix ON ix.indexrelid = c.oid
		JOIN pg_am am ON am.oid = c.relam
		LEFT JOIN pg_stat_user_indexes ON 
			pg_stat_user_indexes.schemaname = i.schemaname AND
			pg_stat_user_indexes.indexrelname = i.indexname
		WHERE i.schemaname = $1
		ORDER BY i.tablename, i.indexname`

	rows, err := a.pool.Query(ctx, query, schemaName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var idx IndexInfo
		var columns []string

		if err := rows.Scan(
			&idx.Name,
			&idx.TableName,
			&columns,
			&idx.IndexType,
			&idx.IsUnique,
			&idx.IsPrimary,
			&idx.IsPartial,
			&idx.SizeBytes,
			&idx.UsageCount,
		); err != nil {
			return err
		}

		idx.Columns = columns
		analysis.Indexes = append(analysis.Indexes, idx)

		// Update table index count
		for i := range analysis.Tables {
			if analysis.Tables[i].Name == idx.TableName {
				analysis.Tables[i].IndexCount++
				break
			}
		}
	}

	return rows.Err()
}

// analyzeConstraints analyzes all constraints in the schema
func (a *SchemaAnalyzer) analyzeConstraints(ctx context.Context, schemaName string, analysis *SchemaAnalysis) error {
	query := `
		SELECT 
			tc.constraint_name,
			tc.table_name,
			tc.constraint_type,
			pg_get_constraintdef(pgc.oid) as definition
		FROM information_schema.table_constraints tc
		JOIN pg_constraint pgc ON pgc.conname = tc.constraint_name
		AND pgc.connamespace = (SELECT oid FROM pg_namespace WHERE nspname = tc.table_schema)
		WHERE tc.table_schema = $1
		AND tc.constraint_type IN ('CHECK', 'UNIQUE', 'EXCLUDE')
		ORDER BY tc.table_name, tc.constraint_name`

	rows, err := a.pool.Query(ctx, query, schemaName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var constraint ConstraintInfo

		if err := rows.Scan(
			&constraint.Name,
			&constraint.TableName,
			&constraint.ConstraintType,
			&constraint.Definition,
		); err != nil {
			return err
		}

		analysis.Constraints = append(analysis.Constraints, constraint)
	}

	return rows.Err()
}

// analyzeRelationships analyzes foreign key relationships
func (a *SchemaAnalyzer) analyzeRelationships(ctx context.Context, schemaName string, analysis *SchemaAnalysis) error {
	query := `
		SELECT 
			tc.constraint_name,
			tc.table_name as child_table,
			string_agg(kcu.column_name, ',' ORDER BY kcu.ordinal_position) as child_columns,
			ccu.table_name as parent_table,
			string_agg(ccu.column_name, ',' ORDER BY kcu.ordinal_position) as parent_columns,
			rc.delete_rule as on_delete,
			rc.update_rule as on_update
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_schema = kcu.constraint_schema
			AND tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_schema = ccu.constraint_schema
			AND tc.constraint_name = ccu.constraint_name
		JOIN information_schema.referential_constraints rc
			ON tc.constraint_schema = rc.constraint_schema
			AND tc.constraint_name = rc.constraint_name
		WHERE tc.table_schema = $1
		AND tc.constraint_type = 'FOREIGN KEY'
		GROUP BY tc.constraint_name, tc.table_name, ccu.table_name, rc.delete_rule, rc.update_rule
		ORDER BY tc.table_name, tc.constraint_name`

	rows, err := a.pool.Query(ctx, query, schemaName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var rel RelationshipInfo
		var childCols, parentCols string

		if err := rows.Scan(
			&rel.ConstraintName,
			&rel.ChildTable,
			&childCols,
			&rel.ParentTable,
			&parentCols,
			&rel.OnDelete,
			&rel.OnUpdate,
		); err != nil {
			return err
		}

		rel.ChildColumns = strings.Split(childCols, ",")
		rel.ParentColumns = strings.Split(parentCols, ",")

		analysis.Relationships = append(analysis.Relationships, rel)
	}

	return rows.Err()
}

// checkForIssues checks for common schema issues
func (a *SchemaAnalyzer) checkForIssues(analysis *SchemaAnalysis) {
	// Check for tables without primary keys
	for _, table := range analysis.Tables {
		if !table.HasPrimaryKey {
			analysis.Issues = append(analysis.Issues, SchemaIssue{
				Severity:   "warning",
				Category:   "missing_primary_key",
				Table:      table.Name,
				Message:    fmt.Sprintf("Table '%s' has no primary key", table.Name),
				Suggestion: "Add a primary key to ensure row uniqueness and improve performance",
			})
		}

		// Check for missing table comments
		if table.Description == "" {
			analysis.Issues = append(analysis.Issues, SchemaIssue{
				Severity:   "info",
				Category:   "missing_documentation",
				Table:      table.Name,
				Message:    fmt.Sprintf("Table '%s' has no description", table.Name),
				Suggestion: "Add table comment for documentation",
			})
		}

		// Check for overly generic column names
		for _, col := range table.Columns {
			if col.Name == "data" || col.Name == "value" || col.Name == "info" {
				analysis.Issues = append(analysis.Issues, SchemaIssue{
					Severity:   "info",
					Category:   "naming",
					Table:      table.Name,
					Column:     col.Name,
					Message:    fmt.Sprintf("Column '%s.%s' has generic name", table.Name, col.Name),
					Suggestion: "Use more descriptive column names",
				})
			}
		}
	}

	// Check for unused indexes
	for _, idx := range analysis.Indexes {
		if !idx.IsPrimary && idx.UsageCount == 0 {
			analysis.Issues = append(analysis.Issues, SchemaIssue{
				Severity:   "warning",
				Category:   "unused_index",
				Table:      idx.TableName,
				Message:    fmt.Sprintf("Index '%s' has never been used", idx.Name),
				Suggestion: "Consider dropping unused indexes to save storage and improve write performance",
			})
		}
	}

	// Check for missing indexes on foreign keys
	fkColumns := make(map[string]bool)
	for _, rel := range analysis.Relationships {
		for _, col := range rel.ChildColumns {
			fkColumns[rel.ChildTable+"."+col] = true
		}
	}

	// Check if FK columns have indexes
	for fkCol := range fkColumns {
		parts := strings.Split(fkCol, ".")
		table, column := parts[0], parts[1]

		hasIndex := false
		for _, idx := range analysis.Indexes {
			if idx.TableName == table && len(idx.Columns) > 0 && idx.Columns[0] == column {
				hasIndex = true
				break
			}
		}

		if !hasIndex {
			analysis.Issues = append(analysis.Issues, SchemaIssue{
				Severity:   "warning",
				Category:   "missing_index",
				Table:      table,
				Column:     column,
				Message:    fmt.Sprintf("Foreign key column '%s.%s' lacks an index", table, column),
				Suggestion: "Create index on foreign key columns for better join performance",
			})
		}
	}
}

// generateRecommendations generates schema recommendations
func (a *SchemaAnalyzer) generateRecommendations(analysis *SchemaAnalysis) {
	// Size-based recommendations
	for _, table := range analysis.Tables {
		if table.RowCount > 10000000 { // 10M rows
			analysis.Recommendations = append(analysis.Recommendations,
				fmt.Sprintf("Consider partitioning table '%s' (%d rows) for better performance",
					table.Name, table.RowCount))
		}

		if table.SizeBytes > 1024*1024*1024*10 { // 10GB
			analysis.Recommendations = append(analysis.Recommendations,
				fmt.Sprintf("Table '%s' is large (%.2f GB). Review archival strategy",
					table.Name, float64(table.SizeBytes)/(1024*1024*1024)))
		}
	}

	// Index recommendations
	unusedCount := 0
	for _, idx := range analysis.Indexes {
		if !idx.IsPrimary && idx.UsageCount == 0 {
			unusedCount++
		}
	}
	if unusedCount > 0 {
		analysis.Recommendations = append(analysis.Recommendations,
			fmt.Sprintf("Found %d unused indexes. Review and drop unnecessary ones", unusedCount))
	}

	// General recommendations
	if len(analysis.Relationships) == 0 {
		analysis.Recommendations = append(analysis.Recommendations,
			"No foreign key relationships found. Consider adding referential integrity constraints")
	}

	if analysis.Statistics.TotalTables > 50 {
		analysis.Recommendations = append(analysis.Recommendations,
			"Large number of tables. Consider schema modularization or microservices approach")
	}
}

// calculateStatistics calculates overall schema statistics
func (a *SchemaAnalyzer) calculateStatistics(analysis *SchemaAnalysis) {
	stats := &analysis.Statistics
	stats.TotalTables = len(analysis.Tables)
	stats.TotalIndexes = len(analysis.Indexes)
	stats.TotalConstraints = len(analysis.Constraints)

	for _, table := range analysis.Tables {
		stats.TotalRows += table.RowCount
		stats.TotalSizeBytes += table.SizeBytes
	}

	for _, idx := range analysis.Indexes {
		if !idx.IsPrimary && idx.UsageCount == 0 {
			stats.UnusedIndexes++
		}
	}

	// Count issues by category
	for _, issue := range analysis.Issues {
		if issue.Category == "missing_index" {
			stats.MissingIndexes++
		}
	}
}
