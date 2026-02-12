package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/noatgnu/cauldron-go/internal/generator"
	"github.com/noatgnu/cauldron-go/internal/parser"
	"github.com/noatgnu/cauldron-go/internal/templates"
)

func main() {
	pluginPath := flag.String("plugin", "", "Path to plugin.yaml")
	pluginsDir := flag.String("plugins-dir", "", "Path to directory containing multiple plugins")
	outputDir := flag.String("output", "./nextflow-pipeline", "Output directory")
	registry := flag.String("container-registry", "ghcr.io/noatgnu", "Container registry for images")
	flag.Parse()

	if *pluginPath == "" && *pluginsDir == "" {
		fmt.Println("Usage: plugin-to-nextflow --plugin <path/to/plugin.yaml> [--output <output/dir>] [--container-registry <registry>]")
		fmt.Println("       plugin-to-nextflow --plugins-dir <path/to/plugins> [--output <output/dir>]")
		os.Exit(1)
	}

	if *pluginPath != "" {
		if err := convertPlugin(*pluginPath, *outputDir, *registry); err != nil {
			log.Fatalf("Error converting plugin: %v", err)
		}
	} else if *pluginsDir != "" {
		if err := convertPluginsBatch(*pluginsDir, *outputDir, *registry); err != nil {
			log.Fatalf("Error converting plugins batch: %v", err)
		}
	}
}

func convertPluginsBatch(dir, outDir, registry string) error {
	var pluginFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (filepath.Base(path) == "plugin.yaml" || filepath.Base(path) == "plugin.yml") {
			pluginFiles = append(pluginFiles, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(pluginFiles) == 0 {
		return fmt.Errorf("no plugin.yaml files found in %s", dir)
	}

	fmt.Printf("Found %d plugins for batch conversion\n", len(pluginFiles))

	for _, path := range pluginFiles {
		if err := convertPlugin(path, outDir, registry); err != nil {
			fmt.Printf("[WARNING] Failed to convert plugin at %s: %v\n", path, err)
		}
	}

	return nil
}

func convertPlugin(path, outDir, registry string) error {
	definition, err := parser.ParsePlugin(path)
	if err != nil {
		return err
	}

	fmt.Printf("Converting plugin: %s (%s)\n", definition.Plugin.Name, definition.Plugin.ID)

	// Ensure output directory exists
	moduleDir := filepath.Join(outDir, "modules", "local", definition.Plugin.ID)
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return err
	}

	// Generate Process
	procTmpl, _ := templates.GetTemplate("process.nf.tmpl")
	procContent, err := generator.GenerateProcess(definition, procTmpl, registry)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "main.nf"), []byte(procContent), 0644); err != nil {
		return err
	}

	// Generate Workflow (Main NF)
	wfTmpl, _ := templates.GetTemplate("main.nf.tmpl")
	wfContent, err := generator.GenerateWorkflow(definition, wfTmpl)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "main.nf"), []byte(wfContent), 0644); err != nil {
		return err
	}

	// Generate Config
	cfgTmpl, _ := templates.GetTemplate("nextflow.config.tmpl")
	cfgContent, err := generator.GenerateConfig(definition, cfgTmpl)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "nextflow.config"), []byte(cfgContent), 0644); err != nil {
		return err
	}

	// Generate Nextflow Schema
	schemaContent, err := generator.GenerateSchema(definition)
	if err == nil {
		os.WriteFile(filepath.Join(outDir, "nextflow_schema.json"), []byte(schemaContent), 0644)
	}

	// Generate Dockerfile
	containerDir := filepath.Join(outDir, "containers")
	os.MkdirAll(containerDir, 0755)

	if definition.Runtime.IsDockerRuntime() && definition.Runtime.Docker != nil && definition.Runtime.Docker.Dockerfile != "" {
		// Copy custom Dockerfile
		pluginDir := filepath.Dir(path)
		customDockerPath := filepath.Join(pluginDir, definition.Runtime.Docker.Dockerfile)
		dockerContent, err := os.ReadFile(customDockerPath)
		if err == nil {
			os.WriteFile(filepath.Join(containerDir, "Dockerfile"), dockerContent, 0644)
		}
	} else {
		dockerTmpl, _ := templates.GetTemplate("Dockerfile.tmpl")
		dockerContent, err := generator.GenerateContainer(definition, dockerTmpl)
		if err == nil && dockerContent != "" {
			os.WriteFile(filepath.Join(containerDir, "Dockerfile"), []byte(dockerContent), 0644)
		}
	}

	// Generate README
	if readmeTmpl, err := templates.GetTemplate("README.md.tmpl"); err == nil {
		if readmeContent, err := generator.GenerateREADME(definition, readmeTmpl); err == nil {
			os.WriteFile(filepath.Join(outDir, "README.md"), []byte(readmeContent), 0644)
		}
	}

	// Generate GitHub Action for the plugin
	if actionTmpl, err := templates.GetTemplate("plugin-github-action.yml.tmpl"); err == nil {
		if actionContent, err := generator.GenerateGithubAction(actionTmpl); err == nil {
			githubDir := filepath.Join(outDir, ".github", "workflows")
			os.MkdirAll(githubDir, 0755)
			os.WriteFile(filepath.Join(githubDir, "nextflow-export.yml"), []byte(actionContent), 0644)
		}
	}

	fmt.Printf("Successfully generated Nextflow pipeline in: %s\n", outDir)
	return nil
}
