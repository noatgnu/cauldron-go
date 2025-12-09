#!/usr/bin/env python3
"""
Plugin Documentation Generator for CauldronGO
Automatically generates markdown documentation from plugin YAML files
"""

import yaml
import sys
import os
from pathlib import Path
from typing import Dict, List, Any


def load_plugin(yaml_path: str) -> Dict[str, Any]:
    """Load plugin YAML file."""
    with open(yaml_path, 'r') as f:
        return yaml.safe_load(f)


def format_type(input_def: Dict[str, Any]) -> str:
    """Format input type for documentation."""
    input_type = input_def.get('type', 'unknown')

    if input_type == 'number':
        parts = [input_type]
        if 'min' in input_def:
            parts.append(f"min: {input_def['min']}")
        if 'max' in input_def:
            parts.append(f"max: {input_def['max']}")
        if 'step' in input_def:
            parts.append(f"step: {input_def['step']}")
        return f"{input_type} ({', '.join(parts[1:])})" if len(parts) > 1 else input_type

    if input_type == 'select':
        options = input_def.get('options', [])
        return f"select ({', '.join(options)})"

    if input_type == 'column-selector':
        multiple = input_def.get('multiple', False)
        return f"column-selector ({'multiple' if multiple else 'single'})"

    return input_type


def format_visibility(input_def: Dict[str, Any]) -> str:
    """Format visibility condition."""
    if 'visibleWhen' not in input_def:
        return "Always visible"

    condition = input_def['visibleWhen']
    field = condition.get('field', '?')

    if 'equals' in condition:
        return f"Visible when `{field}` = `{condition['equals']}`"

    if 'equalsAny' in condition:
        values = ', '.join([f"`{v}`" for v in condition['equalsAny']])
        return f"Visible when `{field}` is one of: {values}"

    return "Conditional"


def generate_input_table(inputs: List[Dict[str, Any]]) -> str:
    """Generate markdown table for inputs."""
    if not inputs:
        return "_No inputs defined._\n"

    lines = [
        "| Name | Label | Type | Required | Default | Visibility |",
        "|------|-------|------|----------|---------|------------|"
    ]

    for inp in inputs:
        name = inp.get('name', '')
        label = inp.get('label', '')
        input_type = format_type(inp)
        required = "✓" if inp.get('required', False) else "✗"
        default = inp.get('default', '-')
        if default is True:
            default = "true"
        elif default is False:
            default = "false"
        elif isinstance(default, list):
            default = ', '.join(str(d) for d in default)
        visibility = format_visibility(inp)

        lines.append(f"| `{name}` | {label} | {input_type} | {required} | {default} | {visibility} |")

    return '\n'.join(lines) + '\n'


def generate_input_details(inputs: List[Dict[str, Any]]) -> str:
    """Generate detailed input descriptions."""
    if not inputs:
        return ""

    lines = ["### Input Details\n"]

    for inp in inputs:
        name = inp.get('name', '')
        label = inp.get('label', '')
        description = inp.get('description', '')
        placeholder = inp.get('placeholder', '')

        lines.append(f"#### {label} (`{name}`)\n")

        if description:
            lines.append(f"{description}\n")

        if placeholder:
            lines.append(f"- **Placeholder**: `{placeholder}`")

        if 'options' in inp:
            lines.append(f"- **Options**: {', '.join([f'`{opt}`' for opt in inp['options']])}")

        if 'sourceFile' in inp:
            lines.append(f"- **Column Source**: `{inp['sourceFile']}`")

        lines.append("")

    return '\n'.join(lines)


def generate_output_table(outputs: List[Dict[str, Any]]) -> str:
    """Generate markdown table for outputs."""
    if not outputs:
        return "_No outputs defined._\n"

    lines = [
        "| Name | File | Type | Format | Description |",
        "|------|------|------|--------|-------------|"
    ]

    for out in outputs:
        name = out.get('name', '')
        path = out.get('path', '')
        out_type = out.get('type', '')
        fmt = out.get('format', '')
        desc = out.get('description', '')

        lines.append(f"| `{name}` | `{path}` | {out_type} | {fmt} | {desc} |")

    return '\n'.join(lines) + '\n'


