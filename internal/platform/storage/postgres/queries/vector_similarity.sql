-- name: SearchEmbeddingsByTypeWithMetadata :many
-- Search embeddings with metadata filtering
SELECT 
    id,
    content_type,
    content_id,
    content_text,
    embedding,
    metadata,
    created_at,
    1 - (embedding <=> $1::vector) as similarity
FROM embeddings
WHERE content_type = $2
    AND ($3::jsonb IS NULL OR metadata @> $3::jsonb)
    AND 1 - (embedding <=> $1::vector) > $4
ORDER BY embedding <=> $1::vector
LIMIT $5;

-- name: DeleteEmbeddingsByContentType :exec
-- Delete all embeddings for a specific content type
DELETE FROM embeddings
WHERE content_type = $1;

-- name: DeleteEmbeddingsByMetadata :exec
-- Delete embeddings matching specific metadata criteria
DELETE FROM embeddings
WHERE metadata @> $1::jsonb;

-- name: CountEmbeddingsByType :one
-- Count embeddings by content type
SELECT COUNT(*) as count
FROM embeddings
WHERE content_type = $1;

-- name: GetEmbeddingStats :many
-- Get statistics about embeddings grouped by content type
SELECT 
    content_type,
    COUNT(*) as count,
    MAX(created_at) as latest_created,
    AVG(array_length(embedding, 1)) as avg_dimensions
FROM embeddings
GROUP BY content_type
ORDER BY count DESC;

-- name: DeleteExpiredEmbeddings :exec
-- Delete embeddings older than a specific date
DELETE FROM embeddings
WHERE created_at < $1
    AND ($2::text IS NULL OR content_type = $2);

-- name: UpdateEmbeddingMetadata :exec
-- Update metadata for a specific embedding
UPDATE embeddings
SET metadata = $2
WHERE id = $1;

-- name: BulkDeleteEmbeddings :exec
-- Delete multiple embeddings by IDs
DELETE FROM embeddings
WHERE id = ANY($1::uuid[]);