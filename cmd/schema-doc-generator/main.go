package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type JSONSchema struct {
	Schema      string                `json:"$schema"`
	ID          string                `json:"$id"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Type        string                `json:"type"`
	Required    []string              `json:"required"`
	Properties  map[string]Property   `json:"properties"`
	Definitions map[string]Definition `json:"definitions"`
}

type Property struct {
	Type        interface{}         `json:"type"`
	Description string              `json:"description"`
	Ref         string              `json:"$ref"`
	Required    []string            `json:"required"`
	Properties  map[string]Property `json:"properties"`
	Items       *Property           `json:"items"`
	Enum        []interface{}       `json:"enum"`
	Pattern     string              `json:"pattern"`
	Default     interface{}         `json:"default"`
	Examples    []interface{}       `json:"examples"`
	MinLength   *int                `json:"minLength"`
	MinItems    *int                `json:"minItems"`
	Deprecated  bool                `json:"deprecated"`
	OneOf       []Property          `json:"oneOf"`
	AllOf       []Property          `json:"allOf"`
	AnyOf       []Property          `json:"anyOf"`
	If          *Property           `json:"if"`
	Then        *Property           `json:"then"`
}

type Definition struct {
	Type                 interface{}         `json:"type"`
	Description          string              `json:"description"`
	Required             []string            `json:"required"`
	Properties           map[string]Property `json:"properties"`
	Items                *Property           `json:"items"`
	Enum                 []interface{}       `json:"enum"`
	OneOf                []Property          `json:"oneOf"`
	AllOf                []Property          `json:"allOf"`
	AnyOf                []Property          `json:"anyOf"`
	AdditionalProperties interface{}         `json:"additionalProperties"`
}

func loadSchema(path string) (*JSONSchema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var schema JSONSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}

	return &schema, nil
}

func formatType(t interface{}) string {
	switch v := t.(type) {
	case string:
		return v
	case []interface{}:
		types := make([]string, len(v))
		for i, item := range v {
			types[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(types, " | ")
	default:
		return fmt.Sprintf("%v", t)
	}
}

func formatEnum(enum []interface{}) string {
	if len(enum) == 0 {
		return ""
	}
	values := make([]string, len(enum))
	for i, v := range enum {
		values[i] = fmt.Sprintf("`%v`", v)
	}
	return strings.Join(values, ", ")
}

func formatExamples(examples []interface{}) string {
	if len(examples) == 0 {
		return ""
	}
	values := make([]string, len(examples))
	for i, v := range examples {
		values[i] = fmt.Sprintf("`%v`", v)
	}
	return strings.Join(values, ", ")
}

func generateInputTypesDocs(schema *JSONSchema) string {
	inputDef, ok := schema.Definitions["input"]
	if !ok {
		return ""
	}

	typeProp, ok := inputDef.Properties["type"]
	if !ok {
		return ""
	}

	lines := []string{
		"## Input Types",
		"",
		"The dynamic form system supports the following input types:",
		"",
	}

	typeEnum := typeProp.Enum
	for _, t := range typeEnum {
		typeName := fmt.Sprintf("%v", t)
		lines = append(lines, fmt.Sprintf("### `%s`", typeName))
		lines = append(lines, "")
		lines = append(lines, generateInputTypeDetails(typeName, inputDef)...)
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func generateInputTypeDetails(typeName string, inputDef Definition) []string {
	lines := []string{}

	switch typeName {
	case "file":
		lines = append(lines,
			"File upload input for selecting files from the local filesystem.",
			"",
			"**Properties:**",
			"| Property | Type | Description |",
			"|----------|------|-------------|",
			"| `accept` | string | Accepted file extensions (e.g., `.csv,.tsv,.txt`) |",
			"| `disableAnnotationManagement` | boolean | Disable automatic annotation management |",
			"| `tableColumns` | array | Column definitions for table editor |",
			"",
			"**YAML Example:**",
			"```yaml",
			"- name: input_file",
			"  label: Input Data",
			"  type: file",
			"  required: true",
			"  accept: \".csv,.tsv,.txt\"",
			"  description: Upload your data file",
			"```",
		)

	case "directory":
		lines = append(lines,
			"Directory selector for choosing folders.",
			"",
			"**YAML Example:**",
			"```yaml",
			"- name: output_folder",
			"  label: Output Directory",
			"  type: directory",
			"  required: true",
			"  description: Select output directory",
			"```",
		)

	case "text":
		lines = append(lines,
			"Single-line text input field.",
			"",
			"**Properties:**",
			"| Property | Type | Description |",
			"|----------|------|-------------|",
			"| `placeholder` | string | Placeholder text |",
			"| `default` | string | Default value |",
			"",
			"**YAML Example:**",
			"```yaml",
			"- name: sample_name",
			"  label: Sample Name",
			"  type: text",
			"  required: true",
			"  placeholder: Enter sample name",
			"  default: \"sample_1\"",
			"```",
		)

	case "number":
		lines = append(lines,
			"Numeric input with optional min/max/step constraints.",
			"",
			"**Properties:**",
			"| Property | Type | Description |",
			"|----------|------|-------------|",
			"| `min` | number | Minimum allowed value |",
			"| `max` | number | Maximum allowed value |",
			"| `step` | number | Step increment |",
			"| `default` | number | Default value |",
			"",
			"**YAML Example:**",
			"```yaml",
			"- name: n_components",
			"  label: Number of Components",
			"  type: number",
			"  required: true",
			"  min: 1",
			"  max: 100",
			"  step: 1",
			"  default: 5",
			"```",
		)

	case "boolean":
		lines = append(lines,
			"Toggle/checkbox input for true/false values.",
			"",
			"**Properties:**",
			"| Property | Type | Description |",
			"|----------|------|-------------|",
			"| `default` | boolean | Default value (true/false) |",
			"",
			"**YAML Example:**",
			"```yaml",
			"- name: log_scale",
			"  label: Use Log Scale",
			"  type: boolean",
			"  default: false",
			"  description: Apply logarithmic transformation",
			"```",
		)

	case "select":
		lines = append(lines,
			"Dropdown select with predefined options.",
			"",
			"**Properties:**",
			"| Property | Type | Description |",
			"|----------|------|-------------|",
			"| `options` | array | List of options (strings or {value, label} objects) |",
			"| `optionsFromFile` | string | Load options from a text file |",
			"| `default` | string | Default selected value |",
			"",
			"**YAML Example (inline options):**",
			"```yaml",
			"- name: method",
			"  label: Analysis Method",
			"  type: select",
			"  required: true",
			"  default: \"pca\"",
			"  options:",
			"    - pca",
			"    - tsne",
			"    - umap",
			"```",
			"",
			"**YAML Example (value/label options):**",
			"```yaml",
			"- name: normalization",
			"  label: Normalization Method",
			"  type: select",
			"  options:",
			"    - value: \"none\"",
			"      label: \"No Normalization\"",
			"    - value: \"median\"",
			"      label: \"Median Centering\"",
			"    - value: \"quantile\"",
			"      label: \"Quantile Normalization\"",
			"```",
			"",
			"**YAML Example (from file):**",
			"```yaml",
			"- name: database",
			"  label: Database",
			"  type: select",
			"  optionsFromFile: databases.txt",
			"```",
		)

	case "multiselect-grouped":
		lines = append(lines,
			"Multi-select with grouped options organized by category.",
			"",
			"**Properties:**",
			"| Property | Type | Description |",
			"|----------|------|-------------|",
			"| `groups` | array | Array of groups, each with name and options |",
			"| `groupsFromFile` | string | Load groups from a JSON file |",
			"| `default` | array | Default selected values |",
			"",
			"**YAML Example:**",
			"```yaml",
			"- name: selected_fields",
			"  label: Fields to Include",
			"  type: multiselect-grouped",
			"  groups:",
			"    - name: \"Protein Information\"",
			"      options:",
			"        - value: \"protein_id\"",
			"          label: \"Protein ID\"",
			"        - value: \"gene_name\"",
			"          label: \"Gene Name\"",
			"    - name: \"Quantification\"",
			"      options:",
			"        - value: \"intensity\"",
			"          label: \"Intensity\"",
			"        - value: \"lfq\"",
			"          label: \"LFQ Intensity\"",
			"```",
			"",
			"**YAML Example (from file):**",
			"```yaml",
			"- name: columns",
			"  label: Columns",
			"  type: multiselect-grouped",
			"  groupsFromFile: fields.json",
			"```",
		)

	case "column-selector":
		lines = append(lines,
			"Dynamic column selector that reads columns from a file input.",
			"",
			"**Properties:**",
			"| Property | Type | Description |",
			"|----------|------|-------------|",
			"| `sourceFile` | string | Name of file input to read columns from (required) |",
			"| `multiple` | boolean | Allow multiple column selection |",
			"| `default` | string/array | Default selected column(s) |",
			"",
			"**YAML Example (single column):**",
			"```yaml",
			"- name: id_column",
			"  label: ID Column",
			"  type: column-selector",
			"  sourceFile: input_file",
			"  multiple: false",
			"  required: true",
			"```",
			"",
			"**YAML Example (multiple columns):**",
			"```yaml",
			"- name: sample_columns",
			"  label: Sample Columns",
			"  type: column-selector",
			"  sourceFile: input_file",
			"  multiple: true",
			"  description: Select columns containing sample data",
			"```",
		)

	case "color":
		lines = append(lines,
			"Color picker for selecting a single color value.",
			"",
			"**Properties:**",
			"| Property | Type | Description |",
			"|----------|------|-------------|",
			"| `default` | string | Default color value (hex format) |",
			"",
			"**YAML Example:**",
			"```yaml",
			"- name: highlight_color",
			"  label: Highlight Color",
			"  type: color",
			"  default: \"#FF5722\"",
			"```",
		)

	case "color-map":
		lines = append(lines,
			"Color mapping input for assigning colors to categories.",
			"",
			"**Properties:**",
			"| Property | Type | Description |",
			"|----------|------|-------------|",
			"| `keysFrom` | string | Name of input providing category keys (e.g., column-selector) |",
			"",
			"**YAML Example:**",
			"```yaml",
			"- name: condition_colors",
			"  label: Condition Colors",
			"  type: color-map",
			"  keysFrom: selected_conditions",
			"  description: Assign colors to each condition",
			"```",
		)
	}

	return lines
}

