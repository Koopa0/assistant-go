# CLI 功能和使用情境文檔

本文檔詳細說明 Assistant 智能開發助手的命令列介面（CLI）功能、使用情境和操作指南。

**最後更新**: 2025-01-06  
**CLI 版本**: v0.1.0  
**文檔版本**: 2.0

## 🚀 快速開始

### 安裝和設置

```bash
# 克隆專案
git clone https://github.com/koopa0/assistant-go.git
cd assistant-go

# 設置開發環境（安裝工具和依賴）
make setup

# 編譯專案
make build

# 查看版本資訊
./bin/assistant version
```

### 必要環境設定

```bash
# 設定 AI 服務 API Key（至少需要其中一個）
export CLAUDE_API_KEY="your-claude-api-key"
export GEMINI_API_KEY="your-gemini-api-key"

# 設定資料庫連接
export DATABASE_URL="postgres://user:password@localhost:5432/assistant?sslmode=disable"
```

## 📋 支援的操作模式

Assistant CLI 提供三種主要操作模式：

### 1. 🖥️ 伺服器模式 (Server Mode)

**命令**: `assistant serve` 或 `assistant server`

**功能**: 啟動 HTTP API 伺服器，提供完整的 RESTful API 和 WebSocket 服務

**使用情境**:
- 生產環境部署
- 與前端應用程式整合
- 多使用者並行存取
- 微服務架構整合

**啟動範例**:
```bash
# 使用預設設定啟動
./bin/assistant serve

# 自訂端口
export SERVER_PORT=8080
./bin/assistant serve

# 指定設定檔
export CONFIG_FILE=./configs/production.yaml
./bin/assistant serve

# 啟用除錯模式
export LOG_LEVEL=debug
./bin/assistant serve
```

**伺服器特性**:
- ✅ Go 1.24 新路由模式
- ✅ WebSocket 即時通訊
- ✅ JWT 令牌認證
- ✅ 請求限流和速率控制
- ✅ 健康檢查端點
- ✅ OpenTelemetry 整合
- ✅ 優雅關閉處理

### 2. 💬 互動模式 (Interactive CLI)

**命令**: `assistant cli` 或 `assistant interactive`

**功能**: 啟動即時對話介面，支援持續對話和豐富的命令

**使用情境**:
- 本地開發調試
- 即時程式碼諮詢
- 工具測試和探索
- 學習 Assistant 功能

**啟動範例**:
```bash
# 啟動互動模式
./bin/assistant cli

# 使用特定 AI 提供者
export ANTHROPIC_MODEL=claude-3-opus-20240229
./bin/assistant cli
```

### 3. ❓ 直接查詢模式 (Direct Query)

**命令**: `assistant ask "<問題>"`

**功能**: 單次問答，適合腳本整合和快速查詢

**使用情境**:
- Shell 腳本整合
- CI/CD 管道中的程式碼分析
- 批次處理任務
- 快速單次查詢

**使用範例**:
```bash
# 基本查詢
./bin/assistant ask "Go 中如何處理錯誤？"

# 分析檔案內容
./bin/assistant ask "分析這段程式碼的效能問題: $(cat main.go)"

# 在腳本中使用
#!/bin/bash
CODE_REVIEW=$(./bin/assistant ask "檢查程式碼品質: $(git diff HEAD~1)")
echo "$CODE_REVIEW" > review.md
```

## 🎯 互動模式詳細功能

### 基本命令

| 命令 | 別名 | 功能 |
|------|------|------|
| `help` | `?` | 顯示幫助資訊 |
| `exit` | `quit`, `bye` | 退出程式 |
| `clear` | `cls` | 清除螢幕 |
| `status` | - | 顯示系統狀態 |
| `history` | - | 檢視命令歷史 |
| `theme` | - | 切換色彩主題 |

### 當前實現的互動功能

```bash
$ ./bin/assistant cli
🤖 Assistant 互動模式 v0.1.0

Type 'help' for available commands, 'exit' to quit

assistant> help
可用命令:
  help, ?          顯示此幫助訊息
  exit, quit, bye  退出助手
  clear, cls       清除螢幕
  status          顯示系統狀態
  tools           列出可用工具
  history         顯示命令歷史
  theme <style>   切換色彩主題 (dark/light)

assistant> status
🔧 系統狀態
數據庫: ✅ 已連接
AI 服務: ✅ Claude (claude-3-sonnet-20240229)
可用工具: 3
記憶體使用: 45.2 MB
運行時間: 00:02:15

assistant> tools
📋 可用工具:
┌─────────────┬─────────────┬────────────────────────┬────────┐
│ 名稱        │ 分類        │ 描述                   │ 狀態   │
├─────────────┼─────────────┼────────────────────────┼────────┤
│ godev       │ development │ Go 開發工具            │ ✅ 啟用 │
│ docker      │ devops      │ Docker 管理工具        │ ✅ 啟用 │
│ postgres    │ database    │ PostgreSQL 管理工具    │ ✅ 啟用 │
└─────────────┴─────────────┴────────────────────────┴────────┘
```

