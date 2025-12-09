package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	pluginsDir := "plugins"

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		fmt.Printf("[ERROR] Failed to read plugins directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generating documentation for all plugins...")
	fmt.Println("==========================================")
	fmt.Println()

	successCount := 0
	failCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join(pluginsDir, entry.Name())
		yamlPath := filepath.Join(pluginDir, "plugin.yaml")

		if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
			yamlPath = filepath.Join(pluginDir, "plugin.yml")
			if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
				fmt.Printf("[WARNING] Skipping %s (no plugin.yaml found)\n", entry.Name())
				continue
			}
		}

		fmt.Printf("Generating docs for: %s\n", entry.Name())

		cmd := exec.Command("./bin/plugin-doc-generator", pluginDir)
		output, err := cmd.CombinedOutput()

		if err != nil {
			fmt.Printf("[ERROR] Failed: %v\n", err)
			fmt.Println(string(output))
			failCount++
		} else {
			fmt.Println("[SUCCESS] Documentation generated")
			successCount++
		}

		fmt.Println()
	}

	fmt.Println("==========================================")
	fmt.Println("Summary")
	fmt.Println("==========================================")
	fmt.Printf("Success: %d\n", successCount)
	fmt.Printf("Failed: %d\n", failCount)
	fmt.Printf("Total: %d\n", successCount+failCount)
	fmt.Println()

	if failCount > 0 {
		fmt.Println("[ERROR] Some plugins failed documentation generation")
		os.Exit(1)
	}

	fmt.Println("[SUCCESS] All plugin documentation generated successfully!")
}
