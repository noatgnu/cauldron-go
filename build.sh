#!/bin/bash

set -e

PROJECT_ROOT="/mnt/d/GoLandProjects/cauldronGO/cauldronGO"
FRONTEND_DIR="$PROJECT_ROOT/frontend"

print_header() {
    echo ""
    echo "=================================="
    echo "$1"
    echo "=================================="
    echo ""
}

print_success() {
    echo -e "\033[0;32m✓ $1\033[0m"
}

print_error() {
    echo -e "\033[0;31m✗ $1\033[0m"
}

build_frontend() {
    print_header "Building Frontend"
    cd "$FRONTEND_DIR"

    if npm run build 2>&1 | tee /tmp/frontend-build.log; then
        print_success "Frontend build completed"
    else
        print_error "Frontend build failed"
        echo "Check /tmp/frontend-build.log for details"
        exit 1
    fi
}

copy_resources() {
    print_header "Copying Resources"
    cd "$PROJECT_ROOT"

    if [ ! -d "$PROJECT_ROOT/build/bin" ]; then
        print_error "Build bin directory not found"
        return 1
    fi

    local has_error=0

    if [ -d "$PROJECT_ROOT/examples" ]; then
        if cp -r "$PROJECT_ROOT/examples" "$PROJECT_ROOT/build/bin/"; then
            print_success "Examples copied successfully"
        else
            print_error "Failed to copy examples"
            has_error=1
        fi
    else
        print_error "Examples directory not found"
        has_error=1
    fi

    if [ -d "$PROJECT_ROOT/backend/scripts" ]; then
        if cp -r "$PROJECT_ROOT/backend/scripts" "$PROJECT_ROOT/build/bin/"; then
            print_success "Scripts copied successfully"
        else
            print_error "Failed to copy scripts"
            has_error=1
        fi
    else
        print_error "Scripts directory not found"
        has_error=1
    fi

    return $has_error
}

build_wails() {
    print_header "Building Wails Application"
    cd "$PROJECT_ROOT"

    PLATFORM="${1:-windows/amd64}"

    if ~/go/bin/wails build -platform "$PLATFORM" 2>&1 | tee /tmp/wails-build.log; then
        copy_resources || print_error "Warning: Failed to copy some resources, but build succeeded"
        print_success "Wails build completed for $PLATFORM"
        echo ""
        echo "Executable location: $PROJECT_ROOT/build/bin/"
        ls -lh "$PROJECT_ROOT/build/bin/"
    else
        print_error "Wails build failed"
        echo "Check /tmp/wails-build.log for details"
        exit 1
    fi
}

clean_build() {
    print_header "Cleaning Build Artifacts"
    cd "$PROJECT_ROOT"

    rm -rf "$FRONTEND_DIR/dist"
    rm -rf "$PROJECT_ROOT/build"

    print_success "Build artifacts cleaned"
}

show_help() {
    cat << EOF
Cauldron Build Script

Usage: ./build.sh [COMMAND] [OPTIONS]

Commands:
  frontend         Build only the frontend
  wails [PLATFORM] Build the Wails application (default: windows/amd64)
  all [PLATFORM]   Build both frontend and Wails app (default)
  clean            Clean all build artifacts
  rebuild [PLATFORM] Clean and rebuild everything
  help             Show this help message

Platforms:
  windows/amd64    Windows 64-bit (default)
  linux/amd64      Linux 64-bit
  darwin/amd64     macOS 64-bit (Intel)
  darwin/arm64     macOS 64-bit (Apple Silicon)

Examples:
  ./build.sh                    # Build everything for Windows
  ./build.sh frontend           # Build only frontend
  ./build.sh wails linux/amd64  # Build Wails app for Linux
  ./build.sh rebuild            # Clean and rebuild everything
  ./build.sh clean              # Clean build artifacts

EOF
}

case "${1:-all}" in
    frontend)
        build_frontend
        ;;
    wails)
        build_wails "${2:-windows/amd64}"
        ;;
    all)
        build_frontend
        build_wails "${2:-windows/amd64}"
        print_header "Build Complete!"
        print_success "Application ready to run"
        ;;
    clean)
        clean_build
        ;;
    rebuild)
        clean_build
        build_frontend
        build_wails "${2:-windows/amd64}"
        print_header "Rebuild Complete!"
        print_success "Application ready to run"
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        echo ""
        show_help
        exit 1
        ;;
esac
