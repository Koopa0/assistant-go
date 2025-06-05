# Storage 存儲層

這個包提供了 Assistant 項目的數據持久化層，實現了智能開發伴侶所需的存儲功能。

## 架構概述

存儲層採用分層架構，支援多種存儲後端：

```
storage/
├── postgres/          # PostgreSQL 實現
│   ├── client.go      # 主要客戶端
│   ├── sqlc/          # SQLC 生成的查詢
│   ├── migrations/    # 數據庫遷移
│   └── queries/       # SQL 查詢定義
└── cache/             # 緩存層實現
```

## 主要特性

### 🗄️ PostgreSQL 集成
- **類型安全查詢**: 使用 SQLC 生成類型安全的 SQL 查詢
- **連接池優化**: PostgreSQL 17+ 優化的連接池配置
- **向量支持**: 集成 pgvector 進行語義搜索
- **事務管理**: 完整的事務支持和錯誤處理

### 🧠 智能功能存儲
- **學習系統**: 存儲學習事件和模式識別結果
- **多層記憶**: 實現工作記憶、情節記憶、語義記憶和程序記憶
- **知識圖譜**: 節點和邊的關係存儲
- **Agent 協作**: 智能代理間的協作數據

### 📊 性能優化
- **智能索引**: 基於查詢模式的索引策略
- **批量操作**: 高效的批量插入和更新
- **查詢優化**: 使用 EXPLAIN ANALYZE 進行查詢分析
- **連接池監控**: 實時連接池統計和調優

## 快速開始

### 基本使用

```go
// 創建 PostgreSQL 客戶端
client, err := postgres.NewClient(ctx, config.DatabaseURL, logger)
if err != nil {
    return fmt.Errorf("failed to create database client: %w", err)
}
defer client.Close()

// 健康檢查
if err := client.Health(ctx); err != nil {
    return fmt.Errorf("database health check failed: %w", err)
}

// 創建對話
conversation, err := client.CreateConversation(ctx, userID, "Test Conversation", metadata)
if err != nil {
    return fmt.Errorf("failed to create conversation: %w", err)
}
```

### 向量操作

```go
// 創建嵌入向量
embedding := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
record, err := client.CreateEmbedding(ctx, "document", "doc-123", content, embedding, metadata)
if err != nil {
    return fmt.Errorf("failed to create embedding: %w", err)
}

// 相似性搜索
queryEmbedding := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
results, err := client.SearchSimilarEmbeddings(ctx, queryEmbedding, "document", 10, 0.7)
if err != nil {
    return fmt.Errorf("failed to search embeddings: %w", err)
}
```

### 智能記憶操作

```go
// 存儲情節記憶
episodicMemory := &EpisodicMemory{
    UserID:      userID,
    SessionID:   sessionID,
    Event:       "code_completion",
    Context:     contextData,
    Outcome:     "success",
    Confidence:  0.95,
}

err := client.CreateEpisodicMemory(ctx, episodicMemory)
if err != nil {
    return fmt.Errorf("failed to store episodic memory: %w", err)
}

// 檢索相關記憶
memories, err := client.GetRelevantMemories(ctx, userID, "code_completion", 10)
if err != nil {
    return fmt.Errorf("failed to retrieve memories: %w", err)
}
```

## 數據庫架構

### 核心表結構

#### 對話管理
- `conversations`: 對話元數據
- `messages`: 對話消息歷史
- `executions`: 工具執行記錄

#### 智能功能
- `learning_events`: 學習事件記錄
- `learned_patterns`: 識別的模式
- `user_skills`: 用戶技能評估

#### 記憶系統
- `working_memory`: 工作記憶（短期）
- `episodic_memories`: 情節記憶（經歷）
- `semantic_memories`: 語義記憶（知識）
- `procedural_memories`: 程序記憶（技能）

#### 知識圖譜
- `knowledge_nodes`: 知識節點
- `knowledge_edges`: 節點間關係
- `knowledge_evolution`: 知識演化追踪

#### Agent 協作
- `agent_definitions`: Agent 定義
- `agent_collaborations`: 協作記錄
- `agent_knowledge_shares`: 知識共享

### 索引策略

```sql
-- 向量相似性搜索索引
CREATE INDEX idx_embeddings_vector ON embeddings USING ivfflat (embedding vector_cosine_ops);

-- 時間序列查詢索引
CREATE INDEX idx_learning_events_time ON learning_events (created_at DESC, user_id);

-- 複合查詢索引
CREATE INDEX idx_memories_user_type ON episodic_memories (user_id, event_type, created_at DESC);

-- 部分索引（活躍會話）
CREATE INDEX idx_conversations_active ON conversations (user_id, updated_at DESC) 
WHERE status = 'active';
```

