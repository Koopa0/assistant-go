# Assistant

An intelligent development companion built with Go that provides AI-powered assistance for your development workflow. Assistant combines PostgreSQL with pgvector for semantic search, LangChain integration for AI capabilities, and a comprehensive tool ecosystem to help you build better software faster.

## Features

### Multi-Mode Operation

Assistant operates in three modes to fit your workflow:

- **HTTP API Server** (`assistant serve`) - RESTful API for integration with other tools
- **Interactive CLI** (`assistant cli`) - Rich command-line interface with colors and auto-completion
- **Direct Query** (`assistant ask "query"`) - Quick one-off queries from command line

### AI-Powered Capabilities

- **LangChain Integration**: Memory management, agent architecture, and chain execution
- **Vector Search**: PostgreSQL with pgvector for semantic document search and RAG
- **AI Providers**: Support for Claude (Anthropic) and Gemini (Google) APIs
- **Intelligent Memory**: Long-term, short-term, and personalization memory systems

### Go Development Tools

Complete suite of Go development tools with semantic understanding:

- **Code Analysis**: AST parsing and code structure analysis
- **Formatting**: Intelligent code formatting
- **Testing**: Test execution and result analysis  
- **Building**: Build management and optimization
- **Dependency Analysis**: Dependency tracking and management

### Database Features

- **PostgreSQL 17+** with connection pooling optimization
- **pgvector Extension** for embedding storage and similarity search
- **Complete Schema** with migrations for conversations, tools, and AI tracking
- **Vector Embeddings** for document storage and retrieval
- **Search Caching** for improved performance

### HTTP API

RESTful API with comprehensive endpoints:

```bash
# Health and Status
GET /api/health              # System health check
GET /api/status              # Detailed system status

# Query Processing  
POST /api/query              # Process AI queries

# Conversation Management
GET /api/conversations       # List conversations
GET /api/conversations/{id}  # Get specific conversation
DELETE /api/conversations/{id} # Delete conversation

# Tool Management
GET /api/tools               # List available tools
GET /api/tools/{name}        # Get tool information
POST /api/tools/{name}/execute # Execute tool directly

# LangChain API
GET /api/langchain/agents    # List available agents
POST /api/langchain/agents/{type}/execute # Execute agent
GET /api/langchain/chains    # List available chains
POST /api/langchain/chains/{type}/execute # Execute chain
POST /api/langchain/memory   # Store memory
POST /api/langchain/memory/search # Search memory
GET /api/langchain/providers # List LLM providers
GET /api/langchain/health    # LangChain health check
```

### CLI Interface

Rich interactive CLI with:

- **Command Auto-completion**: Tab completion for commands and arguments
- **Colored Output**: Syntax highlighting and status colors
- **Table Formatting**: Structured display of results
- **Theme Support**: Dark and light color themes
- **Command History**: Navigate previous commands
- **Progress Indicators**: Visual feedback for long operations

#### CLI Commands

```bash
# Basic Commands
help, ?                    # Show help
exit, quit, bye           # Exit CLI
clear, cls                # Clear screen
status                    # Show system status
tools                     # List available tools
history                   # Command history
theme <dark|light>        # Change color theme

# Development Commands
sql <query>               # Execute SQL query
k8s <command>            # Kubernetes operations
docker <command>         # Docker operations

# LangChain Commands
langchain, lc             # LangChain operations
agents                    # List available LangChain agents
chains                    # List available LangChain chains
langchain agents execute <type> <query>  # Execute specific agent
langchain chains execute <type> <input>  # Execute specific chain
langchain memory <command>               # Memory operations
```

## Installation

### Prerequisites

- **Go 1.24+**: Latest Go version for optimal performance
- **PostgreSQL 17+**: Database with pgvector extension
- **AI API Key**: Claude or Gemini API key for AI capabilities

### Install from Source

```bash
# Clone the repository
git clone https://github.com/koopa0/assistant-go.git
cd assistant-go

# Build and install
make setup
make build

# Or install directly
go install github.com/koopa0/assistant-go/cmd/assistant@latest
```

### Database Setup

```bash
# Install PostgreSQL 17+ with pgvector
# Ubuntu/Debian:
sudo apt install postgresql-17 postgresql-17-pgvector

# macOS with Homebrew:
brew install postgresql@17 pgvector

# Create database
createdb assistant

# Set environment variables
export DATABASE_URL="postgres://user:password@localhost:5432/assistant?sslmode=disable"
export CLAUDE_API_KEY="your_claude_api_key"
# OR
export GEMINI_API_KEY="your_gemini_api_key"
```

### Configuration

Create a `.env` file or set environment variables:

```bash
# Required
DATABASE_URL=postgres://user:password@localhost:5432/assistant?sslmode=disable
CLAUDE_API_KEY=your_claude_api_key_here

# Optional
SERVER_ADDRESS=:8080
LOG_LEVEL=info
LOG_FORMAT=json
```

## Usage

### Start HTTP API Server

```bash
# Start server on default port (8080)
assistant serve

# Start on custom port
assistant serve --port 9000

# Server provides REST API at http://localhost:8080
```

