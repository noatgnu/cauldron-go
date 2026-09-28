package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	if err := recipeService.ExportRecipe(recipe.ID, exportPath, false); err != nil {
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

const recipeTestInstallableYAML = `plugin:
  id: recipe-installable
  name: Recipe Installable
  version: "1.0.0"
  description: has repository and requirements set
  repository: https://github.com/example/recipe-installable

runtime:
  environments: ["python"]
  entrypoint: main.py

execution:
  requirements:
    packages: ["numpy", "pandas"]
`

func TestRecipeService_ExportRecipe_OmitsInstallInfoByDefault(t *testing.T) {
	tempDir := t.TempDir()
	_, recipeService, _ := setupRecipeTestServices(t)

	recipe, err := recipeService.SaveRecipe("No Install Info", "", testRecipeStages())
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	exportPath := filepath.Join(tempDir, "recipe.json")
	if err := recipeService.ExportRecipe(recipe.ID, exportPath, false); err != nil {
		t.Fatalf("ExportRecipe failed: %v", err)
	}

	raw, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}
	if strings.Contains(string(raw), "repository") || strings.Contains(string(raw), "commitHash") {
		t.Errorf("expected no install info fields in export, got: %s", raw)
	}
}

func TestRecipeService_ExportRecipe_IncludesLiveInstallInfoWhenRequested(t *testing.T) {
	tempDir := t.TempDir()
	db, err := newDatabaseServiceFromPath(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pluginsRoot := filepath.Join(tempDir, "plugins")
	writeChainTestPlugin(t, pluginsRoot, "recipe-installable", recipeTestInstallableYAML)
	pluginLoader := NewPluginLoaderV2(pluginsRoot, db, nil)
	if err := pluginLoader.LoadPlugins(); err != nil {
		t.Fatalf("LoadPlugins failed: %v", err)
	}
	recipeService := NewRecipeService(db, pluginLoader, nil)

	recipe, err := recipeService.SaveRecipe("Installable Recipe", "", []RecipeStageSpec{
		{PluginID: "recipe-installable", PluginVersion: "1.0.0", Params: map[string]interface{}{}, Bindings: map[string]models.RecipeStageBinding{}},
	})
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	exportPath := filepath.Join(tempDir, "recipe.json")
	if err := recipeService.ExportRecipe(recipe.ID, exportPath, true); err != nil {
		t.Fatalf("ExportRecipe failed: %v", err)
	}

	var data RecipeData
	raw, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("failed to parse export file: %v", err)
	}
	if len(data.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(data.Stages))
	}
	if data.Stages[0].Repository != "https://github.com/example/recipe-installable" {
		t.Errorf("expected repository to be carried through, got %q", data.Stages[0].Repository)
	}
	if data.Stages[0].Requirements == nil || len(data.Stages[0].Requirements.Packages) != 2 {
		t.Errorf("expected requirements to be carried through, got %+v", data.Stages[0].Requirements)
	}
}

func TestRecipeService_ExportRecipe_FallsBackToStoredInstallInfoWhenPluginMissing(t *testing.T) {
	tempDir := t.TempDir()
	_, recipeService, _ := setupRecipeTestServices(t)

	recipe, err := recipeService.SaveRecipe("Missing Plugin Recipe", "", []RecipeStageSpec{
		{
			PluginID:      "totally-not-installed",
			PluginVersion: "2.0.0",
			Params:        map[string]interface{}{},
			Bindings:      map[string]models.RecipeStageBinding{},
			Repository:    "https://github.com/example/missing-plugin",
			CommitHash:    "deadbeef",
			Requirements:  &models.Requirements{Packages: []string{"scipy"}},
		},
	})
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	exportPath := filepath.Join(tempDir, "recipe.json")
	if err := recipeService.ExportRecipe(recipe.ID, exportPath, true); err != nil {
		t.Fatalf("ExportRecipe failed: %v", err)
	}

	var data RecipeData
	raw, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("failed to parse export file: %v", err)
	}
	if data.Stages[0].Repository != "https://github.com/example/missing-plugin" {
		t.Errorf("expected stored repository to survive export, got %q", data.Stages[0].Repository)
	}
	if data.Stages[0].CommitHash != "deadbeef" {
		t.Errorf("expected stored commit hash to survive export, got %q", data.Stages[0].CommitHash)
	}
	if data.Stages[0].Requirements == nil || len(data.Stages[0].Requirements.Packages) != 1 || data.Stages[0].Requirements.Packages[0] != "scipy" {
		t.Errorf("expected stored requirements to survive export, got %+v", data.Stages[0].Requirements)
	}
}

