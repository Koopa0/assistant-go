# GoAssistant Operations Guide

Quick reference for using GoAssistant in your daily development workflow.

## Quick Start

### 1. Start Assistant
```bash
# Interactive mode (recommended for exploration)
./bin/assistant cli

# API mode (for integration)
./bin/assistant serve

# Direct query (for scripts)
./bin/assistant ask "your question"
```

### 2. Basic Operations

#### Get Help
```bash
# In CLI mode
> help

# Direct mode
./bin/assistant ask "what can you help me with?"
```

#### Check Status
```bash
# In CLI mode
> status

# API mode
curl http://localhost:8080/api/status
```

#### List Available Tools
```bash
# In CLI mode
> tools

# API mode
curl http://localhost:8080/api/tools
```

## Common Development Tasks

### Code Analysis & Review

#### Analyze Go Code
```bash
# CLI mode
> analyze my Go project structure
> review this function for performance issues

# API mode
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Analyze my Go code for potential improvements",
    "context": {
      "project_type": "go",
      "file_path": "./main.go"
    }
  }'
```

#### Code Quality Checks
```bash
# Ask for best practices
./bin/assistant ask "What are Go best practices for error handling?"

# Get code review
./bin/assistant ask "Review this Go function: $(cat main.go)"
```

### Container & Docker Operations

#### Docker Assistance
```bash
# In CLI mode
> help me create a Dockerfile for my Go application
> optimize my docker-compose.yml
> docker ps                    # Direct Docker command

# Direct mode
./bin/assistant ask "Create a multi-stage Dockerfile for Go"
```

#### Container Troubleshooting
```bash
./bin/assistant ask "My Docker container keeps crashing, how to debug?"
./bin/assistant ask "Optimize Docker image size for production"
```

### Kubernetes Operations

#### K8s Management
```bash
# In CLI mode
> k8s get pods               # Direct kubectl command
> help me deploy to Kubernetes
> create a Kubernetes service for my app

# Direct mode
./bin/assistant ask "Create Kubernetes deployment YAML for my Go service"
```

#### Troubleshooting
```bash
./bin/assistant ask "My pod is in CrashLoopBackOff, how to fix?"
./bin/assistant ask "How to check Kubernetes logs and events?"
```

### Database Operations

#### SQL Assistance
```bash
# In CLI mode
> sql SELECT * FROM users LIMIT 10    # Direct SQL execution
> help me optimize this PostgreSQL query
> design a database schema for my app

# Direct mode
./bin/assistant ask "Write a PostgreSQL migration for user authentication"
```

#### Database Design
```bash
./bin/assistant ask "Design a database schema for a blog system"
./bin/assistant ask "How to implement database connection pooling in Go?"
```

## Advanced Features

### Memory & Context

#### View Memory Statistics
```bash
# CLI mode
> memory stats

# API mode
curl http://localhost:8080/api/memory/stats
```

#### Trigger Memory Consolidation
```bash
# CLI mode
> memory consolidate

# API mode
curl -X POST http://localhost:8080/api/memory/consolidate
```

#### Search Across Memory
```bash
# CLI mode
> memory search "Docker best practices"

# Find related conversations
> conversations
> conversation abc123
```

### Tool Integration

#### Execute System Commands
```bash
# In CLI mode
> docker ps                  # Direct Docker command
> k8s get services          # Direct kubectl command  
> sql SELECT version()      # Direct SQL query

# Check tool status
> tools docker
> tools kubernetes
```

## Workflow Examples

### 1. Building a New Go Service

```bash
# Start interactive session
./bin/assistant cli

# Plan the service
> I want to build a REST API for user management in Go. Help me plan the architecture.

# Create project structure
> Create a Go project structure for a REST API with PostgreSQL

# Generate code
> Write a Go HTTP server with user CRUD endpoints

# Add Docker support
> Create a Dockerfile for this Go service

# Add Kubernetes deployment
> Create Kubernetes manifests for deployment
```

### 2. Debugging Production Issues

```bash
# Check container status
> docker ps
> k8s get pods -n production

# Analyze logs
> How to check application logs in Kubernetes?

# Database investigation
> sql SELECT COUNT(*) FROM error_logs WHERE created_at > NOW() - INTERVAL '1 hour'

# Get troubleshooting guidance
> My Go service is using too much memory, how to investigate?
```

### 3. Code Review & Optimization

```bash
# Review code quality
> analyze my Go code structure and suggest improvements

# Performance optimization
> help me optimize this database query: SELECT ...

# Security review
> review my Docker setup for security best practices

# Documentation
> generate API documentation for my Go endpoints
```

## Configuration & Customization

### Environment Setup

#### Development Environment
```bash
# Set to use cheapest model
export CLAUDE_MODEL=claude-3-haiku-20240307

# Enable debug logging
export LOG_LEVEL=debug

# Demo mode (no database)
export ASSISTANT_DEMO_MODE=true
```