func generateCommonProperties(schema *JSONSchema) string {
	inputDef, ok := schema.Definitions["input"]
	if !ok {
		return ""
	}

	lines := []string{
		"## Common Input Properties",
		"",
		"All input types support these common properties:",
		"",
		"| Property | Type | Required | Description |",
		"|----------|------|----------|-------------|",
	}

	props := []struct {
		name     string
		typeName string
		required bool
		desc     string
	}{
		{"name", "string", true, "Unique identifier for the input (snake_case or camelCase)"},
		{"label", "string", true, "Human-readable label displayed in the UI"},
		{"type", "string", true, "Input type (file, text, number, boolean, select, etc.)"},
		{"required", "boolean", false, "Whether this input is required (default: false)"},
		{"default", "any", false, "Default value for the input"},
		{"description", "string", false, "Help text displayed below the input"},
		{"placeholder", "string", false, "Placeholder text for text inputs"},
		{"visibleWhen", "object", false, "Conditional visibility based on other inputs"},
	}

	for _, p := range props {
		req := "No"
		if p.required {
			req = "Yes"
		}
		lines = append(lines, fmt.Sprintf("| `%s` | %s | %s | %s |", p.name, p.typeName, req, p.desc))
	}

	lines = append(lines, "")
	lines = append(lines, "**YAML Example with common properties:**")
	lines = append(lines, "```yaml")
	lines = append(lines, "- name: sample_count")
	lines = append(lines, "  label: Sample Count")
	lines = append(lines, "  type: number")
	lines = append(lines, "  required: true")
	lines = append(lines, "  default: 10")
	lines = append(lines, "  description: Number of samples to process")
	lines = append(lines, "  min: 1")
	lines = append(lines, "  max: 100")
	lines = append(lines, "```")
	lines = append(lines, "")

	_ = inputDef
	return strings.Join(lines, "\n")
}

