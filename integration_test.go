package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
)

// TestWailsBindings tests that all Wails-exposed methods work correctly
func TestWailsBindings(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("GetAllJobs", func(t *testing.T) {
		jobs := app.GetAllJobs()
		if jobs == nil {
			t.Error("GetAllJobs returned nil")
		}
		t.Logf("✓ GetAllJobs returned %d jobs", len(jobs))

		// Verify it returns a slice, not null
		jsonBytes, err := json.Marshal(jobs)
		if err != nil {
			t.Fatalf("Failed to marshal jobs: %v", err)
		}
		t.Logf("✓ JSON: %s", string(jsonBytes))

		// Should be [] not null
		if len(jobs) == 0 && string(jsonBytes) != "[]" {
			t.Errorf("Empty jobs should serialize to [], got: %s", string(jsonBytes))
		}
	})

	t.Run("GetImportedFiles", func(t *testing.T) {
		files, err := app.GetImportedFiles()
		if err != nil {
			t.Fatalf("GetImportedFiles failed: %v", err)
		}
		if files == nil {
			t.Error("GetImportedFiles returned nil")
		}
		t.Logf("✓ GetImportedFiles returned %d files", len(files))

		jsonBytes, err := json.Marshal(files)
		if err != nil {
			t.Fatalf("Failed to marshal files: %v", err)
		}
		t.Logf("✓ JSON: %s", string(jsonBytes))

		if len(files) == 0 && string(jsonBytes) != "[]" {
			t.Errorf("Empty files should serialize to [], got: %s", string(jsonBytes))
		}
	})

	t.Run("GetSettings", func(t *testing.T) {
		config := app.GetSettings()
		if config == nil {
			t.Fatal("GetSettings returned nil")
		}
		t.Logf("✓ GetSettings returned: %+v", config)

		jsonBytes, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}
		t.Logf("✓ JSON: %s", string(jsonBytes))
	})

	t.Run("CreateAndGetJob", func(t *testing.T) {
		// Get PCA plugin ID
		plugins := app.GetPluginsV2()
		var pcaPluginID uint
		for _, p := range plugins {
			if p.Definition.Plugin.ID == "pca-analysis" {
				pcaPluginID = p.ID
				break
			}
		}
		if pcaPluginID == 0 {
			t.Skip("PCA plugin not installed, skipping test")
		}

		// Create a job using Plugin V2
		req := models.PluginExecutionRequestV2{
			PluginID: pcaPluginID,
			Parameters: map[string]interface{}{
				"n_components": 2,
			},
		}

		jobID, err := app.ExecutePluginV2(req)
		if err != nil {
			t.Fatalf("ExecutePluginV2 failed: %v", err)
		}
		t.Logf("✓ ExecutePluginV2 returned ID: %s", jobID)

		// Get the job
		job, err := app.GetJob(jobID)
		if err != nil {
			t.Fatalf("GetJob failed: %v", err)
		}

		jsonBytes, err := json.Marshal(job)
		if err != nil {
			t.Fatalf("Failed to marshal job: %v", err)
		}
		t.Logf("✓ GetJob JSON: %s", string(jsonBytes))

		// Verify fields
		if job.ID != jobID {
			t.Errorf("Job ID mismatch: expected %s, got %s", jobID, job.ID)
		}
		if job.Name != "Integration Test Job" {
			t.Errorf("Job name mismatch: expected 'Integration Test Job', got '%s'", job.Name)
		}
		if job.Type != "pca" {
			t.Errorf("Job type mismatch: expected 'pca', got '%s'", job.Type)
		}

		// Check that Args serializes properly (should be array, not null)
		type JobJSON struct {
			Args json.RawMessage `json:"args"`
		}
		var jobData JobJSON
		if err := json.Unmarshal(jsonBytes, &jobData); err != nil {
			t.Fatalf("Failed to unmarshal job: %v", err)
		}

		argsStr := string(jobData.Args)
		t.Logf("✓ Args serializes as: %s", argsStr)

		if argsStr == "null" {
			t.Error("Args should be an array [], not null")
		}

		// Clean up
		app.DeleteJob(jobID)
	})

	t.Run("GetPythonVersion", func(t *testing.T) {
		version, err := app.GetPythonVersion()
		if err != nil {
			t.Logf("Python not detected (OK): %v", err)
		} else {
			t.Logf("✓ Python version: %s", version)
		}
	})

	t.Run("GetRVersion", func(t *testing.T) {
		version, err := app.GetRVersion()
		if err != nil {
			t.Logf("R not detected (OK): %v", err)
		} else {
			t.Logf("✓ R version: %s", version)
		}
	})
}

