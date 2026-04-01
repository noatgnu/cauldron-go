package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/backend/services"
)

func TestE2ESettingsManagement(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("Get and verify settings", func(t *testing.T) {
		settings := app.GetSettings()
		if settings == nil {
			t.Fatal("GetSettings returned nil")
		}
		t.Logf("Settings: PythonPath=%s, OutputDir=%s, RPath=%s",
			settings.PythonPath, settings.OutputDirectory, settings.RPath)
	})

	t.Run("SetSetting updates configuration", func(t *testing.T) {
		testOutputDir := "/tmp/test-output-dir"
		err := app.SetSetting("outputDirectory", testOutputDir)
		if err != nil {
			t.Fatalf("SetSetting failed: %v", err)
		}

		settings := app.GetSettings()
		if settings.OutputDirectory != testOutputDir {
			t.Errorf("OutputDirectory not updated, got: %s", settings.OutputDirectory)
		}
		t.Log("SetSetting works correctly")
	})

	t.Run("DetectPythonPath finds Python", func(t *testing.T) {
		pythonPath, err := app.DetectPythonPath()
		if err != nil {
			t.Logf("DetectPythonPath error (may be expected): %v", err)
		} else {
			t.Logf("Detected Python path: %s", pythonPath)
		}
	})

	t.Run("DetectRPath finds R", func(t *testing.T) {
		rPath, err := app.DetectRPath()
		if err != nil {
			t.Logf("DetectRPath error (may be expected if R not installed): %v", err)
		} else {
			t.Logf("Detected R path: %s", rPath)
		}
	})
}

func TestE2EEnvironmentDetection(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("DetectPythonEnvironments returns environments", func(t *testing.T) {
		envs, err := app.DetectPythonEnvironments()
		if err != nil {
			t.Logf("DetectPythonEnvironments error: %v", err)
		}
		t.Logf("Found %d Python environments", len(envs))
		for i, env := range envs {
			if i >= 5 {
				t.Logf("... and %d more", len(envs)-5)
				break
			}
			t.Logf("  - %s (%s) [%s]", env.Name, env.Path, env.Type)
		}
	})

	t.Run("DetectREnvironments returns environments", func(t *testing.T) {
		envs, err := app.DetectREnvironments()
		if err != nil {
			t.Logf("DetectREnvironments error (may be expected): %v", err)
		}
		t.Logf("Found %d R environments", len(envs))
		for i, env := range envs {
			if i >= 3 {
				break
			}
			t.Logf("  - %s (%s)", env.Name, env.Path)
		}
	})

	t.Run("GetVirtualEnvironments returns venvs", func(t *testing.T) {
		venvs, err := app.GetVirtualEnvironments()
		if err != nil {
			t.Fatalf("GetVirtualEnvironments failed: %v", err)
		}
		t.Logf("Found %d virtual environments", len(venvs))
		for _, venv := range venvs {
			t.Logf("  - %s (%s)", venv.Name, venv.Path)
		}
	})

	t.Run("GetRenvEnvironments returns renvs", func(t *testing.T) {
		renvs, err := app.GetRenvEnvironments()
		if err != nil {
			t.Fatalf("GetRenvEnvironments failed: %v", err)
		}
		t.Logf("Found %d R environments", len(renvs))
	})
}

func TestE2EFileOperations(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_data.csv")
	testContent := "col1,col2,col3\na,b,c\n1,2,3\n"
	os.WriteFile(testFilePath, []byte(testContent), 0644)

	t.Run("ReadFile reads file content", func(t *testing.T) {
		content, err := app.ReadFile(testFilePath)
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}
		if len(content) == 0 {
			t.Error("ReadFile returned empty content")
		}
		t.Logf("Read %d bytes from file", len(content))
	})

	t.Run("ReadFilePreview returns limited lines", func(t *testing.T) {
		lines, err := app.ReadFilePreview(testFilePath, 2)
		if err != nil {
			t.Fatalf("ReadFilePreview failed: %v", err)
		}
		if len(lines) > 2 {
			t.Errorf("Expected at most 2 lines, got %d", len(lines))
		}
		t.Logf("Read %d preview lines", len(lines))
	})

	t.Run("ParseDataFile parses CSV", func(t *testing.T) {
		preview, err := app.ParseDataFile(testFilePath, 5)
		if err != nil {
			t.Fatalf("ParseDataFile failed: %v", err)
		}
		if preview == nil {
			t.Fatal("ParseDataFile returned nil")
		}
		t.Logf("Parsed file: %d headers, %d total rows", len(preview.Headers), preview.TotalRows)
	})

	t.Run("ImportDataFile imports file", func(t *testing.T) {
		id, err := app.ImportDataFile(testFilePath)
		if err != nil {
			t.Fatalf("ImportDataFile failed: %v", err)
		}
		t.Logf("Imported file with ID: %d", id)

		files, err := app.GetImportedFiles()
		if err != nil {
			t.Fatalf("GetImportedFiles failed: %v", err)
		}
		t.Logf("Total imported files: %d", len(files))

		err = app.DeleteImportedFile(id)
		if err != nil {
			t.Fatalf("DeleteImportedFile failed: %v", err)
		}
		t.Log("File deleted successfully")
	})
}

