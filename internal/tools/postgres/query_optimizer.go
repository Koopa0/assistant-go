package postgres

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// QueryOptimizer suggests optimizations for SQL queries
type QueryOptimizer struct {
	logger *slog.Logger
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(logger *slog.Logger) *QueryOptimizer {
	return &QueryOptimizer{
		logger: logger,
	}
}

// OptimizationResult represents the result of query optimization
type OptimizationResult struct {
	OriginalQuery    string               `json:"original_query"`
	OptimizedQuery   string               `json:"optimized_query"`
	Optimizations    []OptimizationDetail `json:"optimizations"`
	ExpectedBenefit  string               `json:"expected_benefit"`
	Warnings         []string             `json:"warnings"`
	IndexSuggestions []IndexSuggestion    `json:"index_suggestions"`
}

// OptimizationDetail represents a single optimization made
type OptimizationDetail struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Reasoning   string `json:"reasoning"`
	Impact      string `json:"impact"` // "high", "medium", "low"
}

// IndexSuggestion represents a suggested index
type IndexSuggestion struct {
	TableName   string   `json:"table_name"`
	Columns     []string `json:"columns"`
	IndexType   string   `json:"index_type"` // "btree", "gin", "gist", etc.
	Reasoning   string   `json:"reasoning"`
	CreateQuery string   `json:"create_query"`
}

// Optimize analyzes and optimizes a SQL query
func (o *QueryOptimizer) Optimize(query string) (*OptimizationResult, error) {
	result := &OptimizationResult{
		OriginalQuery:    query,
		OptimizedQuery:   query,
		Optimizations:    []OptimizationDetail{},
		Warnings:         []string{},
		IndexSuggestions: []IndexSuggestion{},
	}

	// Apply various optimizations
	o.optimizeSelectStar(result)
	o.optimizeJoins(result)
	o.optimizeSubqueries(result)
	o.optimizeInClauses(result)
	o.optimizePagination(result)
	o.optimizeDistinct(result)
	o.optimizeFunctions(result)
	o.optimizeExists(result)

	// Suggest indexes based on query patterns
	o.suggestIndexes(result)

	// Calculate expected benefit
	o.calculateBenefit(result)

	// Add warnings if no optimizations found
	if len(result.Optimizations) == 0 {
		result.Warnings = append(result.Warnings,
			"No automatic optimizations found. Consider manual review for complex logic optimization.")
	}

	return result, nil
}

// optimizeSelectStar replaces SELECT * with specific columns
func (o *QueryOptimizer) optimizeSelectStar(result *OptimizationResult) {
	if !regexp.MustCompile(`(?i)SELECT\s+\*`).MatchString(result.OptimizedQuery) {
		return
	}

	// For demonstration, suggest column specification
	result.Optimizations = append(result.Optimizations, OptimizationDetail{
		Type:        "select_columns",
		Description: "Replace SELECT * with specific column names",
		Reasoning:   "Reduces data transfer, improves query plan stability, and prevents breaking changes when schema changes",
		Impact:      "high",
	})

	result.Warnings = append(result.Warnings,
		"Cannot automatically determine columns - please specify exact columns needed")
}

// optimizeJoins optimizes JOIN operations
func (o *QueryOptimizer) optimizeJoins(result *OptimizationResult) {
	query := result.OptimizedQuery

	// Check for implicit cross joins
	if regexp.MustCompile(`(?i)FROM\s+\w+\s*,\s*\w+`).MatchString(query) {
		result.Optimizations = append(result.Optimizations, OptimizationDetail{
			Type:        "explicit_join",
			Description: "Convert implicit joins to explicit JOIN syntax",
			Reasoning:   "Explicit JOINs are clearer and allow better optimization by the query planner",
			Impact:      "medium",
		})

		// Convert implicit to explicit (simplified)
		result.OptimizedQuery = regexp.MustCompile(`(?i)(FROM\s+)(\w+)\s*,\s*(\w+)`).
			ReplaceAllString(result.OptimizedQuery, "$1$2 CROSS JOIN $3")
	}

	// Suggest join order optimization
	joinCount := len(regexp.MustCompile(`(?i)JOIN`).FindAllString(query, -1))
	if joinCount > 2 {
		result.Optimizations = append(result.Optimizations, OptimizationDetail{
			Type:        "join_order",
			Description: "Consider reordering JOINs to filter earlier",
			Reasoning:   "Join smaller result sets first to reduce intermediate data size",
			Impact:      "medium",
		})
	}
}

