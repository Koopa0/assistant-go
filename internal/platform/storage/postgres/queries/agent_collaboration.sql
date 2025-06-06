-- Agent collaboration queries

-- =====================================================
-- AGENT DEFINITIONS QUERIES
-- =====================================================

-- name: CreateAgentDefinition :one
INSERT INTO agent_definitions (
    agent_name,
    agent_type,
    capabilities,
    expertise_domains,
    collaboration_preferences,
    performance_metrics
) VALUES (
    $1, $2, $3, $4, $5, $6
) ON CONFLICT (agent_name)
DO UPDATE SET
    agent_type = $2,
    capabilities = $3,
    expertise_domains = $4,
    collaboration_preferences = $5,
    performance_metrics = $6,
    updated_at = NOW()
RETURNING *;

-- name: GetAgentDefinition :one
SELECT * FROM agent_definitions
WHERE id = $1;

-- name: GetAgentDefinitionByName :one
SELECT * FROM agent_definitions
WHERE agent_name = $1
  AND is_active = true;

-- name: GetAgentDefinitions :many
SELECT * FROM agent_definitions
WHERE (agent_type = ANY($1::text[]) OR $1 IS NULL)
  AND is_active = true
ORDER BY agent_name;

-- name: GetAgentsByCapability :many
SELECT * FROM agent_definitions
WHERE capabilities @> $1
  AND is_active = true
ORDER BY (capabilities->>'confidence')::float DESC;

-- name: GetAgentsByDomain :many
SELECT * FROM agent_definitions
WHERE $1 = ANY(expertise_domains)
  AND is_active = true
ORDER BY agent_name;

-- name: UpdateAgentPerformance :one
UPDATE agent_definitions
SET performance_metrics = performance_metrics || $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateAgentCapabilities :one
UPDATE agent_definitions
SET capabilities = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateAgent :exec
UPDATE agent_definitions
SET is_active = false,
    updated_at = NOW()
WHERE id = $1;

-- =====================================================
-- AGENT COLLABORATIONS QUERIES
-- =====================================================

