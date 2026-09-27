package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func setupRecipeTestServices(t *testing.T) (*DatabaseService, *RecipeService, *PluginLoaderV2) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pluginsRoot := filepath.Join(tempDir, "plugins")
	writeChainTestPlugin(t, pluginsRoot, "chain-upstream", chainTestUpstreamYAML)
	writeChainTestPlugin(t, pluginsRoot, "chain-downstream", chainTestDownstreamYAML)

	pluginLoader := NewPluginLoaderV2(pluginsRoot, db, nil)
	if err := pluginLoader.LoadPlugins(); err != nil {
		t.Fatalf("LoadPlugins failed: %v", err)
	}

	recipeService := NewRecipeService(db, pluginLoader, nil)
	return db, recipeService, pluginLoader
}

func testRecipeStages() []RecipeStageSpec {
	return []RecipeStageSpec{
		{
			PluginID:      "chain-upstream",
			PluginVersion: "1.0.0",
			Params:        map[string]interface{}{},
			Bindings:      map[string]models.RecipeStageBinding{},
		},
		{
			PluginID:      "chain-downstream",
			PluginVersion: "1.0.0",
			Params:        map[string]interface{}{},
			Bindings: map[string]models.RecipeStageBinding{
				"input_file": {Stage: 0, Output: "result"},
			},
		},
	}
}

func TestRecipeService_SaveAndFetch(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	recipe, err := recipeService.SaveRecipe("My Recipe", "A test recipe", testRecipeStages())
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}
	if recipe.ID == "" {
		t.Fatal("expected SaveRecipe to generate a non-empty ID")
	}

	stages, err := recipeService.GetRecipeStages(recipe.ID)
	if err != nil {
		t.Fatalf("GetRecipeStages failed: %v", err)
	}
	if len(stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(stages))
	}
	if stages[0].PluginID != "chain-upstream" || stages[1].PluginID != "chain-downstream" {
		t.Errorf("unexpected stage plugin IDs: %q, %q", stages[0].PluginID, stages[1].PluginID)
	}
}

func TestRecipeService_ExportImportRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	_, recipeService, _ := setupRecipeTestServices(t)

	recipe, err := recipeService.SaveRecipe("Round Trip Recipe", "desc", testRecipeStages())
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	exportPath := filepath.Join(tempDir, "recipe.json")
	if err := recipeService.ExportRecipe(recipe.ID, exportPath); err != nil {
		t.Fatalf("ExportRecipe failed: %v", err)
	}
	if _, err := os.Stat(exportPath); err != nil {
		t.Fatalf("expected export file to exist: %v", err)
	}

	if err := recipeService.DeleteRecipe(recipe.ID); err != nil {
		t.Fatalf("DeleteRecipe failed: %v", err)
	}

	result, err := recipeService.ImportRecipeFromFile(exportPath)
	if err != nil {
		t.Fatalf("ImportRecipeFromFile failed: %v", err)
	}
	if result.Recipe.Label != "Round Trip Recipe" || result.Recipe.Description != "desc" {
		t.Errorf("imported recipe metadata does not match: %+v", result.Recipe)
	}

	stages, err := recipeService.GetRecipeStages(result.Recipe.ID)
	if err != nil {
		t.Fatalf("GetRecipeStages failed: %v", err)
	}
	if len(stages) != 2 {
		t.Fatalf("expected 2 imported stages, got %d", len(stages))
	}
	bindingRaw, ok := stages[1].Bindings["input_file"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected imported binding to round-trip, got: %+v", stages[1].Bindings)
	}
	if outputName, _ := bindingRaw["output"].(string); outputName != "result" {
		t.Errorf("expected imported binding output %q, got %q", "result", outputName)
	}

	if !result.Compatibility.AllOK {
		t.Errorf("expected the freshly-imported recipe to be compatible, got: %+v", result.Compatibility.Stages)
	}
}

