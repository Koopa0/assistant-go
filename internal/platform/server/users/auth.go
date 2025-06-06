// Package auth provides authentication and authorization services for the Assistant API server.
package users

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication and authorization logic
type AuthService struct {
	queries   *sqlc.Queries
	logger    *slog.Logger
	metrics   *observability.Metrics
	jwtSecret []byte
	jwtIssuer string
}

// NewAuthService creates a new authentication service
func NewAuthService(queries *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics, jwtSecret string) *AuthService {
	return &AuthService{
		queries:   queries,
		logger:    observability.ServerLogger(logger, "auth_service"),
		metrics:   metrics,
		jwtSecret: []byte(jwtSecret),
		jwtIssuer: "assistant-api",
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

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// Login authenticates a user and returns tokens
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

	// Generate tokens
	accessToken, err := s.generateToken(userID, user.Email, "user", "access", 1*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateToken(userID, user.Email, "user", "refresh", 7*24*time.Hour)
	if err != nil {
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

// RefreshToken validates a refresh token and returns a new access token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	// Validate refresh token
	claims, err := s.validateToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.TokenType != "refresh" {
		return "", fmt.Errorf("invalid token type")
	}

	// Generate new access token
	accessToken, err := s.generateToken(claims.UserID, claims.Email, claims.Role, "access", 1*time.Hour)
	if err != nil {
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

// ValidateToken validates a JWT token and returns the claims
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	return s.validateToken(tokenString)
}

// Helper methods

func (s *AuthService) generateToken(userID, email, role, tokenType string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.jwtIssuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) validateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

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
	// TODO: Implement proper token validation with database and expiry
	return len(token) > 20
}

// validateEmailToken validates an email verification token
func (s *AuthService) validateEmailToken(token string) bool {
	// TODO: Implement proper token validation with database
	return len(token) > 10
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
