-- 額外的測試資料 - 豐富 Assistant 系統的資料集
-- 增加更多使用者偏好設定、上下文、搜尋快取、工具快取、代理執行和鏈執行記錄

-- 1. 增加更多使用者偏好設定 (User Preferences)
INSERT INTO user_preferences (id, user_id, category, preference_key, preference_value, value_type, description, metadata) VALUES 
-- 編輯器設定
('c6100000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'editor', 'font_size', '14'::jsonb, 'number',
 '編輯器字體大小',
 '{"unit": "px", "min": 10, "max": 24}'::jsonb),

('c6100000-0000-4000-8000-000000000002'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'editor', 'tab_size', '4'::jsonb, 'number',
 'Tab 縮排空格數',
 '{"language_overrides": {"python": 4, "javascript": 2}}'::jsonb),

('c6100000-0000-4000-8000-000000000003'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'editor', 'line_numbers', 'true'::jsonb, 'boolean',
 '顯示行號',
 '{"applies_to": ["code_display", "code_editor"]}'::jsonb),

-- AI 互動偏好
('c6100000-0000-4000-8000-000000000004'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'ai', 'response_style', '"detailed"'::jsonb, 'string',
 'AI 回應風格',
 '{"options": ["concise", "detailed", "educational"], "confidence": 0.85}'::jsonb),

('c6100000-0000-4000-8000-000000000005'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'ai', 'code_explanation_level', '"intermediate"'::jsonb, 'string',
 '程式碼解釋詳細度',
 '{"options": ["beginner", "intermediate", "expert"], "auto_adjust": true}'::jsonb),

('c6100000-0000-4000-8000-000000000006'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'ai', 'max_tokens', '2048'::jsonb, 'number',
 'AI 回應最大 token 數',
 '{"provider_specific": {"claude": 4096, "gemini": 2048}}'::jsonb),

-- 開發工具偏好
('c6100000-0000-4000-8000-000000000007'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'tools', 'auto_format', 'true'::jsonb, 'boolean',
 '自動格式化程式碼',
 '{"triggers": ["save", "paste"], "languages": ["go", "javascript", "python"]}'::jsonb),

('c6100000-0000-4000-8000-000000000008'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'tools', 'linter_severity', '"warning"'::jsonb, 'string',
 'Linter 嚴格程度',
 '{"options": ["error", "warning", "info"], "ignore_rules": ["line-length"]}'::jsonb),

-- 通知偏好
('c6100000-0000-4000-8000-000000000009'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'notifications', 'email_frequency', '"daily"'::jsonb, 'string',
 '電子郵件通知頻率',
 '{"options": ["realtime", "hourly", "daily", "weekly", "never"]}'::jsonb),

('c6100000-0000-4000-8000-000000000010'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'notifications', 'desktop_notifications', 'true'::jsonb, 'boolean',
 '桌面通知',
 '{"types": ["errors", "completions", "mentions"]}'::jsonb);

-- 2. 增加更多使用者上下文 (User Context)
INSERT INTO user_context (id, user_id, context_type, context_key, context_value, importance, expires_at) VALUES 
-- 專案相關上下文
('d7100000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'project', 'recent_files',
 '[
   "/internal/assistant/assistant.go",
   "/internal/langchain/service.go",
   "/internal/storage/postgres/client.go",
   "/cmd/assistant/main.go",
   "/internal/api/handlers/users.go"
 ]'::jsonb,
 0.9, 
 NOW() + INTERVAL '7 days'),

('d7100000-0000-4000-8000-000000000002'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'project', 'active_branches',
 '{"main": "latest", "feature/langchain-integration": "in-progress", "fix/database-connection": "merged"}'::jsonb,
 0.8, 
 NOW() + INTERVAL '14 days'),

-- 開發習慣上下文
('d7100000-0000-4000-8000-000000000003'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'habits', 'coding_patterns',
 '{
   "error_handling": "always_wrap_errors",
   "testing": "table_driven_tests",
   "naming": "clear_descriptive_names",
   "comments": "explain_why_not_what"
 }'::jsonb,
 0.85, 
 NOW() + INTERVAL '60 days'),

