-- Complete Seed Data for Assistant System
-- 建立完整的測試資料集，包含管理員 Koopa 和相關資料

-- 清理現有資料
TRUNCATE TABLE users CASCADE;

-- 1. 建立管理員使用者 Koopa
-- 密碼: koopa123 (使用正確的 bcrypt hash)
INSERT INTO users (id, username, email, password_hash, full_name, avatar_url, preferences, is_active) VALUES 
('a0000000-0000-4000-8000-000000000001'::uuid, 'koopa', 'koopa@assistant.dev', 
 -- bcrypt hash for "koopa123"
 '$2a$10$rBmUzHlF7TDcwqwMO7hqKOaKxGfGGMxJvJzgY3xKFL4c.2Yiu6M7i', 
 'Koopa', 
 'https://avatars.githubusercontent.com/koopa', 
'{
  "language": "zh-TW",
  "theme": "dark",
  "defaultProgrammingLanguage": "Go",
  "emailNotifications": true,
  "timezone": "Asia/Taipei",
  "favoriteTools": ["go_analyzer", "go_formatter", "postgres_tool"],
  "role": "admin",
  "experienceLevel": "expert",
  "codeStyle": "clean_architecture"
}'::jsonb, true);

-- 2. 建立對話記錄
INSERT INTO conversations (id, user_id, title, summary, metadata, is_archived) VALUES 
-- Koopa 的對話
('c0000000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 '實現微服務架構', '討論如何使用 Go 實現微服務架構，包含 gRPC、服務發現和分散式追蹤', 
 '{"category": "Backend", "tags": ["microservices", "grpc", "go"], "difficulty": "advanced"}'::jsonb, false),

('c0000000-0000-4000-8000-000000000002'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'PostgreSQL 效能優化', '深入探討 PostgreSQL 17 的效能優化技巧，包含索引策略和查詢優化', 
 '{"category": "Database", "tags": ["postgresql", "performance", "optimization"], "difficulty": "expert"}'::jsonb, false),

('c0000000-0000-4000-8000-000000000003'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'Kubernetes 部署策略', '討論在 K8s 上部署 Go 應用的最佳實踐', 
 '{"category": "DevOps", "tags": ["kubernetes", "deployment", "containers"], "difficulty": "advanced"}'::jsonb, false),

('c0000000-0000-4000-8000-000000000004'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'AI 整合開發', '探討如何整合 Claude 和 Gemini API 到應用程式中', 
 '{"category": "AI", "tags": ["claude", "gemini", "langchain"], "difficulty": "intermediate"}'::jsonb, false),

('c0000000-0000-4000-8000-000000000005'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'Angular 前端開發', '討論 Angular 17 的新特性和最佳實踐', 
 '{"category": "Frontend", "tags": ["angular", "typescript", "frontend"], "difficulty": "intermediate"}'::jsonb, false),

('c0000000-0000-4000-8000-000000000006'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 '系統架構設計', '設計高可用、可擴展的系統架構', 
 '{"category": "Architecture", "tags": ["system-design", "scalability", "high-availability"], "difficulty": "expert"}'::jsonb, false);

-- 3. 建立訊息記錄
INSERT INTO messages (id, conversation_id, role, content, metadata, token_count) VALUES 
-- 微服務架構對話
('d1000000-0000-4000-8000-000000000001'::uuid, 'c0000000-0000-4000-8000-000000000001'::uuid, 'user', 
 '我想了解如何在 Go 中實現微服務架構，特別是服務間通訊的部分', 
 '{"intent": "learn", "topic": "microservices"}'::jsonb, 25),

('d1000000-0000-4000-8000-000000000002'::uuid, 'c0000000-0000-4000-8000-000000000001'::uuid, 'assistant', 
 '在 Go 中實現微服務架構，服務間通訊主要有以下幾種方式：

1. **gRPC** - 最推薦的方式
   - 高效能的二進位協議
   - 支援串流
   - 自動產生客戶端和服務端程式碼

2. **REST API** - 簡單直觀
   - 使用 HTTP/JSON
   - 易於除錯和測試
   - 適合公開 API

3. **訊息佇列** - 非同步通訊
   - 使用 RabbitMQ、Kafka 等
   - 解耦服務
   - 支援事件驅動架構

讓我為您展示一個 gRPC 的範例...', 
 '{"provider": "claude", "model": "claude-3-opus"}'::jsonb, 150),

-- PostgreSQL 優化對話
('d1000000-0000-4000-8000-000000000003'::uuid, 'c0000000-0000-4000-8000-000000000002'::uuid, 'user', 
 '我的 PostgreSQL 查詢很慢，如何優化？', 
 '{"intent": "troubleshoot", "topic": "database"}'::jsonb, 15),

