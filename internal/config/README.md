# Configuration Management

## Overview

The configuration management system provides a robust, secure, and flexible configuration framework following 12-factor app methodology. It supports environment-based configuration with YAML file defaults, comprehensive validation, and intelligent configuration adaptation based on learned preferences.

## Architecture

```
internal/config/
├── config.go          # Core configuration structures and types
├── loader.go          # Configuration loading and merging logic
├── validator.go       # Configuration validation and sanitization
├── config_test.go     # Comprehensive test suite
└── testdata/
    └── migrations/
        └── 001_test.up.sql
```

## Key Features

### 🔒 **Security-First Design**
- **Environment Variable Priority**: Secrets in environment variables only
- **YAML Structure**: Non-sensitive configuration in version-controlled files
- **Key Rotation Support**: Graceful handling of API key updates
- **Validation**: Comprehensive security setting validation
- **Audit Trails**: Configuration change tracking

### 🎯 **12-Factor App Compliance**
- **Environment-Based**: Configuration through environment variables
- **Separation of Concerns**: Code, config, and secrets clearly separated
- **Portability**: Easy deployment across environments
- **No Secrets in Code**: Zero hardcoded secrets or credentials

### 🧠 **Intelligent Adaptation**
- **Preference Learning**: Adapts based on user patterns
- **Context-Aware Suggestions**: Configuration recommendations
- **Performance Optimization**: Auto-tuning based on usage patterns
- **Environment Detection**: Automatic development vs production settings

## Configuration Structure

### Core Configuration

```go
type Config struct {
    // Server configuration
    Server   ServerConfig   `yaml:"server" env:"SERVER"`
    
    // Database configuration
    Database DatabaseConfig `yaml:"database" env:"DATABASE"`
    
    // AI provider configuration
    AI       AIConfig       `yaml:"ai" env:"AI"`
    
    // Observability configuration
    Observability ObservabilityConfig `yaml:"observability" env:"OBSERVABILITY"`
    
    // Security configuration
    Security SecurityConfig `yaml:"security" env:"SECURITY"`
    
    // Feature flags
    Features FeatureConfig `yaml:"features" env:"FEATURES"`
}
```

### Server Configuration

```go
type ServerConfig struct {
    Host            string        `yaml:"host" env:"HOST" default:"localhost"`
    Port            int           `yaml:"port" env:"PORT" default:"8080"`
    ReadTimeout     time.Duration `yaml:"read_timeout" env:"READ_TIMEOUT" default:"30s"`
    WriteTimeout    time.Duration `yaml:"write_timeout" env:"WRITE_TIMEOUT" default:"30s"`
    ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"SHUTDOWN_TIMEOUT" default:"10s"`
    TLS             TLSConfig     `yaml:"tls" env:"TLS"`
}

type TLSConfig struct {
    Enabled  bool   `yaml:"enabled" env:"ENABLED" default:"false"`
    CertFile string `yaml:"cert_file" env:"CERT_FILE"`
    KeyFile  string `yaml:"key_file" env:"KEY_FILE"`
}
```

### Database Configuration

```go
type DatabaseConfig struct {
    URL             string        `yaml:"url" env:"DATABASE_URL" required:"true"`
    MaxConnections  int           `yaml:"max_connections" env:"MAX_CONNECTIONS" default:"30"`
    MinConnections  int           `yaml:"min_connections" env:"MIN_CONNECTIONS" default:"5"`
    ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"CONN_MAX_LIFETIME" default:"1h"`
    ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" env:"CONN_MAX_IDLE_TIME" default:"15m"`
    HealthCheckPeriod time.Duration `yaml:"health_check_period" env:"HEALTH_CHECK_PERIOD" default:"1m"`
    
    // Migration settings
    Migration MigrationConfig `yaml:"migration" env:"MIGRATION"`
}

