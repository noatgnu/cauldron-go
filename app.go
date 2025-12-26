package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/backend/services"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/yaml.v3"
)

type App struct {
	ctx                   context.Context
	db                    *services.DatabaseService
	settings              *services.SettingsService
	fileService           *services.FileService
	jobQueue              *services.JobQueueService
	pythonRunner          *services.PythonRunner
	rRunner               *services.RRunner
	envService            *services.EnvironmentService
	scriptExecutor        *services.ScriptExecutor
	portableEnvService    *services.PortableEnvService
	pluginService         *services.PluginService
	pluginLoaderV2        *services.PluginLoaderV2
	pluginExecutor        *services.PluginExecutor
	pluginInstaller       *services.PluginInstaller
	protocolHandler       *services.ProtocolHandler
	httpInstallServer     *services.HTTPInstallServer
	pluginRegistryService *services.PluginRegistryService
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	log.Println("[App.startup] Starting application...")
	log.Println("[App.startup] OnBeforeClose handler will handle window close events")
	a.ctx = ctx

	if ctx.Value("wails-test") == nil {
		log.Println("[App.startup] Setting up menu...")
		menu := services.BuildApplicationMenu(ctx, a)
		runtime.MenuSetApplicationMenu(ctx, menu)
	}

	log.Println("[App.startup] Initializing database...")
	db, err := services.NewDatabaseService(ctx)
	if err != nil {
		log.Printf("[App.startup] ERROR: Failed to initialize database: %v\n", err)
		fmt.Printf("[App.startup] ERROR: Failed to initialize database: %v\n", err)
		return
	}
	a.db = db
	log.Println("[App.startup] Database initialized successfully")

	log.Println("[App.startup] Initializing services...")
	a.settings = services.NewSettingsService(ctx, db)
	a.fileService = services.NewFileService(ctx)
	a.pythonRunner = services.NewPythonRunner(a.settings)
	a.rRunner = services.NewRRunner(a.settings)
	directRunner := services.NewDirectRunner()
	a.envService = services.NewEnvironmentService(ctx, db, a.settings, services.NewProgressNotifier(ctx))
	a.portableEnvService = services.NewPortableEnvService(ctx, a.fileService)

	log.Println("[App.startup] Initializing job queue...")
	a.jobQueue = services.NewJobQueueService(ctx, db)
	log.Println("[App.startup] Setting job queue runners...")
	a.jobQueue.SetRunners(a.pythonRunner, a.rRunner, directRunner, a.settings)

	log.Println("[App.startup] Initializing script executor...")
	a.scriptExecutor = services.NewScriptExecutor(a.settings, a.db)
	a.scriptExecutor.SetUpdateCallback(func(jobID string, update models.Job) {
		job, err := a.jobQueue.GetJob(jobID)
		if err != nil {
			log.Printf("[App] Failed to get job %s: %v", jobID, err)
			return
		}

		if update.Status != "" {
			job.Status = update.Status
		}
		if update.Progress > 0 {
			job.Progress = update.Progress
		}
		if update.Error != "" {
			job.Error = update.Error
		}
		if update.OutputPath != "" {
			job.OutputPath = update.OutputPath
		}
		if update.Status == "completed" {
			now := time.Now()
			job.CompletedAt = &now
		}

		if err := a.db.GetDB().Save(job).Error; err != nil {
			log.Printf("[App] Failed to save job %s: %v", jobID, err)
		}

		runtime.EventsEmit(a.ctx, "job:update", job)
	})

	a.scriptExecutor.SetOutputCallback(func(jobID string, line string) {
		job, err := a.jobQueue.GetJob(jobID)
		if err != nil {
			return
		}

		job.TerminalOutput = append(job.TerminalOutput, line)
		maxLines := 100
		if len(job.TerminalOutput) > maxLines {
			job.TerminalOutput = job.TerminalOutput[len(job.TerminalOutput)-maxLines:]
		}

		if err := a.db.GetDB().Save(job).Error; err != nil {
			log.Printf("[App] Failed to save job output %s: %v", jobID, err)
		}

		runtime.EventsEmit(a.ctx, "job:output", map[string]interface{}{
			"jobId":  jobID,
			"output": line,
		})
	})

	log.Println("[App.startup] Initializing plugin service...")
	a.pluginService = services.NewPluginService()

	log.Println("[App.startup] Initializing plugin system V2...")
	a.pluginLoaderV2 = services.NewPluginLoaderV2("", a.db)
	if err := a.pluginLoaderV2.LoadPlugins(); err != nil {
		log.Printf("[App.startup] Failed to load plugins: %v", err)
	}
	a.pluginExecutor = services.NewPluginExecutor()

	exePath, _ := os.Executable()
	pluginsDir := filepath.Join(filepath.Dir(exePath), "plugins")
	a.pluginInstaller = services.NewPluginInstaller(pluginsDir, a.db, a.pluginLoaderV2)
	log.Println("[App.startup] Plugin installer initialized")

	log.Println("[App.startup] Initializing plugin registry service...")
	a.pluginRegistryService = services.NewPluginRegistryService(ctx, a.settings)
	log.Println("[App.startup] Plugin registry service initialized")

	log.Println("[App.startup] Initializing protocol handler...")
	a.protocolHandler = services.NewProtocolHandler(a.pluginInstaller)
	a.protocolHandler.SetContext(ctx)
	if err := a.protocolHandler.RegisterProtocol(); err != nil {
		log.Printf("[App.startup] Warning: Failed to register protocol handler: %v", err)
	}
	log.Println("[App.startup] Protocol handler initialized")

	log.Println("[App.startup] Starting HTTP install server...")
	a.httpInstallServer = services.NewHTTPInstallServer(a.protocolHandler)
	if err := a.httpInstallServer.Start(ctx); err != nil {
		log.Printf("[App.startup] Warning: Failed to start HTTP install server: %v", err)
	} else {
		log.Printf("[App.startup] HTTP install server ready at http://localhost:%d", a.httpInstallServer.GetPort())
	}

	log.Println("[App.startup] Checking for protocol URL...")
	a.handleProtocolURL()

	log.Println("[App.startup] Plugin system V2 initialized")

	log.Println("[App.startup] Checking for unfinished jobs...")
	go a.checkUnfinishedJobs()

	log.Println("[App.startup] Application startup complete!")
}

func (a *App) shutdown(ctx context.Context) {
	if a.httpInstallServer != nil {
		log.Println("[App.shutdown] Stopping HTTP install server...")
		if err := a.httpInstallServer.Stop(); err != nil {
			log.Printf("[App.shutdown] Error stopping HTTP install server: %v", err)
		}
	}
	if a.jobQueue != nil {
		a.jobQueue.Shutdown()
	}
	if a.db != nil {
		a.db.Close()
	}
}

func (a *App) GetSettings() *models.Config {
	return a.settings.GetConfig()
}

func (a *App) SetSetting(key string, value interface{}) error {
	return a.settings.Set(key, value)
}

func (a *App) DetectPythonPath() (string, error) {
	return a.settings.DetectPythonPath()
}

func (a *App) DetectRPath() (string, error) {
	return a.settings.DetectRPath()
}

func (a *App) OpenFile(title string) (string, error) {
	return a.fileService.OpenFileDialog(title, nil)
}

func (a *App) OpenDirectory(title string) (string, error) {
	return a.fileService.OpenDirectoryDialog(title)
}

func (a *App) SaveFile(title string, defaultName string) (string, error) {
	return a.fileService.SaveFileDialog(title, defaultName)
}

func (a *App) OpenDirectoryInExplorer(path string) error {
	return a.fileService.OpenDirectoryInExplorer(path)
}

