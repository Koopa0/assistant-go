#!/bin/bash

# Go Project Status Check Script
# Quickly checks the health status of the project

set -e

# Color Definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Log Functions
log_header() {
    echo -e "\n${CYAN}🔍 $1${NC}"
    echo "----------------------------------------"
}

log_info() {
    echo -e "${BLUE}   $1${NC}"
}

log_success() {
    echo -e "${GREEN}   ✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}   ⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}   ❌ $1${NC}"
}

# Check Go Environment
check_go_environment() {
    log_header "Go Environment Check"

    if command -v go &> /dev/null; then
        local go_version=$(go version)
        log_success "Go is installed: $go_version"

        local go_mod_version=$(grep "^go " go.mod | awk '{print $2}')
        log_info "Project Go version: $go_mod_version"
    else
        log_error "Go is not installed or not in PATH"
        return 1
    fi
}

# Check Module Status
check_module_status() {
    log_header "Go Module Status"

    if [[ -f go.mod ]]; then
        log_success "go.mod exists"

        local module_name=$(grep "^module " go.mod | awk '{print $2}')
        log_info "Module name: $module_name"

        # Check go.sum
        if [[ -f go.sum ]]; then
            log_success "go.sum exists"
            local sum_entries=$(wc -l < go.sum)
            log_info "Dependency checksum entries: $sum_entries"
        else
            log_warning "go.sum does not exist"
        fi

        # Check module verification
        if go mod verify &> /dev/null; then
            log_success "Module verification passed"
        else
            log_error "Module verification failed"
        fi
    else
        log_error "go.mod does not exist"
        return 1
    fi
}

# Check Dependency Status
check_dependencies() {
    log_header "Core Dependency Check"

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
        if go list -m "$dep" &> /dev/null; then
            local version=$(go list -m "$dep" | awk '{print $2}')
            log_success "$dep $version"
        else
            log_error "$dep not found"
        fi
    done
}

# Check Replace Directives
check_replace_directives() {
    log_header "Replace Directive Check"

    if grep -q "^replace " go.mod; then
        log_info "Found replace directives:"
        while IFS= read -r line; do
            if [[ $line =~ ^replace ]]; then
                log_warning "  $line"
            fi
        done < go.mod
    else
        log_success "No replace directives"
    fi
}

# Check Build Status
check_build_status() {
    log_header "Build Status Check"

    if go build ./... &> /dev/null; then
        log_success "Project build successful"
    else
        log_error "Project build failed"
        log_info "Run 'go build ./...' to see detailed errors"
        return 1
    fi
}

# Check Test Status
check_test_status() {
    log_header "Test Status Check"

    # Check if there are any test files
    if find . -name "*_test.go" -type f | head -1 | grep -q .; then
        log_info "Test files found"

        if go test ./... &> /dev/null; then
            log_success "All tests passed"
        else
            log_warning "Some tests failed"
            log_info "Run 'go test ./...' to see detailed results"
        fi
    else
        log_warning "No test files found"
    fi
}

# Check Project Structure
check_project_structure() {
    log_header "Project Structure Check"

    local important_dirs=(
        "cmd"
        "internal"
        "pkg"
        "scripts"
        "docs"
    )

    for dir in "${important_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            local file_count=$(find "$dir" -type f | wc -l)
            log_success "$dir/ ($file_count files)"
        else
            log_info "$dir/ does not exist"
        fi
    done

    # Check important files
    local important_files=(
        "README.md"
        "Makefile"
        ".gitignore"
        ".env.example"
    )

    for file in "${important_files[@]}"; do
        if [[ -f "$file" ]]; then
            log_success "$file"
        else
            log_info "$file does not exist"
        fi
    done
}

# Check Git Status
check_git_status() {
    log_header "Git Status Check"

    if git rev-parse --git-dir &> /dev/null; then
        log_success "Git repository initialized"

        local branch=$(git branch --show-current)
        log_info "Current branch: $branch"

        local status=$(git status --porcelain)
        if [[ -z "$status" ]]; then
            log_success "Working directory clean"
        else
            local changed_files=$(echo "$status" | wc -l)
            log_warning "There are $changed_files uncommitted files"
        fi

        # Check remote repository
        if git remote -v | grep -q .; then
            local remote=$(git remote -v | head -1 | awk '{print $2}')
            log_info "Remote repository: $remote"
        else
            log_warning "Remote repository not configured"
        fi
    else
        log_warning "Not a Git repository"
    fi
}

# Generate Summary Report
generate_summary() {
    log_header "Status Summary"

    echo -e "${CYAN}📊 Project Health Assessment:${NC}"

    # Calculate health score (simple example)
    local score=0
    local total=6

    # Go Environment
    if command -v go &> /dev/null; then ((score++)); fi

    # Module Status
    if [[ -f go.mod ]] && go mod verify &> /dev/null; then ((score++)); fi

    # Build Status
    if go build ./... &> /dev/null; then ((score++)); fi

    # Project Structure
    if [[ -d cmd ]] || [[ -d internal ]]; then ((score++)); fi

    # Git Status
    if git rev-parse --git-dir &> /dev/null; then ((score++)); fi

    # Documentation
    if [[ -f README.md ]]; then ((score++)); fi

    local percentage=$((score * 100 / total))

    if [[ $percentage -ge 80 ]]; then
        log_success "Health: $percentage% ($score/$total) - Excellent"
    elif [[ $percentage -ge 60 ]]; then
        log_warning "Health: $percentage% ($score/$total) - Good"
    else
        log_error "Health: $percentage% ($score/$total) - Needs Improvement"
    fi

    echo -e "\n${CYAN}🚀 Recommended Actions:${NC}"
    echo -e "${BLUE}   • Run dependency verification: go run scripts/verify-dependencies.go${NC}"
    echo -e "${BLUE}   • Upgrade dependencies: ./scripts/upgrade-dependencies.sh core${NC}"
    echo -e "${BLUE}   • View detailed documentation: docs/DEPENDENCY_MANAGEMENT.md${NC}"
}

# Main Function
main() {
    echo -e "${CYAN}🔍 Assistant-Go Project Status Check${NC}"
    echo -e "${CYAN}======================================${NC}"

    check_go_environment
    check_module_status
    check_dependencies
    check_replace_directives
    check_build_status
    check_test_status
    check_project_structure
    check_git_status
    generate_summary

    echo -e "\n${GREEN}✅ Status check complete!${NC}"
}

# Execute Main Function
main "$@"