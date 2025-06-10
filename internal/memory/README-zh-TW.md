# 🧠 Memory Package - 智慧記憶系統套件

## 📋 概述

Memory 套件實現了 Assistant 的多層次記憶系統，模擬人類認知記憶結構。這個套件提供工作記憶（短期）、情節記憶（經驗）、語義記憶（知識）、程序記憶（技能）等多種記憶類型，讓 AI 助理能夠學習、記住並應用過去的經驗。

## 🏗️ 架構設計

### 記憶層次結構

```go
// MemorySystem 統一管理所有記憶類型
type MemorySystem struct {
    working    *WorkingMemory     // 工作記憶（當前任務）
    episodic   *EpisodicMemory    // 情節記憶（經驗）
    semantic   *SemanticMemory    // 語義記憶（知識）
    procedural *ProceduralMemory  // 程序記憶（技能）
    prospective *ProspectiveMemory // 前瞻記憶（計劃）
}

// Memory 基礎介面
type Memory interface {
    Store(ctx context.Context, item MemoryItem) error
    Retrieve(ctx context.Context, query Query) ([]MemoryItem, error)
    Update(ctx context.Context, id string, updates map[string]any) error
    Delete(ctx context.Context, id string) error
    Consolidate(ctx context.Context) error
}
```

### 記憶項目結構

```go
// MemoryItem 統一的記憶項目格式
type MemoryItem struct {
    ID          string              // 唯一識別碼
    Type        MemoryType          // 記憶類型
    Content     interface{}         // 記憶內容
    Context     map[string]any      // 上下文資訊
    Timestamp   time.Time           // 創建時間
    AccessCount int                 // 訪問次數
    Importance  float64             // 重要性分數
    Decay       float64             // 遺忘係數
    Embeddings  []float64           // 向量表示
    Metadata    map[string]any      // 元資料
}
```

## 🔧 核心功能

### 1. 工作記憶 (Working Memory)

處理當前任務的短期記憶，容量有限但快速存取：

```go
type WorkingMemory struct {
    capacity   int                 // 容量限制（預設 7±2）
    items      []MemoryItem        // 當前項目
    focus      int                 // 注意力焦點
    duration   time.Duration       // 保持時間
}

// 功能特性
- 容量限制：模擬人類工作記憶的 7±2 法則
- 快速存取：O(1) 時間複雜度
- 自動清理：超時項目自動移除
- 焦點管理：追蹤當前注意力焦點

// 使用範例
wm := memory.NewWorkingMemory(memory.WorkingMemoryConfig{
    Capacity: 7,
    Duration: 5 * time.Minute,
})

// 存儲當前任務上下文
wm.Store(ctx, MemoryItem{
    Content: "用戶要求優化 SQL 查詢",
    Context: map[string]any{
        "query": "SELECT * FROM users",
        "table": "users",
        "goal":  "performance",
    },
})
```

### 2. 情節記憶 (Episodic Memory)

儲存具體經驗和事件：

```go
type EpisodicMemory struct {
    storage    Storage             // 持久化存儲
    index      *Index              // 快速索引
    maxItems   int                 // 最大項目數
    importance ImportanceCalculator // 重要性計算
}

// 情節結構
type Episode struct {
    Event       string              // 事件描述
    Context     Context             // 發生上下文
    Actions     []Action            // 執行的動作
    Result      Result              // 結果
    Learning    []Insight           // 學到的經驗
    Timestamp   time.Time           // 時間戳記
}

// 功能特性
- 時序組織：按時間順序組織經驗
- 相似搜尋：基於向量找相似經驗
- 重要性排序：重要經驗優先保留
- 經驗總結：自動提取經驗教訓
```

### 3. 語義記憶 (Semantic Memory)

組織和儲存概念知識：

```go
type SemanticMemory struct {
    graph      *KnowledgeGraph     // 知識圖譜
    embeddings *EmbeddingStore     // 向量存儲
    reasoner   *Reasoner           // 推理引擎
}

// 知識節點
type KnowledgeNode struct {
    ID         string              // 節點 ID
    Concept    string              // 概念名稱
    Definition string              // 定義
    Properties map[string]any      // 屬性
    Relations  []Relation          // 關係
    Confidence float64             // 置信度
}

// 功能特性
- 知識圖譜：概念間的關係網絡
- 推理能力：基於已知推導未知
- 向量搜尋：語義相似性搜尋
- 知識更新：動態更新和修正
```