### 智能對話功能

在互動模式中，任何不是命令的輸入都會被當作查詢發送給 AI：

```bash
assistant> 解釋 Go 的 context 包
🤔 正在思考...

Context 包是 Go 中處理請求範圍數據、取消信號和截止時間的標準方式...

assistant> 如何優化這段代碼的性能？
[貼上代碼]
🤔 分析中...

根據我的分析，這段代碼有以下優化建議：
1. 使用 sync.Pool 重用物件...
2. 避免在循環中進行內存分配...
```

## 🛠️ 工具整合

### 已實現的工具

#### 1. Go 開發工具 (godev)

**功能**:
- 工作區分析和專案類型檢測
- AST 語法樹分析
- 代碼複雜度計算
- 依賴關係分析
- 測試覆蓋率報告

**使用範例**:
```bash
# 分析當前工作區
assistant> analyze workspace

# 檢查代碼複雜度
assistant> 分析 main.go 的圈複雜度

# 生成依賴關係圖
assistant> 顯示專案的依賴關係
```

#### 2. Docker 工具

**功能**:
- Dockerfile 分析和優化
- 容器管理操作
- 多階段構建建議
- 安全掃描
- 構建優化

**使用範例**:
```bash
# 分析 Dockerfile
assistant> 分析 Dockerfile 並提供優化建議

# 檢查映像大小
assistant> 如何減小 Docker 映像大小？

# 安全檢查
assistant> 檢查 Dockerfile 的安全問題
```

#### 3. PostgreSQL 工具

**功能**:
- SQL 查詢分析和優化
- 遷移檔案生成
- 架構分析
- 索引建議
- 性能檢查

**使用範例**:
```bash
# 優化查詢
assistant> 優化這個 SQL 查詢：SELECT * FROM users WHERE created_at > NOW() - INTERVAL '7 days'

# 生成遷移
assistant> 為新增 email_verified 欄位生成遷移檔案

# 索引建議
assistant> 分析 users 表並建議索引
```

### 即將推出的工具

- **Kubernetes**: K8s 資源管理、Manifest 優化
- **Git**: 版本控制操作、Commit 分析
- **Cloudflare**: CDN 管理、Workers 部署
- **Monitoring**: Prometheus/Grafana 整合

## 📊 進階功能

### 增強的 Prompt 系統

Assistant 現在使用專門的 Prompt 模板來處理不同類型的任務：

| 任務類型 | 觸發關鍵字 | 功能 |
|----------|-----------|------|
| 代碼分析 | 分析、檢查、review | 深入的代碼品質分析 |
| 重構 | 重構、優化、改進 | 代碼重構建議 |
| 性能 | 性能、效能、速度 | 性能瓶頸分析 |
| 架構 | 架構、設計、結構 | 系統架構審查 |
| 測試 | 測試、test、單元測試 | 測試代碼生成 |
| 錯誤診斷 | 錯誤、bug、問題 | 錯誤根因分析 |
| 工作區 | 專案、workspace、項目 | 專案結構分析 |

### 記憶系統

Assistant 具有持久化記憶功能：

```bash
# 系統會記住你的偏好
assistant> 我偏好使用 testify 進行測試
✅ 已記住您的偏好

# 之後的建議會考慮你的偏好
assistant> 幫我生成單元測試
🤔 生成使用 testify 的測試代碼...
```

### 上下文感知

Assistant 能夠理解當前的開發上下文：

```bash
# 自動檢測專案類型
assistant> 分析當前專案
🔍 檢測到 Go 微服務專案
- 使用 Echo 框架
- PostgreSQL 資料庫
- Docker 容器化部署

# 基於上下文的建議
assistant> 如何改進專案結構？
基於您的微服務架構，建議採用以下結構...
```

## 🔧 配置管理

### 環境變數配置

```bash
# AI 服務配置
export CLAUDE_API_KEY="sk-ant-..."
export GEMINI_API_KEY="AIza..."
export ANTHROPIC_MODEL="claude-3-sonnet-20240229"

# 資料庫配置
export DATABASE_URL="postgres://user:pass@localhost:5432/assistant"
export DATABASE_MAX_CONNECTIONS=50

# 服務器配置
export SERVER_PORT=8100
export SERVER_READ_TIMEOUT=30s
export SERVER_WRITE_TIMEOUT=30s

# 日誌配置
export LOG_LEVEL=info  # debug, info, warn, error
export LOG_FORMAT=json # json, text

# 性能配置
export GOMAXPROCS=4
export GOGC=100
```

### 配置檔案

支援 YAML 格式的配置檔案：

```yaml
# configs/development.yaml
mode: development
server:
  port: 8100
  timeout:
    read: 30s
    write: 30s
ai:
  default_provider: claude
  claude:
    model: claude-3-sonnet-20240229
  gemini:
    model: gemini-pro
database:
  url: ${DATABASE_URL}
  max_open_conns: 25
  max_idle_conns: 5
logging:
  level: debug
  format: text
```

