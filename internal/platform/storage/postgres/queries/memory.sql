-- name: CreateMemoryEntry :one
-- Creates a new memory entry with explicit column selection
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
    @memory_type::text,
    @user_id::uuid,
    @session_id::text,
    @content::text,
    @importance::float,
    COALESCE(@access_count::int, 0),
    COALESCE(@last_access::timestamptz, NOW()),
    @expires_at::timestamptz,
    COALESCE(@metadata::jsonb, '{}'::jsonb)
) RETURNING 
    id,
    memory_type,
    user_id,
    session_id,
    content,
    importance,
    access_count,
    last_access,
    expires_at,
    metadata,
    created_at,
    updated_at;

-- name: GetMemoryEntry :one
-- Retrieves a memory entry by ID with explicit columns
SELECT 
    id,
    memory_type,
    user_id,
    session_id,
    content,
    importance,
    access_count,
    last_access,
    expires_at,
    metadata,
    created_at,
    updated_at
FROM memory_entries
WHERE id = @id::uuid;

-- name: GetMemoryEntriesByUser :many
-- Gets all non-expired memory entries for a user
SELECT 
    id,
    memory_type,
    user_id,
    session_id,
    content,
    importance,
    access_count,
    last_access,
    expires_at,
    metadata,
    created_at,
    updated_at
FROM memory_entries
WHERE user_id = @user_id::uuid
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY importance DESC, last_access DESC;

-- name: GetMemoryEntriesBySession :many
-- Gets memory entries for a specific session with pagination
SELECT 
    id,
    memory_type,
    user_id,
    session_id,
    content,
    importance,
    access_count,
    last_access,
    expires_at,
    metadata,
    created_at,
    updated_at
FROM memory_entries
WHERE user_id = @user_id::uuid
  AND session_id = @session_id::text
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY last_access DESC
LIMIT @limit_val::int OFFSET @offset_val::int;

-- name: UpdateMemoryEntry :one
-- Updates a memory entry with optimistic locking
UPDATE memory_entries
SET 
    content = @content::text,
    importance = @importance::float,
    access_count = @access_count::int,
    last_access = @last_access::timestamptz,
    expires_at = @expires_at::timestamptz,
    metadata = COALESCE(@metadata::jsonb, metadata),
    updated_at = NOW()
WHERE id = @id::uuid
  AND (@expected_version::int IS NULL OR updated_at = @expected_updated_at::timestamptz)
RETURNING 
    id,
    memory_type,
    user_id,
    session_id,
    content,
    importance,
    access_count,
    last_access,
    expires_at,
    metadata,
    created_at,
    updated_at;

-- name: IncrementMemoryAccess :exec
-- Atomically increments access count and updates last access time
UPDATE memory_entries
SET 
    access_count = access_count + 1,
    last_access = NOW(),
    updated_at = NOW()
WHERE id = @id::uuid;

-- name: DeleteMemoryEntry :exec
-- Deletes a memory entry by ID
DELETE FROM memory_entries
WHERE id = @id::uuid;

-- name: DeleteExpiredMemoryEntries :execrows
-- Deletes expired memory entries and returns count
DELETE FROM memory_entries
WHERE expires_at IS NOT NULL 
  AND expires_at <= NOW();

-- name: DeleteMemoryEntriesByUser :execrows
-- Deletes memory entries by user with optional filters
DELETE FROM memory_entries
WHERE user_id = @user_id::uuid
  AND (@memory_types::text[] IS NULL OR memory_type = ANY(@memory_types::text[]))
  AND (@created_before::timestamptz IS NULL OR created_at < @created_before::timestamptz);

-- name: GetMemoryStats :many
-- Gets memory statistics grouped by type
SELECT
    memory_type,
    COUNT(*)::bigint as entry_count,
    AVG(importance)::float as avg_importance,
    MIN(created_at) as oldest_entry,
    MAX(created_at) as newest_entry,
    SUM(access_count)::bigint as total_accesses
FROM memory_entries
WHERE user_id = @user_id::uuid
  AND (expires_at IS NULL OR expires_at > NOW())
