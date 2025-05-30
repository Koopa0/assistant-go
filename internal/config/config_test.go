package config

import (
	"os"
	"testing"
	"time"
)

// TestConfigDefaults tests that default configuration values are set correctly
func TestConfigDefaults(t *testing.T) {
	// Clear environment variables that might affect the test
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test default values
	if cfg.Mode == "" {
		t.Errorf("Expected default mode to be set")
	}

	if cfg.Server.Address == "" {
		t.Errorf("Expected default server address to be set")
	}

	if cfg.Server.ReadTimeout == 0 {
		t.Errorf("Expected default read timeout to be set")
	}

	if cfg.Server.WriteTimeout == 0 {
		t.Errorf("Expected default write timeout to be set")
	}

	if cfg.AI.DefaultProvider == "" {
		t.Errorf("Expected default AI provider to be set")
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_config",
			setupEnv: func() {
				os.Setenv("SERVER_ADDRESS", ":8080")
				os.Setenv("DATABASE_URL", "postgres://localhost/test")
				os.Setenv("DB_MIGRATIONS_PATH", "")
			},
			expectError: false,
		},
		{
			name: "invalid_port",
			setupEnv: func() {
				os.Setenv("SERVER_ADDRESS", "invalid")
			},
			expectError: true,
		},
		{
			name: "negative_port",
			setupEnv: func() {
				os.Setenv("SERVER_ADDRESS", ":-1")
			},
			expectError: true,
		},
		{
			name: "port_too_high",
			setupEnv: func() {
				os.Setenv("SERVER_ADDRESS", ":99999")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment before each test
			os.Clearenv()

			// Setup test environment
			tt.setupEnv()

			cfg, err := Load()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if cfg == nil {
					t.Errorf("Expected config but got nil")
				}
			}

			// Clean up
			os.Clearenv()
		})
	}
}

// TestEnvironmentOverrides tests that environment variables override defaults
func TestEnvironmentOverrides(t *testing.T) {
	// Clear environment variables
	os.Clearenv()

	// Set test environment variables
	testAddress := ":9090"
	testMode := "production"

	os.Setenv("SERVER_ADDRESS", testAddress)
	os.Setenv("APP_MODE", testMode)
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("DB_MIGRATIONS_PATH", "")

	defer os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Server.Address != testAddress {
		t.Errorf("Expected address %s, got %s", testAddress, cfg.Server.Address)
	}

	if cfg.Mode != testMode {
		t.Errorf("Expected mode %s, got %s", testMode, cfg.Mode)
	}
}

// TestConfigValidationEdgeCases tests edge cases in configuration validation
func TestConfigValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "valid_minimal_config",
			config: &Config{
				Mode: "development",
				Server: ServerConfig{
					Address:      ":8080",
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  60 * time.Second,
				},
				Database: DatabaseConfig{
					URL:            "postgres://localhost/test",
					MaxConnections: 25,
					MinConnections: 5,
					MigrationsPath: "",
				},
				AI: AIConfig{
					DefaultProvider: "claude",
					Claude: Claude{
						APIKey: "test-key",
					},
				},
			},
			expected: true,
		},
		{
			name: "missing_database_url",
			config: &Config{
				Mode: "development",
				Server: ServerConfig{
					Address: ":8080",
				},
				Database: DatabaseConfig{
					URL: "", // Missing URL
				},
			},
			expected: false,
		},
		{
			name: "invalid_mode",
			config: &Config{
				Mode: "invalid_mode",
				Server: ServerConfig{
					Address: ":8080",
				},
				Database: DatabaseConfig{
					URL:            "postgres://localhost/test",
					MaxConnections: 25,
					MinConnections: 5,
					MigrationsPath: "",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.config)

			if tt.expected && err != nil {
				t.Errorf("Expected valid config but got error: %v", err)
			}

			if !tt.expected && err == nil {
				t.Errorf("Expected validation error but got none")
			}
		})
	}
}

// TestConfigSensitiveDataHandling tests that sensitive data is handled properly
func TestConfigSensitiveDataHandling(t *testing.T) {
	os.Clearenv()

	// Set sensitive environment variables
	os.Setenv("DATABASE_URL", "postgres://user:password@localhost/db")
	os.Setenv("CLAUDE_API_KEY", "secret-key-123")
	os.Setenv("GEMINI_API_KEY", "another-secret-key")
	os.Setenv("DB_MIGRATIONS_PATH", "")

	defer os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify sensitive data is loaded but not logged
	if cfg.Database.URL == "" {
		t.Errorf("Expected database URL to be loaded")
	}

	if cfg.AI.Claude.APIKey == "" {
		t.Errorf("Expected Claude API key to be loaded")
	}

	if cfg.AI.Gemini.APIKey == "" {
		t.Errorf("Expected Gemini API key to be loaded")
	}

	// Note: Config doesn't implement String() method for security reasons
	// This prevents accidental logging of sensitive data
}
