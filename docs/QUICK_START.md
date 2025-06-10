# 🚀 Assistant Go 快速開始指南

## 📋 前置要求

- Go 1.22+ 
- PostgreSQL 15+ (含 pgvector 擴充套件)
- Make
- Git

## 🔧 快速設定步驟

### 1. 克隆專案

```bash
git clone https://github.com/koopa0/assistant-go.git
cd assistant-go
```

### 2. 設定環境變數

```bash
# 複製環境變數範例檔案
cp .env.example .env

# 編輯 .env 檔案，填入必要的設定
# 最少需要設定：
# - DATABASE_URL
# - SECURITY_JWT_SECRET  
# - CLAUDE_API_KEY 或 GEMINI_API_KEY
```

**必需的環境變數**：
```bash
# 資料庫連接
export DATABASE_URL="postgresql://user:password@localhost:5432/assistant?sslmode=disable"

# JWT Secret（使用強密碼）
export SECURITY_JWT_SECRET="$(openssl rand -base64 32)"

# AI API Key（至少一個）
export CLAUDE_API_KEY="your-claude-api-key"
```

### 3. 設定資料庫

```bash
# 建立資料庫
createdb assistant

# 安裝 pgvector 擴充套件
psql -d assistant -c "CREATE EXTENSION IF NOT EXISTS vector;"

# 執行資料庫遷移
make migrate-up

# （可選）載入測試資料
psql -d assistant < scripts/seed_user_koopa.sql
```

### 4. 安裝依賴和建置

```bash
# 安裝開發工具
make setup

# 安裝 Go 依賴
go mod download

# 建置專案
make build
```

### 5. 啟動服務

#### API 伺服器模式
```bash
# 開發模式（含熱重載）
make dev

# 或直接執行
./bin/assistant serve
```

#### CLI 模式
```bash
./bin/assistant cli
```

#### 直接查詢模式
```bash
./bin/assistant ask "如何優化 PostgreSQL 查詢效能？"
```

## 🔐 登入系統

### 使用預設的 Koopa 使用者

如果您執行了 seed_user_koopa.sql：
- **Email**: koopa@assistant.local
- **Password**: KoopaAssistant2024!

### API 登入範例

```bash
# 取得 JWT token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "koopa@assistant.local",
    "password": "KoopaAssistant2024!"
  }'

# 使用 token 呼叫 API
curl -X POST http://localhost:8080/api/query \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "解釋 Go 的 interface{} 用法"
  }'
```

### CLI 登入

```bash
# 啟動 CLI
./bin/assistant cli

# 在提示符下輸入
> login
Email: koopa@assistant.local
Password: *********************

# 登入成功後即可使用
> 請幫我分析這個專案的架構
```

## 📊 驗證安裝

### 1. 檢查服務健康狀態
```bash
curl http://localhost:8080/api/health
```

### 2. 檢查系統狀態
```bash
curl http://localhost:8080/api/status
```

### 3. 執行品質檢查
```bash
make quick-check
```

## 🛠️ 常用 Make 指令

```bash
make help          # 顯示所有可用指令
make dev           # 開發模式（熱重載）
make test          # 執行單元測試
make lint          # 程式碼檢查
make quick-check   # 快速品質檢查
make generate      # 重新生成 sqlc 程式碼
```

## ⚠️ 常見問題

### 1. JWT Secret 錯誤
**問題**：系統 panic 提示 "JWT secret is required"
**解決**：確保設定了 SECURITY_JWT_SECRET 環境變數

### 2. 資料庫連接失敗
**問題**：無法連接到 PostgreSQL
**解決**：
- 確認 PostgreSQL 服務正在執行
- 檢查 DATABASE_URL 格式是否正確
- 確認使用者權限和密碼

### 3. pgvector 未安裝
**問題**：遷移失敗，提示 vector 類型不存在
**解決**：
```bash
# Ubuntu/Debian
sudo apt-get install postgresql-15-pgvector

# macOS (使用 Homebrew)
brew install pgvector

# 然後在資料庫中啟用
psql -d assistant -c "CREATE EXTENSION vector;"
```

### 4. CORS 錯誤
**問題**：前端無法呼叫 API
**解決**：在開發環境中，預設允許 localhost:3000 和 localhost:8080。生產環境需要設定 ALLOWED_ORIGINS。

## 📚 更多資源

- [完整 API 文檔](./CLI_AND_API_REFERENCE.md)
- [架構指南](./CLAUDE-ARCHITECTURE.md)
- [安全報告](./SECURITY_FIXES_AND_COMPLIANCE.md)
- [開發指南](./DEVELOPMENT_GUIDE.md)

## 💡 提示

1. **開發環境**：使用 `make dev` 可以啟動熱重載模式
2. **測試**：執行 `make test-integration` 進行完整測試
3. **效能**：設定 `ENABLE_PROFILING=true` 啟用效能分析
4. **安全**：定期更新依賴套件 `go get -u ./...`

## 🆘 需要幫助？

如果遇到問題：
1. 檢查日誌輸出
2. 執行 `make quick-check` 確認環境正確
3. 查看 [故障排除指南](./TROUBLESHOOTING.md)
4. 在 GitHub 提出 issue

祝您使用愉快！🎉