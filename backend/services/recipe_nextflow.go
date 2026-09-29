package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/internal/generator"
	"github.com/noatgnu/cauldron-go/internal/templates"
)

// ExportRecipeNextflow generates a Nextflow pipeline for a saved Recipe: one
// process module per distinct plugin (shared across stages that reuse the
// same plugin via an aliased include), a single workflow wiring stage-to-stage
// bindings into real Nextflow channels, and a namespaced nextflow.config.
// Requires every stage's plugin to currently be installed, the same rule
// InstantiateChain already enforces before running a recipe.
func (r *RecipeService) ExportRecipeNextflow(recipeID, outputDir string) error {
	recipe, err := r.GetRecipe(recipeID)
	if err != nil {
		return err
	}

	report, err := r.CheckCompatibility(recipeID)
	if err != nil {
		return err
	}
	if !report.AllOK {
		return fmt.Errorf("recipe is not fully compatible with installed plugins; run CheckCompatibility for details")
	}

	stageRows, err := r.GetRecipeStages(recipeID)
	if err != nil {
		return err
	}
	if len(stageRows) == 0 {
		return fmt.Errorf("recipe has no stages")
	}

	diagStages := diagramStagesFromRecipeStages(stageRows, map[int]CompatibilityStatus{})

	procTmpl, err := templates.GetTemplate("process.nf.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load process template: %w", err)
	}
	dockerTmpl, err := templates.GetTemplate("Dockerfile.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load Dockerfile template: %w", err)
	}
	workflowTmpl, err := templates.GetTemplate("recipe-main.nf.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load recipe workflow template: %w", err)
	}
	configTmpl, err := templates.GetTemplate("nextflow.config.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load config template: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	callAliasByStage := make(map[int]string, len(diagStages))
	writtenModules := map[string]bool{}
	var workflowStages []generator.RecipeWorkflowStage
	var configParams []generator.ConfigParam

	for _, s := range diagStages {
		installed, err := r.pluginLoader.GetPluginByStringID(s.PluginID)
		if err != nil {
			return fmt.Errorf("stage %d: plugin %q is no longer installed", s.Index, s.PluginID)
		}

		if !writtenModules[s.PluginID] {
			if err := writeNextflowModule(installed, outputDir, procTmpl, dockerTmpl); err != nil {
				return fmt.Errorf("stage %d (%s): %w", s.Index, s.PluginID, err)
			}
			writtenModules[s.PluginID] = true
		}

		processName := generator.ToNextflowID(s.PluginID)
		callAlias := fmt.Sprintf("%s_S%d", processName, s.Index)
		callAliasByStage[s.Index] = callAlias

		boundInputs := make(map[string]recipeBinding, len(s.Bindings))
		for _, b := range s.Bindings {
			boundInputs[b.InputName] = b
		}

		stage := generator.RecipeWorkflowStage{
			ProcessName: processName,
			CallAlias:   callAlias,
			ModulePath:  s.PluginID,
		}

		for _, input := range installed.Definition.Inputs {
			isFile := input.Type == models.PluginInputTypeFile

			if binding, bound := boundInputs[input.Name]; bound {
				sourceAlias, ok := callAliasByStage[binding.SourceStage]
				if !ok {
					return fmt.Errorf("stage %d: binding %q references stage %d, which must come earlier in the recipe", s.Index, input.Name, binding.SourceStage)
				}
				stage.Args = append(stage.Args, generator.RecipeArgSource{
					Name:            input.Name,
					IsFile:          isFile,
					FromParam:       false,
					SourceCallAlias: sourceAlias,
					SourceEmit:      generator.ToNextflowID(binding.SourceOutput),
				})
				continue
			}

			paramName := fmt.Sprintf("stage%d_%s", s.Index, input.Name)
			stage.Args = append(stage.Args, generator.RecipeArgSource{
				Name:      input.Name,
				IsFile:    isFile,
				FromParam: true,
				ParamName: paramName,
			})
			configParams = append(configParams, generator.ConfigParam{
				Name:    paramName,
				Default: generator.FormatDefaultValue(input),
			})
		}

		workflowStages = append(workflowStages, stage)
	}

	workflowContent, err := generator.GenerateRecipeWorkflow(workflowStages, workflowTmpl)
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "main.nf"), []byte(workflowContent), 0644); err != nil {
		return fmt.Errorf("failed to write main.nf: %w", err)
	}

	configContent, err := generator.GenerateRecipeConfig(configParams, configTmpl)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "nextflow.config"), []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write nextflow.config: %w", err)
	}

	if err := writeRecipeNextflowReadme(recipe, diagStages, outputDir); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	return nil
}

func writeNextflowModule(installed *models.PluginV2, outputDir string, procTmpl, dockerTmpl string) error {
	moduleDir := filepath.Join(outputDir, "modules", "local", installed.Definition.Plugin.ID)
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return err
	}

	procContent, err := generator.GenerateProcess(&installed.Definition, procTmpl)
	if err != nil {
		return fmt.Errorf("failed to generate process: %w", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "main.nf"), []byte(procContent), 0644); err != nil {
		return err
	}

	containerDir := filepath.Join(outputDir, "containers", installed.Definition.Plugin.ID)
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		return fmt.Errorf("failed to create containers directory: %w", err)
	}

	if installed.Definition.Runtime.IsDockerRuntime() && installed.Definition.Runtime.Docker != nil && installed.Definition.Runtime.Docker.Dockerfile != "" {
		customPath := filepath.Join(installed.FolderPath, installed.Definition.Runtime.Docker.Dockerfile)
		if content, err := os.ReadFile(customPath); err == nil {
			return os.WriteFile(filepath.Join(containerDir, "Dockerfile"), content, 0644)
		}
	}

	dockerContent, err := generator.GenerateContainer(&installed.Definition, dockerTmpl)
	if err != nil || dockerContent == "" {
		return nil
	}
	return os.WriteFile(filepath.Join(containerDir, "Dockerfile"), []byte(dockerContent), 0644)
}

func writeRecipeNextflowReadme(recipe *models.Recipe, stages []diagramStage, outputDir string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", recipe.Label)
	if recipe.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", recipe.Description)
	}
	b.WriteString("Nextflow pipeline generated from a CauldronGO recipe.\n\n## Stages\n\n")
	for _, s := range stages {
		fmt.Fprintf(&b, "%d. `%s`", s.Index+1, s.PluginID)
		if s.PluginVersion != "" {
			fmt.Fprintf(&b, " v%s", s.PluginVersion)
		}
		b.WriteString("\n")
	}
	return os.WriteFile(filepath.Join(outputDir, "README.md"), []byte(b.String()), 0644)
}
