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
	Description               string
	Author                    string
	OutputDirFlag             string
	PrimaryEnv                string
	DockerPlatform            string
	BuildArgs                 map[string]string
}

type ArgData struct {
	InputName   string
	InputType   string
	Flag        string
	Transform   string
	When        string
	StaticValue string
	PassAsValue bool
	IsMultiple  bool
}

func GenerateProcess(definition *models.PluginDefinition, tmplStr string) (string, error) {
	tmpl, err := template.New("process").Funcs(template.FuncMap{
		"upper":   strings.ToUpper,
		"replace": strings.ReplaceAll,
	}).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse process template: %w", err)
	}

	imageName := discoverImage(definition)
	singularityImage := imageName
	if !strings.Contains(imageName, "://") && strings.Contains(imageName, "/") {
		// If it looks like a full registry path, help Singularity with the protocol
		singularityImage = "docker://" + imageName
	}

	data := ProcessData{
		ProcessName:               definition.Plugin.ID,
		Label:                     "process_medium",
		ContainerImageSingularity: singularityImage,
		ContainerImageDocker:      imageName,
		Outputs:                   definition.Outputs,
		Entrypoint:                definition.Runtime.GetEntrypoint(),
		ToolName:                  definition.Plugin.Name,
		Version:                   definition.Plugin.Version,
		Description:               definition.Plugin.Description,
		Author:                    definition.Plugin.Author,
		OutputDirFlag:             definition.Execution.OutputDir,
		Inputs:                    definition.Inputs,
		PrimaryEnv:                definition.Runtime.GetPrimaryEnvironment(),
	}

	if definition.Runtime.Docker != nil {
		data.DockerPlatform = definition.Runtime.Docker.Platform
		data.BuildArgs = definition.Runtime.Docker.BuildArgs
	}

	// Map ArgsMapping to ArgData
	for inputName, mapping := range definition.Execution.ArgsMapping {
		arg := ArgData{
			InputName: inputName,
		}

		// Find the corresponding input definition
		for _, input := range definition.Inputs {
			if input.Name == inputName {
				arg.InputType = string(input.Type)
				arg.IsMultiple = input.Multiple
				break
			}
		}

		switch v := mapping.(type) {
		case string:
			arg.Flag = v
		case map[string]interface{}:
			if f, ok := v["flag"].(string); ok {
				arg.Flag = f
			}
			if t, ok := v["transform"].(string); ok {
				arg.Transform = t
			}
			if w, ok := v["when"].(string); ok {
				arg.When = w
			}
			if sv, ok := v["value"].(string); ok {
				arg.StaticValue = sv
			}
			if pav, ok := v["passAsValue"].(bool); ok {
				arg.PassAsValue = pav
			}
		}

		if arg.Flag != "" || arg.StaticValue != "" {
			data.Args = append(data.Args, arg)
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute process template: %w", err)
	}

	return buf.String(), nil
}

func discoverImage(definition *models.PluginDefinition) string {
	// 1. If explicit docker image is provided in YAML, use it exactly as defined
	if definition.Runtime.Docker != nil && definition.Runtime.Docker.Image != "" {
		return definition.Runtime.Docker.Image
	}

	// 2. Fallback to a local tag (no hardcoded registry)
	return definition.Plugin.ID + ":" + definition.Plugin.Version
}

func GenerateREADME(definition *models.PluginDefinition, tmplStr string) (string, error) {
	tmpl, err := template.New("readme").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse readme template: %w", err)
	}

	data := ProcessData{
		ProcessName: definition.Plugin.ID,
		ToolName:    definition.Plugin.Name,
		Version:     definition.Plugin.Version,
		Description: definition.Plugin.Description,
		Author:      definition.Plugin.Author,
		Inputs:      definition.Inputs,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute readme template: %w", err)
	}

	return buf.String(), nil
}

func GenerateGithubAction(tmplStr string) (string, error) {
	tmpl, err := template.New("github-action").Delims("[[", "]]").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse github-action template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return "", fmt.Errorf("failed to execute github-action template: %w", err)
	}

	return buf.String(), nil
}
