# HTTP Server

## Overview

The Server package provides a high-performance, production-ready HTTP server for the Assistant intelligent development companion. It implements RESTful APIs, WebSocket support for real-time communication, comprehensive middleware chains, and intelligent request handling. The server is built on Go's standard library with carefully selected enhancements for observability, security, and scalability.

## Architecture

```
internal/server/
├── server.go       # Main server implementation and lifecycle
├── middleware.go   # Core middleware functions
├── middleware/     # Middleware implementations
├── handlers/       # HTTP request handlers
└── ws/            # WebSocket implementation
```

## Key Features

### 🚀 **High Performance**
- **Connection Pooling**: Optimized HTTP/2 and HTTP/1.1 support
- **Request Pipelining**: Efficient request processing
- **Zero-Allocation Routing**: Memory-efficient request routing
- **Graceful Shutdown**: Safe connection draining

### 🛡️ **Security Features**
- **TLS Configuration**: Modern cipher suites and protocols
- **CORS Management**: Flexible cross-origin resource sharing
- **Rate Limiting**: Per-client and global rate limits
- **Security Headers**: Comprehensive security header injection

### 📊 **Observability**
- **Request Tracing**: OpenTelemetry integration
- **Metrics Collection**: Prometheus-compatible metrics
- **Structured Logging**: Context-aware request logging
- **Health Checks**: Liveness and readiness probes

### 🔌 **Real-time Communication**
- **WebSocket Support**: Bi-directional streaming
- **Server-Sent Events**: Unidirectional event streams
- **Long Polling**: Fallback for restricted environments
- **Automatic Reconnection**: Client connection resilience

## Core Components

### Server Interface

```go
type Server struct {
    // Core components
    httpServer   *http.Server
    router       *Router
    middleware   *MiddlewareChain
    
    // WebSocket support
    wsHub        *websocket.Hub
    wsUpgrader   *websocket.Upgrader
    
    // Configuration
    config       ServerConfig
    tlsConfig    *tls.Config
    
    // Lifecycle management
    listener     net.Listener
    shutdown     chan struct{}
    wg           sync.WaitGroup
    
    // Observability
    logger       *slog.Logger
    metrics      *ServerMetrics
    tracer       trace.Tracer
}

type ServerConfig struct {
    // Network configuration
    Host              string        `yaml:"host" env:"HOST" default:"0.0.0.0"`
    Port              int           `yaml:"port" env:"PORT" default:"8080"`
    ReadTimeout       time.Duration `yaml:"read_timeout" default:"30s"`
    WriteTimeout      time.Duration `yaml:"write_timeout" default:"30s"`
    IdleTimeout       time.Duration `yaml:"idle_timeout" default:"120s"`
    ShutdownTimeout   time.Duration `yaml:"shutdown_timeout" default:"30s"`
    
    // HTTP/2 configuration
    EnableHTTP2       bool          `yaml:"enable_http2" default:"true"`
    MaxConcurrentStreams uint32     `yaml:"max_concurrent_streams" default:"1000"`
    
    // TLS configuration
    TLS               TLSConfig     `yaml:"tls"`
    
    // Request limits
    MaxHeaderBytes    int           `yaml:"max_header_bytes" default:"1048576"`
    MaxRequestBodySize int64        `yaml:"max_request_body_size" default:"10485760"`
    
    // WebSocket configuration
    WebSocket         WebSocketConfig `yaml:"websocket"`
}
```

### Server Implementation

