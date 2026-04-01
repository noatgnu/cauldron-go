#!/bin/bash

set -e

PROJECT_ROOT=$(cd "$(dirname "$0")/.." && pwd)
RESOURCES_DIR="$PROJECT_ROOT/resources"
BUILD_DIR="$PROJECT_ROOT/build"
SOURCE_ICON="$RESOURCES_DIR/appicon.png"

print_success() {
    echo -e "\033[0;32m✓ $1\033[0m"
}

print_error() {
    echo -e "\033[0;31m✗ $1\033[0m"
}

print_info() {
    echo -e "\033[0;34mℹ $1\033[0m"
}

GO_WINRES=""

find_go_winres() {
    if command -v go-winres &> /dev/null; then
        GO_WINRES="go-winres"
        return 0
    elif [ -x "$HOME/go/bin/go-winres" ]; then
        GO_WINRES="$HOME/go/bin/go-winres"
        return 0
    elif [ -x "$(go env GOPATH)/bin/go-winres" ]; then
        GO_WINRES="$(go env GOPATH)/bin/go-winres"
        return 0
    fi
    return 1
}

check_dependencies() {
    local missing=()

    if ! command -v convert &> /dev/null; then
        missing+=("ImageMagick (convert)")
    fi

    if ! find_go_winres; then
        print_info "go-winres not found, will try to install..."
        if go install github.com/tc-hib/go-winres@latest 2>/dev/null; then
            print_success "go-winres installed"
            find_go_winres
        else
            missing+=("go-winres (go install github.com/tc-hib/go-winres@latest)")
        fi
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        print_error "Missing dependencies:"
        for dep in "${missing[@]}"; do
            echo "  - $dep"
        done
        echo ""
        echo "Install ImageMagick:"
        echo "  Ubuntu/Debian: sudo apt install imagemagick"
        echo "  macOS: brew install imagemagick"
        echo "  Windows: choco install imagemagick"
        return 1
    fi

    return 0
}

generate_png_sizes() {
    print_info "Generating PNG icons in multiple sizes..."

    local sizes=(16 32 48 64 128 256 512 1024)

    mkdir -p "$RESOURCES_DIR/icons"

    for size in "${sizes[@]}"; do
        convert "$SOURCE_ICON" -resize "${size}x${size}" "$RESOURCES_DIR/icons/icon-${size}.png"
        print_success "Generated icon-${size}.png"
    done
}

generate_ico() {
    print_info "Generating Windows .ico file..."

    local ico_sizes=(16 32 48 64 128 256)
    local ico_files=()

    for size in "${ico_sizes[@]}"; do
        ico_files+=("$RESOURCES_DIR/icons/icon-${size}.png")
    done

    convert "${ico_files[@]}" "$RESOURCES_DIR/icon.ico"
    print_success "Generated icon.ico"
}

generate_icns() {
    print_info "Generating macOS .icns file..."

    local iconset_dir="$RESOURCES_DIR/AppIcon.iconset"
    mkdir -p "$iconset_dir"

    convert "$SOURCE_ICON" -resize 16x16 "$iconset_dir/icon_16x16.png"
    convert "$SOURCE_ICON" -resize 32x32 "$iconset_dir/icon_16x16@2x.png"
    convert "$SOURCE_ICON" -resize 32x32 "$iconset_dir/icon_32x32.png"
    convert "$SOURCE_ICON" -resize 64x64 "$iconset_dir/icon_32x32@2x.png"
    convert "$SOURCE_ICON" -resize 128x128 "$iconset_dir/icon_128x128.png"
    convert "$SOURCE_ICON" -resize 256x256 "$iconset_dir/icon_128x128@2x.png"
    convert "$SOURCE_ICON" -resize 256x256 "$iconset_dir/icon_256x256.png"
    convert "$SOURCE_ICON" -resize 512x512 "$iconset_dir/icon_256x256@2x.png"
    convert "$SOURCE_ICON" -resize 512x512 "$iconset_dir/icon_512x512.png"
    convert "$SOURCE_ICON" -resize 1024x1024 "$iconset_dir/icon_512x512@2x.png"

    if command -v iconutil &> /dev/null; then
        iconutil -c icns "$iconset_dir" -o "$RESOURCES_DIR/appicon.icns"
        print_success "Generated appicon.icns"
    else
        print_info "iconutil not available (macOS only), skipping .icns generation"
        print_info "Iconset created at $iconset_dir - convert on macOS with: iconutil -c icns AppIcon.iconset"
    fi
}

