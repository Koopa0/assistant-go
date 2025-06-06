package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PerformanceChecker checks PostgreSQL database performance metrics
type PerformanceChecker struct {
	logger *slog.Logger
	pool   *pgxpool.Pool
}

// NewPerformanceChecker creates a new performance checker
func NewPerformanceChecker(logger *slog.Logger, pool *pgxpool.Pool) *PerformanceChecker {
	return &PerformanceChecker{
		logger: logger,
		pool:   pool,
	}
}

// PerformanceReport represents a comprehensive performance analysis
type PerformanceReport struct {
	Timestamp       time.Time               `json:"timestamp"`
	DatabaseInfo    DatabaseInfo            `json:"database_info"`
	ConnectionStats ConnectionStats         `json:"connection_stats"`
	QueryStats      []QueryPerformanceStats `json:"query_stats"`
	TableStats      []TablePerformanceStats `json:"table_stats"`
	IndexStats      []IndexPerformanceStats `json:"index_stats"`
	CacheStats      CacheStats              `json:"cache_stats"`
	LockStats       LockStats               `json:"lock_stats"`
	Issues          []PerformanceIssue      `json:"issues"`
	Recommendations []string                `json:"recommendations"`
}

// DatabaseInfo contains general database information
type DatabaseInfo struct {
	Version        string    `json:"version"`
	Uptime         string    `json:"uptime"`
	DatabaseSize   int64     `json:"database_size_bytes"`
	ActiveQueries  int       `json:"active_queries"`
	SlowQueries    int       `json:"slow_queries"`
	LastVacuum     time.Time `json:"last_vacuum,omitempty"`
	LastAutoVacuum time.Time `json:"last_autovacuum,omitempty"`
}

// ConnectionStats contains connection pool statistics
type ConnectionStats struct {
	MaxConnections     int     `json:"max_connections"`
	CurrentConnections int     `json:"current_connections"`
	IdleConnections    int     `json:"idle_connections"`
	WaitingClients     int     `json:"waiting_clients"`
	ConnectionUsage    float64 `json:"connection_usage_percent"`
}

// QueryPerformanceStats contains statistics for slow queries
type QueryPerformanceStats struct {
	QueryID         string  `json:"query_id"`
	Query           string  `json:"query"`
	Calls           int64   `json:"calls"`
	TotalTime       float64 `json:"total_time_ms"`
	MeanTime        float64 `json:"mean_time_ms"`
	MaxTime         float64 `json:"max_time_ms"`
	StdDevTime      float64 `json:"stddev_time_ms"`
	RowsPerCall     float64 `json:"rows_per_call"`
	CacheHitPercent float64 `json:"cache_hit_percent"`
}

// TablePerformanceStats contains performance statistics for tables
type TablePerformanceStats struct {
	TableName        string    `json:"table_name"`
	SizeBytes        int64     `json:"size_bytes"`
	RowCount         int64     `json:"row_count"`
	SeqScans         int64     `json:"seq_scans"`
	SeqTuplesRead    int64     `json:"seq_tuples_read"`
	IndexScans       int64     `json:"index_scans"`
	IndexTuplesRead  int64     `json:"index_tuples_read"`
	LastVacuum       time.Time `json:"last_vacuum,omitempty"`
	LastAutoVacuum   time.Time `json:"last_autovacuum,omitempty"`
	DeadTuples       int64     `json:"dead_tuples"`
	ModificationRate float64   `json:"modification_rate"`
	CacheHitRate     float64   `json:"cache_hit_rate"`
}

// IndexPerformanceStats contains performance statistics for indexes
type IndexPerformanceStats struct {
	IndexName     string  `json:"index_name"`
	TableName     string  `json:"table_name"`
	IndexSize     int64   `json:"index_size_bytes"`
	IndexScans    int64   `json:"index_scans"`
	TuplesRead    int64   `json:"tuples_read"`
	TuplesFetched int64   `json:"tuples_fetched"`
	Bloat         float64 `json:"bloat_percent"`
	UsageRate     float64 `json:"usage_rate"`
}

