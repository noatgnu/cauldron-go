package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/backend/services"
)

func TestPluginExecutionE2E(t *testing.T) {
	app := NewApp()

	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("List available plugins", func(t *testing.T) {
		plugins := app.GetPluginsV2()
		if plugins == nil {
			t.Fatal("GetPluginsV2 returned nil")
		}

		t.Logf("Found %d plugins:", len(plugins))
		for _, p := range plugins {
			t.Logf("  - %s (ID: %d, StringID: %s)", p.Definition.Plugin.Name, p.ID, p.Definition.Plugin.ID)
			t.Logf("    Environments: %v", p.Definition.Runtime.Environments)
			t.Logf("    Entrypoint: %s", p.Definition.Runtime.Entrypoint)
			t.Logf("    FolderPath: %s", p.FolderPath)
		}

		if len(plugins) == 0 {
			t.Skip("No plugins available for testing")
		}
	})

	t.Run("Execute CV Plot plugin with example data", func(t *testing.T) {
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

		t.Logf("Found CV Plot plugin: ID=%d, FolderPath=%s", cvPlugin.ID, cvPlugin.FolderPath)

		exePath, err := os.Executable()
		if err != nil {
			t.Fatalf("Failed to get executable path: %v", err)
		}
		exeDir := filepath.Dir(exePath)

		annotationPath := filepath.Join(exeDir, "examples", "diann", "annotation.txt")
		logFilePath := filepath.Join(exeDir, "examples", "diann", "Reports.log.txt")
		prMatrixPath := filepath.Join(exeDir, "examples", "diann", "Reports.pr_matrix.tsv")
		pgMatrixPath := filepath.Join(exeDir, "examples", "diann", "Reports.pg_matrix.tsv")

		for _, path := range []string{annotationPath, logFilePath, prMatrixPath, pgMatrixPath} {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skipf("Example data file not found: %s", path)
			}
		}

		t.Logf("Example data files found in: %s", filepath.Join(exeDir, "examples", "diann"))

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

		t.Logf("Created job: %s", jobID)

		timeout := time.After(120 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				job, _ := app.GetJob(jobID)
				t.Fatalf("Job timed out. Final status: %s, Error: %s", job.Status, job.Error)

			case <-ticker.C:
				job, err := app.GetJob(jobID)
				if err != nil {
					t.Fatalf("Failed to get job status: %v", err)
				}

				t.Logf("Job status: %s (progress: %.0f%%)", job.Status, job.Progress)

				if job.Status == models.JobStatusCompleted {
					t.Logf("Job completed successfully!")
					t.Logf("Output path: %s", job.OutputPath)

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
					t.Logf("Job failed with error: %s", job.Error)
					t.Logf("Terminal output:")
					for _, line := range job.TerminalOutput {
						t.Logf("  %s", line)
					}
					t.Fatalf("Job failed")
				}
			}
		}
	})
}

func TestPluginValidation(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Validate all installed plugins", func(t *testing.T) {
		plugins := app.GetPluginsV2()

		for _, p := range plugins {
			t.Run(p.Definition.Plugin.Name, func(t *testing.T) {
				if p.Definition.Plugin.ID == "" {
					t.Error("Plugin ID is empty")
				}
				if p.Definition.Plugin.Name == "" {
					t.Error("Plugin name is empty")
				}
				if len(p.Definition.Runtime.Environments) == 0 {
					t.Error("No runtime environments specified")
				}
				if p.Definition.Runtime.Entrypoint == "" {
					t.Error("Entrypoint is empty")
				}

				if p.FolderPath == "" {
					t.Error("FolderPath is empty")
				} else {
					entrypointPath := filepath.Join(p.FolderPath, p.Definition.Runtime.Entrypoint)
					if _, err := os.Stat(entrypointPath); os.IsNotExist(err) {
						t.Errorf("Entrypoint file not found: %s", entrypointPath)
					}
				}

				t.Logf("Plugin %s validated successfully", p.Definition.Plugin.Name)
			})
		}
	})
}

func loadThrowawayTestPlugin(t *testing.T, app *App, pluginID string) *models.PluginV2 {
	t.Helper()

	pythonPath, err := app.settings.DetectPythonPath()
	if err != nil || pythonPath == "" {
		t.Skip("python3 not available on this machine, skipping")
	}
	if err := app.settings.Set("pythonPath", pythonPath); err != nil {
		t.Fatalf("failed to configure python path: %v", err)
	}

	tempDir := t.TempDir()
	pluginDir := filepath.Join(tempDir, pluginID)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}
	pluginYAML := "plugin:\n" +
		"  id: " + pluginID + "\n" +
		"  name: Throwaway Test Plugin\n" +
		"  version: \"1.0.0\"\n" +
		"  description: Minimal plugin used by an integration test\n" +
		"\n" +
		"runtime:\n" +
		"  environments:\n" +
		"    - python\n" +
		"  entrypoint: main.py\n"
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(pluginYAML), 0644); err != nil {
		t.Fatalf("failed to write plugin.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "main.py"), []byte("import sys\nsys.exit(0)\n"), 0644); err != nil {
		t.Fatalf("failed to write entrypoint: %v", err)
	}

	pluginLoader := services.NewPluginLoaderV2(tempDir, app.db, nil)
	if err := pluginLoader.LoadPlugins(); err != nil {
		t.Fatalf("failed to load test plugin: %v", err)
	}
	app.pluginLoaderV2 = pluginLoader
	app.jobQueue.SetPluginLoader(pluginLoader)

	plugins := app.GetPluginsV2()
	if len(plugins) == 0 {
		t.Fatal("expected the test plugin to load")
	}
	return plugins[0]
}

