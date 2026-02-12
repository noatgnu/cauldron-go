package generator

import (
	"encoding/json"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func GenerateSamplesheetSchema(definition *models.PluginDefinition) (string, error) {
	schema := map[string]interface{}{
		"items": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"sample": map[string]interface{}{
					"type":        "string",
					"description": "Unique sample identifier",
				},
				"input_file": map[string]interface{}{
					"type":        "string",
					"format":      "file-path",
					"description": "Path to the primary input file for this sample",
				},
				"condition": map[string]interface{}{
					"type":        "string",
					"description": "Experimental condition/group",
				},
				"bioreplicate": map[string]interface{}{
					"type":        "string",
					"description": "Biological replicate identifier",
				},
				"batch": map[string]interface{}{
					"type":        "string",
					"description": "Batch identifier for batch correction",
				},
			},
			"required": []string{"sample", "input_file"},
		},
		"type": "array",
	}

	bytes, err := json.MarshalIndent(schema, "", "    ")
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
