package config

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// ValidationError represents a detailed validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
	Code    string
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", ve.Field, ve.Message)
}

// ValidationContext provides context for validation
type ValidationContext struct {
	Mode         string
	Environment  string
	ValidateAll  bool
	CheckNetwork bool
}

// Validator provides comprehensive configuration validation
type Validator struct {
	context ValidationContext
	errors  []ValidationError
}

// NewValidator creates a new configuration validator
func NewValidator(ctx ValidationContext) *Validator {
	return &Validator{
		context: ctx,
		errors:  make([]ValidationError, 0),
	}
}

// ValidateConfigWithObservability validates configuration with enhanced error reporting and observability
func ValidateConfigWithObservability(cfg *Config) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		slog.Debug("Configuration validation completed",
			"duration", duration,
			"mode", cfg.Mode)
	}()

	validator := NewValidator(ValidationContext{
		Mode:         cfg.Mode,
		Environment:  cfg.Mode,
		ValidateAll:  true,
		CheckNetwork: cfg.Mode == "production",
	})

	err := validator.ValidateConfig(cfg)
	if err != nil {
		slog.Error("Configuration validation failed",
			"error", err,
			"error_count", len(validator.errors))
		return err
	}

	slog.Info("Configuration validation successful", "mode", cfg.Mode)
	return nil
}

// ValidateConfig performs comprehensive configuration validation
func (v *Validator) ValidateConfig(cfg *Config) error {
	slog.Debug("Starting configuration validation", "mode", v.context.Mode)

	// 1. Validate required fields
	v.validateRequired(cfg)

	// 2. Validate business logic
	v.validateBusinessLogic(cfg)

	// 3. Validate component configurations
	v.validateDatabase(cfg.Database)
	v.validateServer(cfg.Server)
	v.validateAI(cfg.AI)
	v.validateSecurity(cfg.Security)
	v.validateTools(cfg.Tools)

	// 4. Validate cross-component dependencies
	v.validateDependencies(cfg)

	// 5. Validate environment-specific requirements
	v.validateEnvironmentRequirements(cfg)

	if len(v.errors) > 0 {
		return v.formatErrors()
	}

	slog.Info("Configuration validation successful", "mode", v.context.Mode)
	return nil
}

// validateRequired validates required fields with detailed error reporting
func (v *Validator) validateRequired(cfg *Config) {
	v.validateRequiredRecursive(reflect.ValueOf(cfg).Elem(), reflect.TypeOf(cfg).Elem(), "")
}

// validateRequiredRecursive recursively validates required fields
func (v *Validator) validateRequiredRecursive(val reflect.Value, typ reflect.Type, prefix string) {
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		fieldName := fieldType.Name
		if prefix != "" {
			fieldName = prefix + "." + fieldName
		}

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			v.validateRequiredRecursive(field, fieldType.Type, fieldName)
			continue
		}

		// Check if field is required
		requiredTag := fieldType.Tag.Get("required")
		if requiredTag != "true" {
			continue
		}

		// Check if field is empty
		if v.isEmptyValue(field) {
			envTag := fieldType.Tag.Get("env")
			message := fmt.Sprintf("required field is empty")
			if envTag != "" {
				message += fmt.Sprintf(". Set environment variable %s", envTag)
			}
			v.addError(fieldName, "", message, "REQUIRED_FIELD_EMPTY")
		}
	}
}

// validateBusinessLogic validates business logic constraints
func (v *Validator) validateBusinessLogic(cfg *Config) {
	// Validate mode is supported
	validModes := []string{"development", "staging", "production"}
	if !contains(validModes, cfg.Mode) {
		v.addError("Mode", cfg.Mode, fmt.Sprintf("must be one of: %s", strings.Join(validModes, ", ")), "INVALID_MODE")
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, cfg.LogLevel) {
		v.addError("LogLevel", cfg.LogLevel, fmt.Sprintf("must be one of: %s", strings.Join(validLogLevels, ", ")), "INVALID_LOG_LEVEL")
	}

	// Production-specific validations
	if cfg.Mode == "production" {
		if cfg.LogLevel == "debug" {
			v.addError("LogLevel", cfg.LogLevel, "debug logging should not be used in production", "PRODUCTION_DEBUG_LOG")
		}
		if cfg.LogFormat != "json" {
			v.addError("LogFormat", cfg.LogFormat, "production should use JSON logging format", "PRODUCTION_LOG_FORMAT")
		}
	}
}

