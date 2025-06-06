package config

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// LoaderOptions provides configuration for the security-first loader
type LoaderOptions struct {
	ConfigFile        string
	EnvironmentPrefix string
	SkipEnvFile       bool
	WatchChanges      bool
	ValidateSecrets   bool
}

// ConfigLoader provides secure configuration loading and management
type ConfigLoader struct {
	options      LoaderOptions
	current      *Config
	mutex        sync.RWMutex
	watchers     []chan<- *Config
	cancel       context.CancelFunc
	rotatingKeys *RotatingKeys
}

// RotatingKeys provides JWT key rotation capabilities
type RotatingKeys struct {
	current  string
	previous string
	mutex    sync.RWMutex
	rotateAt time.Time
}

// NewConfigLoader creates a new security-first configuration loader
func NewConfigLoader(opts LoaderOptions) *ConfigLoader {
	return &ConfigLoader{
		options:      opts,
		watchers:     make([]chan<- *Config, 0),
		rotatingKeys: &RotatingKeys{},
	}
}

// Load loads configuration using security-first principles from CLAUDE.md
func Load() (*Config, error) {
	loader := NewConfigLoader(LoaderOptions{
		ValidateSecrets: true,
	})
	return loader.Load(context.Background())
}

// Load loads configuration with security-first approach
func (cl *ConfigLoader) Load(ctx context.Context) (*Config, error) {
	cfg := &Config{}

	// 1. Load .env file for development (if not explicitly skipped)
	if !cl.options.SkipEnvFile {
		if err := cl.loadEnvFile(); err != nil {
			slog.Debug("No .env file found, continuing with system environment", "error", err)
		}
	}

	// 2. Load base configuration from YAML file and set defaults
	if err := cl.loadFromYAML(cfg); err != nil {
		return nil, fmt.Errorf("config file: %w", err)
	}

	// 3. Override with environment variables (including secrets)
	// cleanenv.ReadEnv automatically handles defaults from struct tags
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, fmt.Errorf("environment: %w", err)
	}

	// 4. Initialize JWT key rotation if needed
	if err := cl.initializeKeyRotation(cfg); err != nil {
		return nil, fmt.Errorf("key rotation: %w", err)
	}

	// 5. Validate critical security settings
	if err := cl.validateSecuritySettings(cfg); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	// 6. Store current configuration
	cl.mutex.Lock()
	cl.current = cfg
	cl.mutex.Unlock()

	// 7. Start hot reload if requested
	if cl.options.WatchChanges {
		go cl.watchForChanges(ctx)
	}

	return cfg, nil
}

// loadEnvFile loads .env file for development
func (cl *ConfigLoader) loadEnvFile() error {
	envFiles := []string{".env.local", ".env"}
	for _, envFile := range envFiles {
		if _, err := os.Stat(envFile); err == nil {
			if err := godotenv.Load(envFile); err != nil {
				return fmt.Errorf("failed to load %s: %w", envFile, err)
			}
			slog.Debug("Loaded environment file", "file", envFile)
			return nil
		}
	}
	return fmt.Errorf("no .env file found")
}

// loadFromYAML loads configuration from YAML files with security focus
func (cl *ConfigLoader) loadFromYAML(cfg *Config) error {
	configFile := cl.determineConfigFile()
	if configFile == "" {
		// No config file found, use defaults and environment variables only
		slog.Info("No configuration file found, using defaults and environment variables")
		// Still need to set defaults when no config file exists
		return cl.setDefaults(cfg)
	}

	slog.Info("Loading configuration file", "file", configFile)
	if err := cleanenv.ReadConfig(configFile, cfg); err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	return nil
}

