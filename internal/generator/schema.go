package generator

import (
	"encoding/json"
	"fmt"

	"github.com/noatgnu/cauldron-go/backend/models"
)

type NextflowSchema struct {
	Schema      string                 `json:"$schema"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Definitions map[string]SchemaGroup `json:"definitions"`
	Properties  map[string]interface{} `json:"properties"`
}

type SchemaGroup struct {
	Title       string                    `json:"title"`
	Description string                    `json:"description"`
	Type        string                    `json:"type"`
	Properties  map[string]SchemaProperty `json:"properties"`
	Required    []string                  `json:"required,omitempty"`
}

type SchemaProperty struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Format      string      `json:"format,omitempty"`
}

func GenerateSchema(definition *models.PluginDefinition) (string, error) {
	schema := NextflowSchema{
		Schema:      "http://json-schema.org/draft-07/schema",
		Title:       fmt.Sprintf("%s pipeline parameters", definition.Plugin.Name),
		Description: definition.Plugin.Description,
		Type:        "object",
		Definitions: make(map[string]SchemaGroup),
		Properties:  make(map[string]interface{}),
	}

	pluginParamsGroup := SchemaGroup{
		Title:       "Plugin parameters",
		Description: fmt.Sprintf("Parameters for the %s plugin.", definition.Plugin.Name),
		Type:        "object",
		Properties:  make(map[string]SchemaProperty),
	}

	for _, input := range definition.Inputs {
		propType := "string"
		format := ""
		switch input.Type {
		case models.PluginInputTypeNumber:
			propType = "number"
		case models.PluginInputTypeBoolean:
			propType = "boolean"
		case models.PluginInputTypeFile:
			format = "file-path"
		}

		pluginParamsGroup.Properties[input.Name] = SchemaProperty{
			Type:        propType,
			Description: input.Description,
			Default:     input.Default,
			Format:      format,
		}
	}

	schema.Definitions["plugin_parameters"] = pluginParamsGroup

	for name := range pluginParamsGroup.Properties {
		schema.Properties[name] = map[string]string{"$ref": fmt.Sprintf("#/definitions/plugin_parameters/properties/%s", name)}
	}

	bytes, err := json.MarshalIndent(schema, "", "    ")
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