func (a *App) ReadJobOutputFile(jobID string, filename string) (string, error) {
	job, err := a.jobQueue.GetJob(jobID)
	if err != nil {
		return "", err
	}

	if job.OutputPath == "" {
		return "", fmt.Errorf("job has no output directory")
	}

	filePath := filepath.Join(job.OutputPath, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	content := string(data)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	return content, nil
}

func (a *App) GetJobExecutionLog(jobID string) (string, error) {
	job, err := a.jobQueue.GetJob(jobID)
	if err != nil {
		return "", err
	}

	if job.OutputPath == "" {
		return "", fmt.Errorf("job has no output directory")
	}

	logFilePath := filepath.Join(job.OutputPath, "execution.log")
	data, err := os.ReadFile(logFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	content := string(data)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	return content, nil
}

func (a *App) ListJobOutputFiles(jobID string) ([]string, error) {
	job, err := a.jobQueue.GetJob(jobID)
	if err != nil {
		return nil, err
	}

	if job.OutputPath == "" {
		return nil, fmt.Errorf("job has no output directory")
	}

	entries, err := os.ReadDir(job.OutputPath)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

func (a *App) WriteJobOutputFile(jobID string, filename string, content string) error {
	job, err := a.jobQueue.GetJob(jobID)
	if err != nil {
		return err
	}

	if job.OutputPath == "" {
		return fmt.Errorf("job has no output directory")
	}

	filePath := filepath.Join(job.OutputPath, filename)
	return os.WriteFile(filePath, []byte(content), 0644)
}

func (a *App) ReadFile(path string) ([]byte, error) {
	log.Printf("[ReadFile] Reading file: %s", path)
	absPath, err := filepath.Abs(path)
	if err == nil {
		log.Printf("[ReadFile] Absolute path: %s", absPath)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		log.Printf("[ReadFile] ERROR: stat failed: %v", err)
		return nil, err
	}
	log.Printf("[ReadFile] File size: %d bytes", fileInfo.Size())
	content, err := a.fileService.ReadFile(path)
	if err != nil {
		log.Printf("[ReadFile] ERROR: read failed: %v", err)
		return nil, err
	}
	log.Printf("[ReadFile] Read %d bytes", len(content))
	return content, nil
}

func (a *App) ReadFilePreview(path string, limit int) ([]string, error) {
	return a.fileService.ReadFileLines(path, limit)
}

func getStringParam(params map[string]interface{}, key string, defaultVal string) string {
	if val, ok := params[key].(string); ok {
		return val
	}
	return defaultVal
}

func getBoolParam(params map[string]interface{}, key string) bool {
	if val, ok := params[key].(bool); ok {
		return val
	}
	return false
}

func getFloatParam(params map[string]interface{}, key string, defaultVal float64) float64 {
	if val, ok := params[key].(float64); ok {
		return val
	}
	return defaultVal
}

func getIntParam(params map[string]interface{}, key string, defaultVal int) int {
	if val, ok := params[key].(float64); ok {
		return int(val)
	}
	return defaultVal
}

func arrayToCommaString(arr []interface{}) string {
	strs := make([]string, len(arr))
	for i, v := range arr {
		strs[i] = fmt.Sprintf("%v", v)
	}
	return strings.Join(strs, ",")
}

func (a *App) CreateJob(req models.JobRequest) (string, error) {
	log.Printf("[CreateJob] Received job request - Type: %s, Name: %s", req.Type, req.Name)
	log.Printf("[CreateJob] Parameters: %+v", req.Parameters)
	log.Printf("[CreateJob] InputFiles: %v", req.InputFiles)

	switch req.Type {
	case "pca", "phate":
		if len(req.InputFiles) == 0 {
			return "", fmt.Errorf("no input file provided for %s analysis", req.Type)
		}
		inputFile := req.InputFiles[0]

		columnsNameInterface, ok := req.Parameters["columns_name"].([]interface{})
		if !ok {
			log.Printf("[CreateJob] ERROR: columns_name not found or wrong type in parameters")
			return "", fmt.Errorf("columns_name parameter is required and must be an array")
		}

		columns := make([]string, len(columnsNameInterface))
		for i, v := range columnsNameInterface {
			columns[i] = fmt.Sprintf("%v", v)
		}
		log.Printf("[CreateJob] Extracted %d columns: %v", len(columns), columns)

		nComponents := getIntParam(req.Parameters, "n_components", 2)
		useLog2 := getBoolParam(req.Parameters, "log2")

		if req.Type == "pca" {
			return a.RunPCAAnalysis(inputFile, "", columns, nComponents, useLog2)
		} else {
			return a.RunPHATEAnalysis(inputFile, "", columns, nComponents, useLog2)
		}

	case "fuzzy-clustering":
		if len(req.InputFiles) < 2 {
			return "", fmt.Errorf("fuzzy clustering requires input file and annotation file")
		}

		args := []string{
			"fuzz_clustering.py",
			"--file_path", req.InputFiles[0],
			"--annotation_path", req.InputFiles[1],
		}

		if centerCount, ok := req.Parameters["center_count"].(string); ok && centerCount != "" {
			args = append(args, "--center_count", centerCount)
		}

		return a.jobQueue.CreateJob(req.Type, req.Name, "python", args)

	case "alphastats":
		if len(req.InputFiles) < 2 {
			return "", fmt.Errorf("alphastats requires input file and annotation file")
		}

		args := []string{
			"alphapept_process.py",
			"--file_path", req.InputFiles[0],
			"--metadata_path", req.InputFiles[1],
			"--engine", getStringParam(req.Parameters, "engine", "generic"),
		}

		if idx := getStringParam(req.Parameters, "index_col", ""); idx != "" {
			args = append(args, "--index", idx)
		}
		if cols, ok := req.Parameters["merge_columns_list"].([]interface{}); ok && len(cols) > 0 {
			args = append(args, "--merge_columns", arrayToCommaString(cols))
		}
		if method := getStringParam(req.Parameters, "method", ""); method != "" {
			args = append(args, "--method", method)
		}
		if impute := getStringParam(req.Parameters, "imputation", ""); impute != "" {
			args = append(args, "--impute", impute)
		}
		if norm := getStringParam(req.Parameters, "normalization", ""); norm != "" {
			args = append(args, "--normalize", norm)
		}
		if dc := getFloatParam(req.Parameters, "data_completeness", 0); dc > 0 {
			args = append(args, "--data_completeness", fmt.Sprintf("%.2f", dc))
		}
		if getBoolParam(req.Parameters, "log2") {
			args = append(args, "--log2")
		}
		if getBoolParam(req.Parameters, "batch_correction") {
			args = append(args, "--batch_correction")
		}
		if evidenceFile := getStringParam(req.Parameters, "evidence_file", ""); evidenceFile != "" {
			args = append(args, "--evidence_file", evidenceFile)
		}
		if comparisons, ok := req.Parameters["comparisons"].([]interface{}); ok && len(comparisons) > 0 {
			compStr := arrayToCommaString(comparisons)
			args = append(args, "--comparison_matrix", compStr)
		}

		return a.jobQueue.CreateJob(req.Type, req.Name, "python", args)

	case "limma":
		if len(req.InputFiles) < 2 {
			return "", fmt.Errorf("limma requires input file and annotation file")
		}

		args := []string{
			"differential_analysis.py",
			"--input_file", req.InputFiles[0],
			"--annotation_file", req.InputFiles[1],
		}

		if idx := getStringParam(req.Parameters, "index_col", ""); idx != "" {
			args = append(args, "--index_col", idx)
		}
		if getBoolParam(req.Parameters, "log2") {
			args = append(args, "--log2")
		}
		if comparisons, ok := req.Parameters["comparisons"].([]interface{}); ok && len(comparisons) > 0 {
			args = append(args, "--comparison_file", "comparisons.csv")
		}

		return a.jobQueue.CreateJob(req.Type, req.Name, "python", args)

	case "imputation":
		if len(req.InputFiles) == 0 {
			return "", fmt.Errorf("no input file provided for imputation")
		}

		cfg := a.settings.GetConfig()
		baseOutputDir := cfg.OutputDirectory
		if baseOutputDir == "" {
			baseOutputDir = "outputs"
		}

		runtime := getStringParam(req.Parameters, "runtime", "r")
		jobOutputDir := filepath.Join(baseOutputDir, fmt.Sprintf("imputation_%s", time.Now().Format("20060102_150405")))
		os.MkdirAll(jobOutputDir, 0755)

		scriptName := "imputation.R"
		if runtime == "python" {
			scriptName = "imputation.py"
		}

		args := []string{scriptName, "--file_path", req.InputFiles[0], "--output_folder", jobOutputDir}

		if cols, ok := req.Parameters["columns"].([]interface{}); ok && len(cols) > 0 {
			args = append(args, "--columns_name", arrayToCommaString(cols))
		}
		if method := getStringParam(req.Parameters, "method", "knn"); method != "" {
			args = append(args, "--imputer_type", method)
		}
		if k := getIntParam(req.Parameters, "k", 5); k > 0 {
			args = append(args, "--n_neighbors", fmt.Sprintf("%d", k))
		}
		if strategy := getStringParam(req.Parameters, "strategy", "mean"); strategy != "" {
			args = append(args, "--simple_strategy", strategy)
		}
		if fillValue := getFloatParam(req.Parameters, "fillValue", 0.0); fillValue != 0.0 {
			args = append(args, "--fill_value", fmt.Sprintf("%.2f", fillValue))
		}
		if iter := getIntParam(req.Parameters, "iterations", 10); iter > 0 {
			args = append(args, "--max_iter", fmt.Sprintf("%d", iter))
		}

		parameters := map[string]interface{}{
			"outputDir": jobOutputDir,
		}

		return a.jobQueue.CreateJobWithParameters(req.Type, req.Name, runtime, args, parameters)

	case "normalization":
		if len(req.InputFiles) == 0 {
			return "", fmt.Errorf("no input file provided for normalization")
		}

		cfg := a.settings.GetConfig()
		baseOutputDir := cfg.OutputDirectory
		if baseOutputDir == "" {
			baseOutputDir = "outputs"
		}

		runtime := getStringParam(req.Parameters, "runtime", "r")
		jobOutputDir := filepath.Join(baseOutputDir, fmt.Sprintf("normalization_%s", time.Now().Format("20060102_150405")))
		os.MkdirAll(jobOutputDir, 0755)

		scriptName := "normalization.R"
		if runtime == "python" {
			scriptName = "normalization.py"
		}

		args := []string{scriptName, "--file_path", req.InputFiles[0], "--output_folder", jobOutputDir}

		if cols, ok := req.Parameters["columns_name"].([]interface{}); ok && len(cols) > 0 {
			args = append(args, "--columns_name", arrayToCommaString(cols))
		}
		if scaler := getStringParam(req.Parameters, "scaler_type", "minmax"); scaler != "" {
			args = append(args, "--scaler_type", scaler)
		}
		if getBoolParam(req.Parameters, "with_centering") {
			args = append(args, "--with_centering", "True")
		} else {
			args = append(args, "--with_centering", "False")
		}
		if getBoolParam(req.Parameters, "with_scaling") {
			args = append(args, "--with_scaling", "True")
		} else {
			args = append(args, "--with_scaling", "False")
		}
		if nQuant := getIntParam(req.Parameters, "n_quantiles", 1000); nQuant > 0 {
			args = append(args, "--n_quantiles", fmt.Sprintf("%d", nQuant))
		}
		if dist := getStringParam(req.Parameters, "output_distribution", "uniform"); dist != "" {
			args = append(args, "--output_distribution", dist)
		}
		if norm := getStringParam(req.Parameters, "norm", "l2"); norm != "" {
			args = append(args, "--norm", norm)
		}
		if power := getStringParam(req.Parameters, "power_method", "yeo-johnson"); power != "" {
			args = append(args, "--power_method", power)
		}

		parameters := map[string]interface{}{
			"outputDir": jobOutputDir,
		}

		return a.jobQueue.CreateJobWithParameters(req.Type, req.Name, runtime, args, parameters)

	case "correlation-matrix":
		if len(req.InputFiles) == 0 {
			return "", fmt.Errorf("no input file provided for correlation matrix")
		}

		cfg := a.settings.GetConfig()
		baseOutputDir := cfg.OutputDirectory
		if baseOutputDir == "" {
			baseOutputDir = "outputs"
		}

		jobOutputDir := filepath.Join(baseOutputDir, fmt.Sprintf("correlation_matrix_%s", time.Now().Format("20060102_150405")))
		os.MkdirAll(jobOutputDir, 0755)

		args := []string{"correlation_matrix.R", "--file_path", req.InputFiles[0], "--output_folder", jobOutputDir}

		if idx := getStringParam(req.Parameters, "index_col", ""); idx != "" {
			args = append(args, "--index_col", idx)
		}
		if cols, ok := req.Parameters["sample_cols"].([]interface{}); ok && len(cols) > 0 {
			args = append(args, "--sample_cols", arrayToCommaString(cols))
		}
		if method := getStringParam(req.Parameters, "method", "pearson"); method != "" {
			args = append(args, "--method", method)
		}
		if minVal := getFloatParam(req.Parameters, "min_value", 0); minVal != 0 {
			args = append(args, "--min_value", fmt.Sprintf("%.2f", minVal))
		}
		if order := getStringParam(req.Parameters, "order", ""); order != "" {
			args = append(args, "--order", order)
		}
		if hclust := getStringParam(req.Parameters, "hclust_method", "ward.D"); hclust != "" {
			args = append(args, "--hclust_method", hclust)
		}
		if pm := getStringParam(req.Parameters, "presenting_method", "ellipse"); pm != "" {
			args = append(args, "--presenting_method", pm)
		}
		if shape := getStringParam(req.Parameters, "cor_shape", "upper"); shape != "" {
			args = append(args, "--cor_shape", shape)
		}
		if palette := getStringParam(req.Parameters, "color_ramp_palette", ""); palette != "" {
			args = append(args, "--color_ramp_palette", palette)
		}
		if plotWidth := getIntParam(req.Parameters, "plot_width", 10); plotWidth > 0 {
			args = append(args, "--plot_width", fmt.Sprintf("%d", plotWidth))
		}
		if plotHeight := getIntParam(req.Parameters, "plot_height", 10); plotHeight > 0 {
			args = append(args, "--plot_height", fmt.Sprintf("%d", plotHeight))
		}
		if textSize := getFloatParam(req.Parameters, "text_label_size", 1.0); textSize > 0 {
			args = append(args, "--text_label_size", fmt.Sprintf("%.2f", textSize))
		}
		if numSize := getFloatParam(req.Parameters, "number_label_size", 1.0); numSize > 0 {
			args = append(args, "--number_label_size", fmt.Sprintf("%.2f", numSize))
		}
		if rotation := getIntParam(req.Parameters, "label_rotation", 45); rotation >= 0 {
			args = append(args, "--label_rotation", fmt.Sprintf("%d", rotation))
		}
		if getBoolParam(req.Parameters, "show_diagonal") {
			args = append(args, "--show_diagonal", "true")
		} else {
			args = append(args, "--show_diagonal", "false")
		}
		if getBoolParam(req.Parameters, "add_grid") {
			args = append(args, "--add_grid", "true")
			if gridColor := getStringParam(req.Parameters, "grid_color", "white"); gridColor != "" {
				args = append(args, "--grid_color", gridColor)
			}
		} else {
			args = append(args, "--add_grid", "false")
		}
		if numDigits := getIntParam(req.Parameters, "number_digits", 2); numDigits >= 0 {
			args = append(args, "--number_digits", fmt.Sprintf("%d", numDigits))
		}
		if plotTitle := getStringParam(req.Parameters, "plot_title", ""); plotTitle != "" {
			args = append(args, "--plot_title", plotTitle)
		}

		parameters := map[string]interface{}{
			"outputDir":  jobOutputDir,
			"inputFiles": req.InputFiles,
		}

		return a.jobQueue.CreateJobWithParameters(req.Type, req.Name, "r", args, parameters)

	case "maxlfq":
		if len(req.InputFiles) == 0 {
			return "", fmt.Errorf("no input file provided for MaxLFQ normalization")
		}

		cfg := a.settings.GetConfig()
		baseOutputDir := cfg.OutputDirectory
		if baseOutputDir == "" {
			baseOutputDir = "outputs"
		}

		jobOutputDir := filepath.Join(baseOutputDir, fmt.Sprintf("maxlfq_%s", time.Now().Format("20060102_150405")))
		os.MkdirAll(jobOutputDir, 0755)

		args := []string{"maxlfq.R", "--file_path", req.InputFiles[0], "--output_folder", jobOutputDir}

		if proteinCol := getStringParam(req.Parameters, "protein_col", "Protein.Group"); proteinCol != "" {
			args = append(args, "--protein_col", proteinCol)
		}
		if peptideCol := getStringParam(req.Parameters, "peptide_col", "Precursor.Id"); peptideCol != "" {
			args = append(args, "--peptide_col", peptideCol)
		}
		if sampleCols, ok := req.Parameters["sample_cols"].([]interface{}); ok && len(sampleCols) > 0 {
			args = append(args, "--sample_cols", arrayToCommaString(sampleCols))
		}
		if minSamples := getIntParam(req.Parameters, "min_samples", 1); minSamples > 0 {
			args = append(args, "--min_samples", fmt.Sprintf("%d", minSamples))
		}
		if getBoolParam(req.Parameters, "use_log2") {
			args = append(args, "--use_log2", "true")
		}
		if getBoolParam(req.Parameters, "normalize") {
			args = append(args, "--normalize", "true")
		}

		parameters := map[string]interface{}{
			"outputDir": jobOutputDir,
		}

		return a.jobQueue.CreateJobWithParameters(req.Type, req.Name, "r", args, parameters)

	case "batch-correction":
		if len(req.InputFiles) == 0 {
			return "", fmt.Errorf("no input file provided for batch correction")
		}

		cfg := a.settings.GetConfig()
		baseOutputDir := cfg.OutputDirectory
		if baseOutputDir == "" {
			baseOutputDir = "outputs"
		}

		jobOutputDir := filepath.Join(baseOutputDir, fmt.Sprintf("batch_correction_%s", time.Now().Format("20060102_150405")))
		os.MkdirAll(jobOutputDir, 0755)

		args := []string{"batch_correction.R", "--file_path", req.InputFiles[0], "--output_folder", jobOutputDir}

		if sampleCols, ok := req.Parameters["sample_cols"].([]interface{}); ok && len(sampleCols) > 0 {
			args = append(args, "--sample_cols", arrayToCommaString(sampleCols))
		}
		if batchInfo := getStringParam(req.Parameters, "batch_info", ""); batchInfo != "" {
			args = append(args, "--batch_info", batchInfo)
		}
		if method := getStringParam(req.Parameters, "method", "combat"); method != "" {
			args = append(args, "--method", method)
		}
		if preserveGroup := getStringParam(req.Parameters, "preserve_group", ""); preserveGroup != "" {
			args = append(args, "--preserve_group", preserveGroup)
		}
		if getBoolParam(req.Parameters, "use_log2") {
			args = append(args, "--use_log2", "true")
		}

		parameters := map[string]interface{}{
			"outputDir": jobOutputDir,
		}

		return a.jobQueue.CreateJobWithParameters(req.Type, req.Name, "r", args, parameters)

	case "venn-diagram":
		if len(req.InputFiles) == 0 {
			return "", fmt.Errorf("no input file provided for venn diagram")
		}

		cfg := a.settings.GetConfig()
		baseOutputDir := cfg.OutputDirectory
		if baseOutputDir == "" {
			baseOutputDir = "outputs"
		}

		jobOutputDir := filepath.Join(baseOutputDir, fmt.Sprintf("venn_diagram_%s", time.Now().Format("20060102_150405")))
		os.MkdirAll(jobOutputDir, 0755)

		args := []string{"venn_diagram.R", "--file_path", req.InputFiles[0], "--output_folder", jobOutputDir}

		if sampleCols, ok := req.Parameters["sample_cols"].([]interface{}); ok && len(sampleCols) > 0 {
			args = append(args, "--sample_cols", arrayToCommaString(sampleCols))
		}
		if setNames := getStringParam(req.Parameters, "set_names", ""); setNames != "" {
			args = append(args, "--set_names", setNames)
		}
		if threshold := getFloatParam(req.Parameters, "threshold", 0.0); threshold >= 0 {
			args = append(args, "--threshold", fmt.Sprintf("%.2f", threshold))
		}
		if getBoolParam(req.Parameters, "use_presence") {
			args = append(args, "--use_presence", "true")
		} else {
			args = append(args, "--use_presence", "false")
		}
		if fillColors := getStringParam(req.Parameters, "fill_colors", ""); fillColors != "" {
			args = append(args, "--fill_colors", fillColors)
		}
		if alpha := getFloatParam(req.Parameters, "alpha", 0.5); alpha >= 0 && alpha <= 1 {
			args = append(args, "--alpha", fmt.Sprintf("%.2f", alpha))
		}

		parameters := map[string]interface{}{
			"outputDir": jobOutputDir,
		}

		return a.jobQueue.CreateJobWithParameters(req.Type, req.Name, "r", args, parameters)

	case "estimation-plot":
		if len(req.InputFiles) < 2 {
			return "", fmt.Errorf("estimation plot requires input file and annotation file")
		}

		args := []string{
			"estimation_plot.py",
			"--file_path", req.InputFiles[0],
			"--sample_annotation", req.InputFiles[1],
		}

		if idx := getStringParam(req.Parameters, "index_col", ""); idx != "" {
			args = append(args, "--index_col", idx)
		}
		if protein := getStringParam(req.Parameters, "selected_protein", ""); protein != "" {
			args = append(args, "--selected_protein", protein)
		}
		if getBoolParam(req.Parameters, "log2") {
			args = append(args, "--log2")
		}
		if order := getStringParam(req.Parameters, "condition_order", ""); order != "" {
			args = append(args, "--condition_order", order)
		}

		return a.jobQueue.CreateJob(req.Type, req.Name, "python", args)

	case "cv-plot":
		args := []string{"cv.py"}

		if logFile := getStringParam(req.Parameters, "log_file_path", ""); logFile != "" {
			args = append(args, "--log_file_path", logFile)
		}
		if reportPR := getStringParam(req.Parameters, "report_pr_file_path", ""); reportPR != "" {
			args = append(args, "--report_pr_file_path", reportPR)
		}
		if reportPG := getStringParam(req.Parameters, "report_pg_file_path", ""); reportPG != "" {
			args = append(args, "--report_pg_file_path", reportPG)
		}
		if intensityCol := getStringParam(req.Parameters, "intensity_col", "Intensity"); intensityCol != "" {
			args = append(args, "--intensity_col", intensityCol)
		}
		if annFile := getStringParam(req.Parameters, "annotation_file", ""); annFile != "" {
			args = append(args, "--annotation_file", annFile)
		}
		if samples := getStringParam(req.Parameters, "sample_names", ""); samples != "" {
			args = append(args, "--sample_names", samples)
		}

		return a.jobQueue.CreateJob(req.Type, req.Name, "python", args)

	case "fold-change-violin":
		if len(req.InputFiles) == 0 {
			return "", fmt.Errorf("no input file provided for fold change violin")
		}

		args := []string{"fold_change_violin_plot.py", "--file_path", req.InputFiles[0]}

		if prefix := getStringParam(req.Parameters, "columns_prefix", "Difference"); prefix != "" {
			args = append(args, "--columns_prefix", prefix)
		}
		if categories := getStringParam(req.Parameters, "categories", ""); categories != "" {
			args = append(args, "--categories", categories)
		}
		if match := getStringParam(req.Parameters, "match_value", "+"); match != "" {
			args = append(args, "--match_value", match)
		}
		if fcCol := getStringParam(req.Parameters, "fold_enrichment_col", "Fold enrichment"); fcCol != "" {
			args = append(args, "--fold_enrichment_col", fcCol)
		}
		if orgCol := getStringParam(req.Parameters, "organelle_col", "Organelle"); orgCol != "" {
			args = append(args, "--organelle_col", orgCol)
		}
		if compCol := getStringParam(req.Parameters, "comparison_col", "Comparison"); compCol != "" {
			args = append(args, "--comparison_col", compCol)
		}
		if colors := getStringParam(req.Parameters, "colors", ""); colors != "" {
			args = append(args, "--colors", colors)
		}
		if figsize := getStringParam(req.Parameters, "figsize", "6,10"); figsize != "" {
			args = append(args, "--figsize", figsize)
		}

		return a.jobQueue.CreateJob(req.Type, req.Name, "python", args)

	case "qfeatures_limma":
		return "", fmt.Errorf("QFeatures + Limma analysis is not yet implemented - backend script pending")

	default:
		return "", fmt.Errorf("unknown job type: %s", req.Type)
	}
}

func (a *App) GetJob(id string) (*models.Job, error) {
	return a.jobQueue.GetJob(id)
}

func (a *App) GetAllJobs() []*models.Job {
	return a.jobQueue.GetAllJobs()
}

func (a *App) DeleteJob(id string) error {
	return a.jobQueue.DeleteJob(id)
}

func (a *App) RerunJob(jobID string, useSameEnvironment bool, pythonEnvPath string, rEnvPath string) (string, error) {
	return a.jobQueue.RerunJob(jobID, useSameEnvironment, pythonEnvPath, rEnvPath)
}

func (a *App) ReExecuteJob(id string) (string, error) {
	job, err := a.jobQueue.GetJob(id)
	if err != nil {
		return "", fmt.Errorf("failed to get job: %w", err)
	}

	switch job.Type {
	case "pca":
		inputFile, _ := job.Parameters["inputFile"].(string)
		outputDir, _ := job.Parameters["outputDir"].(string)
		nComponents := int(job.Parameters["nComponents"].(float64))
		useLog2, _ := job.Parameters["useLog2"].(bool)

		columnsInterface, _ := job.Parameters["columns"].([]interface{})
		columns := make([]string, len(columnsInterface))
		for i, v := range columnsInterface {
			columns[i] = v.(string)
		}

		return a.RunPCAAnalysis(inputFile, outputDir, columns, nComponents, useLog2)

	case "normalization":
		inputFile, _ := job.Parameters["inputFile"].(string)
		outputDir, _ := job.Parameters["outputDir"].(string)
		scalerType, _ := job.Parameters["scalerType"].(string)

		columnsInterface, _ := job.Parameters["columns"].([]interface{})
		columns := make([]string, len(columnsInterface))
		for i, v := range columnsInterface {
			columns[i] = v.(string)
		}

		return a.RunNormalization(inputFile, outputDir, columns, scalerType)

	case "correlation-matrix":
		inputFiles, _ := job.Parameters["inputFiles"].([]interface{})
		inputFile := inputFiles[0].(string)

		cfg := a.settings.GetConfig()
		baseOutputDir := cfg.OutputDirectory
		if baseOutputDir == "" {
			baseOutputDir = "outputs"
		}

		jobOutputDir := filepath.Join(baseOutputDir, fmt.Sprintf("correlation_matrix_%s", time.Now().Format("20060102_150405")))
		os.MkdirAll(jobOutputDir, 0755)

		args := []string{"correlation_matrix.R", "--file_path", inputFile, "--output_folder", jobOutputDir}

		if idx := getStringParam(job.Parameters, "index_col", ""); idx != "" {
			args = append(args, "--index_col", idx)
		}
		if cols, ok := job.Parameters["sample_cols"].([]interface{}); ok && len(cols) > 0 {
			args = append(args, "--sample_cols", arrayToCommaString(cols))
		}
		if method := getStringParam(job.Parameters, "method", "pearson"); method != "" {
			args = append(args, "--method", method)
		}
		if minVal, ok := job.Parameters["min_value"].(float64); ok && minVal != 0 {
			args = append(args, "--min_value", fmt.Sprintf("%.2f", minVal))
		}
		if order := getStringParam(job.Parameters, "order", ""); order != "" {
			args = append(args, "--order", order)
		}
		if hclust := getStringParam(job.Parameters, "hclust_method", "ward.D"); hclust != "" {
			args = append(args, "--hclust_method", hclust)
		}
		if pm := getStringParam(job.Parameters, "presenting_method", "ellipse"); pm != "" {
			args = append(args, "--presenting_method", pm)
		}
		if shape := getStringParam(job.Parameters, "cor_shape", "upper"); shape != "" {
			args = append(args, "--cor_shape", shape)
		}
		if palette := getStringParam(job.Parameters, "color_ramp_palette", ""); palette != "" {
			args = append(args, "--color_ramp_palette", palette)
		}
		if plotWidth := getIntParam(job.Parameters, "plot_width", 10); plotWidth > 0 {
			args = append(args, "--plot_width", fmt.Sprintf("%d", plotWidth))
		}
		if plotHeight := getIntParam(job.Parameters, "plot_height", 10); plotHeight > 0 {
			args = append(args, "--plot_height", fmt.Sprintf("%d", plotHeight))
		}
		if textSize := getFloatParam(job.Parameters, "text_label_size", 1.0); textSize > 0 {
			args = append(args, "--text_label_size", fmt.Sprintf("%.2f", textSize))
		}
		if numSize := getFloatParam(job.Parameters, "number_label_size", 1.0); numSize > 0 {
			args = append(args, "--number_label_size", fmt.Sprintf("%.2f", numSize))
		}
		if rotation := getIntParam(job.Parameters, "label_rotation", 45); rotation >= 0 {
			args = append(args, "--label_rotation", fmt.Sprintf("%d", rotation))
		}
		if getBoolParam(job.Parameters, "show_diagonal") {
			args = append(args, "--show_diagonal", "true")
		} else {
			args = append(args, "--show_diagonal", "false")
		}
		if getBoolParam(job.Parameters, "add_grid") {
			args = append(args, "--add_grid", "true")
			if gridColor := getStringParam(job.Parameters, "grid_color", "white"); gridColor != "" {
				args = append(args, "--grid_color", gridColor)
			}
		} else {
			args = append(args, "--add_grid", "false")
		}
		if numDigits := getIntParam(job.Parameters, "number_digits", 2); numDigits >= 0 {
			args = append(args, "--number_digits", fmt.Sprintf("%d", numDigits))
		}
		if plotTitle := getStringParam(job.Parameters, "plot_title", ""); plotTitle != "" {
			args = append(args, "--plot_title", plotTitle)
		}

		rerunParameters := make(map[string]interface{})
		for k, v := range job.Parameters {
			rerunParameters[k] = v
		}
		rerunParameters["outputDir"] = jobOutputDir

		return a.jobQueue.CreateJobWithParameters(job.Type, job.Name, "r", args, rerunParameters)

	default:
		return "", fmt.Errorf("unsupported job type for re-execution: %s", job.Type)
	}
}

func (a *App) ExecutePythonScript(scriptName string, args []string) (string, error) {
	var output string
	err := a.pythonRunner.ExecuteScript(scriptName, args, func(line string) {
		output += line + "\n"
		runtime.EventsEmit(a.ctx, "script:output", line)
	})
	return output, err
}

func (a *App) ExecuteRScript(scriptName string, args []string) (string, error) {
	var output string
	err := a.rRunner.ExecuteScript(scriptName, args, func(line string) {
		output += line + "\n"
		runtime.EventsEmit(a.ctx, "script:output", line)
	})
	return output, err
}

func (a *App) GetPythonVersion() (string, error) {
	return a.pythonRunner.GetPythonVersion()
}

func (a *App) GetRVersion() (string, error) {
	return a.rRunner.GetRVersion()
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) DetectPythonEnvironments() ([]services.PythonEnvironment, error) {
	return a.envService.DetectPythonEnvironments()
}

func (a *App) DetectREnvironments() ([]services.REnvironment, error) {
	return a.envService.DetectREnvironments()
}

func (a *App) GetActivePythonEnvironment() (*services.PythonEnvironment, error) {
	return a.db.GetActivePythonEnvironment()
}

func (a *App) GetActiveREnvironment() (*services.REnvironment, error) {
	return a.db.GetActiveREnvironment()
}

func (a *App) SetActivePythonEnvironment(path string) error {
	return a.db.SetActivePythonEnvironment(path)
}

func (a *App) SetActiveREnvironment(path string) error {
	return a.db.SetActiveREnvironment(path)
}

func (a *App) InstallPythonPackages(pythonPath string, packages []string) error {
	return a.envService.InstallPythonPackages(pythonPath, packages)
}

func (a *App) InstallPythonRequirements(pythonPath string, requirementsPath string) error {
	return a.envService.InstallPythonRequirements(pythonPath, requirementsPath)
}

func (a *App) InstallRPackages(rPath string, packages []string) error {
	return a.envService.InstallRPackages(rPath, packages)
}

func (a *App) ListPythonPackages(pythonPath string) ([]string, error) {
	return a.envService.ListPythonPackages(pythonPath)
}

func (a *App) ListRPackages(rPath string) ([]string, error) {
	return a.envService.ListRPackages(rPath)
}

func (a *App) CreatePythonVirtualEnv(basePythonPath string, venvPath string, pluginID string) error {
	return a.envService.CreatePythonVirtualEnv(basePythonPath, venvPath, pluginID)
}

func (a *App) GetVirtualEnvironments() ([]services.VirtualEnvironment, error) {
	return a.envService.GetVirtualEnvironments()
}

func (a *App) DeleteVirtualEnvironment(id uint) error {
	return a.envService.DeleteVirtualEnvironment(id)
}

func (a *App) CreateRenvEnvironment(name string, packages []string, pluginID string, useCache bool) error {
	log.Printf("[App] Creating renv environment: %s with %d packages for plugin: %s (useCache: %v)", name, len(packages), pluginID, useCache)
	return a.envService.CreateRenvEnvironment(name, packages, pluginID, useCache)
}

func (a *App) GetRenvEnvironments() ([]services.RenvEnvironment, error) {
	return a.envService.GetRenvEnvironments()
}

func (a *App) DeleteRenvEnvironment(id uint) error {
	return a.envService.DeleteRenvEnvironment(id)
}

func (a *App) BindPluginToEnvironment(pluginID string, envType string, envID uint, envPath string) error {
	log.Printf("[App] Binding plugin %s to %s environment %d", pluginID, envType, envID)
	binding := services.PluginEnvironmentBinding{
		PluginID:        pluginID,
		EnvironmentType: envType,
		EnvironmentID:   envID,
		EnvironmentPath: envPath,
	}
	return a.db.SavePluginEnvironmentBinding(binding)
}

func (a *App) GetPluginEnvironmentBinding(pluginID string, envType string) (*services.PluginEnvironmentBinding, error) {
	return a.db.GetPluginEnvironmentBinding(pluginID, envType)
}

func (a *App) DeletePluginEnvironmentBinding(pluginID string, envType string) error {
	return a.db.DeletePluginEnvironmentBinding(pluginID, envType)
}

func (a *App) GetAllPluginEnvironmentBindings() ([]services.PluginEnvironmentBinding, error) {
	return a.db.GetAllPluginEnvironmentBindings()
}

func (a *App) SaveCustomEnvVar(envVar services.CustomEnvVar) error {
	return a.db.SaveCustomEnvVar(envVar)
}

func (a *App) GetCustomEnvVars(pluginID uint) ([]services.CustomEnvVar, error) {
	return a.db.GetCustomEnvVars(pluginID)
}

func (a *App) GetGlobalCustomEnvVars() ([]services.CustomEnvVar, error) {
	return a.db.GetGlobalCustomEnvVars()
}

func (a *App) DeleteCustomEnvVar(id uint) error {
	return a.db.DeleteCustomEnvVar(id)
}

func (a *App) DeleteCustomEnvVarByKey(pluginID uint, key string) error {
	return a.db.DeleteCustomEnvVarByKey(pluginID, key)
}

func (a *App) GetBundledRequirementsPath(requirementType string) (string, error) {
	return a.envService.GetBundledRequirementsPath(requirementType)
}

func (a *App) LoadRPackagesFromFile(filePath string) ([]string, error) {
	return a.envService.LoadRPackagesFromFile(filePath)
}

func (a *App) GetExampleFilePath(exampleType string, fileName string) (string, error) {
	return a.envService.GetExampleFilePath(exampleType, fileName)
}

func (a *App) OpenDataFileDialog() (string, error) {
	return a.fileService.OpenDataFileDialog()
}

func (a *App) OpenDirectoryDialog(title string) (string, error) {
	return a.fileService.OpenDirectoryDialog(title)
}

func (a *App) ParseDataFile(path string, previewRows int) (*services.DataFilePreview, error) {
	return a.fileService.ParseDataFile(path, previewRows)
}

func (a *App) ImportDataFile(path string) (uint, error) {
	var existingFile services.ImportedFile
	err := a.db.GetDB().Where("path = ?", path).First(&existingFile).Error
	if err == nil {
		return existingFile.ID, nil
	}

	info, err := a.fileService.GetFileInfo(path)
	if err != nil {
		return 0, err
	}

	preview, err := a.fileService.ParseDataFile(path, 5)
	if err != nil {
		return 0, err
	}

	importedFile := &services.ImportedFile{
		Name:       info.Name,
		Path:       path,
		Size:       info.Size,
		ImportedAt: info.ModTime.Unix(),
		FileType:   preview.FileType,
		Preview:    fmt.Sprintf("%d rows, %d columns", preview.TotalRows, len(preview.Headers)),
	}

	if err := a.db.GetDB().Create(importedFile).Error; err != nil {
		return 0, err
	}

	runtime.EventsEmit(a.ctx, "file:imported", importedFile)

	return importedFile.ID, nil
}

func (a *App) GetImportedFiles() ([]services.ImportedFile, error) {
	var files []services.ImportedFile
	err := a.db.GetDB().Order("imported_at DESC").Limit(10).Find(&files).Error
	return files, err
}

func (a *App) DeleteImportedFile(id uint) error {
	return a.db.GetDB().Delete(&services.ImportedFile{}, id).Error
}

func (a *App) RunPCAAnalysis(inputFile string, outputDir string, columns []string, nComponents int, useLog2 bool) (string, error) {
	log.Printf("[RunPCAAnalysis] Starting PCA Analysis")
	log.Printf("[RunPCAAnalysis] Input file: %s", inputFile)
	log.Printf("[RunPCAAnalysis] Number of columns: %d", len(columns))
	log.Printf("[RunPCAAnalysis] Columns: %v", columns)
	log.Printf("[RunPCAAnalysis] nComponents: %d, useLog2: %v", nComponents, useLog2)

	cfg := a.settings.GetConfig()
	baseOutputDir := cfg.OutputDirectory
	if baseOutputDir == "" {
		baseOutputDir = "outputs"
	}

	jobOutputDir := filepath.Join(baseOutputDir, fmt.Sprintf("pca_analysis_%s", time.Now().Format("20060102_150405")))
	os.MkdirAll(jobOutputDir, 0755)

	columnsStr := ""
	for i, col := range columns {
		if i > 0 {
			columnsStr += ","
		}
		columnsStr += col
	}
	log.Printf("[RunPCAAnalysis] Columns string: %s", columnsStr)

	args := []string{
		"pca.py",
		"--input_file", inputFile,
		"--output_folder", jobOutputDir,
		"--columns_name", columnsStr,
		"--n_components", fmt.Sprintf("%d", nComponents),
	}
	if useLog2 {
		args = append(args, "--log2", "true")
	}
	log.Printf("[RunPCAAnalysis] Python script args: %v", args)

	parameters := map[string]interface{}{
		"inputFile":   inputFile,
		"outputDir":   jobOutputDir,
		"columns":     columns,
		"nComponents": nComponents,
		"useLog2":     useLog2,
	}

	jobID, err := a.jobQueue.CreateJobWithParameters("pca", "PCA Analysis", "python", args, parameters)
	if err != nil {
		log.Printf("[RunPCAAnalysis] ERROR creating job: %v", err)
		return "", err
	}
	log.Printf("[RunPCAAnalysis] Job created with ID: %s", jobID)

	go func() {
		err := a.scriptExecutor.ExecutePythonScript(a.ctx, jobID, services.ScriptConfig{
			Type:       "pca",
			ScriptName: "pca.py",
			Args:       args[1:],
			OutputDir:  jobOutputDir,
		})
		if err != nil {
			log.Printf("[RunPCAAnalysis] Error: %v", err)
		}
	}()

	return jobID, nil
}

func (a *App) RunPHATEAnalysis(inputFile string, outputDir string, columns []string, nComponents int, useLog2 bool) (string, error) {
	log.Printf("[RunPHATEAnalysis] Starting PHATE Analysis")
	log.Printf("[RunPHATEAnalysis] Input file: %s", inputFile)
	log.Printf("[RunPHATEAnalysis] Number of columns: %d", len(columns))
	log.Printf("[RunPHATEAnalysis] Columns: %v", columns)
	log.Printf("[RunPHATEAnalysis] nComponents: %d, useLog2: %v", nComponents, useLog2)

	cfg := a.settings.GetConfig()
	baseOutputDir := cfg.OutputDirectory
	if baseOutputDir == "" {
		baseOutputDir = "outputs"
	}

	jobOutputDir := filepath.Join(baseOutputDir, fmt.Sprintf("phate_analysis_%s", time.Now().Format("20060102_150405")))
	os.MkdirAll(jobOutputDir, 0755)

	columnsStr := ""
	for i, col := range columns {
		if i > 0 {
			columnsStr += ","
		}
		columnsStr += col
	}
	log.Printf("[RunPHATEAnalysis] Columns string: %s", columnsStr)

	args := []string{
		"phate_analysis.py",
		"--input_file", inputFile,
		"--output_folder", jobOutputDir,
		"--columns_name", columnsStr,
		"--n_components", fmt.Sprintf("%d", nComponents),
	}
	if useLog2 {
		args = append(args, "--log2", "true")
	}
	log.Printf("[RunPHATEAnalysis] Python script args: %v", args)

	parameters := map[string]interface{}{
		"inputFile":   inputFile,
		"outputDir":   jobOutputDir,
		"columns":     columns,
		"nComponents": nComponents,
		"useLog2":     useLog2,
	}

	jobID, err := a.jobQueue.CreateJobWithParameters("phate", "PHATE Analysis", "python", args, parameters)
	if err != nil {
		log.Printf("[RunPHATEAnalysis] ERROR creating job: %v", err)
		return "", err
	}
	log.Printf("[RunPHATEAnalysis] Job created with ID: %s", jobID)

	go func() {
		err := a.scriptExecutor.ExecutePythonScript(a.ctx, jobID, services.ScriptConfig{
			Type:       "phate",
			ScriptName: "phate_analysis.py",
			Args:       args[1:],
			OutputDir:  jobOutputDir,
		})
		if err != nil {
			log.Printf("[RunPHATEAnalysis] Error: %v", err)
		}
	}()

	return jobID, nil
}

func (a *App) RunNormalization(inputFile string, outputDir string, columns []string, scalerType string) (string, error) {
	cfg := a.settings.GetConfig()
	baseOutputDir := cfg.OutputDirectory
	if baseOutputDir == "" {
		baseOutputDir = "outputs"
	}

	jobOutputDir := filepath.Join(baseOutputDir, fmt.Sprintf("normalization_%s", time.Now().Format("20060102_150405")))
	os.MkdirAll(jobOutputDir, 0755)

	columnsStr := ""
	for i, col := range columns {
		if i > 0 {
			columnsStr += ","
		}
		columnsStr += col
	}

	args := []string{
		"normalization.py",
		"-f", inputFile,
		"-o", jobOutputDir,
		"-c", columnsStr,
		"-s", scalerType,
	}

	parameters := map[string]interface{}{
		"inputFile":  inputFile,
		"outputDir":  jobOutputDir,
		"columns":    columns,
		"scalerType": scalerType,
	}

	jobID, err := a.jobQueue.CreateJobWithParameters("normalization", "Data Normalization", "python", args, parameters)
	if err != nil {
		return "", err
	}

	go func() {
		err := a.scriptExecutor.ExecutePythonScript(a.ctx, jobID, services.ScriptConfig{
			Type:       "normalization",
			ScriptName: "normalization.py",
			Args:       args[1:],
			OutputDir:  jobOutputDir,
		})
		if err != nil {
			log.Printf("[RunNormalization] Error: %v", err)
		}
	}()

	return jobID, nil
}

func (a *App) GetPortableEnvironmentURL(platform, arch, version, environment string) (string, error) {
	return a.portableEnvService.GetPortableEnvironmentURL(platform, arch, version, environment)
}

func (a *App) DownloadPortableEnvironment(url, environment string) error {
	return a.portableEnvService.DownloadPortableEnvironment(url, environment)
}

func (a *App) GetPortableEnvironmentPath(environment string) (string, error) {
	return a.portableEnvService.GetPortableEnvironmentPath(environment)
}

func (a *App) GetPlugins() []*models.Plugin {
	return a.pluginService.GetPlugins()
}

func (a *App) GetPlugin(id string) (*models.Plugin, error) {
	return a.pluginService.GetPlugin(id)
}

func (a *App) ReloadPlugins() error {
	return a.pluginService.ReloadPlugins()
}

func (a *App) GetPluginsDirectory() string {
	return a.pluginService.GetPluginsDirectory()
}

func (a *App) CreateSamplePlugin() error {
	return a.pluginService.CreateSamplePlugin()
}

func (a *App) ExecutePlugin(req models.PluginExecutionRequest) (string, error) {
	plugin, err := a.pluginService.GetPlugin(req.PluginID)
	if err != nil {
		return "", fmt.Errorf("plugin not found: %w", err)
	}

	args := []string{plugin.ScriptPath}
	for _, input := range plugin.Config.Inputs {
		value, ok := req.Parameters[input.Name]
		if !ok {
			if input.Required {
				return "", fmt.Errorf("required parameter missing: %s", input.Name)
			}
			if input.Default != nil {
				value = input.Default
			} else {
				continue
			}
		}

		args = append(args, fmt.Sprintf("--%s", input.Name), fmt.Sprintf("%v", value))
	}

	cfg := a.settings.GetConfig()
	baseOutputDir := cfg.OutputDirectory
	if baseOutputDir == "" {
		baseOutputDir = "outputs"
	}

	outputDir := filepath.Join(baseOutputDir, fmt.Sprintf("plugin_%s_%s", plugin.ID, time.Now().Format("20060102_150405")))
	os.MkdirAll(outputDir, 0755)

	args = append(args, "--output", outputDir)

	parameters := make(map[string]interface{})
	for k, v := range req.Parameters {
		parameters[k] = v
	}
	parameters["outputDir"] = outputDir

	jobName := fmt.Sprintf("Plugin: %s", plugin.Config.Name)

	var jobID string
	switch plugin.Config.Runtime {
	case models.PluginRuntimePython:
		jobID, err = a.jobQueue.CreateJobWithParameters("plugin", jobName, "python", args, parameters)
		if err != nil {
			return "", err
		}

		go func() {
			err := a.scriptExecutor.ExecutePythonScript(a.ctx, jobID, services.ScriptConfig{
				Type:       "plugin",
				ScriptName: plugin.ScriptPath,
				Args:       args[1:],
				OutputDir:  outputDir,
			})
			if err != nil {
				log.Printf("[ExecutePlugin] Error: %v", err)
			}
		}()

	case models.PluginRuntimeR:
		jobID, err = a.jobQueue.CreateJobWithParameters("plugin", jobName, "r", args, parameters)
		if err != nil {
			return "", err
		}

		go func() {
			err := a.scriptExecutor.ExecuteRScript(a.ctx, jobID, services.ScriptConfig{
				Type:       "plugin",
				ScriptName: plugin.ScriptPath,
				Args:       args[1:],
				OutputDir:  outputDir,
			})
			if err != nil {
				log.Printf("[ExecutePlugin] Error: %v", err)
			}
		}()

	case models.PluginRuntimePythonWithR:
		jobID, err = a.jobQueue.CreateJobWithParameters("plugin", jobName, "python", args, parameters)
		if err != nil {
			return "", err
		}

		cfg := a.settings.GetConfig()
		if cfg.RPath != "" {
			args = append(args, "--r_home", cfg.RPath)
		}

		go func() {
			err := a.scriptExecutor.ExecutePythonScript(a.ctx, jobID, services.ScriptConfig{
				Type:       "plugin",
				ScriptName: plugin.ScriptPath,
				Args:       args[1:],
				OutputDir:  outputDir,
			})
			if err != nil {
				log.Printf("[ExecutePlugin] Error: %v", err)
			}
		}()

	default:
		return "", fmt.Errorf("unsupported runtime: %s", plugin.Config.Runtime)
	}

	return jobID, nil
}

func (a *App) GetPluginsV2() []*models.PluginV2 {
	return a.pluginLoaderV2.GetAllPlugins()
}

func (a *App) GetPluginV2(id uint) (*models.PluginV2, error) {
	return a.pluginLoaderV2.GetPlugin(id)
}

func (a *App) ExecutePluginV2(req models.PluginExecutionRequestV2) (string, error) {
	log.Printf("[ExecutePluginV2] Starting execution for plugin ID: %d", req.PluginID)
	plugin, err := a.pluginLoaderV2.GetPlugin(req.PluginID)
	if err != nil {
		log.Printf("[ExecutePluginV2] Failed to get plugin: %v", err)
		return "", err
	}
	log.Printf("[ExecutePluginV2] Plugin loaded: %s [ID:%d] (%s), Runtime type: %s",
		plugin.Definition.Plugin.Name, plugin.ID, plugin.Definition.Plugin.ID, plugin.Definition.Runtime.Type)

	if err := a.pluginExecutor.ValidateParameters(plugin, req.Parameters); err != nil {
		return "", fmt.Errorf("parameter validation failed: %w", err)
	}

	args, err := a.pluginExecutor.BuildArguments(plugin, req.Parameters)
	if err != nil {
		return "", fmt.Errorf("failed to build arguments: %w", err)
	}

	cfg := a.settings.GetConfig()
	baseOutputDir := cfg.OutputDirectory
	if baseOutputDir == "" {
		baseOutputDir = "outputs"
	}

	outputDir := filepath.Join(baseOutputDir, fmt.Sprintf("%s_%s",
		plugin.Definition.Plugin.ID,
		time.Now().Format("20060102_150405")))
	os.MkdirAll(outputDir, 0755)

	log.Printf("[ExecutePluginV2] Args before outputDir append: %v", args)
	if plugin.Definition.Execution.OutputDir != "" {
		args = append(args, plugin.Definition.Execution.OutputDir, outputDir)
		log.Printf("[ExecutePluginV2] Args after outputDir append: %v", args)
	}

	parameters := make(map[string]interface{})
	for k, v := range req.Parameters {
		parameters[k] = v
	}
	parameters["outputDir"] = outputDir
	parameters["pluginId"] = plugin.ID

	jobID, err := a.jobQueue.CreateJobWithParameters(
		plugin.Definition.Plugin.ID,
		plugin.Definition.Plugin.Name,
		plugin.Definition.Runtime.Type,
		args,
		parameters,
	)
	if err != nil {
		return "", err
	}

	go func() {
		jobCtx, cancel := context.WithCancel(a.ctx)
		defer cancel()

		a.jobQueue.RegisterJobCancelFunc(jobID, cancel)
		defer a.jobQueue.UnregisterJobCancelFunc(jobID)

		job, err := a.jobQueue.GetJob(jobID)
		if err == nil {
			now := time.Now()
			job.StartedAt = &now
			job.Status = models.JobStatusInProgress
			a.db.GetDB().Save(job)
			runtime.EventsEmit(a.ctx, "job:update", job)
		}

		var execErr error
		log.Printf("[ExecutePluginV2] Plugin runtime type: %s", plugin.Definition.Runtime.Type)
		log.Printf("[ExecutePluginV2] Plugin folder path: %s", plugin.FolderPath)

		switch plugin.Definition.Runtime.Type {
		case "python", "pythonWithR":
			log.Printf("[ExecutePluginV2] Calling ExecutePythonScript with RuntimeType: %s", plugin.Definition.Runtime.Type)
			execErr = a.scriptExecutor.ExecutePythonScript(jobCtx, jobID, services.ScriptConfig{
				PluginID:    plugin.ID,
				Type:        plugin.Definition.Plugin.ID,
				RuntimeType: plugin.Definition.Runtime.Type,
				ScriptName:  filepath.Base(plugin.ScriptPath),
				Args:        args[1:],
				OutputDir:   outputDir,
				FolderPath:  plugin.FolderPath,
			})
		case "r":
			execErr = a.scriptExecutor.ExecuteRScript(jobCtx, jobID, services.ScriptConfig{
				PluginID:   plugin.ID,
				Type:       plugin.Definition.Plugin.ID,
				ScriptName: filepath.Base(plugin.ScriptPath),
				Args:       args[1:],
				OutputDir:  outputDir,
				FolderPath: plugin.FolderPath,
			})
		}

		if execErr != nil {
			log.Printf("[ExecutePluginV2] Error: %v", execErr)
		}
	}()

	return jobID, nil
}

func (a *App) ReloadPluginsV2() error {
	return a.pluginLoaderV2.ReloadPlugins()
}

func (a *App) SavePluginYAML(pluginID string, yamlContent string) error {
	pluginsDir := a.pluginLoaderV2.GetPluginsDirectory()
	pluginDir := filepath.Join(pluginsDir, pluginID)

	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %v", err)
	}

	yamlPath := filepath.Join(pluginDir, "plugin.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		return fmt.Errorf("failed to write plugin.yaml: %v", err)
	}

	log.Printf("[SavePluginYAML] Saved plugin: %s", pluginID)

	return a.pluginLoaderV2.ReloadPlugins()
}

func (a *App) ValidatePluginYAML(yamlContent string) (bool, []string, error) {
	var definition models.PluginDefinition
	if err := yaml.Unmarshal([]byte(yamlContent), &definition); err != nil {
		return false, []string{fmt.Sprintf("YAML parsing error: %v", err)}, nil
	}

	validator := services.NewPluginValidator()
	valid, errors := validator.ValidateDefinition(&definition)

	return valid, errors, nil
}

func (a *App) ConvertPluginToYAML(definition models.PluginDefinition) (string, error) {
	data, err := yaml.Marshal(definition)
	if err != nil {
		return "", fmt.Errorf("failed to marshal plugin definition: %v", err)
	}

	return string(data), nil
}

func (a *App) ParsePluginYAML(yamlContent string) (models.PluginDefinition, error) {
	var definition models.PluginDefinition
	if err := yaml.Unmarshal([]byte(yamlContent), &definition); err != nil {
		return definition, fmt.Errorf("failed to parse YAML: %v", err)
	}

	return definition, nil
}

func (a *App) GetPluginTemplates() ([]*models.PluginV2, error) {
	allPlugins := a.pluginLoaderV2.GetAllPlugins()
	return allPlugins, nil
}

func (a *App) DeletePlugin(pluginID string) error {
	pluginsDir := a.pluginLoaderV2.GetPluginsDirectory()
	pluginDir := filepath.Join(pluginsDir, pluginID)

	if err := os.RemoveAll(pluginDir); err != nil {
		return fmt.Errorf("failed to delete plugin directory: %v", err)
	}

	log.Printf("[DeletePlugin] Deleted plugin: %s", pluginID)

	return a.pluginLoaderV2.ReloadPlugins()
}

func (a *App) SaveTempFile(filename string, content string) (string, error) {
	tempDir := os.TempDir()
	filePath := filepath.Join(tempDir, "cauldron_temp", filename)

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write temp file: %v", err)
	}

	log.Printf("[SaveTempFile] Saved temp file: %s", filePath)
	return filePath, nil
}

func (a *App) LogToFile(message string) error {
	log.Printf("[Frontend] %s", message)
	return nil
}

func (a *App) GetLogFilePath() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %v", err)
	}

	logDir := filepath.Join(userConfigDir, "cauldron")
	today := time.Now().Format("2006-01-02")
	logFileName := fmt.Sprintf("cauldron-%s.log", today)
	logFilePath := filepath.Join(logDir, logFileName)

	return logFilePath, nil
}

