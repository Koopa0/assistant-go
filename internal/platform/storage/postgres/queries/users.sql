-- name: CreateUser :one
INSERT INTO users (
    username, email, password_hash, full_name, avatar_url, preferences
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING id, username, email, full_name, avatar_url, preferences, is_active, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, username, email, full_name, avatar_url, preferences, is_active, created_at, updated_at 
FROM users 
WHERE id = $1 AND is_active = true;

-- name: GetUserByEmail :one
SELECT id, username, email, password_hash, full_name, avatar_url, preferences, is_active, created_at, updated_at 
FROM users 
WHERE email = $1 AND is_active = true;

-- name: GetUserByUsername :one
SELECT id, username, email, full_name, avatar_url, preferences, is_active, created_at, updated_at 
FROM users 
WHERE username = $1 AND is_active = true;

-- name: UpdateUserProfile :one
UPDATE users SET
    full_name = COALESCE($2, full_name),
    avatar_url = COALESCE($3, avatar_url),
    updated_at = NOW()
WHERE id = $1 AND is_active = true
RETURNING id, username, email, full_name, avatar_url, preferences, is_active, created_at, updated_at;

-- name: UpdateUserPreferences :one
UPDATE users SET
    preferences = $2,
    updated_at = NOW()
WHERE id = $1 AND is_active = true
RETURNING id, username, email, full_name, avatar_url, preferences, is_active, created_at, updated_at;

-- name: UpdateUserPassword :one
UPDATE users SET
    password_hash = $2,
    updated_at = NOW()
WHERE id = $1 AND is_active = true
RETURNING id, username, email, full_name, avatar_url, preferences, is_active, created_at, updated_at;

-- name: DeactivateUser :one
UPDATE users SET
    is_active = false,
    updated_at = NOW()
WHERE id = $1
RETURNING id, username, email, full_name, avatar_url, preferences, is_active, created_at, updated_at;

-- name: GetUserStatistics :one
SELECT 
    u.id,
    u.username,
    u.email,
    u.full_name,
    u.created_at,
    COUNT(DISTINCT c.id)::integer as total_conversations,
    COUNT(DISTINCT te.id)::integer as total_tools_used,
    COALESCE(SUM(apu.input_tokens + apu.output_tokens), 0)::integer as total_tokens_used,
    COALESCE(SUM(apu.cost_cents), 0)::integer as total_cost_cents
FROM users u
LEFT JOIN conversations c ON u.id = c.user_id
LEFT JOIN tool_executions te ON te.message_id IN (
    SELECT m.id FROM messages m 
    JOIN conversations conv ON m.conversation_id = conv.id 
    WHERE conv.user_id = u.id
)
LEFT JOIN ai_provider_usage apu ON u.id = apu.user_id
WHERE u.id = $1 AND u.is_active = true
GROUP BY u.id, u.username, u.email, u.full_name, u.created_at;

-- name: GetActiveUsers :many
SELECT id, username, email, full_name, avatar_url, preferences, is_active, created_at, updated_at
FROM users 
WHERE is_active = true
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountActiveUsers :one
SELECT COUNT(*) FROM users WHERE is_active = true;

-- name: SearchUsers :many
SELECT id, username, email, full_name, avatar_url, preferences, is_active, created_at, updated_at
FROM users 
WHERE is_active = true 
  AND (
    username ILIKE '%' || $1 || '%' OR 
    email ILIKE '%' || $1 || '%' OR 
    full_name ILIKE '%' || $1 || '%'
  )
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetUserActivitySummary :one
SELECT 
    u.id,
    COUNT(DISTINCT c.id)::integer as conversations_count,
    COUNT(DISTINCT m.id)::integer as messages_count,
    COUNT(DISTINCT te.id)::integer as tool_executions_count,
    MAX(m.created_at) as last_activity_at,
    EXTRACT(EPOCH FROM (NOW() - u.created_at))::integer as days_since_signup
FROM users u
LEFT JOIN conversations c ON u.id = c.user_id
LEFT JOIN messages m ON c.id = m.conversation_id
LEFT JOIN tool_executions te ON m.id = te.message_id
WHERE u.id = $1 AND u.is_active = true
GROUP BY u.id, u.created_at;

-- User API Keys related queries (to be implemented after migration)

-- User preferences and settings
-- name: GetUserSettings :one
SELECT 
    id,
    preferences,
    (preferences->>'language')::text as language,
    (preferences->>'theme')::text as theme,
    (preferences->>'defaultProgrammingLanguage')::text as default_programming_language,
    (preferences->>'emailNotifications')::boolean as email_notifications,
    (preferences->>'timezone')::text as timezone
FROM users 
WHERE id = $1 AND is_active = true;

-- name: UpdateUserSettings :one
UPDATE users SET
    preferences = preferences || $2,
    updated_at = NOW()
WHERE id = $1 AND is_active = true
RETURNING id, username, email, full_name, avatar_url, preferences, is_active, created_at, updated_at;

-- User favorite tools
-- name: GetUserFavoriteTools :many
SELECT 
    (preferences->'favoriteTools')::jsonb as favorite_tools
FROM users 
WHERE id = $1 AND is_active = true;

-- name: AddFavoriteTool :one
UPDATE users SET
    preferences = jsonb_set(
        preferences, 
        '{favoriteTools}', 
        COALESCE(preferences->'favoriteTools', '[]'::jsonb) || $2::jsonb,
        true
    ),
    updated_at = NOW()
WHERE id = $1 AND is_active = true
RETURNING id, username, email, full_name, avatar_url, preferences, is_active, created_at, updated_at;

-- name: RemoveFavoriteTool :one
UPDATE users SET
    preferences = jsonb_set(
        preferences,
        '{favoriteTools}',
        (preferences->'favoriteTools') - $2::text,
        true
    ),
    updated_at = NOW()
WHERE id = $1 AND is_active = true
RETURNING id, username, email, full_name, avatar_url, preferences, is_active, created_at, updated_at;