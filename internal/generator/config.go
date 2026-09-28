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

// FormatDefaultValue renders a plugin input's default value as a Nextflow
// config literal (quoted string, bare bool/number, or "null" when unset).
// Shared by GenerateConfig and by recipe-level config assembly so both stay
// in sync on how defaults are formatted.
func FormatDefaultValue(input models.PluginInputV2) string {
	if input.Default == nil {
		return "null"
	}
	switch v := input.Default.(type) {
	case string:
		return fmt.Sprintf("'%s'", v)
	case bool, float64, int:
		return fmt.Sprintf("%v", v)
	default:
		return "null"
	}
}

func GenerateConfig(definition *models.PluginDefinition, tmplStr string) (string, error) {
	tmpl, err := template.New("config").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse config template: %w", err)
	}

	data := ConfigData{}
	data.Params = append(data.Params, ConfigParam{Name: "outdir", Default: "'./results'"})

	for _, input := range definition.Inputs {
		data.Params = append(data.Params, ConfigParam{
			Name:    input.Name,
			Default: FormatDefaultValue(input),
		})
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute config template: %w", err)
	}

	return buf.String(), nil
}

// GenerateRecipeConfig renders a shared nextflow.config for a multi-stage
// Recipe pipeline from an already-namespaced param list (see
// FormatDefaultValue and the "stage<index>_<inputName>" naming convention
// used by RecipeService.ExportRecipeNextflow).
func GenerateRecipeConfig(params []ConfigParam, tmplStr string) (string, error) {
	tmpl, err := template.New("recipe-config").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse recipe config template: %w", err)
	}

	data := ConfigData{Params: append([]ConfigParam{{Name: "outdir", Default: "'./results'"}}, params...)}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute recipe config template: %w", err)
	}

	return buf.String(), nil
}
