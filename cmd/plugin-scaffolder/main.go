package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func prompt(text string, defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)

	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", text, defaultValue)
	} else {
		fmt.Printf("%s: ", text)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" && defaultValue != "" {
		return defaultValue
	}

	return input
}

func selectOption(promptText string, options []string) string {
	fmt.Println(promptText)
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Select option (number): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		var selected int
		if _, err := fmt.Sscanf(input, "%d", &selected); err == nil {
			if selected >= 1 && selected <= len(options) {
				return options[selected-1]
			}
		}

		fmt.Println("[WARNING] Invalid selection, please try again")
	}
}

func confirm(text string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (y/n): ", text)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))
	return input == "y" || input == "yes"
}

func titleCase(s string) string {
	words := strings.Split(s, " ")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func createPluginYAML(dir string, data map[string]string) error {
	content := fmt.Sprintf(`plugin:
  id: "%s"
  name: "%s"
  description: "%s"
  version: "%s"
  author: "%s"
  category: "%s"
  icon: "analytics"

runtime:
  type: "%s"
  script: "%s"

inputs:
  - name: "input_file"
    label: "Input File"
    type: "file"
    required: true
    accept: ".csv,.tsv,.txt"
    description: "Input data file"

  # Add more inputs as needed
  # - name: "parameter1"
  #   label: "Parameter 1"
  #   type: "number"
  #   default: 10
  #   min: 1
  #   max: 100
  #   description: "Example numeric parameter"

outputs:
  - name: "output_data"
    path: "output.txt"
    type: "data"
    description: "Analysis results"
    format: "tsv"

execution:
  argsMapping:
    input_file: "--input_file"
    # Add more argument mappings here

  outputDir: "--output_folder"

  requirements:
`,
		data["id"],
		data["name"],
		data["description"],
		data["version"],
		data["author"],
		data["category"],
		data["runtime"],
		data["script"])

	if data["runtime"] == "python" || data["runtime"] == "pythonWithR" {
		content += `    python: ">=3.11"
    packages:
      - "pandas>=2.0.0"
      - "numpy>=1.24.0"
`
	}

	if data["runtime"] == "r" || data["runtime"] == "pythonWithR" {
		content += `    r: ">=4.0"
    packages:
      - "tidyverse"
`
	}

	content += `
# example:
#   enabled: true
#   values:
#     input_file: "diann/imputed.data.txt"
#     # Add more example values here
`

	yamlPath := filepath.Join(dir, "plugin.yaml")
	return os.WriteFile(yamlPath, []byte(content), 0644)
}

func createPythonScript(dir, scriptName, pluginName, description string) error {
	content := fmt.Sprintf(`#!/usr/bin/env python3
"""
%s
%s
"""

import argparse
import sys
import pandas as pd
from pathlib import Path


def parse_args():
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(description='%s')

    parser.add_argument('--input_file', required=True, help='Input data file')
    parser.add_argument('--output_folder', required=True, help='Output directory')

    # Add more arguments as needed
    # parser.add_argument('--parameter1', type=int, default=10, help='Example parameter')

    return parser.parse_args()


def main():
    """Main execution function."""
    args = parse_args()

    # Create output directory
    output_dir = Path(args.output_folder)
    output_dir.mkdir(parents=True, exist_ok=True)

    print(f"Loading data from {args.input_file}...")
    # TODO: Implement your analysis logic here
    # df = pd.read_csv(args.input_file, sep='\t')

    print("Processing...")
    # TODO: Add processing logic

    # Save results
    output_file = output_dir / 'output.txt'
    print(f"Saving results to {output_file}...")
    # TODO: Save results
    # df.to_csv(output_file, sep='\t', index=False)

    print("Analysis complete!")
    return 0


if __name__ == '__main__':
    sys.exit(main())
`, pluginName, description, description)

	scriptPath := filepath.Join(dir, scriptName)
	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		return err
	}

	return nil
}

func createRScript(dir, scriptName, pluginName, description string) error {
	content := fmt.Sprintf(`#!/usr/bin/env Rscript

# %s
# %s

library(optparse)

# Parse command line arguments
option_list <- list(
  make_option(c("--input_file"), type="character", help="Input data file"),
  make_option(c("--output_folder"), type="character", help="Output directory")
  # Add more options as needed
)

parser <- OptionParser(option_list=option_list)
args <- parse_args(parser)

# Create output directory
dir.create(args$output_folder, showWarnings = FALSE, recursive = TRUE)

cat("Loading data from", args$input_file, "...\n")
# TODO: Implement your analysis logic here
# data <- read.delim(args$input_file)

cat("Processing...\n")
# TODO: Add processing logic

# Save results
output_file <- file.path(args$output_folder, "output.txt")
cat("Saving results to", output_file, "...\n")
# TODO: Save results
# write.table(data, output_file, sep="\t", quote=FALSE, row.names=FALSE)

cat("Analysis complete!\n")
`, pluginName, description)

	scriptPath := filepath.Join(dir, scriptName)
	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		return err
	}

	return nil
}

