package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/noatgnu/cauldron-go/internal/generator"
)

func main() {
	pluginPath := flag.String("plugin", "", "Path to plugin.yaml")
	outputDir := flag.String("output", "./spa-output", "Output directory")
	pyodideVersion := flag.String("pyodide-version", "0.29.3", "Pyodide version to use")
	skipCheck := flag.Bool("skip-check", false, "Skip Pyodide compatibility check")
	noBuild := flag.Bool("no-build", false, "Skip npm install and build")
	generateWorkflow := flag.Bool("generate-workflow", false, "Generate GitHub Actions workflow for plugin repo")
	flag.Parse()

	if *pluginPath == "" {
		fmt.Println("Usage: plugin-to-spa --plugin <path/to/plugin.yaml> [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  --output <dir>          Output directory (default: ./spa-output)")
		fmt.Println("  --pyodide-version <ver> Pyodide version (default: 0.29.3)")
		fmt.Println("  --skip-check            Skip Pyodide compatibility check")
		fmt.Println("  --no-build              Skip npm install and build")
		fmt.Println("  --generate-workflow     Generate GitHub Actions workflow for deployment")
		os.Exit(1)
	}

	if *generateWorkflow {
		pluginDir := getPluginDir(*pluginPath)
		if err := generator.GeneratePluginWorkflow(pluginDir, *pyodideVersion); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating workflow: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ GitHub Actions workflow generated: .github/workflows/deploy-spa.yml")
		return
	}

	fmt.Printf("Generating SPA from plugin: %s\n", *pluginPath)
	fmt.Printf("Output directory: %s\n", *outputDir)

	config := generator.SPAConfig{
		PluginPath:     *pluginPath,
		OutputDir:      *outputDir,
		PyodideVersion: *pyodideVersion,
		SkipCheck:      *skipCheck,
		NoBuild:        *noBuild,
	}

	gen := generator.NewSPAGenerator(config)

	if err := gen.Generate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating SPA: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ SPA generated successfully!")
	fmt.Println("\nNext steps:")
	fmt.Printf("  cd %s\n", *outputDir)
	fmt.Println("  npm install")
	fmt.Println("  npm start")
}

func getPluginDir(pluginPath string) string {
	for i := len(pluginPath) - 1; i >= 0; i-- {
		if pluginPath[i] == '/' || pluginPath[i] == '\\' {
			return pluginPath[:i]
		}
	}
	return "."
}
