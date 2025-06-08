package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// IndexAdvisor suggests indexes based on query patterns and table statistics
type IndexAdvisor struct {
	logger *slog.Logger
	pool   *pgxpool.Pool
}

// NewIndexAdvisor creates a new index advisor
func NewIndexAdvisor(logger *slog.Logger, pool *pgxpool.Pool) *IndexAdvisor {
	return &IndexAdvisor{
		logger: logger,
		pool:   pool,
	}
}

// IndexAdvice represents index suggestions
type IndexAdvice struct {
	TableName        string           `json:"table_name"`
	ExistingIndexes  []ExistingIndex  `json:"existing_indexes"`
	SuggestedIndexes []SuggestedIndex `json:"suggested_indexes"`
	RedundantIndexes []RedundantIndex `json:"redundant_indexes"`
	Statistics       IndexStatistics  `json:"statistics"`
	Analysis         string           `json:"analysis"`
}

// ExistingIndex represents an existing index
type ExistingIndex struct {
	Name       string   `json:"name"`
	Columns    []string `json:"columns"`
	Type       string   `json:"type"`
	Size       int64    `json:"size_bytes"`
	UsageCount int64    `json:"usage_count"`
	IsUnique   bool     `json:"is_unique"`
	IsPrimary  bool     `json:"is_primary"`
	IsPartial  bool     `json:"is_partial"`
}

// SuggestedIndex represents a suggested index
type SuggestedIndex struct {
	Columns        []string `json:"columns"`
	Type           string   `json:"type"`
	Reason         string   `json:"reason"`
	Impact         string   `json:"impact"` // "high", "medium", "low"
	CreateQuery    string   `json:"create_query"`
	EstimatedSize  int64    `json:"estimated_size_bytes"`
	Considerations []string `json:"considerations"`
}

// RedundantIndex represents a redundant index
type RedundantIndex struct {
	IndexName     string `json:"index_name"`
	RedundantWith string `json:"redundant_with"`
	Reason        string `json:"reason"`
	DropQuery     string `json:"drop_query"`
	SizeSavings   int64  `json:"size_savings_bytes"`
}

// IndexStatistics contains index-related statistics
type IndexStatistics struct {
	TotalIndexes     int     `json:"total_indexes"`
	TotalIndexSize   int64   `json:"total_index_size_bytes"`
	UnusedIndexes    int     `json:"unused_indexes"`
	DuplicateIndexes int     `json:"duplicate_indexes"`
	TableSize        int64   `json:"table_size_bytes"`
	IndexSizeRatio   float64 `json:"index_size_ratio"`
}

// SuggestIndexes analyzes a table and suggests indexes
func (a *IndexAdvisor) SuggestIndexes(ctx context.Context, tableName string) (*IndexAdvice, error) {
	if a.pool == nil {
		return a.staticSuggestions(tableName), nil
	}

	advice := &IndexAdvice{
		TableName:        tableName,
		ExistingIndexes:  []ExistingIndex{},
		SuggestedIndexes: []SuggestedIndex{},
		RedundantIndexes: []RedundantIndex{},
	}

	// Get existing indexes
	if err := a.analyzeExistingIndexes(ctx, tableName, advice); err != nil {
		return nil, fmt.Errorf("failed to analyze existing indexes: %w", err)
	}

	// Analyze query patterns
	if err := a.analyzeQueryPatterns(ctx, tableName, advice); err != nil {
		return nil, fmt.Errorf("failed to analyze query patterns: %w", err)
	}

	// Check for missing foreign key indexes
	if err := a.checkForeignKeyIndexes(ctx, tableName, advice); err != nil {
		return nil, fmt.Errorf("failed to check foreign key indexes: %w", err)
	}

	// Identify redundant indexes
	a.identifyRedundantIndexes(advice)

	// Calculate statistics
	a.calculateStatistics(ctx, tableName, advice)

	// Generate analysis summary
	a.generateAnalysis(advice)

	return advice, nil
}

