package config

import (
	"os"
	"strings"
	"testing"
)

// TestConfigDefaults tests default configuration values
func TestConfigDefaults(t *testing.T) {
	// Clear environment to ensure clean state
	clearTestEnv(t)

	// Set minimal required environment variables
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
	t.Setenv("CLAUDE_API_KEY", "test-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test defaults
	if cfg.Server.Address != ":8080" {
		t.Errorf("Expected default address :8080, got %s", cfg.Server.Address)
	}

	if cfg.Mode != "development" {
		t.Errorf("Expected default mode development, got %s", cfg.Mode)
	}

	if cfg.AI.DefaultProvider != "claude" {
		t.Errorf("Expected default provider claude, got %s", cfg.AI.DefaultProvider)
	}
}

// TestConfigValidation tests configuration validation using table-driven tests
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func(*testing.T)
		expectError bool
		errorSubstr string
	}{
		{
			name: "valid_config",
			setupEnv: func(t *testing.T) {
				clearTestEnv(t)
				t.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
				t.Setenv("CLAUDE_API_KEY", "test-key")
				t.Setenv("DATABASE_MIGRATIONS_PATH", "internal/config/testdata/migrations") // Use test path
			},
			expectError: false,
		},
		{
			name: "missing_database_url",
			setupEnv: func(t *testing.T) {
				clearTestEnv(t)
				t.Setenv("CLAUDE_API_KEY", "test-key")
			},
			expectError: true,
			errorSubstr: "Database.URL is empty",
		},
		{
			name: "missing_ai_key",
			setupEnv: func(t *testing.T) {
				clearTestEnv(t)
				t.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
			},
			expectError: true,
			errorSubstr: "API key",
		},
		{
			name: "invalid_server_address",
			setupEnv: func(t *testing.T) {
				clearTestEnv(t)
				t.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
				t.Setenv("CLAUDE_API_KEY", "test-key")
				t.Setenv("SERVER_ADDRESS", "invalid-address")
			},
			expectError: true,
			errorSubstr: "address",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			// Cannot use t.Parallel() with t.Setenv

			tt.setupEnv(t)

			cfg, err := Load()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorSubstr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if cfg == nil {
					t.Errorf("Expected config but got nil")
				}
			}
		})
	}
}

// TestEnvironmentOverrides tests that environment variables override defaults
func TestEnvironmentOverrides(t *testing.T) {
	clearTestEnv(t)

	// Set test environment variables
	testAddress := ":9090"
	testMode := "production"
	testDBURL := "postgres://prod:prod@prodhost/proddb"
	testAPIKey := "prod-api-key"

	t.Setenv("SERVER_ADDRESS", testAddress)
	t.Setenv("MODE", testMode)
	t.Setenv("DATABASE_URL", testDBURL)
	t.Setenv("CLAUDE_API_KEY", testAPIKey)
	t.Setenv("DATABASE_MIGRATIONS_PATH", "testdata/migrations")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify overrides
	if cfg.Server.Address != testAddress {
		t.Errorf("Expected address %s, got %s", testAddress, cfg.Server.Address)
	}

	if cfg.Mode != testMode {
		t.Errorf("Expected mode %s, got %s", testMode, cfg.Mode)
	}

	if cfg.Database.URL != testDBURL {
		t.Errorf("Expected database URL %s, got %s", testDBURL, cfg.Database.URL)
	}

	if cfg.AI.Claude.APIKey != testAPIKey {
		t.Errorf("Expected API key %s, got %s", testAPIKey, cfg.AI.Claude.APIKey)
	}
}

// TestConfigValidationEdgeCases tests edge cases in configuration validation
func TestConfigValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func(*testing.T)
		expectError bool
	}{
		{
			name: "valid_minimal_config",
			setupEnv: func(t *testing.T) {
				clearTestEnv(t)
				t.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
				t.Setenv("CLAUDE_API_KEY", "test-key")
				t.Setenv("DATABASE_MIGRATIONS_PATH", "internal/config/testdata/migrations")
				// Set valid AI configuration
				t.Setenv("AI_CLAUDE_MAX_TOKENS", "4000")
				t.Setenv("AI_CLAUDE_TEMPERATURE", "0.7")
			},
			expectError: false,
		},
		{
			name: "zero_timeout_values",
			setupEnv: func(t *testing.T) {
				clearTestEnv(t)
				t.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
				t.Setenv("CLAUDE_API_KEY", "test-key")
				t.Setenv("DATABASE_MIGRATIONS_PATH", "internal/config/testdata/migrations")
				t.Setenv("SERVER_READ_TIMEOUT", "0")
				t.Setenv("SERVER_WRITE_TIMEOUT", "0")
				t.Setenv("AI_CLAUDE_MAX_TOKENS", "4000")
			},
			expectError: true, // Zero timeouts should be invalid
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Cannot use t.Parallel() with t.Setenv

			tt.setupEnv(t)

			cfg, err := Load()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected valid config but got error: %v", err)
				}
				if cfg == nil {
					t.Errorf("Expected config but got nil")
				}
			}
		})
	}
}

