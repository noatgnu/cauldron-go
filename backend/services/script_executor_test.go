package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTestScriptExecutor(t *testing.T) (*ScriptExecutor, *DatabaseService) {
	db := createTestDB(t)
	ctx := context.WithValue(context.Background(), "wails-test", true)
	settings := NewSettingsService(ctx, db)
	return NewScriptExecutor(settings, db, "test"), db
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

func TestPrepareDockerEnv_ExcludesHostEnvironment(t *testing.T) {
	executor, _ := createTestScriptExecutor(t)

	const hostOnlyVar = "CAULDRON_TEST_HOST_ONLY_VAR"
	t.Setenv(hostOnlyVar, "should-not-leak-into-container")

	env := executor.prepareDockerEnv(ScriptConfig{PluginID: 0})

	for _, e := range env {
		if strings.HasPrefix(e, hostOnlyVar+"=") {
			t.Errorf("expected prepareDockerEnv to exclude host-only env var %q, but found it: %q", hostOnlyVar, e)
		}
	}
}

func TestPrepareDockerEnv_IncludesConfiguredCustomVars(t *testing.T) {
	executor, db := createTestScriptExecutor(t)

	if err := db.SaveCustomEnvVar(CustomEnvVar{PluginID: 0, Key: "GLOBAL_VAR", Value: "global-value"}); err != nil {
		t.Fatalf("failed to save global custom env var: %v", err)
	}
	if err := db.SaveCustomEnvVar(CustomEnvVar{PluginID: 42, Key: "PLUGIN_VAR", Value: "plugin-value"}); err != nil {
		t.Fatalf("failed to save plugin-specific custom env var: %v", err)
	}

	env := executor.prepareDockerEnv(ScriptConfig{PluginID: 42})

	if !containsEnvVar(env, "GLOBAL_VAR=global-value") {
		t.Errorf("expected prepareDockerEnv to include the configured global var, got: %v", env)
	}
	if !containsEnvVar(env, "PLUGIN_VAR=plugin-value") {
		t.Errorf("expected prepareDockerEnv to include the configured plugin-specific var, got: %v", env)
	}
}

func TestPrepareEnv_IncludesCauldronPathVars(t *testing.T) {
	executor, _ := createTestScriptExecutor(t)

	pluginDir := filepath.Join(t.TempDir(), "plugins", "my-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	config := ScriptConfig{
		PluginID:      7,
		Type:          "my-plugin",
		PluginVersion: "2.3.4",
		FolderPath:    pluginDir,
	}

	env := executor.prepareEnv(config)

	if !containsEnvVar(env, "CAULDRON_VERSION=test") {
		t.Errorf("expected CAULDRON_VERSION=test, got: %v", env)
	}
	if !containsEnvVar(env, "CAULDRON_PLUGINS_DIR="+filepath.Dir(pluginDir)) {
		t.Errorf("expected CAULDRON_PLUGINS_DIR=%s, got: %v", filepath.Dir(pluginDir), env)
	}
	if !containsEnvVar(env, "CAULDRON_PLUGIN_DIR="+pluginDir) {
		t.Errorf("expected CAULDRON_PLUGIN_DIR=%s, got: %v", pluginDir, env)
	}
	if !containsEnvVar(env, "CAULDRON_PLUGIN_ID=my-plugin") {
		t.Errorf("expected CAULDRON_PLUGIN_ID=my-plugin, got: %v", env)
	}
	if !containsEnvVar(env, "CAULDRON_PLUGIN_VERSION=2.3.4") {
		t.Errorf("expected CAULDRON_PLUGIN_VERSION=2.3.4, got: %v", env)
	}

	hasAppDataDir := false
	for _, e := range env {
		if strings.HasPrefix(e, "CAULDRON_APP_DATA_DIR=") {
			hasAppDataDir = true
			break
		}
	}
	if !hasAppDataDir {
		t.Errorf("expected a CAULDRON_APP_DATA_DIR entry, got: %v", env)
	}
}

func TestPrepareDockerEnv_OmitsHostPaths(t *testing.T) {
	executor, _ := createTestScriptExecutor(t)

	config := ScriptConfig{
		PluginID:      7,
		Type:          "my-plugin",
		PluginVersion: "2.3.4",
		FolderPath:    filepath.Join(t.TempDir(), "plugins", "my-plugin"),
	}

	env := executor.prepareDockerEnv(config)

	if !containsEnvVar(env, "CAULDRON_VERSION=test") {
		t.Errorf("expected CAULDRON_VERSION=test, got: %v", env)
	}
	if !containsEnvVar(env, "CAULDRON_PLUGIN_ID=my-plugin") {
		t.Errorf("expected CAULDRON_PLUGIN_ID=my-plugin, got: %v", env)
	}
	if !containsEnvVar(env, "CAULDRON_PLUGIN_VERSION=2.3.4") {
		t.Errorf("expected CAULDRON_PLUGIN_VERSION=2.3.4, got: %v", env)
	}

	for _, e := range env {
		if strings.HasPrefix(e, "CAULDRON_APP_DATA_DIR=") ||
			strings.HasPrefix(e, "CAULDRON_PLUGINS_DIR=") ||
			strings.HasPrefix(e, "CAULDRON_PLUGIN_DIR=") {
			t.Errorf("expected prepareDockerEnv to omit host path vars, but found: %q", e)
		}
	}
}

func containsEnvVar(env []string, want string) bool {
	for _, e := range env {
		if e == want {
			return true
		}
	}
	return false
}
