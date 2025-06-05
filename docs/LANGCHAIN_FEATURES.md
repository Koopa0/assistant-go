# LangChain 功能整合指南

Assistant 整合了 LangChain-Go，提供強大的 AI 鏈式推理、代理執行和記憶體管理功能。本文檔說明如何使用這些功能。

## 概述

LangChain 整合提供以下核心功能：

1. **智能代理 (Agents)**: 專門化的 AI 代理，各自擅長不同領域
2. **處理鏈 (Chains)**: 可組合的處理管道，支援複雜的推理流程
3. **記憶體系統 (Memory)**: 持久化和檢索對話記憶
4. **多 LLM 支援**: 支援多種語言模型提供者

## API 端點

### 代理管理

#### 列出可用代理
```http
GET /api/langchain/agents
```

回應範例：
```json
{
  "success": true,
  "data": {
    "agents": ["development", "database", "infrastructure", "research"],
    "count": 4
  }
}
```

#### 執行代理
```http
POST /api/langchain/agents/{type}/execute
Content-Type: application/json

{
  "user_id": "user123",
  "query": "分析這個 Go 專案的架構",
  "max_steps": 5,
  "context": {
    "project_path": "/path/to/project"
  }
}\n```

回應範例：
```json\n{\n  \"success\": true,\n  \"data\": {\n    \"agent_type\": \"development\",\n    \"response\": \"專案架構分析結果...\",\n    \"steps\": [\n      {\n        \"step\": 1,\n        \"action\": \"讀取專案結構\",\n        \"result\": \"找到 15 個 Go 檔案\"\n      }\n    ],\n    \"execution_time\": \"2.5s\",\n    \"tokens_used\": 1250,\n    \"metadata\": {\n      \"confidence\": 0.95\n    }\n  }\n}\n```

### 處理鏈管理

#### 列出可用處理鏈
```http
GET /api/langchain/chains
```

#### 執行處理鏈
```http
POST /api/langchain/chains/{type}/execute
Content-Type: application/json

{
  \"user_id\": \"user123\",
  \"input\": \"請幫我設計一個使用者認證系統\",
  \"context\": {
    \"technology\": \"Go\",
    \"database\": \"PostgreSQL\"
  }
}
```

### 記憶體系統

#### 儲存記憶
```http
POST /api/langchain/memory
Content-Type: application/json

{
  \"user_id\": \"user123\",
  \"type\": \"episodic\",
  \"content\": \"使用者偏好使用 PostgreSQL 作為主資料庫\",
  \"importance\": 0.8,
  \"metadata\": {
    \"category\": \"preference\",
    \"technology\": \"database\"
  }
}
```

#### 搜尋記憶
```http
POST /api/langchain/memory/search
Content-Type: application/json

{
  \"user_id\": \"user123\",
  \"query\": \"資料庫偏好\",
  \"types\": [\"episodic\", \"semantic\"],
  \"limit\": 10,
  \"threshold\": 0.7
}
```

#### 記憶體統計
```http
GET /api/langchain/memory/stats/{userID}
```

回應範例：
```json
{
  \"success\": true,
  \"data\": {
    \"user_id\": \"user123\",
    \"total_memories\": 45,
    \"memory_types\": {
      \"episodic\": 20,
      \"semantic\": 15,
      \"procedural\": 10
    },
    \"recent_activity\": \"2024-01-15T10:30:00Z\"
  }
}
```

## CLI 使用方式

### 基本命令

```bash
# 顯示 LangChain 幫助
langchain help
# 或簡寫
lc help

# 列出可用代理
agents
# 或
langchain agents

# 列出可用處理鏈
chains
# 或
langchain chains
```

### 代理執行

```bash
# 執行開發代理
langchain agents execute development \"分析 internal/server 目錄的程式碼結構\"

# 執行資料庫代理
langchain agents execute database \"優化這個查詢的效能\"

# 執行基礎架構代理
langchain agents execute infrastructure \"檢查 Kubernetes 部署狀態\"

# 執行研究代理
langchain agents execute research \"比較不同的認證方法\"
```

### 處理鏈執行

```bash
# 執行順序處理鏈
langchain chains execute sequential \"設計一個完整的使用者管理系統\"

# 執行條件處理鏈
langchain chains execute conditional \"根據效能需求選擇合適的快取策略\"

# 執行平行處理鏈
langchain chains execute parallel \"同時分析程式碼品質和安全性\"

# 執行 RAG 處理鏈
langchain chains execute rag \"基於專案文檔回答技術問題\"
```

## 代理類型說明

### 開發代理 (Development Agent)
- **專長**: 程式碼分析、架構設計、最佳實踐
- **適用場景**: 程式碼審查、重構建議、技術選型
- **範例查詢**: \"分析這個函數的複雜度\"、\"建議重構方案\"

### 資料庫代理 (Database Agent)
- **專長**: SQL 優化、資料庫設計、效能調優
- **適用場景**: 查詢優化、索引建議、資料庫架構
- **範例查詢**: \"優化這個慢查詢\"、\"設計使用者表結構\"

### 基礎架構代理 (Infrastructure Agent)
- **專長**: 部署、監控、容器化、雲端服務
- **適用場景**: Kubernetes 管理、Docker 優化、CI/CD
- **範例查詢**: \"設計 Kubernetes 部署檔\"、\"Docker 效能調優\"

