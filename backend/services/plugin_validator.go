package services

import (
	"fmt"
	"regexp"

	"github.com/noatgnu/cauldron-go/backend/models"
)

type PluginValidator struct{}

func NewPluginValidator() *PluginValidator {
	return &PluginValidator{}
}

func (v *PluginValidator) ValidateDefinition(def *models.PluginDefinition) (bool, []string) {
	var errors []string

	if def.Plugin.ID == "" {
		errors = append(errors, "Plugin ID is required")
	} else {
		matched, _ := regexp.MatchString(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`, def.Plugin.ID)
		if !matched {
			errors = append(errors, "Plugin ID must be kebab-case (lowercase letters, numbers, and hyphens only)")
		}
	}

	if def.Plugin.Name == "" {
		errors = append(errors, "Plugin name is required")
	}

	if def.Plugin.Description == "" {
		errors = append(errors, "Plugin description is required")
	} else if len(def.Plugin.Description) < 10 {
		errors = append(errors, "Plugin description must be at least 10 characters")
	}

	if def.Plugin.Version == "" {
		errors = append(errors, "Plugin version is required")
	} else {
		matched, _ := regexp.MatchString(`^\d+\.\d+\.\d+$`, def.Plugin.Version)
		if !matched {
			errors = append(errors, "Plugin version must be in semantic versioning format (e.g., 1.0.0)")
		}
	}

	if def.Plugin.Category == "" {
		errors = append(errors, "Plugin category is required")
	}

	if def.Runtime.Type == "" {
		errors = append(errors, "Runtime type is required")
	}

	if def.Runtime.Script == "" {
		errors = append(errors, "Runtime script is required")
	}

	if len(def.Inputs) == 0 {
		errors = append(errors, "At least one input is required")
	}

	inputNames := make(map[string]bool)
	for i, input := range def.Inputs {
		if input.Name == "" {
			errors = append(errors, fmt.Sprintf("Input #%d: name is required", i+1))
		} else {
			if inputNames[input.Name] {
				errors = append(errors, fmt.Sprintf("Input name '%s' is duplicated", input.Name))
			}
			inputNames[input.Name] = true

			matched, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9_]*$`, input.Name)
			if !matched {
				errors = append(errors, fmt.Sprintf("Input '%s': name must start with a letter and contain only letters, numbers, and underscores", input.Name))
			}
		}

		if input.Label == "" {
			errors = append(errors, fmt.Sprintf("Input '%s': label is required", input.Name))
		}

		if input.Type == "" {
			errors = append(errors, fmt.Sprintf("Input '%s': type is required", input.Name))
		}
	}

	outputNames := make(map[string]bool)
	for i, output := range def.Outputs {
		if output.Name == "" {
			errors = append(errors, fmt.Sprintf("Output #%d: name is required", i+1))
		} else {
			if outputNames[output.Name] {
				errors = append(errors, fmt.Sprintf("Output name '%s' is duplicated", output.Name))
			}
			outputNames[output.Name] = true
		}

		if output.Path == "" {
			errors = append(errors, fmt.Sprintf("Output '%s': path is required", output.Name))
		}

		if output.Type == "" {
			errors = append(errors, fmt.Sprintf("Output '%s': type is required", output.Name))
		}
	}

	for i, plot := range def.Plots {
		if plot.ID == "" {
			errors = append(errors, fmt.Sprintf("Plot #%d: id is required", i+1))
		}

		if plot.DataSource != "" {
			if !outputNames[plot.DataSource] {
				errors = append(errors, fmt.Sprintf("Plot '%s': dataSource '%s' does not match any output", plot.ID, plot.DataSource))
			}
		}
	}

	return len(errors) == 0, errors
}