### 4. 程序記憶 (Procedural Memory)

儲存技能和操作流程：

```go
type ProceduralMemory struct {
    procedures map[string]*Procedure // 程序集合
    executor   *Executor             // 執行引擎
    optimizer  *Optimizer            // 優化器
}

// 程序定義
type Procedure struct {
    Name        string              // 程序名稱
    Steps       []Step              // 執行步驟
    Conditions  []Condition         // 前置條件
    Parameters  []Parameter         // 參數定義
    Success     SuccessCriteria     // 成功標準
    Learned     time.Time           // 學習時間
    Usage       int                 // 使用次數
    Performance []PerformanceMetric // 效能指標
}

// 功能特性
- 技能學習：從經驗中提取程序
- 自動優化：基於效能改進流程
- 條件執行：根據條件選擇程序
- 組合能力：組合簡單程序為複雜技能
```

### 5. 前瞻記憶 (Prospective Memory)

管理未來意圖和計劃：

```go
type ProspectiveMemory struct {
    intentions []Intention         // 意圖列表
    scheduler  *Scheduler          // 排程器
    reminders  *ReminderSystem     // 提醒系統
}

// 意圖結構
type Intention struct {
    ID          string             // 意圖 ID
    Goal        string             // 目標描述
    Trigger     Trigger            // 觸發條件
    Actions     []Action           // 計劃動作
    Priority    Priority           // 優先級
    Deadline    *time.Time         // 截止時間
    Status      IntentionStatus    // 狀態
}

// 功能特性
- 意圖管理：追蹤未來目標
- 智慧提醒：在適當時機提醒
- 優先級排序：重要任務優先
- 進度追蹤：監控完成情況
```

## 📊 進階功能

### 1. 記憶鞏固 (Memory Consolidation)

模擬睡眠時的記憶鞏固過程：

```go
// 鞏固過程
func (m *MemorySystem) Consolidate(ctx context.Context) error {
    // 1. 從工作記憶轉移到長期記憶
    important := m.working.ExtractImportant()
    for _, item := range important {
        if item.Type == MemoryTypeEpisode {
            m.episodic.Store(ctx, item)
        }
    }
    
    // 2. 提取經驗中的知識
    patterns := m.episodic.ExtractPatterns()
    for _, pattern := range patterns {
        m.semantic.AddKnowledge(ctx, pattern)
    }
    
    // 3. 優化程序記憶
    m.procedural.OptimizeProcedures(ctx)
    
    // 4. 清理低價值記憶
    m.cleanup(ctx)
    
    return nil
}
```

### 2. 遺忘機制 (Forgetting Mechanism)

實現艾賓浩斯遺忘曲線：

```go
// 遺忘計算
func calculateRetention(item MemoryItem) float64 {
    elapsed := time.Since(item.Timestamp)
    accessBonus := math.Log(float64(item.AccessCount + 1))
    importanceBonus := item.Importance * 2
    
    // 艾賓浩斯公式的修改版
    retention := math.Exp(-elapsed.Hours()/24) * 
                 (1 + accessBonus + importanceBonus)
    
    return math.Min(retention, 1.0)
}

// 定期清理
func (m *MemorySystem) ForgetRoutine(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)
    for {
        select {
        case <-ticker.C:
            m.forgetLowValueMemories(ctx)
        case <-ctx.Done():
            return
        }
    }
}
```

### 3. 記憶檢索 (Memory Retrieval)

多策略記憶檢索系統：

```go
// 檢索策略
type RetrievalStrategy interface {
    Search(query Query, memories []MemoryItem) []MemoryItem
    Score(item MemoryItem, query Query) float64
}

// 組合檢索
func (m *MemorySystem) Retrieve(ctx context.Context, query Query) ([]MemoryItem, error) {
    var results []MemoryItem
    
    // 1. 語義相似性檢索
    semantic := m.semanticSearch(query)
    results = append(results, semantic...)
    
    // 2. 時間相關檢索
    temporal := m.temporalSearch(query)
    results = append(results, temporal...)
    
    // 3. 上下文匹配檢索
    contextual := m.contextualSearch(query)
    results = append(results, contextual...)
    
    // 4. 綜合排序
    return m.rankResults(results, query), nil
}
```

