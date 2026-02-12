package generator

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/noatgnu/cauldron-go/backend/models"
)

type ContainerData struct {
	BaseImage    string
	Requirements []string
	Entrypoint   string
	HasR         bool
	HasPython    bool
}

func GenerateContainer(definition *models.PluginDefinition, tmplStr string) (string, error) {
	tmpl, err := template.New("dockerfile").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse dockerfile template: %w", err)
	}

	data := ContainerData{
		BaseImage:  "python:3.11-slim",
		Entrypoint: definition.Runtime.GetEntrypoint(),
	}

	if definition.Execution.Requirements.R != "" {
		data.HasR = true
		data.BaseImage = "rocker/r-ver:4.3" // Example R base image
	}

	if definition.Execution.Requirements.Python != "" {
		data.HasPython = true
	}

	data.Requirements = definition.Execution.Requirements.Packages

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute dockerfile template: %w", err)
	}

	return buf.String(), nil
}