// setDefaults manually sets default values when no config file is present
func (cl *ConfigLoader) setDefaults(cfg *Config) error {
	// Set basic defaults that match the struct tags
	cfg.Mode = "development"
	cfg.LogLevel = "info"
	cfg.LogFormat = "json"

	// Database defaults
	cfg.Database.MaxConnections = 30
	cfg.Database.MinConnections = 5
	cfg.Database.MaxIdleTime = 15 * time.Minute
	cfg.Database.MaxLifetime = time.Hour
	cfg.Database.ConnectTimeout = 10 * time.Second
	cfg.Database.MigrationsPath = "internal/storage/postgres/migrations"
	cfg.Database.EnableLogging = false

	// Server defaults
	cfg.Server.Address = ":8080"
	cfg.Server.ReadTimeout = 10 * time.Second
	cfg.Server.WriteTimeout = 10 * time.Second
	cfg.Server.IdleTimeout = 60 * time.Second
	cfg.Server.ShutdownTimeout = 30 * time.Second
	cfg.Server.EnableTLS = false

	// CLI defaults
	cfg.CLI.HistoryFile = ".assistant_history"
	cfg.CLI.MaxHistorySize = 1000
	cfg.CLI.EnableColors = true
	cfg.CLI.PromptTemplate = "Assistant> "

	// AI defaults
	cfg.AI.DefaultProvider = "claude"
	cfg.AI.Claude.Model = "claude-3-sonnet-20240229"
	cfg.AI.Claude.MaxTokens = 4096
	cfg.AI.Claude.Temperature = 0.7
	cfg.AI.Claude.BaseURL = "https://api.anthropic.com"
	cfg.AI.Gemini.Model = "gemini-pro"
	cfg.AI.Gemini.MaxTokens = 4096
	cfg.AI.Gemini.Temperature = 0.7
	cfg.AI.Gemini.BaseURL = "https://generativelanguage.googleapis.com"
	cfg.AI.Embeddings.Provider = "claude"
	cfg.AI.Embeddings.Model = "text-embedding-ada-002"
	cfg.AI.Embeddings.Dimensions = 1536

	// Tools defaults
	cfg.Tools.Search.SearXNGURL = "http://localhost:8888"
	cfg.Tools.Search.Timeout = 30 * time.Second
	cfg.Tools.Search.MaxResults = 10
	cfg.Tools.Search.EnableCache = true
	cfg.Tools.Search.CacheTTL = time.Hour

	cfg.Tools.Postgres.QueryTimeout = 30 * time.Second
	cfg.Tools.Postgres.MaxQuerySize = 1048576
	cfg.Tools.Postgres.EnableExplain = true

	cfg.Tools.Kubernetes.Namespace = "default"
	cfg.Tools.Kubernetes.Timeout = 30 * time.Second
	cfg.Tools.Kubernetes.EnableMetrics = true

	cfg.Tools.Docker.Host = "unix:///var/run/docker.sock"
	cfg.Tools.Docker.APIVersion = "1.41"
	cfg.Tools.Docker.Timeout = 30 * time.Second
	cfg.Tools.Docker.TLSVerify = false

	cfg.Tools.LangChain.EnableMemory = true
	cfg.Tools.LangChain.MemorySize = 10
	cfg.Tools.LangChain.MaxIterations = 5
	cfg.Tools.LangChain.Timeout = 60 * time.Second

	// Security defaults
	cfg.Security.JWTExpiration = 24 * time.Hour
	cfg.Security.RateLimitRPS = 100
	cfg.Security.RateLimitBurst = 200
	cfg.Security.EnableCORS = true

	return nil
}

// determineConfigFile determines which config file to use
func (cl *ConfigLoader) determineConfigFile() string {
	// 1. Explicit config file from options or environment
	if cl.options.ConfigFile != "" {
		return cl.options.ConfigFile
	}
	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		return configFile
	}

	// 2. Environment-based selection
	env := os.Getenv("APP_MODE")
	if env == "" {
		env = "development" // Default to development
	}

	// 3. Try environment-specific config first
	environmentConfig := filepath.Join("configs", env+".yaml")
	if _, err := os.Stat(environmentConfig); err == nil {
		return environmentConfig
	}

	// 4. Try common locations
	candidates := []string{
		"configs/development.yaml",
		"configs/production.yaml",
		"config.yaml",
		".config.yaml",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// initializeKeyRotation sets up JWT key rotation
func (cl *ConfigLoader) initializeKeyRotation(cfg *Config) error {
	if cfg.Security.JWTSecret == "" {
		// Generate a secure random key if none provided
		key, err := generateSecureKey(32)
		if err != nil {
			return fmt.Errorf("failed to generate JWT secret: %w", err)
		}
		cfg.Security.JWTSecret = key
		slog.Warn("Generated random JWT secret. Set JWT_SECRET environment variable for production.")
	}

	// Initialize rotating keys
	cl.rotatingKeys.current = cfg.Security.JWTSecret
	cl.rotatingKeys.rotateAt = time.Now().Add(24 * time.Hour) // Rotate daily

	return nil
}

// validateSecuritySettings validates critical security configuration
func (cl *ConfigLoader) validateSecuritySettings(cfg *Config) error {
	if cl.options.ValidateSecrets {
		// Validate JWT secret strength in production
		if cfg.Mode == "production" {
			if len(cfg.Security.JWTSecret) < 32 {
				return fmt.Errorf("JWT secret must be at least 32 characters in production")
			}
		}

		// Validate TLS configuration in production
		if cfg.Mode == "production" && !cfg.Server.EnableTLS {
			slog.Warn("TLS is disabled in production mode. This is not recommended.")
		}

		// Validate AI API keys
		if cfg.AI.Claude.APIKey == "" && cfg.AI.Gemini.APIKey == "" {
			return fmt.Errorf("at least one AI provider API key must be configured")
		}
	}

	return Validate(cfg)
}

// generateSecureKey generates a cryptographically secure random key
func generateSecureKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}

