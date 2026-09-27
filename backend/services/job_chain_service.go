package services

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/noatgnu/cauldron-go/backend/models"
)

type ChainStageSpec struct {
	PluginID      uint
	PluginVersion string
	Params        map[string]interface{}
	Bindings      map[string]models.RecipeStageBinding
}

type ChainStageStatus struct {
	StageIndex int         `json:"stageIndex"`
	PluginID   uint        `json:"pluginId"`
	Status     string      `json:"status"`
	Job        *models.Job `json:"job,omitempty"`
}

type ChainStatus struct {
	ChainID   string              `json:"chainId"`
	RecipeID  string              `json:"recipeId,omitempty"`
	Label     string              `json:"label"`
	CreatedAt time.Time           `json:"createdAt"`
	Status    string              `json:"status"`
	Stages    []*ChainStageStatus `json:"stages"`
}

type ChainService struct {
	db             *DatabaseService
	jobQueue       *JobQueueService
	pluginLoader   *PluginLoaderV2
	pluginExecutor *PluginExecutor
	settings       *SettingsService
}

func NewChainService(db *DatabaseService, jobQueue *JobQueueService, pluginLoader *PluginLoaderV2, pluginExecutor *PluginExecutor, settings *SettingsService) *ChainService {
	return &ChainService{
		db:             db,
		jobQueue:       jobQueue,
		pluginLoader:   pluginLoader,
		pluginExecutor: pluginExecutor,
		settings:       settings,
	}
}

func (c *ChainService) CreateChain(label, recipeID string, stages []ChainStageSpec) (*models.JobChain, error) {
	if len(stages) == 0 {
		return nil, fmt.Errorf("no stages provided for chain")
	}

	chain := &models.JobChain{
		ID:        uuid.New().String(),
		RecipeID:  recipeID,
		Label:     label,
		CreatedAt: time.Now(),
	}
	if err := c.db.GetDB().Create(chain).Error; err != nil {
		return nil, err
	}

	for i, spec := range stages {
		bindings := models.JSONMap{}
		for inputName, b := range spec.Bindings {
			bindings[inputName] = map[string]interface{}{"stage": b.Stage, "output": b.Output}
		}
		stageRow := &models.JobChainStage{
			ChainID:       chain.ID,
			StageIndex:    i,
			PluginID:      spec.PluginID,
			PluginVersion: spec.PluginVersion,
			Params:        spec.Params,
			Bindings:      bindings,
		}
		if err := c.db.GetDB().Create(stageRow).Error; err != nil {
			return nil, fmt.Errorf("failed to create stage %d: %w", i, err)
		}
	}

	if err := c.advanceStage(chain.ID, 0); err != nil {
		return nil, fmt.Errorf("failed to start first stage: %w", err)
	}

	return chain, nil
}

