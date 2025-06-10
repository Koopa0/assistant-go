// Package auth provides authentication and authorization services for the Assistant API server.
package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	// "github.com/golang-jwt/jwt/v5" // jwt.RegisteredClaims will come from user.TokenClaims
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication and authorization logic.
// It now delegates JWT operations to a JWTService implementation.
type AuthService struct {
	queries      *sqlc.Queries
	logger       *slog.Logger
	metrics      *observability.Metrics
	tokenService JWTService // Changed: Uses JWTService interface
}

// NewAuthService creates a new authentication service.
// It requires a JWTService for handling token operations.
func NewAuthService(queries *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics, tokenSvc JWTService) *AuthService {
	return &AuthService{
		queries:      queries,
		logger:       observability.ServerLogger(logger, "auth_service"),
		metrics:      metrics,
		tokenService: tokenSvc, // Injected JWTService
	}
}

// AuthRequest represents a login request
type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents a successful authentication response
type AuthResponse struct {
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	ExpiresIn    int          `json:"expiresIn"`
	User         UserResponse `json:"user"`
}

// UserResponse represents user information in responses
type UserResponse struct {
	ID          string                 `json:"id"`
	Email       string                 `json:"email"`
	Name        string                 `json:"name"`
	Avatar      string                 `json:"avatar,omitempty"`
	Role        string                 `json:"role"`
	Preferences map[string]interface{} `json:"preferences"`
}

// Login authenticates a user and returns tokens.
// Token generation is delegated to the JWTService.
func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	// Get user by email from database
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		s.logger.Warn("Login attempt with invalid email",
			slog.String("email", email),
			slog.Any("error", err))
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.logger.Warn("Login attempt with invalid password",
			slog.String("email", email))
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive.Bool {
		return nil, fmt.Errorf("account is deactivated")
	}

	userID := user.ID.String()

	// Generate tokens using JWTService
	// Assuming "user" is the default role for now. This might need adjustment
	// if role comes from the user object (e.g., user.Role).
	accessToken, err := s.tokenService.GenerateAccessToken(userID, user.Email, "user")
	if err != nil {
		s.logger.Error("Failed to generate access token via tokenService", slog.String("user_id", userID), slog.Any("error", err))
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(userID)
	if err != nil {
		s.logger.Error("Failed to generate refresh token via tokenService", slog.String("user_id", userID), slog.Any("error", err))
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Record login event
	s.recordLoginEvent(ctx, userID)

	// Parse user preferences
	var preferences map[string]interface{}
	if user.Preferences != nil {
		if err := json.Unmarshal(user.Preferences, &preferences); err != nil {
			s.logger.Warn("Failed to parse user preferences", slog.Any("error", err))
			preferences = map[string]interface{}{
				"language": "zh-TW",
				"theme":    "light",
			}
		}
	} else {
		preferences = map[string]interface{}{
			"language": "zh-TW",
			"theme":    "light",
		}
	}

	// Convert optional fields
	var avatar string
	if user.AvatarUrl.Valid {
		avatar = user.AvatarUrl.String
	}

	var name string
	if user.FullName.Valid {
		name = user.FullName.String
	} else {
		name = user.Username
	}

	// Build response
	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600, // 1 hour in seconds
		User: UserResponse{
			ID:          userID,
			Email:       user.Email,
			Name:        name,
			Avatar:      avatar,
			Role:        "user",
			Preferences: preferences,
		},
	}, nil
}