func (a *App) OpenLogFile() error {
	logFilePath, err := a.GetLogFilePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist: %s", logFilePath)
	}

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", logFilePath)
	case "darwin":
		cmd = exec.Command("open", logFilePath)
	case "linux":
		cmd = exec.Command("xdg-open", logFilePath)
	default:
		return fmt.Errorf("unsupported operating system: %s", goruntime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	log.Printf("[OpenLogFile] Opened log file: %s", logFilePath)
	return nil
}

func (a *App) OpenLogDirectory() error {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get user config dir: %v", err)
	}

	logDir := filepath.Join(userConfigDir, "cauldron")

	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		return fmt.Errorf("log directory does not exist: %s", logDir)
	}

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", logDir)
	case "darwin":
		cmd = exec.Command("open", logDir)
	case "linux":
		cmd = exec.Command("xdg-open", logDir)
	default:
		return fmt.Errorf("unsupported operating system: %s", goruntime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open log directory: %v", err)
	}

	log.Printf("[OpenLogDirectory] Opened log directory: %s", logDir)
	return nil
}

func (a *App) HandleQuit() {
	log.Println("[HandleQuit] Quit requested from menu")
	shouldPreventClose := a.beforeClose(a.ctx)
	if !shouldPreventClose {
		runtime.Quit(a.ctx)
	}
}

