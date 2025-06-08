// Package config provides application configuration management with support for
// environment variables, YAML files, and validation.
package config

import (
	"time"
)

// Config represents the application configuration
type Config struct {
	Mode      string         `yaml:"mode" env:"APP_MODE" default:"development"`
	LogLevel  string         `yaml:"log_level" env:"LOG_LEVEL" default:"info"`
	LogFormat string         `yaml:"log_format" env:"LOG_FORMAT" default:"json"`
	Database  DatabaseConfig `yaml:"database"`
	Server    ServerConfig   `yaml:"server"`
	CLI       CLIConfig      `yaml:"cli"`
	AI        AIConfig       `yaml:"ai"`
	Tools     ToolsConfig    `yaml:"tools"`
	Security  SecurityConfig `yaml:"security"`
}

// DatabaseConfig holds PostgreSQL configuration optimized for PostgreSQL 17
type DatabaseConfig struct {
	URL string `yaml:"url" env:"DATABASE_URL" required:"true"`
	// PostgreSQL 17 optimized connection pool settings (based on golang_guide.md)
	MaxConnections int           `yaml:"max_connections" env:"DB_MAX_CONNECTIONS" default:"30"` // Based on CPU cores
	MinConnections int           `yaml:"min_connections" env:"DB_MIN_CONNECTIONS" default:"5"`  // Keep minimum connections
	MaxIdleTime    time.Duration `yaml:"max_idle_time" env:"DB_MAX_IDLE_TIME" default:"15m"`    // Close idle connections
	MaxLifetime    time.Duration `yaml:"max_lifetime" env:"DB_MAX_LIFETIME" default:"1h"`       // Connection rotation
	ConnectTimeout time.Duration `yaml:"connect_timeout" env:"DB_CONNECT_TIMEOUT" default:"10s"`
	MigrationsPath string        `yaml:"migrations_path" env:"DB_MIGRATIONS_PATH" default:"internal/storage/postgres/migrations"`
	EnableLogging  bool          `yaml:"enable_logging" env:"DB_ENABLE_LOGGING" default:"false"`
}

// ServerConfig holds HTTP API server configuration
type ServerConfig struct {
	Address         string          `yaml:"address" env:"SERVER_ADDRESS" default:":8080"`
	ReadTimeout     time.Duration   `yaml:"read_timeout" env:"SERVER_READ_TIMEOUT" default:"10s"`
	WriteTimeout    time.Duration   `yaml:"write_timeout" env:"SERVER_WRITE_TIMEOUT" default:"10s"`
	IdleTimeout     time.Duration   `yaml:"idle_timeout" env:"SERVER_IDLE_TIMEOUT" default:"60s"`
	ShutdownTimeout time.Duration   `yaml:"shutdown_timeout" env:"SERVER_SHUTDOWN_TIMEOUT" default:"30s"`
	EnableTLS       bool            `yaml:"enable_tls" env:"SERVER_ENABLE_TLS" default:"false"`
	TLSCertFile     string          `yaml:"tls_cert_file" env:"SERVER_TLS_CERT_FILE"`
	TLSKeyFile      string          `yaml:"tls_key_file" env:"SERVER_TLS_KEY_FILE"`
	Security        SecurityConfig  `yaml:"security"`
	RateLimit       RateLimitConfig `yaml:"rate_limit"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled" env:"RATE_LIMIT_ENABLED" default:"true"`
	RequestsPerSecond int  `yaml:"requests_per_second" env:"RATE_LIMIT_RPS" default:"10"`
	BurstSize         int  `yaml:"burst_size" env:"RATE_LIMIT_BURST" default:"20"`
}