// TestConfigSensitiveDataHandling tests that sensitive data is handled properly
func TestConfigSensitiveDataHandling(t *testing.T) {
	clearTestEnv(t)

	sensitiveKey := "super-secret-api-key"

	t.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
	t.Setenv("CLAUDE_API_KEY", sensitiveKey)
	t.Setenv("DATABASE_MIGRATIONS_PATH", "testdata/migrations")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify sensitive data is loaded correctly
	if cfg.AI.Claude.APIKey != sensitiveKey {
		t.Errorf("API key not loaded correctly")
	}

	// Test that String() method doesn't expose sensitive data
	configStr := cfg.String()
	if strings.Contains(configStr, sensitiveKey) {
		t.Errorf("Config String() method exposes sensitive API key")
	}
}

// TestConfigYAMLLoading tests loading configuration from YAML files
func TestConfigYAMLLoading(t *testing.T) {
	// This test would require creating test YAML files
	// Skip if no test files exist
	t.Skip("YAML loading tests require test fixtures")
}

// TestTimeoutParsing tests parsing of duration values
func TestTimeoutParsing(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		expectError bool
	}{
		{"valid_seconds", "30s", false},
		{"valid_minutes", "5m", false},
		{"valid_milliseconds", "500ms", false},
		{"invalid_format", "invalid", true},
		{"negative_value", "-10s", true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Cannot use t.Parallel() with t.Setenv

			clearTestEnv(t)
			t.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
			t.Setenv("CLAUDE_API_KEY", "test-key")
			t.Setenv("DATABASE_MIGRATIONS_PATH", "internal/config/testdata/migrations")
			t.Setenv("SERVER_READ_TIMEOUT", tt.value)
			t.Setenv("AI_CLAUDE_MAX_TOKENS", "4000")

			_, err := Load()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for value %q but got none", tt.value)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for value %q: %v", tt.value, err)
				}
			}
		})
	}
}

// Benchmark configuration loading performance
func BenchmarkConfigLoad(b *testing.B) {
	// Setup environment once
	os.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
	os.Setenv("CLAUDE_API_KEY", "test-key")
	os.Setenv("DATABASE_MIGRATIONS_PATH", "testdata/migrations")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("CLAUDE_API_KEY")
		os.Unsetenv("DATABASE_MIGRATIONS_PATH")
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Load()
		if err != nil {
			b.Fatalf("Config load failed: %v", err)
		}
	}
}

// TestValidateAIConfiguration tests AI configuration validation
func TestValidateAIConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		setupAI     func() *AIConfig
		expectErr   bool
		errContains string
	}{
		{
			name: "valid_claude_config",
			setupAI: func() *AIConfig {
				return &AIConfig{
					DefaultProvider: "claude",
					Claude: Claude{
						APIKey:      "test-key",
						Model:       "claude-3-sonnet-20240229",
						MaxTokens:   4000,
						Temperature: 0.7,
						BaseURL:     "https://api.anthropic.com",
					},
				}
			},
			expectErr: false,
		},
		{
			name: "missing_api_key",
			setupAI: func() *AIConfig {
				return &AIConfig{
					DefaultProvider: "claude",
					Claude: Claude{
						Model:       "claude-3-sonnet-20240229",
						MaxTokens:   4000,
						Temperature: 0.7,
						BaseURL:     "https://api.anthropic.com",
					},
				}
			},
			expectErr:   true,
			errContains: "API key",
		},
		{
			name: "invalid_temperature",
			setupAI: func() *AIConfig {
				return &AIConfig{
					DefaultProvider: "claude",
					Claude: Claude{
						APIKey:      "test-key",
						Model:       "claude-3-sonnet-20240229",
						MaxTokens:   4000,
						Temperature: 2.1, // Invalid: > 2.0
						BaseURL:     "https://api.anthropic.com",
					},
				}
			},
			expectErr:   true,
			errContains: "temperature",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Cannot use t.Parallel() with t.Setenv

			config := &Config{AI: *tt.setupAI()}
			err := config.AI.Validate()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got: %v", tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper function to clear test environment
func clearTestEnv(t *testing.T) {
	t.Helper()

	// Clear commonly used environment variables
	envVars := []string{
		"SERVER_ADDRESS", "MODE", "DATABASE_URL", "CLAUDE_API_KEY", "GEMINI_API_KEY",
		"DATABASE_MIGRATIONS_PATH", "SERVER_READ_TIMEOUT", "SERVER_WRITE_TIMEOUT",
		"AI_CLAUDE_MAX_TOKENS", "AI_CLAUDE_TEMPERATURE", "AI_GEMINI_MAX_TOKENS",
	}

	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

// Example showing how to create a test configuration
func ExampleConfig() {
	// Set required environment variables
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost/testdb")
	os.Setenv("CLAUDE_API_KEY", "your-api-key")

	// Load configuration
	cfg, err := Load()
	if err != nil {
		panic(err)
	}

	// Use configuration
	_ = cfg.Server.Address // ":8080"
	_ = cfg.Mode           // "development"

	// Clean up
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("CLAUDE_API_KEY")
}
