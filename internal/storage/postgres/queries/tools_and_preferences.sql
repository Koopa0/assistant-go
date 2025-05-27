-- Tool cache queries

-- name: CreateToolCacheEntry :one
INSERT INTO tool_cache (
    user_id,
    tool_name,
    input_hash,
    input_data,
    output_data,
    execution_time_ms,
    success,
    error_message,
    expires_at,
    metadata
) VALUES (
    $1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10
) ON CONFLICT (user_id, tool_name, input_hash)
DO UPDATE SET
    output_data = EXCLUDED.output_data,
    execution_time_ms = EXCLUDED.execution_time_ms,
    success = EXCLUDED.success,
    error_message = EXCLUDED.error_message,
    hit_count = tool_cache.hit_count + 1,
    last_hit = NOW(),
    expires_at = EXCLUDED.expires_at,
    metadata = EXCLUDED.metadata
RETURNING *;

-- name: GetToolCacheEntry :one
SELECT * FROM tool_cache
WHERE user_id = $1::uuid AND tool_name = $2 AND input_hash = $3
  AND expires_at > NOW();

-- name: UpdateToolCacheHit :one
UPDATE tool_cache
SET hit_count = hit_count + 1,
    last_hit = NOW()
WHERE id = $1
RETURNING *;

-- name: GetToolCacheByUser :many
SELECT * FROM tool_cache
WHERE user_id = $1::uuid
  AND (tool_name = $2 OR $2 IS NULL)
  AND expires_at > NOW()
ORDER BY last_hit DESC
LIMIT $3 OFFSET $4;

-- name: DeleteExpiredToolCache :exec
DELETE FROM tool_cache
WHERE expires_at <= NOW();

-- name: DeleteToolCacheEntry :exec
DELETE FROM tool_cache
WHERE id = $1;

-- name: DeleteToolCacheByUser :exec
DELETE FROM tool_cache
WHERE user_id = $1::uuid
  AND (tool_name = $2 OR $2 IS NULL)
  AND (created_at < $3 OR $3 IS NULL);

-- name: GetToolCacheStats :one
SELECT
    tool_name,
    COUNT(*) as cache_entries,
    SUM(hit_count) as total_hits,
    AVG(execution_time_ms) as avg_execution_time_ms,
    MAX(last_hit) as last_used
FROM tool_cache
WHERE user_id = $1::uuid
  AND expires_at > NOW()
GROUP BY tool_name;

-- User preferences queries

-- name: CreateUserPreference :one
INSERT INTO user_preferences (
    user_id,
    category,
    preference_key,
    preference_value,
    value_type,
    description,
    metadata
) VALUES (
    $1::uuid, $2, $3, $4, $5, $6, $7
) ON CONFLICT (user_id, category, preference_key)
DO UPDATE SET
    preference_value = EXCLUDED.preference_value,
    value_type = EXCLUDED.value_type,
    description = EXCLUDED.description,
    metadata = EXCLUDED.metadata,
    updated_at = NOW()
RETURNING *;

-- name: GetUserPreference :one
SELECT * FROM user_preferences
WHERE user_id = $1::uuid AND category = $2 AND preference_key = $3;

-- name: GetUserPreferencesByCategory :many
SELECT * FROM user_preferences
WHERE user_id = $1::uuid AND category = $2
ORDER BY preference_key;

-- name: GetAllUserPreferences :many
SELECT * FROM user_preferences
WHERE user_id = $1::uuid
ORDER BY category, preference_key;

-- name: UpdateUserPreference :one
UPDATE user_preferences
SET preference_value = $4,
    value_type = $5,
    description = $6,
    metadata = $7
WHERE user_id = $1::uuid AND category = $2 AND preference_key = $3
RETURNING *;

-- name: DeleteUserPreference :exec
DELETE FROM user_preferences
WHERE user_id = $1::uuid AND category = $2 AND preference_key = $3;

-- name: DeleteUserPreferencesByCategory :exec
DELETE FROM user_preferences
WHERE user_id = $1::uuid AND category = $2;

-- User context queries

-- name: CreateUserContext :one
INSERT INTO user_context (
    user_id,
    context_type,
    context_key,
    context_value,
    importance,
    expires_at,
    metadata
) VALUES (
    $1::uuid, $2, $3, $4, $5, $6, $7
) ON CONFLICT (user_id, context_type, context_key)
DO UPDATE SET
    context_value = EXCLUDED.context_value,
    importance = EXCLUDED.importance,
    expires_at = EXCLUDED.expires_at,
    metadata = EXCLUDED.metadata,
    updated_at = NOW()
RETURNING *;

-- name: GetUserContext :one
SELECT * FROM user_context
WHERE user_id = $1::uuid AND context_type = $2 AND context_key = $3
  AND (expires_at IS NULL OR expires_at > NOW());

-- name: GetUserContextByType :many
SELECT * FROM user_context
WHERE user_id = $1::uuid AND context_type = $2
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY importance DESC, updated_at DESC;

-- name: GetAllUserContext :many
SELECT * FROM user_context
WHERE user_id = $1::uuid
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY context_type, importance DESC;

-- name: UpdateUserContext :one
UPDATE user_context
SET context_value = $4,
    importance = $5,
    expires_at = $6,
    metadata = $7
WHERE user_id = $1::uuid AND context_type = $2 AND context_key = $3
RETURNING *;

-- name: DeleteUserContext :exec
DELETE FROM user_context
WHERE user_id = $1::uuid AND context_type = $2 AND context_key = $3;

-- name: DeleteExpiredUserContext :exec
DELETE FROM user_context
WHERE expires_at IS NOT NULL AND expires_at <= NOW();

-- name: DeleteUserContextByType :exec
DELETE FROM user_context
WHERE user_id = $1::uuid AND context_type = $2;
