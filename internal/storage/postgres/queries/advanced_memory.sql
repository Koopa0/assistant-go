-- Advanced memory system queries

-- =====================================================
-- EPISODIC MEMORIES QUERIES
-- =====================================================

-- name: CreateEpisodicMemory :one
INSERT INTO episodic_memories (
    user_id,
    episode_type,
    episode_summary,
    full_context,
    emotional_valence,
    importance,
    vividness,
    embedding,
    temporal_context,
    spatial_context,
    social_context,
    causal_links,
    decay_rate
) VALUES (
    $1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: GetEpisodicMemory :one
SELECT * FROM episodic_memories
WHERE id = $1;

-- name: GetEpisodicMemories :many
SELECT * FROM episodic_memories
WHERE user_id = $1::uuid
  AND (episode_type = $2 OR $2 IS NULL)
  AND importance >= COALESCE($3, 0.0)
  AND vividness >= COALESCE($4, 0.0)
ORDER BY importance DESC, vividness DESC, created_at DESC
LIMIT $5 OFFSET $6;

-- name: SearchEpisodicMemoriesBySimilarity :many
SELECT 
    id,
    episode_type,
    episode_summary,
    importance,
    vividness,
    created_at,
    embedding <=> $2::vector as distance
FROM episodic_memories
WHERE user_id = $1::uuid
  AND embedding IS NOT NULL
ORDER BY embedding <=> $2::vector
LIMIT $3;

-- name: UpdateEpisodicMemoryAccess :one
UPDATE episodic_memories
SET access_count = access_count + 1,
    last_accessed = NOW(),
    vividness = vividness * (1 - decay_rate)
WHERE id = $1
RETURNING *;

-- name: GetMemoriesByTemporalContext :many
SELECT * FROM episodic_memories
WHERE user_id = $1::uuid
  AND temporal_context @> $2
ORDER BY importance DESC
LIMIT $3;

-- name: GetMemoriesByCausalLink :many
SELECT * FROM episodic_memories
WHERE user_id = $1::uuid
  AND $2::uuid = ANY(causal_links)
ORDER BY importance DESC;

-- name: UpdateEpisodicMemoryImportance :one
UPDATE episodic_memories
SET importance = $2,
    vividness = GREATEST(vividness, $2 * 0.8) -- Boost vividness for important memories
WHERE id = $1
RETURNING *;

-- name: AddCausalLink :one
UPDATE episodic_memories
SET causal_links = array_append(causal_links, $2::uuid)
WHERE id = $1
  AND NOT ($2::uuid = ANY(causal_links))
RETURNING *;

-- name: DeleteDecayedMemories :exec
DELETE FROM episodic_memories
WHERE user_id = $1::uuid
  AND vividness < $2
  AND importance < $3
  AND created_at < NOW() - INTERVAL '1 year';

-- =====================================================
-- SEMANTIC MEMORIES QUERIES
-- =====================================================

-- name: CreateSemanticMemory :one
INSERT INTO semantic_memories (
    user_id,
    knowledge_type,
    subject,
    predicate,
    object,
    confidence,
    source_type,
    source_references,
    embedding
) VALUES (
    $1::uuid, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetSemanticMemory :one
SELECT * FROM semantic_memories
WHERE id = $1;

-- name: GetSemanticMemories :many
SELECT * FROM semantic_memories
WHERE user_id = $1::uuid
  AND (knowledge_type = ANY($2::text[]) OR $2 IS NULL)
  AND is_active = true
  AND confidence >= COALESCE($3, 0.0)
ORDER BY confidence DESC, updated_at DESC
LIMIT $4 OFFSET $5;

-- name: SearchSemanticMemoriesBySubject :many
SELECT * FROM semantic_memories
WHERE user_id = $1::uuid
  AND to_tsvector('english', subject) @@ plainto_tsquery('english', $2)
  AND is_active = true
ORDER BY confidence DESC
LIMIT $3;

-- name: SearchSemanticMemoriesBySimilarity :many
SELECT 
    id,
    knowledge_type,
    subject,
    predicate,
    object,
    confidence,
    embedding <=> $2::vector as distance
FROM semantic_memories
WHERE user_id = $1::uuid
  AND embedding IS NOT NULL
  AND is_active = true
ORDER BY embedding <=> $2::vector
LIMIT $3;

-- name: GetSemanticRelationships :many
SELECT * FROM semantic_memories
WHERE user_id = $1::uuid
  AND (subject ILIKE '%' || $2 || '%' OR object::text ILIKE '%' || $2 || '%')
  AND is_active = true
ORDER BY confidence DESC;

-- name: UpdateSemanticMemoryConfidence :one
UPDATE semantic_memories
SET confidence = $2,
    confirmation_count = confirmation_count + CASE WHEN $3 THEN 1 ELSE 0 END,
    contradiction_count = contradiction_count + CASE WHEN NOT $3 THEN 1 ELSE 0 END,
    last_confirmed = CASE WHEN $3 THEN NOW() ELSE last_confirmed END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateConflictingMemories :exec
UPDATE semantic_memories
SET is_active = false,
    updated_at = NOW()
WHERE user_id = $1::uuid
  AND subject = $2
  AND predicate = $3
  AND object != $4
  AND confidence < $5;

-- name: GetKnowledgeGraph :many
SELECT 
    sm.subject,
    sm.predicate,
    sm.object,
    sm.confidence,
    sm.knowledge_type
FROM semantic_memories sm
WHERE sm.user_id = $1::uuid
  AND sm.is_active = true
  AND sm.confidence >= $2
ORDER BY sm.confidence DESC;

-- =====================================================
-- PROCEDURAL MEMORIES QUERIES
-- =====================================================

-- name: CreateProceduralMemory :one
INSERT INTO procedural_memories (
    user_id,
    procedure_name,
    procedure_type,
    trigger_conditions,
    steps,
    prerequisites,
    expected_outcomes
) VALUES (
    $1::uuid, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetProceduralMemory :one
SELECT * FROM procedural_memories
WHERE id = $1;

-- name: GetProceduralMemories :many
SELECT * FROM procedural_memories
WHERE user_id = $1::uuid
  AND (procedure_type = $2 OR $2 IS NULL)
ORDER BY success_rate DESC, execution_count DESC
LIMIT $3 OFFSET $4;

-- name: SearchProceduralMemoriesByConditions :many
SELECT * FROM procedural_memories
WHERE user_id = $1::uuid
  AND trigger_conditions @> $2
ORDER BY success_rate DESC, automation_confidence DESC
LIMIT $3;

-- name: UpdateProceduralExecution :one
UPDATE procedural_memories
SET execution_count = execution_count + 1,
    success_rate = (success_rate * execution_count + CASE WHEN $2 THEN 1.0 ELSE 0.0 END) / (execution_count + 1),
    average_duration_ms = CASE 
        WHEN $3 IS NOT NULL THEN 
            (COALESCE(average_duration_ms, 0) * execution_count + $3) / (execution_count + 1)
        ELSE average_duration_ms
    END,
    last_executed = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: OptimizeProcedure :one
UPDATE procedural_memories
SET steps = $2,
    optimization_history = optimization_history || jsonb_build_object(
        'timestamp', EXTRACT(EPOCH FROM NOW()),
        'old_steps', steps,
        'improvement_reason', $3,
        'performance_gain', $4
    ),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateAutomationConfidence :one
UPDATE procedural_memories
SET automation_confidence = $2,
    is_automated = CASE WHEN $2 >= $3 THEN true ELSE is_automated END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetAutomatableProcedures :many
SELECT * FROM procedural_memories
WHERE user_id = $1::uuid
  AND success_rate >= $2
  AND execution_count >= $3
  AND automation_confidence >= $4
  AND NOT is_automated
ORDER BY automation_confidence DESC, success_rate DESC;

-- =====================================================
-- WORKING MEMORY QUERIES
-- =====================================================

-- name: CreateWorkingMemorySlot :one
INSERT INTO working_memory (
    user_id,
    session_id,
    memory_slot,
    content_type,
    content,
    activation_level,
    linked_memories,
    expires_at
) VALUES (
    $1::uuid, $2, $3, $4, $5, $6, $7, $8
) ON CONFLICT (user_id, session_id, memory_slot)
DO UPDATE SET
    content_type = $4,
    content = $5,
    activation_level = $6,
    linked_memories = $7,
    reference_count = working_memory.reference_count + 1,
    last_accessed = NOW(),
    expires_at = $8
RETURNING *;

-- name: GetWorkingMemory :many
SELECT * FROM working_memory
WHERE user_id = $1::uuid
  AND session_id = $2
  AND expires_at > NOW()
ORDER BY memory_slot;

-- name: GetWorkingMemorySlot :one
SELECT * FROM working_memory
WHERE user_id = $1::uuid
  AND session_id = $2
  AND memory_slot = $3
  AND expires_at > NOW();

-- name: UpdateWorkingMemoryActivation :one
UPDATE working_memory
SET activation_level = $2,
    reference_count = reference_count + 1,
    last_accessed = NOW()
WHERE id = $1
RETURNING *;

-- name: ClearWorkingMemorySlot :exec
DELETE FROM working_memory
WHERE user_id = $1::uuid
  AND session_id = $2
  AND memory_slot = $3;

-- name: ClearExpiredWorkingMemory :exec
DELETE FROM working_memory
WHERE expires_at <= NOW();

-- name: ExtendWorkingMemoryExpiry :one
UPDATE working_memory
SET expires_at = $2,
    last_accessed = NOW()
WHERE id = $1
RETURNING *;

-- name: GetMostActiveWorkingMemory :many
SELECT * FROM working_memory
WHERE user_id = $1::uuid
  AND session_id = $2
  AND expires_at > NOW()
ORDER BY activation_level DESC, reference_count DESC
LIMIT $3;

-- name: ConsolidateWorkingMemory :many
SELECT 
    content_type,
    content,
    activation_level,
    reference_count,
    linked_memories
FROM working_memory
WHERE user_id = $1::uuid
  AND session_id = $2
  AND activation_level >= $3
  AND reference_count >= $4
  AND expires_at > NOW()
ORDER BY activation_level DESC;