// validateDependencies validates cross-component dependencies
func (v *Validator) validateDependencies(cfg *Config) {
	// Validate AI provider consistency - only enforce in production
	if v.context.Mode == "production" {
		if cfg.AI.DefaultProvider == "claude" && cfg.AI.Claude.APIKey == "" {
			v.addError("AI.Claude.APIKey", "", "API key required when Claude is the default provider", "MISSING_PROVIDER_KEY")
		}
		if cfg.AI.DefaultProvider == "gemini" && cfg.AI.Gemini.APIKey == "" {
			v.addError("AI.Gemini.APIKey", "", "API key required when Gemini is the default provider", "MISSING_PROVIDER_KEY")
		}
	}

	// Validate TLS dependency
	if cfg.Server.EnableTLS {
		if cfg.Server.TLSCertFile == "" {
			v.addError("Server.TLSCertFile", "", "TLS certificate file required when TLS is enabled", "MISSING_TLS_CERT")
		}
		if cfg.Server.TLSKeyFile == "" {
			v.addError("Server.TLSKeyFile", "", "TLS key file required when TLS is enabled", "MISSING_TLS_KEY")
		}
	}
}

// validateEnvironmentRequirements validates environment-specific requirements
func (v *Validator) validateEnvironmentRequirements(cfg *Config) {
	switch cfg.Mode {
	case "production":
		v.validateProductionRequirements(cfg)
	case "development":
		v.validateDevelopmentRequirements(cfg)
	case "staging":
		v.validateStagingRequirements(cfg)
	}
}

// validateProductionRequirements validates production-specific requirements
func (v *Validator) validateProductionRequirements(cfg *Config) {
	// Security requirements
	if len(cfg.Security.JWTSecret) < 32 {
		v.addError("Security.JWTSecret", "[REDACTED]", "must be at least 32 characters in production", "WEAK_JWT_SECRET")
	}

	// TLS should be enabled
	if !cfg.Server.EnableTLS {
		v.addError("Server.EnableTLS", false, "TLS should be enabled in production", "PRODUCTION_NO_TLS")
	}

	// Rate limiting should be reasonable
	if cfg.Security.RateLimitRPS > 1000 {
		v.addError("Security.RateLimitRPS", cfg.Security.RateLimitRPS, "rate limit seems too high for production", "HIGH_RATE_LIMIT")
	}
}

// validateDevelopmentRequirements validates development-specific requirements
func (v *Validator) validateDevelopmentRequirements(cfg *Config) {
	// Warn about potential development issues
	if cfg.Database.MaxConnections > 50 {
		v.addError("Database.MaxConnections", cfg.Database.MaxConnections, "high connection count may not be needed in development", "HIGH_DEV_CONNECTIONS")
	}
}

// validateStagingRequirements validates staging-specific requirements
func (v *Validator) validateStagingRequirements(cfg *Config) {
	// Staging should mirror production closely
	if cfg.LogFormat != "json" {
		v.addError("LogFormat", cfg.LogFormat, "staging should use JSON format to match production", "STAGING_LOG_FORMAT")
	}
}