('d1000000-0000-4000-8000-000000000004'::uuid, 'c0000000-0000-4000-8000-000000000002'::uuid, 'assistant', 
 '讓我幫您分析 PostgreSQL 查詢效能問題。首先，請使用 EXPLAIN ANALYZE 來查看執行計畫：

```sql
EXPLAIN (ANALYZE, BUFFERS, TIMING) 
SELECT * FROM your_table WHERE conditions;
```

常見的優化策略包括：
1. 建立適當的索引
2. 更新統計資訊
3. 調整查詢寫法
4. 使用部分索引
5. 考慮使用 BRIN 索引對時間序列資料', 
 '{"provider": "claude", "model": "claude-3-opus"}'::jsonb, 120);

-- 4. 建立工具執行記錄
INSERT INTO tool_executions (id, message_id, tool_name, input_data, output_data, status, execution_time_ms, started_at, completed_at) VALUES 
('e2000000-0000-4000-8000-000000000001'::uuid, 'd1000000-0000-4000-8000-000000000002'::uuid, 'go_analyzer', 
 '{"code": "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, Microservices!\")\n}", "analysis_type": "syntax"}'::jsonb,
 '{"issues": [], "suggestions": ["Consider adding error handling", "Add comments for documentation"], "complexity": 1}'::jsonb,
 'completed', 250, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days' + INTERVAL '250 milliseconds'),

('e2000000-0000-4000-8000-000000000002'::uuid, 'd1000000-0000-4000-8000-000000000004'::uuid, 'postgres_tool', 
 '{"query": "EXPLAIN ANALYZE SELECT * FROM users WHERE created_at > NOW() - INTERVAL ''7 days''", "database": "assistant"}'::jsonb,
 '{"plan": "Seq Scan on users", "execution_time": "0.5ms", "rows": 10, "suggestions": ["Consider adding index on created_at"]}'::jsonb,
 'completed', 500, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day' + INTERVAL '500 milliseconds');

-- 5. 建立 AI 使用記錄
INSERT INTO ai_provider_usage (user_id, provider, model, operation_type, input_tokens, output_tokens, cost_cents, request_id) VALUES 
('a0000000-0000-4000-8000-000000000001'::uuid, 'claude', 'claude-3-opus', 'chat_completion', 150, 500, 15, 'req_001'),
('a0000000-0000-4000-8000-000000000001'::uuid, 'claude', 'claude-3-opus', 'chat_completion', 200, 600, 18, 'req_002'),
('a0000000-0000-4000-8000-000000000001'::uuid, 'gemini', 'gemini-pro', 'chat_completion', 100, 400, 8, 'req_003'),
('a0000000-0000-4000-8000-000000000001'::uuid, 'claude', 'claude-3-haiku', 'chat_completion', 50, 200, 3, 'req_004');

-- 6. 建立嵌入向量資料（用於 RAG）
INSERT INTO embeddings (id, content_type, content_id, content_text, embedding, metadata) VALUES 
('f3000000-0000-4000-8000-000000000001'::uuid, 'message', 'd1000000-0000-4000-8000-000000000002'::uuid, 
 '在 Go 中實現微服務架構，服務間通訊主要有以下幾種方式...', 
 -- 生成正確維度的向量（使用隨機值填充 1536 維）
 ('['||array_to_string(ARRAY(SELECT random() FROM generate_series(1,1536)), ',')||']')::vector,
 '{"topic": "microservices", "language": "go"}'::jsonb),

('f3000000-0000-4000-8000-000000000002'::uuid, 'message', 'd1000000-0000-4000-8000-000000000004'::uuid, 
 '讓我幫您分析 PostgreSQL 查詢效能問題...', 
 ('['||array_to_string(ARRAY(SELECT random() FROM generate_series(1,1536)), ',')||']')::vector,
 '{"topic": "database", "language": "sql"}'::jsonb);

-- 7. 建立記憶體條目（使用正確的 memory_type）
INSERT INTO memory_entries (id, user_id, memory_type, content, importance, metadata) VALUES 
('a4000000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 'personalization', 
 'Koopa 偏好使用 Go 語言進行後端開發，特別擅長微服務架構', 0.9,
 '{"source": "conversation_analysis", "confidence": 0.95, "tags": ["go", "microservices", "backend"]}'::jsonb),

('a4000000-0000-4000-8000-000000000002'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 'long_term', 
 'Koopa 對 PostgreSQL 17 的新特性非常了解，包括 vacuum 優化和 WAL 性能提升', 0.85,
 '{"source": "direct_statement", "confidence": 0.9, "tags": ["postgresql", "database", "performance"]}'::jsonb),

('a4000000-0000-4000-8000-000000000003'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 'short_term', 
 'Koopa 正在開發一個智慧助手系統，使用 Go + PostgreSQL + Angular 技術棧', 0.95,
 '{"source": "project_context", "confidence": 1.0, "tags": ["project", "assistant", "fullstack"]}'::jsonb);

-- 8. 建立工具快取資料（使用正確的欄位）
INSERT INTO tool_cache (id, user_id, tool_name, input_hash, input_data, output_data, execution_time_ms, success, metadata, expires_at) VALUES 
('b5000000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'go_analyzer', 
 'hash_abc123',
 '{"code": "package main\n\nfunc main() {}"}'::jsonb,
 '{"result": "Code analysis completed", "issues": 0}'::jsonb,
 250,
 true,
 '{"hit_count": 5, "last_hit": "2025-06-04T10:00:00Z"}'::jsonb,
 NOW() + INTERVAL '7 days');

-- 9. 建立使用者偏好設定（使用正確的欄位結構）
INSERT INTO user_preferences (id, user_id, category, preference_key, preference_value, value_type, description, metadata) VALUES 
('c6000000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'coding', 'code_style', '"clean_architecture"'::jsonb, 'string',
 '偏好使用 Clean Architecture 設計模式',
 '{"confidence": 0.95}'::jsonb),

('c6000000-0000-4000-8000-000000000002'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'ai', 'preferred_provider', '"claude"'::jsonb, 'string',
 '偏好使用 Claude 作為 AI 提供者',
 '{"reason": "更精準的程式碼分析能力"}'::jsonb),

('c6000000-0000-4000-8000-000000000003'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'development', 'default_language', '"Go"'::jsonb, 'string',
 '預設的程式語言',
 '{"confidence": 0.95}'::jsonb);

-- 10. 建立使用者上下文（使用正確的欄位結構）
INSERT INTO user_context (id, user_id, context_type, context_key, context_value, importance, expires_at) VALUES 
('d7000000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'project', 'current_project',
 '{"name": "Assistant Go", "description": "智慧開發助手系統", "tech_stack": ["Go", "PostgreSQL", "Angular"], "status": "active"}'::jsonb,
 1.0, 
 NOW() + INTERVAL '30 days'),

('d7000000-0000-4000-8000-000000000002'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'workflow', 'working_hours',
 '{"timezone": "Asia/Taipei", "start": "09:00", "end": "18:00", "days": ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday"]}'::jsonb,
 0.7, 
 NOW() + INTERVAL '7 days');

-- 11. 建立資料庫連線設定（給 PostgreSQL 工具使用）
INSERT INTO database_connections (id, user_id, name, connection_string, description, is_default, metadata) VALUES 
('e8000000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'assistant_main', 
 'postgres://koopa:@localhost:5432/assistant?sslmode=disable',
 'Assistant 主資料庫', 
 true,
 '{"version": "17.5", "extensions": ["pgvector", "uuid-ossp"]}'::jsonb),

('e8000000-0000-4000-8000-000000000002'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'assistant_dev', 
 'postgres://koopa:@localhost:5432/assistant_dev?sslmode=disable',
 'Assistant 開發資料庫', 
 false,
 '{"version": "17.5", "purpose": "development"}'::jsonb);

-- 12. 建立 Kubernetes 叢集設定
INSERT INTO kubernetes_clusters (id, user_id, name, config_path, context, namespace, is_default, metadata) VALUES 
('f9000000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'local-k8s', 
 '~/.kube/config',
 'docker-desktop',
 'assistant-dev',
 true,
 '{"version": "1.28", "provider": "docker-desktop"}'::jsonb);

-- 13. 建立 Docker 連線設定
INSERT INTO docker_connections (id, user_id, name, host, api_version, tls_verify, is_default, metadata) VALUES 
('a9100000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'local-docker', 
 'unix:///var/run/docker.sock',
 '1.41',
 false,
 true,
 '{"version": "20.10", "platform": "linux/amd64"}'::jsonb);

-- 14. 建立搜尋快取
INSERT INTO search_cache (query_hash, query_text, results, source, expires_at) VALUES 
('hash_microservices_go', 
 'Go microservices best practices',
 '{
   "results": [
     {"title": "Go Microservices Guide", "url": "https://example.com/go-microservices"},
     {"title": "gRPC in Go", "url": "https://example.com/grpc-go"}
   ]
 }'::jsonb,
 'web_search',
 NOW() + INTERVAL '24 hours'),

('hash_postgresql_optimization', 
 'PostgreSQL 17 performance optimization',
 '{
   "results": [
     {"title": "PostgreSQL 17 新特性", "url": "https://www.postgresql.org/docs/17/"},
     {"title": "Vacuum 優化指南", "url": "https://example.com/pg-vacuum"}
   ]
 }'::jsonb,
 'web_search',
 NOW() + INTERVAL '24 hours');