func (a *App) PauseJobQueue() error {
	if a.jobQueue == nil {
		return fmt.Errorf("job queue not initialized")
	}
	return a.jobQueue.PauseQueue()
}

func (a *App) StopJobQueueImmediate() error {
	if a.jobQueue == nil {
		return fmt.Errorf("job queue not initialized")
	}
	return a.jobQueue.StopQueueImmediate()
}

func (a *App) ResumeJobQueue() error {
	if a.jobQueue == nil {
		return fmt.Errorf("job queue not initialized")
	}
	return a.jobQueue.ResumeQueue()
}

func (a *App) GetJobQueueStatus() map[string]interface{} {
	if a.jobQueue == nil {
		return map[string]interface{}{
			"error": "job queue not initialized",
		}
	}
	return a.jobQueue.GetQueueStatus()
}

func (a *App) HasInProgressJobs() bool {
	if a.jobQueue == nil {
		log.Println("[HasInProgressJobs] jobQueue is nil")
		return false
	}
	jobs := a.jobQueue.GetJobsByStatus(models.JobStatusInProgress)
	log.Printf("[HasInProgressJobs] Found %d in-progress jobs", len(jobs))
	if len(jobs) > 0 {
		for _, job := range jobs {
			log.Printf("[HasInProgressJobs] Job ID: %s, Name: %s, Status: %s", job.ID, job.Name, job.Status)
		}
	}
	return len(jobs) > 0
}

