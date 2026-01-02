package services

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func createTestEnvironmentService(t *testing.T) (*EnvironmentService, *DatabaseService, *SettingsService) {
	db := createTestDB(t)
	ctx := context.WithValue(context.Background(), "wails-test", true)
	settings := NewSettingsService(ctx, db)
	progress := NewProgressNotifier(ctx)
	envService := NewEnvironmentService(ctx, db, settings, progress)
	return envService, db, settings
}

func TestGetRenvCachePath(t *testing.T) {
	envService, _, _ := createTestEnvironmentService(t)

	cachePath, err := envService.getRenvCachePath()
	if err != nil {
		t.Fatalf("Failed to get renv cache path: %v", err)
	}

	if cachePath == "" {
		t.Error("Expected non-empty cache path")
	}

	if !filepath.IsAbs(cachePath) {
		t.Errorf("Expected absolute path, got: %s", cachePath)
	}

	// Verify directory was created
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("Expected cache directory to be created at: %s", cachePath)
	}
}

func TestSettingsUseRenvCache(t *testing.T) {
	_, _, settings := createTestEnvironmentService(t)

	// Test default (should be false based on recent changes)
	if settings.GetConfig().UseRenvCache != false {
		t.Errorf("Expected default UseRenvCache to be false, got: %v", settings.GetConfig().UseRenvCache)
	}

	// Test setting to true
	err := settings.Set("useRenvCache", true)
	if err != nil {
		t.Fatalf("Failed to set useRenvCache: %v", err)
	}

	if settings.GetConfig().UseRenvCache != true {
		t.Errorf("Expected UseRenvCache to be true after setting")
	}

	// Verify persistence
	newSettings := NewSettingsService(context.Background(), settings.db)
	if newSettings.GetConfig().UseRenvCache != true {
		t.Error("UseRenvCache setting did not persist to database")
	}
}

func TestRenvEnvironmentModel(t *testing.T) {
	_, db, _ := createTestEnvironmentService(t)

	env := RenvEnvironment{
		Name:           "test-env",
		Path:           "/tmp/test-env",
		ProjectPath:    "/tmp/test-env",
		BaseRPath:      "/usr/bin/Rscript",
		UseGlobalCache: true,
		CreatedAt:      123456789,
	}

	err := db.SaveRenvEnvironment(env)
	if err != nil {
		t.Fatalf("Failed to save RenvEnvironment: %v", err)
	}

	retrieved, err := db.GetRenvEnvironments()
	if err != nil {
		t.Fatalf("Failed to get RenvEnvironments: %v", err)
	}

	found := false
	for _, e := range retrieved {
		if e.Name == "test-env" {
			found = true
			if e.UseGlobalCache != true {
				t.Errorf("Expected UseGlobalCache to be true, got: %v", e.UseGlobalCache)
			}
			break
		}
	}

	if !found {
		t.Error("RenvEnvironment was not found in database")
	}
}

func TestLoadRPackagesFromFile(t *testing.T) {
	envService, _, _ := createTestEnvironmentService(t)

	tmpFile := filepath.Join(t.TempDir(), "r-packages.txt")
	content := "ggplot2\n# Comment\nlimma\n\"dplyr\"\n"
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	packages, err := envService.LoadRPackagesFromFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load packages: %v", err)
	}

	expected := []string{"ggplot2", "limma", "dplyr"}
	if len(packages) != len(expected) {
		t.Errorf("Expected %d packages, got %d", len(expected), len(packages))
	}

	for i, pkg := range packages {
		if pkg != expected[i] {
			t.Errorf("Expected package %s, got %s", expected[i], pkg)
		}
	}
}

func TestPythonEnvironmentDetection(t *testing.T) {
	// This test primarily checks that the DB operations don't crash
	// actual detection depends on the host system
	envService, db, _ := createTestEnvironmentService(t)

	envs, err := envService.DetectPythonEnvironments()
	if err != nil {
		t.Fatalf("DetectPythonEnvironments failed: %v", err)
	}

	// Should at least be able to get what was saved from DB
	dbEnvs, err := db.GetPythonEnvironments()
	if err != nil {
		t.Fatalf("Failed to get environments from DB: %v", err)
	}

	if len(envs) != len(dbEnvs) {
		t.Errorf("Mismatch between returned and DB environments")
	}
}

func TestREnvironmentDetection(t *testing.T) {
	envService, db, _ := createTestEnvironmentService(t)

	envs, err := envService.DetectREnvironments()
	if err != nil {
		t.Fatalf("DetectREnvironments failed: %v", err)
	}

	dbEnvs, err := db.GetREnvironments()
	if err != nil {
		t.Fatalf("Failed to get R environments from DB: %v", err)
	}

	if len(envs) != len(dbEnvs) {
		t.Errorf("Mismatch between returned and DB R environments")
	}
}

func TestVenvCreationWithExternalPlugin(t *testing.T) {
	envService, db, settings := createTestEnvironmentService(t)
	pythonPath, err := settings.DetectPythonPath()
	if err != nil {
		t.Skip("Python not found, skipping venv creation test")
	}
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	err = os.MkdirAll(pluginsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create plugins dir: %v", err)
	}
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)
	pluginURL := "https://github.com/noatgnu/export_asap_plugin"
	pluginID := "export_asap_plugin"
	pluginPath := filepath.Join(pluginsDir, pluginID)
	cmd := exec.Command("git", "clone", pluginURL, pluginPath)
	err = cmd.Run()
	if err != nil {
		t.Skipf("Failed to clone plugin for test: %v", err)
	}
	reg := models.PluginRegistry{
		PluginID:   pluginID,
		Name:       "Export ASAP",
		FolderPath: pluginPath,
	}
	db.GetDB().Create(&reg)
	venvPath := filepath.Join(tmpDir, "test_venv")
	err = envService.CreatePythonVirtualEnv(pythonPath, venvPath, strconv.FormatUint(uint64(reg.ID), 10), pluginPath)
	if err != nil {
		t.Fatalf("CreatePythonVirtualEnv failed: %v", err)
	}
	pythonExe := filepath.Join(venvPath, "bin", "python")
	if runtime.GOOS == "windows" {
		pythonExe = filepath.Join(venvPath, "Scripts", "python.exe")
	}
	if _, err := os.Stat(pythonExe); os.IsNotExist(err) {
		t.Errorf("Python executable not found in venv: %s", pythonExe)
	}
}