// CacheStats contains cache hit statistics
type CacheStats struct {
	BufferHitRate  float64 `json:"buffer_hit_rate"`
	TempFileSize   int64   `json:"temp_file_size_bytes"`
	TempFileCount  int64   `json:"temp_file_count"`
	CheckpointRate float64 `json:"checkpoint_rate"`
	SharedBuffers  string  `json:"shared_buffers"`
	EffectiveCache string  `json:"effective_cache_size"`
}

// LockStats contains lock-related statistics
type LockStats struct {
	ActiveLocks     int             `json:"active_locks"`
	WaitingLocks    int             `json:"waiting_locks"`
	DeadlockCount   int64           `json:"deadlock_count"`
	LongestWaitTime float64         `json:"longest_wait_time_ms"`
	BlockingQueries []BlockingQuery `json:"blocking_queries"`
}

// BlockingQuery represents a query that is blocking others
type BlockingQuery struct {
	PID          int32   `json:"pid"`
	Query        string  `json:"query"`
	Duration     float64 `json:"duration_ms"`
	BlockedPIDs  []int32 `json:"blocked_pids"`
	BlockedCount int     `json:"blocked_count"`
}

// PerformanceIssue represents a performance issue found
type PerformanceIssue struct {
	Severity   string `json:"severity"` // "critical", "warning", "info"
	Category   string `json:"category"`
	Message    string `json:"message"`
	Impact     string `json:"impact"`
	Resolution string `json:"resolution"`
}

// Check performs a comprehensive performance check
func (pc *PerformanceChecker) Check(ctx context.Context) (*PerformanceReport, error) {
	report := &PerformanceReport{
		Timestamp:       time.Now(),
		Issues:          []PerformanceIssue{},
		Recommendations: []string{},
	}

	// Get database information
	dbInfo, err := pc.getDatabaseInfo(ctx)
	if err != nil {
		pc.logger.Warn("Failed to get database info", slog.String("error", err.Error()))
	} else {
		report.DatabaseInfo = *dbInfo
	}

	// Get connection statistics
	connStats, err := pc.getConnectionStats(ctx)
	if err != nil {
		pc.logger.Warn("Failed to get connection stats", slog.String("error", err.Error()))
	} else {
		report.ConnectionStats = *connStats
		pc.analyzeConnectionStats(connStats, report)
	}

	// Get query performance statistics
	queryStats, err := pc.getQueryStats(ctx)
	if err != nil {
		pc.logger.Warn("Failed to get query stats", slog.String("error", err.Error()))
	} else {
		report.QueryStats = queryStats
		pc.analyzeQueryStats(queryStats, report)
	}

	// Get table performance statistics
	tableStats, err := pc.getTableStats(ctx)
	if err != nil {
		pc.logger.Warn("Failed to get table stats", slog.String("error", err.Error()))
	} else {
		report.TableStats = tableStats
		pc.analyzeTableStats(tableStats, report)
	}

	// Get index performance statistics
	indexStats, err := pc.getIndexStats(ctx)
	if err != nil {
		pc.logger.Warn("Failed to get index stats", slog.String("error", err.Error()))
	} else {
		report.IndexStats = indexStats
		pc.analyzeIndexStats(indexStats, report)
	}

	// Get cache statistics
	cacheStats, err := pc.getCacheStats(ctx)
	if err != nil {
		pc.logger.Warn("Failed to get cache stats", slog.String("error", err.Error()))
	} else {
		report.CacheStats = *cacheStats
		pc.analyzeCacheStats(cacheStats, report)
	}

	// Get lock statistics
	lockStats, err := pc.getLockStats(ctx)
	if err != nil {
		pc.logger.Warn("Failed to get lock stats", slog.String("error", err.Error()))
	} else {
		report.LockStats = *lockStats
		pc.analyzeLockStats(lockStats, report)
	}

	// Add general recommendations
	pc.addGeneralRecommendations(report)

	return report, nil
}