// staticSuggestions provides suggestions without database connection
func (a *IndexAdvisor) staticSuggestions(tableName string) *IndexAdvice {
	return &IndexAdvice{
		TableName: tableName,
		Analysis:  "Static analysis only - connect to database for comprehensive index advice",
		SuggestedIndexes: []SuggestedIndex{
			{
				Columns: []string{"common_query_column"},
				Type:    "btree",
				Reason:  "General recommendation: Index columns used in WHERE clauses",
				Impact:  "medium",
				Considerations: []string{
					"Verify column selectivity before creating",
					"Monitor query patterns to confirm usage",
				},
			},
		},
	}
}

// analyzeExistingIndexes gets information about existing indexes
func (a *IndexAdvisor) analyzeExistingIndexes(ctx context.Context, tableName string, advice *IndexAdvice) error {
	query := `
		SELECT 
			i.indexname,
			string_to_array(replace(replace(split_part(indexdef, '(', 2), ')', ''), ' ', ''), ',') as columns,
			am.amname as index_type,
			pg_relation_size(c.oid) as size_bytes,
			COALESCE(pg_stat_user_indexes.idx_scan, 0) as usage_count,
			ix.indisunique,
			ix.indisprimary,
			ix.indpred IS NOT NULL as is_partial
		FROM pg_indexes i
		JOIN pg_class c ON c.relname = i.indexname
		JOIN pg_index ix ON ix.indexrelid = c.oid
		JOIN pg_am am ON am.oid = c.relam
		JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = i.schemaname
		LEFT JOIN pg_stat_user_indexes ON 
			pg_stat_user_indexes.schemaname = i.schemaname AND
			pg_stat_user_indexes.indexrelname = i.indexname
		WHERE i.tablename = $1
		ORDER BY i.indexname`

	rows, err := a.pool.Query(ctx, query, tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var idx ExistingIndex
		var columns []string

		if err := rows.Scan(
			&idx.Name,
			&columns,
			&idx.Type,
			&idx.Size,
			&idx.UsageCount,
			&idx.IsUnique,
			&idx.IsPrimary,
			&idx.IsPartial,
		); err != nil {
			return err
		}

		idx.Columns = columns
		advice.ExistingIndexes = append(advice.ExistingIndexes, idx)
	}

	return rows.Err()
}

// analyzeQueryPatterns analyzes pg_stat_statements for query patterns
func (a *IndexAdvisor) analyzeQueryPatterns(ctx context.Context, tableName string, advice *IndexAdvice) error {
	// Check if pg_stat_statements is available
	var hasStatStatements bool
	err := a.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements')").Scan(&hasStatStatements)
	if err != nil || !hasStatStatements {
		a.logger.Info("pg_stat_statements not available, skipping query pattern analysis")
		return nil
	}

	// Analyze slow queries on this table
	query := `
		SELECT 
			query,
			calls,
			total_exec_time,
			mean_exec_time
		FROM pg_stat_statements
		WHERE query ILIKE '%' || $1 || '%'
		AND query NOT ILIKE '%pg_stat_statements%'
		AND mean_exec_time > 100  -- queries slower than 100ms
		ORDER BY mean_exec_time DESC
		LIMIT 10`

	rows, err := a.pool.Query(ctx, query, tableName)
	if err != nil {
		a.logger.Warn("Failed to query pg_stat_statements", slog.String("error", err.Error()))
		return nil
	}
	defer rows.Close()

	whereColumns := make(map[string]int)
	joinColumns := make(map[string]int)
	orderColumns := make(map[string]int)

	for rows.Next() {
		var queryText string
		var calls int64
		var totalTime, meanTime float64

		if err := rows.Scan(&queryText, &calls, &totalTime, &meanTime); err != nil {
			continue
		}

		// Simple pattern extraction (in production, use proper SQL parsing)
		a.extractColumnPatterns(queryText, tableName, whereColumns, joinColumns, orderColumns)
	}

	// Suggest indexes based on patterns
	a.suggestFromPatterns(tableName, whereColumns, joinColumns, orderColumns, advice)

	return nil
}

