# GoAssistant Makefile
# Following Go 1.24+ best practices and Architecture.md specifications

# Variables
BINARY_NAME=goassistant
MAIN_PATH=./cmd/assistant
BUILD_DIR=./bin
DOCKER_IMAGE=goassistant
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"
GO_BUILD_FLAGS=-trimpath $(LDFLAGS)

# Tools
GOLANGCI_LINT_VERSION=v1.55.2
SQLC_VERSION=v1.25.0

# Default target
.PHONY: all
all: clean lint test build

# Help target
.PHONY: help
help: ## Show this help message
	@echo "GoAssistant - AI-powered development assistant"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development setup
.PHONY: setup
setup: ## Setup development environment
	@echo "Setting up development environment..."
	go mod download
	go mod tidy
	$(MAKE) install-tools
	$(MAKE) generate
	@echo "Development environment setup complete!"

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	go install github.com/a-h/templ/cmd/templ@latest
	@echo "Development tools installed!"

# Build targets
.PHONY: build
build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-race
build-race: ## Build with race detector
	@echo "Building $(BINARY_NAME) with race detector..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build -race $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-race $(MAIN_PATH)

.PHONY: build-all
build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

# Run targets
.PHONY: run
run: ## Run the application
	go run $(MAIN_PATH)

.PHONY: run-server
run-server: ## Run the web server
	go run $(MAIN_PATH) serve

.PHONY: run-cli
run-cli: ## Run the CLI interface
	go run $(MAIN_PATH) cli

# Development targets
.PHONY: dev
dev: ## Run in development mode with hot reload
	@echo "Starting development server..."
	go run $(MAIN_PATH) serve

.PHONY: watch
watch: ## Watch for changes and rebuild
	@echo "Watching for changes..."
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

# Testing targets
.PHONY: test
test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-short
test-short: ## Run short tests
	go test -v -short ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	go test -v -tags=integration ./test/integration/...

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests
	go test -v -tags=e2e ./test/e2e/...

.PHONY: test-coverage
test-coverage: test ## Generate test coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: benchmark
benchmark: ## Run benchmarks
	go test -bench=. -benchmem ./...

# Code quality targets
.PHONY: lint
lint: ## Run linter
	golangci-lint run

.PHONY: lint-fix
lint-fix: ## Run linter with auto-fix
	golangci-lint run --fix

.PHONY: fmt
fmt: ## Format code
	go fmt ./...
	goimports -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: mod-tidy
mod-tidy: ## Tidy go modules
	go mod tidy

.PHONY: mod-verify
mod-verify: ## Verify go modules
	go mod verify

# Code generation targets
.PHONY: generate
generate: ## Run go generate
	go generate ./...

.PHONY: sqlc-generate
sqlc-generate: ## Generate SQL code with sqlc
	@if [ -f sqlc.yaml ]; then sqlc generate; else echo "sqlc.yaml not found, skipping SQL generation"; fi

.PHONY: templ-generate
templ-generate: ## Generate templates with templ
	@echo "Generating templates..."
	@find . -name "*.templ" -exec templ generate {} \;

# Database targets
.PHONY: migrate-up
migrate-up: ## Run database migrations up
	go run $(MAIN_PATH) migrate up

.PHONY: migrate-down
migrate-down: ## Run database migrations down
	go run $(MAIN_PATH) migrate down

.PHONY: migrate-status
migrate-status: ## Show migration status
	go run $(MAIN_PATH) migrate status

# Docker targets
.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

.PHONY: docker-run
docker-run: ## Run Docker container
	docker run --rm -p 8080:8080 $(DOCKER_IMAGE):latest

.PHONY: docker-push
docker-push: ## Push Docker image
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

# Kubernetes targets
.PHONY: k8s-deploy
k8s-deploy: ## Deploy to Kubernetes
	kubectl apply -f deployments/k8s/

.PHONY: k8s-delete
k8s-delete: ## Delete from Kubernetes
	kubectl delete -f deployments/k8s/

