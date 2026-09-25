package services

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
)

const dockerTestPluginYAML = `plugin:
  id: "docker-hardening-test-plugin"
  name: "Docker Hardening Test Plugin"
  description: "Throwaway plugin used only to exercise ExecuteDockerScript"
  version: "1.0.0"
  category: "utilities"

runtime:
  environments: ["docker"]
  entrypoint: "sh"
  docker:
    image: "alpine:latest"
`

const dockerTestPluginYAMLBridge = `plugin:
  id: "docker-hardening-test-plugin-bridge"
  name: "Docker Hardening Test Plugin (bridge)"
  description: "Throwaway plugin used only to exercise ExecuteDockerScript's network opt-out"
  version: "1.0.0"
  category: "utilities"

runtime:
  environments: ["docker"]
  entrypoint: "sh"
  docker:
    image: "alpine:latest"
    network: "bridge"
`

func requireDockerDaemon(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("docker daemon not reachable, skipping docker execution test")
	}
	if err := exec.Command("docker", "image", "inspect", "alpine:latest").Run(); err != nil {
		t.Skip("alpine:latest image not available locally, skipping docker execution test")
	}
}

func loadDockerTestPlugin(t *testing.T, yamlContent, pluginID string) (*ScriptExecutor, *models.PluginV2, string) {
	t.Helper()

	pluginsDir := t.TempDir()
	pluginDir := filepath.Join(pluginsDir, pluginID)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write plugin.yaml: %v", err)
	}

	db := createTestDB(t)
	imageBuilder := NewDockerImageBuilder(db)
	loader := NewPluginLoaderV2(pluginsDir, db, imageBuilder)
	if err := loader.LoadPlugins(); err != nil {
		t.Fatalf("LoadPlugins error: %v", err)
	}

	ctx := context.WithValue(context.Background(), "wails-test", true)
	settings := NewSettingsService(ctx, db)
	executor := NewScriptExecutor(settings, db, "test")
	executor.SetPluginLoader(loader)

	plugins := loader.GetAllPlugins()
	for _, p := range plugins {
		if p.Definition.Plugin.ID == pluginID {
			return executor, p, pluginDir
		}
	}
	t.Fatalf("plugin %q did not load", pluginID)
	return nil, nil, ""
}

func runDockerScript(t *testing.T, executor *ScriptExecutor, plugin *models.PluginV2, pluginDir string, shellCmd string) string {
	t.Helper()

	outputDir := t.TempDir()
	config := ScriptConfig{
		PluginID:     plugin.ID,
		Type:         plugin.Definition.Plugin.ID,
		Environments: []string{"docker"},
		ScriptName:   "sh",
		Args:         []string{"-c", shellCmd},
		OutputDir:    outputDir,
		FolderPath:   pluginDir,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := executor.ExecuteDockerScript(ctx, "docker-test-job-"+plugin.Definition.Plugin.ID, config); err != nil {
		t.Fatalf("ExecuteDockerScript error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outputDir, "result.txt"))
	if err != nil {
		t.Fatalf("failed to read result.txt: %v", err)
	}
	return strings.TrimSpace(string(data))
}

func TestExecuteDockerScript_WritesOutputWithHardenedDefaults(t *testing.T) {
	requireDockerDaemon(t)

	executor, plugin, pluginDir := loadDockerTestPlugin(t, dockerTestPluginYAML, "docker-hardening-test-plugin")

	result := runDockerScript(t, executor, plugin, pluginDir, "echo hello > /output/result.txt")
	if result != "hello" {
		t.Errorf("expected output file to contain %q, got %q", "hello", result)
	}
}

func TestExecuteDockerScript_DefaultNetworkIsBlocked(t *testing.T) {
	requireDockerDaemon(t)

	executor, plugin, pluginDir := loadDockerTestPlugin(t, dockerTestPluginYAML, "docker-hardening-test-plugin")

	result := runDockerScript(t, executor, plugin, pluginDir, "ip route show default 2>&1 | wc -l > /output/result.txt")
	lines, err := strconv.Atoi(result)
	if err != nil {
		t.Fatalf("unexpected result content %q: %v", result, err)
	}
	if lines != 0 {
		t.Errorf("expected no default route with --network none, got %d route line(s)", lines)
	}
}

func TestExecuteDockerScript_NetworkOptOutAllowsAccess(t *testing.T) {
	requireDockerDaemon(t)

	executor, plugin, pluginDir := loadDockerTestPlugin(t, dockerTestPluginYAMLBridge, "docker-hardening-test-plugin-bridge")

	result := runDockerScript(t, executor, plugin, pluginDir, "ip route show default 2>&1 | wc -l > /output/result.txt")
	lines, err := strconv.Atoi(result)
	if err != nil {
		t.Fatalf("unexpected result content %q: %v", result, err)
	}
	if lines == 0 {
		t.Error("expected a default route when the plugin declares network: bridge, got none")
	}
}