generate_windows_resource() {
    print_info "Generating Windows resource file (.syso)..."

    mkdir -p "$BUILD_DIR"

    cat > "$PROJECT_ROOT/winres.json" << 'EOF'
{
    "RT_GROUP_ICON": {
        "APP": {
            "0000": "resources/icon.ico"
        }
    },
    "RT_MANIFEST": {
        "#1": {
            "0409": {
                "identity": {
                    "name": "Cauldron",
                    "version": "1.0.0.0"
                },
                "description": "Proteomics data visualization and analysis",
                "minimum-os": "win7",
                "execution-level": "as invoker",
                "ui-access": false,
                "auto-elevate": false,
                "dpi-awareness": "system",
                "disable-theming": false,
                "disable-window-filtering": false,
                "high-resolution-scrolling-aware": true,
                "ultra-high-resolution-scrolling-aware": true,
                "long-path-aware": true,
                "printer-driver-isolation": false,
                "gdi-scaling": false,
                "segment-heap": false,
                "use-common-controls-v6": true
            }
        }
    },
    "RT_VERSION": {
        "#1": {
            "0000": {
                "fixed": {
                    "file_version": "1.0.0.0",
                    "product_version": "1.0.0.0"
                },
                "info": {
                    "0409": {
                        "Comments": "Proteomics data visualization and analysis",
                        "CompanyName": "",
                        "FileDescription": "Cauldron Application",
                        "FileVersion": "1.0.0.0",
                        "InternalName": "cauldron",
                        "LegalCopyright": "",
                        "OriginalFilename": "cauldron.exe",
                        "ProductName": "Cauldron",
                        "ProductVersion": "1.0.0.0"
                    }
                }
            }
        }
    }
}
EOF

    if [ -n "$GO_WINRES" ] || find_go_winres; then
        cd "$PROJECT_ROOT"
        "$GO_WINRES" make --in winres.json --out rsrc --arch amd64,386,arm64
        print_success "Generated Windows resource files (.syso)"
    else
        print_error "go-winres not available, skipping .syso generation"
        return 1
    fi
}

cleanup_temp() {
    rm -f "$PROJECT_ROOT/winres.json"
}

show_help() {
    cat << EOF
Icon Generation Script for Cauldron

Usage: ./scripts/generate-icons.sh [COMMAND]

Commands:
  all         Generate all icon formats (default)
  png         Generate PNG icons in multiple sizes
  ico         Generate Windows .ico file
  icns        Generate macOS .icns file (requires macOS)
  winres      Generate Windows resource file (.syso)
  help        Show this help message

Requirements:
  - ImageMagick (convert command)
  - go-winres (for Windows resource files)
  - iconutil (for .icns, macOS only)

EOF
}

case "${1:-all}" in
    all)
        if check_dependencies; then
            generate_png_sizes
            generate_ico
            generate_icns
            generate_windows_resource
            cleanup_temp
            echo ""
            print_success "All icons generated successfully!"
        fi
        ;;
    png)
        if command -v convert &> /dev/null; then
            generate_png_sizes
        else
            print_error "ImageMagick not found"
            exit 1
        fi
        ;;
    ico)
        if command -v convert &> /dev/null; then
            generate_png_sizes
            generate_ico
        else
            print_error "ImageMagick not found"
            exit 1
        fi
        ;;
    icns)
        if command -v convert &> /dev/null; then
            generate_icns
        else
            print_error "ImageMagick not found"
            exit 1
        fi
        ;;
    winres)
        find_go_winres || {
            print_info "go-winres not found, will try to install..."
            go install github.com/tc-hib/go-winres@latest 2>/dev/null
            find_go_winres
        }
        generate_windows_resource
        cleanup_temp
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
