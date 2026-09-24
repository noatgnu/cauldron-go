package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/backend/services"
)

// newBatchTestApp builds a minimal, fully-local App (real DB, real job queue, a
// throwaway plugin written to a temp dir) suitable for exercising
// App.ExecutePluginBatchV2 without depending on any pre-installed plugin.
func newBatchTestApp(t *testing.T, scriptBody string) (*App, *models.PluginV2) {
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

	batchService := services.NewBatchServiceV3(db, jobQueue)

	pluginsDir := filepath.Join(tempDir, "plugins", "batch-test-plugin")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin dir: %v", err)
	}
	pluginYAML := `plugin:
  id: batch-test-plugin
  name: Batch Test Plugin
  version: "1.0.0"
  description: Minimal plugin for batch execution tests

runtime:
  environments:
    - python
  entrypoint: main.py
`
	if err := os.WriteFile(filepath.Join(pluginsDir, "plugin.yaml"), []byte(pluginYAML), 0644); err != nil {
		t.Fatalf("Failed to write plugin.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "main.py"), []byte(scriptBody), 0644); err != nil {
		t.Fatalf("Failed to write main.py: %v", err)
	}

	pluginLoader := services.NewPluginLoaderV2(filepath.Join(tempDir, "plugins"), db, nil)
	if err := pluginLoader.LoadPlugins(); err != nil {
		t.Fatalf("Failed to load plugins: %v", err)
	}
	jobQueue.SetPluginLoader(pluginLoader)

	plugins := pluginLoader.GetAllPlugins()
	if len(plugins) == 0 {
		t.Fatal("No test plugin loaded")
	}
	plugin := plugins[0]

	app := &App{
		db:             db,
		settings:       settings,
		jobQueue:       jobQueue,
		batchService:   batchService,
		pluginLoaderV2: pluginLoader,
		pluginExecutor: services.NewPluginExecutor(),
	}

	return app, plugin
}

func TestExecutePluginBatchV2_RejectsEmptyJobsList(t *testing.T) {
	app, plugin := newBatchTestApp(t, "import sys\nsys.exit(0)\n")

	_, err := app.ExecutePluginBatchV2(models.PluginBatchExecutionRequestV2{
		PluginID: plugin.ID,
		Jobs:     []map[string]interface{}{},
	})
	if err == nil {
		t.Fatal("expected an error for an empty jobs list, got nil")
	}
}

func TestExecutePluginBatchV2_CreatesOneJobPerEntryAllSharingBatchID(t *testing.T) {
	app, plugin := newBatchTestApp(t, "import sys\nsys.exit(0)\n")

	batchID, err := app.ExecutePluginBatchV2(models.PluginBatchExecutionRequestV2{
		PluginID: plugin.ID,
		Label:    "Test Batch",
		Jobs: []map[string]interface{}{
			{"pluginId": plugin.ID},
			{"pluginId": plugin.ID},
			{"pluginId": plugin.ID},
		},
	})
	if err != nil {
		t.Fatalf("ExecutePluginBatchV2 failed: %v", err)
	}
	if batchID == "" {
		t.Fatal("expected a non-empty batch ID")
	}

	batch, err := app.GetJobBatch(batchID)
	if err != nil {
		t.Fatalf("GetJobBatch failed: %v", err)
	}
	if batch.Label != "Test Batch" || batch.ExpectedCount != 3 {
		t.Errorf("unexpected batch metadata: %+v", batch)
	}

	jobs := app.jobQueue.GetJobsByBatchID(batchID)
	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs in the batch, got %d", len(jobs))
	}
	for _, j := range jobs {
		if j.BatchID != batchID {
			t.Errorf("job %s has BatchID %q, want %q", j.ID, j.BatchID, batchID)
		}
	}
}

func TestExecutePluginBatchV2_JobsHaveIndependentParameters(t *testing.T) {
	app, plugin := newBatchTestApp(t, "import sys\nsys.exit(0)\n")

	batchID, err := app.ExecutePluginBatchV2(models.PluginBatchExecutionRequestV2{
		PluginID: plugin.ID,
		Jobs: []map[string]interface{}{
			{"pluginId": plugin.ID, "annotation_file": "a.tsv"},
			{"pluginId": plugin.ID, "annotation_file": "b.tsv"},
		},
	})
	if err != nil {
		t.Fatalf("ExecutePluginBatchV2 failed: %v", err)
	}

	jobs := app.jobQueue.GetJobsByBatchID(batchID)
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}

	seen := map[string]bool{}
	for _, j := range jobs {
		val, _ := j.Parameters["annotation_file"].(string)
		seen[val] = true
	}
	if !seen["a.tsv"] || !seen["b.tsv"] {
		t.Errorf("expected one job with annotation_file=a.tsv and one with annotation_file=b.tsv, got parameters: %v, %v",
			jobs[0].Parameters, jobs[1].Parameters)
	}
}

// TestExecutePluginBatchV2_JobsGoThroughPluginV2Path is a regression guard for the
// processJob dual uint/float64 type assertion on Parameters["pluginId"]: it lets a
// batch-created job actually run to completion, confirming the real plugin-v2 script
// execution path is taken (not silently skipped) for jobs created via the batch loop.
func TestExecutePluginBatchV2_JobsGoThroughPluginV2Path(t *testing.T) {
	app, plugin := newBatchTestApp(t, "import sys\nsys.exit(0)\n")

	batchID, err := app.ExecutePluginBatchV2(models.PluginBatchExecutionRequestV2{
		PluginID: plugin.ID,
		Jobs: []map[string]interface{}{
			{"pluginId": plugin.ID},
		},
	})
	if err != nil {
		t.Fatalf("ExecutePluginBatchV2 failed: %v", err)
	}

	jobs := app.jobQueue.GetJobsByBatchID(batchID)
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	jobID := jobs[0].ID

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		job, err := app.jobQueue.GetJob(jobID)
		if err != nil {
			t.Fatalf("GetJob failed: %v", err)
		}
		if job.Status == models.JobStatusCompleted {
			return
		}
		if job.Status == models.JobStatusFailed {
			t.Fatalf("job failed instead of completing: %s", job.Error)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("job did not complete within 10s; likely stuck because processJob's pluginId type assertion failed and the real script path was skipped")
}
