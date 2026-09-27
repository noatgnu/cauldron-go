package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/backend/services"
)

const recipeChainPluginScript = `import sys
import os

args = sys.argv[1:]
output_folder = None
input_file = None
for i, a in enumerate(args):
    if a == "--output_folder" and i + 1 < len(args):
        output_folder = args[i + 1]
    if a == "--input_file" and i + 1 < len(args):
        input_file = args[i + 1]

content = ""
if input_file:
    with open(input_file) as f:
        content = f.read().strip()

if output_folder:
    os.makedirs(output_folder, exist_ok=True)
    with open(os.path.join(output_folder, "%s"), "w") as f:
        f.write(content + "%s\n")

sys.exit(0)
`

func writeRecipeChainPlugin(t *testing.T, pluginsRoot, pluginID, yamlBody, outputFileName, suffix string) {
	t.Helper()
	pluginDir := filepath.Join(pluginsRoot, pluginID)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(yamlBody), 0644); err != nil {
		t.Fatalf("Failed to write plugin.yaml: %v", err)
	}
	script := fmt.Sprintf(recipeChainPluginScript, outputFileName, suffix)
	if err := os.WriteFile(filepath.Join(pluginDir, "main.py"), []byte(script), 0644); err != nil {
		t.Fatalf("Failed to write main.py: %v", err)
	}
}

// newRecipeTestApp builds a fully-wired App (real DB, real job queue, real script
// executor) with three chained throwaway plugins (A -> B -> C, each bound to the
// previous stage's declared output) suitable for exercising a recipe's real,
// end-to-end execution through the actual job queue and worker pool.
func newRecipeTestApp(t *testing.T) (*App, [3]*models.PluginV2) {
	t.Helper()
	tempDir := t.TempDir()

	db, err := services.NewDatabaseServiceV3(tempDir)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	settings := services.NewSettingsServiceV3(db)
	pythonPath, err := settings.DetectPythonPath()
	if err != nil || pythonPath == "" {
		t.Skip("python3 not available on this machine, skipping")
	}
	if err := settings.Set("pythonPath", pythonPath); err != nil {
		t.Fatalf("Failed to configure python path: %v", err)
	}

	jobQueue := services.NewJobQueueServiceV3(db, nil)
	t.Cleanup(func() { jobQueue.Shutdown() })

	pluginsRoot := filepath.Join(tempDir, "plugins")
	writeRecipeChainPlugin(t, pluginsRoot, "recipe-chain-a", `plugin:
  id: recipe-chain-a
  name: Recipe Chain A
  version: "1.0.0"
  description: writes out_a.tsv

runtime:
  environments: ["python"]
  entrypoint: main.py

execution:
  outputDir: "--output_folder"

outputs:
  - name: out_a
    path: out_a.tsv
    type: data
`, "out_a.tsv", "_from_a")

	writeRecipeChainPlugin(t, pluginsRoot, "recipe-chain-b", `plugin:
  id: recipe-chain-b
  name: Recipe Chain B
  version: "1.0.0"
  description: reads input_file, writes out_b.tsv

runtime:
  environments: ["python"]
  entrypoint: main.py

execution:
  outputDir: "--output_folder"
  argsMapping:
    input_file: "--input_file"

inputs:
  - name: input_file
    label: Input
    type: file
    required: true

outputs:
  - name: out_b
    path: out_b.tsv
    type: data
`, "out_b.tsv", "_from_b")

	writeRecipeChainPlugin(t, pluginsRoot, "recipe-chain-c", `plugin:
  id: recipe-chain-c
  name: Recipe Chain C
  version: "1.0.0"
  description: reads input_file, writes out_c.tsv

runtime:
  environments: ["python"]
  entrypoint: main.py

execution:
  outputDir: "--output_folder"
  argsMapping:
    input_file: "--input_file"

inputs:
  - name: input_file
    label: Input
    type: file
    required: true

outputs:
  - name: out_c
    path: out_c.tsv
    type: data
`, "out_c.tsv", "_from_c")

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

	pluginA, err := pluginLoader.GetPluginByStringID("recipe-chain-a")
	if err != nil {
		t.Fatalf("failed to load plugin A: %v", err)
	}
	pluginB, err := pluginLoader.GetPluginByStringID("recipe-chain-b")
	if err != nil {
		t.Fatalf("failed to load plugin B: %v", err)
	}
	pluginC, err := pluginLoader.GetPluginByStringID("recipe-chain-c")
	if err != nil {
		t.Fatalf("failed to load plugin C: %v", err)
	}

	return app, [3]*models.PluginV2{pluginA, pluginB, pluginC}
}

func TestRecipe_InstantiateChain_RunsThreeStagesEndToEnd(t *testing.T) {
	app, plugins := newRecipeTestApp(t)

	recipe, err := app.recipeService.SaveRecipe("E2E Recipe", "", []services.RecipeStageSpec{
		{PluginID: plugins[0].Definition.Plugin.ID, PluginVersion: plugins[0].Definition.Plugin.Version, Params: map[string]interface{}{}},
		{
			PluginID: plugins[1].Definition.Plugin.ID, PluginVersion: plugins[1].Definition.Plugin.Version,
			Params:   map[string]interface{}{},
			Bindings: map[string]models.RecipeStageBinding{"input_file": {Stage: 0, Output: "out_a"}},
		},
		{
			PluginID: plugins[2].Definition.Plugin.ID, PluginVersion: plugins[2].Definition.Plugin.Version,
			Params:   map[string]interface{}{},
			Bindings: map[string]models.RecipeStageBinding{"input_file": {Stage: 1, Output: "out_b"}},
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
			if s.Job != nil {
				errMsg = s.Job.Error
			}
			t.Logf("stage %d: status=%s error=%q", s.StageIndex, s.Status, errMsg)
		}
		t.Fatalf("expected chain to complete, got status %q", finalStatus.Status)
	}

	if len(finalStatus.Stages) != 3 {
		t.Fatalf("expected 3 stages, got %d", len(finalStatus.Stages))
	}

	finalContent, err := os.ReadFile(filepath.Join(finalStatus.Stages[2].Job.OutputPath, "out_c.tsv"))
	if err != nil {
		t.Fatalf("failed to read final stage output: %v", err)
	}
	expected := "_from_a_from_b_from_c\n"
	if string(finalContent) != expected {
		t.Errorf("expected final output %q (proving each stage's binding carried real content from the previous stage), got %q", expected, string(finalContent))
	}
}
