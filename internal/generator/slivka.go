package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/noatgnu/cauldron-go/backend/models"
)

type SlivkaParameter struct {
	Name        string
	Label       string
	Description string
	Type        string
	Required    bool
	Default     string
	MediaType   string
	Min         string
	Max         string
	Options     []string
}

type SlivkaArg struct {
	Name      string
	Arg       string
	Symlink   string
	Condition string
}

type SlivkaOutput struct {
	Name      string
	Path      string
	MediaType string
}

type SlivkaTestParam struct {
	Name  string
	Value string
}

type SlivkaEnvVar struct {
	Name        string
	Value       string
	Description string
}

type SlivkaData struct {
	Name        string
	Description string
	Author      string
	Version     string
	Parameters  []SlivkaParameter
	Command     []string
	Args        []SlivkaArg
	Outputs     []SlivkaOutput
	EnvVars     []SlivkaEnvVar
	HasExample  bool
	TestParams  []SlivkaTestParam
}

func cauldronTypeToSlivka(inputType models.PluginInputType, accept string) (string, string) {
	mediaType := ""
	switch inputType {
	case models.PluginInputTypeFile:
		slivkaType := "file"
		if accept != "" {
			switch {
			case strings.Contains(accept, ".fasta") || strings.Contains(accept, ".fa"):
				mediaType = "application/fasta"
			case strings.Contains(accept, ".csv"):
				mediaType = "text/csv"
			case strings.Contains(accept, ".tsv") || strings.Contains(accept, ".txt"):
				mediaType = "text/tab-separated-values"
			case strings.Contains(accept, ".json"):
				mediaType = "application/json"
			case strings.Contains(accept, ".pdb"):
				mediaType = "chemical/x-pdb"
			case strings.Contains(accept, ".xml"):
				mediaType = "application/xml"
			}
		}
		return slivkaType, mediaType
	case models.PluginInputTypeNumber:
		return "float", ""
	case models.PluginInputTypeBoolean:
		return "boolean", ""
	case models.PluginInputTypeSelect:
		return "choice", ""
	default:
		return "text", ""
	}
}

type SlivkaArgConfig struct {
	Flag        string
	IsBoolean   bool
	PassAsValue bool
	When        string
}

func formatSlivkaArg(config SlivkaArgConfig) string {
	if config.IsBoolean {
		if config.PassAsValue {
			return fmt.Sprintf("%s=$(value)", config.Flag)
		}
		return config.Flag
	}
	return fmt.Sprintf("%s=$(value)", config.Flag)
}

func getSlivkaArgCondition(when string) string {
	switch when {
	case "true":
		return "$(value)"
	case "false":
		return "$(not value)"
	case "not-empty":
		return "$(value)"
	case "empty":
		return "$(not value)"
	default:
		return ""
	}
}

func parseArgMappingForSlivka(mappingInterface interface{}) (*models.ArgMapping, error) {
	switch v := mappingInterface.(type) {
	case string:
		return &models.ArgMapping{Flag: &v}, nil

	case map[string]interface{}:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var mapping models.ArgMapping
		if err := json.Unmarshal(jsonBytes, &mapping); err != nil {
			return nil, err
		}
		return &mapping, nil

	case map[interface{}]interface{}:
		converted := make(map[string]interface{})
		for key, val := range v {
			if keyStr, ok := key.(string); ok {
				converted[keyStr] = val
			}
		}
		return parseArgMappingForSlivka(converted)

	default:
		return nil, fmt.Errorf("unsupported mapping type: %T", mappingInterface)
	}
}