// getDatabaseInfo retrieves general database information
func (pc *PerformanceChecker) getDatabaseInfo(ctx context.Context) (*DatabaseInfo, error) {
	info := &DatabaseInfo{}

	// Get PostgreSQL version
	var version string
	err := pc.pool.QueryRow(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}
	info.Version = version

	// Get database size
	var size int64
	err = pc.pool.QueryRow(ctx, `
		SELECT pg_database_size(current_database())
	`).Scan(&size)
	if err != nil {
		return nil, fmt.Errorf("failed to get database size: %w", err)
	}
	info.DatabaseSize = size

	// Get active query count
	var activeQueries int
	err = pc.pool.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM pg_stat_activity 
		WHERE state = 'active' AND pid != pg_backend_pid()
	`).Scan(&activeQueries)
	if err != nil {
		pc.logger.Warn("Failed to get active queries", slog.String("error", err.Error()))
	}
	info.ActiveQueries = activeQueries

	// Get slow query count (queries running > 1 minute)
	var slowQueries int
	err = pc.pool.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM pg_stat_activity 
		WHERE state = 'active' 
		AND pid != pg_backend_pid()
		AND now() - query_start > interval '1 minute'
	`).Scan(&slowQueries)
	if err != nil {
		pc.logger.Warn("Failed to get slow queries", slog.String("error", err.Error()))
	}
	info.SlowQueries = slowQueries

	return info, nil
}

// getConnectionStats retrieves connection pool statistics
func (pc *PerformanceChecker) getConnectionStats(ctx context.Context) (*ConnectionStats, error) {
	stats := &ConnectionStats{}

	// Get max connections
	var maxConn int
	err := pc.pool.QueryRow(ctx, "SHOW max_connections").Scan(&maxConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get max_connections: %w", err)
	}
	stats.MaxConnections = maxConn

	// Get current connection count
	var currentConn int
	err = pc.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM pg_stat_activity
	`).Scan(&currentConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get current connections: %w", err)
	}
	stats.CurrentConnections = currentConn

	// Get idle connection count
	var idleConn int
	err = pc.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'idle'
	`).Scan(&idleConn)
	if err != nil {
		pc.logger.Warn("Failed to get idle connections", slog.String("error", err.Error()))
	}
	stats.IdleConnections = idleConn

	// Calculate usage percentage
	if maxConn > 0 {
		stats.ConnectionUsage = float64(currentConn) / float64(maxConn) * 100
	}

	return stats, nil
}

