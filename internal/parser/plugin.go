package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/noatgnu/cauldron-go/backend/models"
	"gopkg.in/yaml.v3"
)

// ParsePlugin parses a plugin.yaml file into a PluginDefinition.
func ParsePlugin(path string) (*models.PluginDefinition, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat plugin path: %w", err)
	}

	pluginFilePath := path
	if info.IsDir() {
		// Try plugin.yaml or plugin.yml in the directory
		yamlPath := filepath.Join(path, "plugin.yaml")
		ymlPath := filepath.Join(path, "plugin.yml")

		if _, err := os.Stat(yamlPath); err == nil {
			pluginFilePath = yamlPath
		} else if _, err := os.Stat(ymlPath); err == nil {
			pluginFilePath = ymlPath
		} else {
			return nil, fmt.Errorf("no plugin.yaml or plugin.yml found in directory: %s", path)
		}
	}

	data, err := os.ReadFile(pluginFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin file: %w", err)
	}

	var definition models.PluginDefinition
	if err := yaml.Unmarshal(data, &definition); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plugin YAML: %w", err)
	}

	return &definition, nil
}
