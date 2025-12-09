package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/noatgnu/cauldron-go/backend/models"
	"gopkg.in/yaml.v3"
)

type PluginLoaderV2 struct {
	pluginsDir string
	plugins    map[string]*models.PluginV2
}

func NewPluginLoaderV2(pluginsDir string) *PluginLoaderV2 {
	if pluginsDir == "" {
		execPath, _ := os.Executable()
		pluginsDir = filepath.Join(filepath.Dir(execPath), "plugins")
	}

	return &PluginLoaderV2{
		pluginsDir: pluginsDir,
		plugins:    make(map[string]*models.PluginV2),
	}
}

func (l *PluginLoaderV2) LoadPlugins() error {
	if _, err := os.Stat(l.pluginsDir); os.IsNotExist(err) {
		log.Printf("[PluginLoader] Plugins directory does not exist: %s", l.pluginsDir)
		if err := os.MkdirAll(l.pluginsDir, 0755); err != nil {
			return fmt.Errorf("failed to create plugins directory: %w", err)
		}
		log.Printf("[PluginLoader] Created plugins directory: %s", l.pluginsDir)
		return nil
	}

	entries, err := os.ReadDir(l.pluginsDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	loadedCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(l.pluginsDir, entry.Name())
		plugin, err := l.loadPlugin(pluginPath)
		if err != nil {
			log.Printf("[PluginLoader] Failed to load plugin from %s: %v", pluginPath, err)
			continue
		}

		l.plugins[plugin.Definition.Plugin.ID] = plugin
		loadedCount++
		log.Printf("[PluginLoader] Loaded plugin: %s (%s) from %s",
			plugin.Definition.Plugin.Name,
			plugin.Definition.Plugin.ID,
			pluginPath)
	}

	log.Printf("[PluginLoader] Successfully loaded %d plugins", loadedCount)
	return nil
}

func (l *PluginLoaderV2) loadPlugin(pluginDir string) (*models.PluginV2, error) {
	configPath := filepath.Join(pluginDir, "plugin.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = filepath.Join(pluginDir, "plugin.yml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("plugin.yaml not found in %s", pluginDir)
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin config: %w", err)
	}

	var definition models.PluginDefinition
	if err := yaml.Unmarshal(data, &definition); err != nil {
		return nil, fmt.Errorf("failed to parse plugin config: %w", err)
	}

	if err := l.validateDefinition(&definition); err != nil {
		return nil, fmt.Errorf("invalid plugin definition: %w", err)
	}

	scriptPath := filepath.Join(pluginDir, definition.Runtime.Script)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("script file not found: %s", scriptPath)
	}

	plugin := &models.PluginV2{
		Definition: definition,
		FolderPath: pluginDir,
		ScriptPath: scriptPath,
	}

	return plugin, nil
}

func (l *PluginLoaderV2) validateDefinition(def *models.PluginDefinition) error {
	if def.Plugin.ID == "" {
		return fmt.Errorf("plugin ID is required")
	}

	if def.Plugin.Name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if def.Runtime.Type == "" {
		return fmt.Errorf("runtime type is required")
	}

	if def.Runtime.Script == "" {
		return fmt.Errorf("runtime script is required")
	}

	validRuntimes := map[string]bool{
		"python":      true,
		"r":           true,
		"pythonWithR": true,
	}
	if !validRuntimes[def.Runtime.Type] {
		return fmt.Errorf("invalid runtime type: %s", def.Runtime.Type)
	}

	inputNames := make(map[string]bool)
	for _, input := range def.Inputs {
		if input.Name == "" {
			return fmt.Errorf("input name is required")
		}
		if inputNames[input.Name] {
			return fmt.Errorf("duplicate input name: %s", input.Name)
		}
		inputNames[input.Name] = true

		if input.Type == "" {
			return fmt.Errorf("input type is required for: %s", input.Name)
		}
	}

	outputNames := make(map[string]bool)
	for _, output := range def.Outputs {
		if output.Name == "" {
			return fmt.Errorf("output name is required")
		}
		if outputNames[output.Name] {
			return fmt.Errorf("duplicate output name: %s", output.Name)
		}
		outputNames[output.Name] = true

		if output.Path == "" {
			return fmt.Errorf("output path is required for: %s", output.Name)
		}
	}

	for _, plot := range def.Plots {
		if plot.ID == "" {
			return fmt.Errorf("plot ID is required")
		}
		if plot.DataSource == "" {
			return fmt.Errorf("plot data source is required for: %s", plot.ID)
		}
		if !outputNames[plot.DataSource] {
			return fmt.Errorf("plot %s references non-existent output: %s", plot.ID, plot.DataSource)
		}
	}

	return nil
}

func (l *PluginLoaderV2) GetPlugin(id string) (*models.PluginV2, error) {
	plugin, exists := l.plugins[id]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", id)
	}
	return plugin, nil
}

func (l *PluginLoaderV2) GetAllPlugins() []*models.PluginV2 {
	plugins := make([]*models.PluginV2, 0, len(l.plugins))
	for _, plugin := range l.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

func (l *PluginLoaderV2) GetPluginsByCategory(category models.PluginCategory) []*models.PluginV2 {
	plugins := make([]*models.PluginV2, 0)
	for _, plugin := range l.plugins {
		if plugin.Definition.Plugin.Category == category {
			plugins = append(plugins, plugin)
		}
	}
	return plugins
}

func (l *PluginLoaderV2) ReloadPlugins() error {
	l.plugins = make(map[string]*models.PluginV2)
	return l.LoadPlugins()
}

func (l *PluginLoaderV2) GetPluginsDirectory() string {
	return l.pluginsDir
}
