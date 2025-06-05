-- Seed Data for Assistant System
-- 建立完整的測試資料集，包含管理員 Koopa 和相關資料

-- 1. 建立管理員使用者 Koopa
-- 密碼: koopa123 (使用 bcrypt hash)
INSERT INTO users (id, username, email, password_hash, full_name, avatar_url, preferences, is_active) VALUES 
('a0000000-0000-0000-0000-000000000001'::uuid, 'koopa', 'koopa@assistant.dev', '$2a$10$YourHashHere123456789012345678901234567890', 'Koopa', 'https://avatars.githubusercontent.com/koopa', 
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
('c0000000-0000-0000-0000-000000000001'::uuid, 'a0000000-0000-0000-0000-000000000001'::uuid, 
 '實現微服務架構', '討論如何使用 Go 實現微服務架構，包含 gRPC、服務發現和分散式追蹤', 
 '{"category": "Backend", "tags": ["microservices", "grpc", "go"], "difficulty": "advanced"}'::jsonb, false),

('c0000000-0000-0000-0000-000000000002'::uuid, 'a0000000-0000-0000-0000-000000000001'::uuid, 
 'PostgreSQL 效能優化', '深入探討 PostgreSQL 17 的效能優化技巧，包含索引策略和查詢優化', 
 '{"category": "Database", "tags": ["postgresql", "performance", "optimization"], "difficulty": "expert"}'::jsonb, false),

('c0000000-0000-0000-0000-000000000003'::uuid, 'a0000000-0000-0000-0000-000000000001'::uuid, 
 'Kubernetes 部署策略', '討論在 K8s 上部署 Go 應用的最佳實踐', 
 '{"category": "DevOps", "tags": ["kubernetes", "deployment", "containers"], "difficulty": "advanced"}'::jsonb, false),

('c0000000-0000-0000-0000-000000000004'::uuid, 'a0000000-0000-0000-0000-000000000001'::uuid, 
 'AI 整合開發', '探討如何整合 Claude 和 Gemini API 到應用程式中', 
 '{"category": "AI", "tags": ["claude", "gemini", "langchain"], "difficulty": "intermediate"}'::jsonb, false);

-- 3. 建立訊息記錄
INSERT INTO messages (id, conversation_id, role, content, metadata, token_count) VALUES 
-- 微服務架構對話
('m0000000-0000-0000-0000-000000000001'::uuid, 'c0000000-0000-0000-0000-000000000001'::uuid, 'user', 
 '我想了解如何在 Go 中實現微服務架構，特別是服務間通訊的部分', 
 '{"intent": "learn", "topic": "microservices"}'::jsonb, 25),

('m0000000-0000-0000-0000-000000000002'::uuid, 'c0000000-0000-0000-0000-000000000001'::uuid, 'assistant', 
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
('m0000000-0000-0000-0000-000000000003'::uuid, 'c0000000-0000-0000-0000-000000000002'::uuid, 'user', 
 '我的 PostgreSQL 查詢很慢，如何優化？', 
 '{"intent": "troubleshoot", "topic": "database"}'::jsonb, 15),

('m0000000-0000-0000-0000-000000000004'::uuid, 'c0000000-0000-0000-0000-000000000002'::uuid, 'assistant', 
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
('te000000-0000-0000-0000-000000000001'::uuid, 'm0000000-0000-0000-0000-000000000002'::uuid, 'go_analyzer', 
 '{"code": "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, Microservices!\")\n}", "analysis_type": "syntax"}'::jsonb,
 '{"issues": [], "suggestions": ["Consider adding error handling", "Add comments for documentation"], "complexity": 1}'::jsonb,
 'completed', 250, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days' + INTERVAL '250 milliseconds'),

('te000000-0000-0000-0000-000000000002'::uuid, 'm0000000-0000-0000-0000-000000000004'::uuid, 'postgres_tool', 
 '{"query": "EXPLAIN ANALYZE SELECT * FROM users WHERE created_at > NOW() - INTERVAL ''7 days''", "database": "assistant"}'::jsonb,
 '{"plan": "Seq Scan on users", "execution_time": "0.5ms", "rows": 10, "suggestions": ["Consider adding index on created_at"]}'::jsonb,
 'completed', 500, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day' + INTERVAL '500 milliseconds');

-- 5. 建立 AI 使用記錄
INSERT INTO ai_provider_usage (user_id, provider, model, operation_type, input_tokens, output_tokens, cost_cents, request_id) VALUES 
('a0000000-0000-0000-0000-000000000001'::uuid, 'claude', 'claude-3-opus', 'chat_completion', 150, 500, 15, 'req_001'),
('a0000000-0000-0000-0000-000000000001'::uuid, 'claude', 'claude-3-opus', 'chat_completion', 200, 600, 18, 'req_002'),
('a0000000-0000-0000-0000-000000000001'::uuid, 'gemini', 'gemini-pro', 'chat_completion', 100, 400, 8, 'req_003');

-- 6. 建立嵌入向量資料（用於 RAG）
INSERT INTO embeddings (id, content_type, content_id, content_text, embedding, metadata) VALUES 
('e0000000-0000-0000-0000-000000000001'::uuid, 'message', 'm0000000-0000-0000-0000-000000000002'::uuid, 
 '在 Go 中實現微服務架構，服務間通訊主要有以下幾種方式...', 
 -- 簡化的向量資料（實際應該是 1536 維）
 '[0.1, 0.2, 0.3, 0.4, 0.5]'::vector,
 '{"topic": "microservices", "language": "go"}'::jsonb),

('e0000000-0000-0000-0000-000000000002'::uuid, 'message', 'm0000000-0000-0000-0000-000000000004'::uuid, 
 '讓我幫您分析 PostgreSQL 查詢效能問題...', 
 '[0.2, 0.3, 0.4, 0.5, 0.6]'::vector,
 '{"topic": "database", "language": "sql"}'::jsonb);

-- 7. 建立記憶體條目
INSERT INTO memory_entries (id, user_id, memory_type, content, importance, context, embedding, metadata) VALUES 
('me000000-0000-0000-0000-000000000001'::uuid, 'a0000000-0000-0000-0000-000000000001'::uuid, 'preference', 
 'Koopa 偏好使用 Go 語言進行後端開發，特別擅長微服務架構', 0.9,
 '{"source": "conversation_analysis", "confidence": 0.95}'::jsonb,
 '[0.3, 0.4, 0.5, 0.6, 0.7]'::vector,
 '{"tags": ["go", "microservices", "backend"]}'::jsonb),

('me000000-0000-0000-0000-000000000002'::uuid, 'a0000000-0000-0000-0000-000000000001'::uuid, 'knowledge', 
 'Koopa 對 PostgreSQL 17 的新特性非常了解，包括 vacuum 優化和 WAL 性能提升', 0.85,
 '{"source": "direct_statement", "confidence": 0.9}'::jsonb,
 '[0.4, 0.5, 0.6, 0.7, 0.8]'::vector,
 '{"tags": ["postgresql", "database", "performance"]}'::jsonb),

('me000000-0000-0000-0000-000000000003'::uuid, 'a0000000-0000-0000-0000-000000000001'::uuid, 'context', 
 'Koopa 正在開發一個智慧助手系統，使用 Go + PostgreSQL + Angular 技術棧', 0.95,
 '{"source": "project_context", "confidence": 1.0}'::jsonb,
 '[0.5, 0.6, 0.7, 0.8, 0.9]'::vector,
 '{"tags": ["project", "assistant", "fullstack"]}'::jsonb);

-- 8. 建立工具快取資料
INSERT INTO tool_cache (id, tool_name, cache_key, cached_result, metadata, expires_at) VALUES 
('tc000000-0000-0000-0000-000000000001'::uuid, 'go_analyzer', 
 'analyze:hash:abc123', 
 '{"result": "Code analysis completed", "issues": 0}'::jsonb,
 '{"hit_count": 5, "last_hit": "2025-06-04T10:00:00Z"}'::jsonb,
 NOW() + INTERVAL '7 days');

-- 9. 建立使用者偏好設定
INSERT INTO user_preferences (id, user_id, preference_key, preference_value, metadata) VALUES 
('up000000-0000-0000-0000-000000000001'::uuid, 'a0000000-0000-0000-0000-000000000001'::uuid, 
 'code_style', 'clean_architecture', 
 '{"description": "偏好使用 Clean Architecture 設計模式"}'::jsonb),

('up000000-0000-0000-0000-000000000002'::uuid, 'a0000000-0000-0000-0000-000000000001'::uuid, 
 'ai_provider', 'claude', 
 '{"reason": "更精準的程式碼分析能力"}'::jsonb);

-- 10. 建立使用者上下文
INSERT INTO user_context (id, user_id, context_type, context_data, importance, expires_at) VALUES 
('uc000000-0000-0000-0000-000000000001'::uuid, 'a0000000-0000-0000-0000-000000000001'::uuid, 
 'current_project', 
 '{
   "name": "Assistant Go",
   "description": "智慧開發助手系統",
   "tech_stack": ["Go", "PostgreSQL", "Angular"],
   "current_focus": "API 開發和資料庫優化"
 }'::jsonb, 
 1.0, 
 NOW() + INTERVAL '30 days');

-- 11. 建立資料庫連線設定（給 PostgreSQL 工具使用）
INSERT INTO database_connections (id, user_id, name, connection_string, description, is_default, metadata) VALUES 
('dc000000-0000-0000-0000-000000000001'::uuid, 'a0000000-0000-0000-0000-000000000001'::uuid, 
 'assistant_main', 
 'postgres://koopa:@localhost:5432/assistant?sslmode=disable',
 'Assistant 主資料庫', 
 true,
 '{"version": "17.5", "extensions": ["pgvector", "uuid-ossp"]}'::jsonb);

-- 12. 建立 Kubernetes 叢集設定
INSERT INTO kubernetes_clusters (id, user_id, name, config_path, context, namespace, is_default, metadata) VALUES 
('kc000000-0000-0000-0000-000000000001'::uuid, 'a0000000-0000-0000-0000-000000000001'::uuid, 
 'local-k8s', 
 '~/.kube/config',
 'docker-desktop',
 'assistant-dev',
 true,
 '{"version": "1.28", "provider": "docker-desktop"}'::jsonb);

-- 13. 建立 Docker 連線設定
INSERT INTO docker_connections (id, user_id, name, host, api_version, tls_verify, is_default, metadata) VALUES 
('do000000-0000-0000-0000-000000000001'::uuid, 'a0000000-0000-0000-0000-000000000001'::uuid, 
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
 NOW() + INTERVAL '24 hours');

-- 15. 建立 API 金鑰（用於測試）
INSERT INTO user_api_keys (id, user_id, name, key_hash, permissions, status, usage_count, expires_at) VALUES 
('ak000000-0000-0000-0000-000000000001'::uuid, 'a0000000-0000-0000-0000-000000000001'::uuid, 
 'Development Key', 
 '$2a$10$TestKeyHashHere123456789012345678901234567890',
 '["read", "write", "admin"]'::jsonb,
 'active',
 0,
 NOW() + INTERVAL '1 year');

-- 顯示建立結果摘要
SELECT 'Data seeding completed!' as status;
SELECT 'Users created:' as entity, COUNT(*) as count FROM users
UNION ALL
SELECT 'Conversations:', COUNT(*) FROM conversations
UNION ALL
SELECT 'Messages:', COUNT(*) FROM messages
UNION ALL
SELECT 'Tool executions:', COUNT(*) FROM tool_executions
UNION ALL
SELECT 'Memory entries:', COUNT(*) FROM memory_entries
UNION ALL
SELECT 'AI usage records:', COUNT(*) FROM ai_provider_usage;