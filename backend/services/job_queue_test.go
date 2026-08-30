package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func TestJobQueueServiceV3HasValidContext(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	if jobQueue.ctx == nil {
		t.Fatal("JobQueueService ctx is nil - this will cause panic in processJob")
	}

	t.Log("JobQueueService has valid context")
}

func TestJobQueueContextWithCancel(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("context.WithCancel panicked: %v", r)
		}
	}()

	jobCtx, cancel := context.WithCancel(jobQueue.ctx)
	defer cancel()

	if jobCtx == nil {
		t.Fatal("jobCtx is nil after WithCancel")
	}

	t.Log("context.WithCancel works correctly with JobQueueService context")
}

func TestProcessJobWithNilWailsApp(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	settingsService := newSettingsServiceInternal(db)
	scriptExecutor := NewScriptExecutor(settingsService, db)
	jobQueue.SetScriptExecutor(scriptExecutor)

	jobID, err := jobQueue.CreateJob("test", "Test Job", "", []string{})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	job, err := jobQueue.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.Status != models.JobStatusCompleted {
		t.Logf("Job status: %s, error: %s", job.Status, job.Error)
	}

	t.Logf("Job processed without crash, status: %s", job.Status)
}

func TestProcessPluginJobWithContext(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	settingsService := newSettingsServiceInternal(db)
	scriptExecutor := NewScriptExecutor(settingsService, db)
	jobQueue.SetScriptExecutor(scriptExecutor)

	pluginsDir := filepath.Join(tempDir, "plugins", "test-plugin")
	os.MkdirAll(pluginsDir, 0755)

	pluginYAML := `plugin:
  id: test-plugin
  name: Test Plugin
  version: "1.0.0"
  description: Test plugin for automated testing

runtime:
  environments:
    - python
  entrypoint: main.py

inputs:
  - name: input_file
    type: file
    label: Input File
    required: true
`
	os.WriteFile(filepath.Join(pluginsDir, "plugin.yaml"), []byte(pluginYAML), 0644)

	mainPy := `import sys
print("Test plugin executed")
sys.exit(0)
`
	os.WriteFile(filepath.Join(pluginsDir, "main.py"), []byte(mainPy), 0644)

	pluginLoader := NewPluginLoaderV2(filepath.Join(tempDir, "plugins"), db, nil)
	if err := pluginLoader.LoadPlugins(); err != nil {
		t.Fatalf("Failed to load plugins: %v", err)
	}
	jobQueue.SetPluginLoader(pluginLoader)

	plugins := pluginLoader.GetAllPlugins()
	if len(plugins) == 0 {
		t.Fatal("No test plugin loaded")
	}

	plugin := plugins[0]
	t.Logf("Test plugin loaded: %s (ID: %d)", plugin.Definition.Plugin.Name, plugin.ID)

	outputDir := filepath.Join(tempDir, "output")
	os.MkdirAll(outputDir, 0755)

	parameters := map[string]interface{}{
		"pluginId":  plugin.ID,
		"outputDir": outputDir,
	}

	jobID, err := jobQueue.CreateJob(
		plugin.Definition.Plugin.ID,
		plugin.Definition.Plugin.Name,
		"",
		[]string{"main.py"},
	)
	if err != nil {
		t.Fatalf("Failed to create plugin job: %v", err)
	}

	job, _ := jobQueue.GetJob(jobID)
	job.Parameters = parameters
	db.GetDB().Save(job)

	time.Sleep(500 * time.Millisecond)

	job, err = jobQueue.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	t.Logf("Plugin job processed, status: %s, error: %s", job.Status, job.Error)
}

func TestJobQueueEmitEventWithNilWailsApp(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("emitEvent panicked with nil wailsApp: %v", r)
		}
	}()

	jobQueue.emitEvent("test:event", map[string]string{"test": "data"})

	t.Log("emitEvent with nil wailsApp did not panic")
}