def generate_example_section(example: Dict[str, Any]) -> str:
    """Generate example data section."""
    if not example or not example.get('enabled', False):
        return ""

    lines = [
        "## Example Data\n",
        "This plugin includes example data for testing:\n",
        "```yaml"
    ]

    values = example.get('values', {})
    for key, value in values.items():
        lines.append(f"  {key}: {value}")

    lines.append("```\n")
    lines.append("Load example data by clicking the **Load Example** button in the UI.\n")

    return '\n'.join(lines)


def generate_plugin_doc(plugin: Dict[str, Any], plugin_dir: str) -> str:
    """Generate complete plugin documentation."""
    metadata = plugin.get('plugin', {})
    runtime = plugin.get('runtime', {})
    inputs = plugin.get('inputs', [])
    outputs = plugin.get('outputs', [])
    plots = plugin.get('plots', [])
    execution = plugin.get('execution', {})
    example = plugin.get('example')

    lines = [
        f"# {metadata.get('name', 'Unknown Plugin')}\n",
        f"**ID**: `{metadata.get('id', 'unknown')}`  ",
        f"**Version**: {metadata.get('version', '1.0.0')}  ",
        f"**Category**: {metadata.get('category', 'uncategorized')}  ",
        f"**Author**: {metadata.get('author', 'Unknown')}\n",
        "## Description\n",
        f"{metadata.get('description', 'No description provided.')}\n",
        "## Runtime\n",
        f"- **Type**: `{runtime.get('type', 'unknown')}`",
        f"- **Script**: `{runtime.get('script', 'unknown')}`\n",
        "## Inputs\n",
        generate_input_table(inputs),
        generate_input_details(inputs),
        "## Outputs\n",
        generate_output_table(outputs)
    ]

    if plots:
        lines.extend([
            "## Visualizations\n",
            f"This plugin generates {len(plots)} plot(s):\n"
        ])
        for plot in plots:
            plot_name = plot.get('name', 'Unknown Plot')
            plot_type = plot.get('type', 'unknown')
            plot_component = plot.get('component', 'unknown')
            lines.append(f"- **{plot_name}** ({plot_type}) - Component: `{plot_component}`")
        lines.append("")

    # Requirements
    reqs = execution.get('requirements', {})
    if reqs:
        lines.extend([
            "## Requirements\n"
        ])
        if 'python' in reqs:
            lines.append(f"- **Python**: {reqs['python']}")
        if 'r' in reqs:
            lines.append(f"- **R**: {reqs['r']}")
        if 'packages' in reqs:
            lines.append(f"- **Packages**:")
            for pkg in reqs['packages']:
                lines.append(f"  - {pkg}")
        lines.append("")

    # Example data
    example_section = generate_example_section(example)
    if example_section:
        lines.append(example_section)

    # Usage
    lines.extend([
        "## Usage\n",
        "### Via UI\n",
        f"1. Navigate to **{metadata.get('category', 'Analysis')}** → **{metadata.get('name', 'Plugin')}**",
        "2. Fill in the required inputs",
        "3. Click **Run Analysis**\n",
        "### Via Plugin System\n",
        "```typescript",
        f"const jobId = await pluginService.executePlugin('{metadata.get('id', 'unknown')}', {{",
        "  // Add parameters here",
        "});",
        "```\n"
    ])

    return '\n'.join(lines)


def main():
    if len(sys.argv) < 2:
        print("Usage: generate-plugin-docs.py <plugin_directory>")
        print("Example: generate-plugin-docs.py plugins/pca-analysis")
        sys.exit(1)

    plugin_dir = sys.argv[1]
    yaml_path = os.path.join(plugin_dir, 'plugin.yaml')

    if not os.path.exists(yaml_path):
        yaml_path = os.path.join(plugin_dir, 'plugin.yml')

    if not os.path.exists(yaml_path):
        print(f"Error: No plugin.yaml found in {plugin_dir}")
        sys.exit(1)

    try:
        plugin = load_plugin(yaml_path)
        doc = generate_plugin_doc(plugin, plugin_dir)

        # Output to README.md in plugin directory
        output_path = os.path.join(plugin_dir, 'README.md')
        with open(output_path, 'w') as f:
            f.write(doc)

        print(f"✅ Documentation generated: {output_path}")

        # Also print to stdout
        print("\n" + "="*80 + "\n")
        print(doc)

    except Exception as e:
        print(f"Error generating documentation: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()