func TestE2EJobOutputOperations(t *testing.T) {
	buildDir := os.Getenv("BUILD_DIR")
	if buildDir == "" {
		buildDir = "/mnt/d/GoLandProjects/cauldronGO/build/bin"
	}
	pluginsDir := filepath.Join(buildDir, "plugins")

	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		t.Skipf("Build plugins directory not found: %s", pluginsDir)
	}

	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	db := app.db
	pluginLoader := services.NewPluginLoaderV2(pluginsDir, db, nil)
	if err := pluginLoader.LoadPlugins(); err != nil {
		t.Fatalf("Failed to load plugins: %v", err)
	}

	app.pluginLoaderV2 = pluginLoader
	app.jobQueue.SetPluginLoader(pluginLoader)

	plugins := pluginLoader.GetAllPlugins()
	if len(plugins) == 0 {
		t.Skip("No plugins found")
	}

	var cvPlugin *models.PluginV2
	for _, p := range plugins {
		if p.Definition.Plugin.ID == "cv-plot" {
			cvPlugin = p
			break
		}
	}

	if cvPlugin == nil {
		t.Skip("cv-plot plugin not found")
	}

	settings := app.GetSettings()
	pythonPath := settings.PythonPath
	if pythonPath == "" {
		var err error
		pythonPath, err = app.DetectPythonPath()
		if err != nil {
			t.Skipf("Failed to detect Python: %v", err)
		}
	}
	t.Logf("Using Python at: %s", pythonPath)

	venvPath, err := app.GetDefaultVenvPath(cvPlugin.Definition.Plugin.ID)
	if err != nil {
		t.Fatalf("Failed to get default venv path: %v", err)
	}
	defer os.RemoveAll(venvPath)
	t.Logf("Creating virtual environment at: %s", venvPath)

	err = app.CreatePythonVirtualEnv(pythonPath, venvPath, cvPlugin.Definition.Plugin.ID)
	if err != nil {
		t.Fatalf("Failed to create virtual environment: %v", err)
	}
	t.Logf("Created virtual environment with requirements installed")

	venvPythonPath := filepath.Join(venvPath, "bin", "python")
	err = app.BindPluginToEnvironment(cvPlugin.Definition.Plugin.ID, "python", 0, venvPythonPath)
	if err != nil {
		t.Fatalf("Failed to bind Python environment: %v", err)
	}
	t.Logf("Bound cv-plot plugin to virtual environment: %s", venvPythonPath)

	examplesDir := filepath.Join(buildDir, "examples", "diann")
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

	timeout := time.After(120 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var completedJob *models.Job
WaitLoop:
	for {
		select {
		case <-timeout:
			job, _ := app.GetJob(jobID)
			t.Fatalf("Job timed out. Status: %s, Error: %s", job.Status, job.Error)
		case <-ticker.C:
			job, _ := app.GetJob(jobID)
			if job.Status == models.JobStatusCompleted {
				completedJob = job
				break WaitLoop
			}
			if job.Status == models.JobStatusFailed {
				t.Fatalf("Job failed: %s", job.Error)
			}
		}
	}

	t.Run("ListJobOutputFiles returns files", func(t *testing.T) {
		files, err := app.ListJobOutputFiles(jobID)
		if err != nil {
			t.Fatalf("ListJobOutputFiles failed: %v", err)
		}
		t.Logf("Output files: %v", files)
		if len(files) == 0 {
			t.Error("Expected output files")
		}
	})

	t.Run("GetJobExecutionLog returns log content", func(t *testing.T) {
		log, err := app.GetJobExecutionLog(jobID)
		if err != nil {
			t.Fatalf("GetJobExecutionLog failed: %v", err)
		}
		t.Logf("Execution log length: %d bytes", len(log))
	})

	t.Run("ReadJobOutputFile reads SVG", func(t *testing.T) {
		content, err := app.ReadJobOutputFile(jobID, "pr_cv.svg")
		if err != nil {
			t.Logf("ReadJobOutputFile error: %v", err)
		} else {
			t.Logf("Read SVG file: %d bytes", len(content))
		}
	})

	t.Run("WriteJobOutputFile writes file", func(t *testing.T) {
		testContent := "test output content"
		err := app.WriteJobOutputFile(jobID, "test_output.txt", testContent)
		if err != nil {
			t.Fatalf("WriteJobOutputFile failed: %v", err)
		}

		readContent, err := app.ReadJobOutputFile(jobID, "test_output.txt")
		if err != nil {
			t.Fatalf("Failed to read written file: %v", err)
		}
		if readContent != testContent {
			t.Errorf("Content mismatch: got %s", readContent)
		}
		t.Log("WriteJobOutputFile works correctly")
	})

	t.Logf("Job output path: %s", completedJob.OutputPath)
}

