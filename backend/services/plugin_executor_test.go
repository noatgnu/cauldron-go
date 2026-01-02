package services

import (
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func TestBuildArgumentsOrderDeterministic(t *testing.T) {
	executor := NewPluginExecutor()

	plugin := &models.PluginV2{
		ScriptPath: "/path/to/script.py",
		Definition: models.PluginDefinition{
			Inputs: []models.PluginInputV2{
				{Name: "input_file", Type: models.PluginInputTypeFile},
				{Name: "columns_name", Type: models.PluginInputTypeColumnSelector},
				{Name: "n_components", Type: models.PluginInputTypeNumber},
				{Name: "log2", Type: models.PluginInputTypeBoolean},
			},
			Execution: models.PluginExecution{
				ArgsMapping: map[string]interface{}{
					"input_file": "--input_file",
					"columns_name": map[string]interface{}{
						"flag":      "--columns_name",
						"transform": "comma-join",
					},
					"n_components": "--n_components",
					"log2": map[string]interface{}{
						"flag":  "--log2",
						"when":  "true",
						"value": "true",
					},
				},
				OutputDir: "--output_folder",
			},
		},
	}

	parameters := map[string]interface{}{
		"input_file":   "/path/to/data.csv",
		"columns_name": []interface{}{"col1", "col2", "col3"},
		"n_components": float64(2),
		"log2":         true,
	}

	args1, err := executor.BuildArguments(plugin, parameters)
	if err != nil {
		t.Fatalf("BuildArguments failed: %v", err)
	}

	args2, err := executor.BuildArguments(plugin, parameters)
	if err != nil {
		t.Fatalf("BuildArguments failed on second call: %v", err)
	}

	if len(args1) != len(args2) {
		t.Fatalf("Argument lengths differ between calls: %d vs %d", len(args1), len(args2))
	}

	for i := range args1 {
		if args1[i] != args2[i] {
			t.Errorf("Argument mismatch at position %d: '%s' vs '%s'", i, args1[i], args2[i])
		}
	}

	expectedOrder := []string{
		"/path/to/script.py",
		"--input_file",
		"/path/to/data.csv",
		"--columns_name",
		"col1,col2,col3",
		"--n_components",
		"2",
		"--log2",
	}

	if len(args1) != len(expectedOrder) {
		t.Fatalf("Expected %d arguments, got %d: %v", len(expectedOrder), len(args1), args1)
	}

	for i, expected := range expectedOrder {
		if args1[i] != expected {
			t.Errorf("Position %d: expected '%s', got '%s'", i, expected, args1[i])
		}
	}

	t.Log("✓ Arguments are generated in deterministic order based on inputs array")
}

func TestBuildArgumentsBooleanFlagWithoutWhen(t *testing.T) {
	executor := NewPluginExecutor()

	plugin := &models.PluginV2{
		ScriptPath: "/path/to/script.py",
		Definition: models.PluginDefinition{
			Inputs: []models.PluginInputV2{
				{Name: "verbose", Type: models.PluginInputTypeBoolean},
			},
			Execution: models.PluginExecution{
				ArgsMapping: map[string]interface{}{
					"verbose": "--verbose",
				},
			},
		},
	}

	parameters := map[string]interface{}{
		"verbose": true,
	}

	args, err := executor.BuildArguments(plugin, parameters)
	if err != nil {
		t.Fatalf("BuildArguments failed: %v", err)
	}

	expectedArgs := []string{
		"/path/to/script.py",
		"--verbose",
	}

	if len(args) != len(expectedArgs) {
		t.Fatalf("Expected %d arguments, got %d: %v", len(expectedArgs), len(args), args)
	}

	for i, expected := range expectedArgs {
		if args[i] != expected {
			t.Errorf("Position %d: expected '%s', got '%s'", i, expected, args[i])
		}
	}

	t.Log("✓ Boolean flag without 'when' condition includes the value")
}

func TestBuildArgumentsSkipsUnprovidedParameters(t *testing.T) {
	executor := NewPluginExecutor()

	plugin := &models.PluginV2{
		ScriptPath: "/path/to/script.py",
		Definition: models.PluginDefinition{
			Inputs: []models.PluginInputV2{
				{Name: "input_file", Type: models.PluginInputTypeFile},
				{Name: "optional_param", Type: models.PluginInputTypeText},
				{Name: "another_param", Type: models.PluginInputTypeNumber},
			},
			Execution: models.PluginExecution{
				ArgsMapping: map[string]interface{}{
					"input_file":     "--input",
					"optional_param": "--optional",
					"another_param":  "--another",
				},
			},
		},
	}

	parameters := map[string]interface{}{
		"input_file":    "/path/to/file.csv",
		"another_param": float64(42),
	}

	args, err := executor.BuildArguments(plugin, parameters)
	if err != nil {
		t.Fatalf("BuildArguments failed: %v", err)
	}

	expectedArgs := []string{
		"/path/to/script.py",
		"--input",
		"/path/to/file.csv",
		"--another",
		"42",
	}

	if len(args) != len(expectedArgs) {
		t.Fatalf("Expected %d arguments, got %d: %v", len(expectedArgs), len(args), args)
	}

	for i, expected := range expectedArgs {
		if args[i] != expected {
			t.Errorf("Position %d: expected '%s', got '%s'", i, expected, args[i])
		}
	}

	t.Log("✓ Unprovided parameters are correctly skipped")
}
