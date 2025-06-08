-- Additional conversation queries that were missing

-- name: ArchiveConversation :exec
UPDATE conversations
SET is_archived = true,
    updated_at = NOW()
WHERE id = $1;

-- name: UnarchiveConversation :exec
UPDATE conversations
SET is_archived = false,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateConversationSummary :exec
UPDATE conversations
SET summary = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: GetArchivedConversations :many
SELECT id, user_id, title, summary, metadata, is_archived, created_at, updated_at
FROM conversations
WHERE user_id = $1 AND is_archived = true
ORDER BY updated_at DESC;

-- name: GetActiveConversations :many
SELECT id, user_id, title, summary, metadata, is_archived, created_at, updated_at
FROM conversations
WHERE user_id = $1 AND is_archived = false
ORDER BY updated_at DESC
LIMIT $2;