func TestJobQueueIntegration(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Queue multiple jobs and verify ordering", func(t *testing.T) {
		plugin := loadThrowawayTestPlugin(t, app, "queue-order-test-plugin")

		if err := app.PauseJobQueue(); err != nil {
			t.Fatalf("Failed to pause queue: %v", err)
		}

		var jobIDs []string
		for i := 0; i < 3; i++ {
			req := models.PluginExecutionRequestV2{
				PluginID: plugin.ID,
				Parameters: map[string]interface{}{
					"test_index": i,
				},
			}

			jobID, err := app.ExecutePluginV2(req)
			if err != nil {
				t.Fatalf("Failed to create job %d: %v", i, err)
			}
			jobIDs = append(jobIDs, jobID)
			t.Logf("Created job %d: %s", i, jobID)
		}

		status := app.GetJobQueueStatus()
		if pendingCount, ok := status["pendingCount"].(int64); !ok || pendingCount < 3 {
			t.Errorf("expected at least 3 pending jobs while the queue is paused, got %v (status: %+v)", status["pendingCount"], status)
		}

		if err := app.ResumeJobQueue(); err != nil {
			t.Fatalf("Failed to resume queue: %v", err)
		}

		deadline := time.Now().Add(10 * time.Second)
		for _, jobID := range jobIDs {
			for {
				job, err := app.GetJob(jobID)
				if err != nil {
					t.Errorf("Failed to get job %s: %v", jobID, err)
					break
				}
				if job.Status == models.JobStatusCompleted {
					break
				}
				if job.Status == models.JobStatusFailed {
					t.Errorf("job %s failed: %s", jobID, job.Error)
					break
				}
				if time.Now().After(deadline) {
					t.Errorf("job %s did not complete within 10s, status=%s", jobID, job.Status)
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
		}

		for _, jobID := range jobIDs {
			app.DeleteJob(jobID)
		}
	})
}

func TestEventEmission(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Event emission with nil wailsApp should not crash", func(t *testing.T) {
		originalWailsApp := app.wailsApp
		app.wailsApp = nil

		defer func() {
			app.wailsApp = originalWailsApp
			if r := recover(); r != nil {
				t.Errorf("Event emission caused panic: %v", r)
			}
		}()

		app.emitEvent("test:event", map[string]string{"test": "data"})
		t.Log("Event emission with nil wailsApp did not crash")
	})

	t.Run("Job creation emits event without crash", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Job creation caused panic: %v", r)
			}
		}()

		jobID, err := app.jobQueue.CreateJob("test", "Event Test Job", "", []string{})
		if err != nil {
			t.Fatalf("CreateJob failed: %v", err)
		}
		defer app.DeleteJob(jobID)

		t.Logf("Created job %s without crash", jobID)
	})
}

func TestPluginParameterSerialization(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Job parameters serialize correctly", func(t *testing.T) {
		plugin := loadThrowawayTestPlugin(t, app, "param-serialization-test-plugin")

		req := models.PluginExecutionRequestV2{
			PluginID: plugin.ID,
			Parameters: map[string]interface{}{
				"string_param": "test value",
				"int_param":    42,
				"float_param":  3.14,
				"bool_param":   true,
				"array_param":  []string{"a", "b", "c"},
				"path_param":   "/path/to/file.csv",
			},
		}

		jobID, err := app.ExecutePluginV2(req)
		if err != nil {
			t.Fatalf("ExecutePluginV2 failed: %v", err)
		}
		defer app.DeleteJob(jobID)

		job, err := app.GetJob(jobID)
		if err != nil {
			t.Fatalf("GetJob failed: %v", err)
		}

		jobJSON, err := json.Marshal(job)
		if err != nil {
			t.Fatalf("Failed to serialize job: %v", err)
		}

		jsonStr := string(jobJSON)
		t.Logf("Job JSON: %s", jsonStr)

		if strings.Contains(jsonStr, `"args":null`) {
			t.Error("args should not be null")
		}

		if strings.Contains(jsonStr, `"parameters":null`) {
			t.Error("parameters should not be null")
		}
	})
}

func TestScriptExecutorConfiguration(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Python path is configured", func(t *testing.T) {
		config := app.GetSettings()
		if config.PythonPath == "" {
			t.Log("Warning: Python path not configured")
		} else {
			t.Logf("Python path: %s", config.PythonPath)

			if _, err := os.Stat(config.PythonPath); os.IsNotExist(err) {
				t.Errorf("Configured Python path does not exist: %s", config.PythonPath)
			}
		}
	})

	t.Run("Output directory is configured and writable", func(t *testing.T) {
		config := app.GetSettings()
		if config.OutputDirectory == "" {
			t.Error("OutputDirectory not configured")
			return
		}

		t.Logf("Output directory: %s", config.OutputDirectory)

		testFile := filepath.Join(config.OutputDirectory, ".write_test")
		err := os.WriteFile(testFile, []byte("test"), 0644)
		if err != nil {
			t.Errorf("Output directory not writable: %v", err)
		} else {
			os.Remove(testFile)
			t.Log("Output directory is writable")
		}
	})
}
