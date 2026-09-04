#!/bin/bash

set -e
set -o pipefail

PROJECT_ROOT=$(cd "$(dirname "$0")" && pwd)
FRONTEND_DIR="$PROJECT_ROOT/frontend"
SHARED_LIB_DIR="$PROJECT_ROOT/shared-lib"
SKIP_LICENSES=false
COMMAND=""
PLATFORM=""

WAILS_CMD="${WAILS_CMD:-wails3}"

VERSION="${VERSION:-$(git -C "$PROJECT_ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)}"
LDFLAGS="-s -w -X main.appVersion=$VERSION"

MINGW_CC_AMD64="${MINGW_CC_AMD64:-x86_64-w64-mingw32-gcc}"
MINGW_CC_386="${MINGW_CC_386:-i686-w64-mingw32-gcc}"
OSX_CC_AMD64="${OSX_CC_AMD64:-o64-clang}"
OSX_CC_ARM64="${OSX_CC_ARM64:-oa64-clang}"

for arg in "$@"; do
    if [ "$arg" = "--skip-licenses" ]; then
        SKIP_LICENSES=true
    elif [ -z "$COMMAND" ]; then
        COMMAND="$arg"
    elif [ -z "$PLATFORM" ]; then
        PLATFORM="$arg"
    fi
done

if [ -z "$COMMAND" ]; then
    COMMAND="all"
fi

if [ -z "$PLATFORM" ]; then
    PLATFORM="windows/amd64"
fi

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

print_info() {
    echo -e "\033[0;34mℹ $1\033[0m"
}

check_cross_compiler() {
    local compiler="$1"
    if command -v "$compiler" &> /dev/null; then
        return 0
    fi
    return 1
}

needs_gtk3_tag() {
    if command -v pkg-config &> /dev/null && pkg-config --exists gtk4 2>/dev/null; then
        if pkg-config --atleast-version=4.10 gtk4; then
            return 1
        fi
        return 0
    fi
    return 1
}

get_windows_cc() {
    local arch="$1"
    if [ "$arch" = "amd64" ]; then
        if check_cross_compiler "$MINGW_CC_AMD64"; then
            echo "$MINGW_CC_AMD64"
            return 0
        fi
    elif [ "$arch" = "386" ]; then
        if check_cross_compiler "$MINGW_CC_386"; then
            echo "$MINGW_CC_386"
            return 0
        fi
    fi
    return 1
}

get_darwin_cc() {
    local arch="$1"
    if [ "$arch" = "amd64" ]; then
        if check_cross_compiler "$OSX_CC_AMD64"; then
            echo "$OSX_CC_AMD64"
            return 0
        fi
    elif [ "$arch" = "arm64" ]; then
        if check_cross_compiler "$OSX_CC_ARM64"; then
            echo "$OSX_CC_ARM64"
            return 0
        fi
    fi
    return 1
}

create_placeholder_frontend() {
    print_header "Creating Placeholder Frontend for Bindings"
    cd "$PROJECT_ROOT"

    mkdir -p "$FRONTEND_DIR/dist/browser"
    echo "<!DOCTYPE html><html><head><title>Placeholder</title></head><body>Placeholder</body></html>" > "$FRONTEND_DIR/dist/browser/index.html"

    print_success "Placeholder frontend created"
}

generate_bindings() {
    print_header "Generating Wails v3 TypeScript Bindings"
    cd "$PROJECT_ROOT"

    if $WAILS_CMD generate bindings -ts 2>&1 | tee /tmp/bindings-gen.log; then
        print_success "TypeScript bindings generated successfully"
        return 0
    else
        print_error "Bindings generation failed"
        echo "Check /tmp/bindings-gen.log for details"
        return 1
    fi
}