func (a *App) checkUnfinishedJobs() {
	time.Sleep(1 * time.Second)

	if a.jobQueue == nil || a.db == nil {
		return
	}

	var unfinishedJobs []*models.Job
	a.db.GetDB().Where("status IN ?", []models.JobStatus{models.JobStatusPending, models.JobStatusInProgress}).
		Order("created_at DESC").
		Find(&unfinishedJobs)

	if len(unfinishedJobs) == 0 {
		log.Println("[checkUnfinishedJobs] No unfinished jobs found")
		return
	}

	log.Printf("[checkUnfinishedJobs] Found %d unfinished job(s)", len(unfinishedJobs))

	selection, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         "Unfinished Jobs Found",
		Message:       fmt.Sprintf("Found %d unfinished job(s) from previous session. Would you like to restart them?", len(unfinishedJobs)),
		Buttons:       []string{"Restart Jobs", "Mark as Failed", "Leave as Is"},
		DefaultButton: "Restart Jobs",
	})

	if err != nil {
		log.Printf("[checkUnfinishedJobs] Error showing dialog: %v", err)
		return
	}

	log.Printf("[checkUnfinishedJobs] User selected: %s", selection)

	switch selection {
	case "Restart Jobs":
		for _, job := range unfinishedJobs {
			job.Status = models.JobStatusPending
			job.Progress = 0
			job.Error = ""
			job.StartedAt = nil
			job.CompletedAt = nil
			job.TerminalOutput = []string{}

			if err := a.db.GetDB().Save(job).Error; err != nil {
				log.Printf("[checkUnfinishedJobs] Failed to reset job %s: %v", job.ID, err)
				continue
			}

			a.jobQueue.RequeueJob(job)
			runtime.EventsEmit(a.ctx, "job:update", job)
			log.Printf("[checkUnfinishedJobs] Restarted job: %s - %s", job.ID, job.Name)
		}

	case "Mark as Failed":
		now := time.Now()
		for _, job := range unfinishedJobs {
			job.Status = models.JobStatusFailed
			job.Error = "Job was interrupted by application shutdown"
			job.CompletedAt = &now

			if err := a.db.GetDB().Save(job).Error; err != nil {
				log.Printf("[checkUnfinishedJobs] Failed to mark job %s as failed: %v", job.ID, err)
				continue
			}

			runtime.EventsEmit(a.ctx, "job:update", job)
			log.Printf("[checkUnfinishedJobs] Marked job as failed: %s - %s", job.ID, job.Name)
		}

	case "Leave as Is":
		log.Println("[checkUnfinishedJobs] Leaving jobs as is")
	}
}

