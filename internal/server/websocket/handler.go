package websocket

import (
	"log/slog"
	"net/http"
	"strings"
)

// HTTPHandler 處理 WebSocket 的 HTTP 請求
type HTTPHandler struct {
	service *WebSocketService
	logger  *slog.Logger
}

// NewHTTPHandler 建立新的 HTTP 處理器
func NewHTTPHandler(service *WebSocketService, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes 註冊 WebSocket 路由
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	// WebSocket 端點
	mux.HandleFunc("GET /ws", h.handleWebSocket)

	// WebSocket 管理 API
	mux.HandleFunc("GET /api/v1/websocket/stats", h.handleGetStats)
	mux.HandleFunc("GET /api/v1/websocket/users", h.handleGetConnectedUsers)
}

// handleWebSocket 處理 WebSocket 連接請求
func (h *HTTPHandler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 驗證 JWT token
	userID, err := h.extractUserFromToken(r)
	if err != nil {
		h.logger.Warn("WebSocket authentication failed", slog.Any("error", err))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 處理 WebSocket 升級
	if err := h.service.HandleWebSocket(w, r, userID); err != nil {
		h.logger.Error("Failed to handle WebSocket connection", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleGetStats 處理取得 WebSocket 統計的請求
func (h *HTTPHandler) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.service.GetConnectionStats()

	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    stats,
		"message": "WebSocket 統計取得成功",
	})
}

// handleGetConnectedUsers 處理取得已連接使用者的請求
func (h *HTTPHandler) handleGetConnectedUsers(w http.ResponseWriter, r *http.Request) {
	users := h.service.GetConnectedUsers()

	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"users": users,
			"total": len(users),
		},
		"message": "已連接使用者列表取得成功",
	})
}

// extractUserFromToken 從 JWT token 中提取使用者 ID
func (h *HTTPHandler) extractUserFromToken(r *http.Request) (string, error) {
	// 從查詢參數取得 token
	token := r.URL.Query().Get("token")
	if token == "" {
		// 從 Authorization header 取得 token
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		// 暫時使用預設的 Koopa 用戶進行測試
		h.logger.Debug("No token provided, using default user")
		return "a0000000-0000-4000-8000-000000000001", nil
	}

	// TODO: 實現真實的 JWT 驗證
	// 目前暫時返回預設用戶
	h.logger.Debug("Token validation not implemented, using default user")
	return "a0000000-0000-4000-8000-000000000001", nil
}

// writeJSONResponse 寫入 JSON 回應
func (h *HTTPHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	// 簡單的 JSON 編碼（生產環境應使用更完整的實現）
	response := `{"success": true, "message": "WebSocket service running"}`
	w.Write([]byte(response))
}
