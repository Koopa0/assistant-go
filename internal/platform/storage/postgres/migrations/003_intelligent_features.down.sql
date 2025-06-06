-- Rollback migration for intelligent features

-- Drop triggers first
DROP TRIGGER IF EXISTS update_code_patterns_updated_at ON code_patterns;
DROP TRIGGER IF EXISTS update_event_projections_updated_at ON event_projections;
DROP TRIGGER IF EXISTS update_agent_definitions_updated_at ON agent_definitions;
DROP TRIGGER IF EXISTS update_knowledge_edges_updated_at ON knowledge_edges;
DROP TRIGGER IF EXISTS update_knowledge_nodes_updated_at ON knowledge_nodes;
DROP TRIGGER IF EXISTS update_procedural_memories_updated_at ON procedural_memories;
DROP TRIGGER IF EXISTS update_semantic_memories_updated_at ON semantic_memories;
DROP TRIGGER IF EXISTS update_user_skills_updated_at ON user_skills;
DROP TRIGGER IF EXISTS update_learned_patterns_updated_at ON learned_patterns;

DROP TRIGGER IF EXISTS update_node_importance ON knowledge_nodes;
DROP TRIGGER IF EXISTS decay_vividness_on_access ON episodic_memories;

-- Drop trigger functions
DROP FUNCTION IF EXISTS update_knowledge_node_importance();
DROP FUNCTION IF EXISTS decay_episodic_memory_vividness();
DROP FUNCTION IF EXISTS update_learned_patterns_from_event();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS code_patterns;
DROP TABLE IF EXISTS development_sessions;
DROP TABLE IF EXISTS event_projections;
DROP TABLE IF EXISTS system_events;
DROP TABLE IF EXISTS agent_knowledge_shares;
DROP TABLE IF EXISTS agent_collaborations;
DROP TABLE IF EXISTS agent_definitions;
DROP TABLE IF EXISTS knowledge_evolution;
DROP TABLE IF EXISTS knowledge_edges;
DROP TABLE IF EXISTS knowledge_nodes;
DROP TABLE IF EXISTS working_memory;
DROP TABLE IF EXISTS procedural_memories;
DROP TABLE IF EXISTS semantic_memories;
DROP TABLE IF EXISTS episodic_memories;
DROP TABLE IF EXISTS user_skills;
DROP TABLE IF EXISTS learned_patterns;
DROP TABLE IF EXISTS learning_events;

-- Drop extensions if no other tables use them
-- Note: Only drop if you're sure no other schemas use these extensions
-- DROP EXTENSION IF EXISTS "pg_trgm";
-- DROP EXTENSION IF EXISTS "ltree";