func (a *App) beforeClose(ctx context.Context) bool {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[beforeClose] PANIC recovered: %v", r)
		}
		log.Println("[beforeClose] Function exiting")
	}()

	log.Println("[beforeClose] ========== CLOSE REQUESTED ==========")
	log.Printf("[beforeClose] Context: %v", ctx)
	log.Printf("[beforeClose] App: %v", a)

	if a == nil {
		log.Println("[beforeClose] ERROR: App is nil!")
		return false
	}

	if a.jobQueue == nil {
		log.Println("[beforeClose] Job queue is nil, allowing close")
		return false
	}

	hasInProgress := a.HasInProgressJobs()
	log.Printf("[beforeClose] Has in-progress jobs: %v", hasInProgress)

	if hasInProgress {
		log.Println("[beforeClose] Showing jobs in progress dialog")
		selection, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.QuestionDialog,
			Title:   "Jobs in Progress",
			Message: "There are jobs still running.\n\nClick YES to stop all jobs and exit.\nClick NO to cancel and keep jobs running.",
		})

		if err != nil {
			log.Printf("[beforeClose] Error showing dialog: %v", err)
			return false
		}

		log.Printf("[beforeClose] User selected: %s", selection)

		if selection == "Yes" {
			log.Println("[beforeClose] User chose to stop jobs and exit")
			if a.jobQueue != nil {
				a.jobQueue.StopQueueImmediate()
				time.Sleep(2 * time.Second)
				log.Println("[beforeClose] Waited 2 seconds for jobs to terminate")
			}
			return false
		}

		log.Println("[beforeClose] User chose to cancel close")
		return true
	}

	log.Println("[beforeClose] No in-progress jobs, allowing close")
	return false
}