type MigrationConfig struct {
    Enabled   bool   `yaml:"enabled" env:"ENABLED" default:"true"`
    Directory string `yaml:"directory" env:"DIRECTORY" default:"migrations"`
    TableName string `yaml:"table_name" env:"TABLE_NAME" default:"schema_migrations"`
}
```

### AI Configuration

```go
type AIConfig struct {
    Providers map[string]ProviderConfig `yaml:"providers" env:"PROVIDERS"`
    Retry     RetryConfig               `yaml:"retry" env:"RETRY"`
    Timeout   time.Duration             `yaml:"timeout" env:"TIMEOUT" default:"30s"`
}

type ProviderConfig struct {
    Enabled      bool              `yaml:"enabled" env:"ENABLED" default:"true"`
    APIKey       string            `yaml:"-" env:"API_KEY" required:"true"`
    Model        string            `yaml:"model" env:"MODEL"`
    Temperature  float64           `yaml:"temperature" env:"TEMPERATURE" default:"0.7"`
    MaxTokens    int               `yaml:"max_tokens" env:"MAX_TOKENS" default:"4096"`
    RateLimit    RateLimitConfig   `yaml:"rate_limit" env:"RATE_LIMIT"`
    Safety       SafetyConfig      `yaml:"safety" env:"SAFETY"`
}
```

### Observability Configuration

```go
type ObservabilityConfig struct {
    Logging LoggingConfig `yaml:"logging" env:"LOGGING"`
    Metrics MetricsConfig `yaml:"metrics" env:"METRICS"`
    Tracing TracingConfig `yaml:"tracing" env:"TRACING"`
}

type LoggingConfig struct {
    Level      string `yaml:"level" env:"LEVEL" default:"info"`
    Format     string `yaml:"format" env:"FORMAT" default:"json"`
    Output     string `yaml:"output" env:"OUTPUT" default:"stdout"`
    Structured bool   `yaml:"structured" env:"STRUCTURED" default:"true"`
}

type MetricsConfig struct {
    Enabled    bool   `yaml:"enabled" env:"ENABLED" default:"true"`
    Port       int    `yaml:"port" env:"PORT" default:"9090"`
    Path       string `yaml:"path" env:"PATH" default:"/metrics"`
    Namespace  string `yaml:"namespace" env:"NAMESPACE" default:"assistant"`
}
```

## Configuration Loading

### Loading Priority

1. **Default Values**: Struct tags define sensible defaults
2. **YAML Files**: Environment-specific configuration files
3. **Environment Variables**: Override YAML settings
4. **Validation**: Ensure configuration is valid and secure

### Loader Implementation

```go
type Loader struct {
    environment string
    configPaths []string
    validator   *Validator
    logger      *slog.Logger
}

func NewLoader(env string, paths ...string) *Loader {
    return &Loader{
        environment: env,
        configPaths: paths,
        validator:   NewValidator(),
        logger:      slog.Default(),
    }
}

func (l *Loader) Load() (*Config, error) {
    cfg := &Config{}
    
    // 1. Load defaults from struct tags
    if err := l.loadDefaults(cfg); err != nil {
        return nil, fmt.Errorf("loading defaults: %w", err)
    }
    
    // 2. Load from YAML files
    if err := l.loadYAML(cfg); err != nil {
        return nil, fmt.Errorf("loading YAML: %w", err)
    }
    
    // 3. Override with environment variables
    if err := l.loadEnvironment(cfg); err != nil {
        return nil, fmt.Errorf("loading environment: %w", err)
    }
    
    // 4. Validate configuration
    if err := l.validator.Validate(cfg); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    return cfg, nil
}
```

### Environment-Specific Loading

```go
func (l *Loader) loadYAML(cfg *Config) error {
    for _, path := range l.configPaths {
        filename := filepath.Join(path, fmt.Sprintf("%s.yaml", l.environment))
        
        if _, err := os.Stat(filename); os.IsNotExist(err) {
            l.logger.Debug("Config file not found", "file", filename)
            continue
        }
        
        data, err := os.ReadFile(filename)
        if err != nil {
            return fmt.Errorf("reading config file %s: %w", filename, err)
        }
        
        if err := yaml.Unmarshal(data, cfg); err != nil {
            return fmt.Errorf("parsing config file %s: %w", filename, err)
        }
        
        l.logger.Info("Loaded config file", "file", filename)
    }
    
    return nil
}
```

## Configuration Validation

### Validator Interface

```go
type Validator struct {
    rules []ValidationRule
}

