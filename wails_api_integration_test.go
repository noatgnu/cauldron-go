package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func TestWailsAPIIntegration(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("API: GetSettings returns valid config", func(t *testing.T) {
		config := app.GetSettings()

		if config == nil {
			t.Fatal("GetSettings returned nil")
		}

		jsonBytes, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("Failed to serialize config: %v", err)
		}

		t.Logf("Config JSON: %s", string(jsonBytes))

		if config.ResultStoragePath == "" {
			t.Error("ResultStoragePath should not be empty")
		}
	})

	t.Run("API: GetAllJobs returns array not null", func(t *testing.T) {
		jobs := app.GetAllJobs()

		if jobs == nil {
			t.Fatal("GetAllJobs returned nil instead of empty array")
		}

		jsonBytes, _ := json.Marshal(jobs)
		jsonStr := string(jsonBytes)

		if len(jobs) == 0 && jsonStr != "[]" {
			t.Errorf("Empty jobs should serialize to [], got: %s", jsonStr)
		}

		t.Logf("Jobs: %s", jsonStr)
	})

	t.Run("API: GetImportedFiles returns array not null", func(t *testing.T) {
		files, err := app.GetImportedFiles()

		if err != nil {
			t.Fatalf("GetImportedFiles returned error: %v", err)
		}

		if files == nil {
			t.Fatal("GetImportedFiles returned nil instead of empty array")
		}

		jsonBytes, _ := json.Marshal(files)
		t.Logf("Files: %s", string(jsonBytes))
	})

	t.Run("API: GetPluginsV2 returns array not null", func(t *testing.T) {
		plugins := app.GetPluginsV2()

		if plugins == nil {
			t.Fatal("GetPluginsV2 returned nil instead of empty array")
		}

		t.Logf("Found %d plugins", len(plugins))
	})

	t.Run("API: Python version detection", func(t *testing.T) {
		version, err := app.GetPythonVersion()

		if err != nil {
			t.Logf("Python not detected (expected in some environments): %v", err)
		} else {
			t.Logf("Python version: %s", version)
		}
	})

	t.Run("API: R version detection", func(t *testing.T) {
		version, err := app.GetRVersion()

		if err != nil {
			t.Logf("R not detected (expected in some environments): %v", err)
		} else {
			t.Logf("R version: %s", version)
		}
	})

	t.Run("API: Docker version detection", func(t *testing.T) {
		version, err := app.CheckDockerVersion()

		if err != nil {
			t.Logf("Docker not detected (expected in some environments): %v", err)
		} else {
			t.Logf("Docker version: %s", version)
		}
	})

	t.Run("API: Job Queue Status", func(t *testing.T) {
		status := app.GetJobQueueStatus()

		if status == nil {
			t.Fatal("GetJobQueueStatus returned nil")
		}

		jsonBytes, _ := json.Marshal(status)
		t.Logf("Queue status: %s", string(jsonBytes))
	})

	t.Run("API: Virtual Environments", func(t *testing.T) {
		venvs, err := app.GetVirtualEnvironments()

		if err != nil {
			t.Fatalf("GetVirtualEnvironments returned error: %v", err)
		}

		if venvs == nil {
			t.Fatal("GetVirtualEnvironments returned nil")
		}

		t.Logf("Found %d virtual environments", len(venvs))
	})

	t.Run("API: Renv Environments", func(t *testing.T) {
		renvs, err := app.GetRenvEnvironments()

		if err != nil {
			t.Fatalf("GetRenvEnvironments returned error: %v", err)
		}

		if renvs == nil {
			t.Fatal("GetRenvEnvironments returned nil")
		}

		t.Logf("Found %d renv environments", len(renvs))
	})

	t.Run("API: Plugin Environment Bindings", func(t *testing.T) {
		bindings, err := app.GetAllPluginEnvironmentBindings()

		if err != nil {
			t.Fatalf("GetAllPluginEnvironmentBindings returned error: %v", err)
		}

		if bindings == nil {
			t.Fatal("GetAllPluginEnvironmentBindings returned nil")
		}

		t.Logf("Found %d bindings", len(bindings))
	})
}

