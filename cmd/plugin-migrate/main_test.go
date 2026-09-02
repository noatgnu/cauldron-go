package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/internal/pluginmigrations"
)

func samplePluginDefinition() *models.PluginDefinition {
	return &models.PluginDefinition{
		Plugin: models.PluginMetadata{
			ID:       "sample-plugin",
			Name:     "Sample Plugin",
			Version:  "1.0.0",
			Category: models.PluginCategoryAnalysis,
		},
		Runtime: models.PluginRuntimeV2{
			Environments: []string{"python"},
			Entrypoint:   "run.py",
		},
		Inputs: []models.PluginInputV2{
			{Name: "input_file", Label: "Input File", Type: "file", Required: true},
			{Name: "threshold", Label: "Threshold", Type: "number", Default: 0.05},
		},
		Outputs: []models.PluginOutputV2{
			{Name: "result", Path: "result.tsv", Type: "data", Format: "tsv"},
		},
		Execution: models.PluginExecution{
			ArgsMapping: map[string]interface{}{
				"input_file": "--input_file",
				"threshold":  "--threshold",
			},
			OutputDir: "--output_folder",
			Requirements: models.Requirements{
				Python:   ">=3.11",
				Packages: []string{"numpy>=1.24.0", "pandas>=2.0.0"},
			},
			EnvVariables: []models.PluginInputV2{
				{Name: "API_KEY", Label: "API Key", Type: "text"},
			},
		},
		Example: &models.ExampleData{
			Enabled: true,
			Values: map[string]interface{}{
				"input_file": "example.txt",
				"threshold":  0.05,
			},
		},
	}
}

func writePluginYAML(t *testing.T, dir string, def *models.PluginDefinition) string {
	t.Helper()
	path := filepath.Join(dir, "plugin.yaml")
	if err := savePluginDefinition(path, def); err != nil {
		t.Fatalf("failed to write plugin.yaml: %v", err)
	}
	return path
}

func writeMigration(t *testing.T, dir, name, content string) {
	t.Helper()
	migrationsDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatalf("failed to create migrations dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write migration %s: %v", name, err)
	}
}

func TestRenameInput_UpdatesInputArgsMappingAndExample(t *testing.T) {
	def := samplePluginDefinition()

	if err := renameInput(def, "threshold", "pvalue_threshold"); err != nil {
		t.Fatalf("renameInput failed: %v", err)
	}

	found := false
	for _, in := range def.Inputs {
		if in.Name == "pvalue_threshold" {
			found = true
		}
		if in.Name == "threshold" {
			t.Error("old input name still present")
		}
	}
	if !found {
		t.Error("renamed input not found")
	}

	if _, ok := def.Execution.ArgsMapping["threshold"]; ok {
		t.Error("old argsMapping key still present")
	}
	if v, ok := def.Execution.ArgsMapping["pvalue_threshold"]; !ok || v != "--threshold" {
		t.Errorf("argsMapping not migrated correctly, got %v", def.Execution.ArgsMapping)
	}

	if _, ok := def.Example.Values["threshold"]; ok {
		t.Error("old example value key still present")
	}
	if _, ok := def.Example.Values["pvalue_threshold"]; !ok {
		t.Error("example value not migrated to new key")
	}
}

func TestRenameInput_UnknownNameFails(t *testing.T) {
	def := samplePluginDefinition()
	if err := renameInput(def, "does_not_exist", "whatever"); err == nil {
		t.Error("expected error for unknown input, got nil")
	}
}

func TestRemoveInput_RemovesFromArgsMappingAndExample(t *testing.T) {
	def := samplePluginDefinition()
	if err := removeInput(def, "threshold"); err != nil {
		t.Fatalf("removeInput failed: %v", err)
	}
	for _, in := range def.Inputs {
		if in.Name == "threshold" {
			t.Error("removed input still present")
		}
	}
	if _, ok := def.Execution.ArgsMapping["threshold"]; ok {
		t.Error("argsMapping entry not removed")
	}
	if _, ok := def.Example.Values["threshold"]; ok {
		t.Error("example value not removed")
	}
}

func TestAddInput_InsertAfter(t *testing.T) {
	def := samplePluginDefinition()
	op := &pluginmigrations.AddInputOp{
		PluginInputV2: models.PluginInputV2{Name: "min_replicates", Label: "Min Replicates", Type: "number"},
		InsertAfter:   "input_file",
	}
	if err := addInput(def, op); err != nil {
		t.Fatalf("addInput failed: %v", err)
	}
	if len(def.Inputs) != 3 || def.Inputs[1].Name != "min_replicates" {
		names := make([]string, len(def.Inputs))
		for i, in := range def.Inputs {
			names[i] = in.Name
		}
		t.Errorf("expected min_replicates inserted at index 1, got order %v", names)
	}
}

