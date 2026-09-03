package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	pending, err := svc.DetectPendingEnvVarMigration(registry, 1)
	if err != nil {
		t.Fatalf("DetectPendingEnvVarMigration failed: %v", err)
	}
	if len(pending.Changes) != 0 {
		t.Errorf("expected no changes with no migrations dir, got %+v", pending.Changes)
	}
}

func TestPendingEnvVarChanges_AlreadyAppliedIsNoOp(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)
	registry.LastAppliedSchemaVersion = 2

	pending, err := svc.DetectPendingEnvVarMigration(registry, 2)
	if err != nil {
		t.Fatalf("DetectPendingEnvVarMigration failed: %v", err)
	}
	if len(pending.Changes) != 0 {
		t.Errorf("expected no changes when already at current schemaVersion, got %+v", pending.Changes)
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

	pending, err := svc.DetectPendingEnvVarMigration(registry, 1)
	if err != nil {
		t.Fatalf("DetectPendingEnvVarMigration failed: %v", err)
	}
	if len(pending.Changes) != 1 || pending.Changes[0].From != "API_KEY" || pending.Changes[0].To != "UNIPROT_API_KEY" || pending.Changes[0].Removed {
		t.Fatalf("unexpected changes: %+v", pending.Changes)
	}
	if pending.TotalOperations != 1 || pending.Large {
		t.Errorf("expected TotalOperations=1, Large=false, got %+v", pending)
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

	pending, err := svc.DetectPendingEnvVarMigration(registry, 2)
	if err != nil {
		t.Fatalf("DetectPendingEnvVarMigration failed: %v", err)
	}
	if len(pending.Changes) != 1 || pending.Changes[0].From != "A" || pending.Changes[0].To != "C" {
		t.Fatalf("expected chained rename A -> C, got %+v", pending.Changes)
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

	pending, err := svc.DetectPendingEnvVarMigration(registry, 1)
	if err != nil {
		t.Fatalf("DetectPendingEnvVarMigration failed: %v", err)
	}
	if len(pending.Changes) != 1 || !pending.Changes[0].Removed || pending.Changes[0].From != "LEGACY_KEY" {
		t.Fatalf("expected a removal change, got %+v", pending.Changes)
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

	pending, err := svc.DetectPendingEnvVarMigration(registry, 1)
	if err != nil {
		t.Fatalf("DetectPendingEnvVarMigration failed: %v", err)
	}
	if len(pending.Changes) != 0 {
		t.Errorf("expected no changes for an unaffected key, got %+v", pending.Changes)
	}
}

// writeLargeMigration writes a single migration file with opCount unrelated renameEnvVar operations, none of which touch any saved key -- large purely by operation count.
func writeLargeMigration(t *testing.T, pluginFolder string, opCount int) {
	t.Helper()
	var b strings.Builder
	b.WriteString("schemaVersion: 1\noperations:\n")
	for i := 0; i < opCount; i++ {
		fmt.Fprintf(&b, "  - renameEnvVar: { from: UNUSED_%d, to: ALSO_UNUSED_%d }\n", i, i)
	}
	writeTestMigration(t, pluginFolder, "0001_large.yaml", b.String())
}

func TestPendingEnvVarChanges_LargeSetIsFlagged(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)
	writeLargeMigration(t, registry.FolderPath, LargeMigrationOperationThreshold+1)

	pending, err := svc.DetectPendingEnvVarMigration(registry, 1)
	if err != nil {
		t.Fatalf("DetectPendingEnvVarMigration failed: %v", err)
	}
	if !pending.Large {
		t.Errorf("expected Large=true for %d operations, got %+v", LargeMigrationOperationThreshold+1, pending)
	}
	if pending.TotalOperations != LargeMigrationOperationThreshold+1 {
		t.Errorf("expected TotalOperations=%d, got %d", LargeMigrationOperationThreshold+1, pending.TotalOperations)
	}
}

func TestPendingEnvVarChanges_AtThresholdIsNotLarge(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)
	writeLargeMigration(t, registry.FolderPath, LargeMigrationOperationThreshold)

	pending, err := svc.DetectPendingEnvVarMigration(registry, 1)
	if err != nil {
		t.Fatalf("DetectPendingEnvVarMigration failed: %v", err)
	}
	if pending.Large {
		t.Errorf("expected exactly-at-threshold (%d) to not be flagged large", LargeMigrationOperationThreshold)
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

	if err := svc.ApplyPendingEnvVarMigration(registry, 1, false); err != nil {
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

	if err := svc.ApplyPendingEnvVarMigration(registry, 1, false); err != nil {
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

	if err := svc.ApplyPendingEnvVarMigration(registry, 1, false); err != nil {
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

	if err := svc.ApplyPendingEnvVarMigration(registry, 1, false); err != nil {
		t.Fatalf("ApplyPendingEnvVarMigration failed: %v", err)
	}

	if registry.LastAppliedSchemaVersion != 1 {
		t.Errorf("expected ledger bumped to 1 even with no matching saved keys, got %d", registry.LastAppliedSchemaVersion)
	}
}

func TestApplyPendingEnvVarMigration_LargeSetRequiresConfirmation(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)
	writeLargeMigration(t, registry.FolderPath, LargeMigrationOperationThreshold+1)

	err := svc.ApplyPendingEnvVarMigration(registry, 1, false)
	if !errors.Is(err, ErrLargeMigrationRequiresConfirmation) {
		t.Fatalf("expected ErrLargeMigrationRequiresConfirmation, got %v", err)
	}
	if registry.LastAppliedSchemaVersion != 0 {
		t.Errorf("expected ledger left untouched when confirmation is required, got %d", registry.LastAppliedSchemaVersion)
	}
}

func TestApplyPendingEnvVarMigration_LargeSetSucceedsWhenConfirmed(t *testing.T) {
	db := createTestDB(t)
	svc, registry := setupMigrationTestPlugin(t, db)
	writeLargeMigration(t, registry.FolderPath, LargeMigrationOperationThreshold+1)

	if err := svc.ApplyPendingEnvVarMigration(registry, 1, true); err != nil {
		t.Fatalf("ApplyPendingEnvVarMigration with confirmedLarge=true failed: %v", err)
	}
	if registry.LastAppliedSchemaVersion != 1 {
		t.Errorf("expected ledger updated once confirmed, got %d", registry.LastAppliedSchemaVersion)
	}
}