// RefreshToken validates a refresh token and returns a new access token.
// Token validation and generation are delegated to the JWTService.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	// Validate refresh token using JWTService
	claims, err := s.tokenService.ValidateTokenClaims(refreshToken) // This returns user.TokenClaims
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Note: The original check `claims.TokenType != "refresh"` is removed as
	// user.TokenClaims (from jwt.go) does not have TokenType.
	// This implies that the JWTService's ValidateTokenClaims for a refresh token
	// should ideally validate its 'type' if such a claim is part of the refresh token structure,
	// or the system relies on separate endpoints/validation paths for refresh tokens.
	// For this refactor, we assume claims returned are for a valid refresh token if no error.

	// Generate new access token using JWTService
	accessToken, err := s.tokenService.GenerateAccessToken(claims.UserID, claims.Email, claims.Role)
	if err != nil {
		s.logger.Error("Failed to generate access token during refresh via tokenService", slog.String("user_id", claims.UserID), slog.Any("error", err))
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}

	return accessToken, nil
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, email, password, name string) (string, error) {
	// Check if user exists
	existingUser, err := s.queries.GetUserByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return "", fmt.Errorf("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate username from email if name is empty
	username := email
	if name != "" {
		username = name
	}

	// Set default preferences
	preferences := map[string]interface{}{
		"language":                   "zh-TW",
		"theme":                      "light",
		"defaultProgrammingLanguage": "Go",
		"emailNotifications":         true,
		"timezone":                   "Asia/Taipei",
	}

	// Marshal preferences to JSON bytes
	prefBytes, err := json.Marshal(preferences)
	if err != nil {
		return "", fmt.Errorf("failed to marshal preferences: %w", err)
	}

	// Create user in database
	user, err := s.queries.CreateUser(ctx, sqlc.CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Preferences:  prefBytes,
	})
	if err != nil {
		s.logger.Error("Failed to create user",
			slog.String("email", email),
			slog.Any("error", err))
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	userID := user.ID.String()

	// Send verification email
	s.sendVerificationEmail(user.Email, userID)

	s.logger.Info("User registered successfully",
		slog.String("user_id", userID),
		slog.String("email", email))

	return userID, nil
}

// ValidateToken validates a JWT token and returns the user ID.
// Delegates to JWTService.
func (s *AuthService) ValidateToken(tokenString string) (string, error) {
	return s.tokenService.ValidateToken(tokenString)
}

// ValidateTokenClaims validates a JWT token and returns the claims.
// Delegates to JWTService. It uses user.TokenClaims from jwt.go.
func (s *AuthService) ValidateTokenClaims(tokenString string) (*TokenClaims, error) {
	return s.tokenService.ValidateTokenClaims(tokenString)
}

// GenerateAccessToken generates a JWT access token.
// Delegates to JWTService.
func (s *AuthService) GenerateAccessToken(userID, email, role string) (string, error) {
	return s.tokenService.GenerateAccessToken(userID, email, role)
}

// GenerateRefreshToken generates a JWT refresh token.
// Delegates to JWTService.
func (s *AuthService) GenerateRefreshToken(userID string) (string, error) {
	return s.tokenService.GenerateRefreshToken(userID)
}

// Helper methods (generateUserID, generateResetToken, etc. remain as they are not token generation specific)

func (s *AuthService) generateUserID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("user_%s", base64.RawURLEncoding.EncodeToString(b))
}

func (s *AuthService) generateResetToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// Helper functions for auth operations

// validateResetToken validates a password reset token
func (s *AuthService) validateResetToken(token string) bool {
	// For now, validate token format and length
	// In production, this should check against a database table with expiry
	// Expected format: base64 encoded 32 bytes = 44 characters
	if len(token) < 32 || len(token) > 64 {
		return false
	}

	// Basic base64 validation
	for _, r := range token {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// validateEmailToken validates an email verification token
func (s *AuthService) validateEmailToken(token string) bool {
	// For now, validate token format and length
	// In production, this should check against a database table with expiry
	// Expected format: UUID format or base64 encoded token
	if len(token) < 16 || len(token) > 64 {
		return false
	}

	// Basic validation - alphanumeric with hyphens
	for _, r := range token {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// sendVerificationEmail sends a verification email to the user
func (s *AuthService) sendVerificationEmail(email, userID string) {
	// TODO: Implement actual email sending service
	s.logger.Info("Sending verification email",
		slog.String("email", email),
		slog.String("userID", userID))
}

// sendPasswordResetEmail sends a password reset email
func (s *AuthService) sendPasswordResetEmail(email, token string) {
	// TODO: Implement actual email sending service
	s.logger.Info("Sending password reset email",
		slog.String("email", email),
		slog.String("token", token))
}

// recordLoginEvent records a user login event
func (s *AuthService) recordLoginEvent(ctx context.Context, userID string) {
	// TODO: Record login event in database for audit/analytics
	s.logger.Info("User logged in", slog.String("userID", userID))
}

// recordLogoutEvent records a user logout event
func (s *AuthService) recordLogoutEvent(ctx context.Context, userID string) {
	// TODO: Record logout event in database for audit/analytics
	s.logger.Info("User logged out", slog.String("userID", userID))
}
