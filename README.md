# Assistant - Go Developer's AI Companion

**The intelligent CLI tool specifically designed for Golang developers** - Inspired by Claude Code's excellent developer experience, Assistant provides AI-powered assistance for your Go development workflow with deep understanding of Go projects, integrated DevOps tools, and intelligent automation.

Assistant combines the power of AI with specialized Go development tools, Docker integration, PostgreSQL management, Kubernetes support, and Cloudflare deployment capabilities to create the ultimate development companion for Go developers.

## Features

### Multi-Mode Operation

Assistant operates in three modes to fit your workflow:

- **HTTP API Server** (`assistant serve`) - RESTful API for integration with other tools
- **Interactive CLI** (`assistant cli`) - Rich command-line interface with colors and auto-completion
- **Direct Query** (`assistant ask "query"`) - Quick one-off queries from command line

### 🎯 AI-Powered Intelligence (COMPLETED)

- **✅ Intelligent Prompt System**: 7 specialized prompt templates for different Go development tasks
- **✅ Context-Aware Enhancement**: Query enhancement with project context and workspace understanding
- **✅ Task Type Detection**: Automatic classification of user queries into appropriate categories
- **✅ LangChain Integration**: Memory management, agent architecture, and chain execution
- **✅ Vector Search**: PostgreSQL with pgvector for semantic document search and RAG
- **✅ AI Providers**: Support for Claude (Anthropic) and Gemini (Google) APIs
- **✅ Intelligent Memory**: Long-term, short-term, and personalization memory systems

### 🐹 Go-First Development Tools (COMPLETED)

**Built by Go developers, for Go developers** - Deep semantic understanding of Go projects:

- **✅ Intelligent Code Analysis**: AST parsing, code structure analysis, and Go idiom detection
- **✅ Workspace Detection**: Automatic Go project structure understanding and context awareness
- **✅ Project Type Detection**: CLI, Web Service, Microservice, Library, and Monorepo classification
- **✅ Dependency Analysis**: go.mod parsing, dependency graph, direct/indirect dependency tracking
- **✅ Code Metrics**: Cyclomatic complexity, test coverage, function/struct/interface analysis
- **✅ Git Integration**: Repository information, branch status, commit history
- **🔄 Smart Refactoring**: Automated refactoring suggestions following Go best practices (PLANNED)
- **🔄 Advanced Testing**: Test generation, execution, coverage analysis, and benchmark optimization (PLANNED)
- **🔄 Build Intelligence**: Build optimization, cross-compilation, and dependency management (PLANNED)
- **🔄 Module Management**: go.mod analysis, dependency graph visualization, and version management (PLANNED)

### 🛠️ Integrated DevOps Tools (ROADMAP)

**Complete development lifecycle support** - Inspired by Claude Code's comprehensive tool integration:

- **✅ Docker Integration**: Container management, Dockerfile optimization, and multi-stage build analysis (COMPLETED)
- **🔄 PostgreSQL Tools**: Query optimization, migration management, performance analysis, and schema design (HIGH PRIORITY)
- **🔄 Kubernetes Support**: Deployment management, Pod debugging, resource monitoring, and cluster operations (MEDIUM PRIORITY)
- **🔄 Cloudflare Integration**: Workers deployment, DNS management, Analytics, and edge optimization (MEDIUM PRIORITY)

### 🧠 Claude Code-Inspired AI Features (PARTIALLY COMPLETED)

**Intelligent development assistance** - Learning from Claude Code's excellent UX patterns:

- **✅ Conversational Interface**: Natural language interaction for complex development tasks
- **✅ Context-Aware Suggestions**: Smart recommendations based on your current project and coding patterns
- **✅ Project Understanding**: Deep comprehension of your codebase, architecture, and dependencies
- **✅ Real-Time Streaming**: Stream AI responses word-by-word for immediate feedback
- **🔄 Workflow Automation**: Intelligent automation of repetitive development tasks (IN PROGRESS)
- **🔄 Code Generation**: AI-powered code generation following Go conventions and best practices (PLANNED)

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
- **Real-Time Streaming**: Watch AI responses appear word-by-word
- **Interactive Menus**: Guided workflows for common tasks

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

