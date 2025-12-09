#!/bin/bash

# Plugin Scaffolding Tool for CauldronGO
# Creates a new plugin with boilerplate files

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGINS_DIR="${SCRIPT_DIR}/../plugins"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

prompt() {
    local prompt_text="$1"
    local default_value="$2"
    local result

    if [ -n "$default_value" ]; then
        read -p "$prompt_text [$default_value]: " result
        echo "${result:-$default_value}"
    else
        read -p "$prompt_text: " result
        echo "$result"
    fi
}

select_option() {
    local prompt_text="$1"
    shift
    local options=("$@")

    echo "$prompt_text"
    select opt in "${options[@]}"; do
        if [ -n "$opt" ]; then
            echo "$opt"
            return
        fi
    done
}

# Main script
print_header "🔧 CauldronGO Plugin Scaffolding Tool"

# Gather plugin information
echo ""
print_info "Let's create a new plugin!"
echo ""

PLUGIN_ID=$(prompt "Plugin ID (lowercase, hyphenated)")
if [ -z "$PLUGIN_ID" ]; then
    print_error "Plugin ID is required"
    exit 1
fi

# Check if plugin already exists
if [ -d "$PLUGINS_DIR/$PLUGIN_ID" ]; then
    print_error "Plugin '$PLUGIN_ID' already exists"
    exit 1
fi

PLUGIN_NAME=$(prompt "Plugin Name (human-readable)" "$(echo $PLUGIN_ID | sed 's/-/ /g' | sed 's/\b\(.\)/\u\1/g')")
PLUGIN_DESC=$(prompt "Description" "Analysis tool for CauldronGO")
PLUGIN_VERSION=$(prompt "Version" "1.0.0")
PLUGIN_AUTHOR=$(prompt "Author" "CauldronGO Team")

echo ""
print_info "Select plugin category:"
PLUGIN_CATEGORY=$(select_option "Category:" "analysis" "preprocessing" "visualization" "utilities")

echo ""
print_info "Select runtime:"
PLUGIN_RUNTIME=$(select_option "Runtime:" "python" "r" "pythonWithR")

SCRIPT_EXT=""
case $PLUGIN_RUNTIME in
    python|pythonWithR)
        SCRIPT_EXT="py"
        ;;
    r)
        SCRIPT_EXT="R"
        ;;
esac

SCRIPT_NAME="${PLUGIN_ID//-/_}.${SCRIPT_EXT}"

echo ""
print_info "Summary:"
echo "  ID:       $PLUGIN_ID"
echo "  Name:     $PLUGIN_NAME"
echo "  Category: $PLUGIN_CATEGORY"
echo "  Runtime:  $PLUGIN_RUNTIME"
echo "  Script:   $SCRIPT_NAME"
echo ""

read -p "Create plugin? (y/n): " confirm
if [[ ! $confirm =~ ^[Yy]$ ]]; then
    print_warning "Cancelled"
    exit 0
fi

# Create plugin directory
print_info "Creating plugin directory..."
PLUGIN_DIR="$PLUGINS_DIR/$PLUGIN_ID"
mkdir -p "$PLUGIN_DIR"

# Create plugin.yaml
print_info "Creating plugin.yaml..."
cat > "$PLUGIN_DIR/plugin.yaml" <<EOF
plugin:
  id: "$PLUGIN_ID"
  name: "$PLUGIN_NAME"
  description: "$PLUGIN_DESC"
  version: "$PLUGIN_VERSION"
  author: "$PLUGIN_AUTHOR"
  category: "$PLUGIN_CATEGORY"
  icon: "analytics"  # Material icon name

runtime:
  type: "$PLUGIN_RUNTIME"
  script: "$SCRIPT_NAME"

inputs:
  - name: "input_file"
    label: "Input File"
    type: "file"
    required: true
    accept: ".csv,.tsv,.txt"
    description: "Input data file"

  # Add more inputs as needed
  # - name: "parameter1"
  #   label: "Parameter 1"
  #   type: "number"
  #   default: 10
  #   min: 1
  #   max: 100
  #   description: "Example numeric parameter"

outputs:
  - name: "output_data"
    path: "output.txt"
    type: "data"
    description: "Analysis results"
    format: "tsv"

execution:
  argsMapping:
    input_file: "--input_file"
    # Add more argument mappings here

  outputDir: "--output_folder"

  requirements:
EOF

if [ "$PLUGIN_RUNTIME" = "python" ] || [ "$PLUGIN_RUNTIME" = "pythonWithR" ]; then
cat >> "$PLUGIN_DIR/plugin.yaml" <<EOF
    python: ">=3.11"
    packages:
      - "pandas>=2.0.0"
      - "numpy>=1.24.0"
EOF
fi

if [ "$PLUGIN_RUNTIME" = "r" ] || [ "$PLUGIN_RUNTIME" = "pythonWithR" ]; then
cat >> "$PLUGIN_DIR/plugin.yaml" <<EOF
    r: ">=4.0"
    packages:
      - "tidyverse"
EOF
fi

# Uncomment to add example data
cat >> "$PLUGIN_DIR/plugin.yaml" <<EOF

# example:
#   enabled: true
#   values:
#     input_file: "diann/imputed.data.txt"
#     # Add more example values here
EOF

print_success "Created plugin.yaml"