func TestRecipeService_CheckCompatibility_Compatible(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	recipe, err := recipeService.SaveRecipe("Compatible Recipe", "", testRecipeStages())
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	report, err := recipeService.CheckCompatibility(recipe.ID)
	if err != nil {
		t.Fatalf("CheckCompatibility failed: %v", err)
	}
	if !report.AllOK {
		t.Fatalf("expected all stages compatible, got: %+v", report.Stages)
	}
	for _, s := range report.Stages {
		if s.Status != CompatibilityCompatible {
			t.Errorf("expected stage %d status %q, got %q", s.StageIndex, CompatibilityCompatible, s.Status)
		}
	}
}

func TestRecipeService_CheckCompatibility_Missing(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	stages := testRecipeStages()
	stages[0].PluginID = "does-not-exist"

	recipe, err := recipeService.SaveRecipe("Missing Plugin Recipe", "", stages)
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	report, err := recipeService.CheckCompatibility(recipe.ID)
	if err != nil {
		t.Fatalf("CheckCompatibility failed: %v", err)
	}
	if report.AllOK {
		t.Fatal("expected AllOK to be false when a stage's plugin is not installed")
	}
	if report.Stages[0].Status != CompatibilityMissing {
		t.Errorf("expected stage 0 status %q, got %q", CompatibilityMissing, report.Stages[0].Status)
	}
}

func TestRecipeService_CheckCompatibility_VersionDiffersButCompatible(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	stages := testRecipeStages()
	stages[0].PluginVersion = "0.0.1"

	recipe, err := recipeService.SaveRecipe("Version Differs Recipe", "", stages)
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	report, err := recipeService.CheckCompatibility(recipe.ID)
	if err != nil {
		t.Fatalf("CheckCompatibility failed: %v", err)
	}
	if !report.AllOK {
		t.Fatalf("expected AllOK true when the version differs but all inputs/outputs still exist, got: %+v", report.Stages)
	}
	if report.Stages[0].Status != CompatibilityCompatibleVersionDiffers {
		t.Errorf("expected stage 0 status %q, got %q", CompatibilityCompatibleVersionDiffers, report.Stages[0].Status)
	}
}

func TestRecipeService_CheckCompatibility_IncompatibleMissingInput(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	stages := testRecipeStages()
	stages[1].PluginVersion = "0.0.1"
	stages[1].Params = map[string]interface{}{"no_longer_exists": "value"}

	recipe, err := recipeService.SaveRecipe("Incompatible Recipe", "", stages)
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	report, err := recipeService.CheckCompatibility(recipe.ID)
	if err != nil {
		t.Fatalf("CheckCompatibility failed: %v", err)
	}
	if report.AllOK {
		t.Fatal("expected AllOK to be false when a stage references a param the installed plugin no longer has")
	}
	if report.Stages[1].Status != CompatibilityIncompatible {
		t.Errorf("expected stage 1 status %q, got %q", CompatibilityIncompatible, report.Stages[1].Status)
	}
	found := false
	for _, name := range report.Stages[1].MissingInputs {
		if name == "no_longer_exists" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing input %q to be reported, got: %v", "no_longer_exists", report.Stages[1].MissingInputs)
	}
}

func TestRecipeService_DeleteRecipe_RemovesStages(t *testing.T) {
	db, recipeService, _ := setupRecipeTestServices(t)

	recipe, err := recipeService.SaveRecipe("Delete Me", "", testRecipeStages())
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	if err := recipeService.DeleteRecipe(recipe.ID); err != nil {
		t.Fatalf("DeleteRecipe failed: %v", err)
	}

	if _, err := recipeService.GetRecipe(recipe.ID); err == nil {
		t.Error("expected the recipe row to be deleted")
	}
	var remaining []models.RecipeStage
	db.GetDB().Where("recipe_id = ?", recipe.ID).Find(&remaining)
	if len(remaining) != 0 {
		t.Errorf("expected all stage rows to be deleted, got %d remaining", len(remaining))
	}
}
