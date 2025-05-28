#!/bin/bash

# Go Dependency Upgrade Script
# Usage: ./scripts/upgrade-dependencies.sh [category]
# Categories: core, tools, web, all

set -e

# Color Definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Log Functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check Go Environment
check_go_env() {
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi

    log_info "Go version: $(go version)"
}

# Create Backup
create_backup() {
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_dir="backups/deps_$timestamp"

    log_info "Creating backup to $backup_dir"
    mkdir -p "$backup_dir"
    cp go.mod "$backup_dir/"
    cp go.sum "$backup_dir/"

    echo "$backup_dir" > .last_backup
    log_success "Backup created: $backup_dir"
}

# Restore Backup
restore_backup() {
    if [[ -f .last_backup ]]; then
        local backup_dir=$(cat .last_backup)
        if [[ -d "$backup_dir" ]]; then
            log_warning "Restoring backup: $backup_dir"
            cp "$backup_dir/go.mod" .
            cp "$backup_dir/go.sum" .
            log_success "Backup restored"
            return 0
        fi
    fi
    log_error "No recoverable backup found"
    return 1
}

# Verify Build
verify_build() {
    log_info "Verifying build..."
    if go build ./...; then
        log_success "Build successful"
        return 0
    else
        log_error "Build failed"
        return 1
    fi
}

# Run Dependency Verification
verify_dependencies() {
    log_info "Running dependency verification..."
    if [[ -f scripts/verify-dependencies.go ]]; then
        if go run scripts/verify-dependencies.go; then
            log_success "Dependency verification passed"
            return 0
        else
            log_error "Dependency verification failed"
            return 1
        fi
    else
        log_warning "Dependency verification script not found, skipping verification"
        return 0
    fi
}

# Upgrade Core Dependencies
upgrade_core() {
    log_info "Upgrading core application dependencies..."

    local core_deps=(
        "github.com/a-h/templ"
        "github.com/google/uuid"
        "github.com/jackc/pgx/v5"
        "github.com/joho/godotenv"
        "github.com/pgvector/pgvector-go"
        "github.com/tmc/langchaingo"
        "gopkg.in/yaml.v3"
    )

    for dep in "${core_deps[@]}"; do
        log_info "Upgrading $dep"
        go get -u "$dep"
    done

    log_success "Core dependency upgrade complete"
}

# Upgrade Tool Dependencies
upgrade_tools() {
    log_info "Upgrading tool library dependencies..."

    local tool_deps=(
        "golang.org/x/exp"
        "go.starlark.net"
        "github.com/AssemblyAI/assemblyai-go-sdk"
        "github.com/PuerkitoBio/goquery"
        "github.com/ledongthuc/pdf"
        "github.com/microcosm-cc/bluemonday"
    )

    for dep in "${tool_deps[@]}"; do
        log_info "Upgrading $dep"
        go get -u "$dep"
    done

    log_success "Tool dependency upgrade complete"
}

# Upgrade Web Dependencies
upgrade_web() {
    log_info "Upgrading template and web dependencies..."

    local web_deps=(
        "github.com/Masterminds/sprig/v3"
        "github.com/nikolalohinski/gonja"
        "nhooyr.io/websocket"
    )

    for dep in "${web_deps[@]}"; do
        log_info "Upgrading $dep"
        go get -u "$dep"
    done

    log_success "Web dependency upgrade complete"
}

# Cleanup and Tidy
cleanup() {
    log_info "Cleaning and tidying modules..."
    go mod tidy
    log_success "Module tidying complete"
}

# Main Function
main() {
    local category="${1:-core}"

    log_info "Starting Go dependency upgrade (Category: $category)"

    # Check environment
    check_go_env

    # Create backup
    create_backup

    # Upgrade based on category
    case "$category" in
        "core")
            upgrade_core
            ;;
        "tools")
            upgrade_tools
            ;;
        "web")
            upgrade_web
            ;;
        "all")
            upgrade_core
            upgrade_tools
            upgrade_web
            ;;
        *)
            log_error "Unknown category: $category"
            log_info "Available categories: core, tools, web, all"
            exit 1
            ;;
    esac

    # Cleanup
    cleanup

    # Verification
    if verify_build && verify_dependencies; then
        log_success "🎉 Dependency upgrade completed successfully!"
        log_info "Backup location: $(cat .last_backup)"
    else
        log_error "Post-upgrade verification failed, restoring backup..."
        if restore_backup; then
            cleanup
            log_warning "Restored to pre-upgrade state"
        fi
        exit 1
    fi
}

# Handle interrupt signals
trap 'log_warning "Upgrade interrupted, restoring backup..."; restore_backup; cleanup; exit 1' INT TERM

# Execute Main Function
main "$@"