('d7100000-0000-4000-8000-000000000004'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'habits', 'common_commands',
 '[
   "go test ./...",
   "make quick-check",
   "go mod tidy",
   "git status",
   "docker-compose up -d"
 ]'::jsonb,
 0.75, 
 NOW() + INTERVAL '30 days'),

-- 團隊協作上下文
('d7100000-0000-4000-8000-000000000005'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'team', 'collaborators',
 '[
   {"name": "Alice", "role": "Frontend Developer", "timezone": "UTC+8"},
   {"name": "Bob", "role": "DevOps Engineer", "timezone": "UTC+8"},
   {"name": "Carol", "role": "Data Scientist", "timezone": "UTC+7"}
 ]'::jsonb,
 0.6, 
 NOW() + INTERVAL '90 days'),

-- 系統狀態上下文
('d7100000-0000-4000-8000-000000000006'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'system', 'resource_usage',
 '{
   "cpu_cores": 8,
   "memory_gb": 16,
   "disk_usage_percent": 65,
   "docker_containers": 5,
   "postgres_connections": 12
 }'::jsonb,
 0.7, 
 NOW() + INTERVAL '1 day');

-- 3. 增加更多搜尋快取 (Search Cache)
INSERT INTO search_cache (query_hash, query_text, results, source, expires_at) VALUES 
('hash_go_concurrency_patterns', 
 'Go concurrency patterns best practices',
 '{
   "results": [
     {"title": "Go 並發模式完整指南", "url": "https://blog.golang.org/pipelines", "score": 0.96, "snippet": "使用 channels 和 goroutines 建構並發管道"},
     {"title": "Context 在並發中的應用", "url": "https://blog.golang.org/context", "score": 0.94, "snippet": "使用 context 管理 goroutine 生命週期"},
     {"title": "Worker Pool 模式實現", "url": "https://gobyexample.com/worker-pools", "score": 0.91, "snippet": "使用 worker pool 處理大量任務"},
     {"title": "Fan-out/Fan-in 模式", "url": "https://go.dev/blog/io2013-talk-concurrency", "score": 0.89, "snippet": "分散處理後聚合結果的並發模式"}
   ],
   "total": 4,
   "query_time_ms": 125,
   "timestamp": "2025-06-04T11:00:00Z"
 }'::jsonb,
 'web_search',
 NOW() + INTERVAL '48 hours'),

('hash_postgresql_indexing_strategies', 
 'PostgreSQL advanced indexing strategies',
 '{
   "results": [
     {"title": "PostgreSQL 索引類型詳解", "url": "https://www.postgresql.org/docs/current/indexes-types.html", "score": 0.98, "snippet": "B-tree、Hash、GiST、SP-GiST、GIN 和 BRIN 索引比較"},
     {"title": "部分索引優化技巧", "url": "https://www.postgresql.org/docs/current/indexes-partial.html", "score": 0.95, "snippet": "使用 WHERE 子句建立部分索引"},
     {"title": "多列索引最佳實踐", "url": "https://use-the-index-luke.com/sql/where-clause/the-equals-operator/composite-index", "score": 0.92, "snippet": "複合索引的列順序優化"}
   ],
   "total": 3,
   "query_time_ms": 98,
   "timestamp": "2025-06-04T11:30:00Z"
 }'::jsonb,
 'web_search',
 NOW() + INTERVAL '36 hours'),

('hash_kubernetes_deployment_strategies', 
 'Kubernetes deployment strategies comparison',
 '{
   "results": [
     {"title": "K8s 部署策略比較", "url": "https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#strategy", "score": 0.97, "snippet": "Rolling Update vs Recreate 策略"},
     {"title": "藍綠部署在 K8s 中的實現", "url": "https://www.redhat.com/en/topics/devops/what-is-blue-green-deployment", "score": 0.93, "snippet": "使用 Service 切換實現零停機部署"},
     {"title": "金絲雀發布最佳實踐", "url": "https://martinfowler.com/bliki/CanaryRelease.html", "score": 0.90, "snippet": "漸進式部署降低風險"}
   ],
   "total": 3,
   "query_time_ms": 145,
   "timestamp": "2025-06-04T12:00:00Z"
 }'::jsonb,
 'web_search',
 NOW() + INTERVAL '72 hours'),

