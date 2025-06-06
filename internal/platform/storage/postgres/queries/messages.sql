-- name: CreateMessage :one
INSERT INTO messages (conversation_id, role, content, token_count, metadata)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, conversation_id, role, content, token_count, metadata, created_at;

-- name: GetMessage :one
SELECT id, conversation_id, role, content, token_count, metadata, created_at
FROM messages
WHERE id = $1;

-- name: GetMessagesByConversation :many
SELECT id, conversation_id, role, content, token_count, metadata, created_at
FROM messages
WHERE conversation_id = $1
ORDER BY created_at ASC
LIMIT $2 OFFSET $3;

-- name: GetAllMessagesByConversation :many
SELECT id, conversation_id, role, content, token_count, metadata, created_at
FROM messages
WHERE conversation_id = $1
ORDER BY created_at ASC;

-- name: UpdateMessage :one
UPDATE messages
SET content = $2, token_count = $3, metadata = $4
WHERE id = $1
RETURNING id, conversation_id, role, content, token_count, metadata, created_at;

-- name: DeleteMessage :exec
DELETE FROM messages
WHERE id = $1;

-- name: DeleteMessagesByConversation :exec
DELETE FROM messages
WHERE conversation_id = $1;

-- name: GetMessageCount :one
SELECT COUNT(*)
FROM messages
WHERE conversation_id = $1;

-- name: GetRecentMessages :many
SELECT id, conversation_id, role, content, token_count, metadata, created_at
FROM messages
WHERE conversation_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: GetMessagesByRole :many
SELECT id, conversation_id, role, content, token_count, metadata, created_at
FROM messages
WHERE conversation_id = $1 AND role = $2
ORDER BY created_at ASC;

-- name: SearchMessages :many
SELECT id, conversation_id, role, content, token_count, metadata, created_at
FROM messages
WHERE conversation_id = $1 
  AND content ILIKE '%' || $2 || '%'
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
