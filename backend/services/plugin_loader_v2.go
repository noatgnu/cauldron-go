package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
	"gopkg.in/yaml.v3"
)

type PluginLoaderV2 struct {
	pluginsDir   string
	plugins      map[uint]*models.PluginV2
	db           *DatabaseService
	imageBuilder *DockerImageBuilder
}

func NewPluginLoaderV2(pluginsDir string, db *DatabaseService, imageBuilder *DockerImageBuilder) *PluginLoaderV2 {
	if pluginsDir == "" {
		execPath, _ := os.Executable()
		pluginsDir = filepath.Join(filepath.Dir(execPath), "plugins")
	}

	return &PluginLoaderV2{
		pluginsDir:   pluginsDir,
		plugins:      make(map[uint]*models.PluginV2),
		db:           db,
		imageBuilder: imageBuilder,
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

		registry, err := l.getOrCreateRegistry(plugin, pluginPath)
		if err != nil {
			log.Printf("[PluginLoader] Failed to register plugin from %s: %v", pluginPath, err)
			continue
		}

		plugin.ID = registry.ID
		plugin.InstallSource = registry.InstallSource
		plugin.CommitHash = registry.CommitHash
		plugin.Repository = registry.Repository
		plugin.Enabled = registry.Enabled
		l.plugins[registry.ID] = plugin
		loadedCount++
		log.Printf("[PluginLoader] Loaded plugin: %s [ID:%d] (%s) from %s [%s]",
			plugin.Definition.Plugin.Name,
			registry.ID,
			plugin.Definition.Plugin.ID,
			pluginPath,
			registry.InstallSource)
	}

	log.Printf("[PluginLoader] Successfully loaded %d plugins", loadedCount)
	return nil
}

func (l *PluginLoaderV2) loadPlugin(pluginDir string) (*models.PluginV2, error) {
	configPath := l.getPlatformSpecificConfig(pluginDir)

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

	if err := l.loadOptionsFromFiles(pluginDir, &definition); err != nil {
		return nil, fmt.Errorf("failed to load options from files: %w", err)
	}

	if err := l.loadRequirementsFromFiles(pluginDir, &definition); err != nil {
		return nil, fmt.Errorf("failed to load requirements from files: %w", err)
	}

	scriptPath := filepath.Join(pluginDir, definition.Runtime.GetEntrypoint())
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("entrypoint file not found: %s", scriptPath)
	}

	if definition.Runtime.IsDockerRuntime() {
		if definition.Runtime.Docker == nil {
			return nil, fmt.Errorf("docker runtime specified but no docker configuration found")
		}

		imageName := definition.Runtime.GetDockerImageName(definition.Plugin.ID)

		if definition.Runtime.Docker.Dockerfile != "" {
			dockerfilePath := filepath.Join(pluginDir, definition.Runtime.Docker.Dockerfile)
			if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
				return nil, fmt.Errorf("dockerfile not found: %s", dockerfilePath)
			}

			needsRebuild, err := l.imageBuilder.ShouldRebuildImage(
				definition.Plugin.ID,
				dockerfilePath,
			)
			if err != nil || needsRebuild {
				log.Printf("[PluginLoader] Building Docker image for plugin %s", definition.Plugin.ID)

				err := l.imageBuilder.BuildImage(
					pluginDir,
					definition.Runtime.Docker.Dockerfile,
					imageName,
					definition.Runtime.Docker.Platform,
					definition.Runtime.Docker.BuildArgs,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to build Docker image: %w", err)
				}

				dockerfileHash, _ := l.imageBuilder.GetDockerfileHash(dockerfilePath)
				l.imageBuilder.RecordBuiltImage(
					definition.Plugin.ID,
					imageName,
					dockerfileHash,
					definition.Runtime.Docker.Platform,
					definition.Runtime.Docker.BuildArgs,
				)
			}
		} else if definition.Runtime.Docker.Image != "" {
			imageName = definition.Runtime.Docker.Image
			if !l.imageBuilder.ImageExists(imageName) {
				log.Printf("[PluginLoader] Pulling Docker image: %s", imageName)
				err := l.imageBuilder.PullImage(imageName)
				if err != nil {
					return nil, fmt.Errorf("failed to pull Docker image: %w", err)
				}
			}
		}
	}

	plugin := &models.PluginV2{
		Definition: definition,
		FolderPath: pluginDir,
		ScriptPath: scriptPath,
	}

	return plugin, nil
}

