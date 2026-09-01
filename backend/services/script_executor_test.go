package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func createTestScriptExecutor(t *testing.T) (*ScriptExecutor, *DatabaseService) {
	db := createTestDB(t)
	ctx := context.WithValue(context.Background(), "wails-test", true)
	settings := NewSettingsService(ctx, db)
	return NewScriptExecutor(settings, db), db
}

func createFakeExecutable(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("failed to create fake executable %s: %v", path, err)
	}
	return path
}

func TestResolveRExecutable_NoBindingNoGlobal(t *testing.T) {
	executor, _ := createTestScriptExecutor(t)

	_, _, _, err := executor.resolveRExecutable("some-plugin")
	if err == nil {
		t.Fatal("expected an error when neither a binding nor a global R path is configured")
	}
}

func TestResolveRExecutable_FallsBackToGlobal(t *testing.T) {
	executor, _ := createTestScriptExecutor(t)
	globalR := createFakeExecutable(t, "Rscript")

	if err := executor.settingsService.Set("rPath", globalR); err != nil {
		t.Fatalf("failed to set global rPath: %v", err)
	}

	rPath, renvProjectPath, envInfo, err := executor.resolveRExecutable("some-plugin")
	if err != nil {
		t.Fatalf("resolveRExecutable error: %v", err)
	}
	if rPath != globalR {
		t.Errorf("expected global R path %q, got %q", globalR, rPath)
	}
	if renvProjectPath != "" {
		t.Errorf("expected no renv project path, got %q", renvProjectPath)
	}
	if envInfo != "R (Global): "+globalR {
		t.Errorf("unexpected envInfo: %q", envInfo)
	}
}

func TestResolveRExecutable_UsesBoundRenvsOwnInterpreter(t *testing.T) {
	executor, db := createTestScriptExecutor(t)
	globalR := createFakeExecutable(t, "global-Rscript")
	renvR := createFakeExecutable(t, "renv-Rscript")

	if err := executor.settingsService.Set("rPath", globalR); err != nil {
		t.Fatalf("failed to set global rPath: %v", err)
	}

	renvEnv := RenvEnvironment{
		Name:        "bound-env",
		Path:        t.TempDir(),
		ProjectPath: t.TempDir(),
		BaseRPath:   renvR,
		CreatedAt:   1,
	}
	if err := db.SaveRenvEnvironment(renvEnv); err != nil {
		t.Fatalf("failed to save renv environment: %v", err)
	}
	envs, err := db.GetRenvEnvironments()
	if err != nil || len(envs) != 1 {
		t.Fatalf("failed to read back renv environment: %v", err)
	}
	saved := envs[0]

	if err := db.SavePluginEnvironmentBinding(PluginEnvironmentBinding{
		PluginID:        "my-plugin",
		EnvironmentType: "r",
		EnvironmentID:   saved.ID,
		EnvironmentPath: saved.ProjectPath,
	}); err != nil {
		t.Fatalf("failed to save plugin environment binding: %v", err)
	}

	rPath, renvProjectPath, envInfo, err := executor.resolveRExecutable("my-plugin")
	if err != nil {
		t.Fatalf("resolveRExecutable error: %v", err)
	}
	if rPath != renvR {
		t.Errorf("expected renv's own R interpreter %q, got %q (should not silently use the global R)", renvR, rPath)
	}
	if renvProjectPath != saved.ProjectPath {
		t.Errorf("expected renv project path %q, got %q", saved.ProjectPath, renvProjectPath)
	}
	if envInfo != "R (Bound renv): "+saved.ProjectPath+" ["+renvR+"]" {
		t.Errorf("unexpected envInfo: %q", envInfo)
	}
}

func TestResolveRExecutable_FallsBackWhenBoundInterpreterMissing(t *testing.T) {
	executor, db := createTestScriptExecutor(t)
	globalR := createFakeExecutable(t, "global-Rscript")
	missingRenvR := filepath.Join(t.TempDir(), "no-longer-there-Rscript")

	if err := executor.settingsService.Set("rPath", globalR); err != nil {
		t.Fatalf("failed to set global rPath: %v", err)
	}

	renvEnv := RenvEnvironment{
		Name:        "stale-env",
		Path:        t.TempDir(),
		ProjectPath: t.TempDir(),
		BaseRPath:   missingRenvR,
		CreatedAt:   1,
	}
	if err := db.SaveRenvEnvironment(renvEnv); err != nil {
		t.Fatalf("failed to save renv environment: %v", err)
	}
	envs, err := db.GetRenvEnvironments()
	if err != nil || len(envs) != 1 {
		t.Fatalf("failed to read back renv environment: %v", err)
	}
	saved := envs[0]

	if err := db.SavePluginEnvironmentBinding(PluginEnvironmentBinding{
		PluginID:        "my-plugin",
		EnvironmentType: "r",
		EnvironmentID:   saved.ID,
		EnvironmentPath: saved.ProjectPath,
	}); err != nil {
		t.Fatalf("failed to save plugin environment binding: %v", err)
	}

	rPath, renvProjectPath, _, err := executor.resolveRExecutable("my-plugin")
	if err != nil {
		t.Fatalf("resolveRExecutable error: %v", err)
	}
	if rPath != globalR {
		t.Errorf("expected fallback to global R %q when the bound interpreter is missing, got %q", globalR, rPath)
	}
	if renvProjectPath != saved.ProjectPath {
		t.Errorf("expected the renv library to still be activated at %q, got %q", saved.ProjectPath, renvProjectPath)
	}
}