func generateVisibilityDocs(schema *JSONSchema) string {
	visDef, ok := schema.Definitions["visibilityCondition"]
	if !ok {
		return ""
	}

	lines := []string{
		"## Conditional Visibility",
		"",
		"Inputs can be conditionally shown/hidden based on other input values using `visibleWhen`.",
		"",
		"### Properties",
		"",
		"| Property | Type | Description |",
		"|----------|------|-------------|",
		"| `field` | string | Name of the field to check (required) |",
		"| `equals` | any | Show when field equals this exact value |",
		"| `equalsAny` | array | Show when field equals any of these values |",
		"",
		"**Note:** Use either `equals` or `equalsAny`, not both.",
		"",
		"### Examples",
		"",
		"**Single value condition:**",
		"```yaml",
		"- name: advanced_options",
		"  label: Advanced Options",
		"  type: boolean",
		"  default: false",
		"",
		"- name: threshold",
		"  label: Threshold",
		"  type: number",
		"  visibleWhen:",
		"    field: advanced_options",
		"    equals: true",
		"```",
		"",
		"**Multiple value condition:**",
		"```yaml",
		"- name: method",
		"  label: Method",
		"  type: select",
		"  options:",
		"    - simple",
		"    - advanced",
		"    - expert",
		"",
		"- name: expert_settings",
		"  label: Expert Settings",
		"  type: text",
		"  visibleWhen:",
		"    field: method",
		"    equalsAny:",
		"      - advanced",
		"      - expert",
		"```",
		"",
	}

	_ = visDef
	return strings.Join(lines, "\n")
}