# Create Python script template
if [ "$SCRIPT_EXT" = "py" ]; then
    print_info "Creating Python script..."
    cat > "$PLUGIN_DIR/$SCRIPT_NAME" <<'EOF'
#!/usr/bin/env python3
"""
PLUGIN_NAME
PLUGIN_DESC
"""

import argparse
import sys
import pandas as pd
from pathlib import Path


def parse_args():
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(description='PLUGIN_DESC')

    parser.add_argument('--input_file', required=True, help='Input data file')
    parser.add_argument('--output_folder', required=True, help='Output directory')

    # Add more arguments as needed
    # parser.add_argument('--parameter1', type=int, default=10, help='Example parameter')

    return parser.parse_args()


def main():
    """Main execution function."""
    args = parse_args()

    # Create output directory
    output_dir = Path(args.output_folder)
    output_dir.mkdir(parents=True, exist_ok=True)

    print(f"Loading data from {args.input_file}...")
    # TODO: Implement your analysis logic here
    # df = pd.read_csv(args.input_file, sep='\t')

    print("Processing...")
    # TODO: Add processing logic

    # Save results
    output_file = output_dir / 'output.txt'
    print(f"Saving results to {output_file}...")
    # TODO: Save results
    # df.to_csv(output_file, sep='\t', index=False)

    print("Analysis complete!")
    return 0


if __name__ == '__main__':
    sys.exit(main())
EOF

    # Replace placeholders
    sed -i "s/PLUGIN_NAME/$PLUGIN_NAME/g" "$PLUGIN_DIR/$SCRIPT_NAME"
    sed -i "s/PLUGIN_DESC/$PLUGIN_DESC/g" "$PLUGIN_DIR/$SCRIPT_NAME"

    chmod +x "$PLUGIN_DIR/$SCRIPT_NAME"
    print_success "Created $SCRIPT_NAME"
fi

# Create R script template
if [ "$SCRIPT_EXT" = "R" ]; then
    print_info "Creating R script..."
    cat > "$PLUGIN_DIR/$SCRIPT_NAME" <<'EOF'
#!/usr/bin/env Rscript

# PLUGIN_NAME
# PLUGIN_DESC

library(optparse)

# Parse command line arguments
option_list <- list(
  make_option(c("--input_file"), type="character", help="Input data file"),
  make_option(c("--output_folder"), type="character", help="Output directory")
  # Add more options as needed
)

parser <- OptionParser(option_list=option_list)
args <- parse_args(parser)

# Create output directory
dir.create(args$output_folder, showWarnings = FALSE, recursive = TRUE)

cat("Loading data from", args$input_file, "...\n")
# TODO: Implement your analysis logic here
# data <- read.delim(args$input_file)

cat("Processing...\n")
# TODO: Add processing logic

# Save results
output_file <- file.path(args$output_folder, "output.txt")
cat("Saving results to", output_file, "...\n")
# TODO: Save results
# write.table(data, output_file, sep="\t", quote=FALSE, row.names=FALSE)

cat("Analysis complete!\n")
EOF

    # Replace placeholders
    sed -i "s/PLUGIN_NAME/$PLUGIN_NAME/g" "$PLUGIN_DIR/$SCRIPT_NAME"
    sed -i "s/PLUGIN_DESC/$PLUGIN_DESC/g" "$PLUGIN_DIR/$SCRIPT_NAME"

    chmod +x "$PLUGIN_DIR/$SCRIPT_NAME"
    print_success "Created $SCRIPT_NAME"
fi

# Create .gitignore
cat > "$PLUGIN_DIR/.gitignore" <<EOF
__pycache__/
*.pyc
.Rhistory
.RData
*.swp
*.swo
*~
EOF
print_success "Created .gitignore"

# Generate documentation
print_info "Generating documentation..."
if python3 "$SCRIPT_DIR/generate-plugin-docs.py" "$PLUGIN_DIR" > /dev/null 2>&1; then
    print_success "Generated README.md"
else
    print_warning "Could not generate README.md (install PyYAML: pip install pyyaml)"
fi

# Validate plugin
print_info "Validating plugin..."
if [ -f "$SCRIPT_DIR/validate-plugin.sh" ]; then
    if "$SCRIPT_DIR/validate-plugin.sh" "$PLUGIN_DIR" 2>&1 | grep -q "validation succeeded"; then
        print_success "Plugin validation passed"
    else
        print_warning "Plugin validation has warnings (check with: ./scripts/validate-plugin.sh $PLUGIN_DIR)"
    fi
else
    print_warning "Validation script not found"
fi

echo ""
print_header "🎉 Plugin Created Successfully!"
echo ""
print_info "Next steps:"
echo "  1. Edit $PLUGIN_DIR/plugin.yaml to add inputs/outputs"
echo "  2. Implement analysis logic in $PLUGIN_DIR/$SCRIPT_NAME"
echo "  3. Add example data (optional)"
echo "  4. Validate: ./scripts/validate-plugin.sh $PLUGIN_DIR"
echo "  5. Generate docs: python3 ./scripts/generate-plugin-docs.py $PLUGIN_DIR"
echo "  6. Test in UI: Navigate to Plugin View → $PLUGIN_CATEGORY → $PLUGIN_NAME"
echo ""
print_success "Happy coding! 🚀"