build_shared_lib() {
    print_header "Building Shared Library"
    cd "$SHARED_LIB_DIR"

    if [ ! -d "$SHARED_LIB_DIR/node_modules" ]; then
        echo "Installing shared library dependencies..."
        npm install --silent
    fi

    if npm run build 2>&1 | tee /tmp/shared-lib-build.log; then
        print_success "Shared library built successfully"
        return 0
    else
        print_error "Shared library build failed"
        echo "Check /tmp/shared-lib-build.log for details"
        return 1
    fi
}

build_frontend() {
    print_header "Building Frontend"
    cd "$FRONTEND_DIR"

    if [ ! -d "$FRONTEND_DIR/node_modules" ]; then
        echo "Installing frontend dependencies..."
        npm install --silent
    fi

    npm run build 2>&1 | tee /tmp/frontend-build.log
    local build_exit_code=$?

    if grep -qE "(has no exported member|does not exist on type|Cannot find name)" /tmp/frontend-build.log; then
        print_error "Frontend build has missing Wails bindings errors"
        echo "Attempting to bootstrap bindings generation..."

        create_placeholder_frontend

        if generate_bindings; then
            print_success "Bindings generated, retrying frontend build..."

            cd "$FRONTEND_DIR"
            if npm run build 2>&1 | tee /tmp/frontend-build-retry.log; then
                if grep -qE "(has no exported member|does not exist on type|Cannot find name)" /tmp/frontend-build-retry.log; then
                    print_error "Frontend build still has binding errors after retry"
                    echo "Check /tmp/frontend-build-retry.log for details"
                    exit 1
                else
                    print_success "Frontend build completed successfully on retry"
                    return 0
                fi
            else
                print_error "Frontend build failed after bindings generation"
                echo "Check /tmp/frontend-build-retry.log for details"
                exit 1
            fi
        else
            print_error "Failed to generate bindings"
            exit 1
        fi
    elif [ $build_exit_code -ne 0 ]; then
        print_error "Frontend build failed with exit code $build_exit_code"
        echo "Check /tmp/frontend-build.log for details"
        exit 1
    else
        print_success "Frontend build completed"
        return 0
    fi
}

copy_resources() {
    local target_dir="${1:-$PROJECT_ROOT/build/bin}"
    local platform="${2:-$(go env GOOS)/$(go env GOARCH)}"
    local os_part="${platform%/*}"
    local arch_part="${platform#*/}"

    print_header "Copying Resources for $platform to $target_dir"
    cd "$PROJECT_ROOT"

    if [ ! -d "$target_dir" ]; then
        mkdir -p "$target_dir"
    fi

    local has_error=0

    local resources=("examples" "scripts" "plugins")
    for res in "${resources[@]}"; do
        if [ -d "$PROJECT_ROOT/$res" ]; then
            if cp -r "$PROJECT_ROOT/$res" "$target_dir/"; then
                print_success "$res copied successfully"
            else
                print_error "Failed to copy $res"
                has_error=1
            fi
        fi
    done

    if [ -d "$PROJECT_ROOT/schemas" ]; then
        mkdir -p "$target_dir/schemas"
        if cp "$PROJECT_ROOT/schemas/"*.json "$target_dir/schemas/" 2>/dev/null; then
            print_success "Schema JSON files copied successfully"
        else
            print_error "Failed to copy schema JSON files"
            has_error=1
        fi
    fi

    local external_tools=(
        "uniprot-fetcher"
        "wide-to-long"
        "long-to-wide"
    )
    for tool in "${external_tools[@]}"; do
        local exe_name="$tool"
        if [ "$os_part" = "windows" ]; then
            exe_name="$tool.exe"
        fi

        if [ -f "$PROJECT_ROOT/bin/external/$os_part-$arch_part/$exe_name" ]; then
            mkdir -p "$target_dir/plugins/$tool"
            if cp "$PROJECT_ROOT/bin/external/$os_part-$arch_part/$exe_name" "$target_dir/plugins/$tool/"; then
                print_success "$tool executable copied to plugin directory"
            else
                print_error "Failed to copy $tool executable"
                has_error=1
            fi
        elif [ -f "$PROJECT_ROOT/bin/external/$exe_name" ]; then
            mkdir -p "$target_dir/plugins/$tool"
            if cp "$PROJECT_ROOT/bin/external/$exe_name" "$target_dir/plugins/$tool/"; then
                print_success "$tool executable copied to plugin directory (from legacy path)"
            else
                print_error "Failed to copy $tool executable"
                has_error=1
            fi
        fi
    done

    if [ -d "$PROJECT_ROOT/bin/$os_part-$arch_part" ]; then
        mkdir -p "$target_dir/tools"
        (
            cd "$PROJECT_ROOT/bin/$os_part-$arch_part"
            cp * "$target_dir/tools/"
        )
        print_success "Developer tools copied successfully for $platform"
    fi

    return $has_error
}