func generateArgsMappingDocs(schema *JSONSchema) string {
	argsDef, ok := schema.Definitions["argMapping"]
	if !ok {
		return ""
	}

	lines := []string{
		"## Execution and Args Mapping",
		"",
		"The `execution.argsMapping` section defines how inputs are passed to the script.",
		"",
		"### Simple Mapping",
		"",
		"Direct string mapping uses the flag as-is:",
		"",
		"```yaml",
		"execution:",
		"  argsMapping:",
		"    input_file: \"--input\"",
		"    n_components: \"--n-components\"",
		"  outputDir: \"--output\"",
		"```",
		"",
		"### Advanced Mapping",
		"",
		"For complex scenarios, use object mapping:",
		"",
		"| Property | Type | Description |",
		"|----------|------|-------------|",
		"| `flag` | string | Command-line flag (e.g., `--input`) |",
		"| `transform` | string | Value transformation: `comma-join`, `space-join`, `json-encode`, `color-map` |",
		"| `when` | string | Only include when value matches |",
		"| `value` | string | Fixed value to pass instead of actual value |",
		"| `passAsValue` | boolean | For booleans, pass 'true'/'false' as string value |",
		"",
		"### Transform Examples",
		"",
		"**Comma-join (for multiple selections):**",
		"```yaml",
		"execution:",
		"  argsMapping:",
		"    selected_columns:",
		"      flag: \"--columns\"",
		"      transform: comma-join",
		"```",
		"Result: `--columns col1,col2,col3`",
		"",
		"**JSON-encode (for complex data):**",
		"```yaml",
		"execution:",
		"  argsMapping:",
		"    color_map:",
		"      flag: \"--colors\"",
		"      transform: json-encode",
		"```",
		"Result: `--colors '{\"group1\":\"#FF0000\",\"group2\":\"#00FF00\"}'`",
		"",
		"**Conditional inclusion:**",
		"```yaml",
		"execution:",
		"  argsMapping:",
		"    use_log_scale:",
		"      flag: \"--log-scale\"",
		"      when: \"true\"",
		"```",
		"Only includes `--log-scale` when value is true.",
		"",
		"**Boolean as value:**",
		"```yaml",
		"execution:",
		"  argsMapping:",
		"    enabled:",
		"      flag: \"--enabled\"",
		"      passAsValue: true",
		"```",
		"Result: `--enabled true` or `--enabled false`",
		"",
	}

	_ = argsDef
	return strings.Join(lines, "\n")
}

