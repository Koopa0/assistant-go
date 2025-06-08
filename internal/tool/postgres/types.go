package postgres

import "fmt"

// PostgresInput represents typed input parameters for PostgreSQL tool
type PostgresInput struct {
	Action           string `json:"action"`
	Query            string `json:"query,omitempty"`
	Schema           string `json:"schema,omitempty"`
	Table            string `json:"table,omitempty"`
	MigrationType    string `json:"migration_type,omitempty"`
	ConnectionString string `json:"connection_string,omitempty"`
}

// Validate checks if the input parameters are valid
func (p *PostgresInput) Validate() error {
	if p.Action == "" {
		return fmt.Errorf("action is required")
	}

	// Validate action-specific requirements
	switch p.Action {
	case "analyze_query", "optimize_query", "explain_query":
		if p.Query == "" {
			return fmt.Errorf("query is required for %s action", p.Action)
		}
	case "analyze_schema":
		if p.Schema == "" {
			return fmt.Errorf("schema is required for analyze_schema action")
		}
	case "suggest_indexes":
		if p.Table == "" {
			return fmt.Errorf("table is required for suggest_indexes action")
		}
	case "generate_migration":
		if p.MigrationType == "" {
			return fmt.Errorf("migration_type is required for generate_migration action")
		}
	}

	return nil
}

// ToMap converts the typed parameters to map for backward compatibility
func (p *PostgresInput) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"action": p.Action,
	}

	if p.Query != "" {
		m["query"] = p.Query
	}
	if p.Schema != "" {
		m["schema"] = p.Schema
	}
	if p.Table != "" {
		m["table"] = p.Table
	}
	if p.MigrationType != "" {
		m["migration_type"] = p.MigrationType
	}
	if p.ConnectionString != "" {
		m["connection_string"] = p.ConnectionString
	}

	return m
}
