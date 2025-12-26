#!/bin/bash

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGINS_DIR="$PROJECT_ROOT/plugins"

echo "========================================"
echo "CauldronGO Plugin Test Suite"
echo "========================================"
echo ""

if [ ! -f "$PROJECT_ROOT/bin/plugin-validator" ]; then
    echo "Building plugin-validator..."
    go build -o "$PROJECT_ROOT/bin/plugin-validator" "$PROJECT_ROOT/cmd/plugin-validator/main.go"
fi

echo "Running plugin validator on all plugins..."
echo ""

"$PROJECT_ROOT/bin/plugin-validator" "$PLUGINS_DIR"
