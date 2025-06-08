package event

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ValidationMiddleware validates events before processing
type ValidationMiddleware struct {
	validators []EventValidator
	logger     *slog.Logger
}

// EventValidator validates events
type EventValidator interface {
	Validate(event Event) error
	GetType() ValidationType
}

// ValidationType defines types of validation
type ValidationType string

const (
	ValidationRequired ValidationType = "required"
	ValidationFormat   ValidationType = "format"
	ValidationSchema   ValidationType = "schema"
	ValidationBusiness ValidationType = "business"
)

// EnrichmentMiddleware enriches events with additional data
type EnrichmentMiddleware struct {
	enrichers []EventEnricher
	logger    *slog.Logger
}

// EventEnricher enriches events
type EventEnricher interface {
	Enrich(ctx context.Context, event *Event) error
	CanEnrich(eventType EventType) bool
	Priority() int
}

// RateLimitingMiddleware applies rate limiting to events
type RateLimitingMiddleware struct {
	limiters map[string]*RateLimiter
	config   RateLimitConfig
	mu       sync.RWMutex
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	tokens     int
	capacity   int
	refillRate int
	lastRefill time.Time
	mu         sync.Mutex
}

// RateLimitConfig configures rate limiting
type RateLimitConfig struct {
	Enabled      bool
	DefaultLimit int
	RefillRate   int
	WindowSize   time.Duration
	KeyExtractor func(Event) string
	Limits       map[string]int
}

// FilteringMiddleware filters events based on criteria
type FilteringMiddleware struct {
	filters []EventFilter
	mode    FilterMode
	logger  *slog.Logger
}

// FilterMode defines filtering behavior
type FilterMode string

const (
	FilterModeInclude FilterMode = "include" // Only include events matching filters
	FilterModeExclude FilterMode = "exclude" // Exclude events matching filters
)

// TransformationMiddleware transforms events
type TransformationMiddleware struct {
	transformers []EventTransformer
	logger       *slog.Logger
}

// EventTransformer transforms events
type EventTransformer interface {
	Transform(ctx context.Context, event *Event) error
	CanTransform(eventType EventType) bool
	Priority() int
}

// SecurityMiddleware applies security checks to events
type SecurityMiddleware struct {
	checks []SecurityCheck
	policy SecurityPolicy
	logger *slog.Logger
}

// SecurityCheck performs security validation
type SecurityCheck interface {
	Check(ctx context.Context, event Event) error
	GetType() SecurityCheckType
}

// SecurityCheckType defines types of security checks
type SecurityCheckType string

const (
	SecurityCheckAuthentication SecurityCheckType = "authentication"
	SecurityCheckAuthorization  SecurityCheckType = "authorization"
	SecurityCheckSanitization   SecurityCheckType = "sanitization"
	SecurityCheckEncryption     SecurityCheckType = "encryption"
)

// SecurityPolicy defines security policies
type SecurityPolicy struct {
	RequireAuthentication bool
	RequireEncryption     bool
	SensitiveEventTypes   []EventType
	AccessControl         map[EventType][]string
}

// CircuitBreakerMiddleware implements circuit breaker pattern
type CircuitBreakerMiddleware struct {
	breakers map[string]*CircuitBreaker
	config   CircuitBreakerConfig
	mu       sync.RWMutex
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	state        CircuitState
	failures     int
	lastFailure  time.Time
	successes    int
	timeout      time.Duration
	threshold    int
	resetTimeout time.Duration
	mu           sync.Mutex
}

// CircuitState defines circuit breaker states
type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half_open"
)

// CircuitBreakerConfig configures circuit breakers
type CircuitBreakerConfig struct {
	Enabled          bool
	FailureThreshold int
	ResetTimeout     time.Duration
	Timeout          time.Duration
	KeyExtractor     func(Event) string
}

// DeduplicationMiddleware removes duplicate events
type DeduplicationMiddleware struct {
	cache  map[string]time.Time
	ttl    time.Duration
	hasher EventHasher
	mu     sync.RWMutex
}

// EventHasher generates hashes for events
type EventHasher interface {
	Hash(event Event) string
}