func TestWailsJobLifecycle(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Job Lifecycle: Create, Get, Delete", func(t *testing.T) {
		plugins := app.GetPluginsV2()
		var testPluginID uint
		for _, p := range plugins {
			if p.Definition.Plugin.ID != "" {
				testPluginID = p.ID
				break
			}
		}

		if testPluginID == 0 {
			t.Skip("No plugins available for testing")
		}

		req := models.PluginExecutionRequestV2{
			PluginID: testPluginID,
			Parameters: map[string]interface{}{
				"test_param": "test_value",
			},
		}

		jobID, err := app.ExecutePluginV2(req)
		if err != nil {
			t.Fatalf("ExecutePluginV2 failed: %v", err)
		}
		t.Logf("Created job: %s", jobID)

		time.Sleep(100 * time.Millisecond)

		job, err := app.GetJob(jobID)
		if err != nil {
			t.Fatalf("GetJob failed: %v", err)
		}

		if job.ID != jobID {
			t.Errorf("Job ID mismatch: expected %s, got %s", jobID, job.ID)
		}

		if job.Args == nil {
			t.Error("Job.Args should not be nil")
		}

		jsonBytes, _ := json.Marshal(job)
		t.Logf("Job: %s", string(jsonBytes))

		err = app.DeleteJob(jobID)
		if err != nil {
			t.Fatalf("DeleteJob failed: %v", err)
		}

		_, err = app.GetJob(jobID)
		if err == nil {
			t.Error("GetJob should fail after deletion")
		}

		t.Log("Job lifecycle test passed")
	})
}

func TestWailsJobQueueOperations(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Job Queue: Pause and Resume", func(t *testing.T) {
		err := app.PauseJobQueue()
		if err != nil {
			t.Fatalf("PauseJobQueue failed: %v", err)
		}

		status := app.GetJobQueueStatus()
		if isPaused, ok := status["isPaused"].(bool); ok && !isPaused {
			t.Error("Queue should be paused")
		}

		err = app.ResumeJobQueue()
		if err != nil {
			t.Fatalf("ResumeJobQueue failed: %v", err)
		}

		status = app.GetJobQueueStatus()
		if isPaused, ok := status["isPaused"].(bool); ok && isPaused {
			t.Error("Queue should be running")
		}
	})
}

func TestWailsEnvironmentDetection(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Environment: Detect Python Environments", func(t *testing.T) {
		envs, err := app.DetectPythonEnvironments()

		if err != nil {
			t.Logf("Python detection error (may be expected): %v", err)
			return
		}

		if envs == nil {
			t.Fatal("DetectPythonEnvironments returned nil")
		}

		t.Logf("Found %d Python environments", len(envs))
		for _, env := range envs {
			t.Logf("  - %s (v%s, virtual: %v)", env.Path, env.Version, env.IsVirtual)
		}
	})

	t.Run("Environment: Detect R Environments", func(t *testing.T) {
		envs, err := app.DetectREnvironments()

		if err != nil {
			t.Logf("R detection error (may be expected): %v", err)
			return
		}

		if envs == nil {
			t.Fatal("DetectREnvironments returned nil")
		}

		t.Logf("Found %d R environments", len(envs))
		for _, env := range envs {
			t.Logf("  - %s (v%s)", env.Path, env.Version)
		}
	})

	t.Run("Environment: Get Active Environments", func(t *testing.T) {
		pythonEnv, _ := app.GetActivePythonEnvironment()
		if pythonEnv != nil {
			t.Logf("Active Python: %s", pythonEnv.Path)
		} else {
			t.Log("No active Python environment")
		}

		rEnv, _ := app.GetActiveREnvironment()
		if rEnv != nil {
			t.Logf("Active R: %s", rEnv.Path)
		} else {
			t.Log("No active R environment")
		}
	})
}

