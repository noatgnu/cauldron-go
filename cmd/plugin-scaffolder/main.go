package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/noatgnu/cauldron-go/internal/templates"
)

// stdinReader is shared across every prompt: a fresh bufio.Reader per call would buffer and
// then discard any not-yet-read piped input, breaking non-interactive/scripted use.
var stdinReader = bufio.NewReader(os.Stdin)

func readLine() (string, bool) {
	input, err := stdinReader.ReadString('\n')
	if err != nil && input == "" {
		return "", false
	}
	return strings.TrimSpace(input), true
}

func prompt(text string, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", text, defaultValue)
	} else {
		fmt.Printf("%s: ", text)
	}

	input, ok := readLine()
	if !ok {
		fmt.Println("[ERROR] Unexpected end of input")
		os.Exit(1)
	}

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
		fmt.Print("Select option (number): ")
		input, ok := readLine()
		if !ok {
			fmt.Println("[ERROR] Unexpected end of input")
			os.Exit(1)
		}

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
	fmt.Printf("%s (y/n): ", text)
	input, ok := readLine()
	if !ok {
		return false
	}
	input = strings.ToLower(input)
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

func renderTemplate(name string, data interface{}) (string, error) {
	tmplStr, err := templates.GetTemplate(name)
	if err != nil {
		return "", fmt.Errorf("failed to load template %s: %w", name, err)
	}
	tmpl, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}
	return buf.String(), nil
}

func createPluginYAML(dir string, data map[string]string) error {
	content, err := renderTemplate("scaffold-plugin.yaml.tmpl", map[string]interface{}{
		"ID":          data["id"],
		"Name":        data["name"],
		"Description": data["description"],
		"Version":     data["version"],
		"Author":      data["author"],
		"Category":    data["category"],
		"Runtime":     data["runtime"],
		"Script":      data["script"],
		"HasPython":   data["runtime"] == "python" || data["runtime"] == "pythonWithR",
		"HasR":        data["runtime"] == "r" || data["runtime"] == "pythonWithR",
	})
	if err != nil {
		return err
	}

	yamlPath := filepath.Join(dir, "plugin.yaml")
	return os.WriteFile(yamlPath, []byte(content), 0644)
}

func createPythonScript(dir, scriptName, pluginName, description string) error {
	content, err := renderTemplate("scaffold-script.py.tmpl", map[string]string{
		"PluginName":  pluginName,
		"Description": description,
	})
	if err != nil {
		return err
	}

	scriptPath := filepath.Join(dir, scriptName)
	return os.WriteFile(scriptPath, []byte(content), 0755)
}

func createRScript(dir, scriptName, pluginName, description string) error {
	content, err := renderTemplate("scaffold-script.R.tmpl", map[string]string{
		"PluginName":  pluginName,
		"Description": description,
	})
	if err != nil {
		return err
	}

	scriptPath := filepath.Join(dir, scriptName)
	return os.WriteFile(scriptPath, []byte(content), 0755)
}

func createGitignore(dir string) error {
	content, err := templates.GetTemplate("scaffold-gitignore.tmpl")
	if err != nil {
		return err
	}
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
