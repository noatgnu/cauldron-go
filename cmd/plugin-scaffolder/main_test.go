package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
	"gopkg.in/yaml.v3"
)

func scaffoldData(runtime string) map[string]string {
	return map[string]string{
		"id":          "my-plugin",
		"name":        "My Plugin",
		"description": "A test plugin",
		"version":     "1.0.0",
		"author":      "Test Author",
		"category":    "analysis",
		"runtime":     runtime,
		"script":      "my_plugin.py",
	}
}

func TestCreatePluginYAML_AllRuntimesProduceValidYAML(t *testing.T) {
	for _, runtime := range []string{"python", "r", "pythonWithR"} {
		t.Run(runtime, func(t *testing.T) {
			dir := t.TempDir()
			if err := createPluginYAML(dir, scaffoldData(runtime)); err != nil {
				t.Fatalf("createPluginYAML error: %v", err)
			}

			raw, err := os.ReadFile(filepath.Join(dir, "plugin.yaml"))
			if err != nil {
				t.Fatalf("failed to read generated plugin.yaml: %v", err)
			}

			var definition models.PluginDefinition
			if err := yaml.Unmarshal(raw, &definition); err != nil {
				t.Fatalf("generated plugin.yaml for runtime %q is not valid YAML: %v\n%s", runtime, err, raw)
			}
			if definition.Plugin.ID != "my-plugin" {
				t.Errorf("expected plugin id 'my-plugin', got %q", definition.Plugin.ID)
			}
		})
	}
}

func TestCreatePluginYAML_PythonWithR_MergesPackages(t *testing.T) {
	dir := t.TempDir()
	if err := createPluginYAML(dir, scaffoldData("pythonWithR")); err != nil {
		t.Fatalf("createPluginYAML error: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "plugin.yaml"))
	if err != nil {
		t.Fatalf("failed to read generated plugin.yaml: %v", err)
	}

	var definition models.PluginDefinition
	if err := yaml.Unmarshal(raw, &definition); err != nil {
		t.Fatalf("generated plugin.yaml is not valid YAML: %v\n%s", err, raw)
	}

	if definition.Execution.Requirements.Python == "" {
		t.Error("expected Python requirement to be set")
	}
	if definition.Execution.Requirements.R == "" {
		t.Error("expected R requirement to be set")
	}

	joined := strings.Join(definition.Execution.Requirements.Packages, ",")
	if !strings.Contains(joined, "pandas") || !strings.Contains(joined, "tidyverse") {
		t.Errorf("expected merged packages from both ecosystems, got: %v", definition.Execution.Requirements.Packages)
	}
}

func TestCreatePythonScript(t *testing.T) {
	dir := t.TempDir()
	if err := createPythonScript(dir, "my_plugin.py", "My Plugin", "A test plugin"); err != nil {
		t.Fatalf("createPythonScript error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "my_plugin.py"))
	if err != nil {
		t.Fatalf("failed to read generated script: %v", err)
	}
	if !strings.Contains(string(content), "My Plugin") {
		t.Errorf("expected plugin name in script docstring, got:\n%s", content)
	}
}

func TestCreateRScript(t *testing.T) {
	dir := t.TempDir()
	if err := createRScript(dir, "my_plugin.R", "My Plugin", "A test plugin"); err != nil {
		t.Fatalf("createRScript error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "my_plugin.R"))
	if err != nil {
		t.Fatalf("failed to read generated script: %v", err)
	}
	if !strings.Contains(string(content), "My Plugin") {
		t.Errorf("expected plugin name in script header, got:\n%s", content)
	}
}

func TestCreateGitignore(t *testing.T) {
	dir := t.TempDir()
	if err := createGitignore(dir); err != nil {
		t.Fatalf("createGitignore error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("failed to read generated .gitignore: %v", err)
	}
	if !strings.Contains(string(content), "__pycache__/") {
		t.Errorf("expected __pycache__/ in .gitignore, got:\n%s", content)
	}
}

func TestCreateExamplesDir(t *testing.T) {
	dir := t.TempDir()
	if err := createExamplesDir(dir, "my-plugin"); err != nil {
		t.Fatalf("createExamplesDir error: %v", err)
	}

	readme, err := os.ReadFile(filepath.Join(dir, "examples", "README.md"))
	if err != nil {
		t.Fatalf("failed to read examples/README.md: %v", err)
	}
	if !strings.Contains(string(readme), "my-plugin") {
		t.Errorf("expected plugin id in examples/README.md, got:\n%s", readme)
	}

	params, err := os.ReadFile(filepath.Join(dir, "examples", "params.json"))
	if err != nil {
		t.Fatalf("failed to read examples/params.json: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(params, &decoded); err != nil {
		t.Fatalf("examples/params.json is not valid JSON: %v\n%s", err, params)
	}
}

func TestCreateGithubWorkflow(t *testing.T) {
	for _, runtime := range []string{"python", "r", "pythonWithR"} {
		t.Run(runtime, func(t *testing.T) {
			dir := t.TempDir()
			if err := createGithubWorkflow(dir, scaffoldData(runtime)); err != nil {
				t.Fatalf("createGithubWorkflow error: %v", err)
			}

			raw, err := os.ReadFile(filepath.Join(dir, ".github", "workflows", "test-plugin.yml"))
			if err != nil {
				t.Fatalf("failed to read generated workflow: %v", err)
			}

			var workflow map[string]interface{}
			if err := yaml.Unmarshal(raw, &workflow); err != nil {
				t.Fatalf("generated workflow for runtime %q is not valid YAML: %v\n%s", runtime, err, raw)
			}

			content := string(raw)
			if !strings.Contains(content, "my-plugin") {
				t.Errorf("expected plugin id in workflow, got:\n%s", content)
			}
			wantPython := runtime == "python" || runtime == "pythonWithR"
			wantR := runtime == "r" || runtime == "pythonWithR"
			if strings.Contains(content, "setup-python") != wantPython {
				t.Errorf("setup-python step presence = %v, want %v for runtime %q, got:\n%s", strings.Contains(content, "setup-python"), wantPython, runtime, content)
			}
			if strings.Contains(content, "setup-r") != wantR {
				t.Errorf("setup-r step presence = %v, want %v for runtime %q, got:\n%s", strings.Contains(content, "setup-r"), wantR, runtime, content)
			}
		})
	}
}
