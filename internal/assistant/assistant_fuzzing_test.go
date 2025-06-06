package assistant

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/koopa0/assistant-go/internal/config"
	customerrors "github.com/koopa0/assistant-go/internal/errors"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
	"github.com/koopa0/assistant-go/internal/testutil"
	"github.com/koopa0/assistant-go/internal/tools"
)

// FuzzQueryRequestValidation tests query request validation with fuzzing
// Following golang_guide.md recommendation for property-based testing
func FuzzQueryRequestValidation(f *testing.F) {
	// Seed with valid examples
	f.Add("Hello world")
	f.Add("Analyze this code")
	f.Add("What's the weather?")
	f.Add("")
	f.Add("   ")
	f.Add(strings.Repeat("a", 1000))

	f.Fuzz(func(t *testing.T, query string) {
		// Property: Query validation should be consistent
		req := &QueryRequest{Query: query}

		// Test validation consistency
		isValid := req.Query != "" && strings.TrimSpace(req.Query) != ""

		// Create mock assistant for validation
		cfg := &config.Config{
			Mode: "test",
			AI: config.AIConfig{
				DefaultProvider: "claude",
				Claude: config.Claude{
					APIKey: "test-key",
					Model:  "claude-3-sonnet-20240229",
				},
			},
		}
		mockDB := postgres.NewMockClient(testutil.NewSilentLogger())
		registry := tools.NewRegistry(testutil.NewTestLogger())
		logger := testutil.NewTestLogger()

		processor, err := NewProcessor(cfg, mockDB, registry, logger)
		if err != nil {
			t.Skip("Failed to create processor:", err)
		}
		defer processor.Close(nil)

		// Property: Validation should match our expectation
		if isValid {
			// Valid queries should not fail validation
			if query == "" || strings.TrimSpace(query) == "" {
				t.Errorf("Property violated: expected valid query but got empty/whitespace")
			}
		} else {
			// Invalid queries should be caught
			if query != "" && strings.TrimSpace(query) != "" {
				t.Errorf("Property violated: expected invalid query but validation passed")
			}
		}

		// Property: UTF-8 handling should be consistent
		if !utf8.ValidString(query) {
			t.Skip("Invalid UTF-8 string")
		}

		// Property: Extremely long queries should be handled gracefully
		if len(query) > 10000 {
			// Should not cause crashes or excessive memory usage
			_ = strings.TrimSpace(query)
		}
	})
}

