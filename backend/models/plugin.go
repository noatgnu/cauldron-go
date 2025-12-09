package models

type PluginRuntime string

const (
	PluginRuntimePython      PluginRuntime = "python"
	PluginRuntimeR           PluginRuntime = "r"
	PluginRuntimePythonWithR PluginRuntime = "pythonWithR"
)

type PluginInputType string

const (
	PluginInputTypeFile           PluginInputType = "file"
	PluginInputTypeText           PluginInputType = "text"
	PluginInputTypeNumber         PluginInputType = "number"
	PluginInputTypeBoolean        PluginInputType = "boolean"
	PluginInputTypeSelect         PluginInputType = "select"
	PluginInputTypeMultiSelect    PluginInputType = "multiselect"
	PluginInputTypeColumnSelector PluginInputType = "column-selector"
)

type PluginInput struct {
	Name        string          `yaml:"name" json:"name"`
	Label       string          `yaml:"label" json:"label"`
	Type        PluginInputType `yaml:"type" json:"type"`
	Required    bool            `yaml:"required" json:"required"`
	Default     interface{}     `yaml:"default,omitempty" json:"default,omitempty"`
	Options     []string        `yaml:"options,omitempty" json:"options,omitempty"`
	Description string          `yaml:"description,omitempty" json:"description,omitempty"`
	Placeholder string          `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`
}

type PluginOutput struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
}

type PluginScript struct {
	Path string `yaml:"path" json:"path"`
}

type PluginConfig struct {
	Name        string         `yaml:"name" json:"name"`
	Description string         `yaml:"description" json:"description"`
	Version     string         `yaml:"version" json:"version"`
	Author      string         `yaml:"author,omitempty" json:"author,omitempty"`
	Runtime     PluginRuntime  `yaml:"runtime" json:"runtime"`
	Script      PluginScript   `yaml:"script" json:"script"`
	Inputs      []PluginInput  `yaml:"inputs" json:"inputs"`
	Outputs     []PluginOutput `yaml:"outputs,omitempty" json:"outputs,omitempty"`
}

type Plugin struct {
	ID         string       `json:"id"`
	Config     PluginConfig `json:"config"`
	FolderPath string       `json:"folderPath"`
	ScriptPath string       `json:"scriptPath"`
}

type PluginExecutionRequest struct {
	PluginID   string                 `json:"pluginId"`
	Parameters map[string]interface{} `json:"parameters"`
}