// getQueryStats retrieves query performance statistics
func (pc *PerformanceChecker) getQueryStats(ctx context.Context) ([]QueryPerformanceStats, error) {
	var stats []QueryPerformanceStats

	// Check if pg_stat_statements is available
	var extExists bool
	err := pc.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements'
		)
	`).Scan(&extExists)
	if err != nil || !extExists {
		pc.logger.Info("pg_stat_statements extension not available")
		return stats, nil
	}

	// Get top slow queries
	rows, err := pc.pool.Query(ctx, `
		SELECT 
			queryid::text,
			LEFT(query, 100) as query,
			calls,
			total_exec_time as total_time,
			mean_exec_time as mean_time,
			max_exec_time as max_time,
			stddev_exec_time as stddev_time,
			rows / NULLIF(calls, 0) as rows_per_call
		FROM pg_stat_statements
		WHERE query NOT LIKE '%pg_stat_statements%'
		ORDER BY total_exec_time DESC
		LIMIT 10
	`)
	if err != nil {
		pc.logger.Warn("Failed to query pg_stat_statements", slog.String("error", err.Error()))
		return stats, nil
	}
	defer rows.Close()

	for rows.Next() {
		var stat QueryPerformanceStats
		err := rows.Scan(
			&stat.QueryID,
			&stat.Query,
			&stat.Calls,
			&stat.TotalTime,
			&stat.MeanTime,
			&stat.MaxTime,
			&stat.StdDevTime,
			&stat.RowsPerCall,
		)
		if err != nil {
			pc.logger.Warn("Failed to scan query stat", slog.String("error", err.Error()))
			continue
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// getTableStats retrieves table performance statistics
func (pc *PerformanceChecker) getTableStats(ctx context.Context) ([]TablePerformanceStats, error) {
	var stats []TablePerformanceStats

	rows, err := pc.pool.Query(ctx, `
		SELECT 
			schemaname || '.' || tablename as table_name,
			pg_relation_size(schemaname||'.'||tablename) as size_bytes,
			n_live_tup as row_count,
			seq_scan,
			seq_tup_read,
			idx_scan,
			idx_tup_fetch,
			last_vacuum,
			last_autovacuum,
			n_dead_tup as dead_tuples,
			CASE 
				WHEN n_tup_ins + n_tup_upd + n_tup_del > 0 
				THEN n_dead_tup::float / (n_tup_ins + n_tup_upd + n_tup_del)
				ELSE 0 
			END as modification_rate,
			CASE 
				WHEN heap_blks_hit + heap_blks_read > 0 
				THEN heap_blks_hit::float / (heap_blks_hit + heap_blks_read)
				ELSE 0 
			END as cache_hit_rate
		FROM pg_stat_user_tables
		ORDER BY pg_relation_size(schemaname||'.'||tablename) DESC
		LIMIT 20
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get table stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var stat TablePerformanceStats
		var lastVacuum, lastAutoVacuum *time.Time

		err := rows.Scan(
			&stat.TableName,
			&stat.SizeBytes,
			&stat.RowCount,
			&stat.SeqScans,
			&stat.SeqTuplesRead,
			&stat.IndexScans,
			&stat.IndexTuplesRead,
			&lastVacuum,
			&lastAutoVacuum,
			&stat.DeadTuples,
			&stat.ModificationRate,
			&stat.CacheHitRate,
		)
		if err != nil {
			pc.logger.Warn("Failed to scan table stat", slog.String("error", err.Error()))
			continue
		}

		if lastVacuum != nil {
			stat.LastVacuum = *lastVacuum
		}
		if lastAutoVacuum != nil {
			stat.LastAutoVacuum = *lastAutoVacuum
		}

		stats = append(stats, stat)
	}

	return stats, nil
}