func generateOutputsDocs(schema *JSONSchema) string {
	outputDef, ok := schema.Definitions["output"]
	if !ok {
		return ""
	}

	lines := []string{
		"## Outputs Configuration",
		"",
		"Define output files produced by the plugin.",
		"",
		"### Properties",
		"",
		"| Property | Type | Required | Description |",
		"|----------|------|----------|-------------|",
		"| `name` | string | Yes | Unique output identifier |",
		"| `path` | string | Yes | Output file path (supports wildcards) |",
		"| `type` | string | Yes | Output type: `data`, `image`, `plot`, `log` |",
		"| `description` | string | No | Description of the output |",
		"| `format` | string | No | File format: `tsv`, `csv`, `json`, `txt`, `svg`, `png`, `pdf` |",
		"",
		"### Examples",
		"",
		"```yaml",
		"outputs:",
		"  - name: results",
		"    path: \"results.tsv\"",
		"    type: data",
		"    format: tsv",
		"    description: Main analysis results",
		"",
		"  - name: pca_plot",
		"    path: \"pca_plot.svg\"",
		"    type: image",
		"    format: svg",
		"    description: PCA visualization",
		"",
		"  - name: all_plots",
		"    path: \"plots/*.png\"",
		"    type: image",
		"    format: png",
		"    description: All generated plots",
		"```",
		"",
	}

	_ = outputDef
	return strings.Join(lines, "\n")
}

func generatePlotsDocs(schema *JSONSchema) string {
	plotDef, ok := schema.Definitions["plot"]
	if !ok {
		return ""
	}

	lines := []string{
		"## Plots Configuration",
		"",
		"Define interactive visualizations for the plugin.",
		"",
		"### Properties",
		"",
		"| Property | Type | Required | Description |",
		"|----------|------|----------|-------------|",
		"| `id` | string | Yes | Unique plot identifier (kebab-case) |",
		"| `name` | string | Yes | Human-readable plot name |",
		"| `type` | string | Yes | Plot type: `scatter`, `line`, `bar`, `heatmap`, `volcano` |",
		"| `component` | string | Yes | Angular component name |",
		"| `dataSource` | string | Yes | Output name to use as data source |",
		"| `config` | object | No | Plot configuration (axes, etc.) |",
		"| `customization` | array | No | User-customizable options |",
		"",
		"### Examples",
		"",
		"```yaml",
		"plots:",
		"  - id: pca-scatter",
		"    name: PCA Plot",
		"    type: scatter",
		"    component: ScatterPlotComponent",
		"    dataSource: pca_results",
		"    config:",
		"      axes:",
		"        x: PC1",
		"        y: PC2",
		"        colorBy: condition",
		"        labels: sample_name",
		"    customization:",
		"      - name: point_size",
		"        label: Point Size",
		"        type: number",
		"        default: 8",
		"        min: 1",
		"        max: 20",
		"```",
		"",
	}

	_ = plotDef
	return strings.Join(lines, "\n")
}

func generateAnnotationDocs(schema *JSONSchema) string {
	annDef, ok := schema.Definitions["annotation"]
	if !ok {
		return ""
	}

	lines := []string{
		"## Sample Annotation",
		"",
		"Enable sample annotation support for the plugin.",
		"",
		"### Properties",
		"",
		"| Property | Type | Description |",
		"|----------|------|-------------|",
		"| `samplesFrom` | string | Name of input field containing sample names |",
		"| `annotationFile` | string | Name of optional annotation file input |",
		"",
		"### Example",
		"",
		"```yaml",
		"inputs:",
		"  - name: sample_columns",
		"    label: Sample Columns",
		"    type: column-selector",
		"    sourceFile: input_file",
		"    multiple: true",
		"",
		"  - name: annotation_file",
		"    label: Annotation File",
		"    type: file",
		"    accept: \".tsv,.csv\"",
		"    required: false",
		"",
		"annotation:",
		"  samplesFrom: sample_columns",
		"  annotationFile: annotation_file",
		"```",
		"",
	}

	_ = annDef
	return strings.Join(lines, "\n")
}

