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

func TestSettingsService_RegistryURLsPersistAcrossReload(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	svc := newSettingsServiceInternal(db)
	if err := svc.Set("pluginRegistryUrl", "https://plugins.example.com"); err != nil {
		t.Fatalf("Set pluginRegistryUrl failed: %v", err)
	}
	if err := svc.Set("recipeRegistryUrl", "https://recipes.example.com"); err != nil {
		t.Fatalf("Set recipeRegistryUrl failed: %v", err)
	}

	if got := svc.Get("pluginRegistryUrl"); got != "https://plugins.example.com" {
		t.Errorf("expected Get to return the just-set pluginRegistryUrl, got %v", got)
	}
	if got := svc.Get("recipeRegistryUrl"); got != "https://recipes.example.com" {
		t.Errorf("expected Get to return the just-set recipeRegistryUrl, got %v", got)
	}

	reloaded := newSettingsServiceInternal(db)
	if got := reloaded.GetConfig().PluginRegistryURL; got != "https://plugins.example.com" {
		t.Errorf("expected pluginRegistryUrl to survive a reload, got %q", got)
	}
	if got := reloaded.GetConfig().RecipeRegistryURL; got != "https://recipes.example.com" {
		t.Errorf("expected recipeRegistryUrl to survive a reload, got %q", got)
	}
}
