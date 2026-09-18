package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/internal/templates"
)

func testDefinition() *models.PluginDefinition {
	return &models.PluginDefinition{
		Plugin: models.PluginMetadata{
			ID:          "my-test-plugin",
			Name:        "My Test Plugin",
			Description: "A plugin used in tests",
			Version:     "1.0.0",
			Author:      "Test Author",
		},
		Runtime: models.PluginRuntimeV2{
			Environments: []string{"python"},
			Entrypoint:   "run.py",
		},
		Inputs: []models.PluginInputV2{
			{Name: "input_file", Type: models.PluginInputTypeFile, Required: true},
			{Name: "alpha", Type: models.PluginInputTypeText, Description: "alpha flag"},
			{Name: "beta", Type: models.PluginInputTypeBoolean, Description: "beta flag"},
			{Name: "gamma", Type: models.PluginInputTypeNumber, Description: "gamma flag", Default: 5.0},
			{Name: "delta", Type: models.PluginInputTypeText, Description: "delta flag"},
		},
		Outputs: []models.PluginOutputV2{
			{Name: "result", Path: "output.txt", Type: "data"},
		},
		Execution: models.PluginExecution{
			OutputDir: "--output_folder",
			ArgsMapping: map[string]interface{}{
				"alpha": "--alpha",
				"beta":  "--beta",
				"gamma": "--gamma",
				"delta": "--delta",
			},
		},
	}
}

func TestGenerateProcess_ArgsMapping_Deterministic(t *testing.T) {
	definition := testDefinition()
	procTmpl, err := templates.GetTemplate("process.nf.tmpl")
	if err != nil {
		t.Fatalf("failed to load process template: %v", err)
	}

	var first string
	for i := 0; i < 20; i++ {
		content, err := GenerateProcess(definition, procTmpl)
		if err != nil {
			t.Fatalf("GenerateProcess error: %v", err)
		}
		if i == 0 {
			first = content
			continue
		}
		if content != first {
			t.Fatalf("GenerateProcess output is not deterministic across runs (run %d differs)", i)
		}
	}
}

func TestGenerateProcess_PublishDirWired(t *testing.T) {
	definition := testDefinition()
	procTmpl, err := templates.GetTemplate("process.nf.tmpl")
	if err != nil {
		t.Fatalf("failed to load process template: %v", err)
	}

	content, err := GenerateProcess(definition, procTmpl)
	if err != nil {
		t.Fatalf("GenerateProcess error: %v", err)
	}
	if !strings.Contains(content, "publishDir \"${params.outdir}/") {
		t.Errorf("expected generated process to declare a publishDir using params.outdir, got:\n%s", content)
	}
}

func TestGenerateWorkflow_BooleanFalsePreserved(t *testing.T) {
	definition := testDefinition()
	wfTmpl, err := templates.GetTemplate("main.nf.tmpl")
	if err != nil {
		t.Fatalf("failed to load workflow template: %v", err)
	}

	content, err := GenerateWorkflow(definition, wfTmpl)
	if err != nil {
		t.Fatalf("GenerateWorkflow error: %v", err)
	}
	if !strings.Contains(content, "params.beta != null ? params.beta : ''") {
		t.Errorf("expected boolean param channel to use a null-check instead of the elvis operator, got:\n%s", content)
	}
	if strings.Contains(content, "params.beta ?: ''") {
		t.Error("workflow still uses the elvis operator, which drops boolean false values")
	}
}

func TestGenerateConfig_NoDeadInputParam(t *testing.T) {
	definition := testDefinition()
	cfgTmpl, err := templates.GetTemplate("nextflow.config.tmpl")
	if err != nil {
		t.Fatalf("failed to load config template: %v", err)
	}

	content, err := GenerateConfig(definition, cfgTmpl)
	if err != nil {
		t.Fatalf("GenerateConfig error: %v", err)
	}
	if strings.Contains(content, "\n    input = ") {
		t.Errorf("generated config still declares a dead 'input' param, got:\n%s", content)
	}
	if !strings.Contains(content, "input_file = null") {
		t.Errorf("expected generated config to declare input_file (the real param name), got:\n%s", content)
	}
	if !strings.Contains(content, "outdir = './results'") {
		t.Errorf("expected generated config to keep the outdir param, got:\n%s", content)
	}
}