#### Production Environment
```bash
# Use production model
export CLAUDE_MODEL=claude-3-sonnet-20240229

# Database connection
export DATABASE_URL=postgres://user:pass@db:5432/assistant

# API configuration
export SERVER_ADDRESS=:8080
```

### CLI Customization

```bash
# In CLI mode
> theme dark              # Change to dark theme
> theme light            # Change to light theme

# View history
> history

# Clear screen
> clear
```

## Troubleshooting

### Common Issues

#### Assistant Won't Start
```bash
# Check configuration
./bin/assistant --help

# Verify API key
echo $CLAUDE_API_KEY

# Check logs
./bin/assistant serve 2>&1 | grep ERROR
```

#### API Not Responding
```bash
# Check if server is running
curl http://localhost:8080/api/health

# Check port availability
lsof -i :8080

# Test with simple query
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{"query": "hello"}'
```

#### Tool Execution Failures
```bash
# Check tool availability
> tools

# Verify tool-specific requirements
> tools docker
> tools kubernetes

# Test tool directly
docker ps
kubectl version --client
```

### Performance Issues

#### Slow Responses
```bash
# Use faster model
export CLAUDE_MODEL=claude-3-haiku-20240307

# Check memory usage
> memory stats

# Trigger consolidation if needed
> memory consolidate
```

#### High Memory Usage
```bash
# Clear conversation history
> conversation delete <old-id>

# Restart in demo mode
export ASSISTANT_DEMO_MODE=true
./bin/assistant cli
```

## Integration Examples

### Shell Scripts
```bash
#!/bin/bash
# analyze-code.sh

# Quick code review
./bin/assistant ask "Review this Go code for issues: $(cat $1)"

# Performance suggestions
./bin/assistant ask "Suggest performance optimizations for: $(cat $1)"
```

### Git Hooks
```bash
#!/bin/bash
# pre-commit hook

# Get AI review of changes
DIFF=$(git diff --cached)
./bin/assistant ask "Review these code changes: $DIFF"
```

### CI/CD Integration
```yaml
# .github/workflows/ai-review.yml
name: AI Code Review
on: [pull_request]
jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: AI Review
        run: |
          ./bin/assistant ask "Review this PR for best practices: $(git diff origin/main)"
```

### IDE Integration
```bash
# VS Code task (.vscode/tasks.json)
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "AI Code Review",
      "type": "shell",
      "command": "./bin/assistant",
      "args": ["ask", "Review current file: ${file}"],
      "group": "build"
    }
  ]
}
```

## Best Practices

### Query Optimization

#### Effective Queries
```bash
# Good: Specific and contextual
> "Optimize this PostgreSQL query for better performance: SELECT ..."

# Better: Include relevant context
> "I have a Go web service with 1M users. Optimize this user lookup query: ..."

# Best: Provide full context
> "In my Go microservice handling 1000 RPS, this PostgreSQL query is slow. 
   Current indexes: user_email_idx. Query: SELECT ... How to optimize?"
```

#### Context Sharing
```bash
# Share relevant files
> "Review this Go code: $(cat main.go)"

# Include error messages
> "Getting this error: 'connection refused'. Here's my Docker setup: $(cat docker-compose.yml)"

# Provide system info
> "On Ubuntu 20.04, Go 1.21, having issues with: ..."
```

### Conversation Management

#### Organize by Topic
```bash
# Start focused conversations
> # Start new conversation for each major topic
> help me with Docker optimization

# Continue related discussions
> # Keep related questions in same conversation
> now help me with Kubernetes deployment
```

#### Use Descriptive Queries
```bash
# Instead of: "fix this"
# Use: "fix PostgreSQL connection timeout in Go application"

# Instead of: "help me"
# Use: "help me implement JWT authentication in Go REST API"
```

### Tool Usage

#### Layer Your Approach
```bash
# 1. Get guidance first
> "What's the best way to deploy Go apps to Kubernetes?"

# 2. Get specific help
> "Create a Kubernetes deployment YAML for my Go service"

# 3. Verify with tools
> k8s apply -f deployment.yaml --dry-run=client
```

## Tips & Tricks

### Keyboard Shortcuts (CLI Mode)
- `Ctrl+C` - Cancel current operation
- `Ctrl+D` - Exit assistant
- `Up/Down` - Navigate command history
- `Tab` - Auto-complete commands
- `Ctrl+L` - Clear screen

### Quick Commands
```bash
# Fast status check
./bin/assistant ask "status"

# Quick tool list
./bin/assistant ask "tools"

# Fast code review
./bin/assistant ask "review: $(cat file.go)" | head -20
```

### Batch Operations
```bash
# Review multiple files
for file in *.go; do
  echo "=== $file ==="
  ./bin/assistant ask "quick review: $(cat $file)"
done

# Check multiple services
./bin/assistant ask "check health of: $(kubectl get pods -o name)"
```

---

For complete API reference, see [API_REFERENCE.md](./API_REFERENCE.md).
For advanced configuration, see [CONFIGURATION.md](./CONFIGURATION.md).
For development setup, see [DEVELOPMENT.md](./DEVELOPMENT.md).