func (c *ChainService) advanceStage(chainID string, stageIndex int) error {
	var stage models.JobChainStage
	if err := c.db.GetDB().Where("chain_id = ? AND stage_index = ?", chainID, stageIndex).First(&stage).Error; err != nil {
		return err
	}

	plugin, err := c.pluginLoader.GetPlugin(stage.PluginID)
	if err != nil {
		return fmt.Errorf("failed to load plugin for stage %d: %w", stageIndex, err)
	}

	params := make(map[string]interface{}, len(stage.Params))
	for k, v := range stage.Params {
		params[k] = v
	}

	for inputName, raw := range stage.Bindings {
		bindingMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		sourceIndexFloat, _ := bindingMap["stage"].(float64)
		outputName, _ := bindingMap["output"].(string)
		sourceIndex := int(sourceIndexFloat)

		var sourceStage models.JobChainStage
		if err := c.db.GetDB().Where("chain_id = ? AND stage_index = ?", chainID, sourceIndex).First(&sourceStage).Error; err != nil {
			return fmt.Errorf("failed to load source stage %d for binding %q: %w", sourceIndex, inputName, err)
		}
		if sourceStage.JobID == "" {
			return fmt.Errorf("source stage %d for binding %q has not run yet", sourceIndex, inputName)
		}

		sourceJob, err := c.jobQueue.GetJob(sourceStage.JobID)
		if err != nil {
			return fmt.Errorf("failed to load source job for binding %q: %w", inputName, err)
		}

		sourcePlugin, err := c.pluginLoader.GetPlugin(sourceStage.PluginID)
		if err != nil {
			return fmt.Errorf("failed to load source plugin for binding %q: %w", inputName, err)
		}

		outputPath := ""
		found := false
		for _, out := range sourcePlugin.Definition.Outputs {
			if out.Name == outputName {
				outputPath = out.Path
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("source plugin %s has no declared output named %q", sourcePlugin.Definition.Plugin.ID, outputName)
		}

		params[inputName] = filepath.Join(sourceJob.OutputPath, outputPath)
	}

	jobID, err := ExecutePluginJob(c.pluginExecutor, c.jobQueue, c.settings, plugin, params, "", chainID, stageIndex)
	if err != nil {
		return fmt.Errorf("failed to create job for stage %d: %w", stageIndex, err)
	}

	stage.JobID = jobID
	return c.db.GetDB().Save(&stage).Error
}

func (c *ChainService) OnJobTerminal(job *models.Job) {
	if job.ChainID == "" || job.Status != models.JobStatusCompleted {
		return
	}

	var nextStage models.JobChainStage
	if err := c.db.GetDB().Where("chain_id = ? AND stage_index = ?", job.ChainID, job.ChainStageIndex+1).First(&nextStage).Error; err != nil {
		return
	}

	if err := c.advanceStage(job.ChainID, job.ChainStageIndex+1); err != nil {
		log.Printf("[ChainService] Failed to advance chain %s to stage %d: %v", job.ChainID, job.ChainStageIndex+1, err)
	}
}

func (c *ChainService) GetChain(id string) (*models.JobChain, error) {
	var chain models.JobChain
	if err := c.db.GetDB().First(&chain, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &chain, nil
}

func (c *ChainService) GetAllChains(limit, offset int) ([]*models.JobChain, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var chains []*models.JobChain
	err := c.db.GetDB().Order("created_at DESC").Limit(limit).Offset(offset).Find(&chains).Error
	return chains, err
}

func (c *ChainService) GetChainStatus(chainID string) (*ChainStatus, error) {
	chain, err := c.GetChain(chainID)
	if err != nil {
		return nil, err
	}

	var stageRows []models.JobChainStage
	c.db.GetDB().Where("chain_id = ?", chainID).Order("stage_index ASC").Find(&stageRows)

	status := &ChainStatus{
		ChainID:   chain.ID,
		RecipeID:  chain.RecipeID,
		Label:     chain.Label,
		CreatedAt: chain.CreatedAt,
	}

	blocked := false
	anyFailed := false
	anyRunning := false
	allCompleted := len(stageRows) > 0

	for _, stageRow := range stageRows {
		stageStatus := &ChainStageStatus{StageIndex: stageRow.StageIndex, PluginID: stageRow.PluginID}

		if stageRow.JobID == "" {
			if blocked {
				stageStatus.Status = "blocked"
			} else {
				stageStatus.Status = "pending"
			}
			allCompleted = false
		} else if job, err := c.jobQueue.GetJob(stageRow.JobID); err == nil {
			stageStatus.Job = job
			switch job.Status {
			case models.JobStatusCompleted:
				stageStatus.Status = "completed"
			case models.JobStatusFailed:
				stageStatus.Status = "failed"
				blocked = true
				anyFailed = true
				allCompleted = false
			default:
				stageStatus.Status = "running"
				anyRunning = true
				allCompleted = false
			}
		}

		status.Stages = append(status.Stages, stageStatus)
	}

	switch {
	case anyFailed:
		status.Status = "failed"
	case allCompleted:
		status.Status = "completed"
	case anyRunning:
		status.Status = "running"
	default:
		status.Status = "pending"
	}

	return status, nil
}

func (c *ChainService) DeleteChain(chainID string) error {
	jobs := c.jobQueue.GetJobsByChainID(chainID)
	for _, jb := range jobs {
		if err := c.jobQueue.DeleteJob(jb.ID); err != nil {
			log.Printf("[DeleteChain] failed to delete job %s in chain %s: %v", jb.ID, chainID, err)
		}
	}
	if err := c.db.GetDB().Delete(&models.JobChainStage{}, "chain_id = ?", chainID).Error; err != nil {
		return err
	}
	return c.db.GetDB().Delete(&models.JobChain{}, "id = ?", chainID).Error
}
