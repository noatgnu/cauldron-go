package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func TestGenerateRecipeDiagram_Overview(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	recipe, err := recipeService.SaveRecipe("My Recipe", "A test recipe", testRecipeStages())
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	diagram, err := recipeService.GenerateRecipeDiagram(recipe.ID, nil)
	if err != nil {
		t.Fatalf("GenerateRecipeDiagram failed: %v", err)
	}

	if !strings.HasPrefix(diagram, "flowchart TD\n") {
		t.Fatalf("expected diagram to start with flowchart TD, got: %s", diagram)
	}
	if !strings.Contains(diagram, "S0[") || !strings.Contains(diagram, "chain-upstream") {
		t.Errorf("expected stage 0 node for chain-upstream, got: %s", diagram)
	}
	if !strings.Contains(diagram, "S1[") || !strings.Contains(diagram, "chain-downstream") {
		t.Errorf("expected stage 1 node for chain-downstream, got: %s", diagram)
	}
	if !strings.Contains(diagram, `S0 -->|"result"| S1`) {
		t.Errorf("expected binding edge from S0 to S1 labeled result, got: %s", diagram)
	}
	if !strings.Contains(diagram, "class S0 compatible") || !strings.Contains(diagram, "class S1 compatible") {
		t.Errorf("expected both stages classed compatible, got: %s", diagram)
	}
}

func TestGenerateRecipeDiagram_Expanded(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	recipe, err := recipeService.SaveRecipe("My Recipe", "A test recipe", testRecipeStages())
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	diagram, err := recipeService.GenerateRecipeDiagram(recipe.ID, []int{0, 1})
	if err != nil {
		t.Fatalf("GenerateRecipeDiagram failed: %v", err)
	}

	if !strings.Contains(diagram, "subgraph S0_group") {
		t.Errorf("expected subgraph for stage 0, got: %s", diagram)
	}
	if !strings.Contains(diagram, "S0_out_result") {
		t.Errorf("expected output port for stage 0's result output, got: %s", diagram)
	}
	if !strings.Contains(diagram, "S1_in_input_file") {
		t.Errorf("expected input port for stage 1's input_file input, got: %s", diagram)
	}
	if !strings.Contains(diagram, "S0_out_result -.-> S1_in_input_file") {
		t.Errorf("expected dotted binding edge between resolved ports, got: %s", diagram)
	}
}

func TestGenerateRecipeDiagram_MissingPlugin(t *testing.T) {
	_, recipeService, pluginLoader := setupRecipeTestServices(t)

	recipe, err := recipeService.SaveRecipe("My Recipe", "A test recipe", testRecipeStages())
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	pluginLoader.plugins = map[uint]*models.PluginV2{}

	diagram, err := recipeService.GenerateRecipeDiagram(recipe.ID, []int{0, 1})
	if err != nil {
		t.Fatalf("GenerateRecipeDiagram failed: %v", err)
	}
	if !strings.Contains(diagram, "plugin not installed") {
		t.Errorf("expected missing-plugin fallback label, got: %s", diagram)
	}
	if !strings.Contains(diagram, "class S0 missing") {
		t.Errorf("expected stage 0 classed missing, got: %s", diagram)
	}
}

func TestGenerateRecipeDiagram_NoStages(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	if _, err := recipeService.GenerateRecipeDiagram("does-not-exist", nil); err == nil {
		t.Fatal("expected error for a recipe with no stages")
	}
}

func testRecipeDataJSON(t *testing.T) []byte {
	t.Helper()
	data := RecipeData{
		Version: RecipeFormatVersion,
		Label:   "Registry Recipe",
		Stages: []RecipeStageData{
			{PluginID: "chain-upstream", PluginVersion: "1.0.0", Params: map[string]interface{}{}, Bindings: map[string]models.RecipeStageBinding{}},
			{PluginID: "chain-downstream", PluginVersion: "1.0.0", Params: map[string]interface{}{}, Bindings: map[string]models.RecipeStageBinding{
				"input_file": {Stage: 0, Output: "result"},
			}},
		},
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal recipe data: %v", err)
	}
	return raw
}

