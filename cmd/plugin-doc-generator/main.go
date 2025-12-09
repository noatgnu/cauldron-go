package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type VisibilityCondition struct {
	Field     string        `yaml:"field"`
	Equals    interface{}   `yaml:"equals,omitempty"`
	EqualsAny []interface{} `yaml:"equalsAny,omitempty"`
}

type PluginInput struct {
	Name        string               `yaml:"name"`
	Label       string               `yaml:"label"`
	Type        string               `yaml:"type"`
	Required    bool                 `yaml:"required"`
	Default     interface{}          `yaml:"default,omitempty"`
	Options     []string             `yaml:"options,omitempty"`
	Description string               `yaml:"description,omitempty"`
	Placeholder string               `yaml:"placeholder,omitempty"`
	Accept      string               `yaml:"accept,omitempty"`
	Multiple    bool                 `yaml:"multiple,omitempty"`
	SourceFile  string               `yaml:"sourceFile,omitempty"`
	Min         *float64             `yaml:"min,omitempty"`
	Max         *float64             `yaml:"max,omitempty"`
	Step        *float64             `yaml:"step,omitempty"`
	VisibleWhen *VisibilityCondition `yaml:"visibleWhen,omitempty"`
}

type PluginOutput struct {
	Name        string `yaml:"name"`
	Path        string `yaml:"path"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Format      string `yaml:"format"`
}

type PluginMetadata struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
	Author      string `yaml:"author,omitempty"`
	Category    string `yaml:"category"`
	Icon        string `yaml:"icon,omitempty"`
}

type PluginRuntime struct {
	Type   string `yaml:"type"`
	Script string `yaml:"script"`
}

type Requirements struct {
	Python   string   `yaml:"python,omitempty"`
	R        string   `yaml:"r,omitempty"`
	Packages []string `yaml:"packages,omitempty"`
}

type PluginExecution struct {
	ArgsMapping  map[string]interface{} `yaml:"argsMapping"`
	OutputDir    string                 `yaml:"outputDir"`
	Requirements Requirements           `yaml:"requirements,omitempty"`
}

type ExampleData struct {
	Enabled bool                   `yaml:"enabled"`
	Values  map[string]interface{} `yaml:"values"`
}

type PluginPlot struct {
	ID         string                 `yaml:"id"`
	Name       string                 `yaml:"name"`
	Type       string                 `yaml:"type"`
	Component  string                 `yaml:"component"`
	DataSource string                 `yaml:"dataSource"`
	Config     map[string]interface{} `yaml:"config,omitempty"`
}

type PluginConfig struct {
	Plugin    PluginMetadata  `yaml:"plugin"`
	Runtime   PluginRuntime   `yaml:"runtime"`
	Inputs    []PluginInput   `yaml:"inputs"`
	Outputs   []PluginOutput  `yaml:"outputs,omitempty"`
	Plots     []PluginPlot    `yaml:"plots,omitempty"`
	Execution PluginExecution `yaml:"execution"`
	Example   *ExampleData    `yaml:"example,omitempty"`
}

func formatType(input PluginInput) string {
	switch input.Type {
	case "number":
		parts := []string{}
		if input.Min != nil {
			parts = append(parts, fmt.Sprintf("min: %.0f", *input.Min))
		}
		if input.Max != nil {
			parts = append(parts, fmt.Sprintf("max: %.0f", *input.Max))
		}
		if input.Step != nil {
			parts = append(parts, fmt.Sprintf("step: %.0f", *input.Step))
		}
		if len(parts) > 0 {
			return fmt.Sprintf("number (%s)", strings.Join(parts, ", "))
		}
		return "number"

	case "select":
		if len(input.Options) > 0 {
			return fmt.Sprintf("select (%s)", strings.Join(input.Options, ", "))
		}
		return "select"

	case "column-selector":
		if input.Multiple {
			return "column-selector (multiple)"
		}
		return "column-selector (single)"

	default:
		return input.Type
	}
}

func formatVisibility(input PluginInput) string {
	if input.VisibleWhen == nil {
		return "Always visible"
	}

	condition := input.VisibleWhen
	if condition.Equals != nil {
		return fmt.Sprintf("Visible when `%s` = `%v`", condition.Field, condition.Equals)
	}

	if len(condition.EqualsAny) > 0 {
		values := []string{}
		for _, v := range condition.EqualsAny {
			values = append(values, fmt.Sprintf("`%v`", v))
		}
		return fmt.Sprintf("Visible when `%s` is one of: %s", condition.Field, strings.Join(values, ", "))
	}

	return "Conditional"
}

func formatDefault(value interface{}) string {
	if value == nil {
		return "-"
	}

	switch v := value.(type) {
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []interface{}:
		parts := []string{}
		for _, item := range v {
			parts = append(parts, fmt.Sprintf("%v", item))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

func generateInputTable(inputs []PluginInput) string {
	if len(inputs) == 0 {
		return "_No inputs defined._\n"
	}

	lines := []string{
		"| Name | Label | Type | Required | Default | Visibility |",
		"|------|-------|------|----------|---------|------------|",
	}

	for _, input := range inputs {
		required := "No"
		if input.Required {
			required = "Yes"
		}

		line := fmt.Sprintf("| `%s` | %s | %s | %s | %s | %s |",
			input.Name,
			input.Label,
			formatType(input),
			required,
			formatDefault(input.Default),
			formatVisibility(input))

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n") + "\n"
}

func generateInputDetails(inputs []PluginInput) string {
	if len(inputs) == 0 {
		return ""
	}

	lines := []string{"### Input Details\n"}

	for _, input := range inputs {
		lines = append(lines, fmt.Sprintf("#### %s (`%s`)\n", input.Label, input.Name))

		if input.Description != "" {
			lines = append(lines, input.Description+"\n")
		}

		if input.Placeholder != "" {
			lines = append(lines, fmt.Sprintf("- **Placeholder**: `%s`", input.Placeholder))
		}

		if len(input.Options) > 0 {
			optList := []string{}
			for _, opt := range input.Options {
				optList = append(optList, fmt.Sprintf("`%s`", opt))
			}
			lines = append(lines, fmt.Sprintf("- **Options**: %s", strings.Join(optList, ", ")))
		}

		if input.SourceFile != "" {
			lines = append(lines, fmt.Sprintf("- **Column Source**: `%s`", input.SourceFile))
		}

		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func generateOutputTable(outputs []PluginOutput) string {
	if len(outputs) == 0 {
		return "_No outputs defined._\n"
	}

	lines := []string{
		"| Name | File | Type | Format | Description |",
		"|------|------|------|--------|-------------|",
	}

	for _, output := range outputs {
		line := fmt.Sprintf("| `%s` | `%s` | %s | %s | %s |",
			output.Name,
			output.Path,
			output.Type,
			output.Format,
			output.Description)

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n") + "\n"
}

func generateExampleSection(example *ExampleData) string {
	if example == nil || !example.Enabled {
		return ""
	}

	lines := []string{
		"## Example Data\n",
		"This plugin includes example data for testing:\n",
		"```yaml",
	}

	for key, value := range example.Values {
		lines = append(lines, fmt.Sprintf("  %s: %v", key, value))
	}

	lines = append(lines, "```\n")
	lines = append(lines, "Load example data by clicking the **Load Example** button in the UI.\n")

	return strings.Join(lines, "\n")
}

