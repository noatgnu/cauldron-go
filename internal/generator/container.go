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
	// If the plugin already uses a Docker image directly, we don't need to generate a Dockerfile
	// that installs requirements, but for consistency in the export, we'll still provide one
	// that uses their image as a base or simply point them to their image.

	if definition.Runtime.IsDockerRuntime() && definition.Runtime.Docker != nil {
		if definition.Runtime.Docker.Image != "" {
			// If they have an image, we can just use it.
			tmpl, err := template.New("dockerfile").Parse("FROM " + definition.Runtime.Docker.Image + "\nLABEL tool=\"" + definition.Runtime.GetEntrypoint() + "\"\n")
			if err != nil {
				return "", err
			}
			var buf bytes.Buffer
			tmpl.Execute(&buf, nil)
			return buf.String(), nil
		}

		// If they have a custom Dockerfile path, the caller (main.go) will handle
		// copying it, so we don't generate one here.
		if definition.Runtime.Docker.Dockerfile != "" {
			return "", nil
		}
	}

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
		data.BaseImage = "rocker/r-ver:4.3"
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
