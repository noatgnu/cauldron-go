package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func TestLoadDependencyGraph_MissingFileReturnsNil(t *testing.T) {
	graph, err := loadDependencyGraph(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if graph != nil {
		t.Errorf("expected nil graph, got %+v", graph)
	}
}

func TestLoadDependencyGraph_ParsesValidFile(t *testing.T) {
	dir := t.TempDir()
	content := `{"generatedAt":"2026-01-01T00:00:00Z","systemRequirements":{"linux":[{"name":"glpk","packages":["libglpk-dev"]}]}}`
	if err := os.WriteFile(filepath.Join(dir, "dependencies.json"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	graph, err := loadDependencyGraph(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if graph == nil {
		t.Fatal("expected a graph, got nil")
	}
	if len(graph.SystemRequirements["linux"]) != 1 || graph.SystemRequirements["linux"][0].Name != "glpk" {
		t.Errorf("unexpected system requirements: %+v", graph.SystemRequirements)
	}
}

func TestLoadDependencyGraph_InvalidJSONErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dependencies.json"), []byte("not json"), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	if _, err := loadDependencyGraph(dir); err == nil {
		t.Error("expected an error for invalid JSON, got nil")
	}
}

func TestInstallCommandFor(t *testing.T) {
	cases := []struct {
		goos string
		want string
	}{
		{"linux", "sudo apt install libglpk-dev"},
		{"darwin", "brew install libglpk-dev"},
		{"windows", ""},
	}
	for _, c := range cases {
		if got := installCommandFor(c.goos, []string{"libglpk-dev"}); got != c.want {
			t.Errorf("installCommandFor(%q) = %q, want %q", c.goos, got, c.want)
		}
	}
}

func TestReportSystemRequirements_NoFileReportsNothing(t *testing.T) {
	var messages []string
	reportSystemRequirements(t.TempDir(), func(s string) { messages = append(messages, s) })
	if len(messages) != 0 {
		t.Errorf("expected no messages without dependencies.json, got %v", messages)
	}
}

func TestReportSystemRequirements_ReportsCurrentPlatformOnly(t *testing.T) {
	dir := t.TempDir()
	graph := models.PluginDependencyGraph{
		SystemRequirements: map[string][]models.PluginSystemRequirement{
			"linux":  {{Name: "glpk", Packages: []string{"libglpk-dev"}}},
			"darwin": {{Name: "glpk", Packages: []string{"glpk"}}},
		},
	}
	writeDependencyGraphFixture(t, dir, graph)

	var messages []string
	reportSystemRequirements(dir, func(s string) { messages = append(messages, s) })

	if goruntime.GOOS == "windows" {
		if len(messages) != 0 {
			t.Errorf("expected no message on windows, got %v", messages)
		}
		return
	}

	if len(messages) != 1 {
		t.Fatalf("expected exactly 1 message, got %v", messages)
	}
}

func writeDependencyGraphFixture(t *testing.T, dir string, graph models.PluginDependencyGraph) {
	t.Helper()
	data, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("failed to marshal fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dependencies.json"), data, 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
}
