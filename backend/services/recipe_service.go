package services

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/noatgnu/cauldron-go/backend/models"
)

const RecipeFormatVersion = 1

type RecipeStageSpec struct {
	PluginID      string
	PluginVersion string
	Params        map[string]interface{}
	Bindings      map[string]models.RecipeStageBinding

	Repository   string
	CommitHash   string
	Requirements *models.Requirements
}

type RecipeData struct {
	Version     int               `json:"version"`
	Label       string            `json:"label"`
	Description string            `json:"description,omitempty"`
	Stages      []RecipeStageData `json:"stages"`
}

type RecipeStageData struct {
	PluginID      string                               `json:"pluginId"`
	PluginVersion string                               `json:"pluginVersion,omitempty"`
	Params        map[string]interface{}               `json:"params"`
	Bindings      map[string]models.RecipeStageBinding `json:"bindings"`

	Repository   string               `json:"repository,omitempty"`
	CommitHash   string               `json:"commitHash,omitempty"`
	Requirements *models.Requirements `json:"requirements,omitempty"`
}

type CompatibilityStatus string

const (
	CompatibilityCompatible               CompatibilityStatus = "compatible"
	CompatibilityCompatibleVersionDiffers CompatibilityStatus = "compatible_version_differs"
	CompatibilityIncompatible             CompatibilityStatus = "incompatible"
	CompatibilityMissing                  CompatibilityStatus = "missing"
)

type StageCompatibility struct {
	StageIndex       int                 `json:"stageIndex"`
	PluginID         string              `json:"pluginId"`
	RecordedVersion  string              `json:"recordedVersion,omitempty"`
	InstalledVersion string              `json:"installedVersion,omitempty"`
	Status           CompatibilityStatus `json:"status"`
	MissingInputs    []string            `json:"missingInputs,omitempty"`
	MissingOutputs   []string            `json:"missingOutputs,omitempty"`

	Repository string `json:"repository,omitempty"`
	CommitHash string `json:"commitHash,omitempty"`
}

type CompatibilityReport struct {
	RecipeID string                `json:"recipeId"`
	Stages   []*StageCompatibility `json:"stages"`
	AllOK    bool                  `json:"allOk"`
}

type RecipeImportResult struct {
	Recipe        *models.Recipe       `json:"recipe"`
	Compatibility *CompatibilityReport `json:"compatibility"`
}

type RecipeService struct {
	db           *DatabaseService
	pluginLoader *PluginLoaderV2
	chainService *ChainService
}

func NewRecipeService(db *DatabaseService, pluginLoader *PluginLoaderV2, chainService *ChainService) *RecipeService {
	return &RecipeService{db: db, pluginLoader: pluginLoader, chainService: chainService}
}