func TestNewJobQueueServiceWithNilContext(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueService(nil, db)
	defer jobQueue.Shutdown()

	if jobQueue.ctx == nil {
		t.Fatal("JobQueueService ctx should default to Background() when nil is passed")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("context.WithCancel panicked with defaulted context: %v", r)
		}
	}()

	jobCtx, cancel := context.WithCancel(jobQueue.ctx)
	defer cancel()

	if jobCtx == nil {
		t.Fatal("jobCtx should not be nil")
	}

	t.Log("NewJobQueueService correctly defaults nil context to Background()")
}

func TestProcessPluginV2JobContextNotNil(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	settingsService := newSettingsServiceInternal(db)
	scriptExecutor := NewScriptExecutor(settingsService, db)
	jobQueue.SetScriptExecutor(scriptExecutor)

	pluginLoader := NewPluginLoaderV2(filepath.Join(tempDir, "plugins"), db, nil)
	jobQueue.SetPluginLoader(pluginLoader)

	parameters := map[string]interface{}{
		"pluginId":  uint(999),
		"outputDir": filepath.Join(tempDir, "output"),
	}

	jobID, err := jobQueue.CreateJob("test-plugin", "Test Plugin Job", "", []string{"main.py", "--test"})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	job, _ := jobQueue.GetJob(jobID)
	job.Parameters = parameters
	db.GetDB().Save(job)

	time.Sleep(500 * time.Millisecond)

	job, err = jobQueue.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.Status == models.JobStatusFailed && job.Error != "" {
		t.Logf("Job failed as expected (plugin doesn't exist): %s", job.Error)
	} else {
		t.Logf("Job status: %s", job.Status)
	}

	t.Log("Plugin V2 job processing did not panic on context.WithCancel")
}

func TestJobQueueCancelFuncRegistration(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	testJobID := "test-job-123"

	ctx, cancel := context.WithCancel(jobQueue.ctx)
	jobQueue.RegisterJobCancelFunc(testJobID, cancel)

	jobQueue.mu.RLock()
	_, exists := jobQueue.cancelFuncs[testJobID]
	jobQueue.mu.RUnlock()

	if !exists {
		t.Fatal("Cancel function was not registered")
	}

	jobQueue.UnregisterJobCancelFunc(testJobID)

	jobQueue.mu.RLock()
	_, exists = jobQueue.cancelFuncs[testJobID]
	jobQueue.mu.RUnlock()

	if exists {
		t.Fatal("Cancel function was not unregistered")
	}

	select {
	case <-ctx.Done():
		t.Log("Context was properly cancelled during unregistration")
	default:
		t.Log("Context not cancelled (expected if UnregisterJobCancelFunc doesn't call cancel)")
	}

	t.Log("Cancel function registration/unregistration works correctly")
}

func TestGenerateJobOutputDir_Unique(t *testing.T) {
	dir1 := GenerateJobOutputDir("/tmp/outputs", "cv-plot")
	dir2 := GenerateJobOutputDir("/tmp/outputs", "cv-plot")

	if dir1 == dir2 {
		t.Fatalf("expected unique output dirs for same-plugin runs generated close together, got identical: %s", dir1)
	}

	wantPrefix := filepath.Join("/tmp/outputs", "cv-plot_")
	if len(dir1) <= len(wantPrefix) || dir1[:len(wantPrefix)] != wantPrefix {
		t.Errorf("expected dir1 to start with %q, got %q", wantPrefix, dir1)
	}
}