## Quick Start for Go Developers

**Get up and running in minutes** - Assistant automatically detects your Go project and provides context-aware assistance:

```bash
# Navigate to your Go project
cd /path/to/your/go/project

# Set up your AI API key
export CLAUDE_API_KEY="your_claude_api_key"

# Run database migrations (one time setup)
assistant migrate

# Start interactive development session
assistant cli
```

Once in the CLI, Assistant will automatically:

- 🔍 Detect your Go project structure (`go.mod`, packages, dependencies)
- 📋 Analyze your codebase and understand the context
- 🧠 Provide intelligent suggestions based on your project

### Instant Go Development Help

```bash
# In your Go project directory
assistant ask "Review my HTTP handlers for performance issues"
assistant ask "Generate tests for my UserService struct"
assistant ask "Help me optimize this database query"
assistant ask "Suggest improvements for my Dockerfile"
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

## Go Development Examples

### 🔍 Intelligent Code Analysis

```bash
# Analyze your Go project structure
assistant ask "Analyze my Go project structure and suggest improvements"

# Review code for Go best practices
assistant ask "Review this function for Go idioms: $(cat internal/service/user.go)"

# Dependency analysis with visualization
assistant ask "Analyze my go.mod dependencies and create a dependency graph"

# Performance analysis
assistant ask "Identify performance bottlenecks in my HTTP handlers"
```

### 🧪 Smart Testing & Quality

```bash
# Generate comprehensive tests
assistant ask "Generate unit tests for my UserService struct with edge cases"

# Test coverage analysis
assistant ask "Analyze my test coverage and suggest missing test scenarios"

# Benchmark optimization
assistant ask "Help me optimize this benchmark: $(cat internal/benchmark_test.go)"

# Race condition detection
assistant ask "Check this code for potential race conditions: $(cat internal/concurrent.go)"
```

### 🐳 Docker & DevOps Integration (COMPLETED)

```bash
# List Docker containers
assistant ask "docker list_containers"

# Analyze Dockerfile for best practices
assistant ask "docker analyze_dockerfile"

# Optimize Dockerfile for production
assistant ask "docker optimize_dockerfile"

# Analyze Docker build performance
assistant ask "docker build_analyze"

# Inspect container details
assistant ask "docker inspect_container <container_id>"

# Get container logs
assistant ask "docker container_logs <container_id>"
```

### ☸️ Kubernetes & Deployment

```bash
# Kubernetes configuration review
assistant ask "Review my k8s deployment for Go microservice: $(cat k8s/deployment.yaml)"

# Resource optimization
assistant ask "Optimize resource limits and requests for my Go service"

# Health check implementation
assistant ask "Implement proper health checks for Kubernetes deployment"
```

### 🐘 PostgreSQL Integration

```bash
# Query optimization
assistant ask "Optimize this PostgreSQL query for better performance: $(cat queries/user_stats.sql)"

# Migration assistance
assistant ask "Help me write a migration for user authentication tables"

# Schema design review
assistant ask "Review my database schema design for scalability"
```

### 💬 Interactive Development Session

```bash
$ assistant cli
🐹 Assistant Go Developer Companion v1.0.0

Workspace detected: /home/dev/myproject (Go 1.24, 15 modules)
Database: ✅ Connected to PostgreSQL 17.0
AI Provider: ✅ Claude 3.5 Sonnet

assistant> analyze performance issues

🔍 Analyzing your Go project for performance issues...

Found 3 potential optimizations:

1. **Memory allocation in hot path** (handlers/api.go:45)
   • Issue: Creating new structs on every request
   • Impact: High memory pressure, increased GC
   • Solution: Use object pooling with sync.Pool

2. **Database N+1 query pattern** (services/user.go:78)
   • Issue: Loading related data in a loop
   • Impact: Increased database load and latency
   • Solution: Use JOIN or batch loading