// TestEmptyArraySerialization specifically tests that empty arrays serialize correctly
func TestEmptyArraySerialization(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	// Test with fresh database - should return empty arrays
	jobs := app.GetAllJobs()

	// Marshal to JSON like Wails would
	jsonBytes, err := json.Marshal(jobs)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(jsonBytes)
	t.Logf("GetAllJobs() serializes to: %s", jsonStr)

	// CRITICAL: Frontend expects [] not null
	if jsonStr == "null" {
		t.Error("FAIL: GetAllJobs() serializes to null, frontend expects []")
		t.Error("This will cause the infinite loading spinner!")
	} else if jsonStr == "[]" {
		t.Log("✓ PASS: GetAllJobs() correctly serializes to []")
	} else {
		t.Logf("✓ GetAllJobs() has data: %s", jsonStr)
	}
}

// TestJobQueueServiceGetAllJobs tests the job queue service directly
func TestJobQueueServiceGetAllJobs(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	// Get jobs directly from service
	jobs := app.jobQueue.GetAllJobs()

	t.Logf("JobQueue.GetAllJobs() returned: %v", jobs)
	t.Logf("Type: %T", jobs)
	t.Logf("Is nil: %v", jobs == nil)
	t.Logf("Length: %d", len(jobs))

	// Marshal to see what it looks like
	jsonBytes, _ := json.Marshal(jobs)
	t.Logf("Serializes to: %s", string(jsonBytes))

	if jobs == nil {
		t.Error("CRITICAL: GetAllJobs() returns nil, should return empty slice")
	}
}

func TestSettingsWorkflow(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Get and modify settings", func(t *testing.T) {
		config := app.GetSettings()
		if config == nil {
			t.Fatal("GetSettings returned nil")
		}

		t.Logf("Current config: PythonPath=%s, ResultStoragePath=%s", config.PythonPath, config.ResultStoragePath)

		err := app.SetSetting("accessibilityFontScale", "125")
		if err != nil {
			t.Fatalf("SetSetting failed: %v", err)
		}

		updatedConfig := app.GetSettings()
		t.Logf("Updated FontScale: %s", updatedConfig.AccessibilityFontScale)

		err = app.SetSetting("accessibilityFontScale", "100")
		if err != nil {
			t.Fatalf("SetSetting (restore) failed: %v", err)
		}

		t.Logf("Settings workflow completed successfully")
	})

	t.Run("Verify config paths exist", func(t *testing.T) {
		config := app.GetSettings()

		if config.ResultStoragePath == "" {
			t.Error("ResultStoragePath should not be empty")
		}

		t.Logf("ResultStoragePath: %s", config.ResultStoragePath)
		t.Logf("OutputDirectory: %s", config.OutputDirectory)
		t.Logf("PythonPath: %s", config.PythonPath)
	})
}

func TestJobQueueWorkflow(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Complete job queue workflow", func(t *testing.T) {
		initialStatus := app.GetJobQueueStatus()
		t.Logf("Initial queue status: %+v", initialStatus)

		err := app.PauseJobQueue()
		if err != nil {
			t.Fatalf("PauseJobQueue failed: %v", err)
		}

		pausedStatus := app.GetJobQueueStatus()
		if paused, ok := pausedStatus["paused"].(bool); !ok || !paused {
			t.Error("Queue should be paused")
		}

		err = app.ResumeJobQueue()
		if err != nil {
			t.Fatalf("ResumeJobQueue failed: %v", err)
		}

		resumedStatus := app.GetJobQueueStatus()
		if paused, ok := resumedStatus["paused"].(bool); ok && paused {
			t.Error("Queue should not be paused")
		}

		t.Log("Job queue workflow completed successfully")
	})
}

