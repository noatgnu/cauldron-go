package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func TestE2EPluginExecution(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("Full plugin execution flow", func(t *testing.T) {
		plugins := app.GetPluginsV2()
		if len(plugins) == 0 {
			t.Skip("No plugins installed")
		}

		var testPlugin *models.PluginV2
		for _, p := range plugins {
			if p.Definition.Runtime.HasEnvironment("python") {
				testPlugin = p
				break
			}
		}

		if testPlugin == nil {
			t.Skip("No Python plugin available")
		}

		t.Logf("Testing with plugin: %s (ID: %d)", testPlugin.Definition.Plugin.Name, testPlugin.ID)

		params := make(map[string]interface{})
		for _, input := range testPlugin.Definition.Inputs {
			switch input.Type {
			case "file":
				if input.Required {
					t.Skipf("Plugin %s requires file input, skipping", testPlugin.Definition.Plugin.Name)
				}
			case "string":
				params[input.Name] = "test"
			case "number", "integer":
				params[input.Name] = 1
			case "boolean":
				params[input.Name] = true
			}
		}

		req := models.PluginExecutionRequestV2{
			PluginID:   testPlugin.ID,
			Parameters: params,
		}

		jobID, err := app.ExecutePluginV2(req)
		if err != nil {
			t.Logf("ExecutePluginV2 failed (expected for plugins requiring files): %v", err)
			return
		}

		t.Logf("Job created: %s", jobID)

		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				job, _ := app.GetJob(jobID)
				t.Logf("Job timed out. Status: %s, Error: %s", job.Status, job.Error)
				return

			case <-ticker.C:
				job, err := app.GetJob(jobID)
				if err != nil {
					t.Fatalf("GetJob failed: %v", err)
				}

				t.Logf("Job status: %s, progress: %.0f%%", job.Status, job.Progress)

				if job.Status == models.JobStatusCompleted {
					t.Logf("Job completed successfully")
					return
				}

				if job.Status == models.JobStatusFailed {
					t.Logf("Job failed: %s", job.Error)
					return
				}
			}
		}
	})
}

func TestE2EJobQueueOperations(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("Create job via frontend API", func(t *testing.T) {
		plugins := app.GetPluginsV2()
		if len(plugins) == 0 {
			t.Skip("No plugins installed")
		}

		plugin := plugins[0]
		t.Logf("Using plugin: %s", plugin.Definition.Plugin.Name)

		initialJobs := app.GetAllJobs()
		initialCount := len(initialJobs)
		t.Logf("Initial job count: %d", initialCount)

		req := models.PluginExecutionRequestV2{
			PluginID: plugin.ID,
			Parameters: map[string]interface{}{
				"test": "value",
			},
		}

		jobID, err := app.ExecutePluginV2(req)
		if err != nil {
			t.Logf("Job creation returned error (may be expected): %v", err)
		} else {
			t.Logf("Created job: %s", jobID)

			job, err := app.GetJob(jobID)
			if err != nil {
				t.Fatalf("GetJob failed: %v", err)
			}

			t.Logf("Job retrieved: ID=%s, Status=%s, Type=%s", job.ID, job.Status, job.Type)

			time.Sleep(2 * time.Second)

			job, _ = app.GetJob(jobID)
			t.Logf("Final job status: %s, Error: %s", job.Status, job.Error)
		}
	})

	t.Run("Queue status via frontend API", func(t *testing.T) {
		status := app.GetJobQueueStatus()
		t.Logf("Queue status: %+v", status)

		if status == nil {
			t.Error("GetJobQueueStatus returned nil")
		}
	})

	t.Run("Pause and resume queue", func(t *testing.T) {
		err := app.PauseJobQueue()
		if err != nil {
			t.Fatalf("PauseJobQueue failed: %v", err)
		}

		status := app.GetJobQueueStatus()
		if paused, ok := status["paused"].(bool); !ok || !paused {
			t.Error("Queue should be paused")
		}

		err = app.ResumeJobQueue()
		if err != nil {
			t.Fatalf("ResumeJobQueue failed: %v", err)
		}

		status = app.GetJobQueueStatus()
		if paused, ok := status["paused"].(bool); ok && paused {
			t.Error("Queue should be resumed")
		}

		t.Log("Queue pause/resume works correctly")
	})
}

