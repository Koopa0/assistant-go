// Package auth provides authentication and authorization services for the Assistant API server.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/koopa0/assistant-go/internal/observability"
	"github.com/koopa0/assistant-go/internal/storage/postgres/sqlc"
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
	// Get user by email (in a real implementation)
	user := s.mockGetUserByEmail(ctx, email)
	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password
	if !s.mockVerifyPassword(password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate tokens
	accessToken, err := s.generateToken(user.ID, user.Email, user.Role, "access", 1*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateToken(user.ID, user.Email, user.Role, "refresh", 7*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Record login event
	s.recordLoginEvent(ctx, user.ID)

	// Build response
	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600, // 1 hour in seconds
		User: UserResponse{
			ID:     user.ID,
			Email:  user.Email,
			Name:   user.Name,
			Avatar: user.Avatar,
			Role:   user.Role,
			Preferences: map[string]interface{}{
				"language": "zh-TW",
				"theme":    "light",
			},
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
	if s.mockUserExists(email) {
		return "", fmt.Errorf("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	userID := s.generateUserID()
	user := &mockUser{
		ID:           userID,
		Email:        email,
		Name:         name,
		PasswordHash: string(hashedPassword),
		Role:         "user",
		CreatedAt:    time.Now(),
	}

	// Send verification email
	s.sendVerificationEmail(user.Email, userID)

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

// Mock implementations (replace with real database queries)

type mockUser struct {
	ID           string
	Email        string
	Name         string
	Avatar       string
	Role         string
	PasswordHash string
	CreatedAt    time.Time
}

func (s *AuthService) mockGetUserByEmail(ctx context.Context, email string) *mockUser {
	// In real implementation, query database
	if email == "test@example.com" {
		return &mockUser{
			ID:           "user_123",
			Email:        "test@example.com",
			Name:         "測試使用者",
			Avatar:       "https://example.com/avatar.jpg",
			Role:         "user",
			PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // "password123"
		}
	}
	return nil
}

func (s *AuthService) mockVerifyPassword(password, hash string) bool {
	// In real implementation, use bcrypt.CompareHashAndPassword
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *AuthService) mockUserExists(email string) bool {
	// In real implementation, query database
	return email == "existing@example.com"
}

func (s *AuthService) validateResetToken(token string) bool {
	// In real implementation, check token in database with expiry
	return len(token) > 20
}

func (s *AuthService) validateEmailToken(token string) bool {
	// In real implementation, check token in database
	return len(token) > 10
}

func (s *AuthService) sendVerificationEmail(email, userID string) {
	// In real implementation, send actual email
	s.logger.Info("Sending verification email",
		slog.String("email", email),
		slog.String("userID", userID))
}

func (s *AuthService) sendPasswordResetEmail(email, token string) {
	// In real implementation, send actual email
	s.logger.Info("Sending password reset email",
		slog.String("email", email),
		slog.String("token", token))
}

func (s *AuthService) recordLoginEvent(ctx context.Context, userID string) {
	// In real implementation, record to database
	s.logger.Info("User logged in", slog.String("userID", userID))
}

func (s *AuthService) recordLogoutEvent(ctx context.Context, userID string) {
	// In real implementation, record to database
	s.logger.Info("User logged out", slog.String("userID", userID))
}
