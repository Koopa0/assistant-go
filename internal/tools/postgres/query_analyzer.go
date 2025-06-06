package postgres

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// QueryAnalyzer analyzes SQL queries for potential issues and improvements
type QueryAnalyzer struct {
	logger *slog.Logger
}

// NewQueryAnalyzer creates a new query analyzer
func NewQueryAnalyzer(logger *slog.Logger) *QueryAnalyzer {
	return &QueryAnalyzer{
		logger: logger,
	}
}

// QueryAnalysis represents the analysis result of a SQL query
type QueryAnalysis struct {
	QueryType     string        `json:"query_type"`
	Tables        []string      `json:"tables"`
	Issues        []QueryIssue  `json:"issues"`
	Suggestions   []string      `json:"suggestions"`
	Complexity    string        `json:"complexity"` // "simple", "moderate", "complex"
	EstimatedCost string        `json:"estimated_cost"`
	IndexesUsed   []string      `json:"indexes_used,omitempty"`
	JoinAnalysis  *JoinAnalysis `json:"join_analysis,omitempty"`
}

// QueryIssue represents an issue found in the query
type QueryIssue struct {
	Severity   string `json:"severity"` // "error", "warning", "info"
	Type       string `json:"type"`
	Message    string `json:"message"`
	Line       int    `json:"line,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// JoinAnalysis represents analysis of JOIN operations
type JoinAnalysis struct {
	JoinCount   int      `json:"join_count"`
	JoinTypes   []string `json:"join_types"`
	LargeJoins  []string `json:"large_joins,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// Analyze performs static analysis on a SQL query
func (a *QueryAnalyzer) Analyze(query string) (*QueryAnalysis, error) {
	analysis := &QueryAnalysis{
		Tables:      []string{},
		Issues:      []QueryIssue{},
		Suggestions: []string{},
	}

	// Normalize query for analysis
	normalizedQuery := strings.ToUpper(strings.TrimSpace(query))

	// Determine query type
	analysis.QueryType = a.determineQueryType(normalizedQuery)

	// Extract tables
	analysis.Tables = a.extractTables(query)

	// Analyze for common issues
	a.checkSelectStar(query, analysis)
	a.checkMissingWhere(normalizedQuery, analysis)
	a.checkJoinConditions(query, analysis)
	a.checkIndexUsage(query, analysis)
	a.checkSubqueries(query, analysis)
	a.checkFunctions(query, analysis)
	a.checkPagination(query, analysis)

	// Determine complexity
	analysis.Complexity = a.determineComplexity(query, analysis)

	// Add general suggestions
	a.addGeneralSuggestions(analysis)

	return analysis, nil
}

// StaticExplain provides static query explanation without database connection
func (a *QueryAnalyzer) StaticExplain(query string) (*ExplainAnalysis, error) {
	analysis, err := a.Analyze(query)
	if err != nil {
		return nil, err
	}

	return &ExplainAnalysis{
		QueryAnalysis: analysis,
		Note:          "Static analysis only - connect to database for actual EXPLAIN ANALYZE",
		Recommendations: []string{
			"Use EXPLAIN (ANALYZE, BUFFERS) for actual execution statistics",
			"Check pg_stat_statements for historical query performance",
			"Monitor slow query log for performance issues",
		},
	}, nil
}

// AnalyzeExplainOutput analyzes the output from EXPLAIN
func (a *QueryAnalyzer) AnalyzeExplainOutput(lines []string) (*ExplainAnalysis, error) {
	analysis := &ExplainAnalysis{
		PlanNodes:       []PlanNode{},
		TotalCost:       0,
		ExecutionTime:   0,
		Recommendations: []string{},
	}

	// Parse EXPLAIN output
	for _, line := range lines {
		node := a.parsePlanNode(line)
		if node != nil {
			analysis.PlanNodes = append(analysis.PlanNodes, *node)
		}

		// Extract timing information
		if strings.Contains(line, "Execution Time:") {
			fmt.Sscanf(line, "Execution Time: %f ms", &analysis.ExecutionTime)
		}
	}

	// Analyze plan nodes for issues
	a.analyzePlanNodes(analysis)

	return analysis, nil
}

// determineQueryType identifies the type of SQL query
func (a *QueryAnalyzer) determineQueryType(query string) string {
	switch {
	case strings.HasPrefix(query, "SELECT"):
		return "SELECT"
	case strings.HasPrefix(query, "INSERT"):
		return "INSERT"
	case strings.HasPrefix(query, "UPDATE"):
		return "UPDATE"
	case strings.HasPrefix(query, "DELETE"):
		return "DELETE"
	case strings.HasPrefix(query, "CREATE"):
		return "CREATE"
	case strings.HasPrefix(query, "ALTER"):
		return "ALTER"
	case strings.HasPrefix(query, "DROP"):
		return "DROP"
	default:
		return "OTHER"
	}
}

// extractTables extracts table names from the query
func (a *QueryAnalyzer) extractTables(query string) []string {
	tables := []string{}

	// Simple regex patterns for table extraction
	patterns := []string{
		`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)`,
		`(?i)JOIN\s+([a-zA-Z_][a-zA-Z0-9_]*)`,
		`(?i)INTO\s+([a-zA-Z_][a-zA-Z0-9_]*)`,
		`(?i)UPDATE\s+([a-zA-Z_][a-zA-Z0-9_]*)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(query, -1)
		for _, match := range matches {
			if len(match) > 1 {
				table := strings.ToLower(match[1])
				// Avoid duplicates
				found := false
				for _, t := range tables {
					if t == table {
						found = true
						break
					}
				}
				if !found {
					tables = append(tables, table)
				}
			}
		}
	}

	return tables
}

// checkSelectStar checks for SELECT * usage
func (a *QueryAnalyzer) checkSelectStar(query string, analysis *QueryAnalysis) {
	if regexp.MustCompile(`(?i)SELECT\s+\*`).MatchString(query) {
		analysis.Issues = append(analysis.Issues, QueryIssue{
			Severity:   "warning",
			Type:       "select_star",
			Message:    "Using SELECT * can transfer unnecessary data and break when schema changes",
			Suggestion: "Specify exact columns needed",
		})
	}
}

// checkMissingWhere checks for potentially dangerous queries without WHERE
func (a *QueryAnalyzer) checkMissingWhere(query string, analysis *QueryAnalysis) {
	if (strings.Contains(query, "UPDATE") || strings.Contains(query, "DELETE")) &&
		!strings.Contains(query, "WHERE") {
		analysis.Issues = append(analysis.Issues, QueryIssue{
			Severity:   "error",
			Type:       "missing_where",
			Message:    "UPDATE/DELETE without WHERE clause affects all rows",
			Suggestion: "Add WHERE clause to limit affected rows",
		})
	}
}

// checkJoinConditions analyzes JOIN operations
func (a *QueryAnalyzer) checkJoinConditions(query string, analysis *QueryAnalysis) {
	joinPattern := regexp.MustCompile(`(?i)(INNER|LEFT|RIGHT|FULL|CROSS)\s+JOIN`)
	joins := joinPattern.FindAllString(query, -1)

	if len(joins) > 0 {
		analysis.JoinAnalysis = &JoinAnalysis{
			JoinCount: len(joins),
			JoinTypes: joins,
		}

		// Check for CROSS JOIN
		if strings.Contains(strings.ToUpper(query), "CROSS JOIN") {
			analysis.Issues = append(analysis.Issues, QueryIssue{
				Severity:   "warning",
				Type:       "cross_join",
				Message:    "CROSS JOIN produces cartesian product, can be very expensive",
				Suggestion: "Ensure CROSS JOIN is intentional, consider adding join conditions",
			})
		}

		// Check for multiple joins
		if len(joins) > 3 {
			analysis.JoinAnalysis.Suggestions = append(analysis.JoinAnalysis.Suggestions,
				"Consider breaking complex queries with many JOINs into smaller queries or using CTEs")
		}
	}
}

// checkIndexUsage provides hints about index usage
func (a *QueryAnalyzer) checkIndexUsage(query string, analysis *QueryAnalysis) {
	// Check for functions on indexed columns
	if regexp.MustCompile(`(?i)WHERE\s+[A-Z]+\([^)]+\)\s*=`).MatchString(query) {
		analysis.Issues = append(analysis.Issues, QueryIssue{
			Severity:   "warning",
			Type:       "function_on_index",
			Message:    "Using functions on columns in WHERE clause may prevent index usage",
			Suggestion: "Consider functional indexes or rewriting the condition",
		})
	}

	// Check for LIKE with leading wildcard
	if regexp.MustCompile(`(?i)LIKE\s+'%[^']+`).MatchString(query) {
		analysis.Issues = append(analysis.Issues, QueryIssue{
			Severity:   "warning",
			Type:       "leading_wildcard",
			Message:    "LIKE with leading wildcard prevents index usage",
			Suggestion: "Consider full-text search or trigram indexes for pattern matching",
		})
	}

	// Check for OR conditions
	if regexp.MustCompile(`(?i)WHERE.*\sOR\s`).MatchString(query) {
		analysis.Suggestions = append(analysis.Suggestions,
			"OR conditions may prevent optimal index usage. Consider using UNION for better performance")
	}
}

// checkSubqueries analyzes subquery usage
func (a *QueryAnalyzer) checkSubqueries(query string, analysis *QueryAnalysis) {
	// Check for correlated subqueries
	if regexp.MustCompile(`(?i)WHERE.*IN\s*\([^)]+SELECT`).MatchString(query) {
		analysis.Suggestions = append(analysis.Suggestions,
			"Consider using JOIN instead of IN (SELECT ...) for better performance")
	}

	// Check for NOT IN with subquery
	if regexp.MustCompile(`(?i)NOT\s+IN\s*\([^)]+SELECT`).MatchString(query) {
		analysis.Issues = append(analysis.Issues, QueryIssue{
			Severity:   "warning",
			Type:       "not_in_subquery",
			Message:    "NOT IN with subquery can have unexpected behavior with NULLs",
			Suggestion: "Consider using NOT EXISTS or LEFT JOIN ... WHERE ... IS NULL",
		})
	}
}

// checkFunctions checks for expensive function usage
func (a *QueryAnalyzer) checkFunctions(query string, analysis *QueryAnalysis) {
	// Check for DISTINCT
	if regexp.MustCompile(`(?i)SELECT\s+DISTINCT`).MatchString(query) {
		analysis.Suggestions = append(analysis.Suggestions,
			"DISTINCT can be expensive. Ensure it's necessary and consider if data model changes could eliminate duplicates")
	}

	// Check for COUNT(*)
	if regexp.MustCompile(`(?i)COUNT\s*\(\s*\*\s*\)`).MatchString(query) &&
		len(analysis.Tables) > 0 {
		analysis.Suggestions = append(analysis.Suggestions,
			"For approximate counts on large tables, consider using pg_stat_user_tables.n_live_tup")
	}
}

// checkPagination checks for pagination patterns
func (a *QueryAnalyzer) checkPagination(query string, analysis *QueryAnalysis) {
	// Check for OFFSET usage
	offsetPattern := regexp.MustCompile(`(?i)OFFSET\s+(\d+)`)
	if matches := offsetPattern.FindStringSubmatch(query); len(matches) > 1 {
		var offset int
		fmt.Sscanf(matches[1], "%d", &offset)
		if offset > 1000 {
			analysis.Issues = append(analysis.Issues, QueryIssue{
				Severity:   "warning",
				Type:       "large_offset",
				Message:    fmt.Sprintf("Large OFFSET value (%d) can be inefficient", offset),
				Suggestion: "Consider keyset pagination (WHERE id > last_id) for better performance",
			})
		}
	}
}

// determineComplexity estimates query complexity
func (a *QueryAnalyzer) determineComplexity(query string, analysis *QueryAnalysis) string {
	score := 0

	// Factor in number of tables
	score += len(analysis.Tables) * 2

	// Factor in joins
	if analysis.JoinAnalysis != nil {
		score += analysis.JoinAnalysis.JoinCount * 3
	}

	// Factor in subqueries
	subqueryCount := strings.Count(strings.ToUpper(query), "(SELECT")
	score += subqueryCount * 4

	// Factor in aggregations
	if regexp.MustCompile(`(?i)(GROUP BY|HAVING|COUNT|SUM|AVG|MAX|MIN)`).MatchString(query) {
		score += 3
	}

	// Factor in CTEs
	if regexp.MustCompile(`(?i)WITH\s+\w+\s+AS`).MatchString(query) {
		score += 2
	}

	switch {
	case score <= 5:
		return "simple"
	case score <= 15:
		return "moderate"
	default:
		return "complex"
	}
}

// addGeneralSuggestions adds general optimization suggestions
func (a *QueryAnalyzer) addGeneralSuggestions(analysis *QueryAnalysis) {
	if analysis.Complexity == "complex" {
		analysis.Suggestions = append(analysis.Suggestions,
			"Complex queries may benefit from breaking into CTEs or materialized views")
	}

	if len(analysis.Tables) > 5 {
		analysis.Suggestions = append(analysis.Suggestions,
			"Consider if all joined tables are necessary for the query result")
	}

	// Add suggestion for EXPLAIN ANALYZE
	analysis.Suggestions = append(analysis.Suggestions,
		"Use EXPLAIN (ANALYZE, BUFFERS) to see actual execution plan and timings")
}

// parsePlanNode parses a single line from EXPLAIN output
func (a *QueryAnalyzer) parsePlanNode(line string) *PlanNode {
	// Simple parsing - in production, use more sophisticated parsing
	node := &PlanNode{
		NodeType: "Unknown",
		Cost:     0,
		Rows:     0,
		Width:    0,
	}

	// Extract node type
	if strings.Contains(line, "Seq Scan") {
		node.NodeType = "Sequential Scan"
		node.Warning = "Sequential scan may be slow on large tables"
	} else if strings.Contains(line, "Index Scan") {
		node.NodeType = "Index Scan"
	} else if strings.Contains(line, "Nested Loop") {
		node.NodeType = "Nested Loop"
		node.Warning = "Nested loops can be expensive with large datasets"
	}

	return node
}

// analyzePlanNodes analyzes plan nodes for performance issues
func (a *QueryAnalyzer) analyzePlanNodes(analysis *ExplainAnalysis) {
	seqScanCount := 0
	nestedLoopCount := 0

	for _, node := range analysis.PlanNodes {
		if node.NodeType == "Sequential Scan" {
			seqScanCount++
		}
		if node.NodeType == "Nested Loop" {
			nestedLoopCount++
		}
	}

	if seqScanCount > 0 {
		analysis.Recommendations = append(analysis.Recommendations,
			fmt.Sprintf("Found %d sequential scans. Consider adding indexes on filtered columns", seqScanCount))
	}

	if nestedLoopCount > 2 {
		analysis.Recommendations = append(analysis.Recommendations,
			"Multiple nested loops detected. Consider restructuring query or adding indexes")
	}

	if analysis.ExecutionTime > 1000 {
		analysis.Recommendations = append(analysis.Recommendations,
			fmt.Sprintf("Query took %.2f seconds. Consider optimization for queries over 1 second", analysis.ExecutionTime/1000))
	}
}

// ExplainAnalysis represents the analysis of EXPLAIN output
type ExplainAnalysis struct {
	QueryAnalysis   *QueryAnalysis `json:"query_analysis,omitempty"`
	PlanNodes       []PlanNode     `json:"plan_nodes"`
	TotalCost       float64        `json:"total_cost"`
	ExecutionTime   float64        `json:"execution_time_ms"`
	Recommendations []string       `json:"recommendations"`
	Note            string         `json:"note,omitempty"`
}

// PlanNode represents a node in the query execution plan
type PlanNode struct {
	NodeType   string  `json:"node_type"`
	Cost       float64 `json:"cost"`
	Rows       int64   `json:"rows"`
	Width      int     `json:"width"`
	ActualTime float64 `json:"actual_time_ms,omitempty"`
	ActualRows int64   `json:"actual_rows,omitempty"`
	IndexName  string  `json:"index_name,omitempty"`
	Warning    string  `json:"warning,omitempty"`
}
