-- Tools and tool execution related queries

-- name: CreateToolExecution :one
INSERT INTO tool_executions (
    tool_name, message_id, status, input_data, output_data, 
    error_message, execution_time_ms, started_at, completed_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING id, message_id, tool_name, input_data, output_data, status, error_message, execution_time_ms, started_at, completed_at;

-- name: GetToolExecution :one
SELECT id, message_id, tool_name, input_data, output_data, status, error_message, execution_time_ms, started_at, completed_at
FROM tool_executions
WHERE id = $1;

-- name: GetToolExecutionsByUser :many
SELECT te.id, te.message_id, te.tool_name, te.input_data, te.output_data, te.status, te.error_message, te.execution_time_ms, te.started_at, te.completed_at
FROM tool_executions te
JOIN messages m ON te.message_id = m.id
JOIN conversations c ON m.conversation_id = c.id
WHERE c.user_id = $1
  AND (sqlc.narg('tool_name')::text IS NULL OR te.tool_name = sqlc.narg('tool_name'))
ORDER BY te.started_at DESC
LIMIT $2 OFFSET $3;

-- name: GetToolExecutionsByTool :many
SELECT te.id, te.message_id, te.tool_name, te.input_data, te.output_data, te.status, te.error_message, te.execution_time_ms, te.started_at, te.completed_at
FROM tool_executions te
JOIN messages m ON te.message_id = m.id
JOIN conversations c ON m.conversation_id = c.id
WHERE te.tool_name = $1
  AND (sqlc.narg('user_id')::uuid IS NULL OR c.user_id = sqlc.narg('user_id'))
ORDER BY te.started_at DESC
LIMIT $2 OFFSET $3;

-- name: GetToolUsageStats :many
SELECT 
    te.tool_name,
    COUNT(*)::integer as total_executions,
    COUNT(CASE WHEN te.status = 'completed' THEN 1 END)::integer as success_count,
    COUNT(CASE WHEN te.status = 'failed' THEN 1 END)::integer as failure_count,
    ROUND(
        COUNT(CASE WHEN te.status = 'completed' THEN 1 END)::numeric * 100.0 / 
        NULLIF(COUNT(*), 0), 2
    )::float as success_rate,
    COALESCE(AVG(te.execution_time_ms)::integer, 0) as avg_execution_time_ms,
    MAX(te.started_at) as last_used
FROM tool_executions te
JOIN messages m ON te.message_id = m.id
JOIN conversations c ON m.conversation_id = c.id
WHERE c.user_id = $1
GROUP BY te.tool_name
ORDER BY total_executions DESC;

-- name: GetToolUsageStatsByTool :one
SELECT 
    te.tool_name,
    COUNT(*)::integer as total_executions,
    COUNT(CASE WHEN te.status = 'completed' THEN 1 END)::integer as success_count,
    COUNT(CASE WHEN te.status = 'failed' THEN 1 END)::integer as failure_count,
    ROUND(
        COUNT(CASE WHEN te.status = 'completed' THEN 1 END)::numeric * 100.0 / 
        NULLIF(COUNT(*), 0), 2
    )::float as success_rate,
    COALESCE(AVG(te.execution_time_ms)::integer, 0) as avg_execution_time_ms,
    MAX(te.started_at) as last_used
FROM tool_executions te
JOIN messages m ON te.message_id = m.id
JOIN conversations c ON m.conversation_id = c.id
WHERE te.tool_name = $1 
  AND (sqlc.narg('user_id')::uuid IS NULL OR c.user_id = sqlc.narg('user_id'))
GROUP BY te.tool_name;

-- name: UpdateToolExecutionStatus :one
UPDATE tool_executions
SET 
    status = $2,
    output_data = $3,
    error_message = $4,
    execution_time_ms = $5,
    completed_at = $6
WHERE id = $1
RETURNING id, message_id, tool_name, input_data, output_data, status, error_message, execution_time_ms, started_at, completed_at;

-- name: GetRecentToolExecutions :many
SELECT te.id, te.message_id, te.tool_name, te.input_data, te.output_data, te.status, te.error_message, te.execution_time_ms, te.started_at, te.completed_at
FROM tool_executions te
JOIN messages m ON te.message_id = m.id
JOIN conversations c ON m.conversation_id = c.id
WHERE c.user_id = $1
ORDER BY te.started_at DESC
LIMIT $2;

-- name: GetToolExecutionCount :one
SELECT COUNT(*)::integer FROM tool_executions te
JOIN messages m ON te.message_id = m.id
JOIN conversations c ON m.conversation_id = c.id
WHERE c.user_id = $1
  AND (sqlc.narg('tool_name')::text IS NULL OR te.tool_name = sqlc.narg('tool_name'));

-- name: GetMostUsedTools :many
SELECT 
    te.tool_name,
    COUNT(*)::integer as usage_count,
    MAX(te.started_at) as last_used
FROM tool_executions te
JOIN messages m ON te.message_id = m.id
JOIN conversations c ON m.conversation_id = c.id
WHERE c.user_id = $1
GROUP BY te.tool_name
ORDER BY usage_count DESC
LIMIT $2;

-- name: GetToolExecutionTrends :many
SELECT 
    DATE(te.started_at) as execution_date,
    te.tool_name,
    COUNT(*)::integer as executions,
    COUNT(CASE WHEN te.status = 'completed' THEN 1 END)::integer as successes,
    COUNT(CASE WHEN te.status = 'failed' THEN 1 END)::integer as failures
FROM tool_executions te
JOIN messages m ON te.message_id = m.id
JOIN conversations c ON m.conversation_id = c.id
WHERE c.user_id = $1
  AND te.started_at >= $2
  AND te.started_at <= $3
GROUP BY DATE(te.started_at), te.tool_name
ORDER BY execution_date DESC, executions DESC;

-- name: DeleteOldToolExecutions :exec
DELETE FROM tool_executions
WHERE started_at < $1;

-- name: GetToolExecutionsByStatus :many
SELECT te.id, te.message_id, te.tool_name, te.input_data, te.output_data, te.status, te.error_message, te.execution_time_ms, te.started_at, te.completed_at
FROM tool_executions te
JOIN messages m ON te.message_id = m.id
JOIN conversations c ON m.conversation_id = c.id
WHERE c.user_id = $1
  AND te.status = $2
ORDER BY te.started_at DESC
LIMIT $3 OFFSET $4;

-- name: GetToolExecutionsByMessage :many
SELECT id, message_id, tool_name, input_data, output_data, status, error_message, execution_time_ms, started_at, completed_at
FROM tool_executions
WHERE message_id = $1
ORDER BY started_at ASC;