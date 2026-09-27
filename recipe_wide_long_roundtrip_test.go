package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/backend/services"
)

// installRealPlugin copies a real plugin's plugin.yaml from the repo's plugins/
// directory and compiles its cmd/<pluginID> entrypoint into pluginsRoot, so the
// test exercises the actual shipped plugin definition and binary rather than a
// throwaway stand-in.
func installRealPlugin(t *testing.T, pluginsRoot, pluginID string) {
	t.Helper()

	srcYAML, err := os.ReadFile(filepath.Join("plugins", pluginID, "plugin.yaml"))
	if err != nil {
		t.Fatalf("failed to read real plugin.yaml for %s: %v", pluginID, err)
	}

	destDir := filepath.Join(pluginsRoot, pluginID)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("failed to create plugin dir for %s: %v", pluginID, err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "plugin.yaml"), srcYAML, 0644); err != nil {
		t.Fatalf("failed to write plugin.yaml for %s: %v", pluginID, err)
	}

	binPath := filepath.Join(destDir, pluginID)
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/"+pluginID)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cmd/%s: %v\n%s", pluginID, err, out)
	}
}

// newWideLongRoundtripTestApp wires a fully-real App (real DB, real job queue,
// real chain/recipe services) with the actual wide-to-long and long-to-wide
// built-in plugins installed, for exercising a recipe that reshapes a table
// and reshapes it back, end-to-end through the real job queue.
func newWideLongRoundtripTestApp(t *testing.T) (*App, [2]*models.PluginV2) {
	t.Helper()
	tempDir := t.TempDir()

	db, err := services.NewDatabaseServiceV3(tempDir)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	settings := services.NewSettingsServiceV3(db)
	if err := settings.Set("outputDirectory", filepath.Join(tempDir, "outputs")); err != nil {
		t.Fatalf("failed to set outputDirectory: %v", err)
	}

	jobQueue := services.NewJobQueueServiceV3(db, nil)
	t.Cleanup(func() { jobQueue.Shutdown() })

	pluginsRoot := filepath.Join(tempDir, "plugins")
	installRealPlugin(t, pluginsRoot, "wide-to-long")
	installRealPlugin(t, pluginsRoot, "long-to-wide")

	pluginLoader := services.NewPluginLoaderV2(pluginsRoot, db, nil)
	if err := pluginLoader.LoadPlugins(); err != nil {
		t.Fatalf("Failed to load plugins: %v", err)
	}

	pluginExecutor := services.NewPluginExecutor()
	scriptExecutor := services.NewScriptExecutor(settings, db, "test")
	scriptExecutor.SetPluginLoader(pluginLoader)
	jobQueue.SetScriptExecutor(scriptExecutor)
	jobQueue.SetPluginLoader(pluginLoader)

	chainService := services.NewChainService(db, jobQueue, pluginLoader, pluginExecutor, settings)
	jobQueue.SetChainService(chainService)
	recipeService := services.NewRecipeService(db, pluginLoader, chainService)

	app := &App{
		db:             db,
		settings:       settings,
		jobQueue:       jobQueue,
		pluginLoaderV2: pluginLoader,
		pluginExecutor: pluginExecutor,
		scriptExecutor: scriptExecutor,
		chainService:   chainService,
		recipeService:  recipeService,
	}

	wideToLong, err := pluginLoader.GetPluginByStringID("wide-to-long")
	if err != nil {
		t.Fatalf("failed to load wide-to-long plugin: %v", err)
	}
	longToWide, err := pluginLoader.GetPluginByStringID("long-to-wide")
	if err != nil {
		t.Fatalf("failed to load long-to-wide plugin: %v", err)
	}

	return app, [2]*models.PluginV2{wideToLong, longToWide}
}

// TestRecipe_WideToLongThenLongToWide_RoundTripsToOriginalTable exercises a
// recipe chaining the two real built-in reshape plugins (melt then pivot,
// with the second stage's input bound to the first stage's output) and
// asserts the final table is byte-for-byte identical to the original input.
// This also covers the "map" args transform (used by both plugins' delimiter
// input) and the free-text fallback for column-selector inputs whose source
// file is bound rather than literal.
func TestRecipe_WideToLongThenLongToWide_RoundTripsToOriginalTable(t *testing.T) {
	app, plugins := newWideLongRoundtripTestApp(t)
	wideToLong, longToWide := plugins[0], plugins[1]

	original := "Protein.Group\tGenes\tSample1\tSample2\tSample3\n" +
		"P12345\tGENE1\t10.5\t11.2\t9.8\n" +
		"P67890\tGENE2\t20.1\t19.8\t21.0\n" +
		"P11111\tGENE3\t5.5\t6.1\t5.9\n"

	inputDir := t.TempDir()
	inputPath := filepath.Join(inputDir, "original.tsv")
	if err := os.WriteFile(inputPath, []byte(original), 0644); err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}

	recipe, err := app.recipeService.SaveRecipe("Wide-Long Round Trip", "", []services.RecipeStageSpec{
		{
			PluginID:      wideToLong.Definition.Plugin.ID,
			PluginVersion: wideToLong.Definition.Plugin.Version,
			Params: map[string]interface{}{
				"input_file": inputPath,
				"id_vars":    []string{"Protein.Group", "Genes"},
				"value_vars": []string{},
				"var_name":   "Sample",
				"value_name": "Intensity",
				"delimiter":  "tab",
			},
		},
		{
			PluginID:      longToWide.Definition.Plugin.ID,
			PluginVersion: longToWide.Definition.Plugin.Version,
			Params: map[string]interface{}{
				"id_vars":      []string{"Protein.Group", "Genes"},
				"names_from":   "Sample",
				"values_from":  "Intensity",
				"on_duplicate": "error",
				"delimiter":    "tab",
			},
			Bindings: map[string]models.RecipeStageBinding{
				"input_file": {Stage: 0, Output: "long_data"},
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	chain, err := app.recipeService.InstantiateChain(recipe.ID, nil)
	if err != nil {
		t.Fatalf("InstantiateChain failed: %v", err)
	}

	var finalStatus *services.ChainStatus
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		status, err := app.chainService.GetChainStatus(chain.ID)
		if err != nil {
			t.Fatalf("GetChainStatus failed: %v", err)
		}
		if status.Status == "completed" || status.Status == "failed" {
			finalStatus = status
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if finalStatus == nil {
		t.Fatal("chain did not reach a terminal state within 30s")
	}
	if finalStatus.Status != "completed" {
		for _, s := range finalStatus.Stages {
			errMsg := ""
			var terminal []string
			if s.Job != nil {
				errMsg = s.Job.Error
				terminal = s.Job.TerminalOutput
			}
			t.Logf("stage %d: status=%s error=%q output=%v", s.StageIndex, s.Status, errMsg, terminal)
		}
		t.Fatalf("expected chain to complete, got status %q", finalStatus.Status)
	}

	finalJob := finalStatus.Stages[1].Job
	if finalJob == nil {
		t.Fatal("second stage has no job")
	}

	finalContent, err := os.ReadFile(filepath.Join(finalJob.OutputPath, "pivoted.data.tsv"))
	if err != nil {
		t.Fatalf("failed to read final stage output: %v", err)
	}

	if string(finalContent) != original {
		t.Errorf("expected pivoted table to round-trip back to the original input, got:\n%s\nwant:\n%s", finalContent, original)
	}
}
