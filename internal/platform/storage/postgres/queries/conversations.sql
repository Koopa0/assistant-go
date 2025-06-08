-- name: CreateConversation :one
INSERT INTO conversations (user_id, title, summary, metadata)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, title, summary, metadata, is_archived, created_at, updated_at;

-- name: GetConversation :one
SELECT id, user_id, title, summary, metadata, is_archived, created_at, updated_at
FROM conversations
WHERE id = $1;

-- name: GetConversationsByUser :many
SELECT id, user_id, title, summary, metadata, is_archived, created_at, updated_at
FROM conversations
WHERE user_id = $1
ORDER BY updated_at DESC;

-- name: UpdateConversation :one
UPDATE conversations
SET title = $2, metadata = $3, updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, title, summary, metadata, is_archived, created_at, updated_at;

-- name: ArchiveConversation :exec
UPDATE conversations
SET is_archived = true, updated_at = NOW()
WHERE id = $1;

-- name: DeleteConversation :exec
DELETE FROM conversations
WHERE id = $1;

-- name: GetConversationCount :one
SELECT COUNT(*)
FROM conversations
WHERE user_id = $1;

-- name: GetRecentConversations :many
SELECT id, user_id, title, summary, metadata, is_archived, created_at, updated_at
FROM conversations
WHERE user_id = $1
ORDER BY updated_at DESC
LIMIT $2;

-- name: SearchConversations :many
SELECT id, user_id, title, summary, metadata, is_archived, created_at, updated_at
FROM conversations
WHERE user_id = $1 
  AND (title ILIKE '%' || $2 || '%' OR metadata::text ILIKE '%' || $2 || '%')
ORDER BY updated_at DESC
LIMIT $3 OFFSET $4;
