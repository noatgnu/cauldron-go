package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/noatgnu/cauldron-go/backend/models"
)

type WorkflowData struct {
	WorkflowName   string
	ProcessName    string
	ModulePath     string
	Inputs         []models.PluginInputV2
	HasInputFile   bool
	InputFileInput *models.PluginInputV2
}

func GenerateWorkflow(definition *models.PluginDefinition, tmplStr string) (string, error) {
	tmpl, err := template.New("workflow").Funcs(template.FuncMap{
		"upper": strings.ToUpper,
	}).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse workflow template: %w", err)
	}

	data := WorkflowData{
		WorkflowName: ToNextflowID(definition.Plugin.ID),
		ProcessName:  ToNextflowID(definition.Plugin.ID),
		ModulePath:   definition.Plugin.ID,
	}

	for _, input := range definition.Inputs {
		inputCopy := input
		if input.Name == "input_file" {
			data.HasInputFile = true
			data.InputFileInput = &inputCopy
			continue
		}
		data.Inputs = append(data.Inputs, input)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute workflow template: %w", err)
	}

	return buf.String(), nil
}

// RecipeWorkflowStage is one stage of a multi-plugin Recipe pipeline: which
// process to call, under what per-stage alias (so the same plugin can be
// used in more than one stage via Nextflow's `include { X as ALIAS }`), and
// where each of its declared inputs' values come from.
type RecipeWorkflowStage struct {
	ProcessName string // ToNextflowID(pluginID), upper — the process's own declared name in its module file
	CallAlias   string // unique per stage: ToNextflowID(pluginID) + "_S" + stage index
	ModulePath  string // pluginID, matches modules/local/<ModulePath>/main
	Args        []RecipeArgSource
}

// RecipeArgSource describes where one process input's value comes from: an
// unbound (namespaced) pipeline param, or an upstream stage's output channel.
type RecipeArgSource struct {
	Name            string
	IsFile          bool
	FromParam       bool
	ParamName       string
	SourceCallAlias string
	SourceEmit      string
}

func GenerateRecipeWorkflow(stages []RecipeWorkflowStage, tmplStr string) (string, error) {
	tmpl, err := template.New("recipe-workflow").Funcs(template.FuncMap{
		"upper": strings.ToUpper,
	}).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse recipe workflow template: %w", err)
	}

	data := struct {
		Stages []RecipeWorkflowStage
	}{Stages: stages}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute recipe workflow template: %w", err)
	}

	return buf.String(), nil
}