func (l *PluginLoaderV2) getPlatformSpecificConfig(pluginDir string) string {
	platformConfigs := []string{
		fmt.Sprintf("plugin.%s.yaml", goruntime.GOOS),
		fmt.Sprintf("plugin.%s.yml", goruntime.GOOS),
		"plugin.yaml",
		"plugin.yml",
	}

	for _, configName := range platformConfigs {
		configPath := filepath.Join(pluginDir, configName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	return filepath.Join(pluginDir, "plugin.yaml")
}

func (l *PluginLoaderV2) validateDefinition(def *models.PluginDefinition) error {
	if def.Plugin.ID == "" {
		return fmt.Errorf("plugin ID is required")
	}

	if def.Plugin.Name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if def.Runtime.Environments == nil || len(def.Runtime.Environments) == 0 {
		return fmt.Errorf("runtime environments is required")
	}

	if def.Runtime.GetEntrypoint() == "" {
		return fmt.Errorf("runtime entrypoint (or deprecated runtime.script) is required")
	}

	validEnvironments := map[string]bool{
		"python": true,
		"r":      true,
		"direct": true,
		"julia":  true,
		"node":   true,
	}

	for _, env := range def.Runtime.Environments {
		if !validEnvironments[env] {
			return fmt.Errorf("invalid runtime environment: %s", env)
		}
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

func (l *PluginLoaderV2) getOrCreateRegistry(plugin *models.PluginV2, folderPath string) (*models.PluginRegistry, error) {
	var registry models.PluginRegistry

	result := l.db.GetDB().Where("folder_path = ?", folderPath).First(&registry)
	if result.Error == nil {
		return &registry, nil
	}

	installSource := "builtin"
	if plugin.Definition.Plugin.Repository != "" {
		installSource = "remote"
	}

	registry = models.PluginRegistry{
		PluginID:      plugin.Definition.Plugin.ID,
		Name:          plugin.Definition.Plugin.Name,
		Version:       plugin.Definition.Plugin.Version,
		Repository:    plugin.Definition.Plugin.Repository,
		FolderPath:    folderPath,
		InstallSource: installSource,
		Enabled:       true,
		InstalledAt:   time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := l.db.GetDB().Create(&registry).Error; err != nil {
		return nil, err
	}

	log.Printf("[PluginLoader] Created registry entry [ID:%d] for %s", registry.ID, plugin.Definition.Plugin.Name)
	return &registry, nil
}

func (l *PluginLoaderV2) GetPlugin(id uint) (*models.PluginV2, error) {
	plugin, exists := l.plugins[id]
	if !exists {
		return nil, fmt.Errorf("plugin not found with ID: %d", id)
	}
	return plugin, nil
}

func (l *PluginLoaderV2) SetPluginEnabled(id uint, enabled bool) error {
	plugin, exists := l.plugins[id]
	if !exists {
		return fmt.Errorf("plugin not found with ID: %d", id)
	}

	result := l.db.GetDB().Model(&models.PluginRegistry{}).Where("id = ?", id).Update("enabled", enabled)
	if result.Error != nil {
		return fmt.Errorf("failed to update plugin enabled state: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no plugin registry entry found with ID: %d", id)
	}

	plugin.Enabled = enabled
	log.Printf("[PluginLoader] Updated plugin %s [ID:%d] enabled state to: %v", plugin.Definition.Plugin.Name, id, enabled)
	return nil
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
	l.plugins = make(map[uint]*models.PluginV2)
	return l.LoadPlugins()
}

func (l *PluginLoaderV2) GetPluginsDirectory() string {
	return l.pluginsDir
}

func (l *PluginLoaderV2) GetPluginByStringID(pluginID string) (*models.PluginV2, error) {
	for _, plugin := range l.plugins {
		if plugin.Definition.Plugin.ID == pluginID {
			return plugin, nil
		}
	}
	return nil, fmt.Errorf("plugin not found with ID: %s", pluginID)
}

func (l *PluginLoaderV2) loadRequirementsFromFiles(pluginDir string, def *models.PluginDefinition) error {
	if def.Execution.Requirements.PythonRequirementsFile != "" {
		requirementsPath := filepath.Join(pluginDir, def.Execution.Requirements.PythonRequirementsFile)
		packages, err := l.loadPackagesFromTextFile(requirementsPath)
		if err != nil {
			log.Printf("[PluginLoader] Warning: failed to load Python requirements from %s: %v", def.Execution.Requirements.PythonRequirementsFile, err)
		} else {
			def.Execution.Requirements.Packages = append(def.Execution.Requirements.Packages, packages...)
			log.Printf("[PluginLoader] Loaded %d Python packages from %s", len(packages), def.Execution.Requirements.PythonRequirementsFile)
		}
	}

	if def.Execution.Requirements.RPackagesFile != "" {
		packagesPath := filepath.Join(pluginDir, def.Execution.Requirements.RPackagesFile)
		packages, err := l.loadPackagesFromTextFile(packagesPath)
		if err != nil {
			log.Printf("[PluginLoader] Warning: failed to load R packages from %s: %v", def.Execution.Requirements.RPackagesFile, err)
		} else {
			def.Execution.Requirements.Packages = append(def.Execution.Requirements.Packages, packages...)
			log.Printf("[PluginLoader] Loaded %d R packages from %s", len(packages), def.Execution.Requirements.RPackagesFile)
		}
	}

	return nil
}

func (l *PluginLoaderV2) loadPackagesFromTextFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var packages []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			packages = append(packages, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return packages, nil
}

func (l *PluginLoaderV2) loadOptionsFromFiles(pluginDir string, def *models.PluginDefinition) error {
	for i := range def.Inputs {
		input := &def.Inputs[i]

		if input.OptionsFromFile != "" {
			optionsPath := filepath.Join(pluginDir, input.OptionsFromFile)
			options, err := l.loadOptionsFromTextFile(optionsPath)
			if err != nil {
				return fmt.Errorf("failed to load options from %s: %w", input.OptionsFromFile, err)
			}
			input.Options = options
		}

		if input.GroupsFromFile != "" {
			groupsPath := filepath.Join(pluginDir, input.GroupsFromFile)
			groups, err := l.loadGroupsFromJSONFile(groupsPath)
			if err != nil {
				return fmt.Errorf("failed to load groups from %s: %w", input.GroupsFromFile, err)
			}
			input.Groups = groups
		}
	}

	return nil
}

func (l *PluginLoaderV2) loadOptionsFromTextFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var options []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			options = append(options, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return options, nil
}

func (l *PluginLoaderV2) loadGroupsFromJSONFile(filePath string) ([]models.FieldGroup, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var groupsMap map[string][]models.FieldOption
	if err := json.Unmarshal(data, &groupsMap); err != nil {
		return nil, err
	}

	groups := make([]models.FieldGroup, 0, len(groupsMap))
	for name, options := range groupsMap {
		groups = append(groups, models.FieldGroup{
			Name:    name,
			Options: options,
		})
	}

	return groups, nil
}