('hash_angular_performance_optimization', 
 'Angular 17 performance optimization techniques',
 '{
   "results": [
     {"title": "Angular 17 性能優化指南", "url": "https://angular.io/guide/performance-optimizations", "score": 0.95, "snippet": "OnPush 策略和 trackBy 函數優化"},
     {"title": "Lazy Loading 最佳實踐", "url": "https://angular.io/guide/lazy-loading-ngmodules", "score": 0.92, "snippet": "使用動態導入減少初始載入時間"},
     {"title": "信號 (Signals) 提升響應性", "url": "https://angular.io/guide/signals", "score": 0.89, "snippet": "Angular 17 新特性改善變更檢測"}
   ],
   "total": 3,
   "query_time_ms": 112,
   "timestamp": "2025-06-04T12:30:00Z"
 }'::jsonb,
 'web_search',
 NOW() + INTERVAL '24 hours');

-- 4. 增加更多工具快取 (Tool Cache)
INSERT INTO tool_cache (id, user_id, tool_name, input_hash, input_data, output_data, execution_time_ms, success, metadata, expires_at) VALUES 
-- Go 分析工具快取
('b5100000-0000-4000-8000-000000000001'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'go_analyzer', 
 'hash_complex_function_analysis',
 '{
   "code": "func ProcessOrders(ctx context.Context, orders []Order) error {\n    var wg sync.WaitGroup\n    errCh := make(chan error, len(orders))\n    \n    for _, order := range orders {\n        wg.Add(1)\n        go func(o Order) {\n            defer wg.Done()\n            if err := processOrder(ctx, o); err != nil {\n                errCh <- err\n            }\n        }(order)\n    }\n    \n    wg.Wait()\n    close(errCh)\n    \n    if err := <-errCh; err != nil {\n        return fmt.Errorf(\"order processing failed: %w\", err)\n    }\n    return nil\n}",
   "analysis_type": "concurrency"
 }'::jsonb,
 '{
   "issues": [
     {"severity": "warning", "line": 16, "message": "Potential goroutine leak if context is cancelled"},
     {"severity": "info", "line": 11, "message": "Consider using errgroup for better error handling"}
   ],
   "complexity": 8,
   "suggestions": [
     "Use golang.org/x/sync/errgroup for cleaner error handling",
     "Add context cancellation check in goroutine",
     "Consider worker pool pattern for large order counts"
   ]
 }'::jsonb,
 320, true,
 '{"cache_hits": 12, "last_modified": "2025-06-04T10:30:00Z"}'::jsonb,
 NOW() + INTERVAL '3 days'),

-- Go 格式化工具快取
('b5100000-0000-4000-8000-000000000002'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'go_formatter', 
 'hash_format_imports',
 '{
   "code": "package main\nimport \"fmt\"\nimport \"context\"\nimport \"sync\"\nimport \"time\"",
   "options": {"group_imports": true}
 }'::jsonb,
 '{
   "formatted_code": "package main\n\nimport (\n    \"context\"\n    \"fmt\"\n    \"sync\"\n    \"time\"\n)",
   "changes": ["grouped imports", "sorted alphabetically", "added blank line after package"]
 }'::jsonb,
 45, true,
 '{"formatter_version": "1.21", "style": "gofmt"}'::jsonb,
 NOW() + INTERVAL '7 days'),

-- PostgreSQL 查詢優化工具快取
('b5100000-0000-4000-8000-000000000003'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'postgres_tool', 
 'hash_slow_query_analysis',
 '{
   "query": "SELECT u.*, COUNT(m.id) as message_count FROM users u LEFT JOIN messages m ON u.id = m.user_id WHERE u.created_at > NOW() - INTERVAL ''30 days'' GROUP BY u.id",
   "database": "assistant"
 }'::jsonb,
 '{
   "execution_plan": "HashAggregate -> Hash Left Join -> Seq Scan on users",
   "execution_time": "125ms",
   "rows": 50,
   "suggestions": [
     "CREATE INDEX idx_users_created_at ON users(created_at)",
     "Consider materialized view for frequently accessed aggregations",
     "VACUUM ANALYZE users table for updated statistics"
   ],
   "index_recommendations": [
     {"table": "users", "columns": ["created_at"], "type": "btree", "estimated_improvement": "80%"}
   ]
 }'::jsonb,
 450, true,
 '{"database_version": "17.5", "table_sizes": {"users": "1MB", "messages": "25MB"}}'::jsonb,
 NOW() + INTERVAL '2 hours'),