### Interactive CLI Mode

```bash
# Start interactive CLI
assistant cli

# Example session:
Assistant> help
Available commands:
  help, ?          Show this help message
  exit, quit, bye  Exit the assistant
  status          Show system status
  tools           List available tools
  sql <query>     Execute SQL query

Assistant> status
System Status:
  Database: Connected ✓
  Tools: 5 registered
  Memory: Long-term initialized ✓

Assistant> sql SELECT current_database()
┌─────────────────┐
│ current_database│
├─────────────────┤
│ assistant       │
└─────────────────┘
```

### Direct Query Mode

```bash
# Quick queries without interactive session
assistant ask "What Go tools are available?"
assistant ask "Analyze the code structure of my project"
assistant ask "Show me database connection status"
```

### API Usage

```bash
# Query the API
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{"query": "What tools are available?", "context": "development"}'

# List tools
curl http://localhost:8080/api/tools

# Execute a tool
curl -X POST http://localhost:8080/api/tools/go_analyzer/execute \
  -H "Content-Type: application/json" \
  -d '{"path": "./cmd/assistant", "analysis_type": "structure"}'
```

## Development

### Build Commands

```bash
# Setup development environment
make setup

# Run in development mode
make dev

# Build the application
make build

# Run tests
make test

# Run with coverage
make test-coverage

# Code quality checks (IMPORTANT: Run after changes)
make quick-check
```

### Code Quality

**CRITICAL**: Always run code quality checks after making changes:

```bash
# Essential checks (required after every change)
make quick-check

# Comprehensive analysis
make quality-check

# Basic verification
make verify
```

### Database Migrations

```bash
# Run migrations
make migrate-up

# Rollback migrations
make migrate-down

# Generate new SQL code (after schema changes)
make sqlc-generate
```

### Performance Profiling

```bash
# Collect CPU profile (server must be running)
make profile-cpu

# Collect memory profile
make profile-mem

# PGO optimization (Profile-Guided Optimization)
make pgo-collect
make pgo-build
```

## Architecture

### Core Components

- **Agent Network**: Specialized AI agents for different domains (development, database, infrastructure)
- **Memory Systems**: Multi-layered memory (working, episodic, semantic, procedural) 
- **Tool Registry**: Dynamic tool registration and execution framework
- **Context Engine**: Maintains awareness of development environment and user patterns
- **Event System**: Event-driven architecture for learning and automation

### Technology Stack

- **Language**: Go 1.24+ with latest standard library features
- **Database**: PostgreSQL 17+ with pgvector for embeddings
- **AI Integration**: LangChain-Go for AI orchestration
- **API**: RESTful HTTP API with structured JSON responses
- **CLI**: Rich terminal interface with colors and auto-completion
- **Observability**: Structured logging, metrics, and profiling

### Tool Ecosystem

Tools are organized by category:

- **Development**: Go analyzer, formatter, tester, builder, dependency analyzer
- **Database**: PostgreSQL query execution and optimization
- **Infrastructure**: Kubernetes and Docker management (demo implementations)
- **AI**: LangChain agents and memory management
- **Search**: Semantic search with vector embeddings

## Supported Integrations

### AI Providers
- **Claude (Anthropic)**: Primary AI provider with advanced reasoning
- **Gemini (Google)**: Alternative AI provider with competitive performance

### Development Tools
- **Go Tools**: Complete native integration with the Go toolchain
- **PostgreSQL**: Full database management and query optimization
- **Docker**: Container management and deployment
- **Kubernetes**: Cluster management and resource monitoring

### Search Capabilities
- **SearXNG**: Privacy-focused web search integration
- **Vector Search**: Semantic search using pgvector embeddings
- **Caching**: Intelligent result caching for improved performance

## Monitoring and Observability

### Health Monitoring
```bash
# Check system health
curl http://localhost:8080/api/health

# Detailed status
curl http://localhost:8080/api/status
```

### Performance Monitoring
- **pprof Integration**: CPU, memory, and goroutine profiling
- **Database Metrics**: Connection pool monitoring and query performance
- **Request Tracking**: HTTP request/response logging and timing

### Logging
- **Structured Logging**: JSON format with contextual information
- **Configurable Levels**: DEBUG, INFO, WARN, ERROR
- **Request Correlation**: Track requests across components

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes and add tests
4. Run `make quick-check` to verify code quality
5. Submit a pull request

### Development Workflow

1. **Setup**: `make setup` to install tools and dependencies
2. **Develop**: Make changes with hot reload using `make dev`
3. **Test**: Run `make test` for unit tests, `make test-integration` for full tests
4. **Quality**: Always run `make quick-check` before committing
5. **Build**: `make build` to create production binary

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/koopa0/assistant-go/issues)
- **Documentation**: See `/docs` directory for detailed guides
- **Examples**: Check `/examples` for usage patterns

## Roadmap

- **Enhanced AI Capabilities**: More sophisticated reasoning and context awareness
- **Tool Ecosystem**: Expanded tool integrations and community contributions
- **Performance Optimization**: Further improvements in speed and resource usage
- **Team Features**: Shared learning and collaboration capabilities