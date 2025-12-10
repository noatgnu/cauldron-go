package models

type PluginCategory string

const (
	PluginCategoryAnalysis      PluginCategory = "analysis"
	PluginCategoryUtilities     PluginCategory = "utilities"
	PluginCategoryPreprocessing PluginCategory = "preprocessing"
	PluginCategoryVisualization PluginCategory = "visualization"
)

type InputTransform string

const (
	TransformCommaJoin  InputTransform = "comma-join"
	TransformSpaceJoin  InputTransform = "space-join"
	TransformJSONEncode InputTransform = "json-encode"
)

type VisibilityCondition struct {
	Field     string        `yaml:"field" json:"field"`
	Equals    interface{}   `yaml:"equals,omitempty" json:"equals,omitempty"`
	EqualsAny []interface{} `yaml:"equalsAny,omitempty" json:"equalsAny,omitempty"`
}

type FieldOption struct {
	Value string `yaml:"value" json:"value"`
	Label string `yaml:"label" json:"label"`
}

type FieldGroup struct {
	Name    string        `yaml:"name" json:"name"`
	Options []FieldOption `yaml:"options" json:"options"`
}

type PluginInputV2 struct {
	Name            string               `yaml:"name" json:"name"`
	Label           string               `yaml:"label" json:"label"`
	Type            PluginInputType      `yaml:"type" json:"type"`
	Required        bool                 `yaml:"required" json:"required"`
	Default         interface{}          `yaml:"default,omitempty" json:"default,omitempty"`
	Options         []string             `yaml:"options,omitempty" json:"options,omitempty"`
	OptionsFromFile string               `yaml:"optionsFromFile,omitempty" json:"optionsFromFile,omitempty"`
	Groups          []FieldGroup         `yaml:"groups,omitempty" json:"groups,omitempty"`
	GroupsFromFile  string               `yaml:"groupsFromFile,omitempty" json:"groupsFromFile,omitempty"`
	Description     string               `yaml:"description,omitempty" json:"description,omitempty"`
	Placeholder     string               `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`
	Accept          string               `yaml:"accept,omitempty" json:"accept,omitempty"`
	Multiple        bool                 `yaml:"multiple,omitempty" json:"multiple,omitempty"`
	SourceFile      string               `yaml:"sourceFile,omitempty" json:"sourceFile,omitempty"`
	Min             *float64             `yaml:"min,omitempty" json:"min,omitempty"`
	Max             *float64             `yaml:"max,omitempty" json:"max,omitempty"`
	Step            *float64             `yaml:"step,omitempty" json:"step,omitempty"`
	VisibleWhen     *VisibilityCondition `yaml:"visibleWhen,omitempty" json:"visibleWhen,omitempty"`
}

type PluginOutputV2 struct {
	Name        string `yaml:"name" json:"name"`
	Path        string `yaml:"path" json:"path"`
	Type        string `yaml:"type" json:"type"`
	Description string `yaml:"description" json:"description"`
	Format      string `yaml:"format" json:"format"`
}

type PlotCustomization struct {
	Name    string          `yaml:"name" json:"name"`
	Label   string          `yaml:"label" json:"label"`
	Type    PluginInputType `yaml:"type" json:"type"`
	Default interface{}     `yaml:"default,omitempty" json:"default,omitempty"`
	Min     *float64        `yaml:"min,omitempty" json:"min,omitempty"`
	Max     *float64        `yaml:"max,omitempty" json:"max,omitempty"`
}

type PlotAxes struct {
	X       string  `yaml:"x" json:"x"`
	Y       string  `yaml:"y" json:"y"`
	ColorBy *string `yaml:"colorBy,omitempty" json:"colorBy,omitempty"`
	SizeBy  *string `yaml:"sizeBy,omitempty" json:"sizeBy,omitempty"`
	Labels  *string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

type PlotConfigData struct {
	Axes PlotAxes `yaml:"axes" json:"axes"`
}

type PluginPlot struct {
	ID            string              `yaml:"id" json:"id"`
	Name          string              `yaml:"name" json:"name"`
	Type          string              `yaml:"type" json:"type"`
	Component     string              `yaml:"component" json:"component"`
	DataSource    string              `yaml:"dataSource" json:"dataSource"`
	Config        PlotConfigData      `yaml:"config" json:"config"`
	Customization []PlotCustomization `yaml:"customization" json:"customization"`
}

type ArgMapping struct {
	Flag      *string         `yaml:"flag,omitempty" json:"flag,omitempty"`
	Transform *InputTransform `yaml:"transform,omitempty" json:"transform,omitempty"`
	When      *string         `yaml:"when,omitempty" json:"when,omitempty"`
	Value     *string         `yaml:"value,omitempty" json:"value,omitempty"`
}

type Requirements struct {
	Python   string   `yaml:"python,omitempty" json:"python,omitempty"`
	R        string   `yaml:"r,omitempty" json:"r,omitempty"`
	Packages []string `yaml:"packages,omitempty" json:"packages,omitempty"`
}

type PluginExecution struct {
	ArgsMapping  map[string]interface{} `yaml:"argsMapping" json:"argsMapping"`
	OutputDir    string                 `yaml:"outputDir" json:"outputDir"`
	Requirements Requirements           `yaml:"requirements,omitempty" json:"requirements,omitempty"`
}

type PluginRuntimeV2 struct {
	Type   string `yaml:"type" json:"type"`
	Script string `yaml:"script" json:"script"`
}

type PluginMetadata struct {
	ID          string         `yaml:"id" json:"id"`
	Name        string         `yaml:"name" json:"name"`
	Description string         `yaml:"description" json:"description"`
	Version     string         `yaml:"version" json:"version"`
	Author      string         `yaml:"author,omitempty" json:"author,omitempty"`
	Category    PluginCategory `yaml:"category" json:"category"`
	Icon        string         `yaml:"icon,omitempty" json:"icon,omitempty"`
}

type ExampleData struct {
	Enabled bool                   `yaml:"enabled" json:"enabled"`
	Values  map[string]interface{} `yaml:"values" json:"values"`
}

type PluginDefinition struct {
	Plugin    PluginMetadata   `yaml:"plugin" json:"plugin"`
	Runtime   PluginRuntimeV2  `yaml:"runtime" json:"runtime"`
	Inputs    []PluginInputV2  `yaml:"inputs" json:"inputs"`
	Outputs   []PluginOutputV2 `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	Plots     []PluginPlot     `yaml:"plots,omitempty" json:"plots,omitempty"`
	Execution PluginExecution  `yaml:"execution" json:"execution"`
	Example   *ExampleData     `yaml:"example,omitempty" json:"example,omitempty"`
}

type PluginV2 struct {
	Definition PluginDefinition `json:"definition"`
	FolderPath string           `json:"folderPath"`
	ScriptPath string           `json:"scriptPath"`
}

type PluginExecutionRequestV2 struct {
	PluginID   string                 `json:"pluginId"`
	Parameters map[string]interface{} `json:"parameters"`
}
