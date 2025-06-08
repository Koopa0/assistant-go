package postgres

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// MigrationGenerator generates PostgreSQL migrations
type MigrationGenerator struct {
	logger *slog.Logger
}

// NewMigrationGenerator creates a new migration generator
func NewMigrationGenerator(logger *slog.Logger) *MigrationGenerator {
	return &MigrationGenerator{
		logger: logger,
	}
}

// MigrationParams contains parameters for migration generation
type MigrationParams struct {
	Type      string        `json:"type"`
	TableName string        `json:"table_name"`
	Schema    string        `json:"schema"`
	Columns   []interface{} `json:"columns,omitempty"`
	Indexes   []IndexDef    `json:"indexes,omitempty"`
}

// ColumnDef defines a column for table creation
type ColumnDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	NotNull     bool   `json:"not_null"`
	PrimaryKey  bool   `json:"primary_key"`
	Unique      bool   `json:"unique"`
	Default     string `json:"default,omitempty"`
	References  string `json:"references,omitempty"`
	OnDelete    string `json:"on_delete,omitempty"`
	OnUpdate    string `json:"on_update,omitempty"`
	Check       string `json:"check,omitempty"`
	GeneratedAs string `json:"generated_as,omitempty"`
}

// IndexDef defines an index
type IndexDef struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Type    string   `json:"type"` // btree, gin, gist, etc.
	Where   string   `json:"where,omitempty"`
	Include []string `json:"include,omitempty"`
}