func generateTableEditorDocs(schema *JSONSchema) string {
	tableDef, ok := schema.Definitions["tableColumn"]
	if !ok {
		return ""
	}

	lines := []string{
		"## Table Editor",
		"",
		"File inputs can include a built-in table editor for structured data.",
		"",
		"### Column Properties",
		"",
		"| Property | Type | Required | Description |",
		"|----------|------|----------|-------------|",
		"| `name` | string | Yes | Column name in file header |",
		"| `label` | string | Yes | Display label in editor |",
		"| `type` | string | No | Column type: `text`, `number` (default: text) |",
		"| `required` | boolean | No | Column must have non-empty values |",
		"| `description` | string | No | Help text/tooltip |",
		"",
		"### Example",
		"",
		"```yaml",
		"inputs:",
		"  - name: annotation_file",
		"    label: Annotation File",
		"    type: file",
		"    accept: \".tsv\"",
		"    disableAnnotationManagement: true",
		"    tableColumns:",
		"      - name: Run",
		"        label: Run Name",
		"        type: text",
		"        required: true",
		"      - name: Channel",
		"        label: TMT Channel",
		"        type: text",
		"        required: true",
		"      - name: Condition",
		"        label: Condition",
		"        type: text",
		"        required: true",
		"      - name: BioReplicate",
		"        label: Biological Replicate",
		"        type: number",
		"        required: true",
		"```",
		"",
	}

	_ = tableDef
	return strings.Join(lines, "\n")
}

func generateRequirementsDocs(schema *JSONSchema) string {
	reqDef, ok := schema.Definitions["requirements"]
	if !ok {
		return ""
	}

	lines := []string{
		"## Requirements Configuration",
		"",
		"Specify runtime version and package dependencies.",
		"",
		"### Properties",
		"",
		"| Property | Type | Description |",
		"|----------|------|-------------|",
		"| `python` | string | Python version requirement (e.g., `>=3.11`) |",
		"| `r` | string | R version requirement (e.g., `>=4.0`) |",
		"| `packages` | array | Inline list of packages (Python or R) |",
		"| `pythonRequirementsFile` | string | Path to requirements.txt |",
		"| `rPackagesFile` | string | Path to R packages file |",
		"",
		"### Examples",
		"",
		"**Inline packages:**",
		"```yaml",
		"execution:",
		"  requirements:",
		"    python: \">=3.11\"",
		"    packages:",
		"      - pandas>=2.0.0",
		"      - numpy>=1.24.0",
		"      - scikit-learn>=1.3.0",
		"```",
		"",
		"**External requirements file:**",
		"```yaml",
		"execution:",
		"  requirements:",
		"    python: \">=3.11\"",
		"    pythonRequirementsFile: requirements.txt",
		"```",
		"",
		"**R packages:**",
		"```yaml",
		"execution:",
		"  requirements:",
		"    r: \">=4.0\"",
		"    packages:",
		"      - ggplot2",
		"      - dplyr",
		"      - tidyr",
		"```",
		"",
	}

	_ = reqDef
	return strings.Join(lines, "\n")
}

func generateExampleDocs(schema *JSONSchema) string {
	exDef, ok := schema.Definitions["example"]
	if !ok {
		return ""
	}

	lines := []string{
		"## Example Data",
		"",
		"Provide example data for testing the plugin.",
		"",
		"### Properties",
		"",
		"| Property | Type | Description |",
		"|----------|------|-------------|",
		"| `enabled` | boolean | Enable example data support |",
		"| `values` | object | Map of input names to example values |",
		"",
		"### Example",
		"",
		"```yaml",
		"example:",
		"  enabled: true",
		"  values:",
		"    n_components: 5",
		"    method: \"pca\"",
		"    log_scale: true",
		"```",
		"",
		"**Note:** File inputs in examples should reference files in the plugin's `examples/` directory.",
		"",
	}

	_ = exDef
	return strings.Join(lines, "\n")
}