.PHONY: kind-create
kind-create: ## Create Kind cluster
	@if [ -f deployments/kind/cluster.yaml ]; then \
		kind create cluster --config deployments/kind/cluster.yaml; \
	else \
		kind create cluster; \
	fi

.PHONY: kind-delete
kind-delete: ## Delete Kind cluster
	kind delete cluster

# Cleanup targets
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean -cache
	go clean -testcache

.PHONY: clean-all
clean-all: clean ## Clean everything including dependencies
	go clean -modcache

# Security targets
.PHONY: security-scan
security-scan: ## Run security scan
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec ./...

.PHONY: deps-check
deps-check: ## Check for dependency vulnerabilities
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

# Documentation targets
.PHONY: docs
docs: ## Generate documentation
	@echo "Generating documentation..."
	@which godoc > /dev/null || (echo "Installing godoc..." && go install golang.org/x/tools/cmd/godoc@latest)
	@echo "Documentation server will be available at http://localhost:6060"
	godoc -http=:6060

# Release targets
.PHONY: release
release: clean lint test build-all ## Prepare release
	@echo "Release $(VERSION) prepared in $(BUILD_DIR)/"

# Environment targets
.PHONY: env-example
env-example: ## Create .env.example file
	@echo "Creating .env.example..."
	@echo "# GoAssistant Configuration" > .env.example
	@echo "APP_MODE=development" >> .env.example
	@echo "LOG_LEVEL=info" >> .env.example
	@echo "LOG_FORMAT=json" >> .env.example
	@echo "" >> .env.example
	@echo "# Database" >> .env.example
	@echo "DATABASE_URL=postgres://user:password@localhost:5432/goassistant?sslmode=disable" >> .env.example
	@echo "" >> .env.example
	@echo "# Server" >> .env.example
	@echo "SERVER_ADDRESS=:8080" >> .env.example
	@echo "" >> .env.example
	@echo "# AI Providers (at least one required)" >> .env.example
	@echo "CLAUDE_API_KEY=your_claude_api_key_here" >> .env.example
	@echo "GEMINI_API_KEY=your_gemini_api_key_here" >> .env.example
	@echo "" >> .env.example
	@echo "# Tools Configuration" >> .env.example
	@echo "SEARXNG_URL=http://localhost:8888" >> .env.example
	@echo "KUBECONFIG=/path/to/kubeconfig" >> .env.example
	@echo "DOCKER_HOST=unix:///var/run/docker.sock" >> .env.example
	@echo "" >> .env.example
	@echo "# Cloudflare (optional)" >> .env.example
	@echo "CLOUDFLARE_API_TOKEN=your_cloudflare_api_token" >> .env.example
	@echo "CLOUDFLARE_ACCOUNT_ID=your_account_id" >> .env.example
	@echo "CLOUDFLARE_ZONE_ID=your_zone_id" >> .env.example
	@echo ".env.example created!"

# Check if required tools are installed
.PHONY: check-tools
check-tools: ## Check if required tools are installed
	@echo "Checking required tools..."
	@which go > /dev/null || (echo "Go is not installed" && exit 1)
	@which docker > /dev/null || (echo "Docker is not installed" && exit 1)
	@which kubectl > /dev/null || echo "kubectl is not installed (optional for K8s features)"
	@which kind > /dev/null || echo "kind is not installed (optional for local K8s)"
	@echo "Tool check complete!"

# Show project status
.PHONY: status
status: ## Show project status
	@echo "GoAssistant Project Status"
	@echo "========================="
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo ""
	@echo "Go version: $(shell go version)"
	@echo "Module: $(shell go list -m)"
	@echo ""
	@echo "Dependencies:"
	@go list -m all | head -10
	@echo ""
	@echo "Build artifacts:"
	@ls -la $(BUILD_DIR)/ 2>/dev/null || echo "No build artifacts found"
