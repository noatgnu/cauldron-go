#!/bin/bash

# Generate documentation for all plugins

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGINS_DIR="${SCRIPT_DIR}/../plugins"

echo "🔧 Generating documentation for all plugins..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

TOTAL=0
SUCCESS=0
FAILED=0

for plugin_dir in "$PLUGINS_DIR"/*; do
    if [ -d "$plugin_dir" ] && [ -f "$plugin_dir/plugin.yaml" ]; then
        plugin_name=$(basename "$plugin_dir")

        echo ""
        echo "📝 Generating docs for: $plugin_name"

        if python3 "$SCRIPT_DIR/generate-plugin-docs.py" "$plugin_dir" > /dev/null 2>&1; then
            echo "   ✅ Success"
            ((SUCCESS++))
        else
            echo "   ❌ Failed"
            ((FAILED++))
        fi

        ((TOTAL++))
    fi
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Total plugins: $TOTAL"
echo "Success: $SUCCESS"
echo "Failed: $FAILED"

if [ $FAILED -eq 0 ]; then
    echo ""
    echo "✅ All plugin documentation generated successfully!"
    exit 0
else
    echo ""
    echo "⚠️  Some plugins failed to generate documentation"
    exit 1
fi