type ValidationRule interface {
    Validate(cfg *Config) error
    Name() string
}

func (v *Validator) Validate(cfg *Config) error {
    var errors []error
    
    for _, rule := range v.rules {
        if err := rule.Validate(cfg); err != nil {
            errors = append(errors, fmt.Errorf("%s: %w", rule.Name(), err))
        }
    }
    
    if len(errors) > 0 {
        return &ValidationErrors{Errors: errors}
    }
    
    return nil
}
```

### Security Validation

```go
type SecurityValidator struct{}

func (sv *SecurityValidator) Validate(cfg *Config) error {
    // Validate API keys are not empty
    for name, provider := range cfg.AI.Providers {
        if provider.Enabled && provider.APIKey == "" {
            return fmt.Errorf("provider %s is enabled but API key is missing", name)
        }
    }
    
    // Validate TLS configuration
    if cfg.Server.TLS.Enabled {
        if cfg.Server.TLS.CertFile == "" || cfg.Server.TLS.KeyFile == "" {
            return fmt.Errorf("TLS enabled but cert/key files not specified")
        }
    }
    
    // Validate database URL
    if !strings.HasPrefix(cfg.Database.URL, "postgres://") {
        return fmt.Errorf("database URL must use postgres:// scheme")
    }
    
    return nil
}

func (sv *SecurityValidator) Name() string {
    return "security"
}
```

### Performance Validation

```go
type PerformanceValidator struct{}

func (pv *PerformanceValidator) Validate(cfg *Config) error {
    // Validate connection pool settings
    if cfg.Database.MaxConnections < cfg.Database.MinConnections {
        return fmt.Errorf("max connections (%d) must be >= min connections (%d)",
            cfg.Database.MaxConnections, cfg.Database.MinConnections)
    }
    
    // Validate timeout settings
    if cfg.Server.ReadTimeout < time.Second {
        return fmt.Errorf("read timeout too short: %v", cfg.Server.ReadTimeout)
    }
    
    if cfg.Server.WriteTimeout < time.Second {
        return fmt.Errorf("write timeout too short: %v", cfg.Server.WriteTimeout)
    }
    
    return nil
}
```

## Environment Examples

### Development Configuration

```yaml
# configs/development.yaml
server:
  host: "localhost"
  port: 8080
  tls:
    enabled: false

database:
  max_connections: 10
  min_connections: 2

ai:
  providers:
    claude:
      enabled: true
      model: "claude-3-haiku-20240307"
      temperature: 0.0
      max_tokens: 1000
    gemini:
      enabled: false

observability:
  logging:
    level: "debug"
    format: "text"
  metrics:
    enabled: true
    port: 9090
  tracing:
    enabled: false
```

### Production Configuration

```yaml
# configs/production.yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"
  shutdown_timeout: "15s"
  tls:
    enabled: true

database:
  max_connections: 50
  min_connections: 10
  conn_max_lifetime: "2h"
  conn_max_idle_time: "30m"

ai:
  providers:
    claude:
      enabled: true
      model: "claude-3-sonnet-20240229"
      temperature: 0.7
      max_tokens: 4096
      rate_limit:
        requests_per_minute: 100
        burst: 20
    gemini:
      enabled: true
      model: "gemini-pro"
      temperature: 0.7

observability:
  logging:
    level: "info"
    format: "json"
    structured: true
  metrics:
    enabled: true
    port: 9090
  tracing:
    enabled: true
    endpoint: "http://jaeger:14268/api/traces"
