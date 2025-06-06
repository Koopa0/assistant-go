# Assistant 測試計劃

## 測試環境設定
- Claude Model: claude-3-sonnet-20240229 (建議)
- Database: PostgreSQL with pgvector
- Go Version: 1.24.2

## 測試範圍

### 1. 單元測試
- [ ] AI Service 測試
  - [ ] Claude 客戶端
  - [ ] Prompt 增強系統
  - [ ] 任務分類
- [ ] Memory 系統測試
  - [ ] Working Memory
  - [ ] 數據庫持久化
- [ ] Tool Registry 測試
  - [ ] 工具註冊
  - [ ] 工具執行
- [ ] Conversation Manager 測試

### 2. 整合測試
- [ ] API 端點測試
  - [ ] /api/v1/query
  - [ ] /api/v1/conversations
  - [ ] /api/v1/tools
  - [ ] WebSocket 連接
- [ ] CLI 功能測試
  - [ ] 互動模式
  - [ ] 直接查詢模式
  - [ ] 工具執行

### 3. 端到端測試
- [ ] 完整對話流程
- [ ] 工具整合流程
- [ ] 記憶系統整合

### 4. 性能測試
- [ ] API 響應時間
- [ ] 並發處理
- [ ] 記憶體使用

### 5. 功能驗證
- [ ] 7 種 Prompt 模板測試
- [ ] Go 開發工具測試
- [ ] Docker 工具測試
- [ ] PostgreSQL 工具測試