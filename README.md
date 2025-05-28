# GoAssistant

An intelligent AI-powered personal and development assistant built with Go, designed to enhance productivity through deep integration with modern development tools and AI capabilities. Built following Go's idiomatic patterns without any web frameworks, emphasizing simplicity, type safety, and performance.

## Overview

GoAssistant is a comprehensive assistant that seamlessly integrates cutting-edge AI models (Claude, Gemini) with practical development tools through the LangChain Go framework. It provides both CLI and web interfaces for natural interaction, whether you're debugging code, managing infrastructure, optimizing databases, or seeking general assistance.

The project demonstrates advanced Go development practices, including pure standard library usage, sophisticated error handling, and clean architecture principles. It serves as both a powerful productivity tool and a reference implementation for building complex Go applications.

## Key Features

### 🤖 AI Integration
- **Dual AI Support**: Seamlessly switch between Claude and Gemini models
- **Context-Aware Responses**: Maintains conversation history for coherent interactions
- **RAG Capabilities**: Enhanced responses through Retrieval-Augmented Generation with pgvector
- **Agent Framework**: Deep [LangChain Go](https://github.com/tmc/langchaingo) integration for complex automation
  - Multi-step reasoning and planning
  - Tool orchestration and selection
  - Memory management (buffer, vector, summary)
  - Custom agent creation for specialized tasks

### 🔍 Intelligent Search
- **Advanced Web Search**: Self-hosted SearXNG instance with enhanced capabilities
- **Semantic Search**: AI-powered search result ranking and filtering
- **Multi-Source Aggregation**: Combines results from multiple search engines
- **Privacy-Focused**: No tracking or data collection

### 💻 Go Development Assistant
- **Code Analysis**: Deep static analysis with Go AST parsing
- **Performance Profiling**: Real-time CPU and memory profiling with pprof integration
- **Distributed Tracing**: Execution flow visualization and bottleneck detection
- **Runtime Monitoring**: Live metrics and resource usage tracking
- **Debugging Support**: Integrated debugging capabilities
- **LangChain Integration**: AI-powered code suggestions and automated refactoring

### 🗄️ Database Expertise
- **PostgreSQL Mastery**: Advanced operations using [pgx driver](https://github.com/jackc/pgx)
- **Query Optimization**: AI-assisted query analysis and improvement
- **Schema Management**: Intelligent migration with [sqlc](https://sqlc.dev)
- **Performance Tuning**: Database performance analysis and recommendations
- **pgvector Integration**: Vector similarity search for RAG implementation

### ☸️ Infrastructure Management
- **Kubernetes Control**: Native K8s API integration using client-go
- **Resource Management**: Quota monitoring and optimization
- **Docker Operations**: Container lifecycle management via Docker SDK
- **Deployment Automation**: Streamlined deployment workflows
- **Agent-Based Management**: Autonomous infrastructure troubleshooting

### ☁️ Cloud Integration
- **Cloudflare Services**: Deep integration with Cloudflare SDK
  - Tunnel management for secure connections
  - R2 storage operations
  - Pages deployment automation
  - Domain and DNS management
  - Workers and KV integration

### 📊 Observability
- **Comprehensive Monitoring**: OpenTelemetry, Prometheus, Loki, and Jaeger integration
- **Real-time Metrics**: System and application performance metrics
- **Distributed Tracing**: Request flow visualization across components
- **Structured Logging**: Using Go 1.24+ `log/slog` package
- **Agent Observability**: Track agent decisions and tool usage

## Use Cases

- **Development Productivity**: Accelerate Go development with AI-powered code analysis and generation
- **Automated Debugging**: Let agents analyze performance issues and suggest optimizations
- **Infrastructure Management**: Simplify Kubernetes and Docker operations through conversational interface
- **Database Operations**: Natural language to SQL with automatic optimization
- **Research and Analysis**: Multi-agent collaboration for complex research tasks
- **Automation Workflows**: Build custom agents for repetitive development tasks
- **Learning Assistant**: Interactive exploration of Go internals and best practices
- **Incident Response**: Automated troubleshooting with intelligent log analysis

## Technology Stack

- **Language**: Go 1.24+ (utilizing latest stdlib enhancements)
- **Web Server**: Pure Go `net/http` (no frameworks)
- **Database**: PostgreSQL 15+ with pgvector extension
- **Database Driver**: [pgx](https://github.com/jackc/pgx) for native PostgreSQL access
- **SQL Generation**: [sqlc](https://sqlc.dev) for type-safe queries
- **AI Integration**:
  - Claude (Anthropic)
  - Gemini (Google)
  - [LangChain Go](https://github.com/tmc/langchaingo) for agent framework
- **Container Orchestration**: Kubernetes via Kind for local development
- **Search**: Self-hosted SearXNG for privacy-focused web search
- **UI Stack**:
  - [Templ](https://templ.guide) for type-safe templates
  - [HTMX](https://htmx.org) for dynamic interactions
  - Material Design 3 components ([templui.io](https://templui.io))
- **Logging**: Go 1.21+ `log/slog` for structured logging
- **Error Handling**: Wrapped errors with `fmt.Errorf` and `%w`
- **Observability**: OpenTelemetry, Prometheus, Loki, Jaeger

## Installation

### Prerequisites
- Go 1.24 or higher
- PostgreSQL 15+ with pgvector extension
- Docker and Kind (for Kubernetes features)
- Git

### Quick Start

```bash
# Clone the repository
git clone https://github.com/koopa0/assistant-go.git
cd assistant-go

# Check project status and dependencies
./scripts/check-status.sh

# Verify all dependencies are working
go run scripts/verify-dependencies.go

# Install pgvector extension for PostgreSQL
# For Ubuntu/Debian:
sudo apt install postgresql-17-pgvector
# For macOS:
brew install pgvector

# Setup development environment
make setup

# This will:
# - Install Go dependencies
# - Install development tools (sqlc, golangci-lint)
# - Setup Kind cluster with required services
# - Initialize database with pgvector

# Configure environment
cp .env.example .env
# Edit .env with your API keys and configuration

# Run database migrations
make migrate

# Generate sqlc code
make generate

# Start the application
make run
```

### Docker Compose Setup (Alternative)

```bash
# Start all services including PostgreSQL with pgvector
docker-compose up -d

# Run migrations
make migrate

# Start the application
make run
```

## Configuration

GoAssistant uses environment-based configuration following the 12-factor app methodology. See `.env.example` for all available options.

### Required Configuration
- `CLAUDE_API_KEY` or `GEMINI_API_KEY` (at least one AI provider)
- `DATABASE_URL` (PostgreSQL connection string with pgvector)

### Optional Services
- `CLOUDFLARE_API_TOKEN` - For Cloudflare integrations
- `KUBERNETES_CONFIG_PATH` - K8s cluster configuration
- `DOCKER_HOST` - Docker daemon connection
- `SEARXNG_URL` - SearXNG instance URL
- Observability endpoints (Prometheus, Loki, Jaeger)

### Development Configuration
```bash
# Copy example configuration
cp .env.example .env

# Edit with your settings
vim .env

# Required: Set at least one AI provider
export CLAUDE_API_KEY="your-claude-key"
# or
export GEMINI_API_KEY="your-gemini-key"

# Required: PostgreSQL with pgvector
export DATABASE_URL="postgres://user:pass@localhost/goassistant?sslmode=disable"
```

## Usage

### CLI Mode
```bash
# Interactive mode
goassistant cli

# Direct command
goassistant ask "Explain Go's memory model"
```

### Web Interface
```bash
# Start web server
goassistant serve

# Access at http://localhost:8080
```

## Agent Architecture

GoAssistant implements specialized AI agents for different domains:

- **Development Agent**: Analyzes code, suggests improvements, and automates refactoring
- **Database Agent**: Optimizes queries, manages schemas, and troubleshoots performance
- **Infrastructure Agent**: Manages K8s/Docker resources and diagnoses issues
- **Research Agent**: Gathers information from multiple sources and synthesizes insights

Each agent leverages LangChain's ReAct framework for reasoning and can use multiple tools in sequence to accomplish complex tasks.

## Development Philosophy

This project strictly adheres to Go's idiomatic patterns and best practices:

- **No Web Frameworks**: Pure `net/http` for complete control and clarity
- **Interface-Driven Design**: Accept interfaces, return structs
- **Error Handling**: Explicit error handling with wrapped errors
- **Standard Library First**: Maximize use of Go's excellent standard library
- **Type Safety**: Leverage Go's type system with sqlc for database queries
- **Composition over Inheritance**: Build complex behavior through composition
- **Simplicity**: Prefer simple, clear code over clever abstractions

## Documentation

- [Architecture Overview](docs/ARCHITECTURE.md)
- [Development Guide](docs/DEVELOPMENT.md)
- [Dependency Management](docs/DEPENDENCY_MANAGEMENT.md)
- [API Reference](docs/API.md)
- [Configuration Guide](docs/CONFIGURATION.md)
- [Deployment Guide](docs/DEPLOYMENT.md)

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

### Development Process
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Follow Go best practices and project conventions
4. Add comprehensive tests (aim for >80% coverage)
5. Ensure all tests pass (`make test`)
6. Run linters (`make lint`)
7. Commit with clear messages (`git commit -m 'Add amazing feature'`)
8. Push to your fork (`git push origin feature/amazing-feature`)
9. Open a Pull Request with detailed description

### Areas We Need Help
- Additional LangChain tool implementations
- Performance optimizations
- Documentation improvements
- Test coverage expansion
- UI/UX enhancements
- Community tool plugins

## Roadmap

See our [project board](https://github.com/yourusername/goassistant/projects) for current progress.

### Phase 1: Foundation (Current)
- Core architecture with pure Go stdlib
- Basic Claude/Gemini integration
- PostgreSQL with pgvector setup
- LangChain tool adapters
- Web UI with Templ + HTMX

### Phase 2: Agent Development
- Development assistant agent with Go AST integration
- Database expert agent with query optimization
- Infrastructure management agent
- Research and analysis agent
- Memory system implementation

### Phase 3: Advanced Features
- Multi-agent collaboration
- Advanced RAG with re-ranking
- Real-time monitoring dashboards
- Plugin system for custom tools
- MCP (Model Context Protocol) integration when Go support is released

### Phase 4: Production Ready
- Performance optimization
- Comprehensive test coverage
- Security hardening
- Documentation completion
- Community plugins support

## System Requirements

### Minimum Requirements
- Go 1.24+
- PostgreSQL 15+ with pgvector extension
- 4GB RAM
- 10GB disk space

### Recommended Setup
- Go 1.24+ (latest version)
- PostgreSQL 16 with pgvector
- 8GB+ RAM
- 20GB+ disk space
- Docker Desktop for containerized services
- Kind for local Kubernetes development

### Development Tools
- `sqlc` - SQL code generation
- `templ` - Template generation
- `golangci-lint` - Code quality checks
- `go-migrate` - Database migrations
- `air` - Live reload for development

### Project Management Scripts
- `./scripts/check-status.sh` - Comprehensive project health check
- `./scripts/upgrade-dependencies.sh` - Safe dependency upgrade tool
- `go run scripts/verify-dependencies.go` - Dependency verification test

## Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/yourusername/goassistant/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/goassistant/discussions)

## Architecture Highlights

- **Pure Go Standard Library**: No web frameworks, just `net/http` for clarity and control
- **Type-Safe Database Access**: Using sqlc for compile-time SQL verification
- **Advanced Error Handling**: Wrapped errors with full context propagation
- **Interface-Driven Design**: Clean boundaries between components
- **Agent-Based Architecture**: Autonomous agents for complex task execution
- **Vector-Based Memory**: Semantic search with PostgreSQL pgvector
- **Real-Time Streaming**: WebSocket support for responsive interactions
- **Comprehensive Observability**: Full tracing, metrics, and structured logging

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with the principles of idiomatic Go
- Powered by [LangChain Go](https://github.com/tmc/langchaingo) for agent capabilities
- Inspired by the need for a truly integrated development assistant
- Special thanks to the Go community and all contributors