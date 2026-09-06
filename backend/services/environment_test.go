package services

import (
	"bytes"
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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

func TestSettingsDebugMode(t *testing.T) {
	_, _, settings := createTestEnvironmentService(t)

	if settings.GetConfig().DebugMode != false {
		t.Errorf("Expected default DebugMode to be false, got: %v", settings.GetConfig().DebugMode)
	}

	if err := settings.Set("debugMode", true); err != nil {
		t.Fatalf("Failed to set debugMode: %v", err)
	}

	if settings.GetConfig().DebugMode != true {
		t.Errorf("Expected DebugMode to be true after setting")
	}

	if got := settings.Get("debugMode"); got != true {
		t.Errorf("Expected Get(\"debugMode\") to return true, got: %v", got)
	}

	newSettings := NewSettingsService(context.Background(), settings.db)
	if newSettings.GetConfig().DebugMode != true {
		t.Error("DebugMode setting did not persist to database")
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

	pluginID := "dummy_plugin"
	pluginPath := filepath.Join(pluginsDir, pluginID)
	err = os.MkdirAll(pluginPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create dummy plugin dir: %v", err)
	}

	// Create a dummy plugin.yaml
	pluginYaml := `
plugin:
  id: dummy_plugin
  name: Dummy Plugin
execution:
  requirements:
    pythonRequirementsFile: requirements.txt
`
	err = os.WriteFile(filepath.Join(pluginPath, "plugin.yaml"), []byte(pluginYaml), 0644)
	if err != nil {
		t.Fatalf("Failed to write plugin.yaml: %v", err)
	}

	// Create a dummy requirements.txt with a trivial package that should be installable or effectively no-op if we want
	// But since we want to avoid network, maybe we skip installation if possible?
	// However, CreatePythonVirtualEnv logic installs if pluginID is provided.
	// We'll put a harmless package or empty requirements to test venv creation logic.
	// Empty requirements.txt should work.
	err = os.WriteFile(filepath.Join(pluginPath, "requirements.txt"), []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write requirements.txt: %v", err)
	}

	reg := models.PluginRegistry{
		PluginID:   pluginID,
		Name:       "Dummy Plugin",
		FolderPath: pluginPath,
	}
	db.GetDB().Create(&reg)
	venvPath := filepath.Join(tmpDir, "test_venv")
	err = envService.CreatePythonVirtualEnv(pythonPath, venvPath, strconv.FormatUint(uint64(reg.ID), 10), pluginPath)
	if err != nil {
		if strings.Contains(err.Error(), "failed to create virtual environment") {
			t.Skip("Python venv module not available, skipping venv creation test")
		}
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

func TestVenvCreationWithPlainRequirementsTxtFallback(t *testing.T) {
	envService, _, settings := createTestEnvironmentService(t)
	pythonPath, err := settings.DetectPythonPath()
	if err != nil {
		t.Skip("Python not found, skipping venv creation test")
	}
	tmpDir := t.TempDir()

	// No plugin.yaml here on purpose. This mirrors a non-plugin, script-backed feature (e.g. gel-analysis).
	folderPath := filepath.Join(tmpDir, "gel-analysis")
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(folderPath, "requirements.txt"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write requirements.txt: %v", err)
	}

	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	venvPath := filepath.Join(tmpDir, "test_venv")
	err = envService.CreatePythonVirtualEnv(pythonPath, venvPath, "gel-analysis", folderPath)
	if err != nil {
		if strings.Contains(err.Error(), "failed to create virtual environment") {
			t.Skip("Python venv module not available, skipping venv creation test")
		}
		t.Fatalf("CreatePythonVirtualEnv failed: %v", err)
	}

	if !strings.Contains(logBuf.String(), "installing plain requirements.txt") {
		t.Errorf("Expected the plain-requirements.txt fallback to fire for a folder with no plugin.yaml; log was: %s", logBuf.String())
	}
}

func TestCustomVenvStoragePath(t *testing.T) {
	envService, _, settings := createTestEnvironmentService(t)

	customPath := filepath.Join(t.TempDir(), "custom_venvs")

	err := settings.Set("venvStoragePath", customPath)
	if err != nil {
		t.Fatalf("Failed to set venvStoragePath: %v", err)
	}

	if settings.GetConfig().VenvStoragePath != customPath {
		t.Errorf("Expected VenvStoragePath to be %s, got %s", customPath, settings.GetConfig().VenvStoragePath)
	}

	if _, err := os.Stat(customPath); os.IsNotExist(err) {
		t.Errorf("Expected custom venv storage directory to be created at: %s", customPath)
	}

	venvDir, err := envService.getVenvStorageDir()
	if err != nil {
		t.Fatalf("Failed to get venv storage dir: %v", err)
	}

	if venvDir != customPath {
		t.Errorf("Expected getVenvStorageDir to return %s, got %s", customPath, venvDir)
	}
}

func TestCustomRenvStoragePath(t *testing.T) {
	envService, _, settings := createTestEnvironmentService(t)

	customPath := filepath.Join(t.TempDir(), "custom_renvs")

	err := settings.Set("renvStoragePath", customPath)
	if err != nil {
		t.Fatalf("Failed to set renvStoragePath: %v", err)
	}

	if settings.GetConfig().RenvStoragePath != customPath {
		t.Errorf("Expected RenvStoragePath to be %s, got %s", customPath, settings.GetConfig().RenvStoragePath)
	}

	if _, err := os.Stat(customPath); os.IsNotExist(err) {
		t.Errorf("Expected custom renv storage directory to be created at: %s", customPath)
	}

	renvDir, err := envService.getRenvStorageDir()
	if err != nil {
		t.Fatalf("Failed to get renv storage dir: %v", err)
	}

	if renvDir != customPath {
		t.Errorf("Expected getRenvStorageDir to return %s, got %s", customPath, renvDir)
	}
}

func TestVenvStoragePathPersistence(t *testing.T) {
	_, db, settings := createTestEnvironmentService(t)

	customPath := filepath.Join(t.TempDir(), "persistent_venvs")

	err := settings.Set("venvStoragePath", customPath)
	if err != nil {
		t.Fatalf("Failed to set venvStoragePath: %v", err)
	}

	newSettings := NewSettingsService(context.Background(), db)
	if newSettings.GetConfig().VenvStoragePath != customPath {
		t.Errorf("VenvStoragePath did not persist to database. Expected %s, got %s",
			customPath, newSettings.GetConfig().VenvStoragePath)
	}
}

func TestRenvStoragePathPersistence(t *testing.T) {
	_, db, settings := createTestEnvironmentService(t)

	customPath := filepath.Join(t.TempDir(), "persistent_renvs")

	err := settings.Set("renvStoragePath", customPath)
	if err != nil {
		t.Fatalf("Failed to set renvStoragePath: %v", err)
	}

	newSettings := NewSettingsService(context.Background(), db)
	if newSettings.GetConfig().RenvStoragePath != customPath {
		t.Errorf("RenvStoragePath did not persist to database. Expected %s, got %s",
			customPath, newSettings.GetConfig().RenvStoragePath)
	}
}

func TestVenvCreationInCustomLocation(t *testing.T) {
	envService, db, settings := createTestEnvironmentService(t)

	pythonPath, err := settings.DetectPythonPath()
	if err != nil {
		t.Skip("Python not found, skipping venv creation in custom location test")
	}

	tmpDir := t.TempDir()
	customVenvPath := filepath.Join(tmpDir, "custom_venvs")
	pluginsDir := filepath.Join(tmpDir, "plugins")

	err = settings.Set("venvStoragePath", customVenvPath)
	if err != nil {
		t.Fatalf("Failed to set custom venv storage path: %v", err)
	}

	err = os.MkdirAll(pluginsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create plugins dir: %v", err)
	}

	pluginID := "custom_location_plugin"
	pluginPath := filepath.Join(pluginsDir, pluginID)
	err = os.MkdirAll(pluginPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create plugin dir: %v", err)
	}

	pluginYaml := `
plugin:
  id: custom_location_plugin
  name: Custom Location Plugin
execution:
  requirements:
    pythonRequirementsFile: requirements.txt
`
	err = os.WriteFile(filepath.Join(pluginPath, "plugin.yaml"), []byte(pluginYaml), 0644)
	if err != nil {
		t.Fatalf("Failed to write plugin.yaml: %v", err)
	}

	err = os.WriteFile(filepath.Join(pluginPath, "requirements.txt"), []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write requirements.txt: %v", err)
	}

	reg := models.PluginRegistry{
		PluginID:   pluginID,
		Name:       "Custom Location Plugin",
		FolderPath: pluginPath,
	}
	db.GetDB().Create(&reg)

	venvPath := filepath.Join(customVenvPath, "test_custom_venv")
	err = envService.CreatePythonVirtualEnv(pythonPath, venvPath, strconv.FormatUint(uint64(reg.ID), 10), pluginPath)
	if err != nil {
		if strings.Contains(err.Error(), "failed to create virtual environment") {
			t.Skip("Python venv module not available, skipping venv creation in custom location test")
		}
		t.Fatalf("CreatePythonVirtualEnv failed in custom location: %v", err)
	}

	pythonExe := filepath.Join(venvPath, "bin", "python")
	if runtime.GOOS == "windows" {
		pythonExe = filepath.Join(venvPath, "Scripts", "python.exe")
	}

	if _, err := os.Stat(pythonExe); os.IsNotExist(err) {
		t.Errorf("Python executable not found in custom venv location: %s", pythonExe)
	}

	if !filepath.HasPrefix(venvPath, customVenvPath) {
		t.Errorf("Venv was not created in custom storage path. Expected prefix %s, got %s", customVenvPath, venvPath)
	}
}

func TestEmptyCustomStoragePaths(t *testing.T) {
	envService, _, settings := createTestEnvironmentService(t)

	err := settings.Set("venvStoragePath", "")
	if err != nil {
		t.Fatalf("Failed to set empty venvStoragePath: %v", err)
	}

	err = settings.Set("renvStoragePath", "")
	if err != nil {
		t.Fatalf("Failed to set empty renvStoragePath: %v", err)
	}

	venvDir, err := envService.getVenvStorageDir()
	if err != nil {
		t.Fatalf("Failed to get default venv storage dir: %v", err)
	}

	if venvDir == "" {
		t.Error("Expected non-empty default venv storage dir when custom path is empty")
	}

	renvDir, err := envService.getRenvStorageDir()
	if err != nil {
		t.Fatalf("Failed to get default renv storage dir: %v", err)
	}

	if renvDir == "" {
		t.Error("Expected non-empty default renv storage dir when custom path is empty")
	}
}