func TestE2EJobManagement(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	plugins := app.GetPluginsV2()
	if len(plugins) == 0 {
		t.Skip("No plugins installed")
	}

	plugin := plugins[0]
	req := models.PluginExecutionRequestV2{
		PluginID: plugin.ID,
		Parameters: map[string]interface{}{
			"test": "value",
		},
	}

	jobID, err := app.ExecutePluginV2(req)
	if err != nil {
		t.Logf("Job creation error (expected): %v", err)
		t.Skip("Could not create job for testing")
	}

	time.Sleep(2 * time.Second)

	t.Run("GetJob retrieves job details", func(t *testing.T) {
		job, err := app.GetJob(jobID)
		if err != nil {
			t.Fatalf("GetJob failed: %v", err)
		}
		t.Logf("Job: ID=%s, Status=%s, Type=%s", job.ID, job.Status, job.Type)
	})

	t.Run("GetAllJobs includes created job", func(t *testing.T) {
		jobs := app.GetAllJobs()
		found := false
		for _, j := range jobs {
			if j.ID == jobID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Created job not found in GetAllJobs")
		}
		t.Logf("Total jobs: %d", len(jobs))
	})

	t.Run("DeleteJob removes job", func(t *testing.T) {
		err := app.DeleteJob(jobID)
		if err != nil {
			t.Fatalf("DeleteJob failed: %v", err)
		}

		_, err = app.GetJob(jobID)
		if err == nil {
			t.Error("Job should have been deleted")
		}
		t.Log("DeleteJob works correctly")
	})
}