// validateDatabase validates database configuration with enhanced checks
func (v *Validator) validateDatabase(cfg DatabaseConfig) {
	if cfg.URL == "" {
		v.addError("Database.URL", "", "database URL is required. Set DATABASE_URL environment variable", "MISSING_DATABASE_URL")
		return
	}

	// Parse and validate database URL
	parsedURL, err := url.Parse(cfg.URL)
	if err != nil {
		v.addError("Database.URL", cfg.URL, fmt.Sprintf("invalid database URL: %v", err), "INVALID_DATABASE_URL")
		return
	}

	if parsedURL.Scheme != "postgres" && parsedURL.Scheme != "postgresql" {
		v.addError("Database.URL", cfg.URL, "database URL must use postgres:// or postgresql:// scheme", "INVALID_DATABASE_SCHEME")
	}

	// Validate connection pool settings
	if cfg.MaxConnections <= 0 {
		v.addError("Database.MaxConnections", cfg.MaxConnections, "must be greater than 0", "INVALID_MAX_CONNECTIONS")
	}

	if cfg.MinConnections < 0 {
		v.addError("Database.MinConnections", cfg.MinConnections, "must be non-negative", "INVALID_MIN_CONNECTIONS")
	}

	if cfg.MinConnections > cfg.MaxConnections {
		v.addError("Database.MinConnections", cfg.MinConnections, "cannot be greater than max_connections", "MIN_GREATER_THAN_MAX")
	}

	// PostgreSQL 17 optimization recommendations
	if cfg.MaxConnections > 100 {
		v.addError("Database.MaxConnections", cfg.MaxConnections, "consider reducing for optimal PostgreSQL 17 performance", "HIGH_CONNECTION_COUNT")
	}

	// Validate timeouts
	if cfg.ConnectTimeout < time.Second {
		v.addError("Database.ConnectTimeout", cfg.ConnectTimeout, "should be at least 1 second", "SHORT_CONNECT_TIMEOUT")
	}

	if cfg.MaxIdleTime < time.Minute {
		v.addError("Database.MaxIdleTime", cfg.MaxIdleTime, "should be at least 1 minute for connection efficiency", "SHORT_IDLE_TIMEOUT")
	}

	// Validate migrations path - only fail in production or if explicitly set
	if cfg.MigrationsPath != "" && v.context.Mode == "production" {
		if _, err := os.Stat(cfg.MigrationsPath); os.IsNotExist(err) {
			v.addError("Database.MigrationsPath", cfg.MigrationsPath, "migrations directory does not exist", "MISSING_MIGRATIONS_PATH")
		}
	}

	// Network connectivity check in production
	if v.context.CheckNetwork && parsedURL.Host != "" {
		v.validateDatabaseConnectivity(parsedURL.Host)
	}
}

// validateDatabaseConnectivity checks if database host is reachable
func (v *Validator) validateDatabaseConnectivity(host string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		v.addError("Database.URL", host, fmt.Sprintf("cannot connect to database host: %v", err), "DATABASE_UNREACHABLE")
		return
	}
	conn.Close()
}

// validateServer validates server configuration with enhanced checks
func (v *Validator) validateServer(cfg ServerConfig) {
	// Validate address format
	if cfg.Address == "" {
		v.addError("Server.Address", "", "server address is required", "MISSING_SERVER_ADDRESS")
		return
	}

	// Validate address format more thoroughly
	if !v.isValidAddress(cfg.Address) {
		v.addError("Server.Address", cfg.Address, "invalid address format. Use :port or host:port", "INVALID_ADDRESS_FORMAT")
	}

	// Validate TLS configuration
	if cfg.EnableTLS {
		v.validateTLSConfiguration(cfg)
	}

	// Validate timeout values with recommendations
	if cfg.ReadTimeout <= 0 {
		v.addError("Server.ReadTimeout", cfg.ReadTimeout, "must be greater than 0", "INVALID_READ_TIMEOUT")
	} else if cfg.ReadTimeout < 5*time.Second {
		v.addError("Server.ReadTimeout", cfg.ReadTimeout, "consider increasing to at least 5 seconds for reliability", "SHORT_READ_TIMEOUT")
	}

	if cfg.WriteTimeout <= 0 {
		v.addError("Server.WriteTimeout", cfg.WriteTimeout, "must be greater than 0", "INVALID_WRITE_TIMEOUT")
	} else if cfg.WriteTimeout < 5*time.Second {
		v.addError("Server.WriteTimeout", cfg.WriteTimeout, "consider increasing to at least 5 seconds for reliability", "SHORT_WRITE_TIMEOUT")
	}

	if cfg.IdleTimeout <= 0 {
		v.addError("Server.IdleTimeout", cfg.IdleTimeout, "must be greater than 0", "INVALID_IDLE_TIMEOUT")
	}

	if cfg.ShutdownTimeout <= 0 {
		v.addError("Server.ShutdownTimeout", cfg.ShutdownTimeout, "must be greater than 0", "INVALID_SHUTDOWN_TIMEOUT")
	} else if cfg.ShutdownTimeout > 60*time.Second {
		v.addError("Server.ShutdownTimeout", cfg.ShutdownTimeout, "shutdown timeout too long, consider reducing for faster deployments", "LONG_SHUTDOWN_TIMEOUT")
	}
}

// isValidAddress validates server address format
func (v *Validator) isValidAddress(address string) bool {
	// Match :port or host:port patterns
	portOnlyPattern := regexp.MustCompile(`^:[1-9][0-9]*$`)
	hostPortPattern := regexp.MustCompile(`^[a-zA-Z0-9.-]+:[1-9][0-9]*$`)

	return portOnlyPattern.MatchString(address) || hostPortPattern.MatchString(address)
}

