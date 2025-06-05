-- Final Complete Seed Data for Assistant System
-- 建立完整的測試資料集，包含管理員 Koopa 和相關資料

-- 清理現有資料 (CASCADE 會清理所有相關資料)
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
 '{"provider": "claude", "model": "claude-3-opus"}'::jsonb, 120),

-- K8s 部署對話
('d1000000-0000-4000-8000-000000000005'::uuid, 'c0000000-0000-4000-8000-000000000003'::uuid, 'user', 
 '如何在 Kubernetes 上部署我的 Go 微服務？', 
 '{"intent": "deploy", "topic": "kubernetes"}'::jsonb, 20),

('d1000000-0000-4000-8000-000000000006'::uuid, 'c0000000-0000-4000-8000-000000000003'::uuid, 'assistant', 
 '我來幫您設計 Kubernetes 部署策略。首先建立 Deployment 配置：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-microservice
  namespace: assistant-dev
spec:
  replicas: 3
  selector:
    matchLabels:
      app: go-microservice
  template:
    metadata:
      labels:
        app: go-microservice
    spec:
      containers:
      - name: api
        image: koopa/assistant:latest
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
```', 
 '{"provider": "claude", "model": "claude-3-opus"}'::jsonb, 200);

-- 4. 建立工具執行記錄
INSERT INTO tool_executions (id, message_id, tool_name, input_data, output_data, status, execution_time_ms, started_at, completed_at) VALUES 
('e2000000-0000-4000-8000-000000000001'::uuid, 'd1000000-0000-4000-8000-000000000002'::uuid, 'go_analyzer', 
 '{"code": "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, Microservices!\")\n}", "analysis_type": "syntax"}'::jsonb,
 '{"issues": [], "suggestions": ["Consider adding error handling", "Add comments for documentation"], "complexity": 1}'::jsonb,
 'completed', 250, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days' + INTERVAL '250 milliseconds'),

('e2000000-0000-4000-8000-000000000002'::uuid, 'd1000000-0000-4000-8000-000000000004'::uuid, 'postgres_tool', 
 '{"query": "EXPLAIN ANALYZE SELECT * FROM users WHERE created_at > NOW() - INTERVAL ''7 days''", "database": "assistant"}'::jsonb,
 '{"plan": "Seq Scan on users", "execution_time": "0.5ms", "rows": 10, "suggestions": ["Consider adding index on created_at"]}'::jsonb,
 'completed', 500, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day' + INTERVAL '500 milliseconds'),

('e2000000-0000-4000-8000-000000000003'::uuid, 'd1000000-0000-4000-8000-000000000006'::uuid, 'k8s_tool', 
 '{"action": "apply", "manifest": "deployment.yaml", "namespace": "assistant-dev"}'::jsonb,
 '{"status": "deployment created", "replicas": 3, "ready": 3}'::jsonb,
 'completed', 1500, NOW() - INTERVAL '12 hours', NOW() - INTERVAL '12 hours' + INTERVAL '1500 milliseconds');

-- 5. 建立 AI 使用記錄
INSERT INTO ai_provider_usage (user_id, provider, model, operation_type, input_tokens, output_tokens, cost_cents, request_id) VALUES 
('a0000000-0000-4000-8000-000000000001'::uuid, 'claude', 'claude-3-opus', 'chat_completion', 150, 500, 15, 'req_001'),
('a0000000-0000-4000-8000-000000000001'::uuid, 'claude', 'claude-3-opus', 'chat_completion', 200, 600, 18, 'req_002'),
('a0000000-0000-4000-8000-000000000001'::uuid, 'gemini', 'gemini-pro', 'chat_completion', 100, 400, 8, 'req_003'),
('a0000000-0000-4000-8000-000000000001'::uuid, 'claude', 'claude-3-haiku', 'chat_completion', 50, 200, 3, 'req_004'),
('a0000000-0000-4000-8000-000000000001'::uuid, 'claude', 'claude-3-opus', 'chat_completion', 180, 550, 16, 'req_005'),
('a0000000-0000-4000-8000-000000000001'::uuid, 'claude', 'claude-3-opus', 'embeddings', 120, 0, 2, 'req_006');

-- 6. 建立嵌入向量資料（用於 RAG）
INSERT INTO embeddings (id, content_type, content_id, content_text, embedding, metadata) VALUES 
('f3000001-0000-4000-8000-000000000001'::uuid, 'message', 'd1000000-0000-4000-8000-000000000002'::uuid, 
 '在 Go 中實現微服務架構，服務間通訊主要有以下幾種方式...', 
 ('['||array_to_string(ARRAY(SELECT random() FROM generate_series(1,1536)), ',')||']')::vector,
 '{"topic": "microservices", "language": "go"}'::jsonb),

('f3000001-0000-4000-8000-000000000002'::uuid, 'message', 'd1000000-0000-4000-8000-000000000004'::uuid, 
 '讓我幫您分析 PostgreSQL 查詢效能問題...', 
 ('['||array_to_string(ARRAY(SELECT random() FROM generate_series(1,1536)), ',')||']')::vector,
 '{"topic": "database", "language": "sql"}'::jsonb),

('f3000001-0000-4000-8000-000000000003'::uuid, 'message', 'd1000000-0000-4000-8000-000000000006'::uuid, 
 '我來幫您設計 Kubernetes 部署策略...', 
 ('['||array_to_string(ARRAY(SELECT random() FROM generate_series(1,1536)), ',')||']')::vector,
 '{"topic": "kubernetes", "language": "yaml"}'::jsonb);

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
 '{"source": "project_context", "confidence": 1.0, "tags": ["project", "assistant", "fullstack"]}'::jsonb),

('a4000000-0000-4000-8000-000000000004'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 'tool', 
 'Koopa 經常使用 go_analyzer 和 postgres_tool 來優化程式碼', 0.8,
 '{"frequency": "daily", "last_used": "2025-06-04", "effectiveness": 0.9}'::jsonb);

-- 8. 建立工具快取資料
INSERT INTO tool_cache (id, user_id, tool_name, input_hash, input_data, output_data, execution_time_ms, success, metadata, expires_at) VALUES 
('b5000000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'go_analyzer', 
 'hash_abc123',
 '{"code": "package main\n\nfunc main() {}"}'::jsonb,
 '{"result": "Code analysis completed", "issues": 0}'::jsonb,
 250, true,
 '{"cache_version": "1.0"}'::jsonb,
 NOW() + INTERVAL '7 days'),

('b5000000-0000-4000-8000-000000000002'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'postgres_tool', 
 'hash_xyz789',
 '{"query": "SELECT * FROM users LIMIT 10"}'::jsonb,
 '{"rows": 10, "execution_time": "2ms"}'::jsonb,
 50, true,
 '{"cache_version": "1.0"}'::jsonb,
 NOW() + INTERVAL '1 hour');

-- 9. 建立使用者偏好設定
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
 '{"confidence": 0.95}'::jsonb),

('c6000000-0000-4000-8000-000000000004'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'ui', 'theme', '"dark"'::jsonb, 'string',
 '介面主題偏好',
 '{"applies_to": ["web", "cli"]}'::jsonb);

-- 10. 建立使用者上下文
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
 NOW() + INTERVAL '7 days'),

('d7000000-0000-4000-8000-000000000003'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'environment', 'development_setup',
 '{"editor": "VSCode", "terminal": "iTerm2", "shell": "zsh", "go_version": "1.24.2", "node_version": "20.x"}'::jsonb,
 0.8, 
 NOW() + INTERVAL '90 days');

-- 11. 建立資料庫連線設定
INSERT INTO database_connections (id, user_id, name, connection_string, description, is_default, metadata) VALUES 
('e8000000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'assistant_main', 
 'postgres://koopa:@localhost:5432/assistant?sslmode=disable',
 'Assistant 主資料庫', 
 true,
 '{"version": "17.5", "extensions": ["pgvector", "uuid-ossp", "pg_stat_statements"]}'::jsonb),

('e8000000-0000-4000-8000-000000000002'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'assistant_dev', 
 'postgres://koopa:@localhost:5432/assistant_dev?sslmode=disable',
 'Assistant 開發資料庫', 
 false,
 '{"version": "17.5", "purpose": "development", "reset_daily": true}'::jsonb);

-- 12. 建立 Kubernetes 叢集設定
INSERT INTO kubernetes_clusters (id, user_id, name, config_path, context, namespace, is_default, metadata) VALUES 
('f9000000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'local-k8s', 
 '~/.kube/config',
 'docker-desktop',
 'assistant-dev',
 true,
 '{"version": "1.28", "provider": "docker-desktop", "resources": {"cpu": "4", "memory": "8Gi"}}'::jsonb);

-- 13. 建立 Docker 連線設定
INSERT INTO docker_connections (id, user_id, name, host, api_version, tls_verify, is_default, metadata) VALUES 
('a9100000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'local-docker', 
 'unix:///var/run/docker.sock',
 '1.41',
 false,
 true,
 '{"version": "20.10", "platform": "linux/amd64", "experimental": true}'::jsonb);

-- 14. 建立搜尋快取
INSERT INTO search_cache (query_hash, query_text, results, source, expires_at) VALUES 
('hash_microservices_go_v2', 
 'Go microservices best practices',
 '{
   "results": [
     {"title": "Go 微服務完整指南", "url": "https://example.com/go-microservices", "score": 0.95},
     {"title": "使用 gRPC 建構 Go 服務", "url": "https://example.com/grpc-go", "score": 0.92},
     {"title": "Go 微服務架構模式", "url": "https://example.com/patterns", "score": 0.88}
   ],
   "total": 3,
   "timestamp": "2025-06-04T10:00:00Z"
 }'::jsonb,
 'web_search',
 NOW() + INTERVAL '24 hours'),

('hash_postgresql_optimization_v2', 
 'PostgreSQL 17 performance optimization',
 '{
   "results": [
     {"title": "PostgreSQL 17 新特性完整解析", "url": "https://www.postgresql.org/docs/17/", "score": 0.98},
     {"title": "深入理解 PostgreSQL Vacuum 優化", "url": "https://example.com/pg-vacuum", "score": 0.94},
     {"title": "PostgreSQL 索引優化實戰", "url": "https://example.com/pg-index", "score": 0.91}
   ],
   "total": 3,
   "timestamp": "2025-06-04T10:00:00Z"
 }'::jsonb,
 'web_search',
 NOW() + INTERVAL '24 hours');

-- 15. 建立 API 金鑰
INSERT INTO user_api_keys (id, user_id, name, key_hash, permissions, status, usage_count, expires_at) VALUES 
('b9200000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'Development Key', 
 '$2a$10$dev123456789012345678901234567890abcdefghijk',
 '["read", "write", "admin"]'::jsonb,
 'active',
 42,
 NOW() + INTERVAL '1 year'),

('b9200000-0000-4000-8000-000000000002'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'Production Key', 
 '$2a$10$prod123456789012345678901234567890abcdefghijk',
 '["read", "write"]'::jsonb,
 'active',
 0,
 NOW() + INTERVAL '6 months');

-- 16. 建立代理執行記錄
INSERT INTO agent_executions (id, agent_type, user_id, conversation_id, query, response, steps, execution_time_ms, success, metadata) VALUES
('ae000000-0000-4000-8000-000000000001'::uuid, 'development', 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'c0000000-0000-4000-8000-000000000001'::uuid, 
 '幫我分析這段 Go 程式碼的效能問題',
 '我已經分析了您的程式碼，發現以下效能問題：\n1. 循環中重複分配記憶體\n2. 未使用並發處理\n3. 資料庫查詢未優化',
 '[{"action": "analyze_code", "tool": "go_analyzer", "result": "found 3 issues"}, {"action": "suggest_fix", "tool": "go_formatter", "result": "generated optimized code"}]'::jsonb,
 1250, true,
 '{"tools_used": ["go_analyzer", "go_formatter"], "confidence": 0.92, "improvements": {"memory": "30%", "speed": "45%"}}'::jsonb),

('ae000000-0000-4000-8000-000000000002'::uuid, 'database', 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'c0000000-0000-4000-8000-000000000002'::uuid, 
 '優化這個慢查詢',
 '根據 EXPLAIN ANALYZE 結果，建議：\n1. 在 created_at 欄位建立索引\n2. 使用部分索引過濾常用條件\n3. 調整 work_mem 參數',
 '[{"action": "analyze_query", "tool": "postgres_tool", "result": "identified bottlenecks"}, {"action": "generate_index", "result": "CREATE INDEX idx_created_at ON users(created_at)"}]'::jsonb,
 800, true,
 '{"query_time_before": "2500ms", "query_time_after": "15ms", "index_created": true}'::jsonb);

-- 17. 建立鏈執行記錄
INSERT INTO chain_executions (id, chain_type, user_id, conversation_id, input, output, steps, execution_time_ms, tokens_used, success, metadata) VALUES
('ce000000-0000-4000-8000-000000000001'::uuid, 'rag', 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'c0000000-0000-4000-8000-000000000002'::uuid, 
 '如何優化 PostgreSQL 查詢？',
 '根據文檔和最佳實踐，PostgreSQL 查詢優化包括：索引優化、查詢重寫、統計資訊更新、硬體配置調整等多個層面...',
 '[{"step": "retrieval", "documents": 5}, {"step": "reranking", "documents": 3}, {"step": "generation", "model": "claude-3-opus"}]'::jsonb,
 2500, 850, true,
 '{"retrieval_method": "semantic_search", "documents_retrieved": 5, "relevance_score": 0.89}'::jsonb),

('ce000000-0000-4000-8000-000000000002'::uuid, 'sequential', 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'c0000000-0000-4000-8000-000000000003'::uuid, 
 '部署應用到 Kubernetes',
 '已成功完成部署流程：建立映像、推送到 registry、更新 K8s 配置、執行滾動更新',
 '[{"step": "build", "status": "success"}, {"step": "push", "status": "success"}, {"step": "deploy", "status": "success"}]'::jsonb,
 5000, 300, true,
 '{"deployment_name": "assistant-api", "replicas": 3, "version": "v1.2.0"}'::jsonb);

-- 18. 建立 Cloudflare 帳號設定
INSERT INTO cloudflare_accounts (id, user_id, name, api_token, account_id, is_default, metadata) VALUES
('cf000000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid,
 'koopa-cloudflare',
 'cf_token_example_123456',
 'cf_account_123456',
 true,
 '{"zones": ["example.com", "koopa.dev"], "plan": "pro", "features": ["waf", "cdn", "workers"]}'::jsonb);

-- 19. 建立代理會話 (不包含 metadata 欄位)
INSERT INTO agent_sessions (id, user_id, conversation_id, agent_type, session_data, memory_data, status) VALUES
('as000000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid,
 'c0000000-0000-4000-8000-000000000001'::uuid,
 'development',
 '{"current_task": "microservices_implementation", "progress": 0.75, "context": {"language": "go", "framework": "grpc"}}'::jsonb,
 '{"short_term": ["discussing gRPC", "service discovery patterns"], "working_memory": {"current_file": "main.go", "recent_changes": 5}}'::jsonb,
 'active'),

('as000000-0000-4000-8000-000000000002'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid,
 'c0000000-0000-4000-8000-000000000002'::uuid,
 'database',
 '{"current_task": "query_optimization", "progress": 0.50, "focus": "indexing_strategy"}'::jsonb,
 '{"recent_queries": ["CREATE INDEX", "EXPLAIN ANALYZE"], "optimization_history": {"successful": 3, "failed": 0}}'::jsonb,
 'paused');

-- 顯示建立結果摘要
SELECT 'Data seeding completed successfully!' as status;

-- 分組顯示各類資料統計
SELECT '=== 使用者相關資料 ===' as category;
SELECT 'Users:' as entity, COUNT(*) as count FROM users
UNION ALL
SELECT 'User preferences:', COUNT(*) FROM user_preferences
UNION ALL
SELECT 'User context:', COUNT(*) FROM user_context
UNION ALL
SELECT 'API keys:', COUNT(*) FROM user_api_keys;

SELECT '=== 對話與訊息 ===' as category;
SELECT 'Conversations:' as entity, COUNT(*) as count FROM conversations
UNION ALL
SELECT 'Messages:', COUNT(*) FROM messages;

SELECT '=== AI 與工具使用 ===' as category;
SELECT 'AI usage records:' as entity, COUNT(*) as count FROM ai_provider_usage
UNION ALL
SELECT 'Tool executions:', COUNT(*) FROM tool_executions
UNION ALL
SELECT 'Tool cache entries:', COUNT(*) FROM tool_cache;

SELECT '=== 記憶與學習系統 ===' as category;
SELECT 'Memory entries:' as entity, COUNT(*) as count FROM memory_entries
UNION ALL
SELECT 'Embeddings:', COUNT(*) FROM embeddings
UNION ALL
SELECT 'Search cache:', COUNT(*) FROM search_cache;

SELECT '=== 代理與執行 ===' as category;
SELECT 'Agent executions:' as entity, COUNT(*) as count FROM agent_executions
UNION ALL
SELECT 'Chain executions:', COUNT(*) FROM chain_executions
UNION ALL
SELECT 'Agent sessions:', COUNT(*) FROM agent_sessions;

SELECT '=== 基礎設施連線 ===' as category;
SELECT 'Database connections:' as entity, COUNT(*) as count FROM database_connections
UNION ALL
SELECT 'K8s clusters:', COUNT(*) FROM kubernetes_clusters
UNION ALL
SELECT 'Docker connections:', COUNT(*) FROM docker_connections
UNION ALL
SELECT 'Cloudflare accounts:', COUNT(*) FROM cloudflare_accounts;

-- 顯示總計
SELECT '=== 總計 ===' as category;
SELECT 'Total records created:' as description, 
(SELECT COUNT(*) FROM users) +
(SELECT COUNT(*) FROM conversations) +
(SELECT COUNT(*) FROM messages) +
(SELECT COUNT(*) FROM tool_executions) +
(SELECT COUNT(*) FROM memory_entries) +
(SELECT COUNT(*) FROM ai_provider_usage) +
(SELECT COUNT(*) FROM embeddings) +
(SELECT COUNT(*) FROM tool_cache) +
(SELECT COUNT(*) FROM user_preferences) +
(SELECT COUNT(*) FROM user_context) +
(SELECT COUNT(*) FROM database_connections) +
(SELECT COUNT(*) FROM kubernetes_clusters) +
(SELECT COUNT(*) FROM docker_connections) +
(SELECT COUNT(*) FROM search_cache) +
(SELECT COUNT(*) FROM user_api_keys) +
(SELECT COUNT(*) FROM agent_executions) +
(SELECT COUNT(*) FROM chain_executions) +
(SELECT COUNT(*) FROM cloudflare_accounts) +
(SELECT COUNT(*) FROM agent_sessions) as total;