package main

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3"
)

type VisibilityCondition struct {
	Field     string        `yaml:"field"`
	Equals    interface{}   `yaml:"equals,omitempty"`
	EqualsAny []interface{} `yaml:"equalsAny,omitempty"`
}

type FieldOption struct {
	Value string `yaml:"value"`
	Label string `yaml:"label"`
}

type FieldGroup struct {
	Name    string        `yaml:"name"`
	Options []FieldOption `yaml:"options"`
}

type TableColumn struct {
	Name        string `yaml:"name"`
	Label       string `yaml:"label"`
	Type        string `yaml:"type,omitempty"`
	Required    bool   `yaml:"required,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type PluginInput struct {
	Name                        string               `yaml:"name"`
	Label                       string               `yaml:"label"`
	Type                        string               `yaml:"type"`
	Required                    bool                 `yaml:"required"`
	Default                     interface{}          `yaml:"default,omitempty"`
	Options                     []string             `yaml:"options,omitempty"`
	OptionsFromFile             string               `yaml:"optionsFromFile,omitempty"`
	Groups                      []FieldGroup         `yaml:"groups,omitempty"`
	GroupsFromFile              string               `yaml:"groupsFromFile,omitempty"`
	Description                 string               `yaml:"description,omitempty"`
	Placeholder                 string               `yaml:"placeholder,omitempty"`
	Accept                      string               `yaml:"accept,omitempty"`
	Multiple                    bool                 `yaml:"multiple,omitempty"`
	SourceFile                  string               `yaml:"sourceFile,omitempty"`
	Min                         *float64             `yaml:"min,omitempty"`
	Max                         *float64             `yaml:"max,omitempty"`
	Step                        *float64             `yaml:"step,omitempty"`
	VisibleWhen                 *VisibilityCondition `yaml:"visibleWhen,omitempty"`
	DisableAnnotationManagement bool                 `yaml:"disableAnnotationManagement,omitempty"`
	TableColumns                []TableColumn        `yaml:"tableColumns,omitempty"`
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
	Repository  string `yaml:"repository,omitempty"`
}

type PluginRuntime struct {
	Type   string `yaml:"type"`
	Script string `yaml:"script"`
}

type Requirements struct {
	Python                 string   `yaml:"python,omitempty"`
	R                      string   `yaml:"r,omitempty"`
	Packages               []string `yaml:"packages,omitempty"`
	PythonRequirementsFile string   `yaml:"pythonRequirementsFile,omitempty"`
	RPackagesFile          string   `yaml:"rPackagesFile,omitempty"`
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

type PlotAxes struct {
	X       string  `yaml:"x"`
	Y       string  `yaml:"y"`
	Z       string  `yaml:"z,omitempty"`
	ColorBy *string `yaml:"colorBy,omitempty"`
	SizeBy  *string `yaml:"sizeBy,omitempty"`
	Labels  *string `yaml:"labels,omitempty"`
}

type PlotConfigData struct {
	Axes             PlotAxes `yaml:"axes"`
	ImagePattern     string   `yaml:"imagePattern,omitempty"`
	ImagePatternType string   `yaml:"imagePatternType,omitempty"`
}

type PlotCustomization struct {
	Name    string      `yaml:"name"`
	Label   string      `yaml:"label"`
	Type    string      `yaml:"type"`
	Default interface{} `yaml:"default,omitempty"`
	Min     *float64    `yaml:"min,omitempty"`
	Max     *float64    `yaml:"max,omitempty"`
}

type PluginPlot struct {
	ID            string              `yaml:"id"`
	Name          string              `yaml:"name"`
	Type          string              `yaml:"type"`
	Component     string              `yaml:"component,omitempty"`
	DataSource    string              `yaml:"dataSource"`
	Config        PlotConfigData      `yaml:"config"`
	Customization []PlotCustomization `yaml:"customization,omitempty"`
	Title         string              `yaml:"title,omitempty"`
	Description   string              `yaml:"description,omitempty"`
	Default       bool                `yaml:"default,omitempty"`
}

type AnnotationConfig struct {
	SamplesFrom    string `yaml:"samplesFrom,omitempty"`
	AnnotationFile string `yaml:"annotationFile,omitempty"`
}

type PluginConfig struct {
	Plugin     PluginMetadata    `yaml:"plugin"`
	Runtime    PluginRuntime     `yaml:"runtime"`
	Inputs     []PluginInput     `yaml:"inputs"`
	Outputs    []PluginOutput    `yaml:"outputs,omitempty"`
	Plots      []PluginPlot      `yaml:"plots,omitempty"`
	Annotation *AnnotationConfig `yaml:"annotation,omitempty"`
	Execution  PluginExecution   `yaml:"execution"`
	Example    *ExampleData      `yaml:"example,omitempty"`
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
		if input.OptionsFromFile != "" {
			return fmt.Sprintf("select (from %s)", input.OptionsFromFile)
		}
		return "select"

	case "multiselect-grouped":
		if len(input.Groups) > 0 {
			return fmt.Sprintf("multiselect-grouped (%d groups)", len(input.Groups))
		}
		if input.GroupsFromFile != "" {
			return fmt.Sprintf("multiselect-grouped (from %s)", input.GroupsFromFile)
		}
		return "multiselect-grouped"

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

		if input.OptionsFromFile != "" {
			lines = append(lines, fmt.Sprintf("- **Options From File**: `%s`", input.OptionsFromFile))
		}

		if len(input.Groups) > 0 {
			lines = append(lines, fmt.Sprintf("- **Groups**: %d groups defined", len(input.Groups)))
		}

		if input.GroupsFromFile != "" {
			lines = append(lines, fmt.Sprintf("- **Groups From File**: `%s`", input.GroupsFromFile))
		}

		if len(input.TableColumns) > 0 {
			lines = append(lines, fmt.Sprintf("- **Table Editor**: Enabled with %d columns", len(input.TableColumns)))
			lines = append(lines, "  - **Columns**:")
			for _, col := range input.TableColumns {
				reqStr := ""
				if col.Required {
					reqStr = " (required)"
				}
				lines = append(lines, fmt.Sprintf("    - `%s`: %s%s", col.Name, col.Label, reqStr))
				if col.Description != "" {
					lines = append(lines, fmt.Sprintf("      - %s", col.Description))
				}
			}
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

func generateInstallSectionMarkdown(repository string) string {
	if repository == "" {
		return ""
	}

	httpInstallURL := fmt.Sprintf("http://localhost:42069/install?repo=%s", url.QueryEscape(repository))

	lines := []string{
		"",
		"## Installation",
		"",
		fmt.Sprintf("**[⬇️ Click here to install in Cauldron](%s)** _(requires Cauldron to be running)_", httpInstallURL),
		"",
		fmt.Sprintf("> **Repository**: `%s`", repository),
		"",
		"**Manual installation:**",
		"",
		"1. Open Cauldron",
		"2. Go to **Plugins** → **Install from Repository**",
		fmt.Sprintf("3. Paste: `%s`", repository),
		"4. Click **Install**",
		"",
	}

	return strings.Join(lines, "\n")
}

func generateInstallSectionHTML(repository string) string {
	if repository == "" {
		return ""
	}

	cauldronURL := fmt.Sprintf("cauldron://install?repo=%s", url.QueryEscape(repository))

	lines := []string{
		"",
		"## Installation",
		"",
		fmt.Sprintf("<a href=\"%s\">⬇️ <strong>Click here to install in Cauldron</strong></a>", cauldronURL),
		"",
		fmt.Sprintf("> **Repository**: `%s`", repository),
		"",
		"**Manual installation:**",
		"",
		"1. Open Cauldron",
		"2. Go to **Plugins** → **Install from Repository**",
		fmt.Sprintf("3. Paste: `%s`", repository),
		"4. Click **Install**",
		"",
	}

	return strings.Join(lines, "\n")
}

func generateHTMLWrapper(markdownContent string) string {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Table),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithUnsafe(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(markdownContent), &buf); err != nil {
		return markdownContent
	}

	htmlTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Plugin Documentation</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            max-width: 900px;
            margin: 50px auto;
            padding: 20px;
            line-height: 1.6;
            color: #333;
        }
        h1 {
            border-bottom: 2px solid #2196f3;
            padding-bottom: 10px;
            color: #1976d2;
        }
        h2 {
            margin-top: 30px;
            color: #1976d2;
            border-bottom: 1px solid #e0e0e0;
            padding-bottom: 5px;
        }
        h3 {
            margin-top: 20px;
            color: #424242;
        }
        h4 {
            color: #616161;
        }
        a {
            color: #2196f3;
            text-decoration: none;
            font-weight: 500;
        }
        a:hover {
            text-decoration: underline;
        }
        code {
            background: #f5f5f5;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: 'Courier New', monospace;
            font-size: 0.9em;
        }
        pre {
            background: #f5f5f5;
            padding: 15px;
            border-radius: 4px;
            overflow-x: auto;
        }
        pre code {
            background: none;
            padding: 0;
        }
        blockquote {
            border-left: 4px solid #2196f3;
            margin: 0;
            padding-left: 15px;
            color: #666;
            background: #f5f9fc;
            padding: 10px 15px;
            border-radius: 4px;
        }
        table {
            border-collapse: collapse;
            width: 100%;
            margin: 15px 0;
        }
        th, td {
            border: 1px solid #ddd;
            padding: 12px;
            text-align: left;
        }
        th {
            background-color: #2196f3;
            color: white;
            font-weight: 600;
        }
        tr:nth-child(even) {
            background-color: #f9f9f9;
        }
        ul, ol {
            padding-left: 25px;
        }
        li {
            margin: 5px 0;
        }
        strong {
            font-weight: 600;
            color: #1976d2;
        }
    </style>
</head>
<body>
%s
</body>
</html>`

	return fmt.Sprintf(htmlTemplate, buf.String())
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

func readRequirementsFile(pluginDir string, filename string) []string {
	if filename == "" {
		return nil
	}

	filePath := filepath.Join(pluginDir, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	var packages []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			packages = append(packages, line)
		}
	}
	return packages
}