// validateTLSConfiguration validates TLS-specific settings
func (v *Validator) validateTLSConfiguration(cfg ServerConfig) {
	if cfg.TLSCertFile == "" {
		v.addError("Server.TLSCertFile", "", "TLS certificate file required when TLS is enabled", "MISSING_TLS_CERT")
	} else if _, err := os.Stat(cfg.TLSCertFile); os.IsNotExist(err) {
		v.addError("Server.TLSCertFile", cfg.TLSCertFile, "TLS certificate file does not exist", "TLS_CERT_NOT_FOUND")
	}

	if cfg.TLSKeyFile == "" {
		v.addError("Server.TLSKeyFile", "", "TLS private key file required when TLS is enabled", "MISSING_TLS_KEY")
	} else if _, err := os.Stat(cfg.TLSKeyFile); os.IsNotExist(err) {
		v.addError("Server.TLSKeyFile", cfg.TLSKeyFile, "TLS private key file does not exist", "TLS_KEY_NOT_FOUND")
	}
}

// validateAI validates AI configuration with provider-specific checks
func (v *Validator) validateAI(cfg AIConfig) {
	// Validate default provider
	validProviders := []string{"claude", "gemini"}
	if !contains(validProviders, cfg.DefaultProvider) {
		v.addError("AI.DefaultProvider", cfg.DefaultProvider,
			fmt.Sprintf("must be one of: %s", strings.Join(validProviders, ", ")), "INVALID_AI_PROVIDER")
	}

	// Validate that at least one provider is configured
	hasClaudeKey := cfg.Claude.APIKey != ""
	hasGeminiKey := cfg.Gemini.APIKey != ""

	if !hasClaudeKey && !hasGeminiKey {
		v.addError("AI", "no providers", "at least one AI provider (Claude or Gemini) must be configured with an API key", "NO_AI_PROVIDERS")
	}

	// Validate provider configurations
	if hasClaudeKey {
		v.validateClaudeConfig(cfg.Claude)
	}
	if hasGeminiKey {
		v.validateGeminiConfig(cfg.Gemini)
	}

	// Validate embeddings configuration
	v.validateEmbeddingsConfig(cfg.Embeddings)
}

// validateClaudeConfig validates Claude-specific configuration
func (v *Validator) validateClaudeConfig(cfg Claude) {
	// Validate API key format (basic check) - only in production or when key is provided
	if cfg.APIKey != "" && !v.isValidClaudeAPIKey(cfg.APIKey) {
		v.addError("AI.Claude.APIKey", "[REDACTED]", "invalid Claude API key format", "INVALID_CLAUDE_API_KEY")
	}

	// Validate model configuration
	if cfg.MaxTokens <= 0 {
		v.addError("AI.Claude.MaxTokens", cfg.MaxTokens, "must be greater than 0", "INVALID_MAX_TOKENS")
	} else if cfg.MaxTokens > 200000 {
		v.addError("AI.Claude.MaxTokens", cfg.MaxTokens, "exceeds Claude's maximum token limit", "EXCESSIVE_MAX_TOKENS")
	}

	if cfg.Temperature < 0 || cfg.Temperature > 2 {
		v.addError("AI.Claude.Temperature", cfg.Temperature, "must be between 0 and 2", "INVALID_TEMPERATURE")
	}

	if cfg.BaseURL == "" {
		v.addError("AI.Claude.BaseURL", "", "base URL is required", "MISSING_BASE_URL")
	} else {
		if _, err := url.Parse(cfg.BaseURL); err != nil {
			v.addError("AI.Claude.BaseURL", cfg.BaseURL, fmt.Sprintf("invalid URL format: %v", err), "INVALID_BASE_URL")
		}
	}

	// Validate model name
	validClaudeModels := []string{"claude-3-sonnet-20240229", "claude-3-opus-20240229", "claude-3-haiku-20240307"}
	if !contains(validClaudeModels, cfg.Model) {
		v.addError("AI.Claude.Model", cfg.Model, "consider using a supported Claude model", "UNSUPPORTED_CLAUDE_MODEL")
	}
}

