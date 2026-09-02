package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func writeTestMigration(t *testing.T, pluginFolder, name, content string) {
	t.Helper()
	migrationsDir := filepath.Join(pluginFolder, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatalf("failed to create migrations dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write migration %s: %v", name, err)
	}
}

func setupMigrationTestPlugin(t *testing.T, db *DatabaseService) (*PluginMigrationService, *models.PluginRegistry) {
	t.Helper()
	pluginFolder := t.TempDir()

	registry := models.PluginRegistry{
		PluginID:   "sample-plugin",
		Name:       "Sample Plugin",
		FolderPath: pluginFolder,
	}
	if err := db.GetDB().Create(&registry).Error; err != nil {
		t.Fatalf("failed to create registry row: %v", err)
	}

	return NewPluginMigrationService(db), &registry
}

func TestPendingEnvVarChanges_NoMigrationsDir(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)

	changes, err := svc.DetectPendingEnvVarMigration(registry, 1)
	if err != nil {
		t.Fatalf("DetectPendingEnvVarMigration failed: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected no changes with no migrations dir, got %+v", changes)
	}
}

func TestPendingEnvVarChanges_AlreadyAppliedIsNoOp(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)
	registry.LastAppliedSchemaVersion = 2

	changes, err := svc.DetectPendingEnvVarMigration(registry, 2)
	if err != nil {
		t.Fatalf("DetectPendingEnvVarMigration failed: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected no changes when already at current schemaVersion, got %+v", changes)
	}
}

func TestPendingEnvVarChanges_SimpleRename(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)

	writeTestMigration(t, registry.FolderPath, "0001_rename.yaml",
		"schemaVersion: 1\noperations:\n  - renameEnvVar: { from: API_KEY, to: UNIPROT_API_KEY }\n")

	if err := db.SaveCustomEnvVar(CustomEnvVar{PluginID: registry.ID, Key: "API_KEY", Value: "secret"}); err != nil {
		t.Fatalf("failed to seed CustomEnvVar: %v", err)
	}

	changes, err := svc.DetectPendingEnvVarMigration(registry, 1)
	if err != nil {
		t.Fatalf("DetectPendingEnvVarMigration failed: %v", err)
	}
	if len(changes) != 1 || changes[0].From != "API_KEY" || changes[0].To != "UNIPROT_API_KEY" || changes[0].Removed {
		t.Fatalf("unexpected changes: %+v", changes)
	}
}

func TestPendingEnvVarChanges_ChainedRenameAcrossMigrations(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)

	writeTestMigration(t, registry.FolderPath, "0001_first.yaml",
		"schemaVersion: 1\noperations:\n  - renameEnvVar: { from: A, to: B }\n")
	writeTestMigration(t, registry.FolderPath, "0002_second.yaml",
		"schemaVersion: 2\noperations:\n  - renameEnvVar: { from: B, to: C }\n")

	if err := db.SaveCustomEnvVar(CustomEnvVar{PluginID: registry.ID, Key: "A", Value: "v"}); err != nil {
		t.Fatalf("failed to seed CustomEnvVar: %v", err)
	}

	changes, err := svc.DetectPendingEnvVarMigration(registry, 2)
	if err != nil {
		t.Fatalf("DetectPendingEnvVarMigration failed: %v", err)
	}
	if len(changes) != 1 || changes[0].From != "A" || changes[0].To != "C" {
		t.Fatalf("expected chained rename A -> C, got %+v", changes)
	}
}

func TestPendingEnvVarChanges_Removal(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)

	writeTestMigration(t, registry.FolderPath, "0001_remove.yaml",
		"schemaVersion: 1\noperations:\n  - removeEnvVar: { name: LEGACY_KEY }\n")

	if err := db.SaveCustomEnvVar(CustomEnvVar{PluginID: registry.ID, Key: "LEGACY_KEY", Value: "v"}); err != nil {
		t.Fatalf("failed to seed CustomEnvVar: %v", err)
	}

	changes, err := svc.DetectPendingEnvVarMigration(registry, 1)
	if err != nil {
		t.Fatalf("DetectPendingEnvVarMigration failed: %v", err)
	}
	if len(changes) != 1 || !changes[0].Removed || changes[0].From != "LEGACY_KEY" {
		t.Fatalf("expected a removal change, got %+v", changes)
	}
}

func TestPendingEnvVarChanges_IgnoresUnaffectedKeys(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)

	writeTestMigration(t, registry.FolderPath, "0001_rename.yaml",
		"schemaVersion: 1\noperations:\n  - renameEnvVar: { from: OTHER_KEY, to: RENAMED_KEY }\n")

	if err := db.SaveCustomEnvVar(CustomEnvVar{PluginID: registry.ID, Key: "UNRELATED_KEY", Value: "v"}); err != nil {
		t.Fatalf("failed to seed CustomEnvVar: %v", err)
	}

	changes, err := svc.DetectPendingEnvVarMigration(registry, 1)
	if err != nil {
		t.Fatalf("DetectPendingEnvVarMigration failed: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected no changes for an unaffected key, got %+v", changes)
	}
}