generate_licenses() {
    print_header "Generating License Information"
    cd "$PROJECT_ROOT"

    mkdir -p resources/licenses

    if command -v node &> /dev/null; then
        node scripts/generate-go-licenses.js

        if [ ! -d "$FRONTEND_DIR/node_modules" ]; then
            echo "Installing frontend dependencies for license generation..."
            cd "$FRONTEND_DIR"
            npm install --silent
            cd "$PROJECT_ROOT"
        fi
        node scripts/generate-npm-licenses.js
        print_success "License information generated"
    else
        echo "Warning: Node.js not found, skipping license generation"
        echo "[]" > resources/licenses/go-licenses.json
        echo "[]" > resources/licenses/npm-licenses.json
    fi
}

build_dev_tools() {
    local platform="${1:-$(go env GOOS)/$(go env GOARCH)}"
    local os_part="${platform%/*}"
    local arch_part="${platform#*/}"

    print_header "Building Developer Tools for $platform"
    cd "$PROJECT_ROOT"

    local output_base="$PROJECT_ROOT/bin/$os_part-$arch_part"
    mkdir -p "$output_base"

    local exe_ext=""
    if [ "$os_part" = "windows" ]; then
        exe_ext=".exe"
    fi

    local tools=(
        "plugin-validator"
        "plugin-doc-generator"
        "plugin-doc-generator-all"
        "plugin-scaffolder"
        "plugin-migrate"
        "plugin-to-nextflow"
        "plugin-to-slivka"
        "plugin-to-spa"
        "schema-doc-generator"
        "webr-resolver"
    )

    for tool in "${tools[@]}"; do
        if [ -d "./cmd/$tool" ]; then
            if GOOS="$os_part" GOARCH="$arch_part" go build -o "$output_base/$tool$exe_ext" "./cmd/$tool"; then
                print_success "Built $tool for $platform"
            else
                print_error "Failed to build $tool for $platform"
                return 1
            fi
        fi
    done

    print_success "All developer tools built successfully for $platform"
}

build_external_tools() {
    local platform="${1:-$(go env GOOS)/$(go env GOARCH)}"
    local os_part="${platform%/*}"
    local arch_part="${platform#*/}"

    print_header "Building External Utility Programs for $platform"
    cd "$PROJECT_ROOT"

    local output_base="$PROJECT_ROOT/bin/external/$os_part-$arch_part"
    mkdir -p "$output_base"

    local exe_ext=""
    if [ "$os_part" = "windows" ]; then
        exe_ext=".exe"
    fi

    local tools=(
        "uniprot-fetcher"
        "wide-to-long"
        "long-to-wide"
    )

    for tool in "${tools[@]}"; do
        if [ -d "./cmd/$tool" ]; then
            if GOOS="$os_part" GOARCH="$arch_part" go build -o "$output_base/$tool$exe_ext" "./cmd/$tool"; then
                print_success "Built $tool for $platform"
            else
                print_error "Failed to build $tool for $platform"
                return 1
            fi
        fi
    done

    print_success "All external utility programs built successfully for $platform"
}

