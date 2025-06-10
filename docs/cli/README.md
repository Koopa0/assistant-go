# Assistant Go - CLI 參考手冊

## 🎯 認證系統

CLI 需要登入才能使用。預設使用者：
- **Email**: koopa@assistant.local
- **Password**: KoopaAssistant2024!

## 🖥️ CLI 啟動模式

### 1. 互動模式
```bash
./assistant cli
```

### 2. 直接查詢模式
```bash
./assistant ask "你的問題"
```

## 📝 核心指令

### 基本指令
- **help** - 顯示可用指令
- **menu** - 進入互動式任務選單（新功能：互動式任務選單！）
- **exit/quit** - 退出 CLI
- **clear** - 清除螢幕
- **version** - 顯示版本資訊
- **login** - 登入系統

### 會話管理
- **new** - 開始新對話
- **history** - 查看對話歷史
- **save** - 儲存當前對話
- **load** - 載入先前的對話

## 🚀 互動式選單功能

輸入 `menu` 進入功能豐富的互動式選單：

### 🔧 開發工具

#### 1. 程式碼品質分析 (`analyzeCodeQuality`)
- 分析專案結構和組織
- 識別潛在問題和改進點
- 檢查最佳實踐遵循情況
- 提供測試覆蓋率建議

#### 2. 測試生成 (`generateTests`)
- 為指定檔案生成單元測試
- 包含正常情況測試
- 邊界條件測試
- 錯誤處理測試
- 使用 testify 框架

#### 3. 重構建議 (`suggestRefactoring`)
- 分析檔案或整個專案
- 提供詳細重構建議
- 專注於程式碼組織、設計模式和最佳實踐

#### 4. 效能分析 (`analyzePerformance`)
- CPU 效能分析
- 記憶體使用分析
- 並發效能分析
- 整體效能審查

### 🗄️ 資料庫工具

#### 1. SQL 查詢優化 (`optimizeSQL`)
- 分析和優化 SQL 查詢
- 提供索引建議
- 查詢計劃分析

#### 2. 架構設計審查 (`reviewSchema`)
- 審查資料庫架構設計
- 建議改進方案
- 識別潛在問題

#### 3. 查詢效能分析 (`analyzeQueryPerformance`)
- 分析查詢執行計劃
- 識別瓶頸
- 建議優化方案

#### 4. 遷移生成 (`generateMigration`)
- 生成資料庫遷移腳本
- 支援 up/down 遷移

### 🛠️ 基礎設施管理

#### 1. Docker 優化 (`optimizeDocker`)
- 分析 Dockerfile
- 建議優化方案
- 多階段建置建議

#### 2. Kubernetes 配置 (`k8sConfig`)
- 生成 Kubernetes manifests
- 審查現有配置
- 安全最佳實踐

#### 3. CI/CD 管線設定 (`setupCICD`)
- GitHub Actions 配置
- GitLab CI 設定
- 建置優化

#### 4. 安全掃描 (`securityScan`)
- 依賴項漏洞掃描
- 安全最佳實踐審查
- OWASP 合規檢查

### 🤖 AI 工具

#### 1. 上下文感知聊天 (`contextChat`)
- 多輪對話
- 上下文保留
- 專案感知回應

#### 2. 程式碼解釋 (`explainCode`)
- 詳細程式碼解釋
- 架構分析
- 設計模式識別

#### 3. 文件生成 (`generateDocs`)
- API 文件
- README 生成
- 程式碼註解

#### 4. 架構建議 (`architectureAdvice`)
- 系統設計建議
- 可擴展性建議
- 最佳實踐

## 🌊 串流支援

CLI 支援即時串流回應：
- 即時顯示 AI 回應
- 進度指示器顯示長時間操作
- 支援中斷（Ctrl+C）

## ⚙️ 配置選項

### 環境變數
- `CLI_HISTORY_FILE` - 歷史檔案位置（預設：`~/.assistant_history`）
- `CLI_MAX_HISTORY_SIZE` - 最大歷史記錄數（預設：1000）
- `CLI_ENABLE_COLORS` - 啟用彩色輸出（預設：true）
- `CLI_PROMPT_TEMPLATE` - 自訂提示符（預設：`assistant> `）

### 設定檔
CLI 設定儲存在 `configs/development.yaml` 或 `configs/production.yaml`：

```yaml
cli:
  history_file: ".assistant_history"
  max_history_size: 1000
  enable_colors: true
  prompt_template: "assistant> "
  enable_streaming: true
  stream_buffer_size: 1024
  show_execution_time: true
  show_token_usage: true
```

## 💡 使用技巧

### 1. 快速測試生成
```bash
> menu
選擇：開發工具 > 測試生成
輸入檔案路徑：internal/assistant/assistant.go
```

### 2. SQL 優化工作流程
```bash
> menu
選擇：資料庫工具 > SQL 查詢優化
貼上您的 SQL 查詢，輸入 'done' 結束
```

### 3. 安全審查
```bash
> menu
選擇：基礎設施管理 > 安全掃描
系統將自動掃描專案依賴和配置
```

### 4. 持續對話
```bash
> 請解釋這個專案的架構
[AI 回應...]
> 如何改進效能？
[AI 基於前面的上下文回應...]
```

## 🎨 UI 特性

### 顏色編碼
- 🟢 **綠色** - 成功訊息
- 🔵 **藍色** - 資訊提示
- 🟡 **黃色** - 警告訊息
- 🔴 **紅色** - 錯誤訊息
- ⚪ **灰色** - 輔助資訊

### 互動元件
- **選擇列表** - 使用箭頭鍵導航
- **確認提示** - Y/N 確認操作
- **文字輸入** - 支援多行輸入
- **進度條** - 顯示長時間操作進度

## 🔍 進階功能

### 1. 批次處理
```bash
# 使用管道輸入
echo "優化這個查詢: SELECT * FROM users" | ./assistant cli

# 從檔案讀取
./assistant cli < queries.txt
```

### 2. 腳本整合
```bash
#!/bin/bash
# 在腳本中使用 Assistant
RESPONSE=$(./assistant ask "生成 User 模型的測試")
echo "$RESPONSE" > user_test.go
```

### 3. 自訂工具整合
CLI 可以整合自訂工具，透過設定檔配置：
```yaml
tools:
  custom_tool:
    command: "custom-analyzer"
    args: ["--format", "json"]
```

## 🐛 故障排除

### 常見問題

1. **登入失敗**
   - 確認環境變數 `DATABASE_URL` 已設定
   - 檢查資料庫連線
   - 確認使用者憑證正確

2. **選單無法顯示**
   - 確保終端支援 UTF-8
   - 嘗試設定 `CLI_ENABLE_COLORS=false`

3. **歷史記錄遺失**
   - 檢查歷史檔案權限
   - 確認 `CLI_HISTORY_FILE` 路徑可寫入

4. **串流中斷**
   - 增加 `CLI_STREAM_BUFFER_SIZE`
   - 檢查網路連線穩定性

## 📚 相關文件

- [CLI 進階功能](./ADVANCED_FEATURES.md)
- [API 參考手冊](../api/README.md)
- [開發指南](../development/README.md)
- [快速開始](../QUICK_START.md)