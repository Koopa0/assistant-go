package user

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService defines the interface for JWT operations
type JWTService interface {
	GenerateAccessToken(userID, email, role string) (string, error)
	GenerateRefreshToken(userID string) (string, error)
	ValidateToken(tokenString string) (string, error) // Returns userID
	ValidateTokenClaims(tokenString string) (*TokenClaims, error)
}

// TokenClaims represents the claims in a JWT token
type TokenClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// TokenService handles JWT token operations
type TokenService struct {
	secretKey []byte
	issuer    string
	logger    *slog.Logger
}

// NewTokenService creates a new token service
func NewTokenService(secretKey string, issuer string, logger *slog.Logger) *TokenService {
	return &TokenService{
		secretKey: []byte(secretKey),
		issuer:    issuer,
		logger:    logger,
	}
}

// GenerateAccessToken creates a new JWT access token
func (s *TokenService) GenerateAccessToken(userID, email, role string) (string, error) {
	claims := TokenClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// GenerateRefreshToken creates a new refresh token
func (s *TokenService) GenerateRefreshToken(userID string) (string, error) {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)), // 30 days
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    s.issuer,
		Subject:   userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// ValidateToken validates a JWT token and returns the user ID
func (s *TokenService) ValidateToken(tokenString string) (string, error) {
	claims, err := s.ValidateTokenClaims(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

// ValidateTokenClaims validates a JWT token and returns the claims
func (s *TokenService) ValidateTokenClaims(tokenString string) (*TokenClaims, error) {
	// Remove Bearer prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// GenerateSecureToken generates a secure random token
func (s *TokenService) GenerateSecureToken() string {
	// Generate 32 random bytes
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based token
		timestamp := time.Now().Unix()
		data := fmt.Sprintf("%d:%s", timestamp, s.issuer)

		h := hmac.New(sha256.New, s.secretKey)
		h.Write([]byte(data))

		return base64.URLEncoding.EncodeToString(h.Sum(nil))
	}

	return base64.URLEncoding.EncodeToString(b)
}

// ValidateRefreshToken validates a refresh token
func (s *TokenService) ValidateRefreshToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secretKey, nil
	})

	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("user ID not found in token")
	}

	return userID, nil
}

// TokenResponse contains tokens
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
