package users

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AuthHTTPHandler handles HTTP requests for authentication
type AuthHTTPHandler struct {
	service *AuthService
}

// NewAuthHTTPHandler creates a new HTTP handler for authentication
func NewAuthHTTPHandler(service *AuthService) *AuthHTTPHandler {
	return &AuthHTTPHandler{
		service: service,
	}
}

// RegisterRoutes registers all authentication routes
func (h *AuthHTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/login", h.HandleLogin)
	mux.HandleFunc("POST /auth/refresh", h.HandleRefresh)
	mux.HandleFunc("POST /auth/logout", h.HandleLogout)
	mux.HandleFunc("POST /auth/register", h.HandleRegister)
	mux.HandleFunc("POST /auth/forgot-password", h.HandleForgotPassword)
	mux.HandleFunc("POST /auth/reset-password", h.HandleResetPassword)
	mux.HandleFunc("GET /auth/verify-email", h.HandleVerifyEmail)
}

// HandleLogin handles user login
func (h *AuthHTTPHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAuthError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		h.writeAuthError(w, "INVALID_REQUEST", "電子郵件和密碼為必填", http.StatusBadRequest)
		return
	}

	// Authenticate user
	response, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.writeAuthError(w, "UNAUTHORIZED", "電子郵件或密碼錯誤", http.StatusUnauthorized)
		return
	}

	h.writeSuccess(w, response, "登入成功")
}

// HandleRefresh handles token refresh
func (h *AuthHTTPHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	// Extract refresh token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.writeAuthError(w, "UNAUTHORIZED", "缺少授權標頭", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		h.writeAuthError(w, "UNAUTHORIZED", "無效的授權格式", http.StatusUnauthorized)
		return
	}

	// Refresh token
	accessToken, err := h.service.RefreshToken(r.Context(), tokenString)
	if err != nil {
		h.writeAuthError(w, "UNAUTHORIZED", "無效的更新權杖", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"accessToken": accessToken,
		"expiresIn":   3600,
	}

	h.writeSuccess(w, response, "權杖更新成功")
}

// HandleLogout handles user logout
func (h *AuthHTTPHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user ID from context (set by auth middleware)
	userID, ok := ctx.Value("userID").(string)
	if !ok {
		h.writeAuthError(w, "UNAUTHORIZED", "未授權", http.StatusUnauthorized)
		return
	}

	// Record logout event
	h.service.recordLogoutEvent(ctx, userID)

	h.writeSuccess(w, nil, "登出成功")
}

// HandleRegister handles user registration
func (h *AuthHTTPHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAuthError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.Name == "" {
		h.writeAuthError(w, "INVALID_REQUEST", "所有欄位皆為必填", http.StatusBadRequest)
		return
	}

	// Check password strength
	if len(req.Password) < 8 {
		h.writeAuthError(w, "INVALID_REQUEST", "密碼長度至少需要 8 個字元", http.StatusBadRequest)
		return
	}

	// Register user
	userID, err := h.service.Register(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		if err.Error() == "email already registered" {
			h.writeAuthError(w, "INVALID_REQUEST", "此電子郵件已被註冊", http.StatusBadRequest)
			return
		}
		h.writeAuthError(w, "SERVER_ERROR", "註冊失敗", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "註冊成功，請檢查您的電子郵件以驗證帳號",
		"userId":  userID,
	}

	h.writeSuccess(w, response, "註冊成功")
}

// HandleForgotPassword handles password reset requests
func (h *AuthHTTPHandler) HandleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAuthError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		h.writeAuthError(w, "INVALID_REQUEST", "電子郵件為必填", http.StatusBadRequest)
		return
	}

	// Always return success to prevent email enumeration
	// Check if user exists and send reset email if they do
	if _, err := h.service.queries.GetUserByEmail(r.Context(), req.Email); err == nil {
		resetToken := h.service.generateResetToken()
		h.service.sendPasswordResetEmail(req.Email, resetToken)
	}

	h.writeSuccess(w, nil, "如果此電子郵件已註冊，您將收到密碼重設連結")
}

// HandleResetPassword handles password reset
func (h *AuthHTTPHandler) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAuthError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		h.writeAuthError(w, "INVALID_REQUEST", "權杖和新密碼為必填", http.StatusBadRequest)
		return
	}

	// Validate reset token
	if !h.service.validateResetToken(req.Token) {
		h.writeAuthError(w, "INVALID_REQUEST", "無效或過期的重設權杖", http.StatusBadRequest)
		return
	}

	// Validate new password
	if len(req.NewPassword) < 8 {
		h.writeAuthError(w, "INVALID_REQUEST", "密碼長度至少需要 8 個字元", http.StatusBadRequest)
		return
	}

	// Update password (mock implementation)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		h.writeAuthError(w, "SERVER_ERROR", "密碼重設失敗", http.StatusInternalServerError)
		return
	}

	// Mock update password
	h.service.logger.Info("Password reset successful", slog.String("hashedPassword", string(hashedPassword)))

	h.writeSuccess(w, nil, "密碼重設成功")
}

// HandleVerifyEmail handles email verification
func (h *AuthHTTPHandler) HandleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		h.writeAuthError(w, "INVALID_REQUEST", "缺少驗證權杖", http.StatusBadRequest)
		return
	}

	// Verify token
	if !h.service.validateEmailToken(token) {
		h.writeAuthError(w, "INVALID_REQUEST", "無效或過期的驗證權杖", http.StatusBadRequest)
		return
	}

	h.writeSuccess(w, nil, "電子郵件驗證成功")
}

// Response helpers

func (h *AuthHTTPHandler) writeSuccess(w http.ResponseWriter, data interface{}, message string) {
	response := map[string]interface{}{
		"success":   true,
		"data":      data,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHTTPHandler) writeAuthError(w http.ResponseWriter, code, message string, status int) {
	response := map[string]interface{}{
		"success":   false,
		"error":     code,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}