-- 15. 建立 API 金鑰（用於測試）
INSERT INTO user_api_keys (id, user_id, name, key_hash, permissions, status, usage_count, expires_at) VALUES 
('b9200000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'Development Key', 
 '$2a$10$dev123456789012345678901234567890abcdefghijk',
 '["read", "write", "admin"]'::jsonb,
 'active',
 42,
 NOW() + INTERVAL '1 year');

-- 16. 建立代理執行記錄
INSERT INTO agent_executions (id, agent_type, user_id, conversation_id, query, response, steps, execution_time_ms, success, metadata) VALUES
('ae000000-0000-4000-8000-000000000001'::uuid, 'development', 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'c0000000-0000-4000-8000-000000000001'::uuid, 
 '幫我分析這段 Go 程式碼的效能問題',
 '我已經分析了您的程式碼，發現以下效能問題...',
 '[{"action": "analyze_code", "tool": "go_analyzer", "result": "found 3 issues"}]'::jsonb,
 1250, true,
 '{"tools_used": ["go_analyzer", "go_formatter"], "confidence": 0.92}'::jsonb);

-- 17. 建立鏈執行記錄
INSERT INTO chain_executions (id, chain_type, user_id, conversation_id, input, output, steps, execution_time_ms, tokens_used, success, metadata) VALUES
('ce000000-0000-4000-8000-000000000001'::uuid, 'rag', 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'c0000000-0000-4000-8000-000000000002'::uuid, 
 '如何優化 PostgreSQL 查詢？',
 '根據文檔和最佳實踐，PostgreSQL 查詢優化有以下幾個方面...',
 '[{"step": "retrieval", "documents": 5}, {"step": "generation", "model": "claude-3-opus"}]'::jsonb,
 2500, 850, true,
 '{"retrieval_method": "semantic_search", "documents_retrieved": 5}'::jsonb);