func TestJobCreationAndRetrieval(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Create job directly and verify retrieval", func(t *testing.T) {
		jobID, err := app.jobQueue.CreateJob("test", "Integration Test Job", "", []string{"arg1", "arg2"})
		if err != nil {
			t.Fatalf("CreateJob failed: %v", err)
		}
		defer app.DeleteJob(jobID)

		t.Logf("Created job: %s", jobID)

		job, err := app.GetJob(jobID)
		if err != nil {
			t.Fatalf("GetJob failed: %v", err)
		}

		if job.ID != jobID {
			t.Errorf("Job ID mismatch: expected %s, got %s", jobID, job.ID)
		}

		if job.Name != "Integration Test Job" {
			t.Errorf("Job name mismatch: expected 'Integration Test Job', got '%s'", job.Name)
		}

		if job.Type != "test" {
			t.Errorf("Job type mismatch: expected 'test', got '%s'", job.Type)
		}

		if len(job.Args) != 2 {
			t.Errorf("Job args length mismatch: expected 2, got %d", len(job.Args))
		}

		allJobs := app.GetAllJobs()
		found := false
		for _, j := range allJobs {
			if j.ID == jobID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Created job not found in GetAllJobs")
		}

		t.Log("Job creation and retrieval workflow completed successfully")
	})

	t.Run("Verify job serialization", func(t *testing.T) {
		jobID, _ := app.jobQueue.CreateJob("test", "Serialization Test", "", []string{})
		defer app.DeleteJob(jobID)

		job, _ := app.GetJob(jobID)
		jsonBytes, err := json.Marshal(job)
		if err != nil {
			t.Fatalf("JSON serialization failed: %v", err)
		}

		jsonStr := string(jsonBytes)

		if strings.Contains(jsonStr, `"args":null`) {
			t.Error("args should not be null in JSON")
		}

		if strings.Contains(jsonStr, `"terminalOutput":null`) {
			t.Error("terminalOutput should not be null in JSON")
		}

		t.Logf("Job JSON: %s", jsonStr)
	})
}

func TestEnvironmentDetection(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Detect Python installations", func(t *testing.T) {
		envs, err := app.DetectPythonEnvironments()
		if err != nil {
			t.Logf("Python detection returned error (may be expected): %v", err)
			return
		}

		t.Logf("Found %d Python environments", len(envs))
		for i, env := range envs {
			t.Logf("  [%d] %s (v%s, virtual: %v)", i, env.Path, env.Version, env.IsVirtual)
		}
	})

	t.Run("Get Python version", func(t *testing.T) {
		version, err := app.GetPythonVersion()
		if err != nil {
			t.Logf("GetPythonVersion error: %v", err)
			return
		}

		if !strings.Contains(version, "Python") && !strings.Contains(version, ".") {
			t.Errorf("Unexpected Python version format: %s", version)
		}

		t.Logf("Python version: %s", version)
	})

	t.Run("Detect R installations", func(t *testing.T) {
		envs, err := app.DetectREnvironments()
		if err != nil {
			t.Logf("R detection returned error (may be expected): %v", err)
			return
		}

		t.Logf("Found %d R environments", len(envs))
		for i, env := range envs {
			t.Logf("  [%d] %s (v%s)", i, env.Path, env.Version)
		}
	})

	t.Run("Check Docker availability", func(t *testing.T) {
		version, err := app.CheckDockerVersion()
		if err != nil {
			t.Logf("Docker not available: %v", err)
			return
		}

		t.Logf("Docker version: %s", version)
	})
}

func TestPluginOperations(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("List and inspect plugins", func(t *testing.T) {
		plugins := app.GetPluginsV2()
		if plugins == nil {
			t.Fatal("GetPluginsV2 returned nil")
		}

		t.Logf("Found %d plugins", len(plugins))

		for _, p := range plugins {
			t.Logf("Plugin: %s (ID: %d)", p.Definition.Plugin.Name, p.ID)
			t.Logf("  Version: %s", p.Definition.Plugin.Version)
			t.Logf("  Category: %s", p.Definition.Plugin.Category)
			t.Logf("  Environments: %v", p.Definition.Runtime.Environments)
			t.Logf("  Inputs: %d, Outputs: %d", len(p.Definition.Inputs), len(p.Definition.Outputs))

			fetchedPlugin, err := app.GetPluginV2(p.ID)
			if err != nil {
				t.Errorf("GetPluginV2 failed for ID %d: %v", p.ID, err)
				continue
			}

			if fetchedPlugin.ID != p.ID {
				t.Errorf("Plugin ID mismatch: expected %d, got %d", p.ID, fetchedPlugin.ID)
			}
		}
	})

	t.Run("Validate plugin YAML schema", func(t *testing.T) {
		yamlConfigs := []struct {
			name    string
			yaml    string
			isValid bool
		}{
			{
				name: "valid minimal plugin",
				yaml: `plugin:
  id: test-plugin
  name: Test Plugin
  description: A test plugin
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
`,
				isValid: true,
			},
			{
				name: "invalid - missing required fields",
				yaml: `plugin:
  id: incomplete
`,
				isValid: false,
			},
		}

		for _, tc := range yamlConfigs {
			valid, errors, err := app.ValidatePluginYAML(tc.yaml)
			if err != nil {
				t.Logf("[%s] Validation error: %v", tc.name, err)
				continue
			}

			if valid != tc.isValid {
				t.Errorf("[%s] Expected valid=%v, got valid=%v, errors=%v", tc.name, tc.isValid, valid, errors)
			} else {
				t.Logf("[%s] Validation result: valid=%v", tc.name, valid)
			}
		}
	})

	t.Run("Get plugins directory", func(t *testing.T) {
		pluginsDir := app.GetPluginsDirectory()
		if pluginsDir == "" {
			t.Error("GetPluginsDirectory returned empty string")
		}
		t.Logf("Plugins directory: %s", pluginsDir)
	})
}