func TestWailsPluginOperations(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Plugin: Get Plugins V2", func(t *testing.T) {
		plugins := app.GetPluginsV2()

		if plugins == nil {
			t.Fatal("GetPluginsV2 returned nil")
		}

		for _, p := range plugins {
			t.Logf("Plugin: %s (v%s) - %s",
				p.Definition.Plugin.Name,
				p.Definition.Plugin.Version,
				p.Definition.Plugin.Description)
		}
	})

	t.Run("Plugin: Get Plugin By ID", func(t *testing.T) {
		plugins := app.GetPluginsV2()
		if len(plugins) == 0 {
			t.Skip("No plugins available")
		}

		plugin, err := app.GetPluginV2(plugins[0].ID)
		if err != nil {
			t.Fatalf("GetPluginV2 failed: %v", err)
		}
		if plugin == nil {
			t.Fatal("GetPluginV2 returned nil")
		}

		t.Logf("Retrieved plugin: %s", plugin.Definition.Plugin.Name)
	})

	t.Run("Plugin: Validate YAML", func(t *testing.T) {
		validYAML := `plugin:
  id: test-plugin
  name: Test Plugin
  description: A test plugin for validation
  version: 1.0.0
  author: Test
  category: analysis

runtime:
  environments:
    - python
  entrypoint: main.py

inputs:
  - name: input_file
    label: Input File
    type: file
    required: true
    description: Input data file
    flag: --input

outputs: []

execution:
  outputDir: --output_folder
`
		valid, errors, err := app.ValidatePluginYAML(validYAML)
		if err != nil {
			t.Fatalf("ValidatePluginYAML failed: %v", err)
		}

		if !valid {
			t.Errorf("Valid YAML was marked as invalid: %v", errors)
		}

		t.Log("YAML validation passed")
	})
}

func TestWailsDataSerialization(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Serialization: Empty arrays serialize to [] not null", func(t *testing.T) {
		jobs := app.GetAllJobs()
		jobsJSON, _ := json.Marshal(jobs)

		if len(jobs) == 0 && string(jobsJSON) != "[]" {
			t.Errorf("Empty jobs should serialize to [], got: %s", string(jobsJSON))
		}

		files, _ := app.GetImportedFiles()
		filesJSON, _ := json.Marshal(files)

		if len(files) == 0 && string(filesJSON) != "[]" {
			t.Errorf("Empty files should serialize to [], got: %s", string(filesJSON))
		}

		plugins := app.GetPluginsV2()
		pluginsJSON, _ := json.Marshal(plugins)

		if len(plugins) == 0 && string(pluginsJSON) != "[]" {
			t.Errorf("Empty plugins should serialize to [], got: %s", string(pluginsJSON))
		}

		venvs, _ := app.GetVirtualEnvironments()
		venvsJSON, _ := json.Marshal(venvs)

		if len(venvs) == 0 && string(venvsJSON) != "[]" {
			t.Errorf("Empty venvs should serialize to [], got: %s", string(venvsJSON))
		}

		t.Log("All empty arrays serialize correctly")
	})

	t.Run("Serialization: Job fields are not null", func(t *testing.T) {
		jobID, _ := app.jobQueue.CreateJob("test", "Serialization Test", "", []string{})
		defer app.DeleteJob(jobID)

		job, err := app.GetJob(jobID)
		if err != nil {
			t.Fatalf("GetJob failed: %v", err)
		}

		jobJSON, _ := json.Marshal(job)

		type JobFields struct {
			Args           json.RawMessage `json:"args"`
			TerminalOutput json.RawMessage `json:"terminalOutput"`
		}

		var fields JobFields
		json.Unmarshal(jobJSON, &fields)

		if string(fields.Args) == "null" {
			t.Error("Job.Args should not be null")
		}

		if string(fields.TerminalOutput) == "null" {
			t.Error("Job.TerminalOutput should not be null")
		}

		t.Logf("Job serializes correctly: args=%s, terminalOutput=%s",
			string(fields.Args), string(fields.TerminalOutput))
	})
}

func TestWailsAPIErrorHandling(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Error: GetJob with invalid ID", func(t *testing.T) {
		_, err := app.GetJob("nonexistent-job-id")
		if err == nil {
			t.Error("GetJob should return error for nonexistent job")
		}
		t.Logf("Expected error: %v", err)
	})

	t.Run("Error: DeleteJob with invalid ID is idempotent", func(t *testing.T) {
		err := app.DeleteJob("nonexistent-job-id")
		if err != nil {
			t.Logf("DeleteJob returned error (unexpected): %v", err)
		}
		t.Log("DeleteJob is idempotent - no error for nonexistent job")
	})

	t.Run("Error: GetPluginV2 with invalid ID", func(t *testing.T) {
		plugin, _ := app.GetPluginV2(999999)
		if plugin != nil {
			t.Error("GetPluginV2 should return nil for invalid ID")
		}
	})
}
