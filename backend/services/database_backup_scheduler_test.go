package services

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestDatabaseService_BackupTo_ProducesRestorableSnapshot(t *testing.T) {
	dbDir := t.TempDir()
	db, err := newDatabaseServiceFromPath(dbDir)
	if err != nil {
		t.Fatalf("newDatabaseServiceFromPath failed: %v", err)
	}
	defer db.Close()

	if err := db.SaveSetting("backup-check", "present"); err != nil {
		t.Fatalf("SaveSetting failed: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "snapshot.db")
	if err := db.BackupTo(backupPath); err != nil {
		t.Fatalf("BackupTo failed: %v", err)
	}

	sqlDB, err := sql.Open("sqlite", backupPath)
	if err != nil {
		t.Fatalf("failed to open the backup file: %v", err)
	}
	defer sqlDB.Close()

	var value string
	if err := sqlDB.QueryRow("SELECT value FROM settings WHERE key = ?", "backup-check").Scan(&value); err != nil {
		t.Fatalf("failed to read the setting back from the backup: %v", err)
	}
	if value != "present" {
		t.Errorf("expected the backup to contain the setting's value, got %q", value)
	}
}

func TestDatabaseBackupScheduler_BackupNow_WritesOneTimestampedFile(t *testing.T) {
	dbDir := t.TempDir()
	db, err := newDatabaseServiceFromPath(dbDir)
	if err != nil {
		t.Fatalf("newDatabaseServiceFromPath failed: %v", err)
	}
	defer db.Close()

	scheduler := NewDatabaseBackupScheduler(db, 24*time.Hour, 7)
	if err := scheduler.BackupNow(); err != nil {
		t.Fatalf("BackupNow failed: %v", err)
	}

	entries, err := os.ReadDir(scheduler.dir)
	if err != nil {
		t.Fatalf("failed to read backup dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one backup file, got %d", len(entries))
	}
	if !isDBBackupFile(entries[0].Name()) {
		t.Errorf("expected a cauldron-*.db backup file, got %q", entries[0].Name())
	}
}

func TestDatabaseBackupScheduler_Rotate_KeepsOnlyNewestN(t *testing.T) {
	dbDir := t.TempDir()
	db, err := newDatabaseServiceFromPath(dbDir)
	if err != nil {
		t.Fatalf("newDatabaseServiceFromPath failed: %v", err)
	}
	defer db.Close()

	scheduler := NewDatabaseBackupScheduler(db, 24*time.Hour, 3)
	if err := os.MkdirAll(scheduler.dir, 0755); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	names := []string{
		"cauldron-20260101-000000.db",
		"cauldron-20260102-000000.db",
		"cauldron-20260103-000000.db",
		"cauldron-20260104-000000.db",
		"cauldron-20260105-000000.db",
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(scheduler.dir, name), []byte("x"), 0644); err != nil {
			t.Fatalf("failed to write fixture backup %s: %v", name, err)
		}
	}

	if err := scheduler.rotate(); err != nil {
		t.Fatalf("rotate failed: %v", err)
	}

	entries, err := os.ReadDir(scheduler.dir)
	if err != nil {
		t.Fatalf("failed to read backup dir: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 backups to survive rotation, got %d", len(entries))
	}

	remaining := map[string]bool{}
	for _, e := range entries {
		remaining[e.Name()] = true
	}
	for _, expected := range names[2:] {
		if !remaining[expected] {
			t.Errorf("expected newest backup %s to survive rotation, got %v", expected, remaining)
		}
	}
	for _, removed := range names[:2] {
		if remaining[removed] {
			t.Errorf("expected oldest backup %s to be removed by rotation", removed)
		}
	}
}

func TestDatabaseBackupScheduler_DueNow(t *testing.T) {
	dbDir := t.TempDir()
	db, err := newDatabaseServiceFromPath(dbDir)
	if err != nil {
		t.Fatalf("newDatabaseServiceFromPath failed: %v", err)
	}
	defer db.Close()

	scheduler := NewDatabaseBackupScheduler(db, 24*time.Hour, 7)

	if !scheduler.dueNow() {
		t.Error("expected a scheduler with no existing backups to be due immediately")
	}

	if err := os.MkdirAll(scheduler.dir, 0755); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}
	fresh := filepath.Join(scheduler.dir, "cauldron-20260101-000000.db")
	if err := os.WriteFile(fresh, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to write fixture backup: %v", err)
	}
	if err := os.Chtimes(fresh, time.Now(), time.Now()); err != nil {
		t.Fatalf("failed to set mtime: %v", err)
	}
	if scheduler.dueNow() {
		t.Error("expected a scheduler with a fresh backup to not be due yet")
	}

	stale := filepath.Join(scheduler.dir, "cauldron-20250101-000000.db")
	if err := os.WriteFile(stale, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to write fixture backup: %v", err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(fresh, old, old); err != nil {
		t.Fatalf("failed to set mtime: %v", err)
	}
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatalf("failed to set mtime: %v", err)
	}
	if !scheduler.dueNow() {
		t.Error("expected a scheduler whose newest backup is older than the interval to be due")
	}
}

func TestDatabaseBackupScheduler_StartBacksUpImmediatelyWhenNoneExist(t *testing.T) {
	dbDir := t.TempDir()
	db, err := newDatabaseServiceFromPath(dbDir)
	if err != nil {
		t.Fatalf("newDatabaseServiceFromPath failed: %v", err)
	}
	defer db.Close()

	scheduler := NewDatabaseBackupScheduler(db, 24*time.Hour, 7)
	scheduler.Start()
	defer scheduler.Shutdown()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(scheduler.dir)
		if err == nil && len(entries) == 1 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("expected Start to have written a backup within 5 seconds")
}
