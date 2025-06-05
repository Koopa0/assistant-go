-- name: CreateMemoryEntry :one
INSERT INTO memory_entries (
    memory_type,
    user_id,
    session_id,
    content,
    importance,
    access_count,
    last_access,
    expires_at,
    metadata
) VALUES (
    $1, $2::uuid, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetMemoryEntry :one
SELECT * FROM memory_entries
WHERE id = $1;

-- name: GetMemoryEntriesByUser :many
SELECT * FROM memory_entries
WHERE user_id = $1::uuid
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY importance DESC, last_access DESC;

-- name: GetMemoryEntriesBySession :many
SELECT * FROM memory_entries
WHERE user_id = $1::uuid
  AND session_id = $2
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY last_access DESC
LIMIT $3 OFFSET $4;

-- name: UpdateMemoryEntry :one
UPDATE memory_entries
SET content = $2,
    importance = $3,
    access_count = $4,
    last_access = $5,
    expires_at = $6,
    metadata = $7
WHERE id = $1
RETURNING *;

-- name: IncrementMemoryAccess :exec
UPDATE memory_entries
SET access_count = $2,
    last_access = $3
WHERE id = $1;

-- name: DeleteMemoryEntry :exec
DELETE FROM memory_entries
WHERE id = $1;

-- name: DeleteExpiredMemoryEntries :exec
DELETE FROM memory_entries
WHERE expires_at IS NOT NULL AND expires_at <= NOW();

-- name: DeleteMemoryEntriesByUser :exec
DELETE FROM memory_entries
WHERE user_id = $1::uuid
  AND (memory_type = ANY($2::text[]) OR $2 IS NULL)
  AND (created_at < $3 OR $3 IS NULL);

-- name: GetMemoryStats :one
SELECT
    memory_type,
    COUNT(*) as entry_count,
    AVG(importance) as avg_importance,
    MIN(created_at) as oldest_entry,
    MAX(created_at) as newest_entry
FROM memory_entries
WHERE user_id = $1::uuid
  AND (expires_at IS NULL OR expires_at > NOW())
GROUP BY memory_type;

-- name: SearchMemoryEntries :many
SELECT * FROM memory_entries
WHERE user_id = $1::uuid
  AND (memory_type = ANY($2::text[]) OR $2 IS NULL)
  AND content ILIKE '%' || $3 || '%'
  AND (expires_at IS NULL OR expires_at > NOW())
  AND importance >= $4
ORDER BY importance DESC, last_access DESC
LIMIT $5 OFFSET $6;
