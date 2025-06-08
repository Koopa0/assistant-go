-- Knowledge graph queries

-- =====================================================
-- KNOWLEDGE NODES QUERIES
-- =====================================================

-- name: CreateKnowledgeNode :one
INSERT INTO knowledge_nodes (
    user_id,
    node_type,
    node_name,
    display_name,
    description,
    properties,
    embedding,
    importance
) VALUES (
    $1::uuid, $2, $3, $4, $5, $6, $7, $8
) ON CONFLICT (user_id, node_type, node_name)
DO UPDATE SET
    display_name = COALESCE($4, knowledge_nodes.display_name),
    description = COALESCE($5, knowledge_nodes.description),
    properties = knowledge_nodes.properties || COALESCE($6, '{}'::jsonb),
    embedding = COALESCE($7, knowledge_nodes.embedding),
    importance = COALESCE($8, knowledge_nodes.importance),
    updated_at = NOW()
RETURNING id, user_id, node_type, node_name, display_name, description, properties, embedding, importance, access_frequency, last_accessed, created_at, updated_at, is_active;

-- name: GetKnowledgeNode :one
SELECT id, user_id, node_type, node_name, display_name, description, properties, embedding, importance, access_frequency, last_accessed, created_at, updated_at, is_active
FROM knowledge_nodes
WHERE id = $1;

-- name: GetKnowledgeNodeByName :one
SELECT id, user_id, node_type, node_name, display_name, description, properties, embedding, importance, access_frequency, last_accessed, created_at, updated_at, is_active
FROM knowledge_nodes
WHERE user_id = $1::uuid
  AND node_type = $2
  AND node_name = $3
  AND is_active = true;

-- name: GetKnowledgeNodes :many
SELECT id, user_id, node_type, node_name, display_name, description, properties, embedding, importance, access_frequency, last_accessed, created_at, updated_at, is_active
FROM knowledge_nodes
WHERE user_id = $1::uuid
  AND (node_type = ANY($2::text[]) OR $2 IS NULL)
  AND is_active = true
ORDER BY importance DESC, access_frequency DESC
LIMIT $3 OFFSET $4;

-- name: SearchKnowledgeNodesByName :many
SELECT id, user_id, node_type, node_name, display_name, description, properties, embedding, importance, access_frequency, last_accessed, created_at, updated_at, is_active
FROM knowledge_nodes
WHERE user_id = $1::uuid
  AND to_tsvector('english', node_name || ' ' || COALESCE(display_name, '') || ' ' || COALESCE(description, '')) 
      @@ plainto_tsquery('english', $2)
  AND is_active = true
ORDER BY importance DESC
LIMIT $3;

-- name: SearchKnowledgeNodesBySimilarity :many
SELECT 
    id,
    node_type,
    node_name,
    display_name,
    importance,
    embedding <=> $2::vector as distance
FROM knowledge_nodes
WHERE user_id = $1::uuid
  AND embedding IS NOT NULL
  AND is_active = true
ORDER BY embedding <=> $2::vector
LIMIT $3;

-- name: UpdateKnowledgeNodeAccess :one
UPDATE knowledge_nodes
SET access_frequency = access_frequency + 1,
    last_accessed = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, node_type, node_name, display_name, description, properties, embedding, importance, access_frequency, last_accessed, created_at, updated_at, is_active;

-- name: UpdateKnowledgeNodeProperties :one
UPDATE knowledge_nodes
SET properties = properties || $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, node_type, node_name, display_name, description, properties, embedding, importance, access_frequency, last_accessed, created_at, updated_at, is_active;

-- name: DeactivateKnowledgeNode :exec
UPDATE knowledge_nodes
SET is_active = false,
    updated_at = NOW()
WHERE id = $1;

-- name: GetNodesByImportance :many
SELECT id, user_id, node_type, node_name, display_name, description, properties, embedding, importance, access_frequency, last_accessed, created_at, updated_at, is_active
FROM knowledge_nodes
WHERE user_id = $1::uuid
  AND importance >= $2
  AND is_active = true
ORDER BY importance DESC
LIMIT $3;

-- =====================================================
-- KNOWLEDGE EDGES QUERIES
-- =====================================================

-- name: CreateKnowledgeEdge :one
INSERT INTO knowledge_edges (
    user_id,
    source_node_id,
    target_node_id,
    edge_type,
    strength,
    properties,
    evidence_count
) VALUES (
    $1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7
) ON CONFLICT (source_node_id, target_node_id, edge_type)
DO UPDATE SET
    strength = (knowledge_edges.strength + $5) / 2,
    properties = knowledge_edges.properties || COALESCE($6, '{}'::jsonb),
    evidence_count = knowledge_edges.evidence_count + COALESCE($7, 1),
    last_observed = NOW(),
    updated_at = NOW()
