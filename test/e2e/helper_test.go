package e2e

import (
	"os"
	"testing"
)

// getTestEnv returns environment variables needed for testing
func getTestEnv(t *testing.T) []string {
	t.Helper()

	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		t.Skip("CLAUDE_API_KEY not set")
	}

	// Get database URL from environment or use a test default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://test:test@localhost:5432/test?sslmode=disable"
	}

	return append(os.Environ(),
		"CLAUDE_API_KEY="+apiKey,
		"DATABASE_URL="+dbURL,
		"GEMINI_API_KEY=AIzaSyABCDEFGHIJKLMNOPQRSTUVWXYZ1234567",
		"JWT_SECRET=test-secret-key-for-testing-only",
		"LOG_LEVEL=error", // Reduce noise in tests
	)
}

// getConfigPath returns the path to the config file from test directory
func getConfigPath() string {
	return "../../configs/development.yaml"
}