// CLIConfig holds CLI-specific configuration
type CLIConfig struct {
	HistoryFile    string `yaml:"history_file" env:"CLI_HISTORY_FILE" default:".assistant_history"`
	MaxHistorySize int    `yaml:"max_history_size" env:"CLI_MAX_HISTORY_SIZE" default:"1000"`
	EnableColors   bool   `yaml:"enable_colors" env:"CLI_ENABLE_COLORS" default:"true"`
	PromptTemplate string `yaml:"prompt_template" env:"CLI_PROMPT_TEMPLATE" default:"assistant> "`

	// Streaming configuration
	EnableStreaming  bool `yaml:"enable_streaming" env:"CLI_ENABLE_STREAMING" default:"true"`
	StreamBufferSize int  `yaml:"stream_buffer_size" env:"CLI_STREAM_BUFFER_SIZE" default:"1024"`

	// Display options
	ShowExecutionTime bool `yaml:"show_execution_time" env:"CLI_SHOW_EXECUTION_TIME" default:"true"`
	ShowTokenUsage    bool `yaml:"show_token_usage" env:"CLI_SHOW_TOKEN_USAGE" default:"true"`
}

// AIConfig holds AI provider configuration
type AIConfig struct {
	DefaultProvider string    `yaml:"default_provider" env:"AI_DEFAULT_PROVIDER" default:"claude"`
	Claude          Claude    `yaml:"claude"`
	Gemini          Gemini    `yaml:"gemini"`
	Embeddings      Embedding `yaml:"embeddings"`
}

// Claude holds Claude-specific configuration
type Claude struct {
	APIKey      string  `yaml:"api_key" env:"CLAUDE_API_KEY"`
	Model       string  `yaml:"model" env:"CLAUDE_MODEL" default:"claude-3-sonnet-20240229"`
	MaxTokens   int     `yaml:"max_tokens" env:"CLAUDE_MAX_TOKENS" default:"4096"`
	Temperature float64 `yaml:"temperature" env:"CLAUDE_TEMPERATURE" default:"0.7"`
	BaseURL     string  `yaml:"base_url" env:"CLAUDE_BASE_URL" default:"https://api.anthropic.com"`
}

// Gemini holds Gemini-specific configuration
type Gemini struct {
	APIKey      string  `yaml:"api_key" env:"GEMINI_API_KEY"`
	Model       string  `yaml:"model" env:"GEMINI_MODEL" default:"gemini-pro"`
	MaxTokens   int     `yaml:"max_tokens" env:"GEMINI_MAX_TOKENS" default:"4096"`
	Temperature float64 `yaml:"temperature" env:"GEMINI_TEMPERATURE" default:"0.7"`
	BaseURL     string  `yaml:"base_url" env:"GEMINI_BASE_URL" default:"https://generativelanguage.googleapis.com"`
}

// Embedding holds embedding service configuration
type Embedding struct {
	Provider   string `yaml:"provider" env:"EMBEDDING_PROVIDER" default:"claude"`
	Model      string `yaml:"model" env:"EMBEDDING_MODEL" default:"text-embedding-ada-002"`
	Dimensions int    `yaml:"dimensions" env:"EMBEDDING_DIMENSIONS" default:"1536"`
}

// ToolsConfig holds tool-specific configuration
type ToolsConfig struct {
	Search     Search     `yaml:"search"`
	Postgres   Postgres   `yaml:"postgres"`
	Kubernetes Kubernetes `yaml:"kubernetes"`
	Docker     Docker     `yaml:"docker"`
	Cloudflare Cloudflare `yaml:"cloudflare"`
	LangChain  LangChain  `yaml:"langchain"`
}

// Search holds search tool configuration
type Search struct {
	SearXNGURL  string        `yaml:"searxng_url" env:"SEARXNG_URL" default:"http://localhost:8888"`
	Timeout     time.Duration `yaml:"timeout" env:"SEARCH_TIMEOUT" default:"30s"`
	MaxResults  int           `yaml:"max_results" env:"SEARCH_MAX_RESULTS" default:"10"`
	EnableCache bool          `yaml:"enable_cache" env:"SEARCH_ENABLE_CACHE" default:"true"`
	CacheTTL    time.Duration `yaml:"cache_ttl" env:"SEARCH_CACHE_TTL" default:"1h"`
}

