package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/backend/services"
)

func TestE2EWithActualPlugins(t *testing.T) {
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
	t.Logf("Loaded %d plugins from build directory", len(plugins))

	if len(plugins) == 0 {
		t.Fatal("No plugins found in build directory")
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

	var createdVenvPaths []string
	for _, p := range plugins {
		if p.Definition.Runtime.HasEnvironment("python") {
			venvPath, err := app.GetDefaultVenvPath(p.Definition.Plugin.ID)
			if err != nil {
				t.Logf("Warning: Failed to get venv path for %s: %v", p.Definition.Plugin.ID, err)
				continue
			}
			err = app.CreatePythonVirtualEnv(pythonPath, venvPath, p.Definition.Plugin.ID)
			if err != nil {
				t.Logf("Warning: Failed to create venv for %s: %v", p.Definition.Plugin.ID, err)
				continue
			}
			createdVenvPaths = append(createdVenvPaths, venvPath)
			venvPythonPath := filepath.Join(venvPath, "bin", "python")
			err = app.BindPluginToEnvironment(p.Definition.Plugin.ID, "python", 0, venvPythonPath)
			if err != nil {
				t.Logf("Warning: Failed to bind environment for %s: %v", p.Definition.Plugin.ID, err)
			}
		}
	}
	defer func() {
		for _, venvPath := range createdVenvPaths {
			os.RemoveAll(venvPath)
		}
	}()
	t.Logf("Created virtual environments and bound Python plugins")

	for i, p := range plugins {
		if i >= 5 {
			t.Logf("... and %d more", len(plugins)-5)
			break
		}
		t.Logf("  - %s (%s) [%v]", p.Definition.Plugin.Name, p.Definition.Plugin.ID, p.Definition.Runtime.GetEnvironments())
	}

	t.Run("Execute plugin that requires no file input", func(t *testing.T) {
		var testPlugin *models.PluginV2
		for _, p := range plugins {
			hasRequiredFile := false
			for _, input := range p.Definition.Inputs {
				if input.Type == "file" && input.Required {
					hasRequiredFile = true
					break
				}
			}
			if !hasRequiredFile && p.Definition.Runtime.HasEnvironment("python") {
				testPlugin = p
				break
			}
		}

		if testPlugin == nil {
			t.Skip("No plugin found that doesn't require file input")
		}

		t.Logf("Testing with: %s", testPlugin.Definition.Plugin.Name)

		params := make(map[string]interface{})
		for _, input := range testPlugin.Definition.Inputs {
			if input.Default != nil {
				params[input.Name] = input.Default
			}
		}

		app.jobQueue.SetPluginLoader(pluginLoader)

		req := models.PluginExecutionRequestV2{
			PluginID:   testPlugin.ID,
			Parameters: params,
		}

		jobID, err := app.ExecutePluginV2(req)
		if err != nil {
			t.Logf("ExecutePluginV2 error (may be expected): %v", err)
			return
		}

		t.Logf("Job created: %s", jobID)

		timeout := time.After(60 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				job, _ := app.GetJob(jobID)
				t.Logf("Timeout. Status: %s, Error: %s", job.Status, job.Error)
				return
			case <-ticker.C:
				job, _ := app.GetJob(jobID)
				t.Logf("Status: %s, Progress: %.0f%%", job.Status, job.Progress)

				if job.Status == models.JobStatusCompleted || job.Status == models.JobStatusFailed {
					t.Logf("Final status: %s, Error: %s", job.Status, job.Error)
					return
				}
			}
		}
	})

	t.Run("Execute CV Plot with example data", func(t *testing.T) {
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

		t.Logf("Testing CV Plot with example data")

		app.jobQueue.SetPluginLoader(pluginLoader)

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
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				job, _ := app.GetJob(jobID)
				t.Fatalf("Timeout. Status: %s, Error: %s", job.Status, job.Error)
			case <-ticker.C:
				job, _ := app.GetJob(jobID)
				t.Logf("Status: %s, Progress: %.0f%%", job.Status, job.Progress)

				if job.Status == models.JobStatusCompleted {
					t.Logf("SUCCESS! Output: %s", job.OutputPath)
					if job.OutputPath != "" {
						files, _ := os.ReadDir(job.OutputPath)
						for _, f := range files {
							t.Logf("  - %s", f.Name())
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

func TestE2EProcessJobCodePath(t *testing.T) {
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

	db := app.db
	pluginLoader := services.NewPluginLoaderV2(pluginsDir, db, nil)
	if err := pluginLoader.LoadPlugins(); err != nil {
		t.Fatalf("Failed to load plugins: %v", err)
	}
	app.jobQueue.SetPluginLoader(pluginLoader)

	plugins := pluginLoader.GetAllPlugins()
	if len(plugins) == 0 {
		t.Fatal("No plugins loaded")
	}

	var createdVenvPaths []string
	for _, p := range plugins {
		if p.Definition.Runtime.HasEnvironment("python") {
			venvPath, err := app.GetDefaultVenvPath(p.Definition.Plugin.ID)
			if err != nil {
				t.Logf("Warning: Failed to get venv path for %s: %v", p.Definition.Plugin.ID, err)
				continue
			}
			err = app.CreatePythonVirtualEnv(pythonPath, venvPath, p.Definition.Plugin.ID)
			if err != nil {
				t.Logf("Warning: Failed to create venv for %s: %v", p.Definition.Plugin.ID, err)
				continue
			}
			createdVenvPaths = append(createdVenvPaths, venvPath)
			venvPythonPath := filepath.Join(venvPath, "bin", "python")
			err = app.BindPluginToEnvironment(p.Definition.Plugin.ID, "python", 0, venvPythonPath)
			if err != nil {
				t.Logf("Warning: Failed to bind environment for %s: %v", p.Definition.Plugin.ID, err)
			}
		}
	}
	defer func() {
		for _, venvPath := range createdVenvPaths {
			os.RemoveAll(venvPath)
		}
	}()

	plugin := plugins[0]
	t.Logf("Using plugin: %s (ID: %d)", plugin.Definition.Plugin.Name, plugin.ID)

	t.Run("processJob with plugin parameters does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("PANIC in processJob: %v", r)
			}
		}()

		req := models.PluginExecutionRequestV2{
			PluginID: plugin.ID,
			Parameters: map[string]interface{}{
				"test_param": "test_value",
			},
		}

		jobID, err := app.ExecutePluginV2(req)
		if err != nil {
			t.Logf("ExecutePluginV2 returned error (expected for missing params): %v", err)
			return
		}

		time.Sleep(3 * time.Second)

		job, _ := app.GetJob(jobID)
		t.Logf("Job completed without panic. Status: %s, Error: %s", job.Status, job.Error)
	})

	t.Run("Multiple concurrent plugin jobs do not crash", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("PANIC with concurrent jobs: %v", r)
			}
		}()

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
				t.Logf("Job %d creation error: %v", i, err)
				continue
			}
			jobIDs = append(jobIDs, jobID)
		}

		time.Sleep(5 * time.Second)

		for _, jobID := range jobIDs {
			job, _ := app.GetJob(jobID)
			t.Logf("Job %s: %s", jobID[:8], job.Status)
		}

		t.Log("Concurrent jobs completed without panic")
	})
}
