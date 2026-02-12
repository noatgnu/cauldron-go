package parser

import (
	"fmt"
	"os"

	"github.com/noatgnu/cauldron-go/backend/models"
	"gopkg.in/yaml.v3"
)

// ParsePlugin parses a plugin.yaml file into a PluginDefinition.
func ParsePlugin(path string) (*models.PluginDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin file: %w", err)
	}

	var definition models.PluginDefinition
	if err := yaml.Unmarshal(data, &definition); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plugin YAML: %w", err)
	}

	return &definition, nil
}
