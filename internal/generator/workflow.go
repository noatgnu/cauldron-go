package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/noatgnu/cauldron-go/backend/models"
)

type WorkflowData struct {
	WorkflowName string
	ProcessName  string
	Inputs       []models.PluginInputV2
}

func GenerateWorkflow(definition *models.PluginDefinition, tmplStr string) (string, error) {
	tmpl, err := template.New("workflow").Funcs(template.FuncMap{
		"upper": strings.ToUpper,
	}).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse workflow template: %w", err)
	}

	data := WorkflowData{
		WorkflowName: definition.Plugin.ID,
		ProcessName:  definition.Plugin.ID,
	}

	for _, input := range definition.Inputs {
		if input.Name == "input_file" {
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
