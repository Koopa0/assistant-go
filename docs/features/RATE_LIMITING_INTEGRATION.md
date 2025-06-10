# Rate Limiting Integration

## Overview

The rate limiting middleware has been integrated into the Assistant HTTP server to protect API endpoints from abuse and ensure fair resource usage. The implementation uses the `withMiddleware` method in the server package to apply rate limiting to all HTTP requests.

## Architecture

### Middleware Stack

The `withMiddleware` method in `/internal/platform/server/server.go` applies middleware in the following order (outermost to innermost):

1. **Recovery Middleware** - Catches panics and returns 500 errors
2. **Logging Middleware** - Logs all HTTP requests and responses
3. **CORS Middleware** - Handles Cross-Origin Resource Sharing
4. **Request ID Middleware** - Adds unique request IDs
5. **Rate Limiting Middleware** - Enforces request rate limits

### Rate Limiting Configuration

Rate limiting is configured via the `ServerConfig` structure:

```yaml
server:
  rate_limit:
    enabled: true                # Enable/disable rate limiting
    requests_per_second: 10      # Global rate limit
    burst_size: 20              # Burst capacity
```

### Implementation Details

1. **Initialization**: The rate limiter is initialized lazily when first needed via `initRateLimiter()`

2. **Key Generation**: Rate limit keys are generated based on:
   - User ID (if authenticated)
   - IP address (fallback for unauthenticated requests)
   - Endpoint path (for endpoint-specific limits)

3. **Endpoint-Specific Limits**: High-cost endpoints have stricter limits:
   - `/api/v1/chat`: 30 requests/minute
   - `/api/v1/langchain`: 20 requests/minute
   - `/api/v1/tools`: 60 requests/minute

4. **Response Headers**: Rate limit information is included in response headers:
   - `X-RateLimit-Limit`: Total request limit
   - `X-RateLimit-Remaining`: Remaining requests
   - `X-RateLimit-Reset`: Unix timestamp when limit resets
   - `Retry-After`: Seconds until next request allowed (on 429 responses)

## Usage

### Enabling Rate Limiting

Rate limiting is enabled by default. To disable it, set `rate_limit.enabled: false` in your configuration.

### Customizing Limits

1. **Global Limits**: Modify `requests_per_second` and `burst_size` in configuration
2. **Endpoint Limits**: Update the endpoint-specific limits in `initRateLimiter()`
3. **User-Based Limits**: Implement custom `KeyExtractor` function

### Testing Rate Limits

```bash
# Test rate limiting with curl
for i in {1..15}; do
  curl -i http://localhost:8080/api/health
  echo "Request $i"
done
```

You should see 429 responses after exceeding the burst capacity.

## Error Handling

When rate limits are exceeded:

1. HTTP 429 (Too Many Requests) status code is returned
2. JSON error response with retry information:
   ```json
   {
     "error": "rate_limit_exceeded",
     "message": "Too many requests. Please retry after some time.",
     "retry_after": 60,
     "reset_time": 1706234567
   }
   ```

## Monitoring

Rate limit metrics are logged at various levels:
- **Debug**: Successful rate limit checks
- **Warn**: Rate limit exceeded events
- **Info**: Rate limiter initialization

## Future Enhancements

1. **Distributed Rate Limiting**: Redis-backed rate limiting for multi-instance deployments
2. **API Key Tiers**: Different rate limits based on API key tiers
3. **Dynamic Limits**: Adjust limits based on system load
4. **Webhook Notifications**: Alert when rate limits are frequently exceeded

## Related Documentation

- [Rate Limiting Package](/internal/platform/ratelimit/README.md)
- [Server Configuration](/internal/config/README.md)
- [API Documentation](/docs/API_REFERENCE.md)