generate_icons() {
    print_header "Generating Application Icons"
    cd "$PROJECT_ROOT"

    if [ -x "$PROJECT_ROOT/scripts/generate-icons.sh" ]; then
        if "$PROJECT_ROOT/scripts/generate-icons.sh" all 2>&1; then
            print_success "Icons generated successfully"
            return 0
        else
            print_error "Icon generation failed, using existing icons"
        fi
    else
        print_error "Icon generation script not found"
    fi

    return 0
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

    if [ -f "$PROJECT_ROOT/rsrc_windows_amd64.syso" ]; then
        print_success "Windows resource file (amd64) found"
    fi
}

build_wails() {
    PLATFORM="${1:-windows/amd64}"

    if [ ! -f "$PROJECT_ROOT/resources/licenses/go-licenses.json" ] || [ ! -f "$PROJECT_ROOT/resources/licenses/npm-licenses.json" ]; then
        generate_licenses
    fi

    build_external_tools "$PLATFORM"
    prepare_icons
    print_header "Building Wails v3 Application"
    cd "$PROJECT_ROOT"

    local os_part="${PLATFORM%/*}"
    local arch_part="${PLATFORM#*/}"
    local current_os="$(go env GOOS)"
    local current_arch="$(go env GOARCH)"
    local platform_dir="$PROJECT_ROOT/build/bin/${os_part}-${arch_part}"

    mkdir -p "$platform_dir"

    local output_name="cauldron"
    if [ "$os_part" = "windows" ]; then
        output_name="cauldron.exe"
    fi

    echo "Building for $os_part/$arch_part..."

    if [ "$os_part" = "$current_os" ] && [ "$arch_part" = "$current_arch" ]; then
        echo "Native build detected"
        local extra_tags=""
        if [ "$os_part" = "linux" ] && needs_gtk3_tag; then
            print_info "Legacy GTK4 detected (< 4.10, missing GtkFileDialog) -- building with -tags gtk3"
            extra_tags="-tags gtk3"
        fi
        if go build $extra_tags -ldflags="$LDFLAGS" -o "$platform_dir/$output_name" . 2>&1 | tee /tmp/wails-build.log; then
            if grep -q "undefined:" /tmp/wails-build.log; then
                print_error "Wails v3 build failed with errors"
                echo "Check /tmp/wails-build.log for details"
                exit 1
            fi
            copy_resources "$platform_dir" "$PLATFORM" || print_error "Warning: Failed to copy some resources, but build succeeded"
            print_success "Wails v3 build completed for $PLATFORM"
            echo ""
            echo "Executable location: $platform_dir/"
            ls -lh "$platform_dir/"
        else
            print_error "Wails v3 build failed"
            echo "Check /tmp/wails-build.log for details"
            exit 1
        fi
    elif [ "$os_part" = "windows" ]; then
        local windows_cc
        if windows_cc=$(get_windows_cc "$arch_part"); then
            echo "Cross-compiling to Windows with CGO (CC=$windows_cc)..."
            if CGO_ENABLED=1 CC="$windows_cc" GOOS="$os_part" GOARCH="$arch_part" go build -ldflags="$LDFLAGS" -o "$platform_dir/$output_name" . 2>&1 | tee /tmp/wails-build.log; then
                if grep -q "undefined:" /tmp/wails-build.log; then
                    print_error "Wails v3 build failed with errors"
                    echo "Check /tmp/wails-build.log for details"
                    exit 1
                fi
                copy_resources "$platform_dir" "$PLATFORM" || print_error "Warning: Failed to copy some resources, but build succeeded"
                print_success "Wails v3 build completed for $PLATFORM (CGO enabled)"
                echo ""
                echo "Executable location: $platform_dir/"
                ls -lh "$platform_dir/"
            else
                print_error "Wails v3 build failed"
                echo "Check /tmp/wails-build.log for details"
                exit 1
            fi
        else
            print_info "mingw-w64 not found, building without CGO..."
            if CGO_ENABLED=0 GOOS="$os_part" GOARCH="$arch_part" go build -ldflags="$LDFLAGS" -o "$platform_dir/$output_name" . 2>&1 | tee /tmp/wails-build.log; then
                if grep -q "undefined:" /tmp/wails-build.log; then
                    print_error "Wails v3 build failed with errors"
                    echo "Check /tmp/wails-build.log for details"
                    exit 1
                fi
                copy_resources "$platform_dir" "$PLATFORM" || print_error "Warning: Failed to copy some resources, but build succeeded"
                print_success "Wails v3 build completed for $PLATFORM (CGO disabled)"
                echo ""
                echo "Executable location: $platform_dir/"
                ls -lh "$platform_dir/"
            else
                print_error "Wails v3 build failed"
                echo "Check /tmp/wails-build.log for details"
                exit 1
            fi
        fi
    elif [ "$os_part" = "darwin" ]; then
        local darwin_cc
        if darwin_cc=$(get_darwin_cc "$arch_part"); then
            echo "Cross-compiling to macOS with CGO (CC=$darwin_cc)..."
            if CGO_ENABLED=1 CC="$darwin_cc" GOOS="$os_part" GOARCH="$arch_part" go build -ldflags="$LDFLAGS" -o "$platform_dir/$output_name" . 2>&1 | tee /tmp/wails-build.log; then
                if grep -q "undefined:" /tmp/wails-build.log; then
                    print_error "Wails v3 build failed with errors"
                    echo "Check /tmp/wails-build.log for details"
                    exit 1
                fi
                copy_resources "$platform_dir" "$PLATFORM" || print_error "Warning: Failed to copy some resources, but build succeeded"
                print_success "Wails v3 build completed for $PLATFORM (CGO enabled)"
                echo ""
                echo "Executable location: $platform_dir/"
                ls -lh "$platform_dir/"
            else
                print_error "Wails v3 build failed"
                echo "Check /tmp/wails-build.log for details"
                exit 1
            fi
        else
            print_error "Cross-compilation to macOS requires osxcross toolchain"
            echo ""
            echo "Install osxcross or build natively on macOS."
            echo "Set OSX_CC_AMD64 or OSX_CC_ARM64 environment variables if using custom compiler paths."
            exit 1
        fi
    else
        print_error "Cross-compilation to $os_part requires CGO and appropriate toolchain"
        echo ""
        echo "For Linux builds, you need:"
        echo "  - GCC cross-compiler (e.g., x86_64-linux-gnu-gcc)"
        echo "  - GTK3 development libraries"
        echo ""
        echo "Alternatively, build natively on the target platform."
        exit 1
    fi
}

