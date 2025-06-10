package user

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/server/handlers"
)

// HTTPHandler handles HTTP requests for user operations
type HTTPHandler struct {
	*handlers.Handler
	service *Service
}

// NewHTTPHandler creates a new HTTP handler for users
func NewHTTPHandler(service *Service, logger *slog.Logger) *HTTPHandler {
	loggerWithName := observability.ServerLogger(logger, "users_http")
	return &HTTPHandler{
		Handler: handlers.NewHandler(loggerWithName),
		service: service,
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
	h.LogRequest(r, "users.get_profile")
	ctx := r.Context()

	// Get user ID from auth context
	userID, err := h.GetUserID(ctx)
	if err != nil {
		h.WriteUnauthorized(w, "User not authenticated")
		return
	}

	profile, err := h.service.GetUserByID(ctx, userID)
	if err != nil {
		h.LogError(r, "users.get_profile", err)
		h.WriteInternalError(w, err)
		return
	}

	h.WriteSuccess(w, profile, "User profile retrieved successfully")
}

// UpdateUserProfile updates the current user's profile
func (h *HTTPHandler) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := h.GetUserID(ctx)
	if err != nil {
		h.WriteUnauthorized(w, "使用者未認證")
		return
	}

	var req UpdateProfileRequest
	if err := h.DecodeJSON(r, &req); err != nil {
		h.WriteBadRequest(w, "請求格式錯誤", err.Error())
		return
	}

	profile, err := h.service.UpdateProfile(ctx, userID, &req)
	if err != nil {
		h.LogError(r, "users.update_profile", err)
		h.WriteInternalError(w, err)
		return
	}

	h.WriteSuccess(w, profile, "個人資料更新成功")
}

// UpdateUserPreferences updates the current user's preferences
func (h *HTTPHandler) UpdateUserPreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := h.GetUserID(ctx)
	if err != nil {
		h.WriteUnauthorized(w, "使用者未認證")
		return
	}

	var req UpdatePreferencesRequest
	if err := h.DecodeJSON(r, &req); err != nil {
		h.WriteBadRequest(w, "請求格式錯誤", err.Error())
		return
	}

	profile, err := h.service.UpdatePreferences(ctx, userID, &req)
	if err != nil {
		h.LogError(r, "users.update_preferences", err)
		h.WriteInternalError(w, err)
		return
	}

	h.WriteSuccess(w, profile, "偏好設定更新成功")
}

// GetUserStatistics returns comprehensive user statistics
func (h *HTTPHandler) GetUserStatistics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := h.GetUserID(ctx)
	if err != nil {
		h.WriteUnauthorized(w, "使用者未認證")
		return
	}

	stats, err := h.service.GetUserStatistics(ctx, userID)
	if err != nil {
		h.LogError(r, "users.get_statistics", err)
		h.WriteInternalError(w, err)
		return
	}

	h.WriteSuccess(w, stats, "取得統計資料成功")
}

// GetUserActivity returns user activity summary
func (h *HTTPHandler) GetUserActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := h.GetUserID(ctx)
	if err != nil {
		h.WriteUnauthorized(w, "使用者未認證")
		return
	}

	activity, err := h.service.GetUserActivitySummary(ctx, userID)
	if err != nil {
		h.LogError(r, "users.get_activity", err)
		h.WriteInternalError(w, err)
		return
	}

	h.WriteSuccess(w, activity, "取得活動摘要成功")
}

// GetUserSettings returns user settings and preferences
func (h *HTTPHandler) GetUserSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := h.GetUserID(ctx)
	if err != nil {
		h.WriteUnauthorized(w, "使用者未認證")
		return
	}

	settings, err := h.service.GetUserSettings(ctx, userID)
	if err != nil {
		h.LogError(r, "users.get_settings", err)
		h.WriteInternalError(w, err)
		return
	}

	h.WriteSuccess(w, settings, "取得設定成功")
}

// ChangePassword changes the current user's password
func (h *HTTPHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := h.GetUserID(ctx)
	if err != nil {
		h.WriteUnauthorized(w, "使用者未認證")
		return
	}

	var req ChangePasswordRequest
	if err := h.DecodeJSON(r, &req); err != nil {
		h.WriteBadRequest(w, "請求格式錯誤", err.Error())
		return
	}

	// Validate request
	if req.CurrentPassword == "" || req.NewPassword == "" {
		h.WriteBadRequest(w, "目前密碼和新密碼都是必需的")
		return
	}

	if len(req.NewPassword) < 8 {
		h.WriteBadRequest(w, "新密碼長度至少需要 8 個字元")
		return
	}

	err = h.service.ChangePassword(ctx, userID, &req)
	if err != nil {
		if err.Error() == "invalid current password" {
			h.WriteBadRequest(w, "目前密碼錯誤")
			return
		}
		h.LogError(r, "users.change_password", err)
		h.WriteInternalError(w, err)
		return
	}

	h.WriteSuccess(w, nil, "密碼變更成功")
}

// AddFavoriteTool adds a tool to user's favorites
func (h *HTTPHandler) AddFavoriteTool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := h.GetUserID(ctx)
	if err != nil {
		h.WriteUnauthorized(w, "使用者未認證")
		return
	}

	toolID := r.PathValue("toolId")
	if toolID == "" {
		h.WriteBadRequest(w, "工具 ID 是必需的")
		return
	}

	err = h.service.AddFavoriteTool(ctx, userID, toolID)
	if err != nil {
		h.LogError(r, "users.add_favorite_tool", err)
		h.WriteInternalError(w, err)
		return
	}

	h.WriteSuccess(w, nil, "工具已新增至收藏")
}

// RemoveFavoriteTool removes a tool from user's favorites
func (h *HTTPHandler) RemoveFavoriteTool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := h.GetUserID(ctx)
	if err != nil {
		h.WriteUnauthorized(w, "使用者未認證")
		return
	}

	toolID := r.PathValue("toolId")
	if toolID == "" {
		h.WriteBadRequest(w, "工具 ID 是必需的")
		return
	}

	err = h.service.RemoveFavoriteTool(ctx, userID, toolID)
	if err != nil {
		h.LogError(r, "users.remove_favorite_tool", err)
		h.WriteInternalError(w, err)
		return
	}

	h.WriteSuccess(w, nil, "工具已從收藏中移除")
}

// RegisterUser creates a new user account (public endpoint)
func (h *HTTPHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateUserRequest
	if err := h.DecodeJSON(r, &req); err != nil {
		h.WriteBadRequest(w, "請求格式錯誤", err.Error())
		return
	}

	// Validate request
	if req.Username == "" || req.Email == "" || req.Password == "" {
		h.WriteBadRequest(w, "使用者名稱、電子郵件和密碼都是必需的")
		return
	}

	if len(req.Password) < 8 {
		h.WriteBadRequest(w, "密碼長度至少需要 8 個字元")
		return
	}

	profile, err := h.service.CreateUser(ctx, &req)
	if err != nil {
		h.LogError(r, "users.register", err)
		h.WriteInternalError(w, err)
		return
	}

	// Use WriteError instead of WriteSuccess for custom status code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    profile,
		"message": "註冊成功",
	})
}

// DeactivateAccount deactivates the current user's account
func (h *HTTPHandler) DeactivateAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := h.GetUserID(ctx)
	if err != nil {
		h.WriteUnauthorized(w, "使用者未認證")
		return
	}

	err = h.service.DeactivateUser(ctx, userID)
	if err != nil {
		h.LogError(r, "users.deactivate_account", err)
		h.WriteInternalError(w, err)
		return
	}

	h.WriteSuccess(w, nil, "帳戶已停用")
}