// watchForChanges implements hot reloading for configuration
func (cl *ConfigLoader) watchForChanges(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := cl.checkAndReload(); err != nil {
				slog.Error("Failed to reload configuration", "error", err)
			}
			if err := cl.checkKeyRotation(); err != nil {
				slog.Error("Failed to rotate keys", "error", err)
			}
		}
	}
}

// checkAndReload checks if configuration needs reloading
func (cl *ConfigLoader) checkAndReload() error {
	// For now, we'll implement basic environment variable checking
	// In a full implementation, you might watch file changes
	cfg := &Config{}
	if err := cl.loadFromYAML(cfg); err != nil {
		return err
	}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return err
	}

	// Compare with current config and notify watchers if changed
	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	// Simple comparison - in production you'd want more sophisticated change detection
	if cl.current.Mode != cfg.Mode || cl.current.LogLevel != cfg.LogLevel {
		cl.current = cfg
		for _, watcher := range cl.watchers {
			select {
			case watcher <- cfg:
			default:
				// Non-blocking send
			}
		}
		slog.Info("Configuration reloaded")
	}

	return nil
}

// checkKeyRotation checks if JWT keys need rotation
func (cl *ConfigLoader) checkKeyRotation() error {
	if time.Now().After(cl.rotatingKeys.rotateAt) {
		return cl.rotateJWTKeys()
	}
	return nil
}

// rotateJWTKeys rotates JWT signing keys
func (cl *ConfigLoader) rotateJWTKeys() error {
	newKey, err := generateSecureKey(32)
	if err != nil {
		return fmt.Errorf("failed to generate new JWT key: %w", err)
	}

	cl.rotatingKeys.mutex.Lock()
	defer cl.rotatingKeys.mutex.Unlock()

	cl.rotatingKeys.previous = cl.rotatingKeys.current
	cl.rotatingKeys.current = newKey
	cl.rotatingKeys.rotateAt = time.Now().Add(24 * time.Hour)

	slog.Info("JWT keys rotated successfully")
	return nil
}

// GetCurrentKeys returns the current and previous JWT keys for validation
func (cl *ConfigLoader) GetCurrentKeys() (current, previous string) {
	cl.rotatingKeys.mutex.RLock()
	defer cl.rotatingKeys.mutex.RUnlock()
	return cl.rotatingKeys.current, cl.rotatingKeys.previous
}

// ValidateSignature validates a JWT signature with key rotation support
func (cl *ConfigLoader) ValidateSignature(token string) error {
	current, previous := cl.GetCurrentKeys()

	// Try current key first
	if err := validateTokenWithKey(token, current); err == nil {
		return nil
	}

	// Fall back to previous key during rotation window
	if previous != "" {
		return validateTokenWithKey(token, previous)
	}

	return fmt.Errorf("token validation failed")
}

// validateTokenWithKey validates a token with a specific key (placeholder)
func validateTokenWithKey(token, key string) error {
	// This would integrate with your JWT library
	// For now, just a placeholder
	return fmt.Errorf("not implemented")
}

// Watch adds a configuration change watcher
func (cl *ConfigLoader) Watch() <-chan *Config {
	ch := make(chan *Config, 1)
	cl.mutex.Lock()
	cl.watchers = append(cl.watchers, ch)
	cl.mutex.Unlock()
	return ch
}

// Stop stops the configuration loader and cleans up resources
func (cl *ConfigLoader) Stop() {
	if cl.cancel != nil {
		cl.cancel()
	}
	cl.mutex.Lock()
	for _, watcher := range cl.watchers {
		close(watcher)
	}
	cl.watchers = nil
	cl.mutex.Unlock()
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() string {
	if configDir := os.Getenv("CONFIG_DIR"); configDir != "" {
		return configDir
	}

	// Default to configs directory
	return "configs"
}

// GetConfigFile returns the full path to the configuration file
func GetConfigFile(env string) string {
	configDir := GetConfigDir()
	return filepath.Join(configDir, fmt.Sprintf("%s.yaml", env))
}

// LoadWithLoader creates a new loader with specific options
func LoadWithLoader(opts LoaderOptions) (*Config, *ConfigLoader, error) {
	loader := NewConfigLoader(opts)
	cfg, err := loader.Load(context.Background())
	if err != nil {
		return nil, nil, err
	}
	return cfg, loader, nil
}