// TestRerunJob_NonStandardOutputFlag covers a plugin whose output flag isn't
// one of RerunJob's hardcoded fallback spellings (e.g. uniprot-fetcher's
// "--output" vs the assumed "--output_folder"/"--output_dir"/"-o").
func TestRerunJob_NonStandardOutputFlag(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	pluginsDir := filepath.Join(tempDir, "plugins", "custom-output-plugin")
	os.MkdirAll(pluginsDir, 0755)
	pluginYAML := `plugin:
  id: custom-output-plugin
  name: Custom Output Plugin
  version: "1.0.0"
  description: Plugin using a non-standard output flag

runtime:
  environments:
    - python
  entrypoint: main.py

execution:
  outputDir: "--output"
`
	os.WriteFile(filepath.Join(pluginsDir, "plugin.yaml"), []byte(pluginYAML), 0644)
	os.WriteFile(filepath.Join(pluginsDir, "main.py"), []byte("print('ok')"), 0644)

	pluginLoader := NewPluginLoaderV2(filepath.Join(tempDir, "plugins"), db, nil)
	if err := pluginLoader.LoadPlugins(); err != nil {
		t.Fatalf("Failed to load plugins: %v", err)
	}
	jobQueue.SetPluginLoader(pluginLoader)

	plugins := pluginLoader.GetAllPlugins()
	if len(plugins) == 0 {
		t.Fatal("No test plugin loaded")
	}

	oldOutputDir := filepath.Join(tempDir, "outputs", "custom-output-plugin_20260101_000000_aaaaaaaa")
	job := &models.Job{
		ID:     "test-job-rerun-nonstandard",
		Type:   "custom-output-plugin",
		Name:   "Custom Output Plugin",
		Status: models.JobStatusCompleted,
		Args:   []string{"main.py", "--output", oldOutputDir},
		Parameters: map[string]interface{}{
			"outputDir": oldOutputDir,
		},
		CreatedAt: time.Now(),
	}

	jobQueue.mu.Lock()
	jobQueue.jobs[job.ID] = job
	jobQueue.mu.Unlock()
	if err := db.GetDB().Create(job).Error; err != nil {
		t.Fatalf("failed to seed job: %v", err)
	}

	newJobID, err := jobQueue.RerunJob(job.ID, true, "", "")
	if err != nil {
		t.Fatalf("RerunJob error: %v", err)
	}

	newJob, err := jobQueue.GetJob(newJobID)
	if err != nil {
		t.Fatalf("failed to get new job: %v", err)
	}

	newOutputDir, ok := newJob.Parameters["outputDir"].(string)
	if !ok || newOutputDir == "" {
		t.Fatalf("expected new job to have an outputDir parameter, got: %v", newJob.Parameters["outputDir"])
	}
	if newOutputDir == oldOutputDir {
		t.Fatal("expected rerun to use a new output directory, got the same one as the original run (would overwrite its results)")
	}

	found := false
	for i := 0; i < len(newJob.Args)-1; i++ {
		if newJob.Args[i] == "--output" {
			found = true
			if newJob.Args[i+1] != newOutputDir {
				t.Errorf("expected --output arg to be rewritten to %s, got %s", newOutputDir, newJob.Args[i+1])
			}
		}
	}
	if !found {
		t.Error("expected --output flag to still be present in rerun args")
	}
}

// TestRerunJob_UnresolvableOutputFlag_Errors covers a job whose plugin can't
// be resolved and whose args don't match any known output flag spelling --
// RerunJob must fail loudly rather than silently reuse (and overwrite) the
// original run's output directory.
func TestRerunJob_UnresolvableOutputFlag_Errors(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()
	// No plugin loader set, and the args use a flag spelling RerunJob doesn't know about.

	oldOutputDir := filepath.Join(tempDir, "outputs", "mystery-plugin_20260101_000000_aaaaaaaa")
	job := &models.Job{
		ID:     "test-job-rerun-unresolvable",
		Type:   "mystery-plugin",
		Name:   "Mystery Plugin",
		Status: models.JobStatusCompleted,
		Args:   []string{"main.py", "--outdir", oldOutputDir},
		Parameters: map[string]interface{}{
			"outputDir": oldOutputDir,
		},
		CreatedAt: time.Now(),
	}

	jobQueue.mu.Lock()
	jobQueue.jobs[job.ID] = job
	jobQueue.mu.Unlock()
	if err := db.GetDB().Create(job).Error; err != nil {
		t.Fatalf("failed to seed job: %v", err)
	}

	if _, err := jobQueue.RerunJob(job.ID, true, "", ""); err == nil {
		t.Fatal("expected RerunJob to error when it cannot locate the output directory argument, instead of silently overwriting the original run's results")
	}
}