RETURNING id, user_id, source_node_id, target_node_id, edge_type, strength, properties, evidence_count, last_observed, created_at, updated_at, is_active;

-- name: GetKnowledgeEdge :one
SELECT id, user_id, source_node_id, target_node_id, edge_type, strength, properties, evidence_count, last_observed, created_at, updated_at, is_active
FROM knowledge_edges
WHERE id = $1;

-- name: GetKnowledgeEdges :many
SELECT ke.*, 
       sn.node_name as source_name,
       tn.node_name as target_name
FROM knowledge_edges ke
JOIN knowledge_nodes sn ON ke.source_node_id = sn.id
JOIN knowledge_nodes tn ON ke.target_node_id = tn.id
WHERE ke.user_id = $1::uuid
  AND (ke.edge_type = ANY($2::text[]) OR $2 IS NULL)
  AND ke.is_active = true
ORDER BY ke.strength DESC, ke.evidence_count DESC
LIMIT $3 OFFSET $4;

-- name: GetNodeConnections :many
SELECT ke.*,
       CASE 
         WHEN ke.source_node_id = $2::uuid THEN tn.node_name
         ELSE sn.node_name
       END as connected_node_name,
       CASE 
         WHEN ke.source_node_id = $2::uuid THEN tn.id
         ELSE sn.id
       END as connected_node_id,
       CASE 
         WHEN ke.source_node_id = $2::uuid THEN 'outgoing'
         ELSE 'incoming'
       END as direction
FROM knowledge_edges ke
JOIN knowledge_nodes sn ON ke.source_node_id = sn.id
JOIN knowledge_nodes tn ON ke.target_node_id = tn.id
WHERE ke.user_id = $1::uuid
  AND (ke.source_node_id = $2::uuid OR ke.target_node_id = $2::uuid)
  AND ke.is_active = true
ORDER BY ke.strength DESC;

-- name: GetEdgesByType :many
SELECT ke.*, 
       sn.node_name as source_name,
       tn.node_name as target_name
FROM knowledge_edges ke
JOIN knowledge_nodes sn ON ke.source_node_id = sn.id
JOIN knowledge_nodes tn ON ke.target_node_id = tn.id
WHERE ke.user_id = $1::uuid
  AND ke.edge_type = $2
  AND ke.is_active = true
ORDER BY ke.strength DESC
LIMIT $3;

-- name: UpdateKnowledgeEdgeStrength :one
UPDATE knowledge_edges
SET strength = $2,
    evidence_count = evidence_count + 1,
    last_observed = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, source_node_id, target_node_id, edge_type, strength, properties, evidence_count, last_observed, created_at, updated_at, is_active;

-- name: WeakenKnowledgeEdge :one
UPDATE knowledge_edges
SET strength = strength * $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, source_node_id, target_node_id, edge_type, strength, properties, evidence_count, last_observed, created_at, updated_at, is_active;

-- name: DeactivateKnowledgeEdge :exec
UPDATE knowledge_edges
SET is_active = false,
    updated_at = NOW()
WHERE id = $1;

-- name: GetStrongestConnections :many
SELECT ke.*, 
       sn.node_name as source_name,
       tn.node_name as target_name
FROM knowledge_edges ke
JOIN knowledge_nodes sn ON ke.source_node_id = sn.id
JOIN knowledge_nodes tn ON ke.target_node_id = tn.id
WHERE ke.user_id = $1::uuid
  AND ke.strength >= $2
  AND ke.is_active = true
ORDER BY ke.strength DESC
LIMIT $3;

-- name: FindDirectConnections :many
SELECT 
    ke.*,
    tn.node_name as target_name
FROM knowledge_edges ke
JOIN knowledge_nodes tn ON ke.target_node_id = tn.id
WHERE ke.source_node_id = $2::uuid
  AND ke.user_id = $1::uuid
  AND ke.is_active = true
ORDER BY ke.strength DESC;

-- =====================================================
-- KNOWLEDGE EVOLUTION QUERIES
-- =====================================================

-- name: CreateKnowledgeEvolution :one
INSERT INTO knowledge_evolution (
    user_id,
    entity_type,
    entity_id,
    change_type,
    previous_state,
    new_state,
    change_reason,
    confidence
) VALUES (
    $1::uuid, $2, $3::uuid, $4, $5, $6, $7, $8
) RETURNING id, user_id, entity_type, entity_id, change_type, previous_state, new_state, change_reason, confidence, created_at;