```go
func NewServer(config ServerConfig, assistant *assistant.Assistant) (*Server, error) {
    // Initialize router
    router := NewRouter()
    
    // Create server instance
    srv := &Server{
        config:   config,
        router:   router,
        shutdown: make(chan struct{}),
        logger:   slog.With("component", "server"),
        metrics:  NewServerMetrics(),
        tracer:   otel.Tracer("server"),
    }
    
    // Configure TLS if enabled
    if config.TLS.Enabled {
        tlsConfig, err := srv.configureTLS()
        if err != nil {
            return nil, fmt.Errorf("configuring TLS: %w", err)
        }
        srv.tlsConfig = tlsConfig
    }
    
    // Initialize WebSocket hub
    srv.wsHub = websocket.NewHub(config.WebSocket)
    srv.wsUpgrader = &websocket.Upgrader{
        CheckOrigin: srv.checkOrigin,
        ReadBufferSize:  config.WebSocket.ReadBufferSize,
        WriteBufferSize: config.WebSocket.WriteBufferSize,
    }
    
    // Build middleware chain
    srv.middleware = srv.buildMiddlewareChain()
    
    // Register routes
    srv.registerRoutes(assistant)
    
    // Create HTTP server
    srv.httpServer = &http.Server{
        Addr:           fmt.Sprintf("%s:%d", config.Host, config.Port),
        Handler:        srv.middleware.Then(srv.router),
        ReadTimeout:    config.ReadTimeout,
        WriteTimeout:   config.WriteTimeout,
        IdleTimeout:    config.IdleTimeout,
        MaxHeaderBytes: config.MaxHeaderBytes,
        TLSConfig:      srv.tlsConfig,
        ErrorLog:       log.New(&slogWriter{logger: srv.logger}, "", 0),
    }
    
    // Configure HTTP/2
    if config.EnableHTTP2 {
        http2.ConfigureServer(srv.httpServer, &http2.Server{
            MaxConcurrentStreams: config.MaxConcurrentStreams,
        })
    }
    
    return srv, nil
}

func (s *Server) Start(ctx context.Context) error {
    // Create listener
    listener, err := net.Listen("tcp", s.httpServer.Addr)
    if err != nil {
        return fmt.Errorf("creating listener: %w", err)
    }
    s.listener = listener
    
    // Start WebSocket hub
    s.wg.Add(1)
    go func() {
        defer s.wg.Done()
        s.wsHub.Run(ctx)
    }()
    
    // Start metrics collection
    s.wg.Add(1)
    go func() {
        defer s.wg.Done()
        s.collectMetrics(ctx)
    }()
    
    // Start HTTP server
    s.logger.Info("Starting server",
        slog.String("address", s.httpServer.Addr),
        slog.Bool("tls", s.tlsConfig != nil),
        slog.Bool("http2", s.config.EnableHTTP2))
    
    if s.tlsConfig != nil {
        return s.httpServer.ServeTLS(listener, "", "")
    }
    
    return s.httpServer.Serve(listener)
}

func (s *Server) Shutdown(ctx context.Context) error {
    s.logger.Info("Shutting down server")
    
    // Signal shutdown
    close(s.shutdown)
    
    // Shutdown HTTP server
    shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
    defer cancel()
    
    if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
        return fmt.Errorf("shutting down HTTP server: %w", err)
    }
    
    // Wait for goroutines
    done := make(chan struct{})
    go func() {
        s.wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        s.logger.Info("Server shutdown complete")
        return nil
    case <-shutdownCtx.Done():
        return fmt.Errorf("shutdown timeout exceeded")
    }
}
```

## Middleware System

### Middleware Chain

```go
type MiddlewareChain struct {
    middlewares []Middleware
}

type Middleware func(http.Handler) http.Handler

func (mc *MiddlewareChain) Then(handler http.Handler) http.Handler {
    // Apply middlewares in reverse order
    for i := len(mc.middlewares) - 1; i >= 0; i-- {
        handler = mc.middlewares[i](handler)
    }
    return handler
}

func (s *Server) buildMiddlewareChain() *MiddlewareChain {
    return &MiddlewareChain{
        middlewares: []Middleware{
            // Recovery from panics
            RecoveryMiddleware(s.logger),
            
            // Request ID generation
            RequestIDMiddleware(),
            
            // Access logging
            LoggingMiddleware(s.logger),
            
            // Metrics collection
            MetricsMiddleware(s.metrics),
            
            // Distributed tracing
            TracingMiddleware(s.tracer),
            
            // Security headers
            SecurityHeadersMiddleware(s.config.Security),
            
            // CORS handling
            CORSMiddleware(s.config.CORS),
            
            // Rate limiting
            RateLimitMiddleware(s.rateLimiter),
            
            // Request body limiting
            BodyLimitMiddleware(s.config.MaxRequestBodySize),
            
            // Timeout handling
            TimeoutMiddleware(s.config.RequestTimeout),
            
            // Authentication
            AuthenticationMiddleware(s.authService),
            
            // Request validation
            ValidationMiddleware(),
        },
    }
}
```