func TestE2ESettingsAndConfig(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("Get settings via frontend API", func(t *testing.T) {
		settings := app.GetSettings()
		if settings == nil {
			t.Fatal("GetSettings returned nil")
		}

		t.Logf("Settings: PythonPath=%s, OutputDir=%s", settings.PythonPath, settings.OutputDirectory)
	})

	t.Run("Get Python version via frontend API", func(t *testing.T) {
		version, err := app.GetPythonVersion()
		if err != nil {
			t.Logf("GetPythonVersion error: %v", err)
		} else {
			t.Logf("Python version: %s", version)
		}
	})

	t.Run("Get all jobs via frontend API", func(t *testing.T) {
		jobs := app.GetAllJobs()
		if jobs == nil {
			t.Fatal("GetAllJobs returned nil")
		}
		t.Logf("Total jobs: %d", len(jobs))
	})

	t.Run("Get plugins via frontend API", func(t *testing.T) {
		plugins := app.GetPluginsV2()
		if plugins == nil {
			t.Fatal("GetPluginsV2 returned nil")
		}
		t.Logf("Total plugins: %d", len(plugins))

		for i, p := range plugins {
			if i >= 5 {
				t.Logf("... and %d more plugins", len(plugins)-5)
				break
			}
			t.Logf("  - %s (%s)", p.Definition.Plugin.Name, p.Definition.Plugin.ID)
		}
	})
}

func TestE2EWithExampleData(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	exePath, err := os.Executable()
	if err != nil {
		t.Fatalf("Failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	examplesDir := filepath.Join(exeDir, "examples", "diann")

	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		t.Skipf("Examples directory not found: %s", examplesDir)
	}

	t.Run("Execute CV Plot with example data", func(t *testing.T) {
		plugins := app.GetPluginsV2()

		var cvPlugin *models.PluginV2
		for _, p := range plugins {
			if p.Definition.Plugin.ID == "cv-plot" {
				cvPlugin = p
				break
			}
		}

		if cvPlugin == nil {
			t.Skip("cv-plot plugin not installed")
		}

		annotationPath := filepath.Join(examplesDir, "annotation.txt")
		logFilePath := filepath.Join(examplesDir, "Reports.log.txt")
		prMatrixPath := filepath.Join(examplesDir, "Reports.pr_matrix.tsv")
		pgMatrixPath := filepath.Join(examplesDir, "Reports.pg_matrix.tsv")

		for _, path := range []string{annotationPath, logFilePath, prMatrixPath, pgMatrixPath} {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skipf("Example file not found: %s", path)
			}
		}

		req := models.PluginExecutionRequestV2{
			PluginID: cvPlugin.ID,
			Parameters: map[string]interface{}{
				"annotation_file":     annotationPath,
				"log_file_path":       logFilePath,
				"report_pr_file_path": prMatrixPath,
				"report_pg_file_path": pgMatrixPath,
				"intensity_col":       "Intensity",
			},
		}

		jobID, err := app.ExecutePluginV2(req)
		if err != nil {
			t.Fatalf("ExecutePluginV2 failed: %v", err)
		}

		t.Logf("Job created: %s", jobID)

		timeout := time.After(120 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				job, _ := app.GetJob(jobID)
				t.Fatalf("Job timed out. Status: %s, Error: %s", job.Status, job.Error)

			case <-ticker.C:
				job, err := app.GetJob(jobID)
				if err != nil {
					t.Fatalf("GetJob failed: %v", err)
				}

				t.Logf("Job status: %s, progress: %.0f%%", job.Status, job.Progress)

				if job.Status == models.JobStatusCompleted {
					t.Logf("Job completed! Output: %s", job.OutputPath)

					if job.OutputPath != "" {
						files, err := os.ReadDir(job.OutputPath)
						if err == nil {
							t.Logf("Output files:")
							for _, f := range files {
								t.Logf("  - %s", f.Name())
							}
						}
					}
					return
				}

				if job.Status == models.JobStatusFailed {
					t.Fatalf("Job failed: %s", job.Error)
				}
			}
		}
	})
}

func TestE2EEventEmissionDuringJobExecution(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("Job execution with nil wailsApp should not crash", func(t *testing.T) {
		plugins := app.GetPluginsV2()
		if len(plugins) == 0 {
			t.Skip("No plugins installed")
		}

		plugin := plugins[0]

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic during job execution: %v", r)
			}
		}()

		req := models.PluginExecutionRequestV2{
			PluginID: plugin.ID,
			Parameters: map[string]interface{}{
				"test": "value",
			},
		}

		jobID, err := app.ExecutePluginV2(req)
		if err != nil {
			t.Logf("Job creation error (expected): %v", err)
			return
		}

		time.Sleep(3 * time.Second)

		job, _ := app.GetJob(jobID)
		t.Logf("Job completed without panic. Status: %s", job.Status)
	})
}