// MigrationResult contains generated migration
type MigrationResult struct {
	UpSQL       string    `json:"up_sql"`
	DownSQL     string    `json:"down_sql"`
	Filename    string    `json:"filename"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Notes       []string  `json:"notes"`
}

// Generate creates a migration based on parameters
func (g *MigrationGenerator) Generate(params MigrationParams) (*MigrationResult, error) {
	timestamp := time.Now()
	result := &MigrationResult{
		Timestamp: timestamp,
		Notes:     []string{},
	}

	// Set default schema if not provided
	if params.Schema == "" {
		params.Schema = "public"
	}

	switch params.Type {
	case "create_table":
		return g.generateCreateTable(params, result)
	case "add_column":
		return g.generateAddColumn(params, result)
	case "add_index":
		return g.generateAddIndex(params, result)
	case "alter_table":
		return g.generateAlterTable(params, result)
	default:
		// Try to generate based on provided SQL
		return g.generateCustom(params, result)
	}
}

// generateCreateTable generates a CREATE TABLE migration
func (g *MigrationGenerator) generateCreateTable(params MigrationParams, result *MigrationResult) (*MigrationResult, error) {
	if params.TableName == "" {
		return nil, fmt.Errorf("table_name is required for create_table migration")
	}

	var upBuilder strings.Builder
	var downBuilder strings.Builder

	// Generate description
	result.Description = fmt.Sprintf("Create table %s.%s", params.Schema, params.TableName)
	result.Filename = fmt.Sprintf("%s_create_%s_table.sql",
		result.Timestamp.Format("20060102150405"), params.TableName)

	// Build CREATE TABLE statement
	upBuilder.WriteString(fmt.Sprintf("-- Create %s table\n", params.TableName))
	upBuilder.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (\n", params.Schema, params.TableName))

	// Parse columns
	columns := g.parseColumns(params.Columns)
	var columnDefs []string
	var primaryKeys []string

	for _, col := range columns {
		colDef := g.buildColumnDefinition(col)
		columnDefs = append(columnDefs, colDef)
		if col.PrimaryKey {
			primaryKeys = append(primaryKeys, col.Name)
		}
	}

	// Add column definitions
	upBuilder.WriteString("    " + strings.Join(columnDefs, ",\n    "))

	// Add primary key constraint if needed
	if len(primaryKeys) > 0 {
		upBuilder.WriteString(",\n")
		upBuilder.WriteString(fmt.Sprintf("    PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}

	upBuilder.WriteString("\n);\n")

	// Add comments
	upBuilder.WriteString(fmt.Sprintf("\n-- Add table comment\n"))
	upBuilder.WriteString(fmt.Sprintf("COMMENT ON TABLE %s.%s IS '%s table for the application';\n",
		params.Schema, params.TableName, params.TableName))

	// Add column comments
	for _, col := range columns {
		upBuilder.WriteString(fmt.Sprintf("COMMENT ON COLUMN %s.%s.%s IS '%s';\n",
			params.Schema, params.TableName, col.Name, g.generateColumnComment(col)))
	}

	// Add indexes
	for _, idx := range params.Indexes {
		upBuilder.WriteString("\n")
		upBuilder.WriteString(g.buildIndexDefinition(params.Schema, params.TableName, idx))
	}

	// Build DROP statement
	downBuilder.WriteString(fmt.Sprintf("-- Drop %s table\n", params.TableName))
	downBuilder.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s.%s CASCADE;\n", params.Schema, params.TableName))

	result.UpSQL = upBuilder.String()
	result.DownSQL = downBuilder.String()

	// Add notes
	result.Notes = append(result.Notes, "Remember to add appropriate indexes for foreign keys")
	result.Notes = append(result.Notes, "Consider adding CHECK constraints for data validation")
	result.Notes = append(result.Notes, "Review default values and NULL constraints")

	return result, nil
}

// generateAddColumn generates an ADD COLUMN migration
func (g *MigrationGenerator) generateAddColumn(params MigrationParams, result *MigrationResult) (*MigrationResult, error) {
	if params.TableName == "" {
		return nil, fmt.Errorf("table_name is required for add_column migration")
	}

	columns := g.parseColumns(params.Columns)
	if len(columns) == 0 {
		return nil, fmt.Errorf("at least one column definition is required")
	}

	var upBuilder strings.Builder
	var downBuilder strings.Builder

	// Generate description
	columnNames := []string{}
	for _, col := range columns {
		columnNames = append(columnNames, col.Name)
	}
	result.Description = fmt.Sprintf("Add columns %s to %s.%s",
		strings.Join(columnNames, ", "), params.Schema, params.TableName)
	result.Filename = fmt.Sprintf("%s_add_%s_to_%s.sql",
		result.Timestamp.Format("20060102150405"), strings.Join(columnNames, "_"), params.TableName)

	// Build ALTER TABLE statements
	upBuilder.WriteString(fmt.Sprintf("-- Add columns to %s\n", params.TableName))

	for _, col := range columns {
		upBuilder.WriteString(fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN %s;\n",
			params.Schema, params.TableName, g.buildColumnDefinition(col)))

		// Add comment
		upBuilder.WriteString(fmt.Sprintf("COMMENT ON COLUMN %s.%s.%s IS '%s';\n\n",
			params.Schema, params.TableName, col.Name, g.generateColumnComment(col)))

		// Build drop statement
		downBuilder.WriteString(fmt.Sprintf("ALTER TABLE %s.%s DROP COLUMN IF EXISTS %s;\n",
			params.Schema, params.TableName, col.Name))
	}

	result.UpSQL = upBuilder.String()
	result.DownSQL = downBuilder.String()

	// Add notes
	if g.hasNotNullColumns(columns) {
		result.Notes = append(result.Notes, "NOT NULL columns require default values or the table must be empty")
	}
	result.Notes = append(result.Notes, "Consider the impact on existing queries and application code")

	return result, nil
}

// generateAddIndex generates an ADD INDEX migration
func (g *MigrationGenerator) generateAddIndex(params MigrationParams, result *MigrationResult) (*MigrationResult, error) {
	if params.TableName == "" {
		return nil, fmt.Errorf("table_name is required for add_index migration")
	}

	if len(params.Indexes) == 0 {
		return nil, fmt.Errorf("at least one index definition is required")
	}

	var upBuilder strings.Builder
	var downBuilder strings.Builder

	// Generate description
	result.Description = fmt.Sprintf("Add indexes to %s.%s", params.Schema, params.TableName)
	result.Filename = fmt.Sprintf("%s_add_indexes_to_%s.sql",
		result.Timestamp.Format("20060102150405"), params.TableName)

	upBuilder.WriteString(fmt.Sprintf("-- Add indexes to %s\n", params.TableName))

	for _, idx := range params.Indexes {
		upBuilder.WriteString(g.buildIndexDefinition(params.Schema, params.TableName, idx))
		upBuilder.WriteString("\n")

		// Build drop statement
		indexName := idx.Name
		if indexName == "" {
			indexName = g.generateIndexName(params.TableName, idx)
		}
		downBuilder.WriteString(fmt.Sprintf("DROP INDEX IF EXISTS %s.%s;\n", params.Schema, indexName))
	}

	result.UpSQL = upBuilder.String()
	result.DownSQL = downBuilder.String()

	// Add notes
	result.Notes = append(result.Notes, "Monitor index creation time on large tables")
	result.Notes = append(result.Notes, "Consider using CONCURRENTLY for indexes on production tables")
	result.Notes = append(result.Notes, "Review query plans after index creation to ensure usage")

	return result, nil
}

// generateAlterTable generates a generic ALTER TABLE migration
func (g *MigrationGenerator) generateAlterTable(params MigrationParams, result *MigrationResult) (*MigrationResult, error) {
	if params.TableName == "" {
		return nil, fmt.Errorf("table_name is required for alter_table migration")
	}

	result.Description = fmt.Sprintf("Alter table %s.%s", params.Schema, params.TableName)
	result.Filename = fmt.Sprintf("%s_alter_%s.sql",
		result.Timestamp.Format("20060102150405"), params.TableName)

	// Template for common alterations
	var upBuilder strings.Builder
	var downBuilder strings.Builder

	upBuilder.WriteString(fmt.Sprintf("-- Alter table %s\n", params.TableName))
	upBuilder.WriteString(fmt.Sprintf("-- TODO: Add ALTER TABLE statements here\n"))
	upBuilder.WriteString(fmt.Sprintf("-- Examples:\n"))
	upBuilder.WriteString(fmt.Sprintf("-- ALTER TABLE %s.%s ADD CONSTRAINT check_name CHECK (condition);\n", params.Schema, params.TableName))
	upBuilder.WriteString(fmt.Sprintf("-- ALTER TABLE %s.%s ALTER COLUMN column_name SET NOT NULL;\n", params.Schema, params.TableName))
	upBuilder.WriteString(fmt.Sprintf("-- ALTER TABLE %s.%s ADD CONSTRAINT fk_name FOREIGN KEY (column) REFERENCES other_table(id);\n", params.Schema, params.TableName))

	downBuilder.WriteString(fmt.Sprintf("-- Revert alterations to %s\n", params.TableName))
	downBuilder.WriteString(fmt.Sprintf("-- TODO: Add reversal statements here\n"))

	result.UpSQL = upBuilder.String()
	result.DownSQL = downBuilder.String()

	result.Notes = append(result.Notes, "Fill in the specific ALTER TABLE statements needed")
	result.Notes = append(result.Notes, "Ensure down migration properly reverts changes")

	return result, nil
}

// generateCustom generates a custom migration template
func (g *MigrationGenerator) generateCustom(params MigrationParams, result *MigrationResult) (*MigrationResult, error) {
	result.Description = "Custom migration"
	result.Filename = fmt.Sprintf("%s_custom_migration.sql",
		result.Timestamp.Format("20060102150405"))

	var upBuilder strings.Builder
	var downBuilder strings.Builder

	upBuilder.WriteString("-- Custom migration\n")
	upBuilder.WriteString("-- TODO: Add your migration SQL here\n\n")
	upBuilder.WriteString("BEGIN;\n\n")
	upBuilder.WriteString("-- Your SQL statements here\n\n")
	upBuilder.WriteString("COMMIT;\n")

	downBuilder.WriteString("-- Rollback custom migration\n")
	downBuilder.WriteString("-- TODO: Add rollback SQL here\n\n")
	downBuilder.WriteString("BEGIN;\n\n")
	downBuilder.WriteString("-- Your rollback SQL here\n\n")
	downBuilder.WriteString("COMMIT;\n")

	result.UpSQL = upBuilder.String()
	result.DownSQL = downBuilder.String()

	result.Notes = append(result.Notes, "Remember to test both up and down migrations")
	result.Notes = append(result.Notes, "Use transactions to ensure atomicity")
	result.Notes = append(result.Notes, "Consider the impact on application availability")

	return result, nil
}

// Helper functions

// parseColumns converts interface{} to []ColumnDef
func (g *MigrationGenerator) parseColumns(cols []interface{}) []ColumnDef {
	var columns []ColumnDef
	for _, col := range cols {
		if colMap, ok := col.(map[string]interface{}); ok {
			colDef := ColumnDef{
				Name: g.getString(colMap, "name"),
				Type: g.getString(colMap, "type"),
			}
			if notNull, ok := colMap["not_null"].(bool); ok {
				colDef.NotNull = notNull
			}
			if pk, ok := colMap["primary_key"].(bool); ok {
				colDef.PrimaryKey = pk
			}
			if unique, ok := colMap["unique"].(bool); ok {
				colDef.Unique = unique
			}
			colDef.Default = g.getString(colMap, "default")
			colDef.References = g.getString(colMap, "references")
			colDef.Check = g.getString(colMap, "check")
			columns = append(columns, colDef)
		}
	}
	return columns
}

// getString safely extracts string from map
func (g *MigrationGenerator) getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// buildColumnDefinition builds SQL column definition
func (g *MigrationGenerator) buildColumnDefinition(col ColumnDef) string {
	parts := []string{col.Name, col.Type}

	if col.NotNull {
		parts = append(parts, "NOT NULL")
	}

	if col.Unique && !col.PrimaryKey {
		parts = append(parts, "UNIQUE")
	}

	if col.Default != "" {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", col.Default))
	}

	if col.Check != "" {
		parts = append(parts, fmt.Sprintf("CHECK (%s)", col.Check))
	}

	if col.References != "" {
		refParts := []string{fmt.Sprintf("REFERENCES %s", col.References)}
		if col.OnDelete != "" {
			refParts = append(refParts, fmt.Sprintf("ON DELETE %s", col.OnDelete))
		}
		if col.OnUpdate != "" {
			refParts = append(refParts, fmt.Sprintf("ON UPDATE %s", col.OnUpdate))
		}
		parts = append(parts, strings.Join(refParts, " "))
	}

	if col.GeneratedAs != "" {
		parts = append(parts, fmt.Sprintf("GENERATED ALWAYS AS (%s) STORED", col.GeneratedAs))
	}

	return strings.Join(parts, " ")
}

// buildIndexDefinition builds CREATE INDEX statement
func (g *MigrationGenerator) buildIndexDefinition(schema, table string, idx IndexDef) string {
	var parts []string

	// Generate index name if not provided
	indexName := idx.Name
	if indexName == "" {
		indexName = g.generateIndexName(table, idx)
	}

	if idx.Unique {
		parts = append(parts, "CREATE UNIQUE INDEX")
	} else {
		parts = append(parts, "CREATE INDEX")
	}

	parts = append(parts, indexName)
	parts = append(parts, "ON")
	parts = append(parts, fmt.Sprintf("%s.%s", schema, table))

	// Index type
	if idx.Type != "" && idx.Type != "btree" {
		parts = append(parts, "USING", idx.Type)
	}

	// Columns
	parts = append(parts, fmt.Sprintf("(%s)", strings.Join(idx.Columns, ", ")))

	// Include columns (PostgreSQL 11+)
	if len(idx.Include) > 0 {
		parts = append(parts, fmt.Sprintf("INCLUDE (%s)", strings.Join(idx.Include, ", ")))
	}

	// Partial index
	if idx.Where != "" {
		parts = append(parts, "WHERE", idx.Where)
	}

	return strings.Join(parts, " ") + ";"
}

// generateIndexName generates a descriptive index name
func (g *MigrationGenerator) generateIndexName(table string, idx IndexDef) string {
	prefix := "idx"
	if idx.Unique {
		prefix = "uniq"
	}

	// Shorten column names if too long
	colPart := strings.Join(idx.Columns, "_")
	if len(colPart) > 20 {
		colPart = colPart[:20]
	}

	return fmt.Sprintf("%s_%s_%s", prefix, table, colPart)
}

// generateColumnComment generates a descriptive comment for a column
func (g *MigrationGenerator) generateColumnComment(col ColumnDef) string {
	parts := []string{col.Name}

	if col.PrimaryKey {
		parts = append(parts, "(Primary Key)")
	}
	if col.Unique {
		parts = append(parts, "(Unique)")
	}
	if col.References != "" {
		parts = append(parts, fmt.Sprintf("references %s", col.References))
	}

	return strings.Join(parts, " ")
}

// hasNotNullColumns checks if any columns are NOT NULL
func (g *MigrationGenerator) hasNotNullColumns(columns []ColumnDef) bool {
	for _, col := range columns {
		if col.NotNull && col.Default == "" {
			return true
		}
	}
	return false
}