-- Kubernetes 部署工具快取
('b5100000-0000-4000-8000-000000000004'::uuid, 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'k8s_tool', 
 'hash_deployment_validation',
 '{
   "action": "validate",
   "manifest": "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: assistant-api\nspec:\n  replicas: 3\n  selector:\n    matchLabels:\n      app: assistant-api\n  template:\n    metadata:\n      labels:\n        app: assistant-api\n    spec:\n      containers:\n      - name: api\n        image: koopa/assistant:v1.2.0\n        ports:\n        - containerPort: 8080"
 }'::jsonb,
 '{
   "valid": true,
   "warnings": [
     "No resource limits specified",
     "No health checks defined",
     "Consider adding pod disruption budget"
   ],
   "recommendations": [
     "Add resource requests and limits",
     "Configure liveness and readiness probes",
     "Set up horizontal pod autoscaler"
   ]
 }'::jsonb,
 200, true,
 '{"k8s_version": "1.28", "api_versions": ["apps/v1", "v1"]}'::jsonb,
 NOW() + INTERVAL '12 hours');

-- 5. 增加更多代理執行記錄 (Agent Executions)
INSERT INTO agent_executions (id, agent_type, user_id, conversation_id, query, response, steps, execution_time_ms, success, metadata) VALUES
-- 研究代理執行
('ae100000-0000-4000-8000-000000000001'::uuid, 'research', 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'c0000000-0000-4000-8000-000000000004'::uuid, 
 '比較 Claude 和 Gemini 在程式碼生成的優劣',
 '根據我的分析：\n\nClaude 優勢：\n- 更準確的 Go idiom 理解\n- 更好的錯誤處理建議\n- 更詳細的程式碼註釋\n\nGemini 優勢：\n- 更快的回應速度\n- 更好的多語言支援\n- 更強的數學運算能力\n\n建議：程式碼審查使用 Claude，快速原型使用 Gemini',
 '[
   {"action": "search", "query": "Claude vs Gemini code generation", "results": 5},
   {"action": "analyze", "tool": "comparison_analyzer", "metrics": ["accuracy", "speed", "cost"]},
   {"action": "synthesize", "sources": 3, "confidence": 0.88}
 ]'::jsonb,
 3500, true,
 '{"sources_consulted": 5, "comparison_criteria": ["accuracy", "speed", "idiom_understanding", "cost"]}'::jsonb),

-- 基礎設施代理執行
('ae100000-0000-4000-8000-000000000002'::uuid, 'infrastructure', 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'c0000000-0000-4000-8000-000000000003'::uuid, 
 '檢查 Kubernetes 叢集健康狀態',
 '叢集健康檢查完成：\n\n✅ 所有節點運行正常\n✅ 系統 Pod 全部健康\n⚠️  assistant-api 有一個 Pod 重啟了 3 次\n\n建議：\n1. 檢查 assistant-api 的日誌\n2. 可能需要增加記憶體限制\n3. 考慮配置 pod disruption budget',
 '[
   {"action": "check_nodes", "tool": "k8s_tool", "result": "all healthy"},
   {"action": "check_system_pods", "namespace": "kube-system", "result": "all running"},
   {"action": "check_app_pods", "namespace": "assistant-dev", "issues": ["pod restart detected"]},
   {"action": "analyze_logs", "pod": "assistant-api-xxx", "finding": "OOM killed"}
 ]'::jsonb,
 2200, true,
 '{"cluster": "local-k8s", "nodes": 3, "total_pods": 45, "issues_found": 1}'::jsonb),