func (a *App) InstallPluginFromRepo(repoURL string, commitHash string) error {
	log.Printf("[App] Installing plugin from repository: %s (ref: %s)", repoURL, commitHash)
	return a.pluginInstaller.InstallPlugin(repoURL, commitHash, func(status string) {
		runtime.EventsEmit(a.ctx, "plugin:install:progress", map[string]string{
			"repo":   repoURL,
			"status": status,
		})
	})
}

func (a *App) UpdatePluginFromRepo(repoURL string) error {
	log.Printf("[App] Updating plugin from repository: %s", repoURL)
	return a.pluginInstaller.UpdatePlugin(repoURL)
}

func (a *App) UninstallPluginFromRepo(repoURL string) error {
	log.Printf("[App] Uninstalling plugin from repository: %s", repoURL)
	return a.pluginInstaller.UninstallPlugin(repoURL)
}

func (a *App) IsPluginInstalled(repoURL string) (bool, error) {
	return a.pluginInstaller.IsPluginInstalled(repoURL)
}

func (a *App) GetPluginVersion(repoURL string) (string, error) {
	return a.pluginInstaller.GetInstalledVersion(repoURL)
}

func (a *App) DecodePluginRepoURL(encoded string) (string, error) {
	return services.DecodeRepoURL(encoded)
}

