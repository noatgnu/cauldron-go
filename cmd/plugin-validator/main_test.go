package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

const testPluginYAMLBase = `
plugin:
  id: "test-plugin"
  name: "Test Plugin"
  description: "A test plugin"
  version: "1.0.0"
  category: "utilities"

runtime:
  environments: ["r"]
  entrypoint: "run.R"

inputs:
  - name: input_file
    label: Input File
    type: file
    required: true

execution:
  outputDir: "--output_folder"
  argsMapping: {}
%s
`

func writeTestPlugin(t *testing.T, dir string, diagramBlock string, scriptContent string) string {
	t.Helper()
	yamlPath := filepath.Join(dir, "plugin.yaml")
	content := fmt.Sprintf(testPluginYAMLBase, diagramBlock)
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write plugin.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "run.R"), []byte(scriptContent), 0644); err != nil {
		t.Fatalf("failed to write run.R: %v", err)
	}
	return yamlPath
}

func TestValidatePlugin_DiagramFieldParsed(t *testing.T) {
	dir := t.TempDir()
	yamlPath := writeTestPlugin(t, dir, "\ndiagram:\n  enabled: true\n", "# @step: Loading data\n")

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("failed to read plugin.yaml: %v", err)
	}
	var plugin PluginConfig
	if err := yaml.Unmarshal(data, &plugin); err != nil {
		t.Fatalf("failed to unmarshal plugin.yaml: %v", err)
	}
	if plugin.Diagram == nil || !plugin.Diagram.Enabled {
		t.Fatalf("expected diagram.enabled to parse as true, got %+v", plugin.Diagram)
	}
}

func TestValidatePlugin_WarnsWhenDiagramEnabledWithNoStepMarkers(t *testing.T) {
	dir := t.TempDir()
	yamlPath := writeTestPlugin(t, dir, "\ndiagram:\n  enabled: true\n", "x <- 1\n")

	valid, errors := validatePlugin(yamlPath)
	if !valid {
		t.Fatalf("expected plugin to still be valid (warning is non-fatal), got errors: %v", errors)
	}
}

func TestValidatePlugin_SelectOptionsAcceptMappingForm(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "plugin.yaml")
	yamlContent := `
plugin:
  id: "test-plugin"
  name: "Test Plugin"
  description: "A test plugin"
  version: "1.0.0"
  category: "utilities"

runtime:
  environments: ["r"]
  entrypoint: "run.R"

inputs:
  - name: method
    label: Method
    type: select
    required: true
    options:
      - value: "none"
        label: "No Normalization"
      - value: "median"
        label: "Median Centering"

execution:
  outputDir: "--output_folder"
  argsMapping: {}
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write plugin.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "run.R"), []byte("x <- 1\n"), 0644); err != nil {
		t.Fatalf("failed to write run.R: %v", err)
	}

	valid, errors := validatePlugin(yamlPath)
	if !valid {
		t.Fatalf("expected {value, label} options to parse as valid YAML, got errors: %v", errors)
	}
}

func TestScriptHasStepMarkers(t *testing.T) {
	dir := t.TempDir()

	markerPath := filepath.Join(dir, "marker.R")
	os.WriteFile(markerPath, []byte("# @step: Loading data\n"), 0644)
	if !scriptHasStepMarkers(markerPath) {
		t.Error("expected @step marker to be detected")
	}

	decisionPath := filepath.Join(dir, "decision.R")
	os.WriteFile(decisionPath, []byte("# @step-if: Optional QC\n"), 0644)
	if !scriptHasStepMarkers(decisionPath) {
		t.Error("expected @step-if marker to be detected")
	}

	legacyPath := filepath.Join(dir, "legacy.R")
	os.WriteFile(legacyPath, []byte(`message("[1/2] Loading data...")`+"\n"), 0644)
	if !scriptHasStepMarkers(legacyPath) {
		t.Error("expected legacy [N/M] marker to be detected")
	}

	nonePath := filepath.Join(dir, "none.R")
	os.WriteFile(nonePath, []byte("x <- 1\nmessage(\"Loading data...\")\n"), 0644)
	if scriptHasStepMarkers(nonePath) {
		t.Error("expected no markers to be detected in a plain script")
	}

	branchPath := filepath.Join(dir, "branch.R")
	os.WriteFile(branchPath, []byte("# @step[id=load]: Loading data\n"), 0644)
	if !scriptHasStepMarkers(branchPath) {
		t.Error("expected a branch/loop-style @step[...] marker to be detected")
	}
}