```

### Environment Variables

```bash
# Production environment variables
export DATABASE_URL="postgres://user:pass@localhost:5432/assistant_prod"
export CLAUDE_API_KEY="your_claude_api_key"
export GEMINI_API_KEY="your_gemini_api_key"
export SERVER_TLS_CERT_FILE="/etc/ssl/certs/assistant.crt"
export SERVER_TLS_KEY_FILE="/etc/ssl/private/assistant.key"
export OBSERVABILITY_TRACING_ENDPOINT="http://jaeger:14268/api/traces"
```

## Usage Examples

### Basic Configuration Loading

```go
func main() {
    // Load configuration
    loader := config.NewLoader("production", "configs", "/etc/assistant")
    cfg, err := loader.Load()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }
    
    // Use configuration
    server := server.New(&cfg.Server)
    db, err := database.Connect(&cfg.Database)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
}
```

### Environment Detection

```go
func DetectEnvironment() string {
    if env := os.Getenv("ENVIRONMENT"); env != "" {
        return env
    }
    
    if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
        return "production"
    }
    
    if os.Getenv("CI") == "true" {
        return "test"
    }
    
    return "development"
}
```

### Configuration Hot Reload

```go
type ConfigWatcher struct {
    loader   *Loader
    current  *Config
    callback func(*Config)
    watcher  *fsnotify.Watcher
}

func (cw *ConfigWatcher) Watch() error {
    return cw.watcher.Add("configs/")
}

func (cw *ConfigWatcher) handleChange(event fsnotify.Event) {
    if event.Op&fsnotify.Write == fsnotify.Write {
        cfg, err := cw.loader.Load()
        if err != nil {
            log.Printf("Failed to reload config: %v", err)
            return
        }
        
        cw.current = cfg
        if cw.callback != nil {
            cw.callback(cfg)
        }
    }
}
```

## Intelligent Configuration

### Preference Learning

```go
type PreferenceTracker struct {
    config   *Config
    patterns map[string]interface{}
    storage  PreferenceStorage
}

func (pt *PreferenceTracker) TrackUsage(feature string, value interface{}) {
    pt.patterns[feature] = value
    pt.storage.Store(feature, value)
}

func (pt *PreferenceTracker) SuggestOptimizations() []Suggestion {
    var suggestions []Suggestion
    
    // Analyze usage patterns
    if pt.analyzeHighLatency() {
        suggestions = append(suggestions, Suggestion{
            Type:        "performance",
            Description: "Consider increasing connection pool size",
            Config:      "database.max_connections",
            Value:       pt.config.Database.MaxConnections * 2,
        })
    }
    
    return suggestions
}
```

### Context-Aware Defaults

```go
func (l *Loader) loadContextualDefaults(cfg *Config) error {
    context := l.detectContext()
    
    switch context.Type {
    case "development":
        cfg.Observability.Logging.Level = "debug"
        cfg.Database.MaxConnections = 10
        cfg.AI.Providers["claude"].Model = "claude-3-haiku-20240307" // Faster model
        
    case "production":
        cfg.Observability.Logging.Level = "info"
        cfg.Database.MaxConnections = 50
        cfg.Server.TLS.Enabled = true
        
    case "testing":
        cfg.Database.MaxConnections = 5
        cfg.AI.Providers["mock"].Enabled = true
    }
    
    return nil
}
```

## Testing

### Configuration Testing

```go
func TestConfigurationLoading(t *testing.T) {
    tests := []struct {
        name        string
        environment string
        envVars     map[string]string
        expectError bool
    }{
        {
            name:        "development config",
            environment: "development",
            envVars: map[string]string{
                "DATABASE_URL": "postgres://localhost/assistant_test",
                "CLAUDE_API_KEY": "test-key",
            },
            expectError: false,
        },
        {
            name:        "missing required config",
            environment: "production",
            envVars:     map[string]string{},
            expectError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            for key, value := range tt.envVars {
                t.Setenv(key, value)
            }
            
            loader := config.NewLoader(tt.environment, "testdata")
            cfg, err := loader.Load()
            
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, cfg)
            }
        })
    }
}
```

### Validation Testing

```go
func TestConfigValidation(t *testing.T) {
    validator := config.NewValidator()
    
    cfg := &config.Config{
        Database: config.DatabaseConfig{
            MaxConnections: 5,
            MinConnections: 10, // Invalid: min > max
        },
    }
    
    err := validator.Validate(cfg)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "max connections")
}
```

## Security Best Practices

### Secret Management

```go
type SecretManager struct {
    provider SecretProvider // Vault, AWS Secrets Manager, etc.
    cache    map[string]Secret
    ttl      time.Duration
}

