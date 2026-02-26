package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/noatgnu/cauldron-go/internal/generator"
	"github.com/noatgnu/cauldron-go/internal/parser"
)

func main() {
	pluginPath := flag.String("plugin", "", "Path to plugin.yaml")
	outputDir := flag.String("output", "./spa-output", "Output directory")
	pyodideVersion := flag.String("pyodide-version", "0.29.3", "Pyodide version to use")
	webrVersion := flag.String("webr-version", "0.5.0", "WebR version to use")
	skipCheck := flag.Bool("skip-check", false, "Skip compatibility check")
	noBuild := flag.Bool("no-build", false, "Skip npm install and build")
	generateWorkflow := flag.Bool("generate-workflow", false, "Generate GitHub Actions workflow for plugin repo")
	checkWebR := flag.Bool("check-webr", false, "Check WebR compatibility for R plugins")
	checkOnline := flag.Bool("check-online", false, "Check package availability online (slower)")
	examplesDir := flag.String("examples-dir", "", "Global examples directory (for shared example files)")
	spaTemplatePath := flag.String("spa-template", "", "Path to spa-template directory")
	sharedLibPath := flag.String("shared-lib-path", "", "Path to @cauldron/forms library")
	flag.Parse()

	if *pluginPath == "" {
		fmt.Println("Usage: plugin-to-spa --plugin <path/to/plugin.yaml> [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  --output <dir>          Output directory (default: ./spa-output)")
		fmt.Println("  --pyodide-version <ver> Pyodide version (default: 0.29.3)")
		fmt.Println("  --webr-version <ver>    WebR version (default: 0.5.0)")
		fmt.Println("  --skip-check            Skip compatibility check")
		fmt.Println("  --no-build              Skip npm install and build")
		fmt.Println("  --generate-workflow     Generate GitHub Actions workflow for deployment")
		fmt.Println("  --check-webr            Check WebR compatibility for R plugins")
		fmt.Println("  --check-online          Check package availability online (slower)")
		fmt.Println("  --examples-dir <dir>    Global examples directory (for shared example files)")
		fmt.Println("  --spa-template <dir>    Path to spa-template directory")
		fmt.Println("  --shared-lib-path <dir> Path to @cauldron/forms library")
		os.Exit(1)
	}

	pluginDir := getPluginDir(*pluginPath)

	if *checkWebR {
		definition, err := parser.ParsePlugin(*pluginPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing plugin: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Checking WebR compatibility for: %s\n", definition.Plugin.Name)
		fmt.Println("")

		compat := generator.CheckWebRCompatibility(definition, pluginDir)

		fmt.Println("Required R packages:")
		for _, pkg := range compat.Packages {
			fmt.Printf("  - %s\n", pkg)
		}
		fmt.Println("")

		if len(compat.BiocPackages) > 0 {
			fmt.Println("Bioconductor packages (may need R-Universe):")
			for _, pkg := range compat.BiocPackages {
				fmt.Printf("  - %s\n", pkg)
			}
			fmt.Println("")
		}

		if len(compat.MaybeSupport) > 0 {
			fmt.Println("Packages requiring verification:")
			for _, pkg := range compat.MaybeSupport {
				fmt.Printf("  - %s\n", pkg)
			}
			fmt.Println("")
		}

		if *checkOnline && len(compat.Packages) > 0 {
			fmt.Println("Checking online availability...")
			available, unavailable := generator.CheckWebRPackageAvailability(compat.Packages)

			if len(available) > 0 {
				fmt.Println("\n✓ Available packages:")
				for _, pkg := range available {
					fmt.Printf("  - %s\n", pkg)
				}
			}

			if len(unavailable) > 0 {
				fmt.Println("\n✗ Potentially unavailable packages:")
				for _, pkg := range unavailable {
					fmt.Printf("  - %s\n", pkg)
				}
			}
			fmt.Println("")
		}

		if len(compat.Issues) > 0 {
			fmt.Println("Issues found:")
			for _, issue := range compat.Issues {
				fmt.Printf("  ✗ %s\n", issue)
			}
			fmt.Println("")
		}

		if compat.Compatible {
			fmt.Println("✓ Plugin appears compatible with WebR")
		} else {
			fmt.Println("✗ Plugin has compatibility issues with WebR")
			os.Exit(1)
		}
		return
	}

	if *generateWorkflow {
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
		PluginPath:      *pluginPath,
		OutputDir:       *outputDir,
		PyodideVersion:  *pyodideVersion,
		WebRVersion:     *webrVersion,
		SkipCheck:       *skipCheck,
		NoBuild:         *noBuild,
		ExamplesDir:     *examplesDir,
		SpaTemplatePath: *spaTemplatePath,
		SharedLibPath:   *sharedLibPath,
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
	info, err := os.Stat(pluginPath)
	if err != nil {
		for i := len(pluginPath) - 1; i >= 0; i-- {
			if pluginPath[i] == '/' || pluginPath[i] == '\\' {
				return pluginPath[:i]
			}
		}
		return "."
	}
	if info.IsDir() {
		return pluginPath
	}
	for i := len(pluginPath) - 1; i >= 0; i-- {
		if pluginPath[i] == '/' || pluginPath[i] == '\\' {
			return pluginPath[:i]
		}
	}
	return "."
}
