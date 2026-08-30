package generator

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/noatgnu/cauldron-go/backend/models"
)

type EnvVarData struct {
	Name        string
	Value       string
	Description string
}

type ProcessData struct {
	ProcessName               string
	ModulePath                string
	Label                     string
	ContainerImageSingularity string
	ContainerImageDocker      string
	Inputs                    []models.PluginInputV2
	Outputs                   []models.PluginOutputV2
	Entrypoint                string
	Args                      []ArgData
	EnvVars                   []EnvVarData
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

func findFormatOptionsFromInputs(inputs []models.PluginInputV2) []string {
	formatKeywords := []string{"format", "file_format", "output_format", "plot_format"}
	imageFormats := []string{"svg", "png", "pdf", "jpg", "jpeg", "tiff"}

	for _, input := range inputs {
		nameLC := strings.ToLower(input.Name)
		for _, keyword := range formatKeywords {
			if strings.Contains(nameLC, keyword) && len(input.Options) > 0 {
				hasImageFormat := false
				for _, opt := range input.Options {
					optLC := strings.ToLower(opt.Value)
					for _, imgFmt := range imageFormats {
						if optLC == imgFmt {
							hasImageFormat = true
							break
						}
					}
				}
				if hasImageFormat {
					var values []string
					for _, opt := range input.Options {
						values = append(values, opt.Value)
					}
					return values
				}
			}
		}
	}
	return nil
}

func replaceFormatExtensionInPath(path string, formatOptions []string) string {
	for _, ext := range formatOptions {
		extWithDot := "." + strings.ToLower(ext)
		if strings.HasSuffix(strings.ToLower(path), extWithDot) {
			return path[:len(path)-len(extWithDot)] + ".*"
		}
		wildcardExt := "*" + extWithDot
		if strings.HasSuffix(strings.ToLower(path), wildcardExt) {
			return path[:len(path)-len(wildcardExt)] + "*.*"
		}
	}
	return path
}

func processOutputsForDynamicFormat(outputs []models.PluginOutputV2, inputs []models.PluginInputV2) []models.PluginOutputV2 {
	formatOptions := findFormatOptionsFromInputs(inputs)
	if len(formatOptions) == 0 {
		return outputs
	}

	processed := make([]models.PluginOutputV2, len(outputs))
	for i, output := range outputs {
		processed[i] = output
		processed[i].Path = replaceFormatExtensionInPath(output.Path, formatOptions)
	}
	return processed
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

	processedOutputs := processOutputsForDynamicFormat(definition.Outputs, definition.Inputs)

	data := ProcessData{
		ProcessName:               toNextflowID(definition.Plugin.ID),
		ModulePath:                definition.Plugin.ID,
		Label:                     "process_medium",
		ContainerImageSingularity: singularityImage,
		ContainerImageDocker:      imageName,
		Outputs:                   processedOutputs,
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

	for _, envVar := range definition.Execution.EnvVariables {
		defaultVal := ""
		if envVar.Default != nil {
			defaultVal = fmt.Sprintf("%v", envVar.Default)
		}
		data.EnvVars = append(data.EnvVars, EnvVarData{
			Name:        envVar.Name,
			Value:       defaultVal,
			Description: envVar.Description,
		})
	}

	// Map ArgsMapping to ArgData in a deterministic order: declared-input order first, then any leftover keys sorted alphabetically.
	handled := make(map[string]bool, len(definition.Execution.ArgsMapping))
	for _, input := range definition.Inputs {
		mapping, ok := definition.Execution.ArgsMapping[input.Name]
		if !ok {
			continue
		}
		handled[input.Name] = true
		if arg := buildArgData(input.Name, mapping, definition.Inputs); arg.Flag != "" || arg.StaticValue != "" {
			data.Args = append(data.Args, arg)
		}
	}
	remaining := make([]string, 0, len(definition.Execution.ArgsMapping))
	for inputName := range definition.Execution.ArgsMapping {
		if !handled[inputName] {
			remaining = append(remaining, inputName)
		}
	}
	sort.Strings(remaining)
	for _, inputName := range remaining {
		if arg := buildArgData(inputName, definition.Execution.ArgsMapping[inputName], definition.Inputs); arg.Flag != "" || arg.StaticValue != "" {
			data.Args = append(data.Args, arg)
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute process template: %w", err)
	}

	return buf.String(), nil
}

func buildArgData(inputName string, mapping interface{}, inputs []models.PluginInputV2) ArgData {
	arg := ArgData{InputName: inputName}

	for _, input := range inputs {
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

	return arg
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
	tmpl, err := template.New("readme").Funcs(template.FuncMap{
		"upper": strings.ToUpper,
	}).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse readme template: %w", err)
	}

	data := ProcessData{
		ProcessName:          toNextflowID(definition.Plugin.ID),
		ModulePath:           definition.Plugin.ID,
		ToolName:             definition.Plugin.Name,
		Version:              definition.Plugin.Version,
		Description:          definition.Plugin.Description,
		Author:               definition.Plugin.Author,
		Inputs:               definition.Inputs,
		ContainerImageDocker: discoverImage(definition),
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
