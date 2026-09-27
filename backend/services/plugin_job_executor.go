package services

import (
	"fmt"
	"os"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func ExecutePluginJob(pluginExecutor *PluginExecutor, jobQueue *JobQueueService, settings *SettingsService, plugin *models.PluginV2, parameters map[string]interface{}, batchID string, chainID string, chainStageIndex int) (string, error) {
	if err := pluginExecutor.ValidateParameters(plugin, parameters); err != nil {
		return "", fmt.Errorf("parameter validation failed: %w", err)
	}

	args, err := pluginExecutor.BuildArguments(plugin, parameters)
	if err != nil {
		return "", fmt.Errorf("failed to build arguments: %w", err)
	}

	cfg := settings.GetConfig()
	baseOutputDir := cfg.OutputDirectory
	if baseOutputDir == "" {
		baseOutputDir = "outputs"
	}

	outputDir := GenerateJobOutputDir(baseOutputDir, plugin.Definition.Plugin.ID)
	os.MkdirAll(outputDir, 0755)

	if plugin.Definition.Execution.OutputDir != "" {
		args = append(args, plugin.Definition.Execution.OutputDir, outputDir)
	}

	params := make(map[string]interface{}, len(parameters)+2)
	for k, v := range parameters {
		params[k] = v
	}
	params["outputDir"] = outputDir
	params["pluginId"] = plugin.ID

	envs := plugin.Definition.Runtime.GetEnvironments()
	runtimeTypeForJob := ""
	if len(envs) > 1 && plugin.Definition.Runtime.HasEnvironment("python") && plugin.Definition.Runtime.HasEnvironment("r") {
		runtimeTypeForJob = "python+r"
	} else if len(envs) > 0 {
		runtimeTypeForJob = envs[0]
	}

	if chainID != "" {
		return jobQueue.CreateJobWithParametersAndChain(
			plugin.Definition.Plugin.ID,
			plugin.Definition.Plugin.Name,
			runtimeTypeForJob,
			args,
			params,
			plugin.Definition.Plugin.Version,
			plugin.CommitHash,
			chainID,
			chainStageIndex,
		)
	}

	return jobQueue.CreateJobWithParametersAndBatch(
		plugin.Definition.Plugin.ID,
		plugin.Definition.Plugin.Name,
		runtimeTypeForJob,
		args,
		params,
		plugin.Definition.Plugin.Version,
		plugin.CommitHash,
		batchID,
	)
}