func createGitignore(dir string) error {
	content := `__pycache__/
*.pyc
.Rhistory
.RData
*.swp
*.swo
*~
`
	gitignorePath := filepath.Join(dir, ".gitignore")
	return os.WriteFile(gitignorePath, []byte(content), 0644)
}

func main() {
	fmt.Println("CauldronGO Plugin Scaffolding Tool")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("Let's create a new plugin!")
	fmt.Println()

	pluginID := prompt("Plugin ID (lowercase, hyphenated)", "")
	if pluginID == "" {
		fmt.Println("[ERROR] Plugin ID is required")
		os.Exit(1)
	}

	pluginsDir := "plugins"
	pluginDir := filepath.Join(pluginsDir, pluginID)

	if _, err := os.Stat(pluginDir); !os.IsNotExist(err) {
		fmt.Printf("[ERROR] Plugin '%s' already exists\n", pluginID)
		os.Exit(1)
	}

	defaultName := titleCase(strings.ReplaceAll(pluginID, "-", " "))
	pluginName := prompt("Plugin Name (human-readable)", defaultName)
	pluginDesc := prompt("Description", "Analysis tool for CauldronGO")
	pluginVersion := prompt("Version", "1.0.0")
	pluginAuthor := prompt("Author", "CauldronGO Team")

	fmt.Println()
	pluginCategory := selectOption("Select plugin category:", []string{"analysis", "preprocessing", "visualization", "utilities"})

	fmt.Println()
	pluginRuntime := selectOption("Select runtime:", []string{"python", "r", "pythonWithR"})

	scriptExt := ""
	switch pluginRuntime {
	case "python", "pythonWithR":
		scriptExt = "py"
	case "r":
		scriptExt = "R"
	}

	scriptName := strings.ReplaceAll(pluginID, "-", "_") + "." + scriptExt

	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf("  ID:       %s\n", pluginID)
	fmt.Printf("  Name:     %s\n", pluginName)
	fmt.Printf("  Category: %s\n", pluginCategory)
	fmt.Printf("  Runtime:  %s\n", pluginRuntime)
	fmt.Printf("  Script:   %s\n", scriptName)
	fmt.Println()

	if !confirm("Create plugin?") {
		fmt.Println("[WARNING] Cancelled")
		os.Exit(0)
	}

	fmt.Println()
	fmt.Println("Creating plugin directory...")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		fmt.Printf("[ERROR] Failed to create directory: %v\n", err)
		os.Exit(1)
	}

	data := map[string]string{
		"id":          pluginID,
		"name":        pluginName,
		"description": pluginDesc,
		"version":     pluginVersion,
		"author":      pluginAuthor,
		"category":    pluginCategory,
		"runtime":     pluginRuntime,
		"script":      scriptName,
	}

	fmt.Println("Creating plugin.yaml...")
	if err := createPluginYAML(pluginDir, data); err != nil {
		fmt.Printf("[ERROR] Failed to create plugin.yaml: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[SUCCESS] Created plugin.yaml")

	if scriptExt == "py" {
		fmt.Println("Creating Python script...")
		if err := createPythonScript(pluginDir, scriptName, pluginName, pluginDesc); err != nil {
			fmt.Printf("[ERROR] Failed to create script: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[SUCCESS] Created %s\n", scriptName)
	} else if scriptExt == "R" {
		fmt.Println("Creating R script...")
		if err := createRScript(pluginDir, scriptName, pluginName, pluginDesc); err != nil {
			fmt.Printf("[ERROR] Failed to create script: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[SUCCESS] Created %s\n", scriptName)
	}

	fmt.Println("Creating .gitignore...")
	if err := createGitignore(pluginDir); err != nil {
		fmt.Printf("[ERROR] Failed to create .gitignore: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[SUCCESS] Created .gitignore")

	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("Plugin Created Successfully!")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Edit %s/plugin.yaml to add inputs/outputs\n", pluginDir)
	fmt.Printf("  2. Implement analysis logic in %s/%s\n", pluginDir, scriptName)
	fmt.Println("  3. Add example data (optional)")
	fmt.Printf("  4. Validate: ./bin/plugin-validator %s\n", pluginDir)
	fmt.Printf("  5. Generate docs: ./bin/plugin-doc-generator %s\n", pluginDir)
	fmt.Printf("  6. Test in UI: Navigate to Plugin View → %s → %s\n", pluginCategory, pluginName)
	fmt.Println()
	fmt.Println("[SUCCESS] Happy coding!")
}