func TestE2EPluginManagement(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("GetPluginsV2 returns plugins list", func(t *testing.T) {
		plugins := app.GetPluginsV2()
		if plugins == nil {
			t.Fatal("GetPluginsV2 returned nil")
		}
		t.Logf("Total plugins: %d", len(plugins))
		for i, p := range plugins {
			if i >= 5 {
				t.Logf("... and %d more", len(plugins)-5)
				break
			}
			t.Logf("  - %s (%s) [enabled: %v]", p.Definition.Plugin.Name, p.Definition.Plugin.ID, p.Enabled)
		}
	})

	t.Run("GetPluginV2 retrieves specific plugin", func(t *testing.T) {
		plugins := app.GetPluginsV2()
		if len(plugins) == 0 {
			t.Skip("No plugins to test")
		}

		plugin, err := app.GetPluginV2(plugins[0].ID)
		if err != nil {
			t.Fatalf("GetPluginV2 failed: %v", err)
		}
		t.Logf("Plugin: %s, Version: %s", plugin.Definition.Plugin.Name, plugin.Definition.Plugin.Version)
	})

	t.Run("SetPluginEnabled toggles plugin state", func(t *testing.T) {
		plugins := app.GetPluginsV2()
		if len(plugins) == 0 {
			t.Skip("No plugins to test")
		}

		plugin := plugins[0]
		originalState := plugin.Enabled

		err := app.SetPluginEnabled(plugin.ID, !originalState)
		if err != nil {
			t.Fatalf("SetPluginEnabled failed: %v", err)
		}

		err = app.SetPluginEnabled(plugin.ID, originalState)
		if err != nil {
			t.Fatalf("Failed to restore plugin state: %v", err)
		}
		t.Log("SetPluginEnabled works correctly")
	})

	t.Run("ReloadPluginsV2 reloads plugins", func(t *testing.T) {
		err := app.ReloadPluginsV2()
		if err != nil {
			t.Fatalf("ReloadPluginsV2 failed: %v", err)
		}
		t.Log("ReloadPluginsV2 works correctly")
	})

	t.Run("ValidatePluginYAML validates correct YAML", func(t *testing.T) {
		validYAML := `
plugin:
  id: test-plugin
  name: Test Plugin
  version: "1.0.0"
  description: A test plugin
  author: Test Author
  category: analysis

runtime:
  environments:
    - python
  entrypoint: main.py

inputs:
  - name: input_file
    type: file
    label: Input File
    required: true

outputs: []
`
		valid, errors, err := app.ValidatePluginYAML(validYAML)
		if err != nil {
			t.Fatalf("ValidatePluginYAML failed: %v", err)
		}
		if !valid {
			t.Errorf("Expected valid YAML, got errors: %v", errors)
		}
		t.Log("ValidatePluginYAML works for valid YAML")
	})

	t.Run("ValidatePluginYAML detects invalid YAML", func(t *testing.T) {
		invalidYAML := `
plugin:
  name: Missing ID Plugin
`
		valid, errors, err := app.ValidatePluginYAML(invalidYAML)
		if err != nil {
			t.Fatalf("ValidatePluginYAML failed: %v", err)
		}
		if valid {
			t.Error("Expected invalid YAML to fail validation")
		}
		t.Logf("Validation errors: %v", errors)
	})

	t.Run("ParsePluginYAML parses YAML", func(t *testing.T) {
		yamlContent := `
plugin:
  id: test-plugin
  name: Test Plugin
  version: "1.0.0"

runtime:
  environments:
    - python
  entrypoint: main.py
`
		definition, err := app.ParsePluginYAML(yamlContent)
		if err != nil {
			t.Fatalf("ParsePluginYAML failed: %v", err)
		}
		if definition.Plugin.ID != "test-plugin" {
			t.Errorf("Expected plugin ID 'test-plugin', got '%s'", definition.Plugin.ID)
		}
		t.Log("ParsePluginYAML works correctly")
	})

	t.Run("ConvertPluginToYAML converts definition", func(t *testing.T) {
		definition := models.PluginDefinition{
			Plugin: models.PluginMetadata{
				ID:      "converted-plugin",
				Name:    "Converted Plugin",
				Version: "1.0.0",
			},
			Runtime: models.PluginRuntimeV2{
				Environments: []string{"python"},
				Entrypoint:   "main.py",
			},
		}

		yaml, err := app.ConvertPluginToYAML(definition)
		if err != nil {
			t.Fatalf("ConvertPluginToYAML failed: %v", err)
		}
		t.Logf("Generated YAML length: %d bytes", len(yaml))
	})
}

func TestE2EEnvironmentBindings(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	plugins := app.GetPluginsV2()
	if len(plugins) == 0 {
		t.Skip("No plugins to test")
	}

	pluginID := plugins[0].Definition.Plugin.ID
	testEnvPath := "/test/python/path"

	t.Run("BindPluginToEnvironment creates binding", func(t *testing.T) {
		err := app.BindPluginToEnvironment(pluginID, "python", 1, testEnvPath)
		if err != nil {
			t.Fatalf("BindPluginToEnvironment failed: %v", err)
		}
		t.Log("Created environment binding")
	})

	t.Run("GetPluginEnvironmentBinding retrieves binding", func(t *testing.T) {
		binding, err := app.GetPluginEnvironmentBinding(pluginID, "python")
		if err != nil {
			t.Fatalf("GetPluginEnvironmentBinding failed: %v", err)
		}
		if binding.EnvironmentPath != testEnvPath {
			t.Errorf("Expected path %s, got %s", testEnvPath, binding.EnvironmentPath)
		}
		t.Logf("Binding: %+v", binding)
	})

	t.Run("GetAllPluginEnvironmentBindings returns bindings", func(t *testing.T) {
		bindings, err := app.GetAllPluginEnvironmentBindings()
		if err != nil {
			t.Fatalf("GetAllPluginEnvironmentBindings failed: %v", err)
		}
		t.Logf("Total bindings: %d", len(bindings))
	})

	t.Run("DeletePluginEnvironmentBinding removes binding", func(t *testing.T) {
		err := app.DeletePluginEnvironmentBinding(pluginID, "python")
		if err != nil {
			t.Fatalf("DeletePluginEnvironmentBinding failed: %v", err)
		}

		binding, _ := app.GetPluginEnvironmentBinding(pluginID, "python")
		if binding != nil {
			t.Error("Binding should have been deleted")
		}
		t.Log("Deleted environment binding")
	})
}

