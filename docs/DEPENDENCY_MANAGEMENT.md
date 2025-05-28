# Go 模組依賴管理指南

## 📋 概述

本文檔記錄了 assistant-go 項目的 Go 模組依賴管理策略，包括已解決的內部包衝突問題和推薦的升級流程。

## 🚨 已解決的問題

### 原始問題
執行 `go get -u all` 時遇到內部包依賴衝突：

```bash
go: cloud.google.com/go/vertexai@v0.13.4 missing internal/support package
go: go.opentelemetry.io/otel@v1.36.0 missing internal/attribute package  
go: google.golang.org/api@v0.234.0 missing transport/http/internal/propagation package
go: google.golang.org/grpc@v1.72.2 missing internal/grpcrand package
```

### 根本原因
- **內部包不穩定性**: `/internal/` 包不是公共 API，版本間可能變化
- **傳遞依賴衝突**: LangChain-Go 和 Google Cloud 依賴樹複雜
- **批量升級風險**: `go get -u all` 同時升級所有包導致版本不兼容

## ✅ 解決方案

### 1. Replace 指令策略

在 `go.mod` 中使用 replace 指令固定問題版本：

```go
// Replace directives to resolve internal package conflicts
replace (
	// Pin vertexai to avoid internal/support package issues
	cloud.google.com/go/vertexai => cloud.google.com/go/vertexai v0.12.0
	// Use latest compatible OpenTelemetry versions
	go.opentelemetry.io/otel => go.opentelemetry.io/otel v1.36.0
	go.opentelemetry.io/otel/metric => go.opentelemetry.io/otel/metric v1.36.0
	go.opentelemetry.io/otel/trace => go.opentelemetry.io/otel/trace v1.36.0
	// Use latest compatible Google API version
	google.golang.org/api => google.golang.org/api v0.232.0
	// Use latest compatible gRPC version
	google.golang.org/grpc => google.golang.org/grpc v1.72.0
)
```

### 2. 選擇性升級策略

**❌ 避免使用:**
```bash
go get -u all  # 可能導致版本衝突
```

**✅ 推薦使用:**
```bash
# 核心應用依賴
go get -u github.com/a-h/templ github.com/google/uuid github.com/jackc/pgx/v5 \
          github.com/joho/godotenv github.com/pgvector/pgvector-go \
          github.com/tmc/langchaingo gopkg.in/yaml.v3

# 工具庫依賴
go get -u golang.org/x/exp go.starlark.net github.com/AssemblyAI/assemblyai-go-sdk \
          github.com/PuerkitoBio/goquery github.com/ledongthuc/pdf \
          github.com/microcosm-cc/bluemonday

# 模板和Web依賴
go get -u github.com/Masterminds/sprig/v3 github.com/nikolalohinski/gonja \
          nhooyr.io/websocket
```

## 📊 當前依賴狀態

### 核心依賴版本
- **Go 版本**: 1.24.2
- **PostgreSQL 驅動**: github.com/jackc/pgx/v5 v5.7.5
- **向量數據庫**: github.com/pgvector/pgvector-go v0.3.0
- **AI 框架**: github.com/tmc/langchaingo v0.1.13
- **模板引擎**: github.com/a-h/templ v0.3.865

### 成功升級的依賴
| 依賴包 | 舊版本 | 新版本 | 狀態 |
|--------|--------|--------|------|
| golang.org/x/crypto | v0.37.0 | v0.38.0 | ✅ |
| golang.org/x/sys | v0.32.0 | v0.33.0 | ✅ |
| golang.org/x/text | v0.24.0 | v0.25.0 | ✅ |
| github.com/AssemblyAI/assemblyai-go-sdk | v1.3.0 | v1.10.0 | ✅ |
| github.com/PuerkitoBio/goquery | v1.8.1 | v1.10.3 | ✅ |
| github.com/Masterminds/sprig/v3 | v3.2.3 | v3.3.0 | ✅ |
| nhooyr.io/websocket | v1.8.7 | v1.8.17 | ✅ |

## 🔧 維護流程

### 日常開發
1. **驗證依賴**: `go run scripts/verify-dependencies.go`
2. **檢查構建**: `go build ./...`
3. **驗證模組**: `go mod verify`

### 定期升級 (建議每季度)
1. **備份當前狀態**:
   ```bash
   git checkout -b dependency-upgrade-$(date +%Y%m%d)
   cp go.mod go.mod.backup
   cp go.sum go.sum.backup
   ```

2. **按類別升級**:
   ```bash
   # 核心依賴
   go get -u github.com/jackc/pgx/v5 github.com/tmc/langchaingo
   
   # 測試構建
   go build ./...
   
   # 如果成功，繼續其他依賴
   go get -u golang.org/x/crypto golang.org/x/sys golang.org/x/text
   ```

3. **驗證和測試**:
   ```bash
   go mod tidy
   go build ./...
   go run scripts/verify-dependencies.go
   ```

### 緊急安全更新
對於安全漏洞，可以單獨升級特定包：
```bash
go get -u github.com/specific/vulnerable-package@latest
go mod tidy
go build ./...
```

## 🚀 驗證腳本

使用提供的驗證腳本檢查依賴狀態：

```bash
go run scripts/verify-dependencies.go
```

該腳本會測試：
- UUID 生成功能
- PostgreSQL 驅動和 pgvector 支持
- LangChain-Go 基本功能
- 配置文件處理 (YAML)

## 📝 故障排除

### 如果遇到內部包錯誤
1. 檢查是否有新的內部包衝突
2. 在 `go.mod` 中添加相應的 replace 指令
3. 使用已知穩定版本

### 如果構建失敗
1. 恢復備份: `cp go.mod.backup go.mod && cp go.sum.backup go.sum`
2. 清理模組快取: `go clean -modcache`
3. 重新整理: `go mod tidy`

### 版本約束衝突
如果遇到版本約束錯誤，考慮：
1. 調整 replace 指令中的版本
2. 等待上游依賴解決兼容性問題
3. 暫時保持當前穩定版本

## 🎯 最佳實踐

1. **漸進式升級**: 一次升級一個類別的依賴
2. **測試驅動**: 每次升級後都要運行完整測試
3. **文檔記錄**: 記錄重要的版本變更和原因
4. **監控安全**: 定期檢查依賴的安全公告
5. **保持穩定**: 生產環境優先考慮穩定性而非最新版本

---

**最後更新**: 2024年12月
**維護者**: assistant-go 開發團隊
