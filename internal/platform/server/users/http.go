package users

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/koopa0/assistant-go/internal/platform/observability"
)

// HTTPHandler handles HTTP requests for user operations
type HTTPHandler struct {
	service *UserService
	logger  *slog.Logger
}

// NewHTTPHandler creates a new HTTP handler for users
func NewHTTPHandler(service *UserService, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  observability.ServerLogger(logger, "users_http"),
	}
}

// RegisterRoutes registers all user API routes
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	// User profile management
	mux.HandleFunc("GET /api/users/profile", h.GetUserProfile)
	mux.HandleFunc("PUT /api/users/profile", h.UpdateUserProfile)
	mux.HandleFunc("PUT /api/users/preferences", h.UpdateUserPreferences)
	mux.HandleFunc("PUT /api/users/password", h.ChangePassword)

	// User statistics and activity
	mux.HandleFunc("GET /api/users/statistics", h.GetUserStatistics)
	mux.HandleFunc("GET /api/users/activity", h.GetUserActivity)
	mux.HandleFunc("GET /api/users/settings", h.GetUserSettings)

	// Favorite tools management
	mux.HandleFunc("POST /api/users/tools/{toolId}/favorite", h.AddFavoriteTool)
	mux.HandleFunc("DELETE /api/users/tools/{toolId}/favorite", h.RemoveFavoriteTool)

	// User registration (public endpoint)
	mux.HandleFunc("POST /api/users/register", h.RegisterUser)

	// Account management
	mux.HandleFunc("DELETE /api/users/account", h.DeactivateAccount)
}

// GetUserProfile returns the current user's profile
func (h *HTTPHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from auth context
	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	profile, err := h.service.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get user profile",
			slog.String("user_id", userID),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "取得使用者資料失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    profile,
		"message": "取得使用者資料成功",
	})
}

// UpdateUserProfile updates the current user's profile
func (h *HTTPHandler) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "請求格式錯誤")
		return
	}

	profile, err := h.service.UpdateProfile(ctx, userID, &req)
	if err != nil {
		h.logger.Error("Failed to update profile",
			slog.String("user_id", userID),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "更新個人資料失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    profile,
		"message": "個人資料更新成功",
	})
}

// UpdateUserPreferences updates the current user's preferences
func (h *HTTPHandler) UpdateUserPreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	var req UpdatePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "請求格式錯誤")
		return
	}

	profile, err := h.service.UpdatePreferences(ctx, userID, &req)
	if err != nil {
		h.logger.Error("Failed to update preferences",
			slog.String("user_id", userID),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "更新偏好設定失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    profile,
		"message": "偏好設定更新成功",
	})
}

// GetUserStatistics returns comprehensive user statistics
func (h *HTTPHandler) GetUserStatistics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	stats, err := h.service.GetUserStatistics(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get user statistics",
			slog.String("user_id", userID),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "取得統計資料失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    stats,
		"message": "取得統計資料成功",
	})
}

// GetUserActivity returns user activity summary
func (h *HTTPHandler) GetUserActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	activity, err := h.service.GetUserActivitySummary(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get user activity",
			slog.String("user_id", userID),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "取得活動摘要失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    activity,
		"message": "取得活動摘要成功",
	})
}

// GetUserSettings returns user settings and preferences
func (h *HTTPHandler) GetUserSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	settings, err := h.service.GetUserSettings(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get user settings",
			slog.String("user_id", userID),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "取得設定失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    settings,
		"message": "取得設定成功",
	})
}

// ChangePassword changes the current user's password
func (h *HTTPHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "請求格式錯誤")
		return
	}

	// Validate request
	if req.CurrentPassword == "" || req.NewPassword == "" {
		h.writeError(w, http.StatusBadRequest, "目前密碼和新密碼都是必需的")
		return
	}

	if len(req.NewPassword) < 8 {
		h.writeError(w, http.StatusBadRequest, "新密碼長度至少需要 8 個字元")
		return
	}

	err := h.service.ChangePassword(ctx, userID, &req)
	if err != nil {
		if err.Error() == "invalid current password" {
			h.writeError(w, http.StatusBadRequest, "目前密碼錯誤")
			return
		}
		h.logger.Error("Failed to change password",
			slog.String("user_id", userID),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "變更密碼失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "密碼變更成功",
	})
}

// AddFavoriteTool adds a tool to user's favorites
func (h *HTTPHandler) AddFavoriteTool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	toolID := r.PathValue("toolId")
	if toolID == "" {
		h.writeError(w, http.StatusBadRequest, "工具 ID 是必需的")
		return
	}

	err := h.service.AddFavoriteTool(ctx, userID, toolID)
	if err != nil {
		h.logger.Error("Failed to add favorite tool",
			slog.String("user_id", userID),
			slog.String("tool_id", toolID),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "新增收藏工具失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "工具已新增至收藏",
	})
}

// RemoveFavoriteTool removes a tool from user's favorites
func (h *HTTPHandler) RemoveFavoriteTool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	toolID := r.PathValue("toolId")
	if toolID == "" {
		h.writeError(w, http.StatusBadRequest, "工具 ID 是必需的")
		return
	}

	err := h.service.RemoveFavoriteTool(ctx, userID, toolID)
	if err != nil {
		h.logger.Error("Failed to remove favorite tool",
			slog.String("user_id", userID),
			slog.String("tool_id", toolID),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "移除收藏工具失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "工具已從收藏中移除",
	})
}

// RegisterUser creates a new user account (public endpoint)
func (h *HTTPHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "請求格式錯誤")
		return
	}

	// Validate request
	if req.Username == "" || req.Email == "" || req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "使用者名稱、電子郵件和密碼都是必需的")
		return
	}

	if len(req.Password) < 8 {
		h.writeError(w, http.StatusBadRequest, "密碼長度至少需要 8 個字元")
		return
	}

	profile, err := h.service.CreateUser(ctx, &req)
	if err != nil {
		h.logger.Error("Failed to register user",
			slog.String("email", req.Email),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "註冊失敗")
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    profile,
		"message": "註冊成功",
	})
}

// DeactivateAccount deactivates the current user's account
func (h *HTTPHandler) DeactivateAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	err := h.service.DeactivateUser(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to deactivate account",
			slog.String("user_id", userID),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "停用帳戶失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "帳戶已停用",
	})
}

// Helper methods

func (h *HTTPHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", slog.Any("error", err))
	}
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	h.writeJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"error":   http.StatusText(statusCode),
		"message": message,
	})
}

// Pagination helpers
func (h *HTTPHandler) parsePagination(r *http.Request) (limit, offset int) {
	limitStr := r.URL.Query().Get("limit")
	pageStr := r.URL.Query().Get("page")

	limit = 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	page := 1 // default
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	offset = (page - 1) * limit
	return limit, offset
}

func (h *HTTPHandler) paginationResponse(data interface{}, page, limit, total int) map[string]interface{} {
	totalPages := (total + limit - 1) / limit

	return map[string]interface{}{
		"success": true,
		"data":    data,
		"pagination": map[string]interface{}{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": totalPages,
		},
	}
}