### Core Middleware Implementations

#### Recovery Middleware

```go
func RecoveryMiddleware(logger *slog.Logger) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            defer func() {
                if err := recover(); err != nil {
                    // Log the panic
                    logger.Error("Panic recovered",
                        slog.Any("error", err),
                        slog.String("path", r.URL.Path),
                        slog.String("method", r.Method),
                        slog.String("stack", string(debug.Stack())))
                    
                    // Return 500 error
                    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                }
            }()
            
            next.ServeHTTP(w, r)
        })
    }
}
```

#### Logging Middleware

```go
func LoggingMiddleware(logger *slog.Logger) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // Wrap response writer to capture status code
            wrapped := &responseWriter{
                ResponseWriter: w,
                statusCode:     http.StatusOK,
            }
            
            // Process request
            next.ServeHTTP(wrapped, r)
            
            // Log request details
            duration := time.Since(start)
            logger.Info("HTTP request",
                slog.String("method", r.Method),
                slog.String("path", r.URL.Path),
                slog.Int("status", wrapped.statusCode),
                slog.Int64("bytes", wrapped.bytesWritten),
                slog.Duration("duration", duration),
                slog.String("ip", getClientIP(r)),
                slog.String("user_agent", r.UserAgent()),
                slog.String("request_id", GetRequestID(r.Context())))
        })
    }
}

type responseWriter struct {
    http.ResponseWriter
    statusCode    int
    bytesWritten  int64
    wroteHeader   bool
}

func (rw *responseWriter) WriteHeader(code int) {
    if !rw.wroteHeader {
        rw.statusCode = code
        rw.ResponseWriter.WriteHeader(code)
        rw.wroteHeader = true
    }
}

func (rw *responseWriter) Write(b []byte) (int, error) {
    if !rw.wroteHeader {
        rw.WriteHeader(http.StatusOK)
    }
    n, err := rw.ResponseWriter.Write(b)
    rw.bytesWritten += int64(n)
    return n, err
}
```

#### Rate Limiting Middleware