func generatePluginDoc(plugin PluginConfig) string {
	lines := []string{
		fmt.Sprintf("# %s\n", plugin.Plugin.Name),
		fmt.Sprintf("**ID**: `%s`  ", plugin.Plugin.ID),
		fmt.Sprintf("**Version**: %s  ", plugin.Plugin.Version),
		fmt.Sprintf("**Category**: %s  ", plugin.Plugin.Category),
		fmt.Sprintf("**Author**: %s\n", plugin.Plugin.Author),
		"## Description\n",
		fmt.Sprintf("%s\n", plugin.Plugin.Description),
		"## Runtime\n",
		fmt.Sprintf("- **Type**: `%s`", plugin.Runtime.Type),
		fmt.Sprintf("- **Script**: `%s`\n", plugin.Runtime.Script),
		"## Inputs\n",
		generateInputTable(plugin.Inputs),
		generateInputDetails(plugin.Inputs),
		"## Outputs\n",
		generateOutputTable(plugin.Outputs),
	}

	if len(plugin.Plots) > 0 {
		lines = append(lines, "## Visualizations\n")
		lines = append(lines, fmt.Sprintf("This plugin generates %d plot(s):\n", len(plugin.Plots)))
		for _, plot := range plugin.Plots {
			lines = append(lines, fmt.Sprintf("- **%s** (%s) - Component: `%s`", plot.Name, plot.Type, plot.Component))
		}
		lines = append(lines, "")
	}

	reqs := plugin.Execution.Requirements
	hasReqs := reqs.Python != "" || reqs.R != "" || len(reqs.Packages) > 0
	if hasReqs {
		lines = append(lines, "## Requirements\n")
		if reqs.Python != "" {
			lines = append(lines, fmt.Sprintf("- **Python**: %s", reqs.Python))
		}
		if reqs.R != "" {
			lines = append(lines, fmt.Sprintf("- **R**: %s", reqs.R))
		}
		if len(reqs.Packages) > 0 {
			lines = append(lines, "- **Packages**:")
			for _, pkg := range reqs.Packages {
				lines = append(lines, fmt.Sprintf("  - %s", pkg))
			}
		}
		lines = append(lines, "")
	}

	exampleSection := generateExampleSection(plugin.Example)
	if exampleSection != "" {
		lines = append(lines, exampleSection)
	}

	lines = append(lines,
		"## Usage\n",
		"### Via UI\n",
		fmt.Sprintf("1. Navigate to **%s** → **%s**", plugin.Plugin.Category, plugin.Plugin.Name),
		"2. Fill in the required inputs",
		"3. Click **Run Analysis**\n",
		"### Via Plugin System\n",
		"```typescript",
		fmt.Sprintf("const jobId = await pluginService.executePlugin('%s', {", plugin.Plugin.ID),
		"  // Add parameters here",
		"});",
		"```\n",
	)

	return strings.Join(lines, "\n")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: plugin-doc-generator <plugin_directory>")
		fmt.Println("Example: plugin-doc-generator plugins/pca-analysis")
		os.Exit(1)
	}

	pluginDir := os.Args[1]
	yamlPath := filepath.Join(pluginDir, "plugin.yaml")

	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		yamlPath = filepath.Join(pluginDir, "plugin.yml")
	}

	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		fmt.Printf("[ERROR] No plugin.yaml found in %s\n", pluginDir)
		os.Exit(1)
	}

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to read file: %v\n", err)
		os.Exit(1)
	}

	var plugin PluginConfig
	if err := yaml.Unmarshal(data, &plugin); err != nil {
		fmt.Printf("[ERROR] Invalid YAML: %v\n", err)
		os.Exit(1)
	}

	doc := generatePluginDoc(plugin)

	outputPath := filepath.Join(pluginDir, "README.md")
	if err := os.WriteFile(outputPath, []byte(doc), 0644); err != nil {
		fmt.Printf("[ERROR] Failed to write documentation: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[SUCCESS] Documentation generated: %s\n", outputPath)
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
	fmt.Println(doc)
}