func TestRecipeService_ImportRecipeFromFile_PersistsInstallInfoAndSurfacesItOnMissingCompatibility(t *testing.T) {
	tempDir := t.TempDir()
	_, recipeService, _ := setupRecipeTestServices(t)

	source, err := recipeService.SaveRecipe("Source Recipe", "", []RecipeStageSpec{
		{
			PluginID:      "still-not-installed",
			PluginVersion: "3.0.0",
			Params:        map[string]interface{}{},
			Bindings:      map[string]models.RecipeStageBinding{},
			Repository:    "https://github.com/example/still-not-installed",
			CommitHash:    "cafebabe",
		},
	})
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	exportPath := filepath.Join(tempDir, "recipe.json")
	if err := recipeService.ExportRecipe(source.ID, exportPath, true); err != nil {
		t.Fatalf("ExportRecipe failed: %v", err)
	}

	result, err := recipeService.ImportRecipeFromFile(exportPath)
	if err != nil {
		t.Fatalf("ImportRecipeFromFile failed: %v", err)
	}

	stages, err := recipeService.GetRecipeStages(result.Recipe.ID)
	if err != nil {
		t.Fatalf("GetRecipeStages failed: %v", err)
	}
	if stages[0].Repository != "https://github.com/example/still-not-installed" || stages[0].CommitHash != "cafebabe" {
		t.Errorf("expected imported stage to persist install info, got repository=%q commitHash=%q", stages[0].Repository, stages[0].CommitHash)
	}

	if result.Compatibility.AllOK {
		t.Fatal("expected the imported recipe to be incompatible since its plugin isn't installed")
	}
	missing := result.Compatibility.Stages[0]
	if missing.Status != CompatibilityMissing {
		t.Fatalf("expected stage 0 status %q, got %q", CompatibilityMissing, missing.Status)
	}
	if missing.Repository != "https://github.com/example/still-not-installed" || missing.CommitHash != "cafebabe" {
		t.Errorf("expected CheckCompatibility to surface the stored install info on a missing stage, got repository=%q commitHash=%q", missing.Repository, missing.CommitHash)
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

func seedTestChain(t *testing.T, db *DatabaseService, pluginLoader *PluginLoaderV2) *models.JobChain {
	t.Helper()

	upstream, err := pluginLoader.GetPluginByStringID("chain-upstream")
	if err != nil {
		t.Fatalf("failed to look up chain-upstream: %v", err)
	}
	downstream, err := pluginLoader.GetPluginByStringID("chain-downstream")
	if err != nil {
		t.Fatalf("failed to look up chain-downstream: %v", err)
	}

	chain := &models.JobChain{ID: "chain-1", Label: "My Chain Run", CreatedAt: time.Now()}
	if err := db.GetDB().Create(chain).Error; err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}

	stages := []models.JobChainStage{
		{
			ChainID:       chain.ID,
			StageIndex:    0,
			PluginID:      upstream.ID,
			PluginVersion: "1.0.0",
			Params:        models.JSONMap{},
			Bindings:      models.JSONMap{},
		},
		{
			ChainID:       chain.ID,
			StageIndex:    1,
			PluginID:      downstream.ID,
			PluginVersion: "1.0.0",
			Params:        models.JSONMap{},
			Bindings: models.JSONMap{
				"input_file": map[string]interface{}{"stage": float64(0), "output": "result"},
			},
		},
	}
	for i := range stages {
		if err := db.GetDB().Create(&stages[i]).Error; err != nil {
			t.Fatalf("failed to create chain stage %d: %v", i, err)
		}
	}

	return chain
}

func TestRecipeService_CreateRecipeFromChain(t *testing.T) {
	db, recipeService, pluginLoader := setupRecipeTestServices(t)
	chain := seedTestChain(t, db, pluginLoader)

	recipe, err := recipeService.CreateRecipeFromChain(chain.ID, "Recovered Recipe", "recovered from a past run")
	if err != nil {
		t.Fatalf("CreateRecipeFromChain failed: %v", err)
	}
	if recipe.Label != "Recovered Recipe" {
		t.Errorf("expected label %q, got %q", "Recovered Recipe", recipe.Label)
	}

	stages, err := recipeService.GetRecipeStages(recipe.ID)
	if err != nil {
		t.Fatalf("GetRecipeStages failed: %v", err)
	}
	if len(stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(stages))
	}
	if stages[0].PluginID != "chain-upstream" {
		t.Errorf("expected stage 0 plugin id %q, got %q", "chain-upstream", stages[0].PluginID)
	}
	if stages[1].PluginID != "chain-downstream" {
		t.Errorf("expected stage 1 plugin id %q, got %q", "chain-downstream", stages[1].PluginID)
	}
	binding, ok := stages[1].Bindings["input_file"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected stage 1 to carry the input_file binding, got %#v", stages[1].Bindings)
	}
	if outputName, _ := binding["output"].(string); outputName != "result" {
		t.Errorf("expected binding output %q, got %q", "result", outputName)
	}
}

func TestRecipeService_CreateRecipeFromChain_DefaultsLabelToChainLabel(t *testing.T) {
	db, recipeService, pluginLoader := setupRecipeTestServices(t)
	chain := seedTestChain(t, db, pluginLoader)

	recipe, err := recipeService.CreateRecipeFromChain(chain.ID, "", "")
	if err != nil {
		t.Fatalf("CreateRecipeFromChain failed: %v", err)
	}
	if recipe.Label != chain.Label {
		t.Errorf("expected label to default to chain label %q, got %q", chain.Label, recipe.Label)
	}
}

func TestRecipeService_CreateRecipeFromChain_UninstalledPlugin(t *testing.T) {
	db, recipeService, pluginLoader := setupRecipeTestServices(t)
	chain := seedTestChain(t, db, pluginLoader)

	pluginLoader.plugins = map[uint]*models.PluginV2{}

	if _, err := recipeService.CreateRecipeFromChain(chain.ID, "Recovered", ""); err == nil {
		t.Fatal("expected an error when a stage's plugin is no longer installed")
	}
}

func TestRecipeService_CreateRecipeFromChain_ChainNotFound(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	if _, err := recipeService.CreateRecipeFromChain("does-not-exist", "Recovered", ""); err == nil {
		t.Fatal("expected an error for a chain that doesn't exist")
	}
}