## 配置選項

### 連接池配置

```yaml
database:
  url: "postgres://user:pass@localhost:5432/assistant?sslmode=require"
  pool:
    max_conns: 30              # 最大連接數
    min_conns: 5               # 最小連接數
    max_conn_lifetime: 1h      # 連接最大生命週期
    max_conn_idle_time: 15m    # 最大空閒時間
    health_check_period: 1m    # 健康檢查間隔
```

### 查詢優化配置

```yaml
database:
  query_timeout: 30s           # 查詢超時時間
  batch_size: 1000            # 批量操作大小
  enable_query_log: false     # 是否記錄查詢日誌
  slow_query_threshold: 1s    # 慢查詢閾值
```

## 遷移管理

### 創建遷移

```bash
# 生成新遷移文件
make migration name="add_new_feature"

# 應用遷移
make migrate-up

# 回滾遷移
make migrate-down
```

### 遷移最佳實踐

1. **向後兼容**: 確保遷移不會破壞現有功能
2. **數據安全**: 在生產環境前測試所有遷移
3. **索引管理**: 大表索引創建使用 `CONCURRENTLY`
4. **事務控制**: 複雜遷移使用適當的事務邊界

## 性能監控

### 連接池監控

```go
// 獲取連接池統計
stats := client.GetPoolStats()
fmt.Printf("Active connections: %d/%d\n", stats.AcquiredConns, stats.MaxConns)
fmt.Printf("Idle connections: %d\n", stats.IdleConns)
fmt.Printf("Average acquire time: %v\n", stats.AcquireDuration)
```

### 查詢性能分析

```sql
-- 分析查詢計劃
EXPLAIN (ANALYZE, BUFFERS, TIMING) 
SELECT * FROM embeddings 
WHERE content_type = 'document' 
ORDER BY embedding <=> $1::vector 
LIMIT 10;
```

## 最佳實踐

### 1. 錯誤處理
```go
// 使用 %w 包裝錯誤以保持錯誤鏈
if err != nil {
    return fmt.Errorf("failed to create conversation: %w", err)
}

// 檢查特定錯誤類型
if errors.Is(err, pgx.ErrNoRows) {
    return ErrNotFound
}
```

### 2. 事務管理
```go
// 使用事務進行複雜操作
err := client.WithTransaction(ctx, func(tx pgx.Tx) error {
    // 在事務中執行多個操作
    if err := createConversation(tx, ...); err != nil {
        return err
    }
    if err := createMessage(tx, ...); err != nil {
        return err
    }
    return nil
})
```

### 3. 批量操作
```go
// 批量插入以提高性能
embeddings := make([]*EmbeddingRecord, len(documents))
for i, doc := range documents {
    embeddings[i] = &EmbeddingRecord{...}
}

if err := client.BatchCreateEmbeddings(ctx, embeddings); err != nil {
    return fmt.Errorf("failed to batch create embeddings: %w", err)
}
```

### 4. 查詢優化
```go
// 使用預編譯查詢
const selectQuery = `
    SELECT id, content_type, content_id, content_text, embedding, metadata, created_at
    FROM embeddings 
    WHERE content_type = $1 AND created_at > $2
    ORDER BY created_at DESC 
    LIMIT $3`

// 使用索引友好的查詢模式
results, err := client.query(ctx, selectQuery, contentType, since, limit)
```

## 故障排除

### 常見問題

1. **連接池耗盡**
   - 檢查連接洩漏
   - 調整池大小配置
   - 監控長時間運行的查詢

2. **查詢性能問題**
   - 使用 EXPLAIN ANALYZE 分析
   - 檢查索引使用情況
   - 優化查詢條件

3. **向量搜索緩慢**
   - 檢查 ivfflat 索引配置
   - 調整向量維度
   - 使用適當的相似性閾值

### 監控指標

- 連接池使用率
- 查詢執行時間
- 索引命中率
- 向量搜索性能
- 事務回滾率

## 相關文檔

- [SQLC 文檔](https://docs.sqlc.dev/)
- [pgx 驅動文檔](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [pgvector 文檔](https://github.com/pgvector/pgvector)
- [PostgreSQL 17 新特性](https://www.postgresql.org/docs/17/release-17.html)

---

*這個文檔涵蓋了存儲層的主要功能和最佳實踐。如需更多詳細訊息，請參考相應的 Go 文檔和測試文件。*