func TestGenerateRecipeDiagramFromData_Overview(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	diagram, err := recipeService.GenerateRecipeDiagramFromData(testRecipeDataJSON(t), nil)
	if err != nil {
		t.Fatalf("GenerateRecipeDiagramFromData failed: %v", err)
	}

	if !strings.Contains(diagram, "S0[") || !strings.Contains(diagram, "chain-upstream") {
		t.Errorf("expected stage 0 node for chain-upstream, got: %s", diagram)
	}
	if !strings.Contains(diagram, `S0 -->|"result"| S1`) {
		t.Errorf("expected binding edge from S0 to S1 labeled result, got: %s", diagram)
	}
	if !strings.Contains(diagram, "class S0 compatible") {
		t.Errorf("expected stage 0 classed compatible since the plugin is installed, got: %s", diagram)
	}
}

func TestGenerateRecipeDiagramFromData_UninstalledPlugin(t *testing.T) {
	_, recipeService, pluginLoader := setupRecipeTestServices(t)
	pluginLoader.plugins = map[uint]*models.PluginV2{}

	diagram, err := recipeService.GenerateRecipeDiagramFromData(testRecipeDataJSON(t), []int{0, 1})
	if err != nil {
		t.Fatalf("GenerateRecipeDiagramFromData failed: %v", err)
	}

	if !strings.Contains(diagram, "class S0 missing") || !strings.Contains(diagram, "class S1 missing") {
		t.Errorf("expected both stages classed missing since no plugins are installed, got: %s", diagram)
	}
	if !strings.Contains(diagram, "plugin not installed") {
		t.Errorf("expected missing-plugin fallback label, got: %s", diagram)
	}
}

func TestGenerateRecipeDiagramFromData_NoStages(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	raw, _ := json.Marshal(RecipeData{Version: RecipeFormatVersion, Label: "Empty"})
	if _, err := recipeService.GenerateRecipeDiagramFromData(raw, nil); err == nil {
		t.Fatal("expected error for recipe data with no stages")
	}
}

const testPluginStepReadme = "## Workflow Diagram\n\n```mermaid\nflowchart TD\n    Start([Start]) --> step1\n    step1[\"Load input\"]\n    step1 --> step2\n    step2[\"Write output\"]\n    step2 --> End([End])\n```\n"

func TestPluginStepDiagramLines(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(testPluginStepReadme), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	lines, ok := pluginStepDiagramLines(&models.PluginV2{FolderPath: dir})
	if !ok {
		t.Fatal("expected a step diagram to be found")
	}
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "flowchart TD") {
		t.Errorf("expected the flowchart header line to be stripped, got: %s", joined)
	}
	if !strings.Contains(joined, `step1["Load input"]`) {
		t.Errorf("expected step1's label line, got: %s", joined)
	}
}

func TestPluginStepDiagramLines_NoReadme(t *testing.T) {
	_, ok := pluginStepDiagramLines(&models.PluginV2{FolderPath: t.TempDir()})
	if ok {
		t.Error("expected no step diagram when README.md is missing")
	}
}

func TestPluginStepDiagramLines_NoMermaidBlock(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Just a plain readme\n"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	_, ok := pluginStepDiagramLines(&models.PluginV2{FolderPath: dir})
	if ok {
		t.Error("expected no step diagram when the README has no mermaid block")
	}
}

func TestRenamespaceMermaidLines(t *testing.T) {
	lines := []string{
		`Start([Start]) --> step1`,
		`step1["Load input"]`,
		`step1 --> step2`,
		`step2["Write output"]`,
		`step2 --> End([End])`,
	}

	renamed := renamespaceMermaidLines(lines, "S2_step_")
	joined := strings.Join(renamed, "\n")

	if !strings.Contains(joined, `S2_step_Start(["Start"])`) {
		t.Errorf("expected Start's id renamed but its label preserved, got: %s", joined)
	}
	if !strings.Contains(joined, `S2_step_step1["Load input"]`) {
		t.Errorf("expected step1's id renamed but its label preserved, got: %s", joined)
	}
	if !strings.Contains(joined, `S2_step_step1 --> S2_step_step2`) {
		t.Errorf("expected both sides of the edge renamed, got: %s", joined)
	}
	if !strings.Contains(joined, `S2_step_step2 --> S2_step_End(["End"])`) {
		t.Errorf("expected End's id renamed but its label preserved, got: %s", joined)
	}
	if strings.Contains(joined, "Load S2_step_input") {
		t.Errorf("expected label text to be left alone, got: %s", joined)
	}
}