## 🐛 故障排除

### 常見問題

#### 1. AI 服務連接失敗

```bash
# 檢查 API Key
echo $CLAUDE_API_KEY
echo $GEMINI_API_KEY

# 測試連接
./bin/assistant ask "test" --debug

# 查看詳細錯誤
export LOG_LEVEL=debug
./bin/assistant cli
```

#### 2. 資料庫連接問題

```bash
# 檢查資料庫狀態
psql $DATABASE_URL -c "SELECT 1"

# 執行遷移
make migrate-up

# 重置資料庫
make migrate-down
make migrate-up
```

#### 3. 工具執行失敗

```bash
# 列出可用工具
./bin/assistant cli
assistant> tools

# 檢查工具狀態
assistant> status --verbose

# 查看工具日誌
tail -f logs/assistant.log | grep TOOL
```

### 除錯模式

```bash
# 啟用完整除錯
export LOG_LEVEL=debug
export ASSISTANT_DEBUG=true
export ASSISTANT_TRACE=true

# 性能分析
export ASSISTANT_PROFILE=cpu
./bin/assistant serve

# 查看 pprof
go tool pprof http://localhost:8100/debug/pprof/profile
```

## 🔗 整合範例

### CI/CD 整合

**.github/workflows/code-review.yml**:
```yaml
name: AI Code Review
on: [pull_request]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.24'
    
    - name: Install Assistant
      run: |
        go install github.com/koopa0/assistant-go/cmd/assistant@latest
    
    - name: Run Code Review
      env:
        CLAUDE_API_KEY: ${{ secrets.CLAUDE_API_KEY }}
      run: |
        # 分析變更
        git diff origin/main...HEAD > changes.diff
        assistant ask "請審查這些代碼變更並提供改進建議" < changes.diff > review.md
        
    - name: Comment PR
      uses: thollander/actions-comment-pull-request@v2
      with:
        filePath: review.md
```

### Shell 別名和函數

添加到 `~/.bashrc` 或 `~/.zshrc`:

```bash
# Assistant 別名
alias ai='assistant ask'
alias aic='assistant cli'
alias ais='assistant serve'

# 快捷函數
explain() {
    assistant ask "解釋這段代碼：$(cat $1)"
}

review() {
    assistant ask "審查這段代碼並提供改進建議：$(cat $1)"
}

optimize() {
    assistant ask "優化這段代碼的性能：$(cat $1)"
}

# Git 整合
ai-commit() {
    local diff=$(git diff --cached)
    if [ -z "$diff" ]; then
        echo "沒有暫存的變更"
        return 1
    fi
    assistant ask "基於以下變更生成 commit message：$diff"
}
```

### VS Code 整合

創建 `.vscode/tasks.json`:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Assistant: Explain Code",
      "type": "shell",
      "command": "assistant",
      "args": ["ask", "解釋這段代碼: ${file}"],
      "problemMatcher": []
    },
    {
      "label": "Assistant: Review Code",
      "type": "shell",
      "command": "assistant",
      "args": ["ask", "審查這段代碼: ${file}"],
      "problemMatcher": []
    }
  ]
}
```

## 📈 性能優化建議

### 記憶體使用

```bash
# 限制記憶體使用
export GOMEMLIMIT=500MiB
./bin/assistant serve

# 調整 GC
export GOGC=50  # 更頻繁的 GC
./bin/assistant cli
```

### 並發控制

```bash
# 設定工作線程數
export ASSISTANT_MAX_WORKERS=4

# CPU 限制
export GOMAXPROCS=2
```

### 快取配置

```bash
# 啟用查詢快取
export ASSISTANT_CACHE_ENABLED=true
export ASSISTANT_CACHE_TTL=3600

# 記憶體快取大小
export ASSISTANT_CACHE_SIZE=100MB
```

## 🚧 開發路線圖

### 近期計劃 (Q1 2025)

1. **檔案操作整合**
   - 直接讀寫檔案
   - 批次檔案處理
   - 智能檔案搜尋

2. **Git 深度整合**
   - 自動 commit message
   - PR 描述生成
   - 代碼審查自動化

3. **IDE 插件**
   - VS Code 擴展
   - JetBrains 插件
   - Vim/Neovim 整合

4. **團隊協作**
   - 共享知識庫
   - 團隊偏好設定
   - 代碼規範檢查

### 長期願景 (2025+)

1. **自主代理模式**
   - 自動錯誤修復
   - 主動性能優化
   - 智能重構建議

2. **多語言支援**
   - Python, JavaScript, Rust
   - 跨語言分析
   - 多語言專案支援

3. **企業級功能**
   - LDAP/SSO 整合
   - 審計日誌
   - 合規性檢查

---

**維護者**: Assistant 開發團隊  
**支援**: [GitHub Issues](https://github.com/koopa0/assistant-go/issues)  
**授權**: MIT License