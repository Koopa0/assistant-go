-- name: CreateEmbedding :one
INSERT INTO embeddings (content_type, content_id, content_text, embedding, metadata)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (content_type, content_id) 
DO UPDATE SET 
    content_text = EXCLUDED.content_text,
    embedding = EXCLUDED.embedding,
    metadata = EXCLUDED.metadata,
    created_at = NOW()
RETURNING id, content_type, content_id, content_text, embedding, metadata, created_at;

-- name: GetEmbedding :one
SELECT id, content_type, content_id, content_text, embedding, metadata, created_at
FROM embeddings
WHERE content_type = $1 AND content_id = $2;

-- name: GetEmbeddingByID :one
SELECT id, content_type, content_id, content_text, embedding, metadata, created_at
FROM embeddings
WHERE id = $1;

-- name: SearchSimilarEmbeddings :many
SELECT 
    id, 
    content_type, 
    content_id, 
    content_text, 
    embedding, 
    metadata, 
    created_at,
    1 - (embedding <=> sqlc.arg(query_embedding)::vector) AS similarity
FROM embeddings
WHERE content_type = sqlc.arg(content_type)
  AND 1 - (embedding <=> sqlc.arg(query_embedding)::vector) > sqlc.arg(threshold)::float8
ORDER BY embedding <=> sqlc.arg(query_embedding)::vector
LIMIT sqlc.arg(result_limit);

-- name: SearchSimilarEmbeddingsAllTypes :many
SELECT 
    id, 
    content_type, 
    content_id, 
    content_text, 
    embedding, 
    metadata, 
    created_at,
    1 - (embedding <=> $1::vector) AS similarity
FROM embeddings
WHERE 1 - (embedding <=> $1::vector) > $2
ORDER BY embedding <=> $1::vector
LIMIT $3;

-- name: DeleteEmbedding :exec
DELETE FROM embeddings
WHERE content_type = $1 AND content_id = $2;

-- name: DeleteEmbeddingByID :exec
DELETE FROM embeddings
WHERE id = $1;

-- name: GetEmbeddingsByType :many
SELECT id, content_type, content_id, content_text, embedding, metadata, created_at
FROM embeddings
WHERE content_type = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetEmbeddingCount :one
SELECT COUNT(*)
FROM embeddings
WHERE content_type = $1;

-- name: GetAllEmbeddingCount :one
SELECT COUNT(*)
FROM embeddings;

-- name: UpdateEmbedding :one
UPDATE embeddings
SET content_text = $3, embedding = $4, metadata = $5
WHERE content_type = $1 AND content_id = $2
RETURNING id, content_type, content_id, content_text, embedding, metadata, created_at;
