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
	UseAppPrefix              bool
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

func toNextflowID(id string) string {
	result := strings.ReplaceAll(id, "-", "_")
	result = strings.ReplaceAll(result, ".", "_")
	return result
}

func GenerateProcess(definition *models.PluginDefinition, tmplStr string) (string, error) {
	tmpl, err := template.New("process").Funcs(template.FuncMap{
		"upper":      strings.ToUpper,
		"replace":    strings.ReplaceAll,
		"nextflowID": toNextflowID,
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

	entrypoint := definition.Runtime.GetEntrypoint()
	useAppPrefix := true
	if definition.Runtime.Docker != nil && definition.Runtime.Docker.Image != "" {
		useAppPrefix = false
	}
	if strings.HasPrefix(entrypoint, "/") {
		useAppPrefix = false
	}

	data := ProcessData{
		ProcessName:               toNextflowID(definition.Plugin.ID),
		Label:                     "process_medium",
		ContainerImageSingularity: singularityImage,
		ContainerImageDocker:      imageName,
		Outputs:                   definition.Outputs,
		Entrypoint:                entrypoint,
		ToolName:                  definition.Plugin.Name,
		Version:                   definition.Plugin.Version,
		Description:               definition.Plugin.Description,
		Author:                    definition.Plugin.Author,
		OutputDirFlag:             definition.Execution.OutputDir,
		Inputs:                    definition.Inputs,
		PrimaryEnv:                definition.Runtime.GetPrimaryEnvironment(),
		UseAppPrefix:              useAppPrefix,
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

	// 2. Fallback to cauldron namespace to avoid clashing
	return "cauldron/" + definition.Plugin.ID + ":" + definition.Plugin.Version
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

type GithubActionData struct {
	PluginID       string
	Version        string
	DockerPlatform string
	BuildArgs      map[string]string
	HasExample     bool
	ExampleArgs    string
	ExampleDirs    []string
}

func GenerateGithubAction(definition *models.PluginDefinition, tmplStr string) (string, error) {
	tmpl, err := template.New("github-action").Delims("[[", "]]").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse github-action template: %w", err)
	}

	data := GithubActionData{
		PluginID:       definition.Plugin.ID,
		Version:        definition.Plugin.Version,
		DockerPlatform: "linux/amd64",
	}

	if definition.Runtime.Docker != nil {
		if definition.Runtime.Docker.Platform != "" {
			data.DockerPlatform = definition.Runtime.Docker.Platform
		}
		data.BuildArgs = definition.Runtime.Docker.BuildArgs
	}

	if definition.Example != nil && definition.Example.Enabled {
		data.HasExample = true
		exampleDirsMap := make(map[string]bool)
		var args []string
		for key, value := range definition.Example.Values {
			strVal := fmt.Sprintf("%v", value)
			if strings.Contains(strVal, "/") && !strings.HasPrefix(strVal, "-") {
				dir := strings.Split(strVal, "/")[0]
				exampleDirsMap[dir] = true
				args = append(args, fmt.Sprintf("--%s examples/%s", key, strVal))
			} else {
				args = append(args, fmt.Sprintf("--%s '%v'", key, value))
			}
		}
		data.ExampleArgs = strings.Join(args, " \\\n          ")
		for dir := range exampleDirsMap {
			data.ExampleDirs = append(data.ExampleDirs, dir)
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute github-action template: %w", err)
	}

	return buf.String(), nil
}
