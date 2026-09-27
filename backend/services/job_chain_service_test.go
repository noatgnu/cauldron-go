package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func writeChainTestPlugin(t *testing.T, pluginsRoot, pluginID, yamlBody string) {
	t.Helper()
	pluginDir := filepath.Join(pluginsRoot, pluginID)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(yamlBody), 0644); err != nil {
		t.Fatalf("failed to write plugin.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "main.py"), []byte("import sys\nsys.exit(0)\n"), 0644); err != nil {
		t.Fatalf("failed to write main.py: %v", err)
	}
}

const chainTestUpstreamYAML = `plugin:
  id: chain-upstream
  name: Chain Upstream
  version: "1.0.0"
  description: upstream test plugin

runtime:
  environments: ["python"]
  entrypoint: main.py

outputs:
  - name: result
    path: result.tsv
    type: data
`

const chainTestDownstreamYAML = `plugin:
  id: chain-downstream
  name: Chain Downstream
  version: "1.0.0"
  description: downstream test plugin

runtime:
  environments: ["python"]
  entrypoint: main.py

inputs:
  - name: input_file
    label: Input
    type: file
    required: true
`

func setupChainTestServices(t *testing.T) (*DatabaseService, *JobQueueService, *ChainService, *models.PluginV2, *models.PluginV2) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	jobQueue := NewJobQueueServiceV3(db, nil)
	t.Cleanup(func() { jobQueue.Shutdown() })

	pluginsRoot := filepath.Join(tempDir, "plugins")
	writeChainTestPlugin(t, pluginsRoot, "chain-upstream", chainTestUpstreamYAML)
	writeChainTestPlugin(t, pluginsRoot, "chain-downstream", chainTestDownstreamYAML)

	pluginLoader := NewPluginLoaderV2(pluginsRoot, db, nil)
	if err := pluginLoader.LoadPlugins(); err != nil {
		t.Fatalf("LoadPlugins failed: %v", err)
	}
	jobQueue.SetPluginLoader(pluginLoader)

	upstream, err := pluginLoader.GetPluginByStringID("chain-upstream")
	if err != nil {
		t.Fatalf("failed to load upstream plugin: %v", err)
	}
	downstream, err := pluginLoader.GetPluginByStringID("chain-downstream")
	if err != nil {
		t.Fatalf("failed to load downstream plugin: %v", err)
	}

	settings := NewSettingsService(context.WithValue(context.Background(), "wails-test", true), db)
	pluginExecutor := NewPluginExecutor()
	chainService := NewChainService(db, jobQueue, pluginLoader, pluginExecutor, settings)

	return db, jobQueue, chainService, upstream, downstream
}

func seedChainWithUpstreamJob(t *testing.T, db *DatabaseService, upstream, downstream *models.PluginV2, upstreamStatus models.JobStatus) (*models.JobChain, *models.Job) {
	t.Helper()

	chain := &models.JobChain{ID: "test-chain-" + string(upstreamStatus), Label: "Test Chain", CreatedAt: time.Now()}
	if err := db.GetDB().Create(chain).Error; err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}

	upstreamOutputDir := t.TempDir()
	if upstreamStatus == models.JobStatusCompleted {
		if err := os.WriteFile(filepath.Join(upstreamOutputDir, "result.tsv"), []byte("protein\tvalue\nP1\t1.0\n"), 0644); err != nil {
			t.Fatalf("failed to write fake upstream output: %v", err)
		}
	}

	upstreamJob := &models.Job{
		ID:              chain.ID + "-upstream-job",
		Type:            "chain-upstream",
		Name:            "Upstream",
		Status:          upstreamStatus,
		Args:            []string{},
		OutputPath:      upstreamOutputDir,
		ChainID:         chain.ID,
		ChainStageIndex: 0,
		CreatedAt:       time.Now(),
	}
	if err := db.GetDB().Create(upstreamJob).Error; err != nil {
		t.Fatalf("failed to seed upstream job: %v", err)
	}

	upstreamStage := &models.JobChainStage{
		ChainID:       chain.ID,
		StageIndex:    0,
		PluginID:      upstream.ID,
		PluginVersion: upstream.Definition.Plugin.Version,
		Params:        models.JSONMap{},
		Bindings:      models.JSONMap{},
		JobID:         upstreamJob.ID,
	}
	if err := db.GetDB().Create(upstreamStage).Error; err != nil {
		t.Fatalf("failed to seed upstream stage: %v", err)
	}

	downstreamStage := &models.JobChainStage{
		ChainID:       chain.ID,
		StageIndex:    1,
		PluginID:      downstream.ID,
		PluginVersion: downstream.Definition.Plugin.Version,
		Params:        models.JSONMap{},
		Bindings: models.JSONMap{
			"input_file": map[string]interface{}{"stage": float64(0), "output": "result"},
		},
	}
	if err := db.GetDB().Create(downstreamStage).Error; err != nil {
		t.Fatalf("failed to seed downstream stage: %v", err)
	}

	return chain, upstreamJob
}

