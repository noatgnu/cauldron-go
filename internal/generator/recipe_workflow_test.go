package generator

import (
	"strings"
	"testing"

	"github.com/noatgnu/cauldron-go/internal/templates"
)

func testRecipeStages() []RecipeWorkflowStage {
	return []RecipeWorkflowStage{
		{
			ProcessName: "wide_to_long",
			CallAlias:   "wide_to_long_S0",
			ModulePath:  "wide-to-long",
			Args: []RecipeArgSource{
				{Name: "input_file", IsFile: true, FromParam: true, ParamName: "stage0_input_file"},
			},
		},
		{
			ProcessName: "long_to_wide",
			CallAlias:   "long_to_wide_S1",
			ModulePath:  "long-to-wide",
			Args: []RecipeArgSource{
				{Name: "input_file", IsFile: true, FromParam: false, SourceCallAlias: "wide_to_long_S0", SourceEmit: "long_data"},
				{Name: "delimiter", IsFile: false, FromParam: true, ParamName: "stage1_delimiter"},
			},
		},
	}
}

func recipeMainTemplate(t *testing.T) string {
	t.Helper()
	tmpl, err := templates.GetTemplate("recipe-main.nf.tmpl")
	if err != nil {
		t.Fatalf("failed to load recipe-main.nf.tmpl: %v", err)
	}
	return tmpl
}

func TestGenerateRecipeWorkflow_IncludesEachStageWithItsAlias(t *testing.T) {
	content, err := GenerateRecipeWorkflow(testRecipeStages(), recipeMainTemplate(t))
	if err != nil {
		t.Fatalf("GenerateRecipeWorkflow error: %v", err)
	}

	if !strings.Contains(content, "include { WIDE_TO_LONG as WIDE_TO_LONG_S0 } from './modules/local/wide-to-long/main'") {
		t.Errorf("expected aliased include for stage 0, got:\n%s", content)
	}
	if !strings.Contains(content, "include { LONG_TO_WIDE as LONG_TO_WIDE_S1 } from './modules/local/long-to-wide/main'") {
		t.Errorf("expected aliased include for stage 1, got:\n%s", content)
	}
}

func TestGenerateRecipeWorkflow_ParamSourcedFileArg(t *testing.T) {
	content, err := GenerateRecipeWorkflow(testRecipeStages(), recipeMainTemplate(t))
	if err != nil {
		t.Fatalf("GenerateRecipeWorkflow error: %v", err)
	}

	if !strings.Contains(content, "params.stage0_input_file ? Channel.fromPath(params.stage0_input_file).collect() : Channel.of([])") {
		t.Errorf("expected stage 0's param-sourced file channel, got:\n%s", content)
	}
}

func TestGenerateRecipeWorkflow_ChannelSourcedArgReferencesUpstreamOutput(t *testing.T) {
	content, err := GenerateRecipeWorkflow(testRecipeStages(), recipeMainTemplate(t))
	if err != nil {
		t.Fatalf("GenerateRecipeWorkflow error: %v", err)
	}

	if !strings.Contains(content, "WIDE_TO_LONG_S0.out.long_data.collect(),") {
		t.Errorf("expected stage 1's binding to reference stage 0's output channel, got:\n%s", content)
	}
}

func TestGenerateRecipeWorkflow_ParamSourcedValueArg(t *testing.T) {
	content, err := GenerateRecipeWorkflow(testRecipeStages(), recipeMainTemplate(t))
	if err != nil {
		t.Fatalf("GenerateRecipeWorkflow error: %v", err)
	}

	if !strings.Contains(content, "Channel.value(params.stage1_delimiter != null ? params.stage1_delimiter : '')") {
		t.Errorf("expected stage 1's non-file param as a value channel, got:\n%s", content)
	}
}

func TestGenerateRecipeWorkflow_SamePluginTwiceGetsDistinctAliases(t *testing.T) {
	stages := []RecipeWorkflowStage{
		{ProcessName: "wide_to_long", CallAlias: "wide_to_long_S0", ModulePath: "wide-to-long"},
		{ProcessName: "wide_to_long", CallAlias: "wide_to_long_S2", ModulePath: "wide-to-long"},
	}

	content, err := GenerateRecipeWorkflow(stages, recipeMainTemplate(t))
	if err != nil {
		t.Fatalf("GenerateRecipeWorkflow error: %v", err)
	}

	if !strings.Contains(content, "include { WIDE_TO_LONG as WIDE_TO_LONG_S0 }") {
		t.Errorf("expected first alias, got:\n%s", content)
	}
	if !strings.Contains(content, "include { WIDE_TO_LONG as WIDE_TO_LONG_S2 }") {
		t.Errorf("expected second alias, got:\n%s", content)
	}
	if strings.Count(content, "modules/local/wide-to-long/main") != 2 {
		t.Errorf("expected two include lines both pointing at the single shared module, got:\n%s", content)
	}
}

func TestGenerateRecipeConfig_NamespacesParamsAndKeepsOutdir(t *testing.T) {
	cfgTmpl, err := templates.GetTemplate("nextflow.config.tmpl")
	if err != nil {
		t.Fatalf("failed to load nextflow.config.tmpl: %v", err)
	}

	params := []ConfigParam{
		{Name: "stage0_input_file", Default: "null"},
		{Name: "stage1_delimiter", Default: "'tab'"},
	}

	content, err := GenerateRecipeConfig(params, cfgTmpl)
	if err != nil {
		t.Fatalf("GenerateRecipeConfig error: %v", err)
	}

	if !strings.Contains(content, "outdir = './results'") {
		t.Errorf("expected the default outdir param, got:\n%s", content)
	}
	if !strings.Contains(content, "stage0_input_file = null") {
		t.Errorf("expected stage 0's namespaced param, got:\n%s", content)
	}
	if !strings.Contains(content, "stage1_delimiter = 'tab'") {
		t.Errorf("expected stage 1's namespaced param, got:\n%s", content)
	}
}
