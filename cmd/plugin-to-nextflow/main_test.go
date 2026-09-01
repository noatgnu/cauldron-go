package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testPluginYAML = `
plugin:
  id: "%s"
  name: "%s"
  description: "A test plugin"
  version: "1.0.0"
  category: "utilities"

runtime:
  environments:
    - python
  entrypoint: run.py

inputs:
  - name: input_file
    label: Input File
    type: file
    required: true

execution:
  outputDir: "--output_folder"
  argsMapping: {}
`

func writeTestPlugin(t *testing.T, dir, id string) string {
	t.Helper()
	pluginDir := filepath.Join(dir, id)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}
	path := filepath.Join(pluginDir, "plugin.yaml")
	content := fmt.Sprintf(testPluginYAML, id, id)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write plugin.yaml: %v", err)
	}
	return path
}

func TestConvertPluginsBatch_SeparateOutputDirs(t *testing.T) {
	pluginsDir := t.TempDir()
	writeTestPlugin(t, pluginsDir, "plugin-one")
	writeTestPlugin(t, pluginsDir, "plugin-two")

	outDir := t.TempDir()
	if err := convertPluginsBatch(pluginsDir, outDir, false); err != nil {
		t.Fatalf("convertPluginsBatch error: %v", err)
	}

	oneMain, err := os.ReadFile(filepath.Join(outDir, "plugin-one", "main.nf"))
	if err != nil {
		t.Fatalf("expected plugin-one/main.nf to exist: %v", err)
	}
	twoMain, err := os.ReadFile(filepath.Join(outDir, "plugin-two", "main.nf"))
	if err != nil {
		t.Fatalf("expected plugin-two/main.nf to exist: %v", err)
	}

	if !strings.Contains(string(oneMain), "PLUGIN_ONE") {
		t.Errorf("expected plugin-one/main.nf to reference PLUGIN_ONE, got:\n%s", oneMain)
	}
	if !strings.Contains(string(twoMain), "PLUGIN_TWO") {
		t.Errorf("expected plugin-two/main.nf to reference PLUGIN_TWO, got:\n%s", twoMain)
	}

	if _, err := os.Stat(filepath.Join(outDir, "plugin-one", "modules", "local", "plugin-one", "main.nf")); err != nil {
		t.Errorf("expected plugin-one module dir to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "plugin-two", "modules", "local", "plugin-two", "main.nf")); err != nil {
		t.Errorf("expected plugin-two module dir to exist: %v", err)
	}
}