-- 資料庫專家代理執行
('ae100000-0000-4000-8000-000000000003'::uuid, 'database', 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'c0000000-0000-4000-8000-000000000002'::uuid, 
 '分析並優化資料庫整體性能',
 '資料庫性能分析報告：\n\n1. **查詢性能**\n   - 發現 3 個慢查詢\n   - 建議新增 5 個索引\n\n2. **資料表優化**\n   - messages 表需要分區\n   - 建議 VACUUM ANALYZE 3 個表\n\n3. **連接池配置**\n   - 當前：30 連接\n   - 建議：50 連接（基於負載）\n\n4. **pgvector 優化**\n   - 建議調整 lists 參數為 200',
 '[
   {"action": "analyze_slow_queries", "tool": "postgres_tool", "found": 3},
   {"action": "analyze_indexes", "missing_indexes": 5, "unused_indexes": 2},
   {"action": "check_table_sizes", "large_tables": ["messages", "embeddings"]},
   {"action": "analyze_connections", "current": 30, "peak": 28, "recommended": 50},
   {"action": "check_pgvector", "current_lists": 100, "recommended_lists": 200}
 ]'::jsonb,
 4500, true,
 '{"total_tables": 25, "total_size": "500MB", "index_bloat": "12%", "cache_hit_ratio": 0.95}'::jsonb);

-- 6. 增加更多鏈執行記錄 (Chain Executions)
INSERT INTO chain_executions (id, chain_type, user_id, conversation_id, input, output, steps, execution_time_ms, tokens_used, success, metadata) VALUES
-- 複雜 RAG 鏈執行
('ce100000-0000-4000-8000-000000000001'::uuid, 'rag_enhanced', 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'c0000000-0000-4000-8000-000000000004'::uuid, 
 '如何在 Go 中實現高效的記憶體池？',
 '基於檢索到的文檔和最佳實踐，實現高效記憶體池的方法：\n\n1. **使用 sync.Pool**\n```go\nvar bufferPool = sync.Pool{\n    New: func() interface{} {\n        return make([]byte, 1024)\n    },\n}\n```\n\n2. **自定義記憶體池**\n- 預分配固定大小的物件\n- 使用 channel 管理可用物件\n- 實現借用/歸還機制\n\n3. **注意事項**\n- sync.Pool 會被 GC 清理\n- 適合臨時物件重用\n- 注意並發安全性',
 '[
   {"step": "query_embedding", "model": "text-embedding-ada-002", "tokens": 15},
   {"step": "vector_search", "matches": 8, "relevance_threshold": 0.85},
   {"step": "rerank", "method": "cross-encoder", "top_k": 4},
   {"step": "context_building", "tokens": 450},
   {"step": "generation", "model": "claude-3-opus", "tokens": 320}
 ]'::jsonb,
 3200, 785, true,
 '{"retrieval_quality": 0.92, "sources": ["go-blog", "effective-go", "go-patterns"], "cache_hit": false}'::jsonb),

-- 多步驟順序鏈執行
('ce100000-0000-4000-8000-000000000002'::uuid, 'sequential_complex', 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'c0000000-0000-4000-8000-000000000001'::uuid, 
 '分析程式碼 -> 生成測試 -> 執行測試 -> 優化建議',
 '完成多步驟分析流程：\n\n1. ✅ 程式碼分析完成\n   - 發現 2 個潛在問題\n   - 複雜度：中等\n\n2. ✅ 測試生成完成\n   - 生成 5 個測試案例\n   - 覆蓋率預估：85%\n\n3. ✅ 測試執行完成\n   - 5/5 測試通過\n   - 執行時間：125ms\n\n4. ✅ 優化建議\n   - 使用 strings.Builder 替代字串拼接\n   - 考慮並行處理大量資料',
 '[
   {"step": "code_analysis", "tool": "go_analyzer", "issues": 2, "duration": 300},
   {"step": "test_generation", "tool": "test_generator", "tests": 5, "duration": 800},
   {"step": "test_execution", "tool": "go_test", "passed": 5, "failed": 0, "duration": 125},
   {"step": "optimization", "tool": "optimizer", "suggestions": 2, "duration": 200}
 ]'::jsonb,
 1425, 650, true,
 '{"total_steps": 4, "parallel_execution": false, "confidence": 0.91}'::jsonb),