-- name: GetKnowledgeEvolution :many
SELECT id, user_id, entity_type, entity_id, change_type, previous_state, new_state, change_reason, confidence, created_at
FROM knowledge_evolution
WHERE user_id = $1::uuid
  AND (entity_type = $2 OR $2 IS NULL)
  AND (entity_id = $3::uuid OR $3 IS NULL)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: GetEntityEvolutionHistory :many
SELECT id, user_id, entity_type, entity_id, change_type, previous_state, new_state, change_reason, confidence, created_at
FROM knowledge_evolution
WHERE entity_id = $1::uuid
ORDER BY created_at ASC;

-- name: GetRecentChanges :many
SELECT ke.*, 
       CASE 
         WHEN ke.entity_type = 'node' THEN kn.node_name
         ELSE CONCAT('Edge: ', sn.node_name, ' -> ', tn.node_name)
       END as entity_name
FROM knowledge_evolution ke
LEFT JOIN knowledge_nodes kn ON ke.entity_type = 'node' AND ke.entity_id = kn.id
LEFT JOIN knowledge_edges kedge ON ke.entity_type = 'edge' AND ke.entity_id = kedge.id
LEFT JOIN knowledge_nodes sn ON kedge.source_node_id = sn.id
LEFT JOIN knowledge_nodes tn ON kedge.target_node_id = tn.id
WHERE ke.user_id = $1::uuid
  AND ke.created_at >= $2
ORDER BY ke.created_at DESC
LIMIT $3;

-- =====================================================
-- KNOWLEDGE GRAPH ANALYTICS QUERIES
-- =====================================================

-- name: GetGraphStatistics :one
SELECT 
    (SELECT COUNT(*) FROM knowledge_nodes WHERE user_id = $1::uuid AND is_active = true) as total_nodes,
    (SELECT COUNT(*) FROM knowledge_edges WHERE user_id = $1::uuid AND is_active = true) as total_edges,
    (SELECT COUNT(DISTINCT node_type) FROM knowledge_nodes WHERE user_id = $1::uuid AND is_active = true) as node_types,
    (SELECT COUNT(DISTINCT edge_type) FROM knowledge_edges WHERE user_id = $1::uuid AND is_active = true) as edge_types,
    (SELECT AVG(importance) FROM knowledge_nodes WHERE user_id = $1::uuid AND is_active = true) as avg_node_importance,
    (SELECT AVG(strength) FROM knowledge_edges WHERE user_id = $1::uuid AND is_active = true) as avg_edge_strength;

-- name: GetMostConnectedNodes :many
SELECT 
    kn.*,
    (SELECT COUNT(*) FROM knowledge_edges WHERE source_node_id = kn.id OR target_node_id = kn.id) as connection_count
FROM knowledge_nodes kn
WHERE kn.user_id = $1::uuid
  AND kn.is_active = true
ORDER BY connection_count DESC, kn.importance DESC
LIMIT $2;

-- name: GetHighlyConnectedNodes :many
SELECT 
    kn.*,
    COUNT(ke.id) as connection_count
FROM knowledge_nodes kn
LEFT JOIN knowledge_edges ke ON (ke.source_node_id = kn.id OR ke.target_node_id = kn.id)
WHERE kn.user_id = $1::uuid
  AND kn.is_active = true
  AND (ke.is_active = true OR ke.id IS NULL)
GROUP BY kn.id
HAVING COUNT(ke.id) >= $2
ORDER BY COUNT(ke.id) DESC;

-- name: GetOrphanNodes :many
SELECT kn.*
FROM knowledge_nodes kn
LEFT JOIN knowledge_edges ke ON (ke.source_node_id = kn.id OR ke.target_node_id = kn.id)
WHERE kn.user_id = $1::uuid
  AND kn.is_active = true
  AND ke.id IS NULL
ORDER BY kn.created_at DESC;

-- name: GetConnectedNodes :many
SELECT DISTINCT
    kn.id,
    kn.node_name,
    kn.node_type,
    kn.importance
FROM knowledge_nodes kn
JOIN knowledge_edges ke ON (ke.source_node_id = kn.id OR ke.target_node_id = kn.id)
WHERE kn.user_id = $1::uuid
  AND (ke.source_node_id = $2::uuid OR ke.target_node_id = $2::uuid)
  AND kn.id != $2::uuid
  AND kn.is_active = true
  AND ke.is_active = true
ORDER BY kn.importance DESC;