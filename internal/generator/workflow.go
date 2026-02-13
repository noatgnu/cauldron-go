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
		WorkflowName: toNextflowID(definition.Plugin.ID),
		ProcessName:  toNextflowID(definition.Plugin.ID),
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