-- 並行鏈執行
('ce100000-0000-4000-8000-000000000003'::uuid, 'parallel', 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'c0000000-0000-4000-8000-000000000005'::uuid, 
 '同時分析前端性能和後端 API 效能',
 '並行分析結果：\n\n**前端性能分析**\n- First Contentful Paint: 1.2s (良好)\n- Time to Interactive: 2.8s (需優化)\n- Bundle Size: 2.1MB (偏大)\n\n**後端 API 分析**\n- 平均響應時間: 45ms\n- P95 延遲: 120ms\n- 吞吐量: 1000 req/s\n\n**綜合建議**\n1. 前端：實施 lazy loading 和 code splitting\n2. 後端：增加快取層減少資料庫查詢',
 '[
   {"step": "frontend_analysis", "parallel": true, "tools": ["lighthouse", "webpack-analyzer"], "duration": 2500},
   {"step": "backend_analysis", "parallel": true, "tools": ["pprof", "trace"], "duration": 2000},
   {"step": "synthesis", "inputs": 2, "duration": 300}
 ]'::jsonb,
 2800, 450, true,
 '{"parallel_branches": 2, "max_concurrency": 4, "aggregation_method": "weighted_average"}'::jsonb),

-- 條件鏈執行
('ce100000-0000-4000-8000-000000000004'::uuid, 'conditional', 'a0000000-0000-4000-8000-000000000001'::uuid, 
 'c0000000-0000-4000-8000-000000000006'::uuid, 
 '根據系統負載決定部署策略',
 '系統負載分析結果：高負載\n\n採用漸進式部署策略：\n1. 先部署到金絲雀環境 (10% 流量)\n2. 監控 30 分鐘\n3. 逐步增加流量：25% -> 50% -> 100%\n4. 保留舊版本 24 小時以便回滾\n\n預計總部署時間：2 小時',
 '[
   {"step": "load_check", "condition": "check_system_load", "result": "high", "branch": "canary"},
   {"step": "canary_deploy", "percentage": 10, "duration": 300},
   {"step": "monitor", "duration": 1800, "metrics": ["error_rate", "latency"]},
   {"step": "gradual_rollout", "stages": [25, 50, 100], "duration": 3600}
 ]'::jsonb,
 5700, 380, true,
 '{"decision_path": "high_load_canary", "alternatives_considered": ["blue_green", "rolling_update"]}'::jsonb);

-- 顯示新增資料統計
SELECT '=== 新增資料統計 ===' as summary;
SELECT '新增使用者偏好:' as entity, COUNT(*) as count FROM user_preferences WHERE id::text LIKE 'c61%'
UNION ALL
SELECT '新增使用者上下文:', COUNT(*) FROM user_context WHERE id::text LIKE 'd71%'
UNION ALL
SELECT '新增搜尋快取:', COUNT(*) FROM search_cache WHERE query_hash LIKE 'hash_%'
UNION ALL
SELECT '新增工具快取:', COUNT(*) FROM tool_cache WHERE id::text LIKE 'b51%'
UNION ALL
SELECT '新增代理執行:', COUNT(*) FROM agent_executions WHERE id::text LIKE 'ae1%'
UNION ALL
SELECT '新增鏈執行:', COUNT(*) FROM chain_executions WHERE id::text LIKE 'ce1%';

-- 總計
SELECT '=== 總資料量 ===' as summary;
SELECT '總使用者偏好:' as entity, COUNT(*) as total FROM user_preferences
UNION ALL
SELECT '總使用者上下文:', COUNT(*) FROM user_context
UNION ALL
SELECT '總搜尋快取:', COUNT(*) FROM search_cache
UNION ALL
SELECT '總工具快取:', COUNT(*) FROM tool_cache
UNION ALL
SELECT '總代理執行:', COUNT(*) FROM agent_executions
UNION ALL
SELECT '總鏈執行:', COUNT(*) FROM chain_executions;