### 研究代理 (Research Agent)
- **專長**: 技術調研、比較分析、最佳實踐研究
- **適用場景**: 技術選型、競品分析、趨勢研究
- **範例查詢**: \"比較不同的訊息佇列\"、\"分析微服務架構趨勢\"

## 處理鏈類型說明

### 順序處理鏈 (Sequential Chain)
- **功能**: 按順序執行多個步驟
- **適用**: 需要步驟依賴的複雜任務
- **範例**: 程式碼生成 → 測試 → 部署

### 條件處理鏈 (Conditional Chain)
- **功能**: 根據條件分支執行不同邏輯
- **適用**: 決策導向的任務
- **範例**: 根據效能需求選擇不同的實作方案

### 平行處理鏈 (Parallel Chain)
- **功能**: 同時執行多個獨立任務
- **適用**: 可並行處理的任務
- **範例**: 同時進行程式碼分析和文檔生成

### RAG 處理鏈 (RAG Chain)
- **功能**: 基於文檔檢索的增強生成
- **適用**: 需要參考現有知識庫的任務
- **範例**: 基於專案文檔回答技術問題

## 記憶體類型

### 情節記憶 (Episodic Memory)
- **內容**: 具體的對話經歷和事件
- **用途**: 記住使用者的特定需求和偏好
- **範例**: \"使用者上次要求使用 PostgreSQL\"

### 語義記憶 (Semantic Memory)
- **內容**: 通用知識和概念
- **用途**: 儲存技術知識和最佳實踐
- **範例**: \"Go 的並發模式最佳實踐\"

### 程序記憶 (Procedural Memory)
- **內容**: 如何執行特定任務的知識
- **用途**: 記住常用的工作流程和步驟
- **範例**: \"如何設定 Kubernetes 部署\"

## 設定

LangChain 功能可通過設定檔案控制：

```yaml
tools:
  langchain:
    enable_memory: true      # 啟用記憶體功能
    memory_size: 10          # 記憶體大小限制
    max_iterations: 5        # 代理最大迭代次數
    timeout: 60s            # 執行逾時時間
```

環境變數：
```bash
LANGCHAIN_ENABLE_MEMORY=true
LANGCHAIN_MEMORY_SIZE=10
LANGCHAIN_MAX_ITERATIONS=5
LANGCHAIN_TIMEOUT=60s
```

## 使用範例

### 完整的開發工作流程

```bash
# 1. 分析現有程式碼
langchain agents execute development \"分析 internal/server 的架構問題\"

# 2. 設計改進方案
langchain chains execute sequential \"基於分析結果設計重構方案\"

# 3. 儲存決策到記憶體
# (透過 API 或代理自動儲存)

# 4. 實作具體功能
langchain agents execute development \"實作使用者認證中介軟體\"

# 5. 資料庫設計
langchain agents execute database \"設計使用者表和權限表結構\"
```

### API 整合範例

```javascript
// JavaScript 前端整合範例
async function analyzeCode(projectPath) {
  const response = await fetch('/api/langchain/agents/development/execute', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      user_id: 'current_user',
      query: `分析專案 ${projectPath} 的程式碼品質`,
      context: { project_path: projectPath }
    })
  });
  
  const result = await response.json();
  return result.data.response;
}
```

## 最佳實踐

1. **代理選擇**: 根據任務性質選擇合適的代理
2. **鏈式組合**: 複雜任務可以組合多個處理鏈
3. **記憶體利用**: 充分利用記憶體系統避免重複工作
4. **上下文提供**: 提供足夠的上下文資訊提高結果品質
5. **迭代優化**: 根據結果回饋調整代理參數

## 故障排除

### 常見問題

1. **代理執行失敗**
   - 檢查 API 金鑰設定
   - 確認網路連線
   - 檢查請求參數格式

2. **記憶體搜尋無結果**
   - 調整相似度閾值
   - 檢查記憶體是否已儲存
   - 嘗試不同的搜尋詞彙

3. **處理鏈逾時**
   - 增加逾時設定
   - 簡化輸入內容
   - 檢查 LLM 服務狀態

### 監控和日誌

LangChain 功能會記錄詳細的執行日誌：

```bash
# 檢查 LangChain 健康狀態
curl http://localhost:8080/api/langchain/health

# 查看系統日誌
tail -f /var/log/assistant/langchain.log
```

## 進階功能

### 自定義代理

可以通過設定檔案或 API 建立自定義代理：

```yaml
custom_agents:
  security_agent:
    description: \"專門處理資安相關任務\"
    system_prompt: \"你是一個資安專家...\"
    tools: [\"vulnerability_scanner\", \"code_analyzer\"]
```

### 鏈式組合

```bash
# 複雜的鏈式處理
langchain chains execute sequential \"
  1. 分析程式碼架構 (development agent)
  2. 評估安全風險 (security agent) 
  3. 生成改進建議 (research agent)
  4. 建立實作計畫 (sequential chain)
\"
```

這個整合為 Assistant 提供了強大的 AI 能力，使其能夠處理更複雜的開發任務並提供更智能的協助。