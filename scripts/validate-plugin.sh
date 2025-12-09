#!/bin/bash

# Plugin Validation Script for CauldronGO
# Validates plugin YAML structure and checks for common issues

set -e

PLUGIN_DIR="${1:-plugins}"
ERRORS=0
WARNINGS=0

print_error() {
    echo "❌ ERROR: $1" >&2
    ((ERRORS++))
}

print_warning() {
    echo "⚠️  WARNING: $1" >&2
    ((WARNINGS++))
}

print_success() {
    echo "✅ $1"
}

print_info() {
    echo "ℹ️  $1"
}

validate_yaml_syntax() {
    local file="$1"
    if ! python3 -c "import yaml; yaml.safe_load(open('$file'))" 2>/dev/null; then
        print_error "Invalid YAML syntax in $file"
        return 1
    fi
    return 0
}

check_required_fields() {
    local file="$1"
    local plugin_id=$(yq e '.plugin.id' "$file" 2>/dev/null)
    local plugin_name=$(yq e '.plugin.name' "$file" 2>/dev/null)
    local runtime_type=$(yq e '.runtime.type' "$file" 2>/dev/null)
    local runtime_script=$(yq e '.runtime.script' "$file" 2>/dev/null)

    if [ "$plugin_id" == "null" ] || [ -z "$plugin_id" ]; then
        print_error "$file: Missing plugin.id"
    fi

    if [ "$plugin_name" == "null" ] || [ -z "$plugin_name" ]; then
        print_error "$file: Missing plugin.name"
    fi

    if [ "$runtime_type" == "null" ] || [ -z "$runtime_type" ]; then
        print_error "$file: Missing runtime.type"
    fi

    if [ "$runtime_script" == "null" ] || [ -z "$runtime_script" ]; then
        print_error "$file: Missing runtime.script"
    fi
}

check_script_exists() {
    local yaml_file="$1"
    local plugin_dir=$(dirname "$yaml_file")
    local script=$(yq e '.runtime.script' "$yaml_file" 2>/dev/null)

    if [ "$script" != "null" ] && [ -n "$script" ]; then
        if [ ! -f "$plugin_dir/$script" ]; then
            print_error "$yaml_file: Script '$script' not found in $plugin_dir"
        else
            print_success "$yaml_file: Script '$script' exists"
        fi
    fi
}

check_conditional_visibility() {
    local file="$1"
    local inputs=$(yq e '.inputs[] | select(.visibleWhen) | .name' "$file" 2>/dev/null)

    if [ -n "$inputs" ]; then
        while IFS= read -r input_name; do
            local field=$(yq e ".inputs[] | select(.name == \"$input_name\") | .visibleWhen.field" "$file" 2>/dev/null)
            local field_exists=$(yq e ".inputs[] | select(.name == \"$field\") | .name" "$file" 2>/dev/null)

            if [ "$field_exists" == "null" ] || [ -z "$field_exists" ]; then
                print_warning "$file: Input '$input_name' has visibleWhen referencing non-existent field '$field'"
            fi
        done <<< "$inputs"
    fi
}

check_example_files() {
    local file="$1"
    local example_enabled=$(yq e '.example.enabled' "$file" 2>/dev/null)

    if [ "$example_enabled" == "true" ]; then
        print_info "$file: Has example data configured"

        # Check if example values reference valid example directories
        local example_files=$(yq e '.example.values | to_entries | .[] | select(.value | test("/")) | .value' "$file" 2>/dev/null)

        if [ -n "$example_files" ]; then
            while IFS= read -r example_path; do
                local category=$(echo "$example_path" | cut -d'/' -f1)
                local filename=$(echo "$example_path" | cut -d'/' -f2)
                local full_path="examples/$category/$filename"

                if [ ! -f "$full_path" ]; then
                    print_warning "$file: Example file '$full_path' not found"
                fi
            done <<< "$example_files"
        fi
    fi
}

validate_input_types() {
    local file="$1"
    local valid_types=("file" "text" "number" "boolean" "select" "column-selector")
    local inputs=$(yq e '.inputs[].type' "$file" 2>/dev/null)

    if [ -n "$inputs" ]; then
        while IFS= read -r input_type; do
            local is_valid=0
            for valid_type in "${valid_types[@]}"; do
                if [ "$input_type" == "$valid_type" ]; then
                    is_valid=1
                    break
                fi
            done

            if [ $is_valid -eq 0 ]; then
                print_error "$file: Invalid input type '$input_type'"
            fi
        done <<< "$inputs"
    fi
}

check_output_dir_mapping() {
    local file="$1"
    local output_dir=$(yq e '.execution.outputDir' "$file" 2>/dev/null)

    if [ "$output_dir" == "null" ] || [ -z "$output_dir" ]; then
        print_warning "$file: Missing execution.outputDir mapping"
    fi
}

validate_plugin() {
    local yaml_file="$1"

    echo ""
    echo "🔍 Validating: $yaml_file"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    if ! validate_yaml_syntax "$yaml_file"; then
        return 1
    fi

    check_required_fields "$yaml_file"
    check_script_exists "$yaml_file"
    check_conditional_visibility "$yaml_file"
    check_example_files "$yaml_file"
    validate_input_types "$yaml_file"
    check_output_dir_mapping "$yaml_file"

    echo ""
}

# Main execution
echo "🔧 CauldronGO Plugin Validator"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check for required tools
if ! command -v yq &> /dev/null; then
    echo "❌ ERROR: yq is not installed. Install with: brew install yq (macOS) or snap install yq (Linux)"
    exit 1
fi

if ! command -v python3 &> /dev/null; then
    echo "❌ ERROR: python3 is not installed"
    exit 1
fi

# Find all plugin.yaml files
if [ -d "$PLUGIN_DIR" ]; then
    PLUGIN_FILES=$(find "$PLUGIN_DIR" -name "plugin.yaml" -o -name "plugin.yml")

    if [ -z "$PLUGIN_FILES" ]; then
        echo "❌ No plugin.yaml files found in $PLUGIN_DIR"
        exit 1
    fi

    PLUGIN_COUNT=$(echo "$PLUGIN_FILES" | wc -l | tr -d ' ')
    echo "Found $PLUGIN_COUNT plugin(s) to validate"

    while IFS= read -r plugin_file; do
        validate_plugin "$plugin_file"
    done <<< "$PLUGIN_FILES"
else
    echo "❌ ERROR: Plugin directory '$PLUGIN_DIR' not found"
    exit 1
fi

# Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Validation Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Plugins validated: $PLUGIN_COUNT"
echo "Errors: $ERRORS"
echo "Warnings: $WARNINGS"

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "❌ Validation FAILED with $ERRORS error(s)"
    exit 1
elif [ $WARNINGS -gt 0 ]; then
    echo ""
    echo "⚠️  Validation passed with $WARNINGS warning(s)"
    exit 0
else
    echo ""
    echo "✅ All plugins validated successfully!"
    exit 0
fi
