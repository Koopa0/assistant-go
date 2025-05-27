-- Rollback LangChain extensions migration

-- Drop triggers
DROP TRIGGER IF EXISTS update_user_preferences_updated_at ON user_preferences;
DROP TRIGGER IF EXISTS update_user_context_updated_at ON user_context;

-- Drop indexes
DROP INDEX IF EXISTS idx_memory_entries_user_id;
DROP INDEX IF EXISTS idx_memory_entries_type;
DROP INDEX IF EXISTS idx_memory_entries_session;
DROP INDEX IF EXISTS idx_memory_entries_expires;
DROP INDEX IF EXISTS idx_memory_entries_importance;

DROP INDEX IF EXISTS idx_agent_executions_user_id;
DROP INDEX IF EXISTS idx_agent_executions_type;
DROP INDEX IF EXISTS idx_agent_executions_conversation;
DROP INDEX IF EXISTS idx_agent_executions_created_at;
DROP INDEX IF EXISTS idx_agent_executions_success;

DROP INDEX IF EXISTS idx_chain_executions_user_id;
DROP INDEX IF EXISTS idx_chain_executions_type;
DROP INDEX IF EXISTS idx_chain_executions_conversation;
DROP INDEX IF EXISTS idx_chain_executions_created_at;
DROP INDEX IF EXISTS idx_chain_executions_success;

DROP INDEX IF EXISTS idx_tool_cache_user_tool;
DROP INDEX IF EXISTS idx_tool_cache_hash;
DROP INDEX IF EXISTS idx_tool_cache_expires;
DROP INDEX IF EXISTS idx_tool_cache_hit_count;

DROP INDEX IF EXISTS idx_user_preferences_user_id;
DROP INDEX IF EXISTS idx_user_preferences_category;
DROP INDEX IF EXISTS idx_user_preferences_key;

DROP INDEX IF EXISTS idx_user_context_user_id;
DROP INDEX IF EXISTS idx_user_context_type;
DROP INDEX IF EXISTS idx_user_context_key;
DROP INDEX IF EXISTS idx_user_context_expires;
DROP INDEX IF EXISTS idx_user_context_importance;

-- Drop tables
DROP TABLE IF EXISTS user_context;
DROP TABLE IF EXISTS user_preferences;
DROP TABLE IF EXISTS tool_cache;
DROP TABLE IF EXISTS chain_executions;
DROP TABLE IF EXISTS agent_executions;
DROP TABLE IF EXISTS memory_entries;
