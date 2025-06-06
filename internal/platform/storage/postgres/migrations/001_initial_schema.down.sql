-- Drop triggers
DROP TRIGGER IF EXISTS update_agent_sessions_updated_at ON agent_sessions;
DROP TRIGGER IF EXISTS update_cloudflare_accounts_updated_at ON cloudflare_accounts;
DROP TRIGGER IF EXISTS update_docker_connections_updated_at ON docker_connections;
DROP TRIGGER IF EXISTS update_kubernetes_clusters_updated_at ON kubernetes_clusters;
DROP TRIGGER IF EXISTS update_database_connections_updated_at ON database_connections;
DROP TRIGGER IF EXISTS update_conversations_updated_at ON conversations;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_agent_sessions_conversation_id;
DROP INDEX IF EXISTS idx_agent_sessions_user_id;
DROP INDEX IF EXISTS idx_cloudflare_accounts_user_id;
DROP INDEX IF EXISTS idx_docker_connections_user_id;
DROP INDEX IF EXISTS idx_kubernetes_clusters_user_id;
DROP INDEX IF EXISTS idx_database_connections_user_id;
DROP INDEX IF EXISTS idx_ai_provider_usage_created_at;
DROP INDEX IF EXISTS idx_ai_provider_usage_user_id;
DROP INDEX IF EXISTS idx_search_cache_expires_at;
DROP INDEX IF EXISTS idx_search_cache_query_hash;
DROP INDEX IF EXISTS idx_embeddings_vector;
DROP INDEX IF EXISTS idx_embeddings_content_type_id;
DROP INDEX IF EXISTS idx_tool_executions_status;
DROP INDEX IF EXISTS idx_tool_executions_message_id;
DROP INDEX IF EXISTS idx_messages_created_at;
DROP INDEX IF EXISTS idx_messages_conversation_id;
DROP INDEX IF EXISTS idx_conversations_created_at;
DROP INDEX IF EXISTS idx_conversations_user_id;

-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS agent_sessions;
DROP TABLE IF EXISTS cloudflare_accounts;
DROP TABLE IF EXISTS docker_connections;
DROP TABLE IF EXISTS kubernetes_clusters;
DROP TABLE IF EXISTS database_connections;
DROP TABLE IF EXISTS ai_provider_usage;
DROP TABLE IF EXISTS search_cache;
DROP TABLE IF EXISTS embeddings;
DROP TABLE IF EXISTS tool_executions;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS users;

-- Drop extensions (only if no other objects depend on them)
-- Note: Be careful with dropping extensions in production
-- DROP EXTENSION IF EXISTS "vector";
-- DROP EXTENSION IF EXISTS "pgcrypto";
-- DROP EXTENSION IF EXISTS "uuid-ossp";