// extractColumnPatterns extracts column usage patterns from queries
func (a *IndexAdvisor) extractColumnPatterns(query, tableName string,
	whereColumns, joinColumns, orderColumns map[string]int) {

	queryUpper := strings.ToUpper(query)
	tableUpper := strings.ToUpper(tableName)

	// Extract WHERE clause columns (simplified)
	if whereIdx := strings.Index(queryUpper, "WHERE"); whereIdx > 0 {
		whereClause := queryUpper[whereIdx:]
		if endIdx := strings.Index(whereClause, "GROUP BY"); endIdx > 0 {
			whereClause = whereClause[:endIdx]
		} else if endIdx := strings.Index(whereClause, "ORDER BY"); endIdx > 0 {
			whereClause = whereClause[:endIdx]
		}

		// Look for table.column patterns
		pattern := tableUpper + "."
		for i := strings.Index(whereClause, pattern); i >= 0; i = strings.Index(whereClause[i+1:], pattern) {
			if i+len(pattern) < len(whereClause) {
				colEnd := i + len(pattern)
				for colEnd < len(whereClause) && isColumnChar(whereClause[colEnd]) {
					colEnd++
				}
				if colEnd > i+len(pattern) {
					column := strings.ToLower(whereClause[i+len(pattern) : colEnd])
					whereColumns[column]++
				}
			}
		}
	}

	// Extract ORDER BY columns
	if orderIdx := strings.Index(queryUpper, "ORDER BY"); orderIdx > 0 {
		orderClause := queryUpper[orderIdx+8:]
		if limitIdx := strings.Index(orderClause, "LIMIT"); limitIdx > 0 {
			orderClause = orderClause[:limitIdx]
		}

		// Simple extraction
		parts := strings.Split(orderClause, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.Contains(part, tableUpper+".") {
				colStart := strings.Index(part, tableUpper+".") + len(tableUpper) + 1
				colEnd := colStart
				for colEnd < len(part) && isColumnChar(part[colEnd]) {
					colEnd++
				}
				if colEnd > colStart {
					column := strings.ToLower(part[colStart:colEnd])
					orderColumns[column]++
				}
			}
		}
	}
}