func generatePluginDoc(plugin PluginConfig, pluginDir string, useHTTPProtocol bool) string {
	lines := []string{
		fmt.Sprintf("# %s\n", plugin.Plugin.Name),
	}

	var installSection string
	if useHTTPProtocol {
		installSection = generateInstallSectionMarkdown(plugin.Plugin.Repository)
	} else {
		installSection = generateInstallSectionHTML(plugin.Plugin.Repository)
	}

	if installSection != "" {
		lines = append(lines, installSection)
	}

	lines = append(lines,
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
	)

	if plugin.Annotation != nil {
		lines = append(lines, "## Sample Annotation\n")
		lines = append(lines, "This plugin supports sample annotation:\n")
		if plugin.Annotation.SamplesFrom != "" {
			lines = append(lines, fmt.Sprintf("- **Samples From**: `%s`", plugin.Annotation.SamplesFrom))
		}
		if plugin.Annotation.AnnotationFile != "" {
			lines = append(lines, fmt.Sprintf("- **Annotation File**: `%s`", plugin.Annotation.AnnotationFile))
		}
		lines = append(lines, "")
	}

	if len(plugin.Plots) > 0 {
		lines = append(lines, "## Visualizations\n")
		lines = append(lines, fmt.Sprintf("This plugin generates %d plot(s):\n", len(plugin.Plots)))
		for _, plot := range plugin.Plots {
			plotTitle := plot.Name
			if plot.Title != "" {
				plotTitle = plot.Title
			}

			plotInfo := fmt.Sprintf("### %s (`%s`)\n", plotTitle, plot.ID)
			lines = append(lines, plotInfo)

			if plot.Description != "" {
				lines = append(lines, plot.Description+"\n")
			}

			lines = append(lines, fmt.Sprintf("- **Type**: %s", plot.Type))

			if plot.Component != "" {
				lines = append(lines, fmt.Sprintf("- **Component**: `%s`", plot.Component))
			}

			lines = append(lines, fmt.Sprintf("- **Data Source**: `%s`", plot.DataSource))

			if plot.Default {
				lines = append(lines, "- **Default**: Yes")
			}

			if plot.Config.ImagePattern != "" {
				lines = append(lines, fmt.Sprintf("- **Image Pattern**: `%s`", plot.Config.ImagePattern))
				if plot.Config.ImagePatternType != "" {
					lines = append(lines, fmt.Sprintf("- **Pattern Type**: %s", plot.Config.ImagePatternType))
				}
			}

			if len(plot.Customization) > 0 {
				lines = append(lines, fmt.Sprintf("- **Customization Options**: %d available", len(plot.Customization)))
			}

			lines = append(lines, "")
		}
	}

	reqs := plugin.Execution.Requirements
	hasReqs := reqs.Python != "" || reqs.R != "" || len(reqs.Packages) > 0 || reqs.PythonRequirementsFile != "" || reqs.RPackagesFile != ""
	if hasReqs {
		lines = append(lines, "## Requirements\n")

		// Runtime version requirements
		if reqs.Python != "" {
			lines = append(lines, fmt.Sprintf("- **Python Version**: %s", reqs.Python))
		}
		if reqs.R != "" {
			lines = append(lines, fmt.Sprintf("- **R Version**: %s", reqs.R))
		}

		// Determine which package specification method is used
		hasInlinePackages := len(reqs.Packages) > 0
		hasPythonFile := reqs.PythonRequirementsFile != ""
		hasRFile := reqs.RPackagesFile != ""

		if hasInlinePackages {
			lines = append(lines, "\n### Package Dependencies (Inline)\n")
			lines = append(lines, "Packages are defined inline in the plugin configuration:\n")
			for _, pkg := range reqs.Packages {
				lines = append(lines, fmt.Sprintf("- `%s`", pkg))
			}
		}

		if hasPythonFile {
			pythonPackages := readRequirementsFile(pluginDir, reqs.PythonRequirementsFile)
			lines = append(lines, "\n### Python Dependencies (External File)\n")
			lines = append(lines, fmt.Sprintf("Dependencies are defined in: `%s`\n", reqs.PythonRequirementsFile))
			if len(pythonPackages) > 0 {
				for _, pkg := range pythonPackages {
					lines = append(lines, fmt.Sprintf("- `%s`", pkg))
				}
			}
		}

		if hasRFile {
			rPackages := readRequirementsFile(pluginDir, reqs.RPackagesFile)
			lines = append(lines, "\n### R Dependencies (External File)\n")
			lines = append(lines, fmt.Sprintf("Dependencies are defined in: `%s`\n", reqs.RPackagesFile))
			if len(rPackages) > 0 {
				for _, pkg := range rPackages {
					lines = append(lines, fmt.Sprintf("- `%s`", pkg))
				}
			}
		}

		// Add installation note
		if hasInlinePackages || hasPythonFile || hasRFile {
			lines = append(lines, "\n> **Note**: When you create a custom environment for this plugin, these dependencies will be automatically installed.")
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

	markdownDoc := generatePluginDoc(plugin, pluginDir, true)
	htmlDoc := generatePluginDoc(plugin, pluginDir, false)

	mdOutputPath := filepath.Join(pluginDir, "README.md")
	if err := os.WriteFile(mdOutputPath, []byte(markdownDoc), 0644); err != nil {
		fmt.Printf("[ERROR] Failed to write README.md: %v\n", err)
		os.Exit(1)
	}

	htmlOutputPath := filepath.Join(pluginDir, "README.html")
	htmlContent := generateHTMLWrapper(htmlDoc)
	if err := os.WriteFile(htmlOutputPath, []byte(htmlContent), 0644); err != nil {
		fmt.Printf("[ERROR] Failed to write README.html: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[SUCCESS] Documentation generated:\n")
	fmt.Printf("  - Markdown (HTTP protocol): %s\n", mdOutputPath)
	fmt.Printf("  - HTML (custom protocol):   %s\n", htmlOutputPath)
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
	fmt.Println(markdownDoc)
}