build_wails_with_task() {
    PLATFORM="${1:-windows/amd64}"

    if [ ! -f "$PROJECT_ROOT/resources/licenses/go-licenses.json" ] || [ ! -f "$PROJECT_ROOT/resources/licenses/npm-licenses.json" ]; then
        generate_licenses
    fi

    build_external_tools "$PLATFORM"
    prepare_icons
    print_header "Building Wails v3 Application with Task"
    cd "$PROJECT_ROOT"

    local os_part="${PLATFORM%/*}"
    local arch_part="${PLATFORM#*/}"
    local task_name="build:${os_part}:${arch_part}"

    if command -v task &> /dev/null; then
        if task "$task_name" 2>&1 | tee /tmp/wails-build.log; then
            copy_resources "$PROJECT_ROOT/build/bin" "$PLATFORM" || print_error "Warning: Failed to copy some resources, but build succeeded"
            print_success "Wails v3 build completed for $PLATFORM"
            echo ""
            echo "Executable location: $PROJECT_ROOT/build/bin/"
            ls -lh "$PROJECT_ROOT/build/bin/"
        else
            print_error "Wails v3 build failed"
            echo "Check /tmp/wails-build.log for details"
            exit 1
        fi
    else
        echo "Task not found, falling back to direct go build..."
        build_wails "$PLATFORM"
    fi
}

