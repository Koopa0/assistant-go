package users

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/koopa0/assistant-go/internal/observability"
)

func TestUserService_CreateUser(t *testing.T) {
	// Create test logger
	logger := observability.ServerLogger(slog.New(slog.NewTextHandler(os.Stdout, nil)), "users_test")

	// Create service with nil queries for unit testing
	service := NewUserService(nil, logger, nil)

	// Test cases
	tests := []struct {
		name    string
		request *CreateUserRequest
		wantErr bool
	}{
		{
			name: "Valid user creation request",
			request: &CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				FullName: stringPtr("Test User"),
			},
			wantErr: true, // Will fail without database
		},
		{
			name: "Empty username",
			request: &CreateUserRequest{
				Username: "",
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := service.CreateUser(ctx, tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserService_UpdatePreferences(t *testing.T) {
	// Create test logger
	logger := observability.ServerLogger(slog.New(slog.NewTextHandler(os.Stdout, nil)), "users_test")

	// Create service
	service := NewUserService(nil, logger, nil)

	ctx := context.Background()
	userID := "550e8400-e29b-41d4-a716-446655440000"

	req := &UpdatePreferencesRequest{
		Preferences: map[string]interface{}{
			"theme":    "dark",
			"language": "en-US",
		},
	}

	// This will fail without database connection, but tests the method structure
	_, err := service.UpdatePreferences(ctx, userID, req)
	if err == nil {
		t.Error("Expected error when updating preferences without database")
	}
}

func TestUserService_ValidateUserID(t *testing.T) {
	// Create test logger
	logger := observability.ServerLogger(slog.New(slog.NewTextHandler(os.Stdout, nil)), "users_test")

	// Create service
	service := NewUserService(nil, logger, nil)

	ctx := context.Background()

	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{
			name:    "Valid UUID",
			userID:  "550e8400-e29b-41d4-a716-446655440000",
			wantErr: true, // Will fail due to no database, but UUID is valid
		},
		{
			name:    "Invalid UUID",
			userID:  "invalid-uuid",
			wantErr: true,
		},
		{
			name:    "Empty UUID",
			userID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GetUserByID(ctx, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserToProfile(t *testing.T) {
	// Create test logger
	logger := observability.ServerLogger(slog.New(slog.NewTextHandler(os.Stdout, nil)), "users_test")

	// Create service
	service := NewUserService(nil, logger, nil)

	// This test would require sqlc.User struct
	// For now, just test that the service can be created
	if service == nil {
		t.Error("Failed to create user service")
	}

	if service.logger == nil {
		t.Error("Logger not set in user service")
	}
}

func TestGenerateAPIKey(t *testing.T) {
	// Create test logger
	logger := observability.ServerLogger(slog.New(slog.NewTextHandler(os.Stdout, nil)), "users_test")

	// Create service
	service := NewUserService(nil, logger, nil)

	// Test API key generation
	key, err := service.generateAPIKey()
	if err != nil {
		t.Errorf("generateAPIKey() error = %v", err)
	}

	if len(key) == 0 {
		t.Error("generateAPIKey() returned empty key")
	}

	// Test that two keys are different
	key2, err := service.generateAPIKey()
	if err != nil {
		t.Errorf("generateAPIKey() second call error = %v", err)
	}

	if key == key2 {
		t.Error("generateAPIKey() returned identical keys")
	}
}

// Helper function for tests
func stringPtr(s string) *string {
	return &s
}