func GenerateSlivka(definition *models.PluginDefinition, tmplStr string) (string, error) {
	tmpl, err := template.New("slivka").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse slivka template: %w", err)
	}

	data := SlivkaData{
		Name:        definition.Plugin.ID,
		Description: definition.Plugin.Description,
		Author:      definition.Plugin.Author,
		Version:     definition.Plugin.Version,
	}

	entrypoint := definition.Runtime.GetEntrypoint()
	primaryEnv := definition.Runtime.GetPrimaryEnvironment()

	switch primaryEnv {
	case "python":
		data.Command = []string{"python", entrypoint}
	case "r":
		data.Command = []string{"Rscript", "--vanilla", "--slave", entrypoint, "--args"}
	default:
		data.Command = []string{entrypoint}
	}

	for _, input := range definition.Inputs {
		slivkaType, mediaType := cauldronTypeToSlivka(input.Type, input.Accept)

		param := SlivkaParameter{
			Name:        input.Name,
			Label:       input.Label,
			Description: input.Description,
			Type:        slivkaType,
			Required:    input.Required,
			MediaType:   mediaType,
		}

		if input.Default != nil {
			param.Default = fmt.Sprintf("%v", input.Default)
		}

		if input.Min != nil {
			param.Min = fmt.Sprintf("%v", *input.Min)
		}
		if input.Max != nil {
			param.Max = fmt.Sprintf("%v", *input.Max)
		}

		if len(input.Options) > 0 {
			param.Options = input.Options
		}

		data.Parameters = append(data.Parameters, param)

		if mappingInterface, ok := definition.Execution.ArgsMapping[input.Name]; ok {
			mapping, err := parseArgMappingForSlivka(mappingInterface)
			if err != nil {
				continue
			}

			flag := ""
			if mapping.Flag != nil {
				flag = *mapping.Flag
			} else {
				flag = fmt.Sprintf("--%s", input.Name)
			}

			when := ""
			if mapping.When != nil {
				when = *mapping.When
			}

			config := SlivkaArgConfig{
				Flag:        flag,
				IsBoolean:   input.Type == models.PluginInputTypeBoolean,
				PassAsValue: mapping.PassAsValue,
				When:        when,
			}

			arg := SlivkaArg{
				Name:      input.Name,
				Arg:       formatSlivkaArg(config),
				Condition: getSlivkaArgCondition(when),
			}

			if input.Type == models.PluginInputTypeFile {
				arg.Symlink = filepath.Base(input.Name)
			}

			data.Args = append(data.Args, arg)
		}
	}

	if definition.Execution.OutputDir != "" {
		data.Args = append(data.Args, SlivkaArg{
			Name: "_output_dir",
			Arg:  fmt.Sprintf("%s=.", definition.Execution.OutputDir),
		})
	}

	for _, output := range definition.Outputs {
		mediaType := ""
		switch output.Format {
		case "csv":
			mediaType = "text/csv"
		case "tsv":
			mediaType = "text/tab-separated-values"
		case "json":
			mediaType = "application/json"
		case "svg":
			mediaType = "image/svg+xml"
		case "png":
			mediaType = "image/png"
		case "pdf":
			mediaType = "application/pdf"
		case "txt":
			mediaType = "text/plain"
		}

		data.Outputs = append(data.Outputs, SlivkaOutput{
			Name:      output.Name,
			Path:      output.Path,
			MediaType: mediaType,
		})
	}

	for _, envVar := range definition.Execution.EnvVariables {
		defaultVal := ""
		if envVar.Default != nil {
			defaultVal = fmt.Sprintf("%v", envVar.Default)
		}
		data.EnvVars = append(data.EnvVars, SlivkaEnvVar{
			Name:        envVar.Name,
			Value:       defaultVal,
			Description: envVar.Description,
		})
	}

	if definition.Example != nil && definition.Example.Enabled {
		data.HasExample = true
		for key, value := range definition.Example.Values {
			data.TestParams = append(data.TestParams, SlivkaTestParam{
				Name:  key,
				Value: fmt.Sprintf("%v", value),
			})
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute slivka template: %w", err)
	}

	return buf.String(), nil
}

type SlivkaGithubActionData struct {
	PluginID       string
	Version        string
	HasExample     bool
	SlivkaTestArgs string
	ExampleDirs    []string
}

func GenerateSlivkaGithubAction(definition *models.PluginDefinition, tmplStr string) (string, error) {
	tmpl, err := template.New("slivka-github-action").Delims("[[", "]]").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse slivka github-action template: %w", err)
	}

	data := SlivkaGithubActionData{
		PluginID: definition.Plugin.ID,
		Version:  definition.Plugin.Version,
	}

	if definition.Example != nil && definition.Example.Enabled {
		data.HasExample = true
		exampleDirsMap := make(map[string]bool)
		var args []string

		for key, value := range definition.Example.Values {
			strVal := fmt.Sprintf("%v", value)

			for _, input := range definition.Inputs {
				if input.Name == key {
					if input.Type == models.PluginInputTypeFile {
						if strings.Contains(strVal, "/") {
							dir := strings.Split(strVal, "/")[0]
							exampleDirsMap[dir] = true
						}
						args = append(args, fmt.Sprintf("-F \"%s=@testdata/%s\"", key, filepath.Base(strVal)))
					} else {
						args = append(args, fmt.Sprintf("-F \"%s=%v\"", key, value))
					}
					break
				}
			}
		}

		data.SlivkaTestArgs = strings.Join(args, " \\\n            ")
		for dir := range exampleDirsMap {
			data.ExampleDirs = append(data.ExampleDirs, dir)
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute slivka github-action template: %w", err)
	}

	return buf.String(), nil
}
