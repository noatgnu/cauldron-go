package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPortableFlag_StripsFlagAndReportsPresence(t *testing.T) {
	filtered, portable := extractPortableFlag([]string{"recipe", "--portable", "list"})
	if !portable {
		t.Error("expected --portable to be detected")
	}
	if len(filtered) != 2 || filtered[0] != "recipe" || filtered[1] != "list" {
		t.Errorf("expected --portable to be stripped and the rest preserved in order, got %v", filtered)
	}
}

func TestExtractPortableFlag_AbsentWhenNotPassed(t *testing.T) {
	filtered, portable := extractPortableFlag([]string{"recipe", "list"})
	if portable {
		t.Error("expected --portable to be false when not passed")
	}
	if len(filtered) != 2 || filtered[0] != "recipe" || filtered[1] != "list" {
		t.Errorf("expected args unchanged, got %v", filtered)
	}
}

func TestResolveAppDataRoot_PortableUsesExecutableDir(t *testing.T) {
	t.Cleanup(func() { portableMode = false })
	portableMode = true

	dataRoot, err := resolveAppDataRoot()
	if err != nil {
		t.Fatalf("resolveAppDataRoot failed: %v", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable failed: %v", err)
	}
	expected := filepath.Join(filepath.Dir(exePath), "data")
	if dataRoot != expected {
		t.Errorf("expected portable data root %q, got %q", expected, dataRoot)
	}
}

func TestResolveAppDataRoot_NonPortableUsesUserConfigDir(t *testing.T) {
	t.Cleanup(func() { portableMode = false })
	portableMode = false

	dataRoot, err := resolveAppDataRoot()
	if err != nil {
		t.Fatalf("resolveAppDataRoot failed: %v", err)
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("os.UserConfigDir failed: %v", err)
	}
	expected := filepath.Join(userConfigDir, "cauldron")
	if dataRoot != expected {
		t.Errorf("expected non-portable data root %q, got %q", expected, dataRoot)
	}
}