-- name: CreateAgentCollaboration :one
INSERT INTO agent_collaborations (
    user_id,
    session_id,
    lead_agent_id,
    participating_agents,
    collaboration_type,
    task_description,
    task_complexity,
    collaboration_plan
) VALUES (
    $1::uuid, $2, $3::uuid, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetAgentCollaboration :one
SELECT ac.*,
       ad.agent_name as lead_agent_name
FROM agent_collaborations ac
JOIN agent_definitions ad ON ac.lead_agent_id = ad.id
WHERE ac.id = $1;

-- name: GetAgentCollaborations :many
SELECT ac.*,
       ad.agent_name as lead_agent_name
FROM agent_collaborations ac
JOIN agent_definitions ad ON ac.lead_agent_id = ad.id
WHERE ac.user_id = $1::uuid
  AND (ac.collaboration_type = $2 OR $2 IS NULL)
  AND (ac.session_id = $3 OR $3 IS NULL)
ORDER BY ac.created_at DESC
LIMIT $4 OFFSET $5;

-- name: GetActiveCollaborations :many
SELECT ac.*,
       ad.agent_name as lead_agent_name
FROM agent_collaborations ac
JOIN agent_definitions ad ON ac.lead_agent_id = ad.id
WHERE ac.user_id = $1::uuid
  AND ac.completed_at IS NULL
ORDER BY ac.created_at DESC;

-- name: UpdateCollaborationExecution :one
UPDATE agent_collaborations
SET execution_trace = $2,
    resource_usage = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CompleteCollaboration :one
UPDATE agent_collaborations
SET outcome = $2,
    total_duration_ms = EXTRACT(EPOCH FROM (NOW() - created_at)) * 1000,
    completed_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetCollaborationsByAgent :many
SELECT ac.*,
       ad.agent_name as lead_agent_name,
       CASE 
         WHEN ac.lead_agent_id = $2::uuid THEN 'lead'
         ELSE 'participant'
       END as role
FROM agent_collaborations ac
JOIN agent_definitions ad ON ac.lead_agent_id = ad.id
WHERE ac.user_id = $1::uuid
  AND (ac.lead_agent_id = $2::uuid OR $2::uuid = ANY(ac.participating_agents))
ORDER BY ac.created_at DESC
LIMIT $3;

-- name: GetCollaborationStatistics :one
SELECT 
    collaboration_type,
    COUNT(*) as total_collaborations,
    AVG(task_complexity) as avg_complexity,
    AVG(total_duration_ms) as avg_duration_ms,
    COUNT(*) FILTER (WHERE outcome = 'success') as successful_count,
    COUNT(*) FILTER (WHERE outcome = 'failure') as failed_count,
    COUNT(*) FILTER (WHERE completed_at IS NULL) as active_count
FROM agent_collaborations
WHERE user_id = $1::uuid
  AND created_at >= COALESCE($2, NOW() - INTERVAL '30 days')
GROUP BY collaboration_type;

-- name: GetMostEffectiveCollaborations :many
SELECT 
    ac.collaboration_type,
    ac.lead_agent_id,
    ad.agent_name as lead_agent_name,
    ac.participating_agents,
    AVG(ac.task_complexity) as avg_complexity,
    AVG(ac.total_duration_ms) as avg_duration_ms,
    COUNT(*) as collaboration_count,
    COUNT(*) FILTER (WHERE ac.outcome = 'success')::float / COUNT(*) as success_rate
FROM agent_collaborations ac
JOIN agent_definitions ad ON ac.lead_agent_id = ad.id
WHERE ac.user_id = $1::uuid
  AND ac.completed_at IS NOT NULL
GROUP BY ac.collaboration_type, ac.lead_agent_id, ad.agent_name, ac.participating_agents
HAVING COUNT(*) >= $2
ORDER BY success_rate DESC, avg_duration_ms ASC
LIMIT $3;

-- =====================================================
-- AGENT KNOWLEDGE SHARING QUERIES
-- =====================================================

-- name: CreateKnowledgeShare :one
INSERT INTO agent_knowledge_shares (
    source_agent_id,
    target_agent_id,
    knowledge_type,
    knowledge_content,
    relevance_score,
    collaboration_id
) VALUES (
    $1::uuid, $2::uuid, $3, $4, $5, $6::uuid
) RETURNING *;

-- name: GetKnowledgeShare :one
SELECT aks.*,
       sad.agent_name as source_agent_name,
       tad.agent_name as target_agent_name
FROM agent_knowledge_shares aks
JOIN agent_definitions sad ON aks.source_agent_id = sad.id
JOIN agent_definitions tad ON aks.target_agent_id = tad.id
WHERE aks.id = $1;

-- name: GetKnowledgeShares :many
SELECT aks.*,
       sad.agent_name as source_agent_name,
       tad.agent_name as target_agent_name
FROM agent_knowledge_shares aks
JOIN agent_definitions sad ON aks.source_agent_id = sad.id
JOIN agent_definitions tad ON aks.target_agent_id = tad.id
WHERE (aks.source_agent_id = $1::uuid OR $1 IS NULL)
  AND (aks.target_agent_id = $2::uuid OR $2 IS NULL)
  AND (aks.knowledge_type = $3 OR $3 IS NULL)
ORDER BY aks.created_at DESC
LIMIT $4 OFFSET $5;

-- name: GetKnowledgeSharesByCollaboration :many
SELECT aks.*,
       sad.agent_name as source_agent_name,
       tad.agent_name as target_agent_name
FROM agent_knowledge_shares aks
JOIN agent_definitions sad ON aks.source_agent_id = sad.id
JOIN agent_definitions tad ON aks.target_agent_id = tad.id
WHERE aks.collaboration_id = $1::uuid
ORDER BY aks.created_at;

-- name: UpdateKnowledgeShareAcceptance :one
UPDATE agent_knowledge_shares
SET was_accepted = $2,
    integration_result = $3
WHERE id = $1
RETURNING *;

-- name: GetMostSharedKnowledge :many
SELECT 
    knowledge_type,
    COUNT(*) as share_count,
    AVG(relevance_score) as avg_relevance,
    COUNT(*) FILTER (WHERE was_accepted = true)::float / COUNT(*) as acceptance_rate
FROM agent_knowledge_shares
WHERE source_agent_id = $1::uuid
  AND created_at >= COALESCE($2, NOW() - INTERVAL '30 days')
GROUP BY knowledge_type
ORDER BY share_count DESC, acceptance_rate DESC;

-- name: GetKnowledgeReceptionStats :many
SELECT 
    sad.agent_name as source_agent,
    knowledge_type,
    COUNT(*) as received_count,
    AVG(relevance_score) as avg_relevance,
    COUNT(*) FILTER (WHERE was_accepted = true) as accepted_count
FROM agent_knowledge_shares aks
JOIN agent_definitions sad ON aks.source_agent_id = sad.id
WHERE aks.target_agent_id = $1::uuid
  AND aks.created_at >= COALESCE($2, NOW() - INTERVAL '30 days')
GROUP BY sad.agent_name, knowledge_type
ORDER BY received_count DESC;

-- name: GetKnowledgeFlowNetwork :many
SELECT 
    sad.agent_name as source_agent,
    tad.agent_name as target_agent,
    COUNT(*) as knowledge_transfers,
    AVG(relevance_score) as avg_relevance,
    COUNT(*) FILTER (WHERE was_accepted = true)::float / COUNT(*) as acceptance_rate
FROM agent_knowledge_shares aks
JOIN agent_definitions sad ON aks.source_agent_id = sad.id
JOIN agent_definitions tad ON aks.target_agent_id = tad.id
WHERE aks.created_at >= COALESCE($1, NOW() - INTERVAL '30 days')
GROUP BY sad.agent_name, tad.agent_name
HAVING COUNT(*) >= $2
ORDER BY knowledge_transfers DESC;

-- =====================================================
-- AGENT COLLABORATION ANALYTICS QUERIES
-- =====================================================

-- name: GetCollaborationEfficiency :one
SELECT 
    AVG(CASE WHEN outcome = 'success' THEN total_duration_ms END) as avg_success_duration,
    AVG(CASE WHEN outcome = 'failure' THEN total_duration_ms END) as avg_failure_duration,
    COUNT(*) FILTER (WHERE outcome = 'success')::float / COUNT(*) as overall_success_rate,
    AVG(task_complexity) as avg_task_complexity,
    COUNT(DISTINCT lead_agent_id) as unique_lead_agents,
    AVG(array_length(participating_agents, 1)) as avg_participants
FROM agent_collaborations
WHERE user_id = $1::uuid
  AND completed_at IS NOT NULL
  AND created_at >= COALESCE($2, NOW() - INTERVAL '30 days');

-- name: GetAgentPerformanceRankings :many
SELECT 
    ad.agent_name,
    ad.agent_type,
    COUNT(ac.id) as total_collaborations,
    COUNT(ac.id) FILTER (WHERE ac.outcome = 'success') as successful_collaborations,
    COUNT(ac.id) FILTER (WHERE ac.outcome = 'success')::float / COUNT(ac.id) as success_rate,
    AVG(CASE WHEN ac.outcome = 'success' THEN ac.total_duration_ms END) as avg_success_duration,
    AVG(ac.task_complexity) as avg_task_complexity
FROM agent_definitions ad
LEFT JOIN agent_collaborations ac ON ad.id = ac.lead_agent_id
WHERE ad.is_active = true
  AND (ac.user_id = $1::uuid OR ac.user_id IS NULL)
  AND (ac.created_at >= COALESCE($2, NOW() - INTERVAL '30 days') OR ac.created_at IS NULL)
GROUP BY ad.id, ad.agent_name, ad.agent_type
HAVING COUNT(ac.id) > 0
ORDER BY success_rate DESC, avg_success_duration ASC;

-- name: GetOptimalTeamCompositions :many
SELECT 
    collaboration_type,
    participating_agents,
    COUNT(*) as usage_count,
    AVG(task_complexity) as avg_complexity,
    COUNT(*) FILTER (WHERE outcome = 'success')::float / COUNT(*) as success_rate,
    AVG(total_duration_ms) as avg_duration
FROM agent_collaborations
WHERE user_id = $1::uuid
  AND completed_at IS NOT NULL
  AND created_at >= COALESCE($2, NOW() - INTERVAL '30 days')
GROUP BY collaboration_type, participating_agents
HAVING COUNT(*) >= $3
ORDER BY success_rate DESC, avg_duration ASC;

-- name: GetCollaborationTrends :many
SELECT 
    DATE_TRUNC($2::text, created_at) as time_period,
    COUNT(*) as total_collaborations,
    COUNT(*) FILTER (WHERE outcome = 'success') as successful_collaborations,
    AVG(task_complexity) as avg_complexity,
    AVG(total_duration_ms) as avg_duration
FROM agent_collaborations
WHERE user_id = $1::uuid
  AND created_at >= COALESCE($3, NOW() - INTERVAL '90 days')
GROUP BY DATE_TRUNC($2::text, created_at)
ORDER BY time_period;

-- name: RecommendCollaborationPartners :many
SELECT 
    ad.id,
    ad.agent_name,
    ad.agent_type,
    ad.capabilities,
    ad.expertise_domains
FROM agent_definitions ad
WHERE ad.id != $1::uuid
  AND ad.is_active = true
ORDER BY ad.agent_name
LIMIT $2;