func generatePluginMetadataDocs(schema *JSONSchema) string {
	pluginProp, ok := schema.Properties["plugin"]
	if !ok {
		return ""
	}

	lines := []string{
		"## Plugin Metadata",
		"",
		"Required metadata for the plugin.",
		"",
		"### Properties",
		"",
		"| Property | Type | Required | Description |",
		"|----------|------|----------|-------------|",
		"| `id` | string | Yes | Unique plugin identifier (kebab-case) |",
		"| `name` | string | Yes | Human-readable plugin name |",
		"| `description` | string | Yes | Brief description (min 10 chars) |",
		"| `version` | string | Yes | Semantic version (e.g., `1.0.0`) |",
		"| `category` | string | Yes | Plugin category |",
		"| `author` | string | No | Plugin author |",
		"| `subcategory` | string | No | Finer organization |",
		"| `icon` | string | No | Material icon name |",
		"| `repository` | string | No | Git repository URL |",
		"",
		"### Categories",
		"",
		"Valid categories: `analysis`, `preprocessing`, `visualization`, `utilities`, `statistics`",
		"",
		"### Example",
		"",
		"```yaml",
		"plugin:",
		"  id: pca-analysis",
		"  name: PCA Analysis",
		"  description: Principal Component Analysis for dimensionality reduction",
		"  version: \"1.0.0\"",
		"  author: CauldronGO Team",
		"  category: analysis",
		"  icon: analytics",
		"  repository: https://github.com/example/pca-plugin",
		"```",
		"",
	}

	_ = pluginProp
	return strings.Join(lines, "\n")
}

func generateRuntimeDocs(schema *JSONSchema) string {
	runtimeProp, ok := schema.Properties["runtime"]
	if !ok {
		return ""
	}

	lines := []string{
		"## Runtime Configuration",
		"",
		"Configure the execution environment for the plugin.",
		"",
		"### Properties",
		"",
		"| Property | Type | Required | Description |",
		"|----------|------|----------|-------------|",
		"| `environments` | array | Yes | Runtime environments: `python`, `r`, `julia`, `node`, `direct`, `docker` |",
		"| `entrypoint` | string | Yes | Script filename or executable |",
		"| `docker` | object | No | Docker configuration (if using docker environment) |",
		"",
		"### Examples",
		"",
		"**Python script:**",
		"```yaml",
		"runtime:",
		"  environments:",
		"    - python",
		"  entrypoint: analysis.py",
		"```",
		"",
		"**R script:**",
		"```yaml",
		"runtime:",
		"  environments:",
		"    - r",
		"  entrypoint: process.R",
		"```",
		"",
		"**Mixed Python + R:**",
		"```yaml",
		"runtime:",
		"  environments:",
		"    - python",
		"    - r",
		"  entrypoint: main.py",
		"```",
		"",
		"**Docker:**",
		"```yaml",
		"runtime:",
		"  environments:",
		"    - docker",
		"  entrypoint: run.sh",
		"  docker:",
		"    image: python:3.11-slim",
		"```",
		"",
	}

	_ = runtimeProp
	return strings.Join(lines, "\n")
}