func TestVirtualEnvironments(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("List virtual environments", func(t *testing.T) {
		venvs, err := app.GetVirtualEnvironments()
		if err != nil {
			t.Fatalf("GetVirtualEnvironments failed: %v", err)
		}

		t.Logf("Found %d virtual environments", len(venvs))
		for _, v := range venvs {
			t.Logf("  Venv ID %d: %s (Base: %s)", v.ID, v.Path, v.BasePythonPath)
		}
	})

	t.Run("List renv environments", func(t *testing.T) {
		renvs, err := app.GetRenvEnvironments()
		if err != nil {
			t.Fatalf("GetRenvEnvironments failed: %v", err)
		}

		t.Logf("Found %d renv environments", len(renvs))
		for _, r := range renvs {
			t.Logf("  Renv ID %d: %s", r.ID, r.Path)
		}
	})

	t.Run("List plugin environment bindings", func(t *testing.T) {
		bindings, err := app.GetAllPluginEnvironmentBindings()
		if err != nil {
			t.Fatalf("GetAllPluginEnvironmentBindings failed: %v", err)
		}

		t.Logf("Found %d plugin environment bindings", len(bindings))
		for _, b := range bindings {
			t.Logf("  Plugin %s -> %s env %d (%s)", b.PluginID, b.EnvironmentType, b.EnvironmentID, b.EnvironmentPath)
		}
	})
}

func TestFileOperations(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("List imported files", func(t *testing.T) {
		files, err := app.GetImportedFiles()
		if err != nil {
			t.Fatalf("GetImportedFiles failed: %v", err)
		}

		t.Logf("Found %d imported files", len(files))
		for _, f := range files {
			t.Logf("  File ID %d: %s (%s, %d bytes)", f.ID, f.Name, f.FileType, f.Size)
		}
	})

	t.Run("Import and delete a test file", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test_data.csv")

		content := "col1,col2,col3\n1,2,3\n4,5,6\n7,8,9\n"
		err := os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		fileID, err := app.ImportDataFile(testFile)
		if err != nil {
			t.Fatalf("ImportDataFile failed: %v", err)
		}

		t.Logf("Imported file with ID: %d", fileID)

		files, _ := app.GetImportedFiles()
		found := false
		for _, f := range files {
			if f.ID == fileID {
				found = true
				if f.Name != "test_data.csv" {
					t.Errorf("Imported file name mismatch: expected 'test_data.csv', got '%s'", f.Name)
				}
				break
			}
		}
		if !found {
			t.Error("Imported file not found in list")
		}

		err = app.DeleteImportedFile(fileID)
		if err != nil {
			t.Fatalf("DeleteImportedFile failed: %v", err)
		}

		files, _ = app.GetImportedFiles()
		for _, f := range files {
			if f.ID == fileID {
				t.Error("File should have been deleted")
			}
		}

		t.Log("File import/delete workflow completed successfully")
	})
}

func TestConcurrentAccess(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("Concurrent API calls", func(t *testing.T) {
		done := make(chan bool, 10)

		go func() {
			for i := 0; i < 5; i++ {
				_ = app.GetSettings()
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 5; i++ {
				_ = app.GetAllJobs()
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 5; i++ {
				_ = app.GetPluginsV2()
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 5; i++ {
				_ = app.GetJobQueueStatus()
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		timeout := time.After(10 * time.Second)
		completed := 0
		for completed < 4 {
			select {
			case <-done:
				completed++
			case <-timeout:
				t.Fatal("Concurrent access test timed out")
			}
		}

		t.Log("Concurrent access test completed successfully")
	})
}