func TestChainService_OnJobTerminal_AdvancesAndResolvesBinding(t *testing.T) {
	db, jobQueue, chainService, upstream, downstream := setupChainTestServices(t)
	chain, upstreamJob := seedChainWithUpstreamJob(t, db, upstream, downstream, models.JobStatusCompleted)

	chainService.OnJobTerminal(upstreamJob)

	var advancedStage models.JobChainStage
	if err := db.GetDB().Where("chain_id = ? AND stage_index = ?", chain.ID, 1).First(&advancedStage).Error; err != nil {
		t.Fatalf("failed to reload downstream stage: %v", err)
	}
	if advancedStage.JobID == "" {
		t.Fatal("expected downstream stage to have a job created after OnJobTerminal")
	}

	downstreamJob, err := jobQueue.GetJob(advancedStage.JobID)
	if err != nil {
		t.Fatalf("failed to load downstream job: %v", err)
	}
	expectedPath := filepath.Join(upstreamJob.OutputPath, "result.tsv")
	gotPath, _ := downstreamJob.Parameters["input_file"].(string)
	if gotPath != expectedPath {
		t.Errorf("expected downstream job's input_file to resolve to %q, got %q", expectedPath, gotPath)
	}
}

func TestChainService_OnJobTerminal_DoesNotAdvanceOnFailure(t *testing.T) {
	db, _, chainService, upstream, downstream := setupChainTestServices(t)
	chain, upstreamJob := seedChainWithUpstreamJob(t, db, upstream, downstream, models.JobStatusFailed)

	chainService.OnJobTerminal(upstreamJob)

	var advancedStage models.JobChainStage
	if err := db.GetDB().Where("chain_id = ? AND stage_index = ?", chain.ID, 1).First(&advancedStage).Error; err != nil {
		t.Fatalf("failed to reload downstream stage: %v", err)
	}
	if advancedStage.JobID != "" {
		t.Error("expected downstream stage to remain unstarted after upstream failure")
	}

	status, err := chainService.GetChainStatus(chain.ID)
	if err != nil {
		t.Fatalf("GetChainStatus failed: %v", err)
	}
	if status.Status != "failed" {
		t.Errorf("expected chain status %q, got %q", "failed", status.Status)
	}
	if len(status.Stages) != 2 || status.Stages[1].Status != "blocked" {
		t.Errorf("expected stage 1 to be reported as blocked, got: %+v", status.Stages)
	}
}

func TestChainService_DeleteChain_CascadesJobs(t *testing.T) {
	db, jobQueue, chainService, upstream, downstream := setupChainTestServices(t)
	chain, upstreamJob := seedChainWithUpstreamJob(t, db, upstream, downstream, models.JobStatusCompleted)
	chainService.OnJobTerminal(upstreamJob)

	if err := chainService.DeleteChain(chain.ID); err != nil {
		t.Fatalf("DeleteChain failed: %v", err)
	}

	if _, err := jobQueue.GetJob(upstreamJob.ID); err == nil {
		t.Error("expected upstream job to be deleted")
	}
	if _, err := chainService.GetChain(chain.ID); err == nil {
		t.Error("expected the chain row itself to be deleted")
	}
	var remainingStages []models.JobChainStage
	db.GetDB().Where("chain_id = ?", chain.ID).Find(&remainingStages)
	if len(remainingStages) != 0 {
		t.Errorf("expected all stage rows to be deleted, got %d remaining", len(remainingStages))
	}
}