-- 18. 建立 Cloudflare 帳號設定（用於 Cloudflare 工具）
INSERT INTO cloudflare_accounts (id, user_id, name, api_token, account_id, is_default, metadata) VALUES
('cf000000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid,
 'koopa-cloudflare',
 'cf_token_example_123456',
 'cf_account_123456',
 true,
 '{"zones": ["example.com", "koopa.dev"], "plan": "pro"}'::jsonb);

-- 19. 建立代理會話
INSERT INTO agent_sessions (id, user_id, conversation_id, agent_type, session_data, memory_data, status, metadata) VALUES
('as000000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid,
 'c0000000-0000-4000-8000-000000000001'::uuid,
 'development',
 '{"current_task": "microservices_implementation", "progress": 0.75}'::jsonb,
 '{"short_term": ["discussing gRPC", "service discovery patterns"], "context": {"language": "go", "framework": "microservices"}}'::jsonb,
 'active',
 '{"tools_available": ["go_analyzer", "go_formatter", "go_builder"], "confidence_level": 0.9}'::jsonb);

-- 顯示建立結果摘要
SELECT 'Data seeding completed successfully!' as status;
SELECT 'Summary of created data:' as title;
SELECT 'Users:' as entity, COUNT(*) as count FROM users
UNION ALL
SELECT 'Conversations:', COUNT(*) FROM conversations
UNION ALL
SELECT 'Messages:', COUNT(*) FROM messages
UNION ALL
SELECT 'Tool executions:', COUNT(*) FROM tool_executions
UNION ALL
SELECT 'Memory entries:', COUNT(*) FROM memory_entries
UNION ALL
SELECT 'AI usage records:', COUNT(*) FROM ai_provider_usage
UNION ALL
SELECT 'Embeddings:', COUNT(*) FROM embeddings
UNION ALL
SELECT 'Tool cache entries:', COUNT(*) FROM tool_cache
UNION ALL
SELECT 'User preferences:', COUNT(*) FROM user_preferences
UNION ALL
SELECT 'User context:', COUNT(*) FROM user_context
UNION ALL
SELECT 'Database connections:', COUNT(*) FROM database_connections
UNION ALL
SELECT 'K8s clusters:', COUNT(*) FROM kubernetes_clusters
UNION ALL
SELECT 'Docker connections:', COUNT(*) FROM docker_connections
UNION ALL
SELECT 'Search cache:', COUNT(*) FROM search_cache
UNION ALL
SELECT 'API keys:', COUNT(*) FROM user_api_keys
UNION ALL
SELECT 'Agent executions:', COUNT(*) FROM agent_executions
UNION ALL
SELECT 'Chain executions:', COUNT(*) FROM chain_executions
UNION ALL
SELECT 'Cloudflare accounts:', COUNT(*) FROM cloudflare_accounts
UNION ALL
SELECT 'Agent sessions:', COUNT(*) FROM agent_sessions
ORDER BY entity;