func TestGenerateREADME_UsesSanitizedProcessNameAndRawModulePath(t *testing.T) {
	readmeTmpl, err := templates.GetTemplate("README.md.tmpl")
	if err != nil {
		t.Fatalf("failed to load README template: %v", err)
	}

	definition := testDefinition()
	content, err := GenerateREADME(definition, readmeTmpl)
	if err != nil {
		t.Fatalf("GenerateREADME error: %v", err)
	}

	if !strings.Contains(content, "include { MY_TEST_PLUGIN }") {
		t.Errorf("expected sanitized (underscored) process name in include statement, got:\n%s", content)
	}
	if !strings.Contains(content, "./path/to/my-test-plugin/modules/local/my-test-plugin/main") {
		t.Errorf("expected raw plugin ID in module path, got:\n%s", content)
	}
	if strings.Contains(content, "docker build -t  -f") {
		t.Error("ContainerImageDocker was not set, README renders an empty image tag")
	}
}

func TestGenerateREADME_UsageExamplesUseInputFile(t *testing.T) {
	readmeTmpl, err := templates.GetTemplate("README.md.tmpl")
	if err != nil {
		t.Fatalf("failed to load README template: %v", err)
	}

	content, err := GenerateREADME(testDefinition(), readmeTmpl)
	if err != nil {
		t.Fatalf("GenerateREADME error: %v", err)
	}
	if strings.Contains(content, "--input '") {
		t.Errorf("README usage examples should reference --input_file, not the nonexistent --input param, got:\n%s", content)
	}
}

func TestGeneratePackageJSON_MatchesSharedLibAngularVersion(t *testing.T) {
	outDir := t.TempDir()
	g := &SPAGenerator{
		config:     SPAConfig{OutputDir: outDir},
		definition: testDefinition(),
		pluginDir:  outDir,
	}

	if err := g.generatePackageJSON(); err != nil {
		t.Fatalf("generatePackageJSON error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "package.json"))
	if err != nil {
		t.Fatalf("failed to read generated package.json: %v", err)
	}
	content := string(data)

	// @cauldron/forms (shared-lib) is built against Angular 22.
	for _, want := range []string{`"@angular/core": "^22.1.4"`, `"@angular/cli": "^22.1.6"`} {
		if !strings.Contains(content, want) {
			t.Errorf("expected generated package.json to contain %q, got:\n%s", want, content)
		}
	}
	if strings.Contains(content, "21.") {
		t.Errorf("expected no leftover Angular 21 version pins, got:\n%s", content)
	}
}

func TestGenerateGithubWorkflow_UsesSupportedNodeVersion(t *testing.T) {
	outDir := t.TempDir()
	g := &SPAGenerator{
		config:     SPAConfig{OutputDir: outDir},
		definition: testDefinition(),
		pluginDir:  outDir,
	}

	if err := g.generateGithubWorkflow(); err != nil {
		t.Fatalf("generateGithubWorkflow error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, ".github", "workflows", "deploy.yml"))
	if err != nil {
		t.Fatalf("failed to read generated workflow: %v", err)
	}
	content := string(data)

	// Must match this repo's own .github/workflows/*.yml pin.
	if !strings.Contains(content, "node-version: '22.23.1'") {
		t.Errorf("expected generated workflow to pin node-version '22.23.1', got:\n%s", content)
	}
	if strings.Contains(content, "node-version: '20'") {
		t.Errorf("expected no leftover Node 20 pin, got:\n%s", content)
	}
}

func TestGeneratePluginWorkflowWithVersions_UsesSupportedNodeVersion(t *testing.T) {
	for _, tc := range []struct {
		name        string
		environment string
	}{
		{name: "python", environment: "python"},
		{name: "r", environment: "r"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pluginDir := t.TempDir()
			yamlContent := `
plugin:
  id: "test-plugin"
  name: "Test Plugin"
  description: "A test plugin"
  version: "1.0.0"
  category: "utilities"

runtime:
  environments: ["` + tc.environment + `"]
  entrypoint: "run"

execution:
  outputDir: "--output_folder"
  argsMapping: {}
`
			if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(yamlContent), 0644); err != nil {
				t.Fatalf("failed to write plugin.yaml: %v", err)
			}

			if err := GeneratePluginWorkflowWithVersions(pluginDir, "0.28.0", "0.5.0"); err != nil {
				t.Fatalf("GeneratePluginWorkflowWithVersions error: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(pluginDir, ".github", "workflows", "deploy-spa.yml"))
			if err != nil {
				t.Fatalf("failed to read generated workflow: %v", err)
			}
			content := string(data)

			if !strings.Contains(content, "node-version: '22.23.1'") {
				t.Errorf("expected generated workflow to pin node-version '22.23.1', got:\n%s", content)
			}
			if strings.Contains(content, "node-version: '20'") {
				t.Errorf("expected no leftover Node 20 pin, got:\n%s", content)
			}
		})
	}
}