GROUP BY memory_type
ORDER BY entry_count DESC;

-- name: SearchMemoryEntries :many
-- Searches memory entries with full-text search and filters
SELECT 
    id,
    memory_type,
    user_id,
    session_id,
    content,
    importance,
    access_count,
    last_access,
    expires_at,
    metadata,
    created_at,
    updated_at,
    ts_rank(to_tsvector('english', content), plainto_tsquery('english', @search_query::text)) as relevance
FROM memory_entries
WHERE user_id = @user_id::uuid
  AND (@memory_types::text[] IS NULL OR memory_type = ANY(@memory_types::text[]))
  AND (
    @search_query::text IS NULL 
    OR to_tsvector('english', content) @@ plainto_tsquery('english', @search_query::text)
    OR content ILIKE '%' || @search_query::text || '%'
  )
  AND (expires_at IS NULL OR expires_at > NOW())
  AND importance >= COALESCE(@min_importance::float, 0.0)
ORDER BY 
    CASE WHEN @search_query::text IS NOT NULL THEN relevance ELSE 0 END DESC,
    importance DESC, 
    last_access DESC
LIMIT @limit_val::int OFFSET @offset_val::int;

-- =====================================================
-- BATCH OPERATIONS FOR PERFORMANCE
-- =====================================================

-- name: BatchCreateMemoryEntries :copyfrom
-- Bulk insert memory entries for performance
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
    $1, $2, $3, $4, $5, $6, $7, $8, $9
);

-- name: BatchIncrementMemoryAccess :exec
-- Batch update access counts for multiple entries
UPDATE memory_entries
SET 
    access_count = access_count + 1,
    last_access = NOW(),
    updated_at = NOW()
WHERE id = ANY(@ids::uuid[]);

-- =====================================================
-- MEMORY LIFECYCLE OPERATIONS
-- =====================================================

-- name: ArchiveOldMemoryEntries :execrows
-- Archives old memory entries by reducing their importance
UPDATE memory_entries
SET 
    importance = importance * @decay_factor::float,
    updated_at = NOW()
WHERE user_id = @user_id::uuid
  AND last_access < @threshold_date::timestamptz
  AND importance > @min_importance::float
  AND memory_type != 'semantic'; -- Semantic memories don't decay

-- name: ConsolidateMemoryEntries :many
-- Finds similar memory entries for consolidation
WITH ranked_memories AS (
    SELECT 
        m1.id as id1,
        m2.id as id2,
        m1.content as content1,
        m2.content as content2,
        m1.importance + m2.importance as combined_importance,
        similarity(m1.content, m2.content) as content_similarity
    FROM memory_entries m1
    JOIN memory_entries m2 ON m1.user_id = m2.user_id 
        AND m1.memory_type = m2.memory_type
        AND m1.id < m2.id
    WHERE m1.user_id = @user_id::uuid
      AND m1.memory_type = @memory_type::text
      AND similarity(m1.content, m2.content) > @similarity_threshold::float
)
SELECT 
    id1,
    id2,
    content1,
    content2,
    combined_importance,
    content_similarity
FROM ranked_memories
ORDER BY content_similarity DESC
LIMIT @limit_val::int;

-- =====================================================
-- MEMORY RELATIONSHIPS AND GRAPH OPERATIONS
-- =====================================================

-- name: CreateMemoryRelation :one
-- Creates a relationship between two memory entries
INSERT INTO memory_relations (
    from_memory_id,
    to_memory_id,
    relation_type,
    weight,
    metadata
) VALUES (
    @from_id::uuid,
    @to_id::uuid,
    @relation_type::text,
    COALESCE(@weight::float, 0.5),
    COALESCE(@metadata::jsonb, '{}'::jsonb)
) ON CONFLICT (from_memory_id, to_memory_id, relation_type) 
DO UPDATE SET 
    weight = GREATEST(memory_relations.weight, EXCLUDED.weight),
    metadata = memory_relations.metadata || EXCLUDED.metadata,
    updated_at = NOW()
RETURNING 
    id,
    from_memory_id,
    to_memory_id,
    relation_type,
    weight,
    metadata,
    created_at,
    updated_at;

