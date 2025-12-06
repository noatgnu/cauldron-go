package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
)

// TestWailsBindings tests that all Wails-exposed methods work correctly
func TestWailsBindings(t *testing.T) {
	ctx := context.WithValue(context.Background(), "wails-test", true)
	app := NewApp()
	app.startup(ctx)
	defer app.shutdown(ctx)

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
		// Create a job
		req := models.JobRequest{
			Type: "pca",
			Name: "Integration Test Job",
		}

		jobID, err := app.CreateJob(req)
		if err != nil {
			t.Fatalf("CreateJob failed: %v", err)
		}
		t.Logf("✓ CreateJob returned ID: %s", jobID)

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
	ctx := context.WithValue(context.Background(), "wails-test", true)
	app := NewApp()
	app.startup(ctx)
	defer app.shutdown(ctx)

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
	ctx := context.WithValue(context.Background(), "wails-test", true)
	app := NewApp()
	app.startup(ctx)
	defer app.shutdown(ctx)

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