func TestE2ECustomEnvVars(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	plugins := app.GetPluginsV2()
	if len(plugins) == 0 {
		t.Skip("No plugins to test")
	}

	pluginID := plugins[0].ID

	t.Run("SaveCustomEnvVar creates env var", func(t *testing.T) {
		envVar := services.CustomEnvVar{
			PluginID: pluginID,
			Key:      "TEST_VAR",
			Value:    "test_value",
		}
		err := app.SaveCustomEnvVar(envVar)
		if err != nil {
			t.Fatalf("SaveCustomEnvVar failed: %v", err)
		}
		t.Log("Saved custom env var")
	})

	t.Run("GetCustomEnvVars retrieves env vars", func(t *testing.T) {
		vars, err := app.GetCustomEnvVars(pluginID)
		if err != nil {
			t.Fatalf("GetCustomEnvVars failed: %v", err)
		}
		t.Logf("Found %d env vars for plugin", len(vars))
	})

	t.Run("GetGlobalCustomEnvVars retrieves global vars", func(t *testing.T) {
		vars, err := app.GetGlobalCustomEnvVars()
		if err != nil {
			t.Fatalf("GetGlobalCustomEnvVars failed: %v", err)
		}
		t.Logf("Found %d global env vars", len(vars))
	})

	t.Run("DeleteCustomEnvVarByKey removes env var", func(t *testing.T) {
		err := app.DeleteCustomEnvVarByKey(pluginID, "TEST_VAR")
		if err != nil {
			t.Fatalf("DeleteCustomEnvVarByKey failed: %v", err)
		}
		t.Log("Deleted custom env var")
	})
}

func TestE2EVersionChecks(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("GetPythonVersion returns version", func(t *testing.T) {
		version, err := app.GetPythonVersion()
		if err != nil {
			t.Logf("GetPythonVersion error: %v", err)
		} else {
			t.Logf("Python version: %s", version)
		}
	})

	t.Run("GetRVersion returns version", func(t *testing.T) {
		version, err := app.GetRVersion()
		if err != nil {
			t.Logf("GetRVersion error (R may not be installed): %v", err)
		} else {
			t.Logf("R version: %s", version)
		}
	})

	t.Run("CheckDockerVersion returns version", func(t *testing.T) {
		version, err := app.CheckDockerVersion()
		if err != nil {
			t.Logf("CheckDockerVersion error (Docker may not be installed): %v", err)
		} else {
			t.Logf("Docker version: %s", version)
		}
	})
}

func TestE2ELicenseInfo(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("GetLicenseInfo returns licenses", func(t *testing.T) {
		licenses, err := app.GetLicenseInfo()
		if err != nil {
			t.Fatalf("GetLicenseInfo failed: %v", err)
		}
		t.Logf("Go licenses: %d, NPM licenses: %d", len(licenses.Go), len(licenses.NPM))

		if len(licenses.Go) > 0 {
			t.Logf("Sample Go license: %s (%s)", licenses.Go[0].Name, licenses.Go[0].License)
		}
		if len(licenses.NPM) > 0 {
			t.Logf("Sample NPM license: %s (%s)", licenses.NPM[0].Name, licenses.NPM[0].License)
		}
	})
}

func TestE2ELogging(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("LogToFile logs message", func(t *testing.T) {
		err := app.LogToFile("Test log message from E2E test")
		if err != nil {
			t.Fatalf("LogToFile failed: %v", err)
		}
		t.Log("LogToFile works correctly")
	})

	t.Run("GetLogFilePath returns path", func(t *testing.T) {
		path, err := app.GetLogFilePath()
		if err != nil {
			t.Logf("GetLogFilePath error: %v", err)
		} else {
			t.Logf("Log file path: %s", path)
		}
	})
}

