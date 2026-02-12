package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/noatgnu/cauldron-go/backend/models"
)

type ProcessData struct {
	ProcessName               string
	Label                     string
	ContainerImageSingularity string
	ContainerImageDocker      string
	Inputs                    []models.PluginInputV2
	Outputs                   []models.PluginOutputV2
	Entrypoint                string
	Args                      []ArgData
	ToolName                  string
	Version                   string
}

type ArgData struct {
	Flag  string
	Value string
}

func GenerateProcess(definition *models.PluginDefinition, tmplStr string) (string, error) {
	tmpl, err := template.New("process").Funcs(template.FuncMap{
		"upper": strings.ToUpper,
	}).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse process template: %w", err)
	}

	data := ProcessData{
		ProcessName:               definition.Plugin.ID,
		Label:                     "process_medium",
		ContainerImageSingularity: fmt.Sprintf("oras://ghcr.io/noatgnu/%s:%s", definition.Plugin.ID, definition.Plugin.Version),
		ContainerImageDocker:      fmt.Sprintf("ghcr.io/noatgnu/%s:%s", definition.Plugin.ID, definition.Plugin.Version),
		Outputs:                   definition.Outputs,
		Entrypoint:                definition.Runtime.GetEntrypoint(),
		ToolName:                  definition.Plugin.Name,
		Version:                   definition.Plugin.Version,
	}

	// Filter inputs to avoid duplicating input_file which is handled by tuple
	for _, input := range definition.Inputs {
		if input.Name == "input_file" {
			continue
		}
		data.Inputs = append(data.Inputs, input)
	}

	// Map ArgsMapping to ArgData
	for name, mapping := range definition.Execution.ArgsMapping {
		// This is a simplification, need to handle complex mappings
		flag := ""
		switch v := mapping.(type) {
		case string:
			flag = v
		case map[string]interface{}:
			if f, ok := v["flag"].(string); ok {
				flag = f
			}
		}

		if flag != "" {
			data.Args = append(data.Args, ArgData{
				Flag:  flag,
				Value: fmt.Sprintf("${%s}", name),
			})
		}
	}

	// Add outputDir
	if definition.Execution.OutputDir != "" {
		data.Args = append(data.Args, ArgData{
			Flag:  definition.Execution.OutputDir,
			Value: ".",
		})
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute process template: %w", err)
	}

	return buf.String(), nil
}