func TestAddInput_AppendsWhenNoInsertAfter(t *testing.T) {
	def := samplePluginDefinition()
	op := &pluginmigrations.AddInputOp{PluginInputV2: models.PluginInputV2{Name: "extra", Type: "text"}}
	if err := addInput(def, op); err != nil {
		t.Fatalf("addInput failed: %v", err)
	}
	if def.Inputs[len(def.Inputs)-1].Name != "extra" {
		t.Error("expected new input appended at the end")
	}
}

func TestAddInput_DuplicateNameFails(t *testing.T) {
	def := samplePluginDefinition()
	op := &pluginmigrations.AddInputOp{PluginInputV2: models.PluginInputV2{Name: "threshold", Type: "number"}}
	if err := addInput(def, op); err == nil {
		t.Error("expected error adding a duplicate input name, got nil")
	}
}

func TestAddInput_InsertAfterUnknownTargetFails(t *testing.T) {
	def := samplePluginDefinition()
	op := &pluginmigrations.AddInputOp{PluginInputV2: models.PluginInputV2{Name: "x", Type: "text"}, InsertAfter: "does_not_exist"}
	if err := addInput(def, op); err == nil {
		t.Error("expected error for unknown insertAfter target, got nil")
	}
}

func TestRenameEnvVarAndRemoveEnvVar(t *testing.T) {
	def := samplePluginDefinition()
	if err := renameEnvVar(def, "API_KEY", "UNIPROT_API_KEY"); err != nil {
		t.Fatalf("renameEnvVar failed: %v", err)
	}
	if def.Execution.EnvVariables[0].Name != "UNIPROT_API_KEY" {
		t.Errorf("env var not renamed, got %q", def.Execution.EnvVariables[0].Name)
	}

	if err := removeEnvVar(def, "UNIPROT_API_KEY"); err != nil {
		t.Fatalf("removeEnvVar failed: %v", err)
	}
	if len(def.Execution.EnvVariables) != 0 {
		t.Error("env var not removed")
	}

	if err := removeEnvVar(def, "does_not_exist"); err == nil {
		t.Error("expected error removing unknown env var, got nil")
	}
}

func TestRenameOutputAndRemoveOutput(t *testing.T) {
	def := samplePluginDefinition()
	if err := renameOutput(def, "result", "results"); err != nil {
		t.Fatalf("renameOutput failed: %v", err)
	}
	if def.Outputs[0].Name != "results" {
		t.Errorf("output not renamed, got %q", def.Outputs[0].Name)
	}

	if err := removeOutput(def, "results"); err != nil {
		t.Fatalf("removeOutput failed: %v", err)
	}
	if len(def.Outputs) != 0 {
		t.Error("output not removed")
	}
}

func TestAddPackage_UpsertsExistingByBaseName(t *testing.T) {
	def := samplePluginDefinition()
	addPackage(def, "numpy>=1.26.0")

	if len(def.Execution.Requirements.Packages) != 2 {
		t.Fatalf("expected package count unchanged (upsert not append), got %d", len(def.Execution.Requirements.Packages))
	}
	if def.Execution.Requirements.Packages[0] != "numpy>=1.26.0" {
		t.Errorf("expected numpy version bumped in place, got %q", def.Execution.Requirements.Packages[0])
	}
}

func TestAddPackage_AppendsWhenNew(t *testing.T) {
	def := samplePluginDefinition()
	addPackage(def, "scipy>=1.11.0")
	if len(def.Execution.Requirements.Packages) != 3 {
		t.Fatalf("expected new package appended, got %d packages", len(def.Execution.Requirements.Packages))
	}
}

func TestRemovePackage(t *testing.T) {
	def := samplePluginDefinition()
	removePackage(def, "numpy")
	for _, p := range def.Execution.Requirements.Packages {
		if packageBaseName(p) == "numpy" {
			t.Error("numpy was not removed")
		}
	}
}

func TestPackageBaseName(t *testing.T) {
	cases := map[string]string{
		"numpy>=1.24.0": "numpy",
		"pandas==2.0.0": "pandas",
		"scikit-learn":  "scikit-learn",
		"foo~=1.0":      "foo",
	}
	for input, want := range cases {
		if got := packageBaseName(input); got != want {
			t.Errorf("packageBaseName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestLoadMigrations_ValidSequence(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "0001_first.yaml", "schemaVersion: 1\noperations: []\n")
	writeMigration(t, dir, "0002_second.yaml", "schemaVersion: 2\noperations: []\n")

	migrations, err := pluginmigrations.LoadMigrations(filepath.Join(dir, "migrations"))
	if err != nil {
		t.Fatalf("loadMigrations failed: %v", err)
	}
	if len(migrations) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(migrations))
	}
	if migrations[0].SchemaVersion != 1 || migrations[1].SchemaVersion != 2 {
		t.Error("migrations not in expected schemaVersion order")
	}
}

func TestLoadMigrations_GapFails(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "0002_skipped_one.yaml", "schemaVersion: 2\noperations: []\n")

	if _, err := pluginmigrations.LoadMigrations(filepath.Join(dir, "migrations")); err == nil {
		t.Error("expected error for a gapped schemaVersion sequence, got nil")
	}
}