func TestE2EPluginRequirements(t *testing.T) {
	buildDir := os.Getenv("BUILD_DIR")
	if buildDir == "" {
		buildDir = "/mnt/d/GoLandProjects/cauldronGO/build/bin"
	}
	pluginsDir := filepath.Join(buildDir, "plugins")

	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		t.Skipf("Build plugins directory not found: %s", pluginsDir)
	}

	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	db := app.db
	pluginLoader := services.NewPluginLoaderV2(pluginsDir, db, nil)
	if err := pluginLoader.LoadPlugins(); err != nil {
		t.Fatalf("Failed to load plugins: %v", err)
	}

	app.pluginLoaderV2 = pluginLoader

	plugins := pluginLoader.GetAllPlugins()
	if len(plugins) == 0 {
		t.Skip("No plugins found")
	}

	t.Run("GetPluginRequirements returns requirements info", func(t *testing.T) {
		for _, plugin := range plugins {
			info, err := app.GetPluginRequirements(plugin.Definition.Plugin.ID)
			if err != nil {
				continue
			}
			if info.RequirementsExist {
				t.Logf("Plugin %s has requirements:", info.PluginName)
				t.Logf("  Python packages: %d", len(info.PythonPackages))
				t.Logf("  R packages: %d", len(info.RPackages))
				return
			}
		}
		t.Log("No plugins with requirements found")
	})
}

func TestE2EJobQueueAdvanced(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("HasInProgressJobs returns status", func(t *testing.T) {
		hasJobs := app.HasInProgressJobs()
		t.Logf("Has in-progress jobs: %v", hasJobs)
	})

	t.Run("ProcessPendingJobs processes queue", func(t *testing.T) {
		err := app.ProcessPendingJobs()
		if err != nil {
			t.Fatalf("ProcessPendingJobs failed: %v", err)
		}
		t.Log("ProcessPendingJobs works correctly")
	})

	t.Run("Queue pause/resume/status cycle", func(t *testing.T) {
		status := app.GetJobQueueStatus()
		t.Logf("Initial status: %+v", status)

		err := app.PauseJobQueue()
		if err != nil {
			t.Fatalf("PauseJobQueue failed: %v", err)
		}

		status = app.GetJobQueueStatus()
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

		t.Log("Queue pause/resume cycle works correctly")
	})
}

func TestE2ETempFileOperations(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("SaveTempFile saves and returns path", func(t *testing.T) {
		content := "test content for temp file"
		filename := "e2e_test_temp.txt"

		path, err := app.SaveTempFile(filename, content)
		if err != nil {
			t.Fatalf("SaveTempFile failed: %v", err)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Temp file was not created at: %s", path)
		}

		readContent, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read temp file: %v", err)
		}

		if string(readContent) != content {
			t.Errorf("Content mismatch: expected %s, got %s", content, string(readContent))
		}

		os.Remove(path)
		t.Logf("SaveTempFile created file at: %s", path)
	})
}

func TestE2EDefaultVenvPath(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("GetDefaultVenvPath returns valid path", func(t *testing.T) {
		path, err := app.GetDefaultVenvPath("test-plugin")
		if err != nil {
			t.Fatalf("GetDefaultVenvPath failed: %v", err)
		}

		if path == "" {
			t.Error("GetDefaultVenvPath returned empty path")
		}

		t.Logf("Default venv path: %s", path)
	})
}

func TestE2EExampleFiles(t *testing.T) {
	buildDir := os.Getenv("BUILD_DIR")
	if buildDir == "" {
		buildDir = "/mnt/d/GoLandProjects/cauldronGO/build/bin"
	}
	examplesDir := filepath.Join(buildDir, "examples")

	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		t.Skipf("Examples directory not found: %s", examplesDir)
	}

	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	time.Sleep(500 * time.Millisecond)

	t.Run("GetExampleFilePath returns path for diann", func(t *testing.T) {
		path, err := app.GetExampleFilePath("diann", "annotation.txt")
		if err != nil {
			t.Logf("GetExampleFilePath error: %v", err)
		} else {
			t.Logf("Example file path: %s", path)
		}
	})
}