// validateGeminiConfig validates Gemini-specific configuration
func (v *Validator) validateGeminiConfig(cfg Gemini) {
	// Validate API key format (basic check) - only when key is provided
	if cfg.APIKey != "" && !v.isValidGeminiAPIKey(cfg.APIKey) {
		v.addError("AI.Gemini.APIKey", "[REDACTED]", "invalid Gemini API key format", "INVALID_GEMINI_API_KEY")
	}

	// Validate model configuration
	if cfg.MaxTokens <= 0 {
		v.addError("AI.Gemini.MaxTokens", cfg.MaxTokens, "must be greater than 0", "INVALID_MAX_TOKENS")
	} else if cfg.MaxTokens > 1000000 {
		v.addError("AI.Gemini.MaxTokens", cfg.MaxTokens, "exceeds Gemini's maximum token limit", "EXCESSIVE_MAX_TOKENS")
	}

	if cfg.Temperature < 0 || cfg.Temperature > 2 {
		v.addError("AI.Gemini.Temperature", cfg.Temperature, "must be between 0 and 2", "INVALID_TEMPERATURE")
	}

	if cfg.BaseURL == "" {
		v.addError("AI.Gemini.BaseURL", "", "base URL is required", "MISSING_BASE_URL")
	} else {
		if _, err := url.Parse(cfg.BaseURL); err != nil {
			v.addError("AI.Gemini.BaseURL", cfg.BaseURL, fmt.Sprintf("invalid URL format: %v", err), "INVALID_BASE_URL")
		}
	}

	// Validate model name
	validGeminiModels := []string{"gemini-pro", "gemini-pro-vision", "gemini-1.5-pro"}
	if !contains(validGeminiModels, cfg.Model) {
		v.addError("AI.Gemini.Model", cfg.Model, "consider using a supported Gemini model", "UNSUPPORTED_GEMINI_MODEL")
	}
}

// validateEmbeddingsConfig validates embeddings configuration
func (v *Validator) validateEmbeddingsConfig(cfg Embedding) {
	validEmbeddingProviders := []string{"claude", "openai", "gemini"}
	if !contains(validEmbeddingProviders, cfg.Provider) {
		v.addError("AI.Embeddings.Provider", cfg.Provider,
			fmt.Sprintf("must be one of: %s", strings.Join(validEmbeddingProviders, ", ")), "INVALID_EMBEDDING_PROVIDER")
	}

	if cfg.Dimensions <= 0 {
		v.addError("AI.Embeddings.Dimensions", cfg.Dimensions, "must be greater than 0", "INVALID_EMBEDDING_DIMENSIONS")
	} else if cfg.Dimensions > 4096 {
		v.addError("AI.Embeddings.Dimensions", cfg.Dimensions, "dimension count seems too high", "HIGH_EMBEDDING_DIMENSIONS")
	}
}

// validateSecurity validates security configuration
func (v *Validator) validateSecurity(cfg SecurityConfig) {
	// JWT secret validation
	if cfg.JWTSecret == "" {
		v.addError("Security.JWTSecret", "", "JWT secret is required. Set JWT_SECRET environment variable", "MISSING_JWT_SECRET")
	} else {
		v.validateJWTSecret(cfg.JWTSecret)
	}

	// JWT expiration validation
	if cfg.JWTExpiration < time.Minute {
		v.addError("Security.JWTExpiration", cfg.JWTExpiration, "JWT expiration should be at least 1 minute", "SHORT_JWT_EXPIRATION")
	} else if cfg.JWTExpiration > 30*24*time.Hour {
		v.addError("Security.JWTExpiration", cfg.JWTExpiration, "JWT expiration seems too long (>30 days)", "LONG_JWT_EXPIRATION")
	}

	// Rate limiting validation
	if cfg.RateLimitRPS <= 0 {
		v.addError("Security.RateLimitRPS", cfg.RateLimitRPS, "rate limit RPS must be greater than 0", "INVALID_RATE_LIMIT_RPS")
	}

	if cfg.RateLimitBurst <= 0 {
		v.addError("Security.RateLimitBurst", cfg.RateLimitBurst, "rate limit burst must be greater than 0", "INVALID_RATE_LIMIT_BURST")
	}

	if cfg.RateLimitBurst < cfg.RateLimitRPS {
		v.addError("Security.RateLimitBurst", cfg.RateLimitBurst, "burst should be >= RPS for optimal rate limiting", "BURST_LESS_THAN_RPS")
	}

	// CORS validation
	if cfg.EnableCORS {
		v.validateCORSConfig(cfg.AllowedOrigins)
	}
}