// NewValidationMiddleware creates validation middleware
func NewValidationMiddleware(validators []EventValidator, logger *slog.Logger) *ValidationMiddleware {
	return &ValidationMiddleware{
		validators: validators,
		logger:     logger,
	}
}

// Process validates events
func (vm *ValidationMiddleware) Process(ctx context.Context, event Event, next MiddlewareFunc) error {
	for _, validator := range vm.validators {
		if err := validator.Validate(event); err != nil {
			vm.logger.Warn("Event validation failed",
				slog.String("event_id", event.ID),
				slog.String("validator", string(validator.GetType())),
				slog.Any("error", err))
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	return next(ctx, event)
}

// Order returns middleware execution order
func (vm *ValidationMiddleware) Order() int {
	return 100 // Run early
}

// NewEnrichmentMiddleware creates enrichment middleware
func NewEnrichmentMiddleware(enrichers []EventEnricher, logger *slog.Logger) *EnrichmentMiddleware {
	return &EnrichmentMiddleware{
		enrichers: enrichers,
		logger:    logger,
	}
}

// Process enriches events
func (em *EnrichmentMiddleware) Process(ctx context.Context, event Event, next MiddlewareFunc) error {
	// Sort enrichers by priority
	enrichers := make([]EventEnricher, 0)
	for _, enricher := range em.enrichers {
		if enricher.CanEnrich(event.Type) {
			enrichers = append(enrichers, enricher)
		}
	}

	// Sort by priority
	for i := 0; i < len(enrichers); i++ {
		for j := i + 1; j < len(enrichers); j++ {
			if enrichers[i].Priority() > enrichers[j].Priority() {
				enrichers[i], enrichers[j] = enrichers[j], enrichers[i]
			}
		}
	}

	// Apply enrichers
	for _, enricher := range enrichers {
		if err := enricher.Enrich(ctx, &event); err != nil {
			em.logger.Warn("Event enrichment failed",
				slog.String("event_id", event.ID),
				slog.Any("error", err))
			// Continue with other enrichers
		}
	}

	return next(ctx, event)
}

// Order returns middleware execution order
func (em *EnrichmentMiddleware) Order() int {
	return 200 // Run after validation
}

// NewRateLimitingMiddleware creates rate limiting middleware
func NewRateLimitingMiddleware(config RateLimitConfig) *RateLimitingMiddleware {
	return &RateLimitingMiddleware{
		limiters: make(map[string]*RateLimiter),
		config:   config,
	}
}

// Process applies rate limiting to events
func (rlm *RateLimitingMiddleware) Process(ctx context.Context, event Event, next MiddlewareFunc) error {
	if !rlm.config.Enabled {
		return next(ctx, event)
	}

	key := rlm.config.KeyExtractor(event)
	if key == "" {
		key = "default"
	}

	limiter := rlm.getLimiter(key)
	if !limiter.Allow() {
		return fmt.Errorf("rate limit exceeded for key: %s", key)
	}

	return next(ctx, event)
}

// Order returns middleware execution order
func (rlm *RateLimitingMiddleware) Order() int {
	return 150 // Run early, after validation
}

// getLimiter gets or creates a rate limiter for a key
func (rlm *RateLimitingMiddleware) getLimiter(key string) *RateLimiter {
	rlm.mu.RLock()
	limiter, exists := rlm.limiters[key]
	rlm.mu.RUnlock()

	if exists {
		return limiter
	}

	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := rlm.limiters[key]; exists {
		return limiter
	}

	// Get limit for this key
	limit := rlm.config.DefaultLimit
	if keyLimit, exists := rlm.config.Limits[key]; exists {
		limit = keyLimit
	}

	limiter = &RateLimiter{
		tokens:     limit,
		capacity:   limit,
		refillRate: rlm.config.RefillRate,
		lastRefill: time.Now(),
	}

	rlm.limiters[key] = limiter
	return limiter
}

// Allow checks if an operation is allowed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	// Refill tokens
	tokensToAdd := int(elapsed.Seconds()) * rl.refillRate
	rl.tokens += tokensToAdd
	if rl.tokens > rl.capacity {
		rl.tokens = rl.capacity
	}
	rl.lastRefill = now

	// Check if we have tokens
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// NewFilteringMiddleware creates filtering middleware
func NewFilteringMiddleware(filters []EventFilter, mode FilterMode, logger *slog.Logger) *FilteringMiddleware {
	return &FilteringMiddleware{
		filters: filters,
		mode:    mode,
		logger:  logger,
	}
}

// Process filters events
func (fm *FilteringMiddleware) Process(ctx context.Context, event Event, next MiddlewareFunc) error {
	matches := false
	for _, filter := range fm.filters {
		if fm.matchesFilter(event, filter) {
			matches = true
			break
		}
	}

	switch fm.mode {
	case FilterModeInclude:
		if !matches {
			fm.logger.Debug("Event filtered out (include mode)",
				slog.String("event_id", event.ID),
				slog.String("type", string(event.Type)))
			return nil // Don't process further
		}
	case FilterModeExclude:
		if matches {
			fm.logger.Debug("Event filtered out (exclude mode)",
				slog.String("event_id", event.ID),
				slog.String("type", string(event.Type)))
			return nil // Don't process further
		}
	}

	return next(ctx, event)
}

// Order returns middleware execution order
func (fm *FilteringMiddleware) Order() int {
	return 300 // Run after enrichment
}

// matchesFilter checks if an event matches a filter
func (fm *FilteringMiddleware) matchesFilter(event Event, filter EventFilter) bool {
	// Check event types
	if len(filter.EventTypes) > 0 {
		found := false
		for _, eventType := range filter.EventTypes {
			if event.Type == eventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check sources
	if len(filter.Sources) > 0 {
		found := false
		for _, source := range filter.Sources {
			if event.Source == source {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check priority
	if filter.Priority != "" && event.Priority != filter.Priority {
		return false
	}

	// Check custom predicate
	if filter.Predicate != nil {
		return filter.Predicate(event)
	}

	return true
}

// NewTransformationMiddleware creates transformation middleware
func NewTransformationMiddleware(transformers []EventTransformer, logger *slog.Logger) *TransformationMiddleware {
	return &TransformationMiddleware{
		transformers: transformers,
		logger:       logger,
	}
}

// Process transforms events
func (tm *TransformationMiddleware) Process(ctx context.Context, event Event, next MiddlewareFunc) error {
	// Sort transformers by priority
	transformers := make([]EventTransformer, 0)
	for _, transformer := range tm.transformers {
		if transformer.CanTransform(event.Type) {
			transformers = append(transformers, transformer)
		}
	}

	// Sort by priority
	for i := 0; i < len(transformers); i++ {
		for j := i + 1; j < len(transformers); j++ {
			if transformers[i].Priority() > transformers[j].Priority() {
				transformers[i], transformers[j] = transformers[j], transformers[i]
			}
		}
	}

	// Apply transformers
	for _, transformer := range transformers {
		if err := transformer.Transform(ctx, &event); err != nil {
			tm.logger.Warn("Event transformation failed",
				slog.String("event_id", event.ID),
				slog.Any("error", err))
			// Continue with other transformers
		}
	}

	return next(ctx, event)
}

// Order returns middleware execution order
func (tm *TransformationMiddleware) Order() int {
	return 400 // Run after filtering
}

// NewSecurityMiddleware creates security middleware
func NewSecurityMiddleware(checks []SecurityCheck, policy SecurityPolicy, logger *slog.Logger) *SecurityMiddleware {
	return &SecurityMiddleware{
		checks: checks,
		policy: policy,
		logger: logger,
	}
}

// Process applies security checks to events
func (sm *SecurityMiddleware) Process(ctx context.Context, event Event, next MiddlewareFunc) error {
	// Apply security checks
	for _, check := range sm.checks {
		if err := check.Check(ctx, event); err != nil {
			sm.logger.Warn("Security check failed",
				slog.String("event_id", event.ID),
				slog.String("check_type", string(check.GetType())),
				slog.Any("error", err))
			return fmt.Errorf("security check failed: %w", err)
		}
	}

	// Check if event type requires special handling
	for _, sensitiveType := range sm.policy.SensitiveEventTypes {
		if event.Type == sensitiveType {
			// Apply additional security measures
			if sm.policy.RequireEncryption && event.Metadata["encrypted"] != true {
				return fmt.Errorf("sensitive event type requires encryption")
			}
		}
	}

	return next(ctx, event)
}

// Order returns middleware execution order
func (sm *SecurityMiddleware) Order() int {
	return 50 // Run very early
}

// NewCircuitBreakerMiddleware creates circuit breaker middleware
func NewCircuitBreakerMiddleware(config CircuitBreakerConfig) *CircuitBreakerMiddleware {
	return &CircuitBreakerMiddleware{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
	}
}

// Process applies circuit breaker pattern to events
func (cbm *CircuitBreakerMiddleware) Process(ctx context.Context, event Event, next MiddlewareFunc) error {
	if !cbm.config.Enabled {
		return next(ctx, event)
	}

	key := cbm.config.KeyExtractor(event)
	if key == "" {
		key = "default"
	}

	breaker := cbm.getBreaker(key)

	if !breaker.Allow() {
		return fmt.Errorf("circuit breaker open for key: %s", key)
	}

	err := next(ctx, event)

	if err != nil {
		breaker.RecordFailure()
	} else {
		breaker.RecordSuccess()
	}

	return err
}

// Order returns middleware execution order
func (cbm *CircuitBreakerMiddleware) Order() int {
	return 500 // Run in middle of chain
}

// getBreaker gets or creates a circuit breaker for a key
func (cbm *CircuitBreakerMiddleware) getBreaker(key string) *CircuitBreaker {
	cbm.mu.RLock()
	breaker, exists := cbm.breakers[key]
	cbm.mu.RUnlock()

	if exists {
		return breaker
	}

	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	// Double-check after acquiring write lock
	if breaker, exists := cbm.breakers[key]; exists {
		return breaker
	}

	breaker = &CircuitBreaker{
		state:        CircuitClosed,
		threshold:    cbm.config.FailureThreshold,
		timeout:      cbm.config.Timeout,
		resetTimeout: cbm.config.ResetTimeout,
	}

	cbm.breakers[key] = breaker
	return breaker
}

// Allow checks if the circuit breaker allows the operation
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if now.Sub(cb.lastFailure) > cb.resetTimeout {
			cb.state = CircuitHalfOpen
			cb.successes = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	}

	return false
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0

	if cb.state == CircuitHalfOpen {
		cb.successes++
		if cb.successes >= cb.threshold {
			cb.state = CircuitClosed
		}
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.threshold {
		cb.state = CircuitOpen
	}
}

// NewDeduplicationMiddleware creates deduplication middleware
func NewDeduplicationMiddleware(ttl time.Duration, hasher EventHasher) *DeduplicationMiddleware {
	dm := &DeduplicationMiddleware{
		cache:  make(map[string]time.Time),
		ttl:    ttl,
		hasher: hasher,
	}

	// Start cleanup routine
	go dm.cleanup()

	return dm
}

// Process deduplicates events
func (dm *DeduplicationMiddleware) Process(ctx context.Context, event Event, next MiddlewareFunc) error {
	hash := dm.hasher.Hash(event)

	dm.mu.RLock()
	lastSeen, exists := dm.cache[hash]
	dm.mu.RUnlock()

	if exists && time.Since(lastSeen) < dm.ttl {
		// Duplicate event within TTL, skip processing
		return nil
	}

	dm.mu.Lock()
	dm.cache[hash] = time.Now()
	dm.mu.Unlock()

	return next(ctx, event)
}

// Order returns middleware execution order
func (dm *DeduplicationMiddleware) Order() int {
	return 250 // Run early, but after validation and rate limiting
}

// cleanup removes expired entries from the cache
func (dm *DeduplicationMiddleware) cleanup() {
	ticker := time.NewTicker(dm.ttl / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			dm.mu.Lock()
			for hash, timestamp := range dm.cache {
				if now.Sub(timestamp) > dm.ttl {
					delete(dm.cache, hash)
				}
			}
			dm.mu.Unlock()
		}
	}
}

// DefaultEventHasher provides a default implementation of EventHasher
type DefaultEventHasher struct{}

// Hash generates a hash for an event
func (deh *DefaultEventHasher) Hash(event Event) string {
	return fmt.Sprintf("%s_%s_%s_%d",
		event.Type,
		event.Source,
		event.Target,
		event.Timestamp.Unix())
}
