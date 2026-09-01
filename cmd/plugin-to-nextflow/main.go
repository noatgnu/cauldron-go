package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/internal/generator"
	"github.com/noatgnu/cauldron-go/internal/parser"
	"github.com/noatgnu/cauldron-go/internal/templates"
)

func main() {
	pluginPath := flag.String("plugin", "", "Path to plugin.yaml")
	pluginsDir := flag.String("plugins-dir", "", "Path to directory containing multiple plugins")
	outputDir := flag.String("output", "./nextflow-pipeline", "Output directory")
	generateGithubAction := flag.Bool("github-action", false, "Generate GitHub Action workflow for CI/CD")
	flag.Parse()

	if *pluginPath == "" && *pluginsDir == "" {
		fmt.Println("Usage: plugin-to-nextflow --plugin <path/to/plugin.yaml> [--output <output/dir>] [--github-action]")
		fmt.Println("       plugin-to-nextflow --plugins-dir <path/to/plugins> [--output <output/dir>] [--github-action]")
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
		definition, err := parser.ParsePlugin(path)
		if err != nil {
			fmt.Printf("[WARNING] Failed to parse plugin at %s: %v\n", path, err)
			continue
		}

		// Pipeline-level files (main.nf, config, README) are single-plugin-scoped, so each plugin needs its own subdirectory.
		pluginOutDir := filepath.Join(outDir, definition.Plugin.ID)
		if err := convertPluginDefinition(definition, path, pluginOutDir, generateGithubAction); err != nil {
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
	return convertPluginDefinition(definition, path, outDir, generateGithubAction)
}

func convertPluginDefinition(definition *models.PluginDefinition, path, outDir string, generateGithubAction bool) error {
	fmt.Printf("Converting plugin: %s (%s)\n", definition.Plugin.Name, definition.Plugin.ID)

	moduleDir := filepath.Join(outDir, "modules", "local", definition.Plugin.ID)
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return err
	}

	procTmpl, _ := templates.GetTemplate("process.nf.tmpl")
	procContent, err := generator.GenerateProcess(definition, procTmpl)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "main.nf"), []byte(procContent), 0644); err != nil {
		return err
	}

	wfTmpl, _ := templates.GetTemplate("main.nf.tmpl")
	wfContent, err := generator.GenerateWorkflow(definition, wfTmpl)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "main.nf"), []byte(wfContent), 0644); err != nil {
		return err
	}

	cfgTmpl, _ := templates.GetTemplate("nextflow.config.tmpl")
	cfgContent, err := generator.GenerateConfig(definition, cfgTmpl)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "nextflow.config"), []byte(cfgContent), 0644); err != nil {
		return err
	}

	schemaContent, err := generator.GenerateSchema(definition)
	if err != nil {
		fmt.Printf("[WARNING] Failed to generate nextflow_schema.json: %v\n", err)
	} else if err := os.WriteFile(filepath.Join(outDir, "nextflow_schema.json"), []byte(schemaContent), 0644); err != nil {
		fmt.Printf("[WARNING] Failed to write nextflow_schema.json: %v\n", err)
	}

	containerDir := filepath.Join(outDir, "containers")
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		return fmt.Errorf("failed to create containers directory: %w", err)
	}

	if definition.Runtime.IsDockerRuntime() && definition.Runtime.Docker != nil && definition.Runtime.Docker.Dockerfile != "" {
		pluginDir := filepath.Dir(path)
		customDockerPath := filepath.Join(pluginDir, definition.Runtime.Docker.Dockerfile)
		dockerContent, err := os.ReadFile(customDockerPath)
		if err != nil {
			fmt.Printf("[WARNING] Failed to read custom Dockerfile %s: %v\n", customDockerPath, err)
		} else if err := os.WriteFile(filepath.Join(containerDir, "Dockerfile"), dockerContent, 0644); err != nil {
			fmt.Printf("[WARNING] Failed to write Dockerfile: %v\n", err)
		}
	} else {
		dockerTmpl, _ := templates.GetTemplate("Dockerfile.tmpl")
		dockerContent, err := generator.GenerateContainer(definition, dockerTmpl)
		if err != nil {
			fmt.Printf("[WARNING] Failed to generate Dockerfile: %v\n", err)
		} else if dockerContent != "" {
			if err := os.WriteFile(filepath.Join(containerDir, "Dockerfile"), []byte(dockerContent), 0644); err != nil {
				fmt.Printf("[WARNING] Failed to write Dockerfile: %v\n", err)
			}
		}
	}

	readmeTmpl, err := templates.GetTemplate("README.md.tmpl")
	if err != nil {
		fmt.Printf("[WARNING] Failed to load README template: %v\n", err)
	} else if readmeContent, err := generator.GenerateREADME(definition, readmeTmpl); err != nil {
		fmt.Printf("[WARNING] Failed to generate README.md: %v\n", err)
	} else if err := os.WriteFile(filepath.Join(outDir, "README.md"), []byte(readmeContent), 0644); err != nil {
		fmt.Printf("[WARNING] Failed to write README.md: %v\n", err)
	}

	if generateGithubAction {
		actionTmpl, err := templates.GetTemplate("plugin-github-action.yml.tmpl")
		if err != nil {
			fmt.Printf("[WARNING] Failed to load GitHub Action template: %v\n", err)
		} else if actionContent, err := generator.GenerateGithubAction(definition, actionTmpl); err != nil {
			fmt.Printf("[WARNING] Failed to generate GitHub Action: %v\n", err)
		} else {
			githubDir := filepath.Join(outDir, ".github", "workflows")
			if err := os.MkdirAll(githubDir, 0755); err != nil {
				fmt.Printf("[WARNING] Failed to create .github/workflows directory: %v\n", err)
			} else if err := os.WriteFile(filepath.Join(githubDir, "nextflow-export.yml"), []byte(actionContent), 0644); err != nil {
				fmt.Printf("[WARNING] Failed to write GitHub Action: %v\n", err)
			} else {
				fmt.Printf("Successfully generated GitHub Action: %s\n", filepath.Join(githubDir, "nextflow-export.yml"))
			}
		}
	}

	fmt.Printf("Successfully generated Nextflow pipeline in: %s\n", outDir)
	return nil
}