// optimizeSubqueries converts subqueries to JOINs where beneficial
func (o *QueryOptimizer) optimizeSubqueries(result *OptimizationResult) {
	query := result.OptimizedQuery

	// Check for IN (SELECT ...) pattern
	inSubqueryPattern := regexp.MustCompile(`(?i)(\w+)\s+IN\s*\(\s*SELECT\s+(\w+)\s+FROM\s+(\w+)([^)]*)\)`)
	if matches := inSubqueryPattern.FindStringSubmatch(query); len(matches) > 0 {
		result.Optimizations = append(result.Optimizations, OptimizationDetail{
			Type:        "subquery_to_join",
			Description: "Convert IN (SELECT ...) to JOIN for better performance",
			Reasoning:   "JOINs often perform better than correlated subqueries, especially with proper indexes",
			Impact:      "high",
		})

		// Provide example conversion
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Consider converting: %s IN (SELECT ...) to EXISTS or JOIN", matches[1]))
	}

	// Check for correlated subqueries
	if strings.Contains(query, "SELECT") && strings.Count(query, "SELECT") > 1 {
		result.Optimizations = append(result.Optimizations, OptimizationDetail{
			Type:        "correlated_subquery",
			Description: "Review correlated subqueries for optimization",
			Reasoning:   "Correlated subqueries execute once per row and can be very slow",
			Impact:      "high",
		})
	}
}

// optimizeInClauses optimizes IN clauses
func (o *QueryOptimizer) optimizeInClauses(result *OptimizationResult) {
	query := result.OptimizedQuery

	// Check for large IN clauses
	inPattern := regexp.MustCompile(`(?i)IN\s*\([^)]+\)`)
	inClauses := inPattern.FindAllString(query, -1)

	for _, inClause := range inClauses {
		// Count items in IN clause (simple approximation)
		itemCount := strings.Count(inClause, ",") + 1

		if itemCount > 10 {
			result.Optimizations = append(result.Optimizations, OptimizationDetail{
				Type:        "large_in_clause",
				Description: fmt.Sprintf("Large IN clause with ~%d items detected", itemCount),
				Reasoning:   "Large IN clauses can be inefficient. Consider using a temporary table or VALUES clause",
				Impact:      "medium",
			})

			// Suggest VALUES optimization for PostgreSQL
			result.Warnings = append(result.Warnings,
				"Consider using VALUES clause or temporary table for large IN clauses")
		}
	}

	// Check for NOT IN
	if regexp.MustCompile(`(?i)NOT\s+IN`).MatchString(query) {
		result.Optimizations = append(result.Optimizations, OptimizationDetail{
			Type:        "not_in_optimization",
			Description: "Replace NOT IN with NOT EXISTS for NULL-safe behavior",
			Reasoning:   "NOT IN can behave unexpectedly with NULL values and may be slower",
			Impact:      "medium",
		})
	}
}

// optimizePagination optimizes LIMIT/OFFSET patterns
func (o *QueryOptimizer) optimizePagination(result *OptimizationResult) {
	query := result.OptimizedQuery

	// Check for OFFSET usage
	offsetPattern := regexp.MustCompile(`(?i)OFFSET\s+(\d+)`)
	if matches := offsetPattern.FindStringSubmatch(query); len(matches) > 1 {
		var offset int
		fmt.Sscanf(matches[1], "%d", &offset)

		if offset > 100 {
			result.Optimizations = append(result.Optimizations, OptimizationDetail{
				Type:        "pagination",
				Description: fmt.Sprintf("High OFFSET value (%d) detected", offset),
				Reasoning:   "Large OFFSET values cause the database to scan and discard many rows",
				Impact:      "high",
			})

			// Suggest keyset pagination
			result.Warnings = append(result.Warnings,
				"Consider keyset pagination: WHERE id > last_seen_id ORDER BY id LIMIT n")
		}
	}

	// Check for missing ORDER BY with LIMIT
	if strings.Contains(strings.ToUpper(query), "LIMIT") &&
		!strings.Contains(strings.ToUpper(query), "ORDER BY") {
		result.Warnings = append(result.Warnings,
			"LIMIT without ORDER BY may return inconsistent results")
	}
}

// optimizeDistinct optimizes DISTINCT usage
func (o *QueryOptimizer) optimizeDistinct(result *OptimizationResult) {
	query := result.OptimizedQuery

	if regexp.MustCompile(`(?i)SELECT\s+DISTINCT`).MatchString(query) {
		// Check if GROUP BY might be more appropriate
		if regexp.MustCompile(`(?i)COUNT|SUM|AVG|MAX|MIN`).MatchString(query) {
			result.Optimizations = append(result.Optimizations, OptimizationDetail{
				Type:        "distinct_with_aggregation",
				Description: "DISTINCT with aggregation functions detected",
				Reasoning:   "Consider using GROUP BY instead of DISTINCT with aggregations",
				Impact:      "medium",
			})
		}

		// Suggest index for DISTINCT columns
		result.IndexSuggestions = append(result.IndexSuggestions, IndexSuggestion{
			TableName: "affected_table",
			Columns:   []string{"distinct_columns"},
			IndexType: "btree",
			Reasoning: "Index on DISTINCT columns can significantly improve performance",
		})
	}
}

