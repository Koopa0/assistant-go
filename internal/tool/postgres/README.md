# PostgreSQL Tool

The PostgreSQL tool provides comprehensive database management and optimization capabilities for the Assistant. It offers query analysis, migration validation, performance monitoring, and schema optimization features.

## Features

### 1. Query Analysis (`analyze_query`)
- Static SQL query analysis without database connection
- Identifies potential performance issues
- Detects common anti-patterns (SELECT *, missing WHERE clauses, etc.)
- Analyzes JOIN operations and complexity
- Suggests query improvements

### 2. Query Optimization (`optimize_query`)
- Provides specific optimization suggestions
- Identifies missing indexes
- Suggests query rewrites for better performance
- Analyzes subquery usage

### 3. EXPLAIN Analysis (`explain_query`)
- Static analysis when no connection available
- Full EXPLAIN ANALYZE with database connection
- Parse and interpret execution plans
- Identify performance bottlenecks

### 4. Migration Generation (`generate_migration`)
- Generate SQL migrations for common operations
- Supports CREATE TABLE, ADD COLUMN, ADD INDEX, etc.
- Follows PostgreSQL best practices
- Includes proper constraints and indexes

### 5. Migration Validation (`validate_migration`)
- Syntax validation
- Safety checks (destructive operations, locking)
- Best practice validation
- Reversibility analysis
- Provides specific warnings and suggestions

### 6. Schema Analysis (`analyze_schema`)
- Comprehensive schema review
- Table and index statistics
- Foreign key relationship mapping
- Identifies missing indexes and constraints
- Size and performance analysis

### 7. Index Suggestions (`suggest_indexes`)
- Analyzes query patterns
- Identifies missing indexes on foreign keys
- Detects unused indexes
- Provides specific CREATE INDEX statements

### 8. Performance Monitoring (`check_performance`)
- Database connection statistics
- Query performance metrics (requires pg_stat_statements)
- Table and index usage statistics
- Cache hit rates
- Lock monitoring
- Identifies performance issues and bottlenecks

## Usage

### Without Database Connection

Many features work without a database connection, providing static analysis:

```go
tool := postgres.NewPostgresTool(logger)

input := &tools.ToolInput{
    Parameters: map[string]interface{}{
        "action": "analyze_query",
        "query":  "SELECT * FROM users WHERE email = 'user@example.com'",
    },
}

result, err := tool.Execute(ctx, input)
```

### With Database Connection

For full functionality, provide a connection string:

```go
input := &tools.ToolInput{
    Parameters: map[string]interface{}{
        "action":            "check_performance",
        "connection_string": "postgres://user:password@localhost:5432/mydb",
    },
}
```

## Actions

### analyze_query
- **Required**: `query` (string) - SQL query to analyze
- **Returns**: Query analysis with issues, suggestions, and complexity assessment

### optimize_query
- **Required**: `query` (string) - SQL query to optimize
- **Returns**: Optimization suggestions and query rewrites

### explain_query
- **Required**: `query` (string) - SQL query to explain
- **Optional**: `connection_string` (string) - For actual EXPLAIN ANALYZE
- **Returns**: Execution plan analysis

### generate_migration
- **Required**: `migration_type` (string) - Type of migration
- **Optional**: `table` (string), `schema` (string), `columns` (array)
- **Returns**: Generated SQL migration

### validate_migration
- **Required**: `migration` (string) - SQL migration to validate
- **Returns**: Validation results with issues, warnings, and suggestions

### analyze_schema
- **Optional**: `schema` (string) - Schema name (default: "public")
- **Optional**: `connection_string` (string) - For live analysis
- **Returns**: Comprehensive schema analysis

### suggest_indexes
- **Optional**: `table` (string) - Specific table to analyze
- **Optional**: `connection_string` (string) - For query pattern analysis
- **Returns**: Index suggestions with CREATE INDEX statements

### check_performance
- **Required**: `connection_string` (string) - Database connection
- **Returns**: Performance metrics and recommendations

## Best Practices

1. **Start with Static Analysis**: Use query analysis and validation without connection first
2. **Test Migrations**: Always validate migrations before applying
3. **Monitor Performance**: Regular performance checks help prevent issues
4. **Review Suggestions**: Tool suggestions should be reviewed in context
5. **Use CONCURRENTLY**: For index creation on production databases

## Requirements

- PostgreSQL 11+ (recommended: PostgreSQL 15+)
- For full performance analysis: `pg_stat_statements` extension
- For vector operations: `pgvector` extension

## Security

- Connection strings should be stored securely
- Tool provides read-only analysis by default
- Migration execution is not automated - manual review required
- Sensitive data in queries is not logged