// Postgres holds PostgreSQL tool configuration
type Postgres struct {
	DefaultConnection string        `yaml:"default_connection" env:"POSTGRES_DEFAULT_CONNECTION"`
	QueryTimeout      time.Duration `yaml:"query_timeout" env:"POSTGRES_QUERY_TIMEOUT" default:"30s"`
	MaxQuerySize      int           `yaml:"max_query_size" env:"POSTGRES_MAX_QUERY_SIZE" default:"1048576"` // 1MB
	EnableExplain     bool          `yaml:"enable_explain" env:"POSTGRES_ENABLE_EXPLAIN" default:"true"`
}

// Kubernetes holds Kubernetes tool configuration
type Kubernetes struct {
	ConfigPath    string        `yaml:"config_path" env:"KUBECONFIG"`
	Context       string        `yaml:"context" env:"KUBE_CONTEXT"`
	Namespace     string        `yaml:"namespace" env:"KUBE_NAMESPACE" default:"default"`
	Timeout       time.Duration `yaml:"timeout" env:"KUBE_TIMEOUT" default:"30s"`
	EnableMetrics bool          `yaml:"enable_metrics" env:"KUBE_ENABLE_METRICS" default:"true"`
}

// Docker holds Docker tool configuration
type Docker struct {
	Host       string        `yaml:"host" env:"DOCKER_HOST" default:"unix:///var/run/docker.sock"`
	APIVersion string        `yaml:"api_version" env:"DOCKER_API_VERSION" default:"1.41"`
	Timeout    time.Duration `yaml:"timeout" env:"DOCKER_TIMEOUT" default:"30s"`
	TLSVerify  bool          `yaml:"tls_verify" env:"DOCKER_TLS_VERIFY" default:"false"`
}

// Cloudflare holds Cloudflare tool configuration
type Cloudflare struct {
	APIToken  string `yaml:"api_token" env:"CLOUDFLARE_API_TOKEN"`
	APIKey    string `yaml:"api_key" env:"CLOUDFLARE_API_KEY"`
	Email     string `yaml:"email" env:"CLOUDFLARE_EMAIL"`
	AccountID string `yaml:"account_id" env:"CLOUDFLARE_ACCOUNT_ID"`
	ZoneID    string `yaml:"zone_id" env:"CLOUDFLARE_ZONE_ID"`
}

// LangChain holds LangChain tool configuration
type LangChain struct {
	EnableMemory  bool          `yaml:"enable_memory" env:"LANGCHAIN_ENABLE_MEMORY" default:"true"`
	MemorySize    int           `yaml:"memory_size" env:"LANGCHAIN_MEMORY_SIZE" default:"10"`
	MaxIterations int           `yaml:"max_iterations" env:"LANGCHAIN_MAX_ITERATIONS" default:"5"`
	Timeout       time.Duration `yaml:"timeout" env:"LANGCHAIN_TIMEOUT" default:"60s"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	JWTSecret      string        `yaml:"jwt_secret" env:"JWT_SECRET"`
	JWTExpiration  time.Duration `yaml:"jwt_expiration" env:"JWT_EXPIRATION" default:"24h"`
	RateLimitRPS   int           `yaml:"rate_limit_rps" env:"RATE_LIMIT_RPS" default:"100"`
	RateLimitBurst int           `yaml:"rate_limit_burst" env:"RATE_LIMIT_BURST" default:"200"`
	EnableCORS     bool          `yaml:"enable_cors" env:"ENABLE_CORS" default:"true"`
	AllowedOrigins []string      `yaml:"allowed_origins" env:"ALLOWED_ORIGINS"`
}

// String returns a string representation of the Config, with sensitive data masked
func (c Config) String() string {
	return "<Config with masked sensitive data>"
}

// Validate validates the AI configuration
func (cfg AIConfig) Validate() error {
	return validateAI(cfg)
}
