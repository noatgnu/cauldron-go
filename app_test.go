package main

import (
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
)

// TestAppInitialization tests that the App initializes correctly
func TestAppInitialization(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	if app.db == nil {
		t.Fatal("Database service was not initialized")
	}

	if app.settings == nil {
		t.Fatal("Settings service was not initialized")
	}

	if app.fileService == nil {
		t.Fatal("File service was not initialized")
	}

	if app.jobQueue == nil {
		t.Fatal("Job queue service was not initialized")
	}

	t.Log("✓ All services initialized successfully")
}

// TestGetSettings tests the GetSettings endpoint
func TestGetSettings(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	config := app.GetSettings()
	if config == nil {
		t.Fatal("GetSettings returned nil")
	}

	t.Logf("✓ GetSettings returned config: %+v", config)
}

// TestJobLifecycle tests creating, retrieving, and deleting a job
func TestJobLifecycle(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

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
		t.Fatalf("Failed to create job: %v", err)
	}

	t.Logf("✓ Created job with ID: %s", jobID)

	// Get the job
	job, err := app.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.Name != "Test PCA Job" {
		t.Errorf("Expected job name 'Test PCA Job', got '%s'", job.Name)
	}

	if job.Type != "pca" {
		t.Errorf("Expected job type 'pca', got '%s'", job.Type)
	}

	t.Log("✓ Retrieved job successfully")

	// Get all jobs
	jobs := app.GetAllJobs()
	if len(jobs) == 0 {
		t.Error("GetAllJobs returned empty list")
	}

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

	t.Logf("✓ GetAllJobs returned %d job(s)", len(jobs))

	// Wait a bit for job to process
	time.Sleep(100 * time.Millisecond)

	// Delete the job
	err = app.DeleteJob(jobID)
	if err != nil {
		t.Fatalf("Failed to delete job: %v", err)
	}

	t.Log("✓ Deleted job successfully")

	// Verify job is deleted
	_, err = app.GetJob(jobID)
	if err == nil {
		t.Error("Expected error when getting deleted job")
	}

	t.Log("✓ Job deleted from database")
}

// TestImportFile tests importing a data file
func TestImportDataFile(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	// Note: This will fail if the file doesn't exist, but tests the flow
	// In a real test, we'd create a temp CSV file first
	t.Log("Note: ImportDataFile test requires actual file - skipping file import")
	t.Log("Testing GetImportedFiles instead...")

	// Test getting imported files (should return empty list on fresh DB)
	files, err := app.GetImportedFiles()
	if err != nil {
		t.Fatalf("Failed to get imported files: %v", err)
	}

	t.Logf("✓ GetImportedFiles returned %d file(s)", len(files))
}

// TestDatabasePersistence tests that data persists across app restarts
func TestDatabasePersistence(t *testing.T) {
	app1 := NewApp()
	app1.Initialize()

	plugins := app1.GetPluginsV2()
	var normPluginID uint
	for _, p := range plugins {
		if p.Definition.Plugin.ID == "normalization" {
			normPluginID = p.ID
			break
		}
	}
	if normPluginID == 0 {
		app1.Shutdown()
		t.Skip("Normalization plugin not installed, skipping test")
	}

	req := models.PluginExecutionRequestV2{
		PluginID: normPluginID,
		Parameters: map[string]interface{}{
			"scaler_type": "minmax",
		},
	}

	jobID, err := app1.ExecutePluginV2(req)
	if err != nil {
		t.Fatalf("Failed to create job in first instance: %v", err)
	}

	t.Logf("Created job with ID: %s in first instance", jobID)

	app1.Shutdown()
	t.Log("Shut down first app instance")

	app2 := NewApp()
	app2.Initialize()
	defer app2.Shutdown()

	job, err := app2.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to retrieve job in second instance: %v", err)
	}

	if job.Name != "Test Persistence Job" {
		t.Errorf("Job name mismatch. Expected 'Test Persistence Job', got '%s'", job.Name)
	}

	t.Log("✓ Job persisted across app restart")

	// Clean up
	err = app2.DeleteJob(jobID)
	if err != nil {
		t.Fatalf("Failed to delete job: %v", err)
	}

	t.Log("✓ Database persistence test passed")
}

// TestVersionDetection tests Python and R version detection
func TestVersionDetection(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	// Test Python version
	pyVersion, err := app.GetPythonVersion()
	if err != nil {
		t.Logf("Python not detected (expected if not installed): %v", err)
	} else {
		t.Logf("✓ Python detected: %s", pyVersion)
	}

	// Test R version
	rVersion, err := app.GetRVersion()
	if err != nil {
		t.Logf("R not detected (expected if not installed): %v", err)
	} else {
		t.Logf("✓ R detected: %s", rVersion)
	}

	t.Log("✓ Version detection test completed")
}
