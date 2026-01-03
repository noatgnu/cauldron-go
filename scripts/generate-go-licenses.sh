#!/bin/bash

# Generate Go module license information

OUTPUT_FILE="resources/licenses/go-licenses.json"

# Create directory if it doesn't exist
mkdir -p resources/licenses

# Generate licenses
echo "Generating Go module license information..."
go list -m -json all | jq -s '
  map(
    select(.Replace == null) |
    {
      name: .Path,
      version: .Version // "unknown",
      license: "See module repository",
      repository: (if .Path | startswith("github.com") then "https://\(.Path)" else null end)
    }
  ) | unique_by(.name)
' > "$OUTPUT_FILE"

echo "Go licenses generated at $OUTPUT_FILE"