func (a *App) ConfirmPluginInstallation(repoURL string, commitHash string) error {
	log.Printf("[App] User confirmed plugin installation from: %s (ref: %s)", repoURL, commitHash)

	runtime.EventsEmit(a.ctx, "plugin:install:start", map[string]interface{}{
		"repo": repoURL,
		"ref":  commitHash,
	})

	go func() {
		if err := a.pluginInstaller.InstallPlugin(repoURL, commitHash, func(status string) {
			runtime.EventsEmit(a.ctx, "plugin:install:progress", map[string]string{
				"repo":   repoURL,
				"status": status,
			})
		}); err != nil {
			log.Printf("[App] Plugin installation failed: %v", err)
			runtime.EventsEmit(a.ctx, "plugin:install:error", map[string]interface{}{
				"repo":  repoURL,
				"error": err.Error(),
			})
			return
		}

		log.Printf("[App] Plugin installed successfully from: %s", repoURL)
		runtime.EventsEmit(a.ctx, "plugin:install:success", map[string]interface{}{
			"repo": repoURL,
		})
	}()

	return nil
}

func (a *App) handleProtocolURL() {
	if len(os.Args) < 2 {
		return
	}

	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "cauldron://") {
			log.Printf("[App] Detected protocol URL: %s", arg)
			go func(url string) {
				time.Sleep(2 * time.Second)
				if err := a.HandleProtocolURL(url); err != nil {
					log.Printf("[App] Error handling protocol URL: %v", err)
					runtime.EventsEmit(a.ctx, "protocol:error", map[string]string{
						"url":   url,
						"error": err.Error(),
					})
				} else {
					runtime.EventsEmit(a.ctx, "protocol:success", map[string]string{
						"url": url,
					})
				}
			}(arg)
		}
	}
}

func (a *App) HandleProtocolURL(url string) error {
	log.Printf("[App] Handling protocol URL: %s", url)

	if a.protocolHandler == nil {
		return fmt.Errorf("protocol handler not initialized")
	}

	if err := a.protocolHandler.HandleURL(url); err != nil {
		return err
	}

	runtime.EventsEmit(a.ctx, "plugin:installed", nil)
	return nil
}

func (a *App) ListRegistryPlugins(searchQuery string, categoryName string, authorName string, limit int, offset int) (interface{}, error) {
	log.Printf("[App] Listing registry plugins - search: %s, category: %s, author: %s, limit: %d, offset: %d", searchQuery, categoryName, authorName, limit, offset)
	return a.pluginRegistryService.ListPlugins(searchQuery, categoryName, authorName, limit, offset)
}

func (a *App) GetRegistryPlugin(pluginID string) (interface{}, error) {
	log.Printf("[App] Getting registry plugin: %s", pluginID)
	return a.pluginRegistryService.GetPlugin(pluginID)
}

func (a *App) ListRegistryCategories() (interface{}, error) {
	log.Printf("[App] Listing registry categories")
	return a.pluginRegistryService.ListCategories()
}

func (a *App) InstallPluginFromRegistry(pluginID string, commitHash string) error {
	log.Printf("[App] Installing plugin from registry: %s (ref: %s)", pluginID, commitHash)

	plugin, err := a.pluginRegistryService.GetPlugin(pluginID)
	if err != nil {
		return err
	}

	if plugin.Repository == "" {
		return fmt.Errorf("plugin %s does not have a repository URL", pluginID)
	}

	return a.ConfirmPluginInstallation(plugin.Repository, commitHash)
}
