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
	outputDir := flag.String("output", "./slivka-services", "Output directory")
	generateGithubAction := flag.Bool("github-action", false, "Generate GitHub Action workflow for testing")
	flag.Parse()

	if *pluginPath == "" && *pluginsDir == "" {
		fmt.Println("Usage: plugin-to-slivka --plugin <path/to/plugin.yaml> [--output <output/dir>] [--github-action]")
		fmt.Println("       plugin-to-slivka --plugins-dir <path/to/plugins> [--output <output/dir>] [--github-action]")
		os.Exit(1)
	}

	if *pluginPath != "" {
		if err := convertPlugin(*pluginPath, *outputDir, *generateGithubAction); err != nil {
			log.Fatalf("Error converting plugin: %v", err)
		}
	} else if *pluginsDir != "" {
		if err := convertPluginsBatch(*pluginsDir, *outputDir, *generateGithubAction); err != nil {
			log.Fatalf("Error converting plugins batch: %v", err)
		}
	}
}

func convertPluginsBatch(dir, outDir string, generateGithubAction bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var pluginFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			pluginPath := filepath.Join(dir, entry.Name(), "plugin.yaml")
			if _, err := os.Stat(pluginPath); err == nil {
				pluginFiles = append(pluginFiles, pluginPath)
			}
			pluginPath = filepath.Join(dir, entry.Name(), "plugin.yml")
			if _, err := os.Stat(pluginPath); err == nil {
				pluginFiles = append(pluginFiles, pluginPath)
			}
		}
	}

	if len(pluginFiles) == 0 {
		return fmt.Errorf("no plugin.yaml files found in %s", dir)
	}

	fmt.Printf("Found %d plugins for batch conversion\n", len(pluginFiles))

	for _, path := range pluginFiles {
		if err := convertPlugin(path, outDir, generateGithubAction); err != nil {
			fmt.Printf("[WARNING] Failed to convert plugin at %s: %v\n", path, err)
		}
	}

	return nil
}

func convertPlugin(path, outDir string, generateGithubAction bool) error {
	definition, err := parser.ParsePlugin(path)
	if err != nil {
		return err
	}

	fmt.Printf("Converting plugin: %s (%s)\n", definition.Plugin.Name, definition.Plugin.ID)

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	slivkaTmpl, err := templates.GetTemplate("slivka-service.yml.tmpl")
	if err != nil {
		return fmt.Errorf("failed to get slivka template: %w", err)
	}

	slivkaContent, err := generator.GenerateSlivka(definition, slivkaTmpl)
	if err != nil {
		return err
	}

	outputFile := filepath.Join(outDir, definition.Plugin.ID+".service.yaml")
	if err := os.WriteFile(outputFile, []byte(slivkaContent), 0644); err != nil {
		return err
	}

	fmt.Printf("Successfully generated Slivka service: %s\n", outputFile)

	if generateGithubAction {
		githubActionTmpl, err := templates.GetTemplate("slivka-github-action.yml.tmpl")
		if err != nil {
			return fmt.Errorf("failed to get slivka github-action template: %w", err)
		}

		githubActionContent, err := generator.GenerateSlivkaGithubAction(definition, githubActionTmpl)
		if err != nil {
			return err
		}

		pluginDir := filepath.Dir(path)
		githubDir := filepath.Join(pluginDir, ".github", "workflows")
		if err := os.MkdirAll(githubDir, 0755); err != nil {
			return err
		}

		githubActionFile := filepath.Join(githubDir, "slivka-test.yml")
		if err := os.WriteFile(githubActionFile, []byte(githubActionContent), 0644); err != nil {
			return err
		}

		fmt.Printf("Successfully generated GitHub Action: %s\n", githubActionFile)
	}

	return nil
}