```go
func RateLimitMiddleware(limiter RateLimiter) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get client identifier
            clientID := getClientIdentifier(r)
            
            // Check rate limit
            allowed, err := limiter.Allow(r.Context(), clientID)
            if err != nil {
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                return
            }
            
            if !allowed {
                // Get rate limit info
                info := limiter.GetInfo(clientID)
                
                // Set rate limit headers
                w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(info.Limit, 10))
                w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(info.Remaining, 10))
                w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(info.ResetAt.Unix(), 10))
                w.Header().Set("Retry-After", strconv.FormatInt(info.RetryAfter, 10))
                
                http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

## Request Handlers

### Handler Registration

```go
func (s *Server) registerRoutes(assistant *assistant.Assistant) {
    // API version prefix
    api := s.router.PathPrefix("/api/v1").Subrouter()
    
    // Health checks
    s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
    s.router.HandleFunc("/ready", s.handleReady).Methods("GET")
    
    // Chat endpoints
    api.HandleFunc("/chat", s.handleChat(assistant)).Methods("POST")
    api.HandleFunc("/chat/stream", s.handleChatStream(assistant)).Methods("POST")
    
    // Conversation management
    api.HandleFunc("/conversations", s.handleListConversations()).Methods("GET")
    api.HandleFunc("/conversations", s.handleCreateConversation()).Methods("POST")
    api.HandleFunc("/conversations/{id}", s.handleGetConversation()).Methods("GET")
    api.HandleFunc("/conversations/{id}", s.handleUpdateConversation()).Methods("PUT")
    api.HandleFunc("/conversations/{id}", s.handleDeleteConversation()).Methods("DELETE")
    
    // Memory operations
    api.HandleFunc("/memory/search", s.handleMemorySearch()).Methods("POST")
    api.HandleFunc("/memory/store", s.handleMemoryStore()).Methods("POST")
    
    // Tool operations
    api.HandleFunc("/tools", s.handleListTools()).Methods("GET")
    api.HandleFunc("/tools/{name}/execute", s.handleExecuteTool()).Methods("POST")
    
    // WebSocket endpoint
    s.router.HandleFunc("/ws", s.handleWebSocket()).Methods("GET")
    
    // Static files
    s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
}
```

### Chat Handler

```go
func (s *Server) handleChat(assistant *assistant.Assistant) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        // Parse request
        var req ChatRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            s.respondError(w, fmt.Errorf("invalid request: %w", err), http.StatusBadRequest)
            return
        }
        
        // Validate request
        if err := req.Validate(); err != nil {
            s.respondError(w, err, http.StatusBadRequest)
            return
        }
        
        // Get user context
        userID := GetUserID(ctx)
        
        // Process chat request
        response, err := assistant.ProcessChat(ctx, &assistant.ChatRequest{
            UserID:         userID,
            ConversationID: req.ConversationID,
            Message:        req.Message,
            Model:          req.Model,
            Temperature:    req.Temperature,
            MaxTokens:      req.MaxTokens,
        })
        
        if err != nil {
            s.respondError(w, err, http.StatusInternalServerError)
            return
        }
        
        // Send response
        s.respondJSON(w, ChatResponse{
            Message:        response.Message,
            ConversationID: response.ConversationID,
            MessageID:      response.MessageID,
            Model:          response.Model,
            TokensUsed:     response.TokensUsed,
        })
    }
}
```

### Streaming Handler

```go
func (s *Server) handleChatStream(assistant *assistant.Assistant) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        // Parse request
        var req ChatRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            s.respondError(w, fmt.Errorf("invalid request: %w", err), http.StatusBadRequest)
            return
        }
        
        // Set streaming headers
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("Connection", "keep-alive")
        w.Header().Set("X-Accel-Buffering", "no")
        
        // Create flusher
        flusher, ok := w.(http.Flusher)
        if !ok {
            s.respondError(w, errors.New("streaming not supported"), http.StatusInternalServerError)
            return
        }
        
        // Process with streaming
        stream, err := assistant.ProcessChatStream(ctx, &assistant.ChatRequest{
            UserID:         GetUserID(ctx),
            ConversationID: req.ConversationID,
            Message:        req.Message,
        })
        
        if err != nil {
            s.sendSSEError(w, flusher, err)
            return
        }
        
        // Stream responses
        for response := range stream {
            if response.Error != nil {
                s.sendSSEError(w, flusher, response.Error)
                return
            }
            
            // Send SSE event
            event := SSEEvent{
                Event: "message",
                Data:  response,
            }
            
            if err := s.sendSSE(w, flusher, event); err != nil {
                s.logger.Error("Failed to send SSE", slog.String("error", err.Error()))
                return
            }
        }
        
        // Send completion event
        s.sendSSE(w, flusher, SSEEvent{Event: "done"})
    }
}

func (s *Server) sendSSE(w http.ResponseWriter, flusher http.Flusher, event SSEEvent) error {
    data, err := json.Marshal(event.Data)
    if err != nil {
        return fmt.Errorf("marshaling event data: %w", err)
    }
    
    fmt.Fprintf(w, "event: %s\n", event.Event)
    fmt.Fprintf(w, "data: %s\n\n", data)
    flusher.Flush()
    
    return nil
}
```

## WebSocket Support

### WebSocket Hub

```go
type Hub struct {
    // Client management
    clients    map[*Client]bool
    register   chan *Client
    unregister chan *Client
    
    // Message broadcasting
    broadcast  chan Message
    
    // Configuration
    config     WebSocketConfig
    
    // Lifecycle
    ctx        context.Context
    cancel     context.CancelFunc
}

