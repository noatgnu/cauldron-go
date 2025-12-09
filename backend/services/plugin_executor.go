package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/noatgnu/cauldron-go/backend/models"
)

type PluginExecutor struct{}

func NewPluginExecutor() *PluginExecutor {
	return &PluginExecutor{}
}

func (e *PluginExecutor) BuildArguments(plugin *models.PluginV2, parameters map[string]interface{}) ([]string, error) {
	args := []string{plugin.ScriptPath}

	for inputName, mappingInterface := range plugin.Definition.Execution.ArgsMapping {
		paramValue, hasValue := parameters[inputName]

		mapping, err := e.parseArgMapping(mappingInterface)
		if err != nil {
			return nil, fmt.Errorf("invalid argument mapping for %s: %w", inputName, err)
		}

		if mapping.Flag == nil {
			if inputName == "outputDir" || strings.Contains(strings.ToLower(inputName), "output") {
				continue
			}
			return nil, fmt.Errorf("missing flag for input: %s", inputName)
		}

		if !hasValue {
			continue
		}

		if mapping.When != nil {
			shouldInclude := e.evaluateCondition(paramValue, *mapping.When)
			if !shouldInclude {
				continue
			}
		}

		flag := *mapping.Flag
		value := mapping.Value

		if value == nil {
			transformedValue, err := e.transformValue(paramValue, mapping.Transform)
			if err != nil {
				return nil, fmt.Errorf("failed to transform value for %s: %w", inputName, err)
			}
			valueStr := fmt.Sprintf("%v", transformedValue)
			value = &valueStr
		}

		if *value == "true" && mapping.When != nil {
			args = append(args, flag)
		} else if *value != "" {
			args = append(args, flag, *value)
		}
	}

	return args, nil
}

func (e *PluginExecutor) parseArgMapping(mappingInterface interface{}) (*models.ArgMapping, error) {
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
		return e.parseArgMapping(converted)

	default:
		return nil, fmt.Errorf("unsupported mapping type: %T", mappingInterface)
	}
}

func (e *PluginExecutor) transformValue(value interface{}, transform *models.InputTransform) (interface{}, error) {
	if transform == nil {
		return value, nil
	}

	switch *transform {
	case models.TransformCommaJoin:
		if arr, ok := value.([]interface{}); ok {
			strs := make([]string, len(arr))
			for i, v := range arr {
				strs[i] = fmt.Sprintf("%v", v)
			}
			return strings.Join(strs, ","), nil
		}
		return value, nil

	case models.TransformSpaceJoin:
		if arr, ok := value.([]interface{}); ok {
			strs := make([]string, len(arr))
			for i, v := range arr {
				strs[i] = fmt.Sprintf("%v", v)
			}
			return strings.Join(strs, " "), nil
		}
		return value, nil

	case models.TransformJSONEncode:
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		return string(jsonBytes), nil

	default:
		return value, nil
	}
}

func (e *PluginExecutor) evaluateCondition(value interface{}, condition string) bool {
	valueStr := fmt.Sprintf("%v", value)

	switch condition {
	case "true":
		return valueStr == "true" || valueStr == "1"
	case "false":
		return valueStr == "false" || valueStr == "0"
	case "not-empty":
		return valueStr != ""
	case "empty":
		return valueStr == ""
	default:
		return valueStr == condition
	}
}

func (e *PluginExecutor) ValidateParameters(plugin *models.PluginV2, parameters map[string]interface{}) error {
	for _, input := range plugin.Definition.Inputs {
		value, hasValue := parameters[input.Name]

		if input.Required && !hasValue {
			return fmt.Errorf("required parameter missing: %s", input.Name)
		}

		if !hasValue && input.Default != nil {
			parameters[input.Name] = input.Default
			continue
		}

		if !hasValue {
			continue
		}

		if err := e.validateInputType(input, value); err != nil {
			return fmt.Errorf("invalid value for %s: %w", input.Name, err)
		}

		if err := e.validateInputRange(input, value); err != nil {
			return fmt.Errorf("value out of range for %s: %w", input.Name, err)
		}
	}

	return nil
}

func (e *PluginExecutor) validateInputType(input models.PluginInputV2, value interface{}) error {
	switch input.Type {
	case models.PluginInputTypeNumber:
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("expected number, got %T", value)
		}

	case models.PluginInputTypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}

	case models.PluginInputTypeMultiSelect, models.PluginInputTypeColumnSelector:
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("expected array, got %T", value)
		}

	case models.PluginInputTypeSelect:
		if len(input.Options) > 0 {
			valueStr := fmt.Sprintf("%v", value)
			valid := false
			for _, option := range input.Options {
				if option == valueStr {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("value must be one of: %v", input.Options)
			}
		}
	}

	return nil
}

func (e *PluginExecutor) validateInputRange(input models.PluginInputV2, value interface{}) error {
	if input.Type != models.PluginInputTypeNumber {
		return nil
	}

	numValue, ok := value.(float64)
	if !ok {
		return nil
	}

	if input.Min != nil && numValue < *input.Min {
		return fmt.Errorf("value %.2f is less than minimum %.2f", numValue, *input.Min)
	}

	if input.Max != nil && numValue > *input.Max {
		return fmt.Errorf("value %.2f is greater than maximum %.2f", numValue, *input.Max)
	}

	return nil
}
