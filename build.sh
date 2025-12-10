#!/bin/bash

set -e

PROJECT_ROOT="/mnt/d/GoLandProjects/cauldronGO"
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

    if [ -d "$PROJECT_ROOT/scripts" ]; then
        if cp -r "$PROJECT_ROOT/scripts" "$PROJECT_ROOT/build/bin/"; then
            print_success "Scripts copied successfully"
        else
            print_error "Failed to copy scripts"
            has_error=1
        fi
    else
        print_error "Scripts directory not found"
        has_error=1
    fi

    if [ -d "$PROJECT_ROOT/plugins" ]; then
        if cp -r "$PROJECT_ROOT/plugins" "$PROJECT_ROOT/build/bin/"; then
            print_success "Plugins copied successfully"
        else
            print_error "Failed to copy plugins"
            has_error=1
        fi
    else
        print_error "Plugins directory not found"
        has_error=1
    fi

    if [ -d "$PROJECT_ROOT/bin" ]; then
        mkdir -p "$PROJECT_ROOT/build/bin/tools"
        if cp -r "$PROJECT_ROOT/bin/"* "$PROJECT_ROOT/build/bin/tools/"; then
            print_success "Developer tools copied successfully"
        else
            print_error "Failed to copy developer tools"
            has_error=1
        fi
    else
        print_error "Warning: Developer tools not found (run './build.sh tools' first)"
    fi

    return $has_error
}

build_dev_tools() {
    print_header "Building Developer Tools"
    cd "$PROJECT_ROOT"

    mkdir -p "$PROJECT_ROOT/bin"

    if go build -o bin/plugin-validator ./cmd/plugin-validator; then
        print_success "Built plugin-validator"
    else
        print_error "Failed to build plugin-validator"
        return 1
    fi

    if go build -o bin/plugin-doc-generator ./cmd/plugin-doc-generator; then
        print_success "Built plugin-doc-generator"
    else
        print_error "Failed to build plugin-doc-generator"
        return 1
    fi

    if go build -o bin/plugin-doc-generator-all ./cmd/plugin-doc-generator-all; then
        print_success "Built plugin-doc-generator-all"
    else
        print_error "Failed to build plugin-doc-generator-all"
        return 1
    fi

    if go build -o bin/plugin-scaffolder ./cmd/plugin-scaffolder; then
        print_success "Built plugin-scaffolder"
    else
        print_error "Failed to build plugin-scaffolder"
        return 1
    fi

    print_success "All developer tools built successfully"
}

build_external_tools() {
    print_header "Building External Utility Programs"
    cd "$PROJECT_ROOT"

    mkdir -p "$PROJECT_ROOT/bin/external"

    local platform="${1:-linux/amd64}"
    local os_part="${platform%/*}"
    local arch_part="${platform#*/}"

    local output_name="uniprot-fetcher"
    if [ "$os_part" = "windows" ]; then
        output_name="uniprot-fetcher.exe"
    fi

    if GOOS="$os_part" GOARCH="$arch_part" go build -o "bin/external/$output_name" ./cmd/uniprot-fetcher; then
        print_success "Built uniprot-fetcher for $platform"
    else
        print_error "Failed to build uniprot-fetcher"
        return 1
    fi

    print_success "All external utility programs built successfully for $platform"
}

prepare_icons() {
    print_header "Preparing Application Icons"
    cd "$PROJECT_ROOT"

    mkdir -p "$PROJECT_ROOT/build/windows"

    if [ -f "$PROJECT_ROOT/resources/appicon.png" ]; then
        cp "$PROJECT_ROOT/resources/appicon.png" "$PROJECT_ROOT/build/appicon.png"
        print_success "Application icon copied"
    else
        print_error "Warning: resources/appicon.png not found"
    fi

    if [ -f "$PROJECT_ROOT/resources/icon.ico" ]; then
        cp "$PROJECT_ROOT/resources/icon.ico" "$PROJECT_ROOT/build/windows/icon.ico"
        print_success "Windows icon copied"
    else
        print_error "Warning: resources/icon.ico not found"
    fi
}

build_wails() {
    PLATFORM="${1:-windows/amd64}"

    build_external_tools "$PLATFORM"
    prepare_icons
    print_header "Building Wails Application"
    cd "$PROJECT_ROOT"

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
  frontend              Build only the frontend
  wails [PLATFORM]      Build the Wails application (default: windows/amd64)
  tools                 Build developer tools (plugin-validator, etc.)
  external [PLATFORM]   Build external utility programs (uniprot-fetcher, etc.)
  all [PLATFORM]        Build frontend, tools, and Wails app (default)
  clean                 Clean all build artifacts
  rebuild [PLATFORM]    Clean and rebuild everything
  help                  Show this help message

Platforms:
  windows/amd64    Windows 64-bit (default)
  linux/amd64      Linux 64-bit
  darwin/amd64     macOS 64-bit (Intel)
  darwin/arm64     macOS 64-bit (Apple Silicon)

Examples:
  ./build.sh                       # Build everything for Windows
  ./build.sh frontend              # Build only frontend
  ./build.sh tools                 # Build developer tools
  ./build.sh external linux/amd64  # Build external utilities for Linux
  ./build.sh wails linux/amd64     # Build Wails app for Linux
  ./build.sh rebuild               # Clean and rebuild everything
  ./build.sh clean                 # Clean build artifacts

EOF
}

case "${1:-all}" in
    frontend)
        build_frontend
        ;;
    tools)
        build_dev_tools
        ;;
    external)
        build_external_tools "${2:-linux/amd64}"
        ;;
    wails)
        build_wails "${2:-windows/amd64}"
        ;;
    all)
        build_dev_tools
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
        build_dev_tools
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
