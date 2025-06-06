-- Agent execution queries

-- name: CreateAgentExecution :one
INSERT INTO agent_executions (
    agent_type,
    user_id,
    query,
    response,
    steps,
    execution_time_ms,
    success,
    error_message,
    metadata
) VALUES (
    $1, $2::uuid, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetAgentExecution :one
SELECT * FROM agent_executions
WHERE id = $1;

-- name: GetAgentExecutionsByUser :many
SELECT * FROM agent_executions
WHERE user_id = $1::uuid
  AND (agent_type = $2 OR $2 IS NULL)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetRecentAgentExecutions :many
SELECT * FROM agent_executions
WHERE user_id = $1::uuid
ORDER BY created_at DESC
LIMIT $2;

-- name: GetAgentExecutionStats :one
SELECT
    agent_type,
    COUNT(*) as total_executions,
    COUNT(*) FILTER (WHERE success = true) as successful_executions,
    AVG(execution_time_ms) as avg_execution_time_ms,
    MAX(created_at) as last_execution
FROM agent_executions
WHERE user_id = $1::uuid
  AND (agent_type = $2 OR $2 IS NULL)
  AND created_at >= $3
GROUP BY agent_type;

-- name: DeleteAgentExecution :exec
DELETE FROM agent_executions
WHERE id = $1;

-- Chain execution queries

-- name: CreateChainExecution :one
INSERT INTO chain_executions (
    chain_type,
    user_id,
    input,
    output,
    steps,
    execution_time_ms,
    tokens_used,
    success,
    error_message,
    metadata
) VALUES (
    $1, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetChainExecution :one
SELECT * FROM chain_executions
WHERE id = $1;

-- name: GetChainExecutionsByUser :many
SELECT * FROM chain_executions
WHERE user_id = $1::uuid
  AND (chain_type = $2 OR $2 IS NULL)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetRecentChainExecutions :many
SELECT * FROM chain_executions
WHERE user_id = $1::uuid
ORDER BY created_at DESC
LIMIT $2;

-- name: GetChainExecutionStats :one
SELECT
    chain_type,
    COUNT(*) as total_executions,
    COUNT(*) FILTER (WHERE success = true) as successful_executions,
    AVG(execution_time_ms) as avg_execution_time_ms,
    SUM(tokens_used) as total_tokens_used,
    MAX(created_at) as last_execution
FROM chain_executions
WHERE user_id = $1::uuid
  AND (chain_type = $2 OR $2 IS NULL)
  AND created_at >= $3
GROUP BY chain_type;

-- name: DeleteChainExecution :exec
DELETE FROM chain_executions
WHERE id = $1;

-- Combined analytics queries

-- name: GetUserExecutionSummary :one
SELECT
    (SELECT COUNT(*) FROM agent_executions ae WHERE ae.user_id = $1::uuid AND ae.created_at >= $2) as agent_executions,
    (SELECT COUNT(*) FROM chain_executions ce WHERE ce.user_id = $1::uuid AND ce.created_at >= $2) as chain_executions,
    (SELECT AVG(ae.execution_time_ms) FROM agent_executions ae WHERE ae.user_id = $1::uuid AND ae.created_at >= $2) as avg_agent_time_ms,
    (SELECT AVG(ce.execution_time_ms) FROM chain_executions ce WHERE ce.user_id = $1::uuid AND ce.created_at >= $2) as avg_chain_time_ms,
    (SELECT SUM(ce.tokens_used) FROM chain_executions ce WHERE ce.user_id = $1::uuid AND ce.created_at >= $2) as total_tokens_used;

-- name: GetExecutionTrends :many
SELECT
    DATE_TRUNC('day', ae.created_at) as execution_date,
    'agent' as execution_type,
    ae.agent_type as type_name,
    COUNT(*) as execution_count,
    COUNT(*) FILTER (WHERE ae.success = true) as successful_count,
    AVG(ae.execution_time_ms) as avg_execution_time_ms
FROM agent_executions ae
WHERE ae.user_id = $1::uuid AND ae.created_at >= $2
GROUP BY DATE_TRUNC('day', ae.created_at), ae.agent_type
UNION ALL
SELECT
    DATE_TRUNC('day', ce.created_at) as execution_date,
    'chain' as execution_type,
    ce.chain_type as type_name,
    COUNT(*) as execution_count,
    COUNT(*) FILTER (WHERE ce.success = true) as successful_count,
    AVG(ce.execution_time_ms) as avg_execution_time_ms
FROM chain_executions ce
WHERE ce.user_id = $1::uuid AND ce.created_at >= $2
GROUP BY DATE_TRUNC('day', ce.created_at), ce.chain_type
ORDER BY execution_date DESC, execution_type, type_name;