// isColumnChar checks if a character can be part of a column name
func isColumnChar(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// suggestFromPatterns creates index suggestions based on query patterns
func (a *IndexAdvisor) suggestFromPatterns(tableName string,
	whereColumns, joinColumns, orderColumns map[string]int, advice *IndexAdvice) {

	// Sort columns by frequency
	type columnFreq struct {
		column string
		freq   int
	}

	var whereFreqs []columnFreq
	for col, freq := range whereColumns {
		whereFreqs = append(whereFreqs, columnFreq{col, freq})
	}
	sort.Slice(whereFreqs, func(i, j int) bool {
		return whereFreqs[i].freq > whereFreqs[j].freq
	})

	// Check if WHERE columns are already indexed
	for _, cf := range whereFreqs {
		if cf.freq < 5 { // Skip infrequent columns
			continue
		}

		hasIndex := false
		for _, idx := range advice.ExistingIndexes {
			if len(idx.Columns) > 0 && idx.Columns[0] == cf.column {
				hasIndex = true
				break
			}
		}

		if !hasIndex {
			advice.SuggestedIndexes = append(advice.SuggestedIndexes, SuggestedIndex{
				Columns:     []string{cf.column},
				Type:        "btree",
				Reason:      fmt.Sprintf("Column frequently used in WHERE clauses (%d times)", cf.freq),
				Impact:      "high",
				CreateQuery: fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s (%s);", tableName, cf.column, tableName, cf.column),
				Considerations: []string{
					"Check column selectivity before creating",
					"Consider including additional columns for covering index",
				},
			})
		}
	}

	// Suggest composite indexes for common WHERE + ORDER BY patterns
	if len(whereFreqs) > 0 && len(orderColumns) > 0 {
		for orderCol := range orderColumns {
			if whereFreqs[0].freq > 10 { // Frequent WHERE clause
				columns := []string{whereFreqs[0].column, orderCol}
				if !a.hasCompositeIndex(columns, advice.ExistingIndexes) {
					advice.SuggestedIndexes = append(advice.SuggestedIndexes, SuggestedIndex{
						Columns: columns,
						Type:    "btree",
						Reason:  "Composite index for WHERE + ORDER BY pattern",
						Impact:  "high",
						CreateQuery: fmt.Sprintf("CREATE INDEX idx_%s_%s_%s ON %s (%s, %s);",
							tableName, columns[0], columns[1], tableName, columns[0], columns[1]),
						Considerations: []string{
							"Composite index can eliminate sort operations",
							"Column order matters - most selective column first",
						},
					})
				}
			}
		}
	}
}

// checkForeignKeyIndexes checks for missing indexes on foreign keys
func (a *IndexAdvisor) checkForeignKeyIndexes(ctx context.Context, tableName string, advice *IndexAdvice) error {
	query := `
		SELECT 
			kcu.column_name,
			ccu.table_name as referenced_table,
			ccu.column_name as referenced_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_schema = kcu.constraint_schema
			AND tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_schema = ccu.constraint_schema
			AND tc.constraint_name = ccu.constraint_name
		WHERE tc.table_name = $1
		AND tc.constraint_type = 'FOREIGN KEY'`

	rows, err := a.pool.Query(ctx, query, tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var column, refTable, refColumn string
		if err := rows.Scan(&column, &refTable, &refColumn); err != nil {
			continue
		}

		// Check if this FK column has an index
		hasIndex := false
		for _, idx := range advice.ExistingIndexes {
			if len(idx.Columns) > 0 && idx.Columns[0] == column {
				hasIndex = true
				break
			}
		}

		if !hasIndex {
			advice.SuggestedIndexes = append(advice.SuggestedIndexes, SuggestedIndex{
				Columns:     []string{column},
				Type:        "btree",
				Reason:      fmt.Sprintf("Foreign key to %s.%s lacks index", refTable, refColumn),
				Impact:      "high",
				CreateQuery: fmt.Sprintf("CREATE INDEX idx_%s_%s_fk ON %s (%s);", tableName, column, tableName, column),
				Considerations: []string{
					"FK indexes improve JOIN performance",
					"FK indexes speed up cascading deletes",
					"Prevents full table scans on FK checks",
				},
			})
		}
	}

	return rows.Err()
}

// identifyRedundantIndexes finds redundant or duplicate indexes
func (a *IndexAdvisor) identifyRedundantIndexes(advice *IndexAdvice) {
	for i, idx1 := range advice.ExistingIndexes {
		if idx1.IsPrimary {
			continue
		}

		for j, idx2 := range advice.ExistingIndexes {
			if i >= j || idx2.IsPrimary {
				continue
			}

			// Check if idx1 is redundant with idx2
			if a.isRedundant(idx1, idx2) {
				advice.RedundantIndexes = append(advice.RedundantIndexes, RedundantIndex{
					IndexName:     idx1.Name,
					RedundantWith: idx2.Name,
					Reason:        "Index columns are a prefix of another index",
					DropQuery:     fmt.Sprintf("DROP INDEX %s;", idx1.Name),
					SizeSavings:   idx1.Size,
				})
			}
		}

		// Check for unused indexes
		if idx1.UsageCount == 0 && !idx1.IsUnique {
			advice.RedundantIndexes = append(advice.RedundantIndexes, RedundantIndex{
				IndexName:     idx1.Name,
				RedundantWith: "none",
				Reason:        "Index has never been used",
				DropQuery:     fmt.Sprintf("DROP INDEX %s;", idx1.Name),
				SizeSavings:   idx1.Size,
			})
		}
	}
}

// isRedundant checks if idx1 is redundant with idx2
func (a *IndexAdvisor) isRedundant(idx1, idx2 ExistingIndex) bool {
	// Check if idx1 columns are a prefix of idx2 columns
	if len(idx1.Columns) > len(idx2.Columns) {
		return false
	}

	for i, col := range idx1.Columns {
		if idx2.Columns[i] != col {
			return false
		}
	}

	// If idx1 is unique and idx2 is not, idx1 is not redundant
	if idx1.IsUnique && !idx2.IsUnique {
		return false
	}

	return true
}

// hasCompositeIndex checks if a composite index exists with the given columns
func (a *IndexAdvisor) hasCompositeIndex(columns []string, existingIndexes []ExistingIndex) bool {
	for _, idx := range existingIndexes {
		if len(idx.Columns) >= len(columns) {
			match := true
			for i, col := range columns {
				if i >= len(idx.Columns) || idx.Columns[i] != col {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

// calculateStatistics calculates index statistics
func (a *IndexAdvisor) calculateStatistics(ctx context.Context, tableName string, advice *IndexAdvice) {
	stats := &advice.Statistics
	stats.TotalIndexes = len(advice.ExistingIndexes)

	for _, idx := range advice.ExistingIndexes {
		stats.TotalIndexSize += idx.Size
		if idx.UsageCount == 0 && !idx.IsPrimary && !idx.IsUnique {
			stats.UnusedIndexes++
		}
	}

	// Count duplicate indexes
	for _, redundant := range advice.RedundantIndexes {
		if redundant.RedundantWith != "none" {
			stats.DuplicateIndexes++
		}
	}

	// Get table size
	if a.pool != nil {
		var tableSize int64
		err := a.pool.QueryRow(ctx,
			"SELECT pg_total_relation_size($1::regclass) - pg_indexes_size($1::regclass)",
			tableName).Scan(&tableSize)
		if err == nil {
			stats.TableSize = tableSize
			if tableSize > 0 {
				stats.IndexSizeRatio = float64(stats.TotalIndexSize) / float64(tableSize)
			}
		}
	}
}

// generateAnalysis creates a summary analysis
func (a *IndexAdvisor) generateAnalysis(advice *IndexAdvice) {
	var analysis strings.Builder

	analysis.WriteString(fmt.Sprintf("Index analysis for table '%s':\n\n", advice.TableName))

	// Summary
	analysis.WriteString(fmt.Sprintf("- Total indexes: %d\n", advice.Statistics.TotalIndexes))
	analysis.WriteString(fmt.Sprintf("- Total index size: %.2f MB\n",
		float64(advice.Statistics.TotalIndexSize)/(1024*1024)))
	analysis.WriteString(fmt.Sprintf("- Index/table size ratio: %.2f%%\n",
		advice.Statistics.IndexSizeRatio*100))

	if advice.Statistics.UnusedIndexes > 0 {
		analysis.WriteString(fmt.Sprintf("\n⚠️  Found %d unused indexes\n", advice.Statistics.UnusedIndexes))
	}

	if advice.Statistics.DuplicateIndexes > 0 {
		analysis.WriteString(fmt.Sprintf("⚠️  Found %d duplicate/redundant indexes\n", advice.Statistics.DuplicateIndexes))
	}

	if len(advice.SuggestedIndexes) > 0 {
		analysis.WriteString(fmt.Sprintf("\n✅ %d new indexes suggested\n", len(advice.SuggestedIndexes)))
	}

	// Recommendations
	if advice.Statistics.IndexSizeRatio > 0.5 {
		analysis.WriteString("\n⚡ Index size is more than 50% of table size - review index usage\n")
	}

	advice.Analysis = analysis.String()
}
