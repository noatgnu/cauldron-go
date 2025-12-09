#!/usr/bin/env python3
"""
Validate plugin YAML files against JSON schema
"""

import json
import sys
import yaml
from pathlib import Path
from jsonschema import validate, ValidationError, SchemaError


def load_schema(schema_path: Path) -> dict:
    """Load JSON schema."""
    with open(schema_path, 'r') as f:
        return json.load(f)


def load_plugin_yaml(yaml_path: Path) -> dict:
    """Load plugin YAML file."""
    with open(yaml_path, 'r') as f:
        return yaml.safe_load(f)


def validate_plugin(yaml_path: Path, schema: dict) -> tuple[bool, list]:
    """Validate a plugin YAML file against schema."""
    errors = []

    try:
        plugin_data = load_plugin_yaml(yaml_path)
    except yaml.YAMLError as e:
        errors.append(f"YAML syntax error: {e}")
        return False, errors
    except Exception as e:
        errors.append(f"Error loading YAML: {e}")
        return False, errors

    try:
        validate(instance=plugin_data, schema=schema)
        return True, []
    except ValidationError as e:
        # Format validation error
        path = " → ".join(str(p) for p in e.path) if e.path else "root"
        error_msg = f"Validation error at {path}: {e.message}"
        errors.append(error_msg)
        return False, errors
    except SchemaError as e:
        errors.append(f"Schema error: {e}")
        return False, errors


def main():
    if len(sys.argv) < 2:
        print("Usage: validate-plugin-schema.py <plugin.yaml> [schema.json]")
        sys.exit(1)

    yaml_path = Path(sys.argv[1])
    schema_path = Path(sys.argv[2]) if len(sys.argv) > 2 else Path(__file__).parent.parent / "schemas" / "plugin-schema.json"

    if not yaml_path.exists():
        print(f"❌ Error: Plugin file not found: {yaml_path}")
        sys.exit(1)

    if not schema_path.exists():
        print(f"❌ Error: Schema file not found: {schema_path}")
        sys.exit(1)

    try:
        schema = load_schema(schema_path)
    except Exception as e:
        print(f"❌ Error loading schema: {e}")
        sys.exit(1)

    print(f"🔍 Validating {yaml_path} against schema...")

    is_valid, errors = validate_plugin(yaml_path, schema)

    if is_valid:
        print(f"✅ Plugin is valid!")
        sys.exit(0)
    else:
        print(f"❌ Validation failed:")
        for error in errors:
            print(f"   • {error}")
        sys.exit(1)


if __name__ == '__main__':
    main()
