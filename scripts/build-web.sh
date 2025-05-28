#!/bin/bash

# Build script for GoAssistant web interface
# This script compiles Templ templates and builds Tailwind CSS

set -e

echo "🚀 Building GoAssistant Web Interface..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    print_error "go.mod not found. Please run this script from the project root."
    exit 1
fi

# Create necessary directories
print_status "Creating directories..."
mkdir -p internal/web/static/css
mkdir -p internal/web/static/js

# Install dependencies if needed
print_status "Checking dependencies..."

# Check for templ
if ! command -v templ &> /dev/null; then
    print_warning "templ not found. Installing..."
    go install github.com/a-h/templ/cmd/templ@latest
    if [ $? -ne 0 ]; then
        print_error "Failed to install templ"
        exit 1
    fi
    print_success "templ installed successfully"
fi

# Check for Tailwind CSS
if [ ! -f "node_modules/.bin/tailwindcss" ] && ! command -v tailwindcss &> /dev/null; then
    print_warning "Tailwind CSS not found. Installing..."
    
    # Check if npm is available
    if command -v npm &> /dev/null; then
        npm install -D tailwindcss
        if [ $? -ne 0 ]; then
            print_error "Failed to install Tailwind CSS via npm"
            exit 1
        fi
    else
        # Download standalone Tailwind CSS
        print_status "Downloading standalone Tailwind CSS..."
        curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-macos-x64
        chmod +x tailwindcss-macos-x64
        mv tailwindcss-macos-x64 /usr/local/bin/tailwindcss
    fi
    print_success "Tailwind CSS installed successfully"
fi

# Generate Templ templates
print_status "Generating Templ templates..."
templ generate
if [ $? -ne 0 ]; then
    print_error "Failed to generate Templ templates"
    exit 1
fi
print_success "Templ templates generated successfully"

# Build Tailwind CSS
print_status "Building Tailwind CSS..."

# Use local tailwindcss if available, otherwise use global
TAILWIND_CMD="tailwindcss"
if [ -f "node_modules/.bin/tailwindcss" ]; then
    TAILWIND_CMD="./node_modules/.bin/tailwindcss"
fi

$TAILWIND_CMD -i internal/web/static/css/material-design.css -o internal/web/static/css/tailwind.css --watch=false
if [ $? -ne 0 ]; then
    print_error "Failed to build Tailwind CSS"
    exit 1
fi
print_success "Tailwind CSS built successfully"

# Download HTMX if not present
if [ ! -f "internal/web/static/js/htmx.min.js" ]; then
    print_status "Downloading HTMX..."
    curl -sL https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js -o internal/web/static/js/htmx.min.js
    if [ $? -ne 0 ]; then
        print_error "Failed to download HTMX"
        exit 1
    fi
    print_success "HTMX downloaded successfully"
fi

# Build Go application
print_status "Building Go application..."
go build -o bin/goassistant ./cmd/assistant
if [ $? -ne 0 ]; then
    print_error "Failed to build Go application"
    exit 1
fi
print_success "Go application built successfully"

# Verify build
print_status "Verifying build..."

# Check if templates were generated
if [ ! -f "internal/web/templates/layouts/base_templ.go" ]; then
    print_error "Templ templates not generated properly"
    exit 1
fi

# Check if CSS was built
if [ ! -f "internal/web/static/css/tailwind.css" ]; then
    print_error "Tailwind CSS not built properly"
    exit 1
fi

# Check if binary was created
if [ ! -f "bin/goassistant" ]; then
    print_error "Go binary not created"
    exit 1
fi

print_success "Build verification completed"

# Print summary
echo ""
echo "🎉 Build completed successfully!"
echo ""
echo "Generated files:"
echo "  📄 Templ templates: internal/web/templates/**/*_templ.go"
echo "  🎨 Tailwind CSS: internal/web/static/css/tailwind.css"
echo "  📦 Binary: bin/goassistant"
echo ""
echo "To run the application:"
echo "  ./bin/goassistant"
echo ""
echo "To watch for changes during development:"
echo "  templ generate --watch &"
echo "  $TAILWIND_CMD -i internal/web/static/css/material-design.css -o internal/web/static/css/tailwind.css --watch &"
echo "  go run ./cmd/assistant"
echo ""

# Optional: Start development mode
read -p "Start development mode with file watching? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_status "Starting development mode..."
    
    # Start Templ watcher in background
    templ generate --watch &
    TEMPL_PID=$!
    
    # Start Tailwind watcher in background
    $TAILWIND_CMD -i internal/web/static/css/material-design.css -o internal/web/static/css/tailwind.css --watch &
    TAILWIND_PID=$!
    
    # Function to cleanup background processes
    cleanup() {
        print_status "Stopping watchers..."
        kill $TEMPL_PID 2>/dev/null || true
        kill $TAILWIND_PID 2>/dev/null || true
        exit 0
    }
    
    # Set trap to cleanup on script exit
    trap cleanup SIGINT SIGTERM
    
    print_success "Development mode started!"
    print_status "Templ watcher PID: $TEMPL_PID"
    print_status "Tailwind watcher PID: $TAILWIND_PID"
    print_status "Press Ctrl+C to stop watchers and exit"
    
    # Run the application
    go run ./cmd/assistant
    
    # Cleanup when application exits
    cleanup
fi
