## API 端點需求

根據 `API_DOCUMENTATION.md` 的規格，需要後端實作以下端點：

### WebSocket 端點

- `ws://localhost:8100/ws?token=<jwt_token>`

### 記憶系統 API

- `GET /memory/nodes` - 取得記憶節點
- `GET /memory/graph` - 取得記憶圖譜
- `PUT /memory/nodes/:id` - 更新記憶節點

### 對話系統 API

- `GET /conversations` - 取得對話列表
- `POST /conversations` - 建立新對話
- `GET /conversations/:id` - 取得對話詳情
- `POST /conversations/:id/messages` - 發送訊息

### 工具系統 API

- `GET /tools` - 取得工具列表
- `POST /tools/:id/execute` - 執行工具
- `GET /tools/executions/:id` - 取得執行狀態
