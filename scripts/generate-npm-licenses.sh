#!/bin/bash

# Generate NPM package license information

OUTPUT_FILE="resources/licenses/npm-licenses.json"

# Create directory if it doesn't exist
mkdir -p resources/licenses

# Generate licenses
echo "Generating NPM package license information..."
cd frontend
npm list --json --all 2>/dev/null | jq '
  .dependencies // {} |
  to_entries |
  map({
    name: .key,
    version: .value.version // "unknown",
    license: .value.license // "Unknown",
    repository: (
      if .value.repository then
        (if .value.repository.url then .value.repository.url else .value.repository end)
      else
        null
      end
    )
  }) |
  unique_by(.name) |
  sort_by(.name)
' > "../$OUTPUT_FILE"
cd ..

echo "NPM licenses generated at $OUTPUT_FILE"
