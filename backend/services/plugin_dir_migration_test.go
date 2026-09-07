package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestMigrateThirdPartyPlugins_FirstRunRecordsPath(t *testing.T) {
	db := createTestDB(t)
	newDir := t.TempDir()

	MigrateThirdPartyPlugins(db, newDir)

	got, err := db.GetSetting(pluginsDirSettingKey)
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if got != newDir {
		t.Errorf("stored plugins dir = %q, want %q", got, newDir)
	}
}

func TestMigrateThirdPartyPlugins_CopiesMissingFolderFromPreviousDir(t *testing.T) {
	db := createTestDB(t)
	oldDir := t.TempDir()
	newDir := t.TempDir()
	writeTestFile(t, filepath.Join(oldDir, "my-plugin", "plugin.yaml"), "id: my-plugin")
	db.SaveSetting(pluginsDirSettingKey, oldDir)

	MigrateThirdPartyPlugins(db, newDir)

	got, err := os.ReadFile(filepath.Join(newDir, "my-plugin", "plugin.yaml"))
	if err != nil {
		t.Fatalf("expected plugin.yaml carried over, got: %v", err)
	}
	if string(got) != "id: my-plugin" {
		t.Errorf("copied content = %q, want %q", got, "id: my-plugin")
	}
}

func TestMigrateThirdPartyPlugins_DoesNotOverwriteExistingFolderInNewDir(t *testing.T) {
	db := createTestDB(t)
	oldDir := t.TempDir()
	newDir := t.TempDir()
	writeTestFile(t, filepath.Join(oldDir, "shared-plugin", "plugin.yaml"), "old-version")
	writeTestFile(t, filepath.Join(newDir, "shared-plugin", "plugin.yaml"), "bundled-version")
	db.SaveSetting(pluginsDirSettingKey, oldDir)

	MigrateThirdPartyPlugins(db, newDir)

	got, err := os.ReadFile(filepath.Join(newDir, "shared-plugin", "plugin.yaml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "bundled-version" {
		t.Errorf("bundled plugin got overwritten, content = %q", got)
	}
}

func TestMigrateThirdPartyPlugins_SkipsDotPrefixedFolders(t *testing.T) {
	db := createTestDB(t)
	oldDir := t.TempDir()
	newDir := t.TempDir()
	writeTestFile(t, filepath.Join(oldDir, ".temp-123", "partial.txt"), "leftover")
	db.SaveSetting(pluginsDirSettingKey, oldDir)

	MigrateThirdPartyPlugins(db, newDir)

	if _, err := os.Stat(filepath.Join(newDir, ".temp-123")); !os.IsNotExist(err) {
		t.Errorf("expected .temp-123 to be skipped, stat err = %v", err)
	}
}

func TestMigrateThirdPartyPlugins_NoopWhenPathUnchanged(t *testing.T) {
	db := createTestDB(t)
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "my-plugin", "plugin.yaml"), "id: my-plugin")
	db.SaveSetting(pluginsDirSettingKey, dir)

	MigrateThirdPartyPlugins(db, dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected exactly the original plugin folder, got %d entries", len(entries))
	}
}

func TestReconcileOrphanedPluginData_ReattachesEnvVarsAndRemovesStaleRow(t *testing.T) {
	db := createTestDB(t)
	currentFolder := t.TempDir()

	staleFolder := filepath.Join(t.TempDir(), "gone")
	stale := models.PluginRegistry{PluginID: "my-plugin", Name: "My Plugin", FolderPath: staleFolder}
	if err := db.GetDB().Create(&stale).Error; err != nil {
		t.Fatalf("create stale registry: %v", err)
	}
	current := models.PluginRegistry{PluginID: "my-plugin", Name: "My Plugin", FolderPath: currentFolder}
	if err := db.GetDB().Create(&current).Error; err != nil {
		t.Fatalf("create current registry: %v", err)
	}
	if err := db.SaveCustomEnvVar(CustomEnvVar{PluginID: stale.ID, Key: "API_KEY", Value: "secret"}); err != nil {
		t.Fatalf("SaveCustomEnvVar: %v", err)
	}

	ReconcileOrphanedPluginData(db)

	vars, err := db.GetCustomEnvVars(current.ID)
	if err != nil {
		t.Fatalf("GetCustomEnvVars: %v", err)
	}
	if len(vars) != 1 || vars[0].Value != "secret" {
		t.Fatalf("expected env var reattached to current registry ID, got %+v", vars)
	}

	remaining, err := db.GetAllPluginRegistryEntries()
	if err != nil {
		t.Fatalf("GetAllPluginRegistryEntries: %v", err)
	}
	for _, e := range remaining {
		if e.ID == stale.ID {
			t.Errorf("expected stale registry row %d to be removed", stale.ID)
		}
	}
}

func TestReconcileOrphanedPluginData_NoopWhenNoOrphans(t *testing.T) {
	db := createTestDB(t)
	folder := t.TempDir()
	valid := models.PluginRegistry{PluginID: "my-plugin", Name: "My Plugin", FolderPath: folder}
	if err := db.GetDB().Create(&valid).Error; err != nil {
		t.Fatalf("create registry: %v", err)
	}
	db.SaveCustomEnvVar(CustomEnvVar{PluginID: valid.ID, Key: "API_KEY", Value: "secret"})

	ReconcileOrphanedPluginData(db)

	vars, err := db.GetCustomEnvVars(valid.ID)
	if err != nil {
		t.Fatalf("GetCustomEnvVars: %v", err)
	}
	if len(vars) != 1 || vars[0].Value != "secret" {
		t.Errorf("expected env var untouched, got %+v", vars)
	}
}

func TestReconcileOrphanedPluginData_LeavesOrphanWithNoCurrentMatch(t *testing.T) {
	db := createTestDB(t)
	staleFolder := filepath.Join(t.TempDir(), "gone")
	stale := models.PluginRegistry{PluginID: "removed-plugin", Name: "Removed Plugin", FolderPath: staleFolder}
	if err := db.GetDB().Create(&stale).Error; err != nil {
		t.Fatalf("create stale registry: %v", err)
	}
	db.SaveCustomEnvVar(CustomEnvVar{PluginID: stale.ID, Key: "API_KEY", Value: "secret"})

	ReconcileOrphanedPluginData(db)

	vars, err := db.GetCustomEnvVars(stale.ID)
	if err != nil {
		t.Fatalf("GetCustomEnvVars: %v", err)
	}
	if len(vars) != 1 {
		t.Errorf("expected env var left in place with no current match to reattach to, got %+v", vars)
	}
}
