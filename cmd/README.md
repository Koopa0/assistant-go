# Command Line Applications

This directory contains the main executable applications for the Assistant project.

## Structure

```
cmd/
└── assistant/          # Main assistant application
    └── main.go         # Application entry point
```

## Applications

### assistant
The main Assistant application that provides:
- **CLI Mode**: Interactive command-line interface
- **Server Mode**: HTTP API server
- **Direct Query Mode**: Single command execution

## Usage

```bash
# Interactive CLI mode
go run ./cmd/assistant

# Server mode
go run ./cmd/assistant serve

# Direct query
go run ./cmd/assistant ask "help me debug this issue"
```

## Build

```bash
# Build the main binary
go build -o bin/assistant ./cmd/assistant

# Build with optimizations
go build -ldflags="-w -s" -o bin/assistant ./cmd/assistant
```