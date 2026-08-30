package generator

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/noatgnu/cauldron-go/backend/models"
)

type ConfigData struct {
	Params []ConfigParam
}

type ConfigParam struct {
	Name    string
	Default interface{}
}

func GenerateConfig(definition *models.PluginDefinition, tmplStr string) (string, error) {
	tmpl, err := template.New("config").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse config template: %w", err)
	}

	data := ConfigData{}
	data.Params = append(data.Params, ConfigParam{Name: "outdir", Default: "'./results'"})

	for _, input := range definition.Inputs {
		defaultValue := "null"
		if input.Default != nil {
			switch v := input.Default.(type) {
			case string:
				defaultValue = fmt.Sprintf("'%s'", v)
			case bool, float64, int:
				defaultValue = fmt.Sprintf("%v", v)
			}
		}
		data.Params = append(data.Params, ConfigParam{
			Name:    input.Name,
			Default: defaultValue,
		})
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute config template: %w", err)
	}

	return buf.String(), nil
}
