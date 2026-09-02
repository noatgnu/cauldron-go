package pluginmigrations

import (
	"os"
	"path/filepath"
	"testing"
)

func writeMigrationFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}

func TestLoadMigrations_ValidSequence(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "0001_first.yaml", "schemaVersion: 1\ndescription: first\noperations:\n  - renameEnvVar: { from: A, to: B }\n")
	writeMigrationFile(t, dir, "0002_second.yaml", "schemaVersion: 2\noperations:\n  - removeEnvVar: { name: C }\n")

	migrations, err := LoadMigrations(dir)
	if err != nil {
		t.Fatalf("LoadMigrations failed: %v", err)
	}
	if len(migrations) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(migrations))
	}
	if migrations[0].SchemaVersion != 1 || migrations[0].Description != "first" {
		t.Errorf("unexpected first migration: %+v", migrations[0])
	}
	if migrations[0].Operations[0].RenameEnvVar == nil || migrations[0].Operations[0].RenameEnvVar.From != "A" {
		t.Errorf("expected renameEnvVar operation parsed correctly, got %+v", migrations[0].Operations[0])
	}
	if migrations[1].Operations[0].RemoveEnvVar == nil || migrations[1].Operations[0].RemoveEnvVar.Name != "C" {
		t.Errorf("expected removeEnvVar operation parsed correctly, got %+v", migrations[1].Operations[0])
	}
}

func TestLoadMigrations_GapFails(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "0002_skipped_one.yaml", "schemaVersion: 2\noperations: []\n")

	if _, err := LoadMigrations(dir); err == nil {
		t.Error("expected error for a gapped schemaVersion sequence, got nil")
	}
}

func TestLoadMigrations_MissingDirReturnsEmpty(t *testing.T) {
	migrations, err := LoadMigrations(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("expected no error for a missing migrations dir, got %v", err)
	}
	if len(migrations) != 0 {
		t.Error("expected zero migrations for a missing directory")
	}
}

func TestLoadMigrations_IgnoresNonMatchingFileNames(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "README.md", "not a migration")
	writeMigrationFile(t, dir, "0001_valid.yaml", "schemaVersion: 1\noperations: []\n")

	migrations, err := LoadMigrations(dir)
	if err != nil {
		t.Fatalf("LoadMigrations failed: %v", err)
	}
	if len(migrations) != 1 {
		t.Fatalf("expected only the matching migration file to load, got %d", len(migrations))
	}
}

func TestLoadMigrations_SetsPathField(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "0001_first.yaml", "schemaVersion: 1\noperations: []\n")

	migrations, err := LoadMigrations(dir)
	if err != nil {
		t.Fatalf("LoadMigrations failed: %v", err)
	}
	want := filepath.Join(dir, "0001_first.yaml")
	if migrations[0].Path != want {
		t.Errorf("expected Path %q, got %q", want, migrations[0].Path)
	}
}