-- name: GetMemoryRelations :many
-- Gets all relationships for a memory entry
SELECT 
    mr.id,
    mr.from_memory_id,
    mr.to_memory_id,
    mr.relation_type,
    mr.weight,
    mr.metadata,
    mr.created_at,
    mr.updated_at,
    me_from.content as from_content,
    me_to.content as to_content
FROM memory_relations mr
JOIN memory_entries me_from ON mr.from_memory_id = me_from.id
JOIN memory_entries me_to ON mr.to_memory_id = me_to.id
WHERE (mr.from_memory_id = @memory_id::uuid OR mr.to_memory_id = @memory_id::uuid)
ORDER BY mr.weight DESC, mr.created_at DESC;

-- name: GetRelatedMemories :many
-- Gets memory entries related to a given entry through relationships
WITH RECURSIVE related AS (
    -- Base case: direct relationships
    SELECT 
        CASE 
            WHEN mr.from_memory_id = @memory_id::uuid THEN mr.to_memory_id
            ELSE mr.from_memory_id
        END as related_id,
        mr.weight,
        1 as depth
    FROM memory_relations mr
    WHERE (mr.from_memory_id = @memory_id::uuid OR mr.to_memory_id = @memory_id::uuid)
      AND (@relation_types::text[] IS NULL OR mr.relation_type = ANY(@relation_types::text[]))
    
    UNION
    
    -- Recursive case: indirect relationships (up to max depth)
    SELECT 
        CASE 
            WHEN mr.from_memory_id = r.related_id THEN mr.to_memory_id
            ELSE mr.from_memory_id
        END as related_id,
        r.weight * mr.weight as weight,
        r.depth + 1
    FROM related r
    JOIN memory_relations mr ON (mr.from_memory_id = r.related_id OR mr.to_memory_id = r.related_id)
    WHERE r.depth < @max_depth::int
      AND (@relation_types::text[] IS NULL OR mr.relation_type = ANY(@relation_types::text[]))
)
SELECT DISTINCT
    me.id,
    me.memory_type,
    me.user_id,
    me.session_id,
    me.content,
    me.importance,
    me.access_count,
    me.last_access,
    me.expires_at,
    me.metadata,
    me.created_at,
    me.updated_at,
    MAX(r.weight) as relationship_weight,
    MIN(r.depth) as relationship_depth
FROM related r
JOIN memory_entries me ON me.id = r.related_id
WHERE me.id != @memory_id::uuid
GROUP BY me.id
ORDER BY relationship_weight DESC, relationship_depth ASC
LIMIT @limit_val::int;

-- =====================================================
-- PERFORMANCE AND MAINTENANCE QUERIES
-- =====================================================

-- name: GetMemoryUsageByUser :one
-- Gets memory usage statistics for capacity planning
SELECT 
    COUNT(*)::bigint as total_entries,
    COUNT(DISTINCT memory_type) as type_count,
    SUM(LENGTH(content))::bigint as total_content_size,
    AVG(LENGTH(content))::float as avg_content_size,
    MAX(access_count)::int as max_access_count,
    AVG(access_count)::float as avg_access_count,
    COUNT(CASE WHEN expires_at IS NOT NULL THEN 1 END)::bigint as expiring_entries,
    COUNT(CASE WHEN importance > 0.7 THEN 1 END)::bigint as high_importance_entries
FROM memory_entries
WHERE user_id = @user_id::uuid;

-- name: OptimizeMemoryIndices :exec
-- Maintenance query to update memory statistics
ANALYZE memory_entries;

-- name: GetMemoryAccessPatterns :many
-- Analyzes memory access patterns for optimization
SELECT 
    DATE_TRUNC('hour', last_access) as access_hour,
    memory_type,
    COUNT(*)::bigint as access_count,
    AVG(importance)::float as avg_importance
FROM memory_entries
WHERE user_id = @user_id::uuid
  AND last_access >= NOW() - INTERVAL '7 days'
GROUP BY DATE_TRUNC('hour', last_access), memory_type
ORDER BY access_hour DESC, access_count DESC;