// validateJWTSecret validates JWT secret strength
func (v *Validator) validateJWTSecret(secret string) {
	if len(secret) < 16 {
		v.addError("Security.JWTSecret", "[REDACTED]", "should be at least 16 characters", "WEAK_JWT_SECRET")
	}

	// Check for common weak patterns
	if secret == "secret" || secret == "your-secret-key" || secret == "change-me" {
		v.addError("Security.JWTSecret", "[REDACTED]", "using default or example secret key", "DEFAULT_JWT_SECRET")
	}

	// Production strength validation
	if v.context.Mode == "production" && len(secret) < 32 {
		v.addError("Security.JWTSecret", "[REDACTED]", "should be at least 32 characters in production", "PRODUCTION_WEAK_JWT")
	}
}

// validateCORSConfig validates CORS configuration
func (v *Validator) validateCORSConfig(allowedOrigins []string) {
	for _, origin := range allowedOrigins {
		if origin == "*" && v.context.Mode == "production" {
			v.addError("Security.AllowedOrigins", origin, "wildcard CORS should not be used in production", "WILDCARD_CORS_PRODUCTION")
		}
		if !v.isValidOrigin(origin) {
			v.addError("Security.AllowedOrigins", origin, "invalid origin format", "INVALID_CORS_ORIGIN")
		}
	}
}

// validateTools validates tools configuration with comprehensive checks
func (v *Validator) validateTools(cfg ToolsConfig) {
	v.validateSearchConfig(cfg.Search)
	v.validatePostgresConfig(cfg.Postgres)
	v.validateKubernetesConfig(cfg.Kubernetes)
	v.validateDockerConfig(cfg.Docker)
	v.validateCloudflareConfig(cfg.Cloudflare)
	v.validateLangChainConfig(cfg.LangChain)
}

// Helper methods for validation

// isEmptyValue checks if a reflect.Value is empty
func (v *Validator) isEmptyValue(val reflect.Value) bool {
	switch val.Kind() {
	case reflect.String:
		return strings.TrimSpace(val.String()) == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// For time.Duration, 0 is a valid value
		if val.Type() == reflect.TypeOf(time.Duration(0)) {
			return false
		}
		return val.Int() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Bool:
		return false // Boolean false is a valid value
	case reflect.Slice, reflect.Map, reflect.Array:
		return val.Len() == 0
	case reflect.Interface, reflect.Ptr:
		return val.IsNil()
	default:
		return false
	}
}

// addError adds a validation error
func (v *Validator) addError(field string, value interface{}, message, code string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
		Code:    code,
	})
}

// formatErrors formats all validation errors into a single error
func (v *Validator) formatErrors() error {
	if len(v.errors) == 0 {
		return nil
	}

	var messages []string
	for _, err := range v.errors {
		messages = append(messages, fmt.Sprintf("- %s", err.Error()))
	}

	return fmt.Errorf("configuration validation failed:\n%s", strings.Join(messages, "\n"))
}

// API key validation helpers
func (v *Validator) isValidClaudeAPIKey(key string) bool {
	// Allow test keys in non-production environments
	if v.context.Mode != "production" && (key == "test-key" || strings.HasPrefix(key, "test-")) {
		return true
	}
	// Claude API keys typically start with "sk-ant-"
	return strings.HasPrefix(key, "sk-ant-") && len(key) > 20
}

func (v *Validator) isValidGeminiAPIKey(key string) bool {
	// Basic validation for Google API key format
	return len(key) >= 32 && !strings.Contains(key, " ")
}

// Origin validation
func (v *Validator) isValidOrigin(origin string) bool {
	if origin == "*" {
		return true
	}
	if strings.HasPrefix(origin, "http://") || strings.HasPrefix(origin, "https://") {
		_, err := url.Parse(origin)
		return err == nil
	}
	return false
}

// Email validation
func (v *Validator) isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// Additional validation methods for tools configurations
func (v *Validator) validateSearchConfig(cfg Search) {
	if cfg.SearXNGURL != "" {
		if _, err := url.Parse(cfg.SearXNGURL); err != nil {
			v.addError("Tools.Search.SearXNGURL", cfg.SearXNGURL, fmt.Sprintf("invalid URL: %v", err), "INVALID_SEARXNG_URL")
		}
	}

	if cfg.MaxResults <= 0 {
		v.addError("Tools.Search.MaxResults", cfg.MaxResults, "must be greater than 0", "INVALID_MAX_RESULTS")
	} else if cfg.MaxResults > 100 {
		v.addError("Tools.Search.MaxResults", cfg.MaxResults, "consider reducing for better performance", "HIGH_MAX_RESULTS")
	}

	if cfg.Timeout <= 0 {
		v.addError("Tools.Search.Timeout", cfg.Timeout, "must be greater than 0", "INVALID_SEARCH_TIMEOUT")
	}
}

