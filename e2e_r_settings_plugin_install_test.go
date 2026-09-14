package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFakeRscript writes a stand-in Rscript that logs its args to logPath and exits 0.
func writeFakeRscript(t *testing.T, path, logPath string) {
	t.Helper()
	script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "--version" ]; then
  echo "Fake Rscript version 4.4.0"
  exit 0
fi
echo "$*" >> %q
exit 0
`, logPath)
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake Rscript: %v", err)
	}
}

// writeE2ERTestPlugin writes a minimal, loadable R plugin under pluginsDir.
func writeE2ERTestPlugin(t *testing.T, pluginsDir, pluginID string, packages []string) {
	t.Helper()
	dir := filepath.Join(pluginsDir, pluginID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	quoted := make([]string, len(packages))
	for i, p := range packages {
		quoted[i] = fmt.Sprintf("      - %q", p)
	}
	yaml := fmt.Sprintf(`---
plugin:
  id: %q
  name: "E2E R Settings Test Plugin"
  description: "test plugin for manual R settings + plugin install e2e coverage"
  version: "1.0.0"
  author: "test"
  category: "utilities"

runtime:
  environments: ["r"]
  entrypoint: "test.R"

inputs: []
outputs: []

execution:
  argsMapping: {}
  outputDir: "--output_folder"
  requirements:
    r: ">=4.0"
    packages:
%s
`, pluginID, strings.Join(quoted, "\n"))

	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write plugin.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test.R"), []byte("# no-op\n"), 0644); err != nil {
		t.Fatalf("failed to write entrypoint script: %v", err)
	}
}

// TestE2EManualRPathThenPluginInstall covers manual Settings > R > Browse selection feeding into plugin env install.
func TestE2EManualRPathThenPluginInstall(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	// renv projects are stored under XDG_DATA_HOME, separately from XDG_CONFIG_HOME above.
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	exePath, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	pluginsDir := filepath.Join(filepath.Dir(exePath), "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatalf("failed to create plugins dir: %v", err)
	}

	const pluginID = "e2e-r-settings-test-plugin"
	packages := []string{"TestPackageOne", "TestPackageTwo"}
	writeE2ERTestPlugin(t, pluginsDir, pluginID, packages)
	t.Cleanup(func() { os.RemoveAll(filepath.Join(pluginsDir, pluginID)) })

	scratch := t.TempDir()
	fakeRscript := filepath.Join(scratch, "fake-rscript")
	rLog := filepath.Join(scratch, "r-invocations.log")
	writeFakeRscript(t, fakeRscript, rLog)

	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("manual R path selection is picked up like Settings > R > Browse", func(t *testing.T) {
		if err := app.SetSetting("rPath", fakeRscript); err != nil {
			t.Fatalf("SetSetting(rPath) error: %v", err)
		}

		version, err := app.GetRVersion()
		if err != nil {
			t.Fatalf("GetRVersion error: %v", err)
		}
		if !strings.Contains(version, "Fake Rscript version") {
			t.Errorf("GetRVersion() = %q, want it to reflect the manually selected R path", version)
		}
	})

	t.Run("plugin install env uses the manually configured R path and the real plugin folder", func(t *testing.T) {
		plugins := app.GetPluginsV2()
		var found bool
		for _, p := range plugins {
			if p.Definition.Plugin.ID == pluginID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected plugin %q to be loaded from %s, got %d plugins", pluginID, pluginsDir, len(plugins))
		}

		// Mirrors plugin-install-progress.ts: empty packages relies on plugin.yaml auto-discovery.
		err := app.CreateRenvEnvironment("e2e-test-renv", []string{}, pluginID, false)
		if err != nil {
			t.Fatalf("CreateRenvEnvironment error: %v (expected it to fall back to the manually configured R path)", err)
		}

		logBytes, err := os.ReadFile(rLog)
		if err != nil {
			t.Fatalf("expected the fake Rscript to have been invoked, but its log is missing: %v", err)
		}
		log := string(logBytes)
		for _, pkg := range packages {
			if !strings.Contains(log, pkg) {
				t.Errorf("expected R invocations to reference package %q (from the plugin's own requirements), log:\n%s", pkg, log)
			}
		}

		envs, err := app.GetRenvEnvironments()
		if err != nil {
			t.Fatalf("GetRenvEnvironments error: %v", err)
		}
		var createdEnvFound bool
		for _, e := range envs {
			if e.Name == "e2e-test-renv" {
				createdEnvFound = true
				break
			}
		}
		if !createdEnvFound {
			t.Errorf("expected the newly created renv environment to be listed, got %d environments", len(envs))
		}
	})
}