func generateFullDocument(schema *JSONSchema) string {
	lines := []string{
		fmt.Sprintf("# %s", schema.Title),
		"",
		fmt.Sprintf("_%s_", schema.Description),
		"",
		"---",
		"",
		"## Table of Contents",
		"",
		"1. [Plugin Metadata](#plugin-metadata)",
		"2. [Runtime Configuration](#runtime-configuration)",
		"3. [Common Input Properties](#common-input-properties)",
		"4. [Input Types](#input-types)",
		"5. [Conditional Visibility](#conditional-visibility)",
		"6. [Table Editor](#table-editor)",
		"7. [Outputs Configuration](#outputs-configuration)",
		"8. [Plots Configuration](#plots-configuration)",
		"9. [Sample Annotation](#sample-annotation)",
		"10. [Execution and Args Mapping](#execution-and-args-mapping)",
		"11. [Requirements Configuration](#requirements-configuration)",
		"12. [Example Data](#example-data)",
		"13. [Complete Plugin Example](#complete-plugin-example)",
		"",
		"---",
		"",
	}

	lines = append(lines, generatePluginMetadataDocs(schema))
	lines = append(lines, generateRuntimeDocs(schema))
	lines = append(lines, generateCommonProperties(schema))
	lines = append(lines, generateInputTypesDocs(schema))
	lines = append(lines, generateVisibilityDocs(schema))
	lines = append(lines, generateTableEditorDocs(schema))
	lines = append(lines, generateOutputsDocs(schema))
	lines = append(lines, generatePlotsDocs(schema))
	lines = append(lines, generateAnnotationDocs(schema))
	lines = append(lines, generateArgsMappingDocs(schema))
	lines = append(lines, generateRequirementsDocs(schema))
	lines = append(lines, generateExampleDocs(schema))
	lines = append(lines, generateCompleteExample())

	return strings.Join(lines, "\n")
}

func generateCompleteExample() string {
	return `## Complete Plugin Example

Here is a complete example of a plugin.yaml file:

` + "```yaml" + `
plugin:
  id: dose-response-analysis
  name: Dose-Response Analysis
  description: Analyze dose-response relationships and calculate EC50 values
  version: "1.0.0"
  author: CauldronGO Team
  category: analysis
  subcategory: dose-response
  icon: analytics

runtime:
  environments:
    - python
  entrypoint: analysis.py

inputs:
  - name: input_file
    label: Input Data
    type: file
    required: true
    accept: ".csv,.tsv"
    description: Data file with dose and response columns

  - name: dose_column
    label: Dose Column
    type: column-selector
    sourceFile: input_file
    multiple: false
    required: true

  - name: response_columns
    label: Response Columns
    type: column-selector
    sourceFile: input_file
    multiple: true
    required: true

  - name: model_type
    label: Model Type
    type: select
    required: true
    default: "4pl"
    options:
      - value: "4pl"
        label: "4-Parameter Logistic"
      - value: "3pl"
        label: "3-Parameter Logistic"

  - name: log_transform
    label: Log Transform Dose
    type: boolean
    default: true

  - name: confidence_level
    label: Confidence Level
    type: number
    default: 0.95
    min: 0.8
    max: 0.99
    step: 0.01

outputs:
  - name: results
    path: "results.tsv"
    type: data
    format: tsv
    description: EC50 values and model parameters

  - name: dose_response_plots
    path: "plots/*.svg"
    type: image
    format: svg
    description: Dose-response curves

annotation:
  samplesFrom: response_columns

execution:
  argsMapping:
    input_file: "--input"
    dose_column: "--dose-col"
    response_columns:
      flag: "--response-cols"
      transform: comma-join
    model_type: "--model"
    log_transform:
      flag: "--log-dose"
      when: "true"
    confidence_level: "--ci"
  outputDir: "--output"
  requirements:
    python: ">=3.11"
    packages:
      - pandas>=2.0.0
      - numpy>=1.24.0
      - scipy>=1.11.0
      - matplotlib>=3.7.0

example:
  enabled: true
  values:
    model_type: "4pl"
    log_transform: true
    confidence_level: 0.95

diagram:
  enabled: true

citation:
  enabled: true
` + "```" + `
`
}

func main() {
	schemaPath := "schemas/plugin-schema.json"
	if len(os.Args) > 1 {
		schemaPath = os.Args[1]
	}

	schema, err := loadSchema(schemaPath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to load schema: %v\n", err)
		os.Exit(1)
	}

	doc := generateFullDocument(schema)

	outputPath := filepath.Join(filepath.Dir(schemaPath), "PLUGIN_YAML_REFERENCE.md")
	if err := os.WriteFile(outputPath, []byte(doc), 0644); err != nil {
		fmt.Printf("[ERROR] Failed to write documentation: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[SUCCESS] Schema documentation generated: %s\n", outputPath)
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
	fmt.Println(doc)
}
