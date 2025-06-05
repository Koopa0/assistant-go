package auth

import (
	"log/slog"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/observability"
)

func TestAuthService_GenerateToken(t *testing.T) {
	// Create test logger
	logger := slog.New(slog.NewTextHandler(nil, nil))

	// Create auth service with test JWT secret
	service := NewAuthService(nil, logger, nil, "test-secret-key")

	// Test cases
	tests := []struct {
		name      string
		userID    string
		email     string
		role      string
		tokenType string
		duration  time.Duration
		wantErr   bool
	}{
		{
			name:      "Valid access token",
			userID:    "user123",
			email:     "test@example.com",
			role:      "user",
			tokenType: "access",
			duration:  1 * time.Hour,
			wantErr:   false,
		},
		{
			name:      "Valid refresh token",
			userID:    "user456",
			email:     "admin@example.com",
			role:      "admin",
			tokenType: "refresh",
			duration:  7 * 24 * time.Hour,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := service.generateToken(tt.userID, tt.email, tt.role, tt.tokenType, tt.duration)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && token == "" {
				t.Error("generateToken() returned empty token")
			}
		})
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	// Create test logger
	logger := slog.New(slog.NewTextHandler(nil, nil))

	// Create auth service
	service := NewAuthService(nil, logger, nil, "test-secret-key")

	// Generate a test token
	token, err := service.generateToken("user123", "test@example.com", "user", "access", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	// Test valid token
	claims, err := service.validateToken(token)
	if err != nil {
		t.Errorf("validateToken() unexpected error: %v", err)
	}
	if claims.UserID != "user123" {
		t.Errorf("validateToken() UserID = %v, want %v", claims.UserID, "user123")
	}
	if claims.Email != "test@example.com" {
		t.Errorf("validateToken() Email = %v, want %v", claims.Email, "test@example.com")
	}

	// Test invalid token
	_, err = service.validateToken("invalid-token")
	if err == nil {
		t.Error("validateToken() expected error for invalid token")
	}
}

// MockQueries implements a minimal mock for sqlc.Queries
type MockQueries struct {
	users map[string]*MockUser
}

type MockUser struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	Role         string
}

func NewMockQueries() *MockQueries {
	return &MockQueries{
		users: make(map[string]*MockUser),
	}
}

func TestAuthService_MockUserExists(t *testing.T) {
	// Create test logger with observability wrapper
	baseLogger := slog.New(slog.NewTextHandler(nil, nil))
	logger := observability.ServerLogger(baseLogger, "auth_test")

	// Create mock metrics
	metrics := &observability.Metrics{}

	// Create auth service
	service := NewAuthService(nil, logger, metrics, "test-secret-key")

	// Test user exists check
	exists := service.mockUserExists("existing@example.com")
	if !exists {
		t.Error("mockUserExists() expected true for existing@example.com")
	}

	exists = service.mockUserExists("new@example.com")
	if exists {
		t.Error("mockUserExists() expected false for new@example.com")
	}
}

func TestAuthService_MockVerifyPassword(t *testing.T) {
	// Create test logger
	logger := observability.ServerLogger(slog.New(slog.NewTextHandler(nil, nil)), "auth_test")

	// Create auth service
	service := NewAuthService(nil, logger, nil, "test-secret-key")

	// Test password verification
	// The mock hash is for "password123"
	validHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

	valid := service.mockVerifyPassword("password123", validHash)
	if !valid {
		t.Error("mockVerifyPassword() expected true for correct password")
	}

	valid = service.mockVerifyPassword("wrongpassword", validHash)
	if valid {
		t.Error("mockVerifyPassword() expected false for wrong password")
	}
}