func (r *RecipeService) SaveRecipe(label, description string, stages []RecipeStageSpec) (*models.Recipe, error) {
	if len(stages) == 0 {
		return nil, fmt.Errorf("a recipe needs at least one stage")
	}
	now := time.Now()
	recipe := &models.Recipe{
		ID:          uuid.New().String(),
		Label:       label,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := r.db.GetDB().Create(recipe).Error; err != nil {
		return nil, err
	}
	if err := r.saveStages(recipe.ID, stages); err != nil {
		return nil, err
	}
	return recipe, nil
}

func (r *RecipeService) saveStages(recipeID string, stages []RecipeStageSpec) error {
	for i, spec := range stages {
		bindings := models.JSONMap{}
		for inputName, b := range spec.Bindings {
			bindings[inputName] = map[string]interface{}{"stage": b.Stage, "output": b.Output}
		}
		requirements, err := requirementsToJSONMap(spec.Requirements)
		if err != nil {
			return fmt.Errorf("failed to encode requirements for stage %d: %w", i, err)
		}
		stageRow := &models.RecipeStage{
			RecipeID:      recipeID,
			StageIndex:    i,
			PluginID:      spec.PluginID,
			PluginVersion: spec.PluginVersion,
			Params:        spec.Params,
			Bindings:      bindings,
			Repository:    spec.Repository,
			CommitHash:    spec.CommitHash,
			Requirements:  requirements,
		}
		if err := r.db.GetDB().Create(stageRow).Error; err != nil {
			return fmt.Errorf("failed to create stage %d: %w", i, err)
		}
	}
	return nil
}

func requirementsToJSONMap(reqs *models.Requirements) (models.JSONMap, error) {
	if reqs == nil {
		return nil, nil
	}
	raw, err := json.Marshal(reqs)
	if err != nil {
		return nil, err
	}
	var m models.JSONMap
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func requirementsEmpty(reqs *models.Requirements) bool {
	if reqs == nil {
		return true
	}
	return reqs.Python == "" && reqs.R == "" && len(reqs.Packages) == 0 &&
		reqs.PythonRequirementsFile == "" && reqs.RPackagesFile == ""
}

func jsonMapToRequirements(m models.JSONMap) (*models.Requirements, error) {
	if len(m) == 0 {
		return nil, nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var reqs models.Requirements
	if err := json.Unmarshal(raw, &reqs); err != nil {
		return nil, err
	}
	return &reqs, nil
}

func (r *RecipeService) UpdateRecipe(id, label, description string, stages []RecipeStageSpec) (*models.Recipe, error) {
	if len(stages) == 0 {
		return nil, fmt.Errorf("a recipe needs at least one stage")
	}
	var recipe models.Recipe
	if err := r.db.GetDB().First(&recipe, "id = ?", id).Error; err != nil {
		return nil, err
	}
	recipe.Label = label
	recipe.Description = description
	recipe.UpdatedAt = time.Now()
	if err := r.db.GetDB().Save(&recipe).Error; err != nil {
		return nil, err
	}
	if err := r.db.GetDB().Delete(&models.RecipeStage{}, "recipe_id = ?", id).Error; err != nil {
		return nil, err
	}
	if err := r.saveStages(id, stages); err != nil {
		return nil, err
	}
	return &recipe, nil
}

func (r *RecipeService) GetRecipe(id string) (*models.Recipe, error) {
	var recipe models.Recipe
	if err := r.db.GetDB().First(&recipe, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &recipe, nil
}

func (r *RecipeService) GetRecipeStages(id string) ([]models.RecipeStage, error) {
	var stages []models.RecipeStage
	err := r.db.GetDB().Where("recipe_id = ?", id).Order("stage_index ASC").Find(&stages).Error
	return stages, err
}

func (r *RecipeService) GetAllRecipes(limit, offset int) ([]*models.Recipe, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var recipes []*models.Recipe
	err := r.db.GetDB().Order("updated_at DESC").Limit(limit).Offset(offset).Find(&recipes).Error
	return recipes, err
}

func (r *RecipeService) DeleteRecipe(id string) error {
	if err := r.db.GetDB().Delete(&models.RecipeStage{}, "recipe_id = ?", id).Error; err != nil {
		return err
	}
	return r.db.GetDB().Delete(&models.Recipe{}, "id = ?", id).Error
}

func (r *RecipeService) ExportRecipe(id, path string, includeInstallInfo bool) error {
	recipe, err := r.GetRecipe(id)
	if err != nil {
		return err
	}
	stages, err := r.GetRecipeStages(id)
	if err != nil {
		return err
	}

	data := &RecipeData{
		Version:     RecipeFormatVersion,
		Label:       recipe.Label,
		Description: recipe.Description,
	}
	for _, s := range stages {
		bindings := map[string]models.RecipeStageBinding{}
		for inputName, raw := range s.Bindings {
			bindingMap, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			stageIdx, _ := bindingMap["stage"].(float64)
			outputName, _ := bindingMap["output"].(string)
			bindings[inputName] = models.RecipeStageBinding{Stage: int(stageIdx), Output: outputName}
		}

		stageData := RecipeStageData{
			PluginID:      s.PluginID,
			PluginVersion: s.PluginVersion,
			Params:        s.Params,
			Bindings:      bindings,
		}

		if includeInstallInfo {
			repository, commitHash := s.Repository, s.CommitHash
			var reqs *models.Requirements
			if installed, err := r.pluginLoader.GetPluginByStringID(s.PluginID); err == nil {
				repository = installed.Repository
				commitHash = installed.CommitHash
				reqs = &installed.Definition.Execution.Requirements
			} else {
				reqs, err = jsonMapToRequirements(s.Requirements)
				if err != nil {
					return fmt.Errorf("failed to decode stored requirements for stage %d: %w", s.StageIndex, err)
				}
			}
			stageData.Repository = repository
			stageData.CommitHash = commitHash
			if !requirementsEmpty(reqs) {
				stageData.Requirements = reqs
			}
		}

		data.Stages = append(data.Stages, stageData)
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal recipe: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func (r *RecipeService) ImportRecipeFromFile(path string) (*RecipeImportResult, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	return r.ImportRecipeFromData(raw)
}

func (r *RecipeService) ImportRecipeFromData(raw []byte) (*RecipeImportResult, error) {
	var data RecipeData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("failed to parse recipe file: %w", err)
	}
	if data.Version != RecipeFormatVersion {
		return nil, fmt.Errorf("unsupported recipe format version %d (expected %d)", data.Version, RecipeFormatVersion)
	}
	if len(data.Stages) == 0 {
		return nil, fmt.Errorf("recipe file has no stages")
	}

	specs := make([]RecipeStageSpec, len(data.Stages))
	for i, s := range data.Stages {
		specs[i] = RecipeStageSpec{
			PluginID:      s.PluginID,
			PluginVersion: s.PluginVersion,
			Params:        s.Params,
			Bindings:      s.Bindings,
			Repository:    s.Repository,
			CommitHash:    s.CommitHash,
			Requirements:  s.Requirements,
		}
	}

	recipe, err := r.SaveRecipe(data.Label, data.Description, specs)
	if err != nil {
		return nil, err
	}

	report, err := r.CheckCompatibility(recipe.ID)
	if err != nil {
		return nil, err
	}

	return &RecipeImportResult{Recipe: recipe, Compatibility: report}, nil
}

func (r *RecipeService) CheckCompatibility(recipeID string) (*CompatibilityReport, error) {
	stages, err := r.GetRecipeStages(recipeID)
	if err != nil {
		return nil, err
	}

	report := &CompatibilityReport{RecipeID: recipeID, AllOK: true}

	for _, stage := range stages {
		result := &StageCompatibility{
			StageIndex:      stage.StageIndex,
			PluginID:        stage.PluginID,
			RecordedVersion: stage.PluginVersion,
		}

		installed, err := r.pluginLoader.GetPluginByStringID(stage.PluginID)
		if err != nil {
			result.Status = CompatibilityMissing
			result.Repository = stage.Repository
			result.CommitHash = stage.CommitHash
			report.AllOK = false
			report.Stages = append(report.Stages, result)
			continue
		}

		result.InstalledVersion = installed.Definition.Plugin.Version
		if installed.Definition.Plugin.Version == stage.PluginVersion {
			result.Status = CompatibilityCompatible
			report.Stages = append(report.Stages, result)
			continue
		}

		inputNames := map[string]bool{}
		for _, in := range installed.Definition.Inputs {
			inputNames[in.Name] = true
		}
		for paramName := range stage.Params {
			if !inputNames[paramName] {
				result.MissingInputs = append(result.MissingInputs, paramName)
			}
		}
		for inputName, raw := range stage.Bindings {
			if !inputNames[inputName] {
				result.MissingInputs = append(result.MissingInputs, inputName)
			}
			bindingMap, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			sourceIdx, _ := bindingMap["stage"].(float64)
			outputName, _ := bindingMap["output"].(string)
			var sourceStage *models.RecipeStage
			for i := range stages {
				if stages[i].StageIndex == int(sourceIdx) {
					sourceStage = &stages[i]
					break
				}
			}
			if sourceStage == nil {
				continue
			}
			sourcePlugin, err := r.pluginLoader.GetPluginByStringID(sourceStage.PluginID)
			if err != nil {
				continue
			}
			outputFound := false
			for _, out := range sourcePlugin.Definition.Outputs {
				if out.Name == outputName {
					outputFound = true
					break
				}
			}
			if !outputFound {
				result.MissingOutputs = append(result.MissingOutputs, outputName)
			}
		}

		if len(result.MissingInputs) > 0 || len(result.MissingOutputs) > 0 {
			result.Status = CompatibilityIncompatible
			report.AllOK = false
		} else {
			result.Status = CompatibilityCompatibleVersionDiffers
		}
		report.Stages = append(report.Stages, result)
	}

	return report, nil
}

func (r *RecipeService) InstantiateChain(recipeID string, paramOverrides map[int]map[string]interface{}) (*models.JobChain, error) {
	report, err := r.CheckCompatibility(recipeID)
	if err != nil {
		return nil, err
	}
	if !report.AllOK {
		return nil, fmt.Errorf("recipe is not compatible with installed plugins, see compatibility report")
	}

	recipe, err := r.GetRecipe(recipeID)
	if err != nil {
		return nil, err
	}
	stages, err := r.GetRecipeStages(recipeID)
	if err != nil {
		return nil, err
	}

	chainStages := make([]ChainStageSpec, len(stages))
	for i, stage := range stages {
		installed, err := r.pluginLoader.GetPluginByStringID(stage.PluginID)
		if err != nil {
			return nil, fmt.Errorf("plugin %q is no longer installed: %w", stage.PluginID, err)
		}

		params := make(map[string]interface{}, len(stage.Params))
		for k, v := range stage.Params {
			params[k] = v
		}
		if overrides, ok := paramOverrides[stage.StageIndex]; ok {
			for k, v := range overrides {
				params[k] = v
			}
		}

		bindings := map[string]models.RecipeStageBinding{}
		for inputName, raw := range stage.Bindings {
			bindingMap, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			sourceIdx, _ := bindingMap["stage"].(float64)
			outputName, _ := bindingMap["output"].(string)
			bindings[inputName] = models.RecipeStageBinding{Stage: int(sourceIdx), Output: outputName}
		}

		chainStages[i] = ChainStageSpec{
			PluginID:      installed.ID,
			PluginVersion: installed.Definition.Plugin.Version,
			Params:        params,
			Bindings:      bindings,
		}
	}

	return r.chainService.CreateChain(recipe.Label, recipe.ID, chainStages)
}

// CreateRecipeFromChain reconstructs a Recipe from an already-created
// JobChain's stage rows, so a specific run can be recovered as a reusable
// recipe even if the chain's original recipe was since edited or deleted
// (JobChain rows outlive the Recipe they were instantiated from). Every
// stage's plugin must still be installed, since JobChainStage only stores
// the plugin's internal numeric ID, which only PluginLoaderV2 can resolve
// back to the string plugin ID a Recipe needs.
func (r *RecipeService) CreateRecipeFromChain(chainID, label, description string) (*models.Recipe, error) {
	var chain models.JobChain
	if err := r.db.GetDB().First(&chain, "id = ?", chainID).Error; err != nil {
		return nil, fmt.Errorf("chain not found: %w", err)
	}

	var stageRows []models.JobChainStage
	if err := r.db.GetDB().Where("chain_id = ?", chainID).Order("stage_index ASC").Find(&stageRows).Error; err != nil {
		return nil, err
	}
	if len(stageRows) == 0 {
		return nil, fmt.Errorf("chain %s has no stages", chainID)
	}

	specs := make([]RecipeStageSpec, len(stageRows))
	for i, row := range stageRows {
		plugin, err := r.pluginLoader.GetPlugin(row.PluginID)
		if err != nil {
			return nil, fmt.Errorf("cannot reconstruct recipe: the plugin used by stage %d is no longer installed", row.StageIndex)
		}

		bindings := map[string]models.RecipeStageBinding{}
		for inputName, raw := range row.Bindings {
			bindingMap, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			stageIdx, _ := bindingMap["stage"].(float64)
			outputName, _ := bindingMap["output"].(string)
			bindings[inputName] = models.RecipeStageBinding{Stage: int(stageIdx), Output: outputName}
		}

		params := make(map[string]interface{}, len(row.Params))
		for k, v := range row.Params {
			params[k] = v
		}

		specs[i] = RecipeStageSpec{
			PluginID:      plugin.Definition.Plugin.ID,
			PluginVersion: row.PluginVersion,
			Params:        params,
			Bindings:      bindings,
			Repository:    plugin.Repository,
			CommitHash:    plugin.CommitHash,
		}
	}

	if label == "" {
		label = chain.Label
	}

	return r.SaveRecipe(label, description, specs)
}
