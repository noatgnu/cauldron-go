package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func TestFullPluginWorkflowStability(t *testing.T) {
	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	repoURL := "https://github.com/noatgnu/diann-curtainptm-converter-plugin"

	t.Run("Step 1: Fetch Dependencies", func(t *testing.T) {
		result, err := app.FetchPluginDependencies(repoURL)
		if err != nil {
			t.Fatalf("FetchPluginDependencies failed: %v", err)
		}
		if result["id"] == "" {
			t.Fatal("Fetched plugin ID is empty")
		}
	})

	t.Run("Step 2: Install Plugin", func(t *testing.T) {
		installed, _ := app.IsPluginInstalled(repoURL)
		if installed {
			app.UninstallPluginFromRepo(repoURL, true, true, true)
		}

		_, err := app.InstallPluginFromRepo(repoURL, "")
		if err != nil {
			t.Fatalf("InstallPluginFromRepo failed: %v", err)
		}
	})

	t.Run("Step 3: Execute Analysis with Example Data", func(t *testing.T) {
		plugins := app.GetPluginsV2()
		var targetPlugin *models.PluginV2
		for _, p := range plugins {
			if p.Repository == repoURL {
				targetPlugin = p
				break
			}
		}

		if targetPlugin == nil {
			t.Fatal("Plugin not found in registry after installation")
		}

		cwd, _ := os.Getwd()
		pgFile := filepath.Join(cwd, "examples", "diann", "Reports.pg_matrix.tsv")
		prFile := filepath.Join(cwd, "examples", "diann", "Reports.pr_matrix.tsv")

		req := models.PluginExecutionRequestV2{
			PluginID: targetPlugin.ID,
			Parameters: map[string]interface{}{
				"pg_file":           pgFile,
				"pr_file":           prFile,
				"modification_type": "Static",
			},
		}

		jobID, err := app.ExecutePluginV2(req)
		if err != nil {
			t.Fatalf("ExecutePluginV2 failed: %v", err)
		}

		timeout := time.After(15 * time.Second)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				t.Fatal("Job execution timed out")
			case <-ticker.C:
				job, err := app.GetJob(jobID)
				if err != nil {
					t.Fatalf("Failed to get job: %v", err)
				}
				if job.Status == models.JobStatusCompleted {
					return
				}
				if job.Status == models.JobStatusFailed {
					t.Logf("Job failed (behavioral failure is OK, crash is NOT): %s", job.Error)
					return
				}
			}
		}
	})

	t.Run("Step 4: Cleanup", func(t *testing.T) {
		err := app.UninstallPluginFromRepo(repoURL, true, true, true)
		if err != nil {
			t.Errorf("UninstallPluginFromRepo failed: %v", err)
		}
	})
}