// optimizeFunctions optimizes function usage
func (o *QueryOptimizer) optimizeFunctions(result *OptimizationResult) {
	query := result.OptimizedQuery

	// Check for functions in WHERE clause
	whereFunctionPattern := regexp.MustCompile(`(?i)WHERE[^(]*\b(UPPER|LOWER|SUBSTR|DATE|EXTRACT)\s*\(`)
	if whereFunctionPattern.MatchString(query) {
		result.Optimizations = append(result.Optimizations, OptimizationDetail{
			Type:        "function_in_where",
			Description: "Function calls in WHERE clause detected",
			Reasoning:   "Functions on columns prevent index usage",
			Impact:      "high",
		})

		// Suggest functional indexes
		result.IndexSuggestions = append(result.IndexSuggestions, IndexSuggestion{
			TableName: "affected_table",
			Columns:   []string{"expression"},
			IndexType: "btree",
			Reasoning: "Create functional index on the expression used in WHERE",
		})

		result.Warnings = append(result.Warnings,
			"Consider creating functional indexes or rewriting conditions to use bare columns")
	}

	// Check for expensive string operations
	if regexp.MustCompile(`(?i)LIKE\s+'%[^']+%'`).MatchString(query) {
		result.Optimizations = append(result.Optimizations, OptimizationDetail{
			Type:        "wildcard_search",
			Description: "Wildcard search pattern detected",
			Reasoning:   "Leading wildcards prevent index usage",
			Impact:      "high",
		})

		// Suggest text search
		result.Warnings = append(result.Warnings,
			"Consider using PostgreSQL full-text search or trigram indexes for pattern matching")
	}
}

// optimizeExists optimizes EXISTS/NOT EXISTS patterns
func (o *QueryOptimizer) optimizeExists(result *OptimizationResult) {
	query := result.OptimizedQuery

	// Suggest EXISTS over COUNT(*) for existence checks
	if regexp.MustCompile(`(?i)COUNT\s*\(\s*\*\s*\)[^><=]+>\s*0`).MatchString(query) {
		result.Optimizations = append(result.Optimizations, OptimizationDetail{
			Type:        "count_to_exists",
			Description: "COUNT(*) > 0 pattern detected",
			Reasoning:   "EXISTS stops at first match, COUNT(*) processes all rows",
			Impact:      "high",
		})

		result.Warnings = append(result.Warnings,
			"Replace COUNT(*) > 0 with EXISTS for better performance")
	}
}

// suggestIndexes suggests indexes based on query patterns
func (o *QueryOptimizer) suggestIndexes(result *OptimizationResult) {
	query := result.OptimizedQuery

	// Extract WHERE conditions
	wherePattern := regexp.MustCompile(`(?i)WHERE\s+(.+?)(?:\s+GROUP|\s+ORDER|\s+LIMIT|$)`)
	if matches := wherePattern.FindStringSubmatch(query); len(matches) > 1 {
		conditions := matches[1]

		// Extract column names from conditions
		columnPattern := regexp.MustCompile(`(\w+)\s*[=<>]`)
		columns := columnPattern.FindAllStringSubmatch(conditions, -1)

		if len(columns) > 0 {
			indexColumns := []string{}
			for _, col := range columns {
				if len(col) > 1 {
					indexColumns = append(indexColumns, col[1])
				}
			}

			if len(indexColumns) > 0 {
				result.IndexSuggestions = append(result.IndexSuggestions, IndexSuggestion{
					TableName: "target_table",
					Columns:   indexColumns,
					IndexType: "btree",
					Reasoning: "Index on WHERE clause columns for faster filtering",
					CreateQuery: fmt.Sprintf("CREATE INDEX idx_table_columns ON target_table (%s)",
						strings.Join(indexColumns, ", ")),
				})
			}
		}
	}

	// Suggest covering index for SELECT columns
	if !strings.Contains(strings.ToUpper(query), "SELECT *") {
		result.Warnings = append(result.Warnings,
			"Consider covering indexes that include all SELECT columns to enable index-only scans")
	}
}

// calculateBenefit estimates the expected benefit of optimizations
func (o *QueryOptimizer) calculateBenefit(result *OptimizationResult) {
	highImpact := 0
	mediumImpact := 0
	lowImpact := 0

	for _, opt := range result.Optimizations {
		switch opt.Impact {
		case "high":
			highImpact++
		case "medium":
			mediumImpact++
		case "low":
			lowImpact++
		}
	}

	if highImpact > 0 {
		result.ExpectedBenefit = fmt.Sprintf("Significant performance improvement expected (%d high-impact optimizations)", highImpact)
	} else if mediumImpact > 0 {
		result.ExpectedBenefit = fmt.Sprintf("Moderate performance improvement expected (%d medium-impact optimizations)", mediumImpact)
	} else if lowImpact > 0 {
		result.ExpectedBenefit = "Minor performance improvement expected"
	} else {
		result.ExpectedBenefit = "Query appears to be reasonably optimized"
	}
}
