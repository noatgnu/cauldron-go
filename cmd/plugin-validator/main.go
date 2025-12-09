package main

import (
	"fmt"
	"os"
	"path/filepath"

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

type PluginConfig struct {
	Plugin    PluginMetadata  `yaml:"plugin"`
	Runtime   PluginRuntime   `yaml:"runtime"`
	Inputs    []PluginInput   `yaml:"inputs"`
	Outputs   []PluginOutput  `yaml:"outputs,omitempty"`
	Execution PluginExecution `yaml:"execution"`
	Example   *ExampleData    `yaml:"example,omitempty"`
}

func printError(msg string) {
	fmt.Printf("[ERROR] %s\n", msg)
}

func printSuccess(msg string) {
	fmt.Printf("[SUCCESS] %s\n", msg)
}

func printWarning(msg string) {
	fmt.Printf("[WARNING] %s\n", msg)
}

func validatePlugin(pluginPath string) (bool, []string) {
	var errors []string
	var warnings []string

	// Load plugin YAML
	data, err := os.ReadFile(pluginPath)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to read file: %v", err))
		return false, errors
	}

	var plugin PluginConfig
	if err := yaml.Unmarshal(data, &plugin); err != nil {
		errors = append(errors, fmt.Sprintf("Invalid YAML: %v", err))
		return false, errors
	}

	// Validate required fields
	if plugin.Plugin.ID == "" {
		errors = append(errors, "plugin.id is required")
	}
	if plugin.Plugin.Name == "" {
		errors = append(errors, "plugin.name is required")
	}
	if plugin.Plugin.Description == "" {
		errors = append(errors, "plugin.description is required")
	}
	if plugin.Plugin.Version == "" {
		errors = append(errors, "plugin.version is required")
	}
	if plugin.Plugin.Category == "" {
		errors = append(errors, "plugin.category is required")
	}

	// Validate runtime
	if plugin.Runtime.Type == "" {
		errors = append(errors, "runtime.type is required")
	} else {
		validRuntimes := map[string]bool{"python": true, "r": true, "pythonWithR": true}
		if !validRuntimes[plugin.Runtime.Type] {
			errors = append(errors, fmt.Sprintf("Invalid runtime.type: %s", plugin.Runtime.Type))
		}
	}

	if plugin.Runtime.Script == "" {
		errors = append(errors, "runtime.script is required")
	} else {
		// Check if script exists
		pluginDir := filepath.Dir(pluginPath)
		scriptPath := filepath.Join(pluginDir, plugin.Runtime.Script)
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			errors = append(errors, fmt.Sprintf("Script not found: %s", plugin.Runtime.Script))
		}
	}

	// Validate inputs
	if len(plugin.Inputs) == 0 {
		warnings = append(warnings, "No inputs defined")
	}

	inputNames := make(map[string]bool)
	for i, input := range plugin.Inputs {
		if input.Name == "" {
			errors = append(errors, fmt.Sprintf("inputs[%d].name is required", i))
		} else {
			inputNames[input.Name] = true
		}

		if input.Label == "" {
			errors = append(errors, fmt.Sprintf("inputs[%d].label is required", i))
		}

		if input.Type == "" {
			errors = append(errors, fmt.Sprintf("inputs[%d].type is required", i))
		} else {
			validTypes := map[string]bool{
				"file": true, "text": true, "number": true,
				"boolean": true, "select": true, "column-selector": true,
			}
			if !validTypes[input.Type] {
				errors = append(errors, fmt.Sprintf("inputs[%d].type is invalid: %s", i, input.Type))
			}

			// Type-specific validation
			if input.Type == "select" && len(input.Options) == 0 {
				errors = append(errors, fmt.Sprintf("inputs[%d]: select type requires options", i))
			}

			if input.Type == "column-selector" && input.SourceFile == "" {
				errors = append(errors, fmt.Sprintf("inputs[%d]: column-selector requires sourceFile", i))
			}
		}

		// Validate visibility conditions
		if input.VisibleWhen != nil {
			if !inputNames[input.VisibleWhen.Field] {
				warnings = append(warnings, fmt.Sprintf("inputs[%d]: visibleWhen references non-existent field: %s", i, input.VisibleWhen.Field))
			}
		}
	}

	// Validate execution
	if plugin.Execution.OutputDir == "" {
		errors = append(errors, "execution.outputDir is required")
	}

	// Print warnings
	for _, warning := range warnings {
		printWarning(warning)
	}

	return len(errors) == 0, errors
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: plugin-validator <plugin.yaml>")
		fmt.Println("       plugin-validator <plugins-directory>")
		os.Exit(1)
	}

	path := os.Args[1]

	// Check if it's a directory or file
	info, err := os.Stat(path)
	if err != nil {
		printError(fmt.Sprintf("Path not found: %s", path))
		os.Exit(1)
	}

	var pluginFiles []string

	if info.IsDir() {
		// Find all plugin.yaml files in directory
		err := filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !fileInfo.IsDir() && (filepath.Base(filePath) == "plugin.yaml" || filepath.Base(filePath) == "plugin.yml") {
				pluginFiles = append(pluginFiles, filePath)
			}
			return nil
		})

		if err != nil {
			printError(fmt.Sprintf("Error scanning directory: %v", err))
			os.Exit(1)
		}

		if len(pluginFiles) == 0 {
			printError(fmt.Sprintf("No plugin.yaml files found in %s", path))
			os.Exit(1)
		}
	} else {
		pluginFiles = []string{path}
	}

	fmt.Println("CauldronGO Plugin Validator")
	fmt.Println("==========================================")
	fmt.Printf("Found %d plugin(s) to validate\n\n", len(pluginFiles))

	totalErrors := 0
	totalWarnings := 0

	for _, pluginFile := range pluginFiles {
		fmt.Printf("Validating: %s\n", pluginFile)
		fmt.Println("==========================================")

		valid, errors := validatePlugin(pluginFile)

		if valid {
			printSuccess("Plugin is valid")
		} else {
			for _, err := range errors {
				printError(err)
				totalErrors++
			}
		}

		fmt.Println()
	}

	// Summary
	fmt.Println("==========================================")
	fmt.Println("Validation Summary")
	fmt.Println("==========================================")
	fmt.Printf("Plugins validated: %d\n", len(pluginFiles))
	fmt.Printf("Errors: %d\n", totalErrors)
	fmt.Printf("Warnings: %d\n", totalWarnings)

	if totalErrors > 0 {
		fmt.Println()
		printError(fmt.Sprintf("Validation FAILED with %d error(s)", totalErrors))
		os.Exit(1)
	} else if totalWarnings > 0 {
		fmt.Println()
		printWarning(fmt.Sprintf("Validation passed with %d warning(s)", totalWarnings))
		os.Exit(0)
	} else {
		fmt.Println()
		printSuccess("All plugins validated successfully!")
		os.Exit(0)
	}
}
