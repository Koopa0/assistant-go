-- Development workflow queries

-- =====================================================
-- DEVELOPMENT SESSIONS QUERIES
-- =====================================================

-- name: CreateDevelopmentSession :one
INSERT INTO development_sessions (
    user_id,
    session_type,
    project_context,
    goals,
    started_at
) VALUES (
    $1::uuid, $2, $3, $4, COALESCE($5, NOW())
) RETURNING *;

-- name: GetDevelopmentSession :one
SELECT * FROM development_sessions
WHERE id = $1;

-- name: GetDevelopmentSessions :many
SELECT * FROM development_sessions
WHERE user_id = $1::uuid
  AND (session_type = $2 OR $2 IS NULL)
  AND started_at >= COALESCE($3, NOW() - INTERVAL '30 days')
ORDER BY started_at DESC
LIMIT $4 OFFSET $5;

-- name: GetActiveSessions :many
SELECT * FROM development_sessions
WHERE user_id = $1::uuid
  AND ended_at IS NULL
ORDER BY started_at DESC;

-- name: UpdateSessionProgress :one
UPDATE development_sessions
SET actual_outcomes = $2,
    interruption_count = $3,
    productivity_metrics = $4,
    mood_indicators = $5
WHERE id = $1
RETURNING *;

-- name: EndDevelopmentSession :one
UPDATE development_sessions
SET ended_at = COALESCE($2, NOW()),
    total_duration_minutes = EXTRACT(EPOCH FROM (COALESCE($2, NOW()) - started_at)) / 60,
    focus_score = CASE 
        WHEN $3 > 0 THEN GREATEST(0.1, 1.0 - ($3::float / 10.0))
        ELSE 1.0
    END
WHERE id = $1
RETURNING *;

-- name: GetSessionStatistics :one
SELECT 
    session_type,
    COUNT(*) as total_sessions,
    AVG(total_duration_minutes) as avg_duration_minutes,
    AVG(focus_score) as avg_focus_score,
    AVG(interruption_count) as avg_interruptions,
    COUNT(*) FILTER (WHERE ended_at IS NOT NULL) as completed_sessions
FROM development_sessions
WHERE user_id = $1::uuid
  AND started_at >= COALESCE($2, NOW() - INTERVAL '30 days')
GROUP BY session_type
ORDER BY total_sessions DESC;

-- name: GetProductivityTrends :many
SELECT 
    DATE_TRUNC($2::text, started_at) as time_period,
    COUNT(*) as session_count,
    AVG(total_duration_minutes) as avg_duration,
    AVG(focus_score) as avg_focus,
    SUM(total_duration_minutes) as total_time_minutes
FROM development_sessions
WHERE user_id = $1::uuid
  AND started_at >= COALESCE($3, NOW() - INTERVAL '90 days')
  AND ended_at IS NOT NULL
GROUP BY DATE_TRUNC($2::text, started_at)
ORDER BY time_period;

-- name: DeleteOldSessions :exec
DELETE FROM development_sessions
WHERE user_id = $1::uuid
  AND started_at < $2;

-- =====================================================
-- CODE PATTERNS QUERIES
-- =====================================================

-- name: CreateCodePattern :one
INSERT INTO code_patterns (
    user_id,
    pattern_category,
    pattern_name,
    pattern_ast,
    usage_contexts
) VALUES (
    $1::uuid, $2, $3, $4, $5
) ON CONFLICT (user_id, pattern_category, pattern_name)
DO UPDATE SET
    pattern_ast = $4,
    usage_contexts = $5,
    frequency = code_patterns.frequency + 1,
    last_used = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetCodePattern :one
SELECT * FROM code_patterns
WHERE id = $1;

-- name: GetCodePatterns :many
SELECT * FROM code_patterns
WHERE user_id = $1::uuid
  AND (pattern_category = $2 OR $2 IS NULL)
ORDER BY frequency DESC, last_used DESC
LIMIT $3 OFFSET $4;

-- name: SearchCodePatterns :many
SELECT * FROM code_patterns
WHERE user_id = $1::uuid
  AND (pattern_name ILIKE '%' || $2 || '%' OR $2 = ANY(usage_contexts))
ORDER BY frequency DESC, last_used DESC
LIMIT $3;

-- name: UpdatePatternUsage :one
UPDATE code_patterns
SET frequency = frequency + 1,
    last_used = NOW(),
    usage_contexts = CASE 
        WHEN $2 = ANY(usage_contexts) THEN usage_contexts
        ELSE array_append(usage_contexts, $2)
    END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdatePatternQuality :one
UPDATE code_patterns
SET quality_score = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: EvolvePattern :one
UPDATE code_patterns
SET pattern_ast = $2,
    evolution_history = evolution_history || jsonb_build_object(
        'timestamp', EXTRACT(EPOCH FROM NOW()),
        'old_ast', pattern_ast,
        'change_reason', $3,
        'quality_improvement', $4
    ),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetPopularPatterns :many
SELECT 
    pattern_category,
    pattern_name,
    frequency,
    last_used,
    quality_score,
    usage_contexts
FROM code_patterns
WHERE user_id = $1::uuid
  AND frequency >= $2
ORDER BY frequency DESC, quality_score DESC NULLS LAST
LIMIT $3;

-- name: GetPatternEvolution :many
SELECT 
    pattern_name,
    pattern_category,
    frequency,
    quality_score,
    jsonb_array_length(evolution_history) as evolution_count,
    created_at,
    updated_at
FROM code_patterns
WHERE user_id = $1::uuid
  AND jsonb_array_length(evolution_history) > 0
ORDER BY jsonb_array_length(evolution_history) DESC, updated_at DESC;

-- name: GetPatternStatistics :one
SELECT 
    pattern_category,
    COUNT(*) as pattern_count,
    SUM(frequency) as total_usage,
    AVG(frequency) as avg_frequency,
    AVG(quality_score) as avg_quality,
    COUNT(*) FILTER (WHERE quality_score >= 0.8) as high_quality_patterns
FROM code_patterns
WHERE user_id = $1::uuid
GROUP BY pattern_category
ORDER BY total_usage DESC;

-- name: CleanupUnusedPatterns :exec
DELETE FROM code_patterns
WHERE user_id = $1::uuid
  AND frequency <= $2
  AND last_used < $3;