func TestApplyPendingEnvVarMigration_RenamesAndUpdatesLedger(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)

	writeTestMigration(t, registry.FolderPath, "0001_rename.yaml",
		"schemaVersion: 1\noperations:\n  - renameEnvVar: { from: API_KEY, to: UNIPROT_API_KEY }\n")

	if err := db.SaveCustomEnvVar(CustomEnvVar{PluginID: registry.ID, Key: "API_KEY", Value: "secret"}); err != nil {
		t.Fatalf("failed to seed CustomEnvVar: %v", err)
	}

	if err := svc.ApplyPendingEnvVarMigration(registry, 1); err != nil {
		t.Fatalf("ApplyPendingEnvVarMigration failed: %v", err)
	}

	envVars, err := db.GetCustomEnvVars(registry.ID)
	if err != nil {
		t.Fatalf("GetCustomEnvVars failed: %v", err)
	}
	if len(envVars) != 1 || envVars[0].Key != "UNIPROT_API_KEY" || envVars[0].Value != "secret" {
		t.Fatalf("expected renamed env var with value preserved, got %+v", envVars)
	}

	if registry.LastAppliedSchemaVersion != 1 {
		t.Errorf("expected in-memory registry.LastAppliedSchemaVersion updated to 1, got %d", registry.LastAppliedSchemaVersion)
	}
	reloaded, err := db.GetPluginRegistryByID(registry.ID)
	if err != nil {
		t.Fatalf("GetPluginRegistryByID failed: %v", err)
	}
	if reloaded.LastAppliedSchemaVersion != 1 {
		t.Errorf("expected persisted LastAppliedSchemaVersion 1, got %d", reloaded.LastAppliedSchemaVersion)
	}
}

func TestApplyPendingEnvVarMigration_Removal(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)

	writeTestMigration(t, registry.FolderPath, "0001_remove.yaml",
		"schemaVersion: 1\noperations:\n  - removeEnvVar: { name: LEGACY_KEY }\n")

	if err := db.SaveCustomEnvVar(CustomEnvVar{PluginID: registry.ID, Key: "LEGACY_KEY", Value: "v"}); err != nil {
		t.Fatalf("failed to seed CustomEnvVar: %v", err)
	}

	if err := svc.ApplyPendingEnvVarMigration(registry, 1); err != nil {
		t.Fatalf("ApplyPendingEnvVarMigration failed: %v", err)
	}

	envVars, err := db.GetCustomEnvVars(registry.ID)
	if err != nil {
		t.Fatalf("GetCustomEnvVars failed: %v", err)
	}
	if len(envVars) != 0 {
		t.Errorf("expected removed env var to be gone, got %+v", envVars)
	}
}

func TestApplyPendingEnvVarMigration_RenameConflictDropsSource(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)

	writeTestMigration(t, registry.FolderPath, "0001_rename.yaml",
		"schemaVersion: 1\noperations:\n  - renameEnvVar: { from: OLD_KEY, to: NEW_KEY }\n")

	if err := db.SaveCustomEnvVar(CustomEnvVar{PluginID: registry.ID, Key: "OLD_KEY", Value: "old-value"}); err != nil {
		t.Fatalf("failed to seed OLD_KEY: %v", err)
	}
	if err := db.SaveCustomEnvVar(CustomEnvVar{PluginID: registry.ID, Key: "NEW_KEY", Value: "already-set"}); err != nil {
		t.Fatalf("failed to seed NEW_KEY: %v", err)
	}

	if err := svc.ApplyPendingEnvVarMigration(registry, 1); err != nil {
		t.Fatalf("ApplyPendingEnvVarMigration failed: %v", err)
	}

	envVars, err := db.GetCustomEnvVars(registry.ID)
	if err != nil {
		t.Fatalf("GetCustomEnvVars failed: %v", err)
	}
	if len(envVars) != 1 || envVars[0].Key != "NEW_KEY" || envVars[0].Value != "already-set" {
		t.Fatalf("expected only the pre-existing NEW_KEY to remain untouched, got %+v", envVars)
	}
}

func TestApplyPendingEnvVarMigration_NoMatchingKeysStillBumpsLedger(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)

	writeTestMigration(t, registry.FolderPath, "0001_rename.yaml",
		"schemaVersion: 1\noperations:\n  - renameEnvVar: { from: NEVER_SET, to: ALSO_NEVER_SET }\n")

	if err := svc.ApplyPendingEnvVarMigration(registry, 1); err != nil {
		t.Fatalf("ApplyPendingEnvVarMigration failed: %v", err)
	}

	if registry.LastAppliedSchemaVersion != 1 {
		t.Errorf("expected ledger bumped to 1 even with no matching saved keys, got %d", registry.LastAppliedSchemaVersion)
	}
}
