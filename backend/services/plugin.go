package services

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/noatgnu/cauldron-go/backend/models"
	"gopkg.in/yaml.v3"
)

type PluginService struct {
	pluginsDir string
	plugins    map[string]*models.Plugin
}

func NewPluginService() *PluginService {
	service := &PluginService{
		plugins: make(map[string]*models.Plugin),
	}

	service.pluginsDir = service.getPluginsDirectory()

	if err := os.MkdirAll(service.pluginsDir, 0755); err != nil {
		log.Printf("[PluginService] Failed to create plugins directory: %v", err)
	}

	if err := service.loadPlugins(); err != nil {
		log.Printf("[PluginService] Failed to load plugins: %v", err)
	}

	return service
}

func (p *PluginService) getPluginsDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("[PluginService] Failed to get home directory: %v", err)
		return ""
	}

	var appFolder string
	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			appFolder = filepath.Join(localAppData, "cauldron", "plugins")
		} else {
			appFolder = filepath.Join(homeDir, "AppData", "Local", "cauldron", "plugins")
		}
	case "darwin":
		appFolder = filepath.Join(homeDir, "Library", "Application Support", "cauldron", "plugins")
	case "linux":
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome != "" {
			appFolder = filepath.Join(xdgDataHome, "cauldron", "plugins")
		} else {
			appFolder = filepath.Join(homeDir, ".local", "share", "cauldron", "plugins")
		}
	default:
		appFolder = filepath.Join(homeDir, ".cauldron", "plugins")
	}

	return appFolder
}

func (p *PluginService) loadPlugins() error {
	log.Printf("[PluginService] Loading plugins from: %s", p.pluginsDir)

	if _, err := os.Stat(p.pluginsDir); os.IsNotExist(err) {
		log.Printf("[PluginService] Plugins directory does not exist, creating: %s", p.pluginsDir)
		return os.MkdirAll(p.pluginsDir, 0755)
	}

	entries, err := ioutil.ReadDir(p.pluginsDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(p.pluginsDir, entry.Name())
		configPath := filepath.Join(pluginPath, "plugin.yaml")

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			configPath = filepath.Join(pluginPath, "plugin.yml")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				log.Printf("[PluginService] Skipping %s: no plugin.yaml or plugin.yml found", entry.Name())
				continue
			}
		}

		plugin, err := p.loadPlugin(pluginPath, configPath)
		if err != nil {
			log.Printf("[PluginService] Failed to load plugin %s: %v", entry.Name(), err)
			continue
		}

		p.plugins[plugin.ID] = plugin
		log.Printf("[PluginService] Loaded plugin: %s (ID: %s)", plugin.Config.Name, plugin.ID)
	}

	log.Printf("[PluginService] Loaded %d plugins", len(p.plugins))
	return nil
}

func (p *PluginService) loadPlugin(pluginPath, configPath string) (*models.Plugin, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config models.PluginConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.Name == "" {
		return nil, fmt.Errorf("plugin name is required")
	}
	if config.Script.Path == "" {
		return nil, fmt.Errorf("script path is required")
	}

	scriptPath := filepath.Join(pluginPath, config.Script.Path)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("script file not found: %s", scriptPath)
	}

	pluginID := fmt.Sprintf("%x", md5.Sum([]byte(pluginPath)))

	return &models.Plugin{
		ID:         pluginID,
		Config:     config,
		FolderPath: pluginPath,
		ScriptPath: scriptPath,
	}, nil
}

func (p *PluginService) GetPlugins() []*models.Plugin {
	plugins := make([]*models.Plugin, 0, len(p.plugins))
	for _, plugin := range p.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

func (p *PluginService) GetPlugin(id string) (*models.Plugin, error) {
	plugin, ok := p.plugins[id]
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", id)
	}
	return plugin, nil
}

func (p *PluginService) ReloadPlugins() error {
	p.plugins = make(map[string]*models.Plugin)
	return p.loadPlugins()
}

func (p *PluginService) GetPluginsDirectory() string {
	return p.pluginsDir
}

func (p *PluginService) CreateSamplePlugin() error {
	samplePath := filepath.Join(p.pluginsDir, "sample-analysis")
	if err := os.MkdirAll(samplePath, 0755); err != nil {
		return fmt.Errorf("failed to create sample plugin directory: %w", err)
	}

	configContent := `name: "Sample Custom Analysis"
description: "A sample plugin demonstrating custom workflow capabilities"
version: "1.0.0"
author: "Cauldron"
runtime: "python"
script:
  path: "analysis.py"
inputs:
  - name: "input_file"
    label: "Input Data File"
    type: "file"
    required: true
    description: "The data file to analyze"
  - name: "threshold"
    label: "Threshold Value"
    type: "number"
    required: false
    default: 0.05
    description: "P-value threshold for significance"
  - name: "method"
    label: "Analysis Method"
    type: "select"
    options: ["method_a", "method_b", "method_c"]
    required: true
    description: "Statistical method to use"
  - name: "normalize"
    label: "Normalize Data"
    type: "boolean"
    required: false
    default: true
    description: "Apply normalization before analysis"
outputs:
  - name: "results.txt"
    description: "Analysis results in tab-delimited format"
  - name: "plot.pdf"
    description: "Visualization of results"
`

	scriptContent := `#!/usr/bin/env python3
"""
Sample Custom Analysis Plugin
This is a template for creating custom analysis plugins
"""
import sys
import argparse
import pandas as pd

def main():
    parser = argparse.ArgumentParser(description='Sample custom analysis')
    parser.add_argument('--input_file', required=True, help='Input data file')
    parser.add_argument('--threshold', type=float, default=0.05, help='Threshold value')
    parser.add_argument('--method', required=True, help='Analysis method')
    parser.add_argument('--normalize', type=bool, default=True, help='Normalize data')
    parser.add_argument('--output', required=True, help='Output directory')

    args = parser.parse_args()

    print(f"Running sample analysis...")
    print(f"Input file: {args.input_file}")
    print(f"Threshold: {args.threshold}")
    print(f"Method: {args.method}")
    print(f"Normalize: {args.normalize}")
    print(f"Output: {args.output}")

    print("Analysis completed successfully!")
    return 0

if __name__ == '__main__':
    sys.exit(main())
`

	configPath := filepath.Join(samplePath, "plugin.yaml")
	if err := ioutil.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write sample config: %w", err)
	}

	scriptPath := filepath.Join(samplePath, "analysis.py")
	if err := ioutil.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write sample script: %w", err)
	}

	return p.ReloadPlugins()
}