func TestGenerateRecipeDiagram_PartiallyExpanded(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	recipe, err := recipeService.SaveRecipe("My Recipe", "A test recipe", testRecipeStages())
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	diagram, err := recipeService.GenerateRecipeDiagram(recipe.ID, []int{1})
	if err != nil {
		t.Fatalf("GenerateRecipeDiagram failed: %v", err)
	}

	if !strings.Contains(diagram, `S0[`) {
		t.Errorf("expected stage 0 to stay collapsed as a plain box, got: %s", diagram)
	}
	if strings.Contains(diagram, "subgraph S0_group") {
		t.Errorf("expected stage 0 to not be expanded, got: %s", diagram)
	}
	if !strings.Contains(diagram, "subgraph S1_group") {
		t.Errorf("expected stage 1 to be expanded, got: %s", diagram)
	}
	if !strings.Contains(diagram, "S1_in_input_file") {
		t.Errorf("expected stage 1's input port, got: %s", diagram)
	}
	if !strings.Contains(diagram, `S0 -.-> S1_in_input_file`) {
		t.Errorf("expected the binding edge to target stage 1's specific input port since it's expanded, got: %s", diagram)
	}
}

func TestGenerateRecipeDiagram_CollapsedToCollapsedEdgeIsLabeled(t *testing.T) {
	_, recipeService, _ := setupRecipeTestServices(t)

	recipe, err := recipeService.SaveRecipe("My Recipe", "A test recipe", testRecipeStages())
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	diagram, err := recipeService.GenerateRecipeDiagram(recipe.ID, nil)
	if err != nil {
		t.Fatalf("GenerateRecipeDiagram failed: %v", err)
	}

	if !strings.Contains(diagram, `S0 -->|"result"| S1`) {
		t.Errorf("expected a labeled edge between two collapsed stages, got: %s", diagram)
	}
	if strings.Contains(diagram, "-.->") {
		t.Errorf("expected no dotted port edges when nothing is expanded, got: %s", diagram)
	}
}

func TestGenerateRecipeDiagram_Expanded_NestsPluginStepDiagram(t *testing.T) {
	_, recipeService, pluginLoader := setupRecipeTestServices(t)

	upstream, err := pluginLoader.GetPluginByStringID("chain-upstream")
	if err != nil {
		t.Fatalf("failed to look up chain-upstream: %v", err)
	}
	if err := os.WriteFile(filepath.Join(upstream.FolderPath, "README.md"), []byte(testPluginStepReadme), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	recipe, err := recipeService.SaveRecipe("My Recipe", "A test recipe", testRecipeStages())
	if err != nil {
		t.Fatalf("SaveRecipe failed: %v", err)
	}

	diagram, err := recipeService.GenerateRecipeDiagram(recipe.ID, []int{0, 1})
	if err != nil {
		t.Fatalf("GenerateRecipeDiagram failed: %v", err)
	}

	if !strings.Contains(diagram, "S0_step_step1") {
		t.Errorf("expected stage 0's nested step diagram, got: %s", diagram)
	}
	if strings.Contains(diagram, "S0_out_result") {
		t.Errorf("expected stage 0 to skip the port-based rendering when a step diagram is available, got: %s", diagram)
	}
	if !strings.Contains(diagram, "S1_in_input_file") {
		t.Errorf("expected stage 1 (no README) to still fall back to port-based rendering, got: %s", diagram)
	}
	if !strings.Contains(diagram, "S0 -.-> S1_in_input_file") {
		t.Errorf("expected the binding edge to originate from stage 0's plain anchor node since it has no output port, got: %s", diagram)
	}
}