func (h *Hub) Run(ctx context.Context) {
    h.ctx, h.cancel = context.WithCancel(ctx)
    
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
            h.logger.Info("Client connected", 
                slog.String("id", client.ID),
                slog.String("ip", client.IP))
            
        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
                h.logger.Info("Client disconnected",
                    slog.String("id", client.ID))
            }
            
        case message := <-h.broadcast:
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    // Client buffer full, close connection
                    delete(h.clients, client)
                    close(client.send)
                }
            }
            
        case <-h.ctx.Done():
            // Close all client connections
            for client := range h.clients {
                close(client.send)
            }
            return
        }
    }
}
```

### WebSocket Client

```go
type Client struct {
    ID       string
    IP       string
    hub      *Hub
    conn     *websocket.Conn
    send     chan Message
    
    // Rate limiting
    limiter  *rate.Limiter
    
    // Context
    ctx      context.Context
    cancel   context.CancelFunc
}

func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()
    
    c.conn.SetReadLimit(c.hub.config.MaxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(c.hub.config.PongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(c.hub.config.PongWait))
        return nil
    })
    
    for {
        var message Message
        err := c.conn.ReadJSON(&message)
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, 
                websocket.CloseGoingAway, 
                websocket.CloseAbnormalClosure) {
                c.hub.logger.Error("WebSocket error",
                    slog.String("error", err.Error()))
            }
            break
        }
        
        // Rate limit check
        if !c.limiter.Allow() {
            c.sendError("Rate limit exceeded")
            continue
        }
        
        // Process message
        c.processMessage(message)
    }
}

func (c *Client) writePump() {
    ticker := time.NewTicker(c.hub.config.PingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()
    
    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(c.hub.config.WriteWait))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            
            if err := c.conn.WriteJSON(message); err != nil {
                return
            }
            
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(c.hub.config.WriteWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
            
        case <-c.ctx.Done():
            return
        }
    }
}
```

## Security

### TLS Configuration

```go
func (s *Server) configureTLS() (*tls.Config, error) {
    // Load certificates
    cert, err := tls.LoadX509KeyPair(s.config.TLS.CertFile, s.config.TLS.KeyFile)
    if err != nil {
        return nil, fmt.Errorf("loading certificates: %w", err)
    }
    
    // Modern TLS configuration
    return &tls.Config{
        Certificates: []tls.Certificate{cert},
        MinVersion:   tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
            tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
            tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
        PreferServerCipherSuites: true,
        CurvePreferences: []tls.CurveID{
            tls.CurveP256,
            tls.X25519,
        },
    }, nil
}
```

### Security Headers

```go
func SecurityHeadersMiddleware(config SecurityConfig) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // HSTS
            if config.HSTS.Enabled {
                value := fmt.Sprintf("max-age=%d", config.HSTS.MaxAge)
                if config.HSTS.IncludeSubdomains {
                    value += "; includeSubDomains"
                }
                if config.HSTS.Preload {
                    value += "; preload"
                }
                w.Header().Set("Strict-Transport-Security", value)
            }
            
            // Content Security Policy
            if config.CSP != "" {
                w.Header().Set("Content-Security-Policy", config.CSP)
            }
            
            // Other security headers
            w.Header().Set("X-Content-Type-Options", "nosniff")
            w.Header().Set("X-Frame-Options", "DENY")
            w.Header().Set("X-XSS-Protection", "1; mode=block")
            w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
            w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
            
            next.ServeHTTP(w, r)
        })
    }
}
```

## Configuration

### Server Configuration

```yaml
server:
  # Network settings
  host: "0.0.0.0"
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"
  shutdown_timeout: "30s"
  
  # HTTP/2 settings
  enable_http2: true
  max_concurrent_streams: 1000
  
  # Request limits
  max_header_bytes: 1048576
  max_request_body_size: 10485760
  
  # TLS configuration
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/server.crt"
    key_file: "/etc/ssl/private/server.key"
    min_version: "1.2"
    
  # CORS configuration
  cors:
    allowed_origins:
      - "https://example.com"
      - "https://app.example.com"
    allowed_methods:
      - "GET"
      - "POST"
      - "PUT"
      - "DELETE"
      - "OPTIONS"
    allowed_headers:
      - "Content-Type"
      - "Authorization"
      - "X-Request-ID"
    exposed_headers:
      - "X-Request-ID"
    allow_credentials: true
    max_age: 86400
    
  # WebSocket configuration
  websocket:
    read_buffer_size: 1024
    write_buffer_size: 1024
    max_message_size: 65536
    ping_period: "54s"
    pong_wait: "60s"
    write_wait: "10s"
    
  # Security settings
  security:
    hsts:
      enabled: true
      max_age: 31536000
      include_subdomains: true
      preload: true
    csp: "default-src 'self'; script-src 'self' 'unsafe-inline';"
    
  # Rate limiting
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst: 10
    
  # Metrics
  metrics:
    enabled: true
    path: "/metrics"
    port: 9090