3. **Unbounded slice growth** (processors/batch.go:23)
   • Issue: Slice grows without capacity planning
   • Impact: Memory reallocations and copying
   • Solution: Pre-allocate with estimated capacity

assistant> show fix for issue 1

✨ Here's the optimized code using sync.Pool:

// Before: Creating new request object every time
func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
    req := &RequestData{} // ❌ Allocates on every call
    // ... process request
}

// After: Using object pool for efficiency
var requestPool = sync.Pool{
    New: func() interface{} {
        return &RequestData{}
    },
}

func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
    req := requestPool.Get().(*RequestData) // ✅ Reuses objects
    defer requestPool.Put(req)

    // Reset the object state
    req.Reset()

    // ... process request
}

This change can reduce memory allocations by 60-80% in high-traffic scenarios.

assistant> implement this fix in my codebase

🔧 I'll help you implement the sync.Pool optimization...

Would you like me to:
1. Apply this fix directly to handlers/api.go
2. Show you the complete refactored file
3. Create a benchmark to measure the improvement

assistant> option 1

✅ Applied sync.Pool optimization to handlers/api.go

Changes made:
• Added requestPool variable at package level
• Modified HandleRequest method to use pooled objects
• Added Reset() method to RequestData struct
• Updated 3 handler functions to use the pool

Performance improvement: Expected 60-80% reduction in allocations.

Next steps:
• Run benchmarks: `go test -bench=BenchmarkHandler -benchmem`
• Monitor memory usage in production
• Consider applying similar pattern to other hot paths

assistant> help setup monitoring

🔍 Setting up performance monitoring for Go applications...

[Provides detailed monitoring setup with Prometheus, pprof, and observability tools]
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

## Development Roadmap

### 📅 Immediate Next Steps

**HIGH PRIORITY - Core DevOps Integration**

1. **🐳 Docker Tools Implementation** - Container management and Dockerfile analysis

   - Dockerfile parsing and optimization suggestions
   - Multi-stage build analysis and recommendations
   - Container security scanning and best practices
   - Docker Compose configuration validation

2. **🐘 PostgreSQL Tools Enhancement** - Database management and optimization

   - SQL query parsing and performance optimization
   - Migration file analysis and validation
   - Schema design recommendations
   - Connection pool optimization

3. **⚡ Performance Optimization Tools** - Go-specific performance analysis
   - Memory profiling integration (pprof)
   - CPU benchmarking and analysis
   - Go routine leak detection
   - Garbage collection optimization suggestions

### 📅 Medium Term Goals

**MEDIUM PRIORITY - Advanced Features**

4. **☸️ Kubernetes Support** - Container orchestration management

   - Kubernetes manifest validation
   - Resource optimization recommendations
   - Pod debugging and log analysis
   - Health check implementation guidance

5. **🔄 Development Workflow Automation** - CI/CD and automation

   - Git hook integration for quality checks
   - GitHub Actions / GitLab CI configuration
   - Automated code review assistance
   - Release automation recommendations

6. **🎨 Advanced Code Generation** - AI-powered development assistance
   - Struct and interface generation from requirements
   - Test case generation with edge cases
   - API endpoint scaffolding
   - Error handling pattern implementation

### 📅 Long Term Vision

**FUTURE ENHANCEMENTS**

- **☁️ Cloudflare Integration**: Workers deployment, DNS management, Analytics
- **🤝 Team Collaboration**: Shared learning and knowledge management
- **📊 Advanced Analytics**: Development pattern analysis and insights
- **🔌 Plugin System**: Community-driven tool extensions
- **🌐 Multi-Language Support**: Expanding beyond Go to other languages

### 🎯 Success Metrics

- **Developer Experience**: Reduce common Go development tasks by 60%+
- **Code Quality**: Improve test coverage and reduce complexity metrics
- **Performance**: Sub-second response times for most operations
- **Adoption**: Active usage in 10+ Go projects within 6 months