## 🔍 使用範例

### 完整工作流程
```go
// 創建記憶系統
memSystem := memory.NewMemorySystem(memory.Config{
    WorkingCapacity: 7,
    EpisodicMaxItems: 10000,
    ConsolidateInterval: 1 * time.Hour,
})

// 1. 存儲工作記憶
memSystem.Working.Store(ctx, memory.MemoryItem{
    Content: "用戶詢問如何優化數據庫查詢",
    Context: map[string]any{
        "topic": "database optimization",
        "urgency": "high",
    },
})

// 2. 搜尋相關經驗
experiences, _ := memSystem.Episodic.Retrieve(ctx, memory.Query{
    Keywords: []string{"database", "optimization", "SQL"},
    Limit: 5,
})

// 3. 應用已知程序
procedure, _ := memSystem.Procedural.GetProcedure("optimize_sql_query")
result := procedure.Execute(ctx, queryParams)

// 4. 儲存新經驗
memSystem.Episodic.Store(ctx, memory.Episode{
    Event: "成功優化 SQL 查詢",
    Actions: []string{"分析執行計劃", "添加索引", "重寫查詢"},
    Result: result,
    Learning: []string{"複合索引效果更好", "避免 SELECT *"},
})

// 5. 定期鞏固
memSystem.Consolidate(ctx)
```

## 🧪 測試策略

### 記憶效能測試
```go
func TestMemoryPerformance(t *testing.T) {
    mem := setupTestMemory(t)
    
    // 測試存儲效能
    t.Run("storage performance", func(t *testing.T) {
        start := time.Now()
        for i := 0; i < 1000; i++ {
            mem.Store(ctx, generateTestItem(i))
        }
        elapsed := time.Since(start)
        assert.Less(t, elapsed, 1*time.Second)
    })
    
    // 測試檢索效能
    t.Run("retrieval performance", func(t *testing.T) {
        start := time.Now()
        results, _ := mem.Retrieve(ctx, testQuery)
        elapsed := time.Since(start)
        assert.Less(t, elapsed, 100*time.Millisecond)
        assert.NotEmpty(t, results)
    })
}
```

## 🔧 配置選項

```yaml
memory:
  # 工作記憶配置
  working:
    capacity: 7
    duration: 5m
    
  # 情節記憶配置
  episodic:
    max_items: 10000
    importance_threshold: 0.3
    consolidation_batch: 100
    
  # 語義記憶配置
  semantic:
    graph_backend: "neo4j"
    embedding_dimension: 1536
    similarity_threshold: 0.8
    
  # 程序記憶配置
  procedural:
    max_procedures: 1000
    optimization_interval: 24h
    min_usage_for_optimization: 10
    
  # 遺忘配置
  forgetting:
    enabled: true
    check_interval: 1h
    retention_threshold: 0.1
    
  # 持久化配置
  storage:
    backend: "postgres"
    connection_string: "${DATABASE_URL}"
    pool_size: 10
```

## 📈 效能考量

1. **記憶體優化**
   - 使用記憶體映射檔案處理大型記憶
   - 實現分層存儲（熱/溫/冷）
   - 定期壓縮和整理

2. **查詢優化**
   - 向量索引加速語義搜尋
   - 時間索引優化時序查詢
   - 快取熱門查詢結果

3. **並發處理**
   - 讀寫鎖分離
   - 無鎖資料結構
   - 批次處理優化

## 🚀 未來規劃

1. **認知增強**
   - 實現注意力機制
   - 情緒對記憶的影響
   - 創造性聯想

2. **分散式記憶**
   - 跨節點記憶同步
   - 集體智慧整合
   - 隱私保護共享

3. **神經網路整合**
   - 使用 Transformer 改進檢索
   - 神經記憶網路
   - 持續學習能力

## 📚 相關文件

- [Conversation Package](../conversation/README-zh-TW.md) - 對話管理
- [AI Package](../ai/README-zh-TW.md) - AI 服務整合
- [Knowledge Graph](../knowledge/README-zh-TW.md) - 知識圖譜
- [主要架構文件](../../CLAUDE-ARCHITECTURE.md) - 系統架構指南