func (v *Validator) validatePostgresConfig(cfg Postgres) {
	if cfg.QueryTimeout <= 0 {
		v.addError("Tools.Postgres.QueryTimeout", cfg.QueryTimeout, "must be greater than 0", "INVALID_QUERY_TIMEOUT")
	} else if cfg.QueryTimeout > 5*time.Minute {
		v.addError("Tools.Postgres.QueryTimeout", cfg.QueryTimeout, "very long timeout may cause issues", "LONG_QUERY_TIMEOUT")
	}

	if cfg.MaxQuerySize <= 0 {
		v.addError("Tools.Postgres.MaxQuerySize", cfg.MaxQuerySize, "must be greater than 0", "INVALID_MAX_QUERY_SIZE")
	}
}

func (v *Validator) validateKubernetesConfig(cfg Kubernetes) {
	if cfg.ConfigPath != "" {
		if _, err := os.Stat(cfg.ConfigPath); os.IsNotExist(err) {
			v.addError("Tools.Kubernetes.ConfigPath", cfg.ConfigPath, "kubeconfig file does not exist", "MISSING_KUBECONFIG")
		}
	}

	if cfg.Namespace == "" {
		v.addError("Tools.Kubernetes.Namespace", "", "namespace should be specified", "MISSING_NAMESPACE")
	}

	if cfg.Timeout <= 0 {
		v.addError("Tools.Kubernetes.Timeout", cfg.Timeout, "must be greater than 0", "INVALID_KUBE_TIMEOUT")
	}
}

func (v *Validator) validateDockerConfig(cfg Docker) {
	if cfg.Host == "" {
		v.addError("Tools.Docker.Host", "", "Docker host is required", "MISSING_DOCKER_HOST")
	}

	if cfg.APIVersion == "" {
		v.addError("Tools.Docker.APIVersion", "", "Docker API version should be specified", "MISSING_API_VERSION")
	}

	if cfg.Timeout <= 0 {
		v.addError("Tools.Docker.Timeout", cfg.Timeout, "must be greater than 0", "INVALID_DOCKER_TIMEOUT")
	}
}

func (v *Validator) validateCloudflareConfig(cfg Cloudflare) {
	// Either API token or API key + email is required - only validate in production
	hasToken := cfg.APIToken != ""
	hasKeyEmail := cfg.APIKey != "" && cfg.Email != ""

	if v.context.Mode == "production" && !hasToken && !hasKeyEmail {
		v.addError("Tools.Cloudflare", "no auth", "either API token or API key + email is required in production", "MISSING_CLOUDFLARE_AUTH")
	}

	if hasKeyEmail && cfg.Email != "" {
		if !v.isValidEmail(cfg.Email) {
			v.addError("Tools.Cloudflare.Email", cfg.Email, "invalid email format", "INVALID_EMAIL")
		}
	}
}

func (v *Validator) validateLangChainConfig(cfg LangChain) {
	if cfg.MemorySize <= 0 {
		v.addError("Tools.LangChain.MemorySize", cfg.MemorySize, "must be greater than 0", "INVALID_MEMORY_SIZE")
	} else if cfg.MemorySize > 1000 {
		v.addError("Tools.LangChain.MemorySize", cfg.MemorySize, "very large memory size may impact performance", "LARGE_MEMORY_SIZE")
	}

	if cfg.MaxIterations <= 0 {
		v.addError("Tools.LangChain.MaxIterations", cfg.MaxIterations, "must be greater than 0", "INVALID_MAX_ITERATIONS")
	} else if cfg.MaxIterations > 20 {
		v.addError("Tools.LangChain.MaxIterations", cfg.MaxIterations, "high iteration count may cause long processing times", "HIGH_MAX_ITERATIONS")
	}

	if cfg.Timeout <= 0 {
		v.addError("Tools.LangChain.Timeout", cfg.Timeout, "must be greater than 0", "INVALID_LANGCHAIN_TIMEOUT")
	}
}
