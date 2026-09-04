package services

import "testing"

func TestNewSettingsService_DefaultsAutoCheckForUpdatesToTrueOnFirstRun(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	svc := newSettingsServiceInternal(db)

	if !svc.GetConfig().AutoCheckForUpdates {
		t.Error("expected AutoCheckForUpdates to default to true on first run")
	}
}

func TestSettingsService_ReloadDoesNotOverwriteExplicitFalse(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	svc := newSettingsServiceInternal(db)
	if err := svc.Set("autoCheckForUpdates", false); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	reloaded := newSettingsServiceInternal(db)

	if reloaded.GetConfig().AutoCheckForUpdates {
		t.Error("expected a reloaded service to preserve an explicitly saved false, not reset it to the default true")
	}
}

func TestSettingsService_GetSetAutoCheckForUpdates(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	svc := newSettingsServiceInternal(db)

	if err := svc.Set("autoCheckForUpdates", false); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if got := svc.Get("autoCheckForUpdates"); got != false {
		t.Errorf("expected Get to return false, got %v", got)
	}

	if err := svc.Set("autoCheckForUpdates", true); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if got := svc.Get("autoCheckForUpdates"); got != true {
		t.Errorf("expected Get to return true, got %v", got)
	}
}
