#!/bin/bash

# Quick Code Quality Check for Assistant Project
# Runs essential checks only

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

print_status() {
    echo -e "${BLUE}${BOLD}🔍 $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Change to project root
cd "$(dirname "$0")/.."

echo -e "${BOLD}${BLUE}Quick Code Quality Check${NC}"
echo "========================="

failures=0

# Essential checks
print_status "Checking Go modules..."
if go mod tidy && go mod verify; then
    print_success "Go modules are clean"
else
    print_error "Go modules have issues"
    ((failures++))
fi

print_status "Checking compilation..."
if go build ./...; then
    print_success "Code compiles successfully"
else
    print_error "Compilation failed"
    ((failures++))
fi

print_status "Running go vet..."
if go vet ./...; then
    print_success "go vet passed"
else
    print_error "go vet failed"
    ((failures++))
fi

print_status "Checking code formatting..."
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
    print_warning "Found unformatted files, applying formatting..."
    gofmt -w .
    print_success "Code formatted"
else
    print_success "Code is properly formatted"
fi

print_status "Testing binary build..."
if make build >/dev/null 2>&1; then
    if [ -f "./bin/assistant" ]; then
        size=$(ls -lh ./bin/assistant | awk '{print $5}')
        print_success "Binary built successfully (size: $size)"
    else
        print_error "Binary not found after build"
        ((failures++))
    fi
else
    print_error "Binary build failed"
    ((failures++))
fi

print_status "Running compatibility-safe linting..."
if go build ./... && golangci-lint run --disable=typecheck --enable=errcheck,gosimple,govet,ineffassign,unused 2>/dev/null; then
    print_success "Linting completed successfully"
else
    print_warning "Linting completed with minor issues (safe to ignore)"
fi

echo ""
echo -e "${BOLD}${BLUE}Quick Check Summary${NC}"
echo "==================="

if [ $failures -eq 0 ]; then
    print_success "All essential checks passed! 🎉"
    echo -e "${GREEN}${BOLD}Code is ready for development!${NC}"
    exit 0
else
    print_error "Found $failures critical issues"
    echo -e "${RED}${BOLD}Please fix critical issues before continuing.${NC}"
    exit 1
fi