func TestLoadMigrations_NoDirReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	migrations, err := pluginmigrations.LoadMigrations(filepath.Join(dir, "migrations"))
	if err != nil {
		t.Fatalf("expected no error for missing migrations dir, got %v", err)
	}
	if len(migrations) != 0 {
		t.Error("expected zero migrations for a missing directory")
	}
}

func TestCmdNew_CreatesSequentiallyNumberedFile(t *testing.T) {
	dir := t.TempDir()
	if err := cmdNew([]string{dir, "First Migration!"}); err != nil {
		t.Fatalf("cmdNew failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "migrations", "0001_first_migration_.yaml")); err != nil {
		t.Errorf("expected scaffolded file not found: %v", err)
	}

	if err := cmdNew([]string{dir, "second"}); err != nil {
		t.Fatalf("cmdNew failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "migrations", "0002_second.yaml")); err != nil {
		t.Errorf("expected second scaffolded file not found: %v", err)
	}
}

func TestCmdApply_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	def := samplePluginDefinition()
	pluginYamlPath := writePluginYAML(t, dir, def)

	writeMigration(t, dir, "0001_rename_and_bump.yaml", `schemaVersion: 1
description: "test migration"
operations:
  - renameInput: { from: threshold, to: pvalue_threshold }
  - addPackage: { name: "numpy>=1.26.0" }
`)

	if err := cmdApply([]string{dir}, false); err != nil {
		t.Fatalf("cmdApply failed: %v", err)
	}

	updated, err := loadPluginDefinition(pluginYamlPath)
	if err != nil {
		t.Fatalf("failed to reload migrated plugin.yaml: %v", err)
	}
	if updated.Plugin.SchemaVersion != 1 {
		t.Errorf("expected schemaVersion 1, got %d", updated.Plugin.SchemaVersion)
	}
	found := false
	for _, in := range updated.Inputs {
		if in.Name == "pvalue_threshold" {
			found = true
		}
	}
	if !found {
		t.Error("expected renamed input to persist after reload")
	}
}

func TestCmdApply_AlreadyUpToDateIsNoOp(t *testing.T) {
	dir := t.TempDir()
	def := samplePluginDefinition()
	def.Plugin.SchemaVersion = 1
	pluginYamlPath := writePluginYAML(t, dir, def)
	writeMigration(t, dir, "0001_noop.yaml", "schemaVersion: 1\noperations: []\n")

	before, _ := os.ReadFile(pluginYamlPath)
	if err := cmdApply([]string{dir}, false); err != nil {
		t.Fatalf("cmdApply failed: %v", err)
	}
	after, _ := os.ReadFile(pluginYamlPath)
	if string(before) != string(after) {
		t.Error("expected plugin.yaml to be untouched when already at latest schemaVersion")
	}
}

func TestCmdApply_DryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	def := samplePluginDefinition()
	pluginYamlPath := writePluginYAML(t, dir, def)
	writeMigration(t, dir, "0001_rename.yaml", "schemaVersion: 1\noperations:\n  - renameInput: { from: threshold, to: pvalue_threshold }\n")

	before, _ := os.ReadFile(pluginYamlPath)
	if err := cmdApply([]string{dir}, true); err != nil {
		t.Fatalf("cmdApply (dry run) failed: %v", err)
	}
	after, _ := os.ReadFile(pluginYamlPath)
	if string(before) != string(after) {
		t.Error("expected plugin.yaml to be untouched in dry-run mode")
	}
}

func TestCmdApply_FailurePartwayLeavesFileUnmodified(t *testing.T) {
	dir := t.TempDir()
	def := samplePluginDefinition()
	pluginYamlPath := writePluginYAML(t, dir, def)
	writeMigration(t, dir, "0001_bad.yaml", "schemaVersion: 1\noperations:\n  - renameInput: { from: does_not_exist, to: x }\n")

	before, _ := os.ReadFile(pluginYamlPath)
	if err := cmdApply([]string{dir}, false); err == nil {
		t.Fatal("expected cmdApply to fail for an operation referencing an unknown input")
	}
	after, _ := os.ReadFile(pluginYamlPath)
	if string(before) != string(after) {
		t.Error("expected plugin.yaml to be untouched after a failed migration")
	}
}

func TestCmdApply_SchemaVersionAheadOfMigrationsFails(t *testing.T) {
	dir := t.TempDir()
	def := samplePluginDefinition()
	def.Plugin.SchemaVersion = 5
	writePluginYAML(t, dir, def)

	err := cmdApply([]string{dir}, false)
	if err == nil || !strings.Contains(err.Error(), "only 0 migration file") {
		t.Errorf("expected a clear error about schemaVersion exceeding available migrations, got %v", err)
	}
}
