# Configuration Files

This directory contains configuration files for different environments and deployment scenarios.

## Structure

```
configs/
├── development.yaml    # Development environment configuration
└── production.yaml     # Production environment configuration
```

## Configuration Files

### development.yaml
Development environment settings:
- Debug logging enabled
- Local database connections
- Relaxed rate limits
- Development-friendly timeouts

### production.yaml  
Production environment settings:
- Structured logging
- Production database connections
- Appropriate rate limits
- Production-optimized timeouts

## Environment Variables

Configuration supports environment variable overrides following the 12-factor app methodology:

### Required Variables
- `CLAUDE_API_KEY` or `GEMINI_API_KEY`: AI provider API key
- `DATABASE_URL`: PostgreSQL connection string

### Optional Variables
- `PORT`: Server port (default: 8080)
- `LOG_LEVEL`: Logging level (default: info)
- `WORKSPACE_PATH`: Default workspace path
- `RATE_LIMIT_REQUESTS`: Requests per hour limit
- `RATE_LIMIT_TOKENS`: Tokens per hour limit

## Usage

The application automatically selects configuration based on the environment:

```bash
# Development (uses development.yaml)
go run ./cmd/assistant

# Production (uses production.yaml)
APP_ENV=production go run ./cmd/assistant

# Custom configuration file
CONFIG_FILE=custom.yaml go run ./cmd/assistant
```

## Configuration Validation

All configuration files are validated at startup to ensure:
- Required fields are present
- Values are within acceptable ranges
- File paths exist and are accessible
- Database connections can be established