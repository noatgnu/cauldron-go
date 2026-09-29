package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func TestExportRecipeNextflow_TwoStageWiring(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	recipe, err := recipeService.SaveRecipe("Chain Recipe", "desc", testRecipeStages())
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	outDir := t.TempDir()
	if err := recipeService.ExportRecipeNextflow(recipe.ID, outDir); err != nil {
		t.Fatalf("ExportRecipeNextflow failed: %v", err)
	}

	upstreamModule := filepath.Join(outDir, "modules", "local", "chain-upstream", "main.nf")
	downstreamModule := filepath.Join(outDir, "modules", "local", "chain-downstream", "main.nf")
	if _, err := os.Stat(upstreamModule); err != nil {
		t.Errorf("expected upstream module file, got: %v", err)
	}
	if _, err := os.Stat(downstreamModule); err != nil {
		t.Errorf("expected downstream module file, got: %v", err)
	}

	mainContent, err := os.ReadFile(filepath.Join(outDir, "main.nf"))
	if err != nil {
		t.Fatalf("failed to read main.nf: %v", err)
	}
	main := string(mainContent)

	if !strings.Contains(main, "include { CHAIN_UPSTREAM as CHAIN_UPSTREAM_S0 } from './modules/local/chain-upstream/main'") {
		t.Errorf("expected aliased include for stage 0, got:\n%s", main)
	}
	if !strings.Contains(main, "include { CHAIN_DOWNSTREAM as CHAIN_DOWNSTREAM_S1 } from './modules/local/chain-downstream/main'") {
		t.Errorf("expected aliased include for stage 1, got:\n%s", main)
	}
	if !strings.Contains(main, "CHAIN_UPSTREAM_S0.out.result.collect()") {
		t.Errorf("expected stage 1 to reference stage 0's output channel, got:\n%s", main)
	}

	configContent, err := os.ReadFile(filepath.Join(outDir, "nextflow.config"))
	if err != nil {
		t.Fatalf("failed to read nextflow.config: %v", err)
	}
	config := string(configContent)
	if strings.Contains(config, "stage1_input_file") {
		t.Errorf("expected stage 1's bound input_file to be omitted from params (sourced from a channel), got:\n%s", config)
	}

	readme, err := os.ReadFile(filepath.Join(outDir, "README.md"))
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}
	if !strings.Contains(string(readme), "Chain Recipe") {
		t.Errorf("expected README to mention the recipe label, got:\n%s", readme)
	}
}

func TestExportRecipeNextflow_MissingPluginErrorsBeforeWriting(t *testing.T) {
	_, recipeService, pluginLoader := setupRecipeTestServices(t)

	recipe, err := recipeService.SaveRecipe("Chain Recipe", "desc", testRecipeStages())
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	pluginLoader.plugins = nil

	outDir := t.TempDir()
	if err := recipeService.ExportRecipeNextflow(recipe.ID, outDir); err == nil {
		t.Fatal("expected ExportRecipeNextflow to fail when a stage plugin is no longer installed")
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("failed to read output dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected nothing written when compatibility check fails, got: %v", entries)
	}
}

func TestExportRecipeNextflow_UnboundInputBecomesNamespacedParam(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	stages := []RecipeStageSpec{
		{
			PluginID:      "chain-downstream",
			PluginVersion: "1.0.0",
			Params:        map[string]interface{}{},
			Bindings:      map[string]models.RecipeStageBinding{},
		},
	}
	recipe, err := recipeService.SaveRecipe("Solo Recipe", "desc", stages)
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	outDir := t.TempDir()
	if err := recipeService.ExportRecipeNextflow(recipe.ID, outDir); err != nil {
		t.Fatalf("ExportRecipeNextflow failed: %v", err)
	}

	configContent, err := os.ReadFile(filepath.Join(outDir, "nextflow.config"))
	if err != nil {
		t.Fatalf("failed to read nextflow.config: %v", err)
	}
	if !strings.Contains(string(configContent), "stage0_input_file") {
		t.Errorf("expected stage 0's unbound input to become a namespaced param, got:\n%s", configContent)
	}
}

func TestExportRecipeNextflow_SamePluginTwiceSharesOneModule(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	stages := []RecipeStageSpec{
		{PluginID: "chain-upstream", PluginVersion: "1.0.0", Params: map[string]interface{}{}, Bindings: map[string]models.RecipeStageBinding{}},
		{PluginID: "chain-upstream", PluginVersion: "1.0.0", Params: map[string]interface{}{}, Bindings: map[string]models.RecipeStageBinding{}},
	}
	recipe, err := recipeService.SaveRecipe("Repeated Plugin Recipe", "desc", stages)
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	outDir := t.TempDir()
	if err := recipeService.ExportRecipeNextflow(recipe.ID, outDir); err != nil {
		t.Fatalf("ExportRecipeNextflow failed: %v", err)
	}

	moduleDir := filepath.Join(outDir, "modules", "local", "chain-upstream")
	entries, err := os.ReadDir(moduleDir)
	if err != nil {
		t.Fatalf("failed to read module dir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected exactly one shared module dir entry, got: %v", entries)
	}

	mainContent, err := os.ReadFile(filepath.Join(outDir, "main.nf"))
	if err != nil {
		t.Fatalf("failed to read main.nf: %v", err)
	}
	main := string(mainContent)
	if !strings.Contains(main, "CHAIN_UPSTREAM_S0") || !strings.Contains(main, "CHAIN_UPSTREAM_S1") {
		t.Errorf("expected two distinct call aliases, got:\n%s", main)
	}
	if strings.Count(main, "modules/local/chain-upstream/main") != 2 {
		t.Errorf("expected two include lines both pointing at the single shared module, got:\n%s", main)
	}
}
