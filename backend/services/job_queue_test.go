package services

import (
	"context"
	"fmt"
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

	jobID, err := jobQueue.CreateJobWithEnvironments(
		plugin.Definition.Plugin.ID,
		plugin.Definition.Plugin.Name,
		"",
		[]string{"main.py"},
		parameters,
		"", "", "", "", "", "",
	)
	if err != nil {
		t.Fatalf("Failed to create plugin job: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	job, err := jobQueue.GetJob(jobID)
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

	jobID, err := jobQueue.CreateJobWithEnvironments(
		"test-plugin", "Test Plugin Job", "", []string{"main.py", "--test"},
		parameters, "", "", "", "", "", "",
	)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	job, err := jobQueue.GetJob(jobID)
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

func TestProcessJob_FailingScriptReportsFailedStatus(t *testing.T) {
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
	configurePython3ForTest(t, settingsService)
	scriptExecutor := NewScriptExecutor(settingsService, db)
	jobQueue.SetScriptExecutor(scriptExecutor)

	pluginsDir := filepath.Join(tempDir, "plugins", "failing-plugin")
	os.MkdirAll(pluginsDir, 0755)
	pluginYAML := `plugin:
  id: failing-plugin
  name: Failing Plugin
  version: "1.0.0"
  description: Plugin that always fails

runtime:
  environments:
    - python
  entrypoint: main.py
`
	os.WriteFile(filepath.Join(pluginsDir, "plugin.yaml"), []byte(pluginYAML), 0644)
	os.WriteFile(filepath.Join(pluginsDir, "main.py"), []byte("import sys\nsys.exit(1)\n"), 0644)

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

	parameters := map[string]interface{}{"pluginId": plugin.ID}

	jobID, err := jobQueue.CreateJobWithEnvironments(
		plugin.Definition.Plugin.ID, plugin.Definition.Plugin.Name, "", []string{"main.py"},
		parameters, "", "", "", "", "", "",
	)
	if err != nil {
		t.Fatalf("Failed to create plugin job: %v", err)
	}

	time.Sleep(1000 * time.Millisecond)

	job, err := jobQueue.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.Status != models.JobStatusFailed {
		t.Fatalf("expected job.Status = failed for a script that exited 1, got %q (error: %q)", job.Status, job.Error)
	}
	if job.Error == "" {
		t.Error("expected job.Error to be populated for a failed script")
	}
}

func TestGetJob_ReturnsIndependentCopy(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	jobID, err := jobQueue.CreateJob("test", "Independent Copy Job", "", []string{})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	first, err := jobQueue.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	first.Status = "tampered"
	first.TerminalOutput = append(first.TerminalOutput, "tampered line")
	first.Parameters["tampered"] = true

	second, err := jobQueue.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if second.Status == "tampered" {
		t.Error("mutating a job returned by GetJob affected a later GetJob call's Status")
	}
	for _, line := range second.TerminalOutput {
		if line == "tampered line" {
			t.Error("mutating a job returned by GetJob affected a later GetJob call's TerminalOutput")
		}
	}
	if _, ok := second.Parameters["tampered"]; ok {
		t.Error("mutating a job returned by GetJob affected a later GetJob call's Parameters")
	}
}

func configurePython3ForTest(t *testing.T, settingsService *SettingsService) {
	path, err := settingsService.DetectPythonPath()
	if err != nil || path == "" {
		t.Skip("python3 not available on this machine, skipping")
	}
	if err := settingsService.Set("pythonPath", path); err != nil {
		t.Fatalf("failed to configure python path: %v", err)
	}
}

func TestDeleteJob_CancelsInProgressJob(t *testing.T) {
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
	configurePython3ForTest(t, settingsService)
	scriptExecutor := NewScriptExecutor(settingsService, db)
	jobQueue.SetScriptExecutor(scriptExecutor)

	pluginsDir := filepath.Join(tempDir, "plugins", "slow-plugin")
	os.MkdirAll(pluginsDir, 0755)
	pluginYAML := `plugin:
  id: slow-plugin
  name: Slow Plugin
  version: "1.0.0"
  description: Plugin that sleeps long enough to be caught in_progress

runtime:
  environments:
    - python
  entrypoint: main.py
`
	os.WriteFile(filepath.Join(pluginsDir, "plugin.yaml"), []byte(pluginYAML), 0644)
	os.WriteFile(filepath.Join(pluginsDir, "main.py"), []byte("import time\ntime.sleep(10)\n"), 0644)

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

	parameters := map[string]interface{}{"pluginId": plugin.ID}
	jobID, err := jobQueue.CreateJobWithEnvironments(
		plugin.Definition.Plugin.ID, plugin.Definition.Plugin.Name, "", []string{"main.py"},
		parameters, "", "", "", "", "", "",
	)
	if err != nil {
		t.Fatalf("Failed to create plugin job: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		job, err := jobQueue.GetJob(jobID)
		if err == nil && job.Status == models.JobStatusInProgress {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	job, err := jobQueue.GetJob(jobID)
	if err != nil || job.Status != models.JobStatusInProgress {
		t.Fatalf("job never reached in_progress before the delete attempt (status: %v)", job)
	}

	start := time.Now()
	if err := jobQueue.DeleteJob(jobID); err != nil {
		t.Fatalf("DeleteJob failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	if elapsed := time.Since(start); elapsed >= 9*time.Second {
		t.Fatalf("job appears to have run to completion (10s sleep) instead of being cancelled, elapsed: %v", elapsed)
	}

	if _, err := jobQueue.GetJob(jobID); err == nil {
		t.Error("expected the deleted job to be gone from GetJob")
	}
}

func TestLoadFromDatabase_RequeuesPendingJobOlderThanRecentLimit(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	baseTime := time.Now().Add(-24 * time.Hour)

	oldPendingJob := &models.Job{
		ID:        "old-pending-job",
		Type:      "test",
		Name:      "Old Pending Job",
		Status:    models.JobStatusPending,
		Command:   "",
		Args:      []string{},
		CreatedAt: baseTime,
	}
	if err := db.GetDB().Create(oldPendingJob).Error; err != nil {
		t.Fatalf("failed to seed old pending job: %v", err)
	}

	for i := 0; i < 105; i++ {
		newerJob := &models.Job{
			ID:        fmt.Sprintf("filler-job-%d", i),
			Type:      "test",
			Name:      "Filler Job",
			Status:    models.JobStatusCompleted,
			Command:   "",
			Args:      []string{},
			CreatedAt: baseTime.Add(time.Duration(i+1) * time.Minute),
		}
		if err := db.GetDB().Create(newerJob).Error; err != nil {
			t.Fatalf("failed to seed filler job %d: %v", i, err)
		}
	}

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	deadline := time.Now().Add(3 * time.Second)
	var job *models.Job
	for time.Now().Before(deadline) {
		job, err = jobQueue.GetJob(oldPendingJob.ID)
		if err == nil && job.Status != models.JobStatusPending {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("failed to get old pending job: %v", err)
	}
	if job.Status == models.JobStatusPending {
		t.Fatalf("old pending job was never requeued on startup despite 105 newer jobs existing, status stuck at: %s", job.Status)
	}
}

func TestPauseQueue_StopsProcessingUntilResumed(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	if err := jobQueue.PauseQueue(); err != nil {
		t.Fatalf("PauseQueue failed: %v", err)
	}

	jobID, err := jobQueue.CreateJob("test", "Paused Job", "", []string{})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	job, err := jobQueue.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}
	if job.Status != models.JobStatusPending {
		t.Fatalf("expected job to remain pending while paused, got status: %s", job.Status)
	}

	if err := jobQueue.ResumeQueue(); err != nil {
		t.Fatalf("ResumeQueue failed: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		job, err = jobQueue.GetJob(jobID)
		if err == nil && job.Status == models.JobStatusCompleted {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if job.Status != models.JobStatusCompleted {
		t.Fatalf("expected job to complete after resume, got status: %s", job.Status)
	}
}

func TestPauseQueue_ErrorsWhenAlreadyPaused(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	if err := jobQueue.PauseQueue(); err != nil {
		t.Fatalf("first PauseQueue call failed: %v", err)
	}
	if err := jobQueue.PauseQueue(); err == nil {
		t.Fatal("expected second PauseQueue call to error")
	}
}

func TestResumeQueue_ErrorsWhenNotPaused(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	if err := jobQueue.ResumeQueue(); err == nil {
		t.Fatal("expected ResumeQueue to error when the queue isn't paused")
	}
}

func TestStopQueueImmediate_RevertsInProgressJobToPending(t *testing.T) {
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
	configurePython3ForTest(t, settingsService)
	scriptExecutor := NewScriptExecutor(settingsService, db)
	jobQueue.SetScriptExecutor(scriptExecutor)

	pluginsDir := filepath.Join(tempDir, "plugins", "slow-plugin-2")
	os.MkdirAll(pluginsDir, 0755)
	pluginYAML := `plugin:
  id: slow-plugin-2
  name: Slow Plugin 2
  version: "1.0.0"
  description: Plugin that sleeps long enough to be caught in_progress

runtime:
  environments:
    - python
  entrypoint: main.py
`
	os.WriteFile(filepath.Join(pluginsDir, "plugin.yaml"), []byte(pluginYAML), 0644)
	os.WriteFile(filepath.Join(pluginsDir, "main.py"), []byte("import time\ntime.sleep(10)\n"), 0644)

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

	parameters := map[string]interface{}{"pluginId": plugin.ID}
	jobID, err := jobQueue.CreateJobWithEnvironments(
		plugin.Definition.Plugin.ID, plugin.Definition.Plugin.Name, "", []string{"main.py"},
		parameters, "", "", "", "", "", "",
	)
	if err != nil {
		t.Fatalf("Failed to create plugin job: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	var startedAtBeforeStop *time.Time
	for time.Now().Before(deadline) {
		job, err := jobQueue.GetJob(jobID)
		if err == nil && job.Status == models.JobStatusInProgress {
			startedAtBeforeStop = job.StartedAt
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if startedAtBeforeStop == nil {
		t.Fatal("job never reached in_progress before the stop attempt")
	}

	if err := jobQueue.StopQueueImmediate(); err != nil {
		t.Fatalf("StopQueueImmediate failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	job, err := jobQueue.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}
	if job.Status != models.JobStatusPending {
		t.Fatalf("expected job status to revert to pending after StopQueueImmediate, got: %s", job.Status)
	}
	if job.StartedAt == nil || !job.StartedAt.Equal(*startedAtBeforeStop) {
		t.Errorf("expected StartedAt to be preserved after StopQueueImmediate, got: %v (was: %v)", job.StartedAt, startedAtBeforeStop)
	}
	if job.Error == "" {
		t.Error("expected Error to explain the job was stopped by the user, got empty string")
	}
}

func TestGetQueueStatus_ReflectsPendingAndInProgressCounts(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	if err := jobQueue.PauseQueue(); err != nil {
		t.Fatalf("PauseQueue failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		if _, err := jobQueue.CreateJob("test", "Status Count Job", "", []string{}); err != nil {
			t.Fatalf("Failed to create job %d: %v", i, err)
		}
	}
	time.Sleep(200 * time.Millisecond)

	status := jobQueue.GetQueueStatus()
	pendingCount, ok := status["pendingCount"].(int64)
	if !ok || pendingCount != 3 {
		t.Errorf("expected pendingCount = 3, got %v", status["pendingCount"])
	}
	if paused, ok := status["paused"].(bool); !ok || !paused {
		t.Errorf("expected paused = true, got %v", status["paused"])
	}
}

func TestClassifyJobCancellation(t *testing.T) {
	cases := []struct {
		name             string
		ctxErr           error
		stoppedByQueue   bool
		wantTimedOut     bool
		wantWasCancelled bool
	}{
		{"normal completion", nil, false, false, false},
		{"timed out", context.DeadlineExceeded, false, true, false},
		{"explicit cancel via context", context.Canceled, false, false, true},
		{"stopped by queue before context observed cancel", nil, true, false, true},
		{"stopped by queue and context already cancelled", context.Canceled, true, false, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			timedOut, wasCancelled := classifyJobCancellation(c.ctxErr, c.stoppedByQueue)
			if timedOut != c.wantTimedOut {
				t.Errorf("timedOut = %v, want %v", timedOut, c.wantTimedOut)
			}
			if wasCancelled != c.wantWasCancelled {
				t.Errorf("wasCancelled = %v, want %v", wasCancelled, c.wantWasCancelled)
			}
		})
	}
}

func TestNewJobQueueServiceInternal_UsesConfiguredWorkerCount(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.SaveSetting("maxConcurrentJobs", "5"); err != nil {
		t.Fatalf("Failed to seed maxConcurrentJobs setting: %v", err)
	}

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	if jobQueue.workers != 5 {
		t.Errorf("expected workers = 5 from configured setting, got %d", jobQueue.workers)
	}
}

func TestNewJobQueueServiceInternal_DefaultsWorkerCountWhenUnset(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	if jobQueue.workers != 2 {
		t.Errorf("expected default workers = 2, got %d", jobQueue.workers)
	}
}

func TestGetAllJobs_Pagination(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := newDatabaseServiceFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	jobQueue := NewJobQueueServiceV3(db, nil)
	defer jobQueue.Shutdown()

	baseTime := time.Now().Add(-1 * time.Hour)
	for i := 0; i < 5; i++ {
		job := &models.Job{
			ID:        fmt.Sprintf("page-job-%d", i),
			Type:      "test",
			Name:      fmt.Sprintf("Page Job %d", i),
			Status:    models.JobStatusCompleted,
			Args:      []string{},
			CreatedAt: baseTime.Add(time.Duration(i) * time.Minute),
		}
		if err := db.GetDB().Create(job).Error; err != nil {
			t.Fatalf("failed to seed job %d: %v", i, err)
		}
	}

	firstPage := jobQueue.GetAllJobs(2, 0)
	if len(firstPage) != 2 {
		t.Fatalf("expected 2 jobs in first page, got %d", len(firstPage))
	}
	if firstPage[0].ID != "page-job-4" || firstPage[1].ID != "page-job-3" {
		t.Errorf("expected first page to be the 2 newest jobs in descending order, got %s, %s", firstPage[0].ID, firstPage[1].ID)
	}

	secondPage := jobQueue.GetAllJobs(2, 2)
	if len(secondPage) != 2 {
		t.Fatalf("expected 2 jobs in second page, got %d", len(secondPage))
	}
	if secondPage[0].ID != "page-job-2" || secondPage[1].ID != "page-job-1" {
		t.Errorf("expected second page to continue in descending order, got %s, %s", secondPage[0].ID, secondPage[1].ID)
	}

	lastPage := jobQueue.GetAllJobs(2, 4)
	if len(lastPage) != 1 {
		t.Fatalf("expected 1 job in the final partial page, got %d", len(lastPage))
	}
	if lastPage[0].ID != "page-job-0" {
		t.Errorf("expected the last page's job to be page-job-0, got %s", lastPage[0].ID)
	}
}