func (sm *SecretManager) GetSecret(key string) (string, error) {
    if cached, found := sm.cache[key]; found && !cached.Expired() {
        return cached.Value, nil
    }
    
    secret, err := sm.provider.GetSecret(key)
    if err != nil {
        return "", fmt.Errorf("failed to get secret %s: %w", key, err)
    }
    
    sm.cache[key] = Secret{
        Value:     secret,
        ExpiresAt: time.Now().Add(sm.ttl),
    }
    
    return secret, nil
}
```

### Configuration Encryption

```go
type EncryptedConfig struct {
    data []byte
    key  []byte
}

func (ec *EncryptedConfig) Decrypt() (*Config, error) {
    plaintext, err := encrypt.Decrypt(ec.data, ec.key)
    if err != nil {
        return nil, fmt.Errorf("decryption failed: %w", err)
    }
    
    var cfg Config
    if err := json.Unmarshal(plaintext, &cfg); err != nil {
        return nil, fmt.Errorf("unmarshaling failed: %w", err)
    }
    
    return &cfg, nil
}
```

## Migration and Versioning

### Configuration Migration

```go
type ConfigMigration struct {
    Version int
    Up      func(*Config) error
    Down    func(*Config) error
}

var migrations = []ConfigMigration{
    {
        Version: 1,
        Up: func(cfg *Config) error {
            // Migrate old AI provider config to new format
            if cfg.AI.Claude.APIKey != "" {
                cfg.AI.Providers["claude"] = ProviderConfig{
                    APIKey: cfg.AI.Claude.APIKey,
                    Model:  cfg.AI.Claude.Model,
                }
                cfg.AI.Claude = ClaudeConfig{} // Clear old config
            }
            return nil
        },
    },
}
```

### Version Compatibility

```go
func (l *Loader) checkVersion(cfg *Config) error {
    if cfg.Version == 0 {
        cfg.Version = 1 // Default to version 1
    }
    
    if cfg.Version > CurrentConfigVersion {
        return fmt.Errorf("config version %d is newer than supported version %d",
            cfg.Version, CurrentConfigVersion)
    }
    
    return nil
}
```

## Performance Optimization

### Configuration Caching

```go
type CachedLoader struct {
    loader *Loader
    cache  *Config
    hash   string
    mutex  sync.RWMutex
}

func (cl *CachedLoader) Load() (*Config, error) {
    cl.mutex.RLock()
    currentHash := cl.calculateHash()
    if cl.cache != nil && cl.hash == currentHash {
        defer cl.mutex.RUnlock()
        return cl.cache, nil
    }
    cl.mutex.RUnlock()
    
    cl.mutex.Lock()
    defer cl.mutex.Unlock()
    
    cfg, err := cl.loader.Load()
    if err != nil {
        return nil, err
    }
    
    cl.cache = cfg
    cl.hash = currentHash
    return cfg, nil
}
```

## Related Documentation

- [AI Providers](../ai/README.md) - AI provider configuration
- [Storage](../storage/README.md) - Database configuration
- [Observability](../observability/README.md) - Monitoring configuration
- [Security Guidelines](../../docs/SECURITY.md) - Security best practices