run_dev() {
    print_header "Starting Development Server"
    cd "$PROJECT_ROOT"

    if command -v task &> /dev/null; then
        task dev
    else
        echo "Task not found, starting manually..."
        cd "$FRONTEND_DIR"
        npm run dev &
        FRONTEND_PID=$!
        cd "$PROJECT_ROOT"
        go run . &
        GO_PID=$!
        trap "kill $FRONTEND_PID $GO_PID 2>/dev/null" EXIT
        wait
    fi
}

build_all_platforms() {
    print_header "Building for All Platforms"

    local platforms=("windows/amd64" "linux/amd64" "darwin/amd64" "darwin/arm64")
    local failed=()
    local succeeded=()

    for platform in "${platforms[@]}"; do
        echo ""
        print_header "Building for $platform"

        if build_wails_platform "$platform"; then
            succeeded+=("$platform")
        else
            failed+=("$platform")
        fi
    done

    echo ""
    print_header "Build Summary"

    if [ ${#succeeded[@]} -gt 0 ]; then
        print_success "Successfully built for:"
        for p in "${succeeded[@]}"; do
            echo "  - $p"
        done
    fi

    if [ ${#failed[@]} -gt 0 ]; then
        print_error "Failed to build for:"
        for p in "${failed[@]}"; do
            echo "  - $p"
        done
    fi

    echo ""
    echo "Build artifacts location: $PROJECT_ROOT/build/bin/"
    ls -lh "$PROJECT_ROOT/build/bin/" 2>/dev/null || true
}

build_wails_platform() {
    local PLATFORM="$1"
    local os_part="${PLATFORM%/*}"
    local arch_part="${PLATFORM#*/}"
    local current_os="$(go env GOOS)"
    local current_arch="$(go env GOARCH)"

    cd "$PROJECT_ROOT"

    local output_dir="$PROJECT_ROOT/build/bin/${os_part}-${arch_part}"
    mkdir -p "$output_dir"

    local output_name="cauldron"
    if [ "$os_part" = "windows" ]; then
        output_name="cauldron.exe"
    fi

    echo "Building for $os_part/$arch_part..."

    build_dev_tools "$PLATFORM"
    build_external_tools "$PLATFORM"

    if [ "$os_part" = "$current_os" ] && [ "$arch_part" = "$current_arch" ]; then
        echo "Native build"
        local extra_tags=""
        if [ "$os_part" = "linux" ] && needs_gtk3_tag; then
            print_info "Legacy GTK4 detected (< 4.10, missing GtkFileDialog) -- building with -tags gtk3"
            extra_tags="-tags gtk3"
        fi
        if go build $extra_tags -ldflags="$LDFLAGS" -o "$output_dir/$output_name" . 2>&1 | tee /tmp/wails-build-${os_part}-${arch_part}.log; then
            if grep -q "undefined:" /tmp/wails-build-${os_part}-${arch_part}.log; then
                print_error "Build failed for $PLATFORM"
                return 1
            fi
            copy_resources "$output_dir" "$PLATFORM"
            print_success "Built $PLATFORM"
            return 0
        else
            print_error "Build failed for $PLATFORM"
            return 1
        fi
    elif [ "$os_part" = "windows" ]; then
        local windows_cc
        if windows_cc=$(get_windows_cc "$arch_part"); then
            echo "Cross-compiling to Windows with CGO (CC=$windows_cc)"
            if CGO_ENABLED=1 CC="$windows_cc" GOOS="$os_part" GOARCH="$arch_part" go build -ldflags="$LDFLAGS" -o "$output_dir/$output_name" . 2>&1 | tee /tmp/wails-build-${os_part}-${arch_part}.log; then
                if grep -q "undefined:" /tmp/wails-build-${os_part}-${arch_part}.log; then
                    print_error "Build failed for $PLATFORM"
                    return 1
                fi
                copy_resources "$output_dir" "$PLATFORM"
                print_success "Built $PLATFORM (CGO enabled)"
                return 0
            else
                print_error "Build failed for $PLATFORM"
                return 1
            fi
        else
            print_info "mingw-w64 not found, building without CGO"
            if CGO_ENABLED=0 GOOS="$os_part" GOARCH="$arch_part" go build -ldflags="$LDFLAGS" -o "$output_dir/$output_name" . 2>&1 | tee /tmp/wails-build-${os_part}-${arch_part}.log; then
                if grep -q "undefined:" /tmp/wails-build-${os_part}-${arch_part}.log; then
                    print_error "Build failed for $PLATFORM"
                    return 1
                fi
                copy_resources "$output_dir" "$PLATFORM"
                print_success "Built $PLATFORM (CGO disabled)"
                return 0
            else
                print_error "Build failed for $PLATFORM"
                return 1
            fi
        fi
    elif [ "$os_part" = "linux" ] && [ "$current_os" = "linux" ]; then
        echo "Native Linux build"
        local extra_tags=""
        if needs_gtk3_tag; then
            print_info "Legacy GTK4 detected (< 4.10, missing GtkFileDialog) -- building with -tags gtk3"
            extra_tags="-tags gtk3"
        fi
        if CGO_ENABLED=1 go build $extra_tags -ldflags="$LDFLAGS" -o "$output_dir/$output_name" . 2>&1 | tee /tmp/wails-build-${os_part}-${arch_part}.log; then
            if grep -q "undefined:" /tmp/wails-build-${os_part}-${arch_part}.log; then
                print_error "Build failed for $PLATFORM"
                return 1
            fi
            copy_resources "$output_dir" "$PLATFORM"
            print_success "Built $PLATFORM"
            return 0
        else
            print_error "Build failed for $PLATFORM"
            return 1
        fi
    elif [ "$os_part" = "darwin" ]; then
        local darwin_cc
        if darwin_cc=$(get_darwin_cc "$arch_part"); then
            echo "Cross-compiling to macOS with CGO (CC=$darwin_cc)"
            if CGO_ENABLED=1 CC="$darwin_cc" GOOS="$os_part" GOARCH="$arch_part" go build -ldflags="$LDFLAGS" -o "$output_dir/$output_name" . 2>&1 | tee /tmp/wails-build-${os_part}-${arch_part}.log; then
                if grep -q "undefined:" /tmp/wails-build-${os_part}-${arch_part}.log; then
                    print_error "Build failed for $PLATFORM"
                    return 1
                fi
                copy_resources "$output_dir" "$PLATFORM"
                print_success "Built $PLATFORM (CGO enabled)"
                return 0
            else
                print_error "Build failed for $PLATFORM"
                return 1
            fi
        else
            print_error "Cross-compilation to macOS requires osxcross toolchain"
            echo "  Set OSX_CC_AMD64/OSX_CC_ARM64 or build natively on macOS"
            return 1
        fi
    else
        print_error "Cross-compilation to $os_part requires CGO and appropriate toolchain"
        echo "  Skipping $PLATFORM (build natively on target platform)"
        return 1
    fi
}

clean_build() {
    print_header "Cleaning Build Artifacts"
    cd "$PROJECT_ROOT"

    rm -rf "$SHARED_LIB_DIR/dist"
    rm -rf "$FRONTEND_DIR/dist"
    rm -rf "$PROJECT_ROOT/build"

    print_success "Build artifacts cleaned"
}

show_help() {
    cat << EOF
Cauldron Build Script (Wails v3)

Usage: ./build.sh [COMMAND] [OPTIONS]

Commands:
  shared-lib            Build only the shared Angular library
  frontend              Build shared library and frontend
  bindings              Generate Wails v3 bindings
  icons                 Generate application icons (PNG, ICO, ICNS, Windows resources)
  wails [PLATFORM]      Build the Wails v3 application (default: windows/amd64)
  wails-task [PLATFORM] Build using Taskfile (requires 'task' CLI)
  all-platforms         Build for all supported platforms (windows, linux, darwin)
  tools                 Build developer tools (plugin-validator, etc.)
  external [PLATFORM]   Build external utility programs (uniprot-fetcher, etc.)
  all [PLATFORM]        Build shared-lib, frontend, tools, and Wails app (default)
  dev                   Start development server
  clean                 Clean all build artifacts
  rebuild [PLATFORM]    Clean and rebuild everything
  help                  Show this help message

Options:
  --skip-licenses       Skip license generation (faster builds)

Platforms:
  windows/amd64    Windows 64-bit (default)
  linux/amd64      Linux 64-bit
  darwin/amd64     macOS 64-bit (Intel)
  darwin/arm64     macOS 64-bit (Apple Silicon)

Environment Variables:
  WAILS_CMD            Wails CLI command (default: wails3)
  MINGW_CC_AMD64       Windows amd64 cross-compiler (default: x86_64-w64-mingw32-gcc)
  MINGW_CC_386         Windows 386 cross-compiler (default: i686-w64-mingw32-gcc)
  OSX_CC_AMD64         macOS amd64 cross-compiler (default: o64-clang)
  OSX_CC_ARM64         macOS arm64 cross-compiler (default: oa64-clang)

Cross-Compilation:
  Windows: Install mingw-w64 (apt install mingw-w64)
  macOS:   Install osxcross (https://github.com/tpoechtrager/osxcross)

Examples:
  ./build.sh                          # Build everything for Windows
  ./build.sh --skip-licenses          # Build without regenerating licenses
  ./build.sh frontend                 # Build only frontend
  ./build.sh bindings                 # Generate Wails bindings
  ./build.sh icons                    # Generate all icon formats
  ./build.sh tools                    # Build developer tools
  ./build.sh external linux/amd64     # Build external utilities for Linux
  ./build.sh wails linux/amd64        # Build Wails app for Linux
  ./build.sh wails-task linux/amd64   # Build using Taskfile
  ./build.sh dev                      # Start development server
  ./build.sh rebuild --skip-licenses  # Clean and rebuild without licenses
  ./build.sh clean                    # Clean build artifacts

EOF
}

case "$COMMAND" in
    shared-lib)
        build_shared_lib
        ;;
    frontend)
        build_shared_lib
        build_frontend
        ;;
    bindings)
        generate_bindings
        ;;
    icons)
        generate_icons
        ;;
    tools)
        build_dev_tools
        ;;
    external)
        build_external_tools "$PLATFORM"
        ;;
    wails)
        build_wails "$PLATFORM"
        ;;
    wails-task)
        build_wails_with_task "$PLATFORM"
        ;;
    all-platforms)
        build_dev_tools
        if [ "$SKIP_LICENSES" = false ]; then
            generate_licenses
        fi
        generate_icons
        build_shared_lib
        build_frontend
        build_external_tools "windows/amd64"
        build_external_tools "linux/amd64"
        build_all_platforms
        print_header "All Platform Builds Complete!"
        ;;
    dev)
        run_dev
        ;;
    all)
        build_dev_tools
        if [ "$SKIP_LICENSES" = false ]; then
            generate_licenses
        else
            echo "Skipping license generation (--skip-licenses flag provided)"
        fi
        build_shared_lib
        build_frontend
        build_wails "$PLATFORM"
        print_header "Build Complete!"
        print_success "Application ready to run"
        ;;
    clean)
        clean_build
        ;;
    rebuild)
        clean_build
        build_dev_tools
        if [ "$SKIP_LICENSES" = false ]; then
            generate_licenses
        else
            echo "Skipping license generation (--skip-licenses flag provided)"
        fi
        build_shared_lib
        build_frontend
        build_wails "$PLATFORM"
        print_header "Rebuild Complete!"
        print_success "Application ready to run"
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "Unknown command: $COMMAND"
        echo ""
        show_help
        exit 1
        ;;
esac