// getIndexStats retrieves index performance statistics
func (pc *PerformanceChecker) getIndexStats(ctx context.Context) ([]IndexPerformanceStats, error) {
	var stats []IndexPerformanceStats

	rows, err := pc.pool.Query(ctx, `
		SELECT 
			indexrelname as index_name,
			schemaname || '.' || tablename as table_name,
			pg_relation_size(indexrelid) as index_size,
			idx_scan as index_scans,
			idx_tup_read as tuples_read,
			idx_tup_fetch as tuples_fetched,
			CASE 
				WHEN idx_scan > 0 
				THEN idx_scan::float / (seq_scan + idx_scan)
				ELSE 0 
			END as usage_rate
		FROM pg_stat_user_indexes
		JOIN pg_stat_user_tables USING (schemaname, tablename)
		ORDER BY pg_relation_size(indexrelid) DESC
		LIMIT 20
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get index stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var stat IndexPerformanceStats
		err := rows.Scan(
			&stat.IndexName,
			&stat.TableName,
			&stat.IndexSize,
			&stat.IndexScans,
			&stat.TuplesRead,
			&stat.TuplesFetched,
			&stat.UsageRate,
		)
		if err != nil {
			pc.logger.Warn("Failed to scan index stat", slog.String("error", err.Error()))
			continue
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// getCacheStats retrieves cache hit statistics
func (pc *PerformanceChecker) getCacheStats(ctx context.Context) (*CacheStats, error) {
	stats := &CacheStats{}

	// Get buffer cache hit rate
	var hit, read int64
	err := pc.pool.QueryRow(ctx, `
		SELECT 
			sum(heap_blks_hit) as hit,
			sum(heap_blks_read) as read
		FROM pg_statio_user_tables
	`).Scan(&hit, &read)
	if err != nil {
		return nil, fmt.Errorf("failed to get cache stats: %w", err)
	}

	if hit+read > 0 {
		stats.BufferHitRate = float64(hit) / float64(hit+read) * 100
	}

	// Get temp file statistics
	err = pc.pool.QueryRow(ctx, `
		SELECT 
			COALESCE(sum(temp_files), 0),
			COALESCE(sum(temp_bytes), 0)
		FROM pg_stat_database
		WHERE datname = current_database()
	`).Scan(&stats.TempFileCount, &stats.TempFileSize)
	if err != nil {
		pc.logger.Warn("Failed to get temp file stats", slog.String("error", err.Error()))
	}

	// Get shared buffers setting
	err = pc.pool.QueryRow(ctx, "SHOW shared_buffers").Scan(&stats.SharedBuffers)
	if err != nil {
		pc.logger.Warn("Failed to get shared_buffers", slog.String("error", err.Error()))
	}

	// Get effective cache size
	err = pc.pool.QueryRow(ctx, "SHOW effective_cache_size").Scan(&stats.EffectiveCache)
	if err != nil {
		pc.logger.Warn("Failed to get effective_cache_size", slog.String("error", err.Error()))
	}

	return stats, nil
}

// getLockStats retrieves lock-related statistics
func (pc *PerformanceChecker) getLockStats(ctx context.Context) (*LockStats, error) {
	stats := &LockStats{
		BlockingQueries: []BlockingQuery{},
	}

	// Get active lock count
	var activeLocks int
	err := pc.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM pg_locks WHERE granted = true
	`).Scan(&activeLocks)
	if err != nil {
		return nil, fmt.Errorf("failed to get active locks: %w", err)
	}
	stats.ActiveLocks = activeLocks

	// Get waiting lock count
	var waitingLocks int
	err = pc.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM pg_locks WHERE granted = false
	`).Scan(&waitingLocks)
	if err != nil {
		pc.logger.Warn("Failed to get waiting locks", slog.String("error", err.Error()))
	}
	stats.WaitingLocks = waitingLocks

	// Get blocking queries
	rows, err := pc.pool.Query(ctx, `
		SELECT 
			blocking.pid,
			blocking.query,
			EXTRACT(EPOCH FROM (now() - blocking.query_start)) * 1000 as duration_ms,
			COUNT(DISTINCT blocked.pid) as blocked_count
		FROM pg_stat_activity blocking
		JOIN pg_locks blocked_locks ON blocking.pid = blocked_locks.pid
		JOIN pg_locks blocking_locks ON blocked_locks.relation = blocking_locks.relation
			AND blocked_locks.pid != blocking_locks.pid
			AND NOT blocked_locks.granted
			AND blocking_locks.granted
		JOIN pg_stat_activity blocked ON blocked.pid = blocked_locks.pid
		WHERE blocking.pid = blocking_locks.pid
		GROUP BY blocking.pid, blocking.query, blocking.query_start
		ORDER BY blocked_count DESC
		LIMIT 5
	`)
	if err != nil {
		pc.logger.Warn("Failed to get blocking queries", slog.String("error", err.Error()))
		return stats, nil
	}
	defer rows.Close()

	for rows.Next() {
		var bq BlockingQuery
		err := rows.Scan(&bq.PID, &bq.Query, &bq.Duration, &bq.BlockedCount)
		if err != nil {
			pc.logger.Warn("Failed to scan blocking query", slog.String("error", err.Error()))
			continue
		}
		stats.BlockingQueries = append(stats.BlockingQueries, bq)
	}

	return stats, nil
}

// Analyze functions add issues and recommendations based on statistics

func (pc *PerformanceChecker) analyzeConnectionStats(stats *ConnectionStats, report *PerformanceReport) {
	if stats.ConnectionUsage > 80 {
		report.Issues = append(report.Issues, PerformanceIssue{
			Severity:   "warning",
			Category:   "connections",
			Message:    fmt.Sprintf("High connection usage: %.1f%%", stats.ConnectionUsage),
			Impact:     "New connections may be rejected when limit is reached",
			Resolution: "Consider increasing max_connections or using a connection pooler",
		})
	}

	if stats.IdleConnections > stats.CurrentConnections/2 && stats.CurrentConnections > 20 {
		report.Issues = append(report.Issues, PerformanceIssue{
			Severity:   "info",
			Category:   "connections",
			Message:    "Many idle connections detected",
			Impact:     "Idle connections consume memory and connection slots",
			Resolution: "Configure connection pooling with shorter idle timeouts",
		})
	}
}

func (pc *PerformanceChecker) analyzeQueryStats(stats []QueryPerformanceStats, report *PerformanceReport) {
	for _, stat := range stats {
		if stat.MeanTime > 1000 { // Query takes more than 1 second on average
			report.Issues = append(report.Issues, PerformanceIssue{
				Severity:   "warning",
				Category:   "query_performance",
				Message:    fmt.Sprintf("Slow query detected: %.0fms average", stat.MeanTime),
				Impact:     "Slow queries can cause application timeouts and poor user experience",
				Resolution: "Analyze query with EXPLAIN ANALYZE and optimize",
			})
		}
	}

	if len(stats) > 0 {
		report.Recommendations = append(report.Recommendations,
			"Enable pg_stat_statements for detailed query performance tracking",
			"Set log_min_duration_statement to capture slow queries",
			"Use auto_explain for automatic query plan logging",
		)
	}
}

func (pc *PerformanceChecker) analyzeTableStats(stats []TablePerformanceStats, report *PerformanceReport) {
	for _, stat := range stats {
		// Check for tables with high sequential scan rate
		if stat.SeqScans > stat.IndexScans*10 && stat.SeqScans > 1000 {
			report.Issues = append(report.Issues, PerformanceIssue{
				Severity:   "warning",
				Category:   "table_performance",
				Message:    fmt.Sprintf("Table %s has high sequential scan rate", stat.TableName),
				Impact:     "Sequential scans are slower than index scans for large tables",
				Resolution: "Consider adding indexes on frequently filtered columns",
			})
		}

		// Check for tables needing vacuum
		if stat.DeadTuples > int64(float64(stat.RowCount)*0.2) && stat.RowCount > 1000 {
			report.Issues = append(report.Issues, PerformanceIssue{
				Severity:   "warning",
				Category:   "maintenance",
				Message:    fmt.Sprintf("Table %s has many dead tuples (%.0f%%)", stat.TableName, stat.ModificationRate*100),
				Impact:     "Dead tuples waste disk space and slow down queries",
				Resolution: "Run VACUUM or tune autovacuum settings",
			})
		}

		// Check cache hit rate
		if stat.CacheHitRate < 0.9 && stat.SizeBytes > 1024*1024*100 { // Less than 90% for tables > 100MB
			report.Issues = append(report.Issues, PerformanceIssue{
				Severity:   "info",
				Category:   "cache",
				Message:    fmt.Sprintf("Table %s has low cache hit rate: %.1f%%", stat.TableName, stat.CacheHitRate*100),
				Impact:     "Low cache hit rate causes more disk I/O",
				Resolution: "Consider increasing shared_buffers or adding more RAM",
			})
		}
	}
}

func (pc *PerformanceChecker) analyzeIndexStats(stats []IndexPerformanceStats, report *PerformanceReport) {
	for _, stat := range stats {
		// Check for unused indexes
		if stat.IndexScans == 0 && stat.IndexSize > 1024*1024 { // Unused index > 1MB
			report.Issues = append(report.Issues, PerformanceIssue{
				Severity:   "info",
				Category:   "index_usage",
				Message:    fmt.Sprintf("Index %s is not being used", stat.IndexName),
				Impact:     "Unused indexes waste disk space and slow down writes",
				Resolution: "Consider dropping unused indexes after verifying they're not needed",
			})
		}

		// Check for low usage indexes
		if stat.UsageRate < 0.01 && stat.IndexSize > 1024*1024*10 { // Less than 1% usage for indexes > 10MB
			report.Issues = append(report.Issues, PerformanceIssue{
				Severity:   "info",
				Category:   "index_usage",
				Message:    fmt.Sprintf("Index %s has very low usage rate: %.1f%%", stat.IndexName, stat.UsageRate*100),
				Impact:     "Low usage indexes may not justify their maintenance cost",
				Resolution: "Evaluate if this index is necessary",
			})
		}
	}
}

func (pc *PerformanceChecker) analyzeCacheStats(stats *CacheStats, report *PerformanceReport) {
	if stats.BufferHitRate < 90 {
		report.Issues = append(report.Issues, PerformanceIssue{
			Severity:   "warning",
			Category:   "cache",
			Message:    fmt.Sprintf("Low buffer cache hit rate: %.1f%%", stats.BufferHitRate),
			Impact:     "Low cache hit rate increases disk I/O and query latency",
			Resolution: "Increase shared_buffers or add more system RAM",
		})
	}

	if stats.TempFileSize > 1024*1024*1024 { // More than 1GB of temp files
		report.Issues = append(report.Issues, PerformanceIssue{
			Severity:   "warning",
			Category:   "memory",
			Message:    fmt.Sprintf("High temp file usage: %d MB", stats.TempFileSize/1024/1024),
			Impact:     "Temp files indicate insufficient work_mem for complex queries",
			Resolution: "Increase work_mem for sessions running complex queries",
		})
	}
}

func (pc *PerformanceChecker) analyzeLockStats(stats *LockStats, report *PerformanceReport) {
	if stats.WaitingLocks > 10 {
		report.Issues = append(report.Issues, PerformanceIssue{
			Severity:   "warning",
			Category:   "locks",
			Message:    fmt.Sprintf("High number of waiting locks: %d", stats.WaitingLocks),
			Impact:     "Lock contention can cause query timeouts and poor performance",
			Resolution: "Identify and optimize long-running transactions",
		})
	}

	if len(stats.BlockingQueries) > 0 {
		for _, bq := range stats.BlockingQueries {
			if bq.BlockedCount > 5 || bq.Duration > 60000 { // Blocking > 5 queries or running > 1 minute
				report.Issues = append(report.Issues, PerformanceIssue{
					Severity:   "critical",
					Category:   "blocking",
					Message:    fmt.Sprintf("Query blocking %d others for %.1f seconds", bq.BlockedCount, bq.Duration/1000),
					Impact:     "Blocking queries prevent other queries from completing",
					Resolution: fmt.Sprintf("Consider killing PID %d or optimizing the query", bq.PID),
				})
			}
		}
	}
}

func (pc *PerformanceChecker) addGeneralRecommendations(report *PerformanceReport) {
	// Add general performance recommendations
	report.Recommendations = append(report.Recommendations,
		"Monitor key metrics continuously with tools like pg_stat_monitor",
		"Set up alerting for critical performance thresholds",
		"Regularly analyze and update table statistics with ANALYZE",
		"Consider partitioning for very large tables",
		"Use connection pooling (PgBouncer/pgpool) for high connection workloads",
		"Enable query plan caching where appropriate",
		"Review and tune PostgreSQL configuration for your workload",
		"Implement proper indexing strategy based on query patterns",
		"Use EXPLAIN ANALYZE regularly during development",
		"Consider read replicas for read-heavy workloads",
	)

	// Add specific recommendations based on issues found
	hasCacheIssue := false
	hasLockIssue := false
	hasQueryIssue := false

	for _, issue := range report.Issues {
		switch issue.Category {
		case "cache":
			hasCacheIssue = true
		case "locks", "blocking":
			hasLockIssue = true
		case "query_performance":
			hasQueryIssue = true
		}
	}

	if hasCacheIssue {
		report.Recommendations = append(report.Recommendations,
			"Review memory configuration: shared_buffers, work_mem, effective_cache_size",
			"Consider using pg_prewarm to preload critical tables into cache",
		)
	}

	if hasLockIssue {
		report.Recommendations = append(report.Recommendations,
			"Review transaction isolation levels and locking strategies",
			"Consider using advisory locks for application-level locking",
			"Implement query timeouts to prevent long-running transactions",
		)
	}

	if hasQueryIssue {
		report.Recommendations = append(report.Recommendations,
			"Use query optimization tools like pghero or pgbadger",
			"Consider query result caching at application level",
			"Review and optimize ORM-generated queries",
		)
	}
}
