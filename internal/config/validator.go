package config

import (
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strings"
)

// Validate validates the configuration
func Validate(cfg *Config) error {
	if err := validateRequired(cfg); err != nil {
		return fmt.Errorf("required field validation failed: %w", err)
	}

	if err := validateDatabase(cfg.Database); err != nil {
		return fmt.Errorf("database configuration validation failed: %w", err)
	}

	if err := validateServer(cfg.Server); err != nil {
		return fmt.Errorf("server configuration validation failed: %w", err)
	}

	if err := validateAI(cfg.AI); err != nil {
		return fmt.Errorf("AI configuration validation failed: %w", err)
	}

	if err := validateTools(cfg.Tools); err != nil {
		return fmt.Errorf("tools configuration validation failed: %w", err)
	}

	return nil
}

// validateRequired validates required fields
func validateRequired(cfg *Config) error {
	return validateRequiredRecursive(reflect.ValueOf(cfg).Elem(), reflect.TypeOf(cfg).Elem(), "")
}

// validateRequiredRecursive recursively validates required fields
func validateRequiredRecursive(v reflect.Value, t reflect.Type, prefix string) error {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		fieldName := fieldType.Name
		if prefix != "" {
			fieldName = prefix + "." + fieldName
		}

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			if err := validateRequiredRecursive(field, fieldType.Type, fieldName); err != nil {
				return err
			}
			continue
		}

		// Check if field is required
		requiredTag := fieldType.Tag.Get("required")
		if requiredTag != "true" {
			continue
		}

		// Check if field is empty
		if isEmptyValue(field) {
			return fmt.Errorf("required field %s is empty", fieldName)
		}
	}

	return nil
}

// isEmptyValue checks if a reflect.Value is empty
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	default:
		return false
	}
}

// validateDatabase validates database configuration
func validateDatabase(cfg DatabaseConfig) error {
	if cfg.URL == "" {
		return fmt.Errorf("database URL is required")
	}

	// Parse database URL
	parsedURL, err := url.Parse(cfg.URL)
	if err != nil {
		return fmt.Errorf("invalid database URL: %w", err)
	}

	if parsedURL.Scheme != "postgres" && parsedURL.Scheme != "postgresql" {
		return fmt.Errorf("database URL must use postgres:// or postgresql:// scheme")
	}

	// Validate connection pool settings
	if cfg.MaxConnections <= 0 {
		return fmt.Errorf("max_connections must be greater than 0")
	}

	if cfg.MinConnections < 0 {
		return fmt.Errorf("min_connections must be non-negative")
	}

	if cfg.MinConnections > cfg.MaxConnections {
		return fmt.Errorf("min_connections cannot be greater than max_connections")
	}

	// Validate migrations path
	if cfg.MigrationsPath != "" {
		if _, err := os.Stat(cfg.MigrationsPath); os.IsNotExist(err) {
			return fmt.Errorf("migrations path does not exist: %s", cfg.MigrationsPath)
		}
	}

	return nil
}

// validateServer validates server configuration
func validateServer(cfg ServerConfig) error {
	// Validate address format
	if cfg.Address == "" {
		return fmt.Errorf("server address is required")
	}

	// Validate TLS configuration
	if cfg.EnableTLS {
		if cfg.TLSCertFile == "" {
			return fmt.Errorf("TLS cert file is required when TLS is enabled")
		}
		if cfg.TLSKeyFile == "" {
			return fmt.Errorf("TLS key file is required when TLS is enabled")
		}

		// Check if TLS files exist
		if _, err := os.Stat(cfg.TLSCertFile); os.IsNotExist(err) {
			return fmt.Errorf("TLS cert file does not exist: %s", cfg.TLSCertFile)
		}
		if _, err := os.Stat(cfg.TLSKeyFile); os.IsNotExist(err) {
			return fmt.Errorf("TLS key file does not exist: %s", cfg.TLSKeyFile)
		}
	}

	// Validate timeout values
	if cfg.ReadTimeout <= 0 {
		return fmt.Errorf("read_timeout must be greater than 0")
	}
	if cfg.WriteTimeout <= 0 {
		return fmt.Errorf("write_timeout must be greater than 0")
	}
	if cfg.IdleTimeout <= 0 {
		return fmt.Errorf("idle_timeout must be greater than 0")
	}

	return nil
}

// validateAI validates AI configuration
func validateAI(cfg AIConfig) error {
	// Validate default provider
	validProviders := []string{"claude", "gemini"}
	if !contains(validProviders, cfg.DefaultProvider) {
		return fmt.Errorf("invalid default provider: %s (must be one of: %s)",
			cfg.DefaultProvider, strings.Join(validProviders, ", "))
	}

	// Validate that at least one provider is configured
	hasClaudeKey := cfg.Claude.APIKey != ""
	hasGeminiKey := cfg.Gemini.APIKey != ""

	if !hasClaudeKey && !hasGeminiKey {
		return fmt.Errorf("at least one AI provider (Claude or Gemini) must be configured with an API key")
	}

	// Validate Claude configuration if API key is provided
	if hasClaudeKey {
		if err := validateClaudeConfig(cfg.Claude); err != nil {
			return fmt.Errorf("Claude configuration validation failed: %w", err)
		}
	}

	// Validate Gemini configuration if API key is provided
	if hasGeminiKey {
		if err := validateGeminiConfig(cfg.Gemini); err != nil {
			return fmt.Errorf("Gemini configuration validation failed: %w", err)
		}
	}

	return nil
}

// validateClaudeConfig validates Claude-specific configuration
func validateClaudeConfig(cfg Claude) error {
	if cfg.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be greater than 0")
	}
	if cfg.Temperature < 0 || cfg.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if cfg.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}

	// Validate base URL format
	if _, err := url.Parse(cfg.BaseURL); err != nil {
		return fmt.Errorf("invalid base_url: %w", err)
	}

	return nil
}

// validateGeminiConfig validates Gemini-specific configuration
func validateGeminiConfig(cfg Gemini) error {
	if cfg.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be greater than 0")
	}
	if cfg.Temperature < 0 || cfg.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if cfg.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}

	// Validate base URL format
	if _, err := url.Parse(cfg.BaseURL); err != nil {
		return fmt.Errorf("invalid base_url: %w", err)
	}

	return nil
}

// validateTools validates tools configuration
func validateTools(cfg ToolsConfig) error {
	// Validate search configuration
	if cfg.Search.SearXNGURL != "" {
		if _, err := url.Parse(cfg.Search.SearXNGURL); err != nil {
			return fmt.Errorf("invalid SearXNG URL: %w", err)
		}
	}

	// Validate Kubernetes configuration
	if cfg.Kubernetes.ConfigPath != "" {
		if _, err := os.Stat(cfg.Kubernetes.ConfigPath); os.IsNotExist(err) {
			return fmt.Errorf("Kubernetes config file does not exist: %s", cfg.Kubernetes.ConfigPath)
		}
	}

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