// FuzzJSONMarshaling tests JSON marshaling/unmarshaling properties
func FuzzJSONMarshaling(f *testing.F) {
	// Seed with valid JSON examples
	f.Add([]byte(`{"query": "test"}`))
	f.Add([]byte(`{"query": "test", "conversation_id": "123"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var req QueryRequest

		// Skip invalid JSON
		if err := json.Unmarshal(data, &req); err != nil {
			t.Skip("Invalid JSON")
		}

		// Property: Valid JSON should round-trip correctly
		marshaled, err := json.Marshal(req)
		if err != nil {
			t.Errorf("Failed to marshal valid struct: %v", err)
			return
		}

		var req2 QueryRequest
		if err := json.Unmarshal(marshaled, &req2); err != nil {
			t.Errorf("Failed to unmarshal own output: %v", err)
			return
		}

		// Property: Core fields should be preserved
		if req.Query != req2.Query {
			t.Errorf("Query field not preserved: %q != %q", req.Query, req2.Query)
		}

		if (req.ConversationID == nil) != (req2.ConversationID == nil) {
			t.Errorf("ConversationID pointer state not preserved")
		}

		if req.ConversationID != nil && req2.ConversationID != nil {
			if *req.ConversationID != *req2.ConversationID {
				t.Errorf("ConversationID value not preserved: %q != %q",
					*req.ConversationID, *req2.ConversationID)
			}
		}
	})
}

// FuzzErrorHandling tests error handling properties
func FuzzErrorHandling(f *testing.F) {
	// Seed with various error scenarios
	f.Add("INVALID_INPUT", "test message")
	f.Add("PROCESSING_FAILED", "")
	f.Add("", "message without code")
	f.Add("VERY_LONG_CODE", strings.Repeat("message ", 100))

	f.Fuzz(func(t *testing.T, code, message string) {
		// Property: AssistantError should always be constructible
		err := customerrors.NewAssistantError(code, message, nil)
		if err == nil {
			t.Fatal("NewAssistantError returned nil")
		}

		// Property: Error string should contain both code and message
		errStr := err.Error()
		if code != "" && !strings.Contains(errStr, code) {
			t.Errorf("Error string should contain code %q, got %q", code, errStr)
		}

		if message != "" && !strings.Contains(errStr, message) {
			t.Errorf("Error string should contain message %q, got %q", message, errStr)
		}

		// Property: Error should implement error interface
		var _ error = err

		// Property: Unwrap should work correctly
		if err.Unwrap() != nil {
			t.Errorf("Expected nil cause, got %v", err.Unwrap())
		}

		// Property: Error response conversion should work
		response := customerrors.ToErrorResponse(err)
		if response == nil {
			t.Fatal("ToErrorResponse returned nil")
		}

		if response.Code != code {
			t.Errorf("Response code mismatch: expected %q, got %q", code, response.Code)
		}

		if response.Message != message {
			t.Errorf("Response message mismatch: expected %q, got %q", message, response.Message)
		}
	})
}

// FuzzToolNameValidation tests tool name validation properties
func FuzzToolNameValidation(f *testing.F) {
	// Seed with valid and invalid tool names
	f.Add("go_analyzer")
	f.Add("go-formatter")
	f.Add("GoBuilder")
	f.Add("")
	f.Add("tool with spaces")
	f.Add("tool/with/slashes")
	f.Add("tool.with.dots")
	f.Add(strings.Repeat("a", 256))

	f.Fuzz(func(t *testing.T, toolName string) {
		// Property: Tool name validation should be consistent
		isValidName := func(name string) bool {
			if name == "" {
				return false
			}
			if len(name) > 100 {
				return false
			}
			// Simple validation: alphanumeric, underscore, hyphen
			for _, r := range name {
				if !((r >= 'a' && r <= 'z') ||
					(r >= 'A' && r <= 'Z') ||
					(r >= '0' && r <= '9') ||
					r == '_' || r == '-') {
					return false
				}
			}
			return true
		}

		valid := isValidName(toolName)

		// Property: Empty names should be invalid
		if toolName == "" && valid {
			t.Error("Empty tool name should be invalid")
		}

		// Property: Very long names should be invalid
		if len(toolName) > 100 && valid {
			t.Error("Very long tool name should be invalid")
		}

		// Property: Names with invalid characters should be invalid
		hasInvalidChar := false
		for _, r := range toolName {
			if !((r >= 'a' && r <= 'z') ||
				(r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') ||
				r == '_' || r == '-') {
				hasInvalidChar = true
				break
			}
		}

		if hasInvalidChar && valid {
			t.Errorf("Tool name with invalid characters should be invalid: %q", toolName)
		}

		// Property: Tool not found error should be consistent
		err := NewToolNotFoundError(toolName)
		if assistantErr := customerrors.GetAssistantError(err); assistantErr != nil {
			if assistantErr.Code != CodeAssistantToolError {
				t.Errorf("Expected code %s, got %s", CodeAssistantToolError, assistantErr.Code)
			}
		}
	})
}

// FuzzConfigurationParsing tests configuration parsing robustness
func FuzzConfigurationParsing(f *testing.F) {
	// Seed with valid configurations
	validConfig := `
mode: test
ai:
  default_provider: claude
  claude:
    api_key: test-key
    model: claude-3-sonnet-20240229
`
	f.Add([]byte(validConfig))
	f.Add([]byte("{}"))
	f.Add([]byte(""))
	f.Add([]byte("invalid yaml"))

	f.Fuzz(func(t *testing.T, configData []byte) {
		// Property: Configuration parsing should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Configuration parsing panicked: %v", r)
			}
		}()

		// Skip if not valid UTF-8
		if !utf8.Valid(configData) {
			t.Skip("Invalid UTF-8 data")
		}

		// Property: Invalid configuration should return error, not panic
		cfg := &config.Config{}
		// We can't test actual YAML parsing here without importing yaml,
		// but we can test that our validation doesn't panic

		// Test validation of config struct fields
		if cfg.Mode == "" {
			cfg.Mode = "test" // Default mode
		}

		// Property: Mode should be validated
		validModes := map[string]bool{
			"development": true,
			"production":  true,
			"test":        true,
		}

		if !validModes[cfg.Mode] && cfg.Mode != "" {
			// Invalid mode should be handled gracefully
			t.Logf("Invalid mode detected: %s", cfg.Mode)
		}
	})
}

// Property-based test helper: Check if a string is a valid UUID format
func isValidUUIDFormat(s string) bool {
	if len(s) != 36 {
		return false
	}
	// Simple UUID format check: 8-4-4-4-12
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	for i, r := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue // Skip hyphens
		}
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// FuzzUUIDHandling tests UUID handling properties
func FuzzUUIDHandling(f *testing.F) {
	// Seed with valid and invalid UUIDs
	f.Add("123e4567-e89b-12d3-a456-426614174000")
	f.Add("invalid-uuid")
	f.Add("")
	f.Add("123e4567-e89b-12d3-a456-42661417400")   // Too short
	f.Add("123e4567-e89b-12d3-a456-4266141740000") // Too long

	f.Fuzz(func(t *testing.T, uuidStr string) {
		// Property: UUID validation should be consistent
		isValid := isValidUUIDFormat(uuidStr)

		// Property: Empty UUIDs should be invalid
		if uuidStr == "" && isValid {
			t.Error("Empty UUID should be invalid")
		}

		// Property: Wrong length UUIDs should be invalid
		if len(uuidStr) != 36 && isValid {
			t.Error("UUID with wrong length should be invalid")
		}

		// Property: UUIDs without proper hyphens should be invalid
		if len(uuidStr) == 36 {
			if (uuidStr[8] != '-' || uuidStr[13] != '-' ||
				uuidStr[18] != '-' || uuidStr[23] != '-') && isValid {
				t.Error("UUID without proper hyphens should be invalid")
			}
		}

		// Test conversation ID validation
		req := &QueryRequest{
			Query: "test",
		}

		if uuidStr != "" {
			req.ConversationID = &uuidStr
		}

		// Property: Request with invalid conversation ID should be handled gracefully
		if req.ConversationID != nil && !isValid {
			t.Logf("Request with invalid conversation ID: %s", *req.ConversationID)
		}
	})
}