```

## Metrics and Monitoring

### Server Metrics

```go
type ServerMetrics struct {
    // HTTP metrics
    requestsTotal    *prometheus.CounterVec
    requestDuration  *prometheus.HistogramVec
    requestSize      *prometheus.HistogramVec
    responseSize     *prometheus.HistogramVec
    
    // WebSocket metrics
    wsConnections    prometheus.Gauge
    wsMessages       *prometheus.CounterVec
    
    // System metrics
    goroutines       prometheus.Gauge
    memoryUsage      prometheus.Gauge
}

func (sm *ServerMetrics) RecordHTTPRequest(method, path string, status int, duration time.Duration) {
    sm.requestsTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
    sm.requestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}
```

### Health Checks

```go
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    health := HealthStatus{
        Status:    "healthy",
        Timestamp: time.Now(),
        Checks:    make(map[string]CheckResult),
    }
    
    // Check database
    dbCheck := s.checkDatabase(r.Context())
    health.Checks["database"] = dbCheck
    
    // Check dependencies
    for name, checker := range s.healthCheckers {
        health.Checks[name] = checker.Check(r.Context())
    }
    
    // Determine overall status
    for _, check := range health.Checks {
        if !check.Healthy {
            health.Status = "unhealthy"
            w.WriteHeader(http.StatusServiceUnavailable)
            break
        }
    }
    
    s.respondJSON(w, health)
}
```

## Usage Examples

### Starting the Server

```go
func main() {
    // Load configuration
    config := server.LoadConfig()
    
    // Initialize assistant
    assistant, err := assistant.New(assistantConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create server
    srv, err := server.NewServer(config, assistant)
    if err != nil {
        log.Fatal(err)
    }
    
    // Handle graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        cancel()
    }()
    
    // Start server
    if err := srv.Start(ctx); err != nil && err != http.ErrServerClosed {
        log.Fatal(err)
    }
    
    // Shutdown
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()
    
    if err := srv.Shutdown(shutdownCtx); err != nil {
        log.Fatal(err)
    }
}
```

### Custom Handler

```go
func (s *Server) handleCustomEndpoint() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        // Extract request ID for tracing
        requestID := GetRequestID(ctx)
        
        // Start span for tracing
        ctx, span := s.tracer.Start(ctx, "custom_endpoint")
        defer span.End()
        
        // Process request
        result, err := s.processCustomRequest(ctx, r)
        if err != nil {
            span.RecordError(err)
            s.respondError(w, err, http.StatusInternalServerError)
            return
        }
        
        // Record metrics
        s.metrics.RecordCustomOperation(result.Type, time.Since(start))
        
        // Send response
        s.respondJSON(w, result)
    }
}
```

## Related Documentation

- [API Handlers](../api/README.md) - API endpoint implementations
- [Middleware](middleware/README.md) - Middleware components
- [WebSocket](ws/README.md) - WebSocket implementation
- [Configuration](../config/README.md) - Server configuration