package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/backend/services"
	"github.com/noatgnu/cookeR/rversion"
	"github.com/wailsapp/wails/v3/pkg/application"
	"gopkg.in/yaml.v3"
)

//go:embed resources/licenses/go-licenses.json
var goLicensesJSON []byte

//go:embed resources/licenses/npm-licenses.json
var npmLicensesJSON []byte

type App struct {
	wailsApp               *application.App
	mainWindow             *application.WebviewWindow
	db                     *services.DatabaseService
	settings               *services.SettingsService
	fileService            *services.FileService
	jobQueue               *services.JobQueueService
	envService             *services.EnvironmentService
	scriptExecutor         *services.ScriptExecutor
	portableEnvService     *services.PortableEnvService
	uvService              *services.UvService
	rPortableService       *services.RPortableService
	pluginService          *services.PluginService
	pluginLoaderV2         *services.PluginLoaderV2
	pluginExecutor         *services.PluginExecutor
	pluginInstaller        *services.PluginInstaller
	protocolHandler        *services.ProtocolHandler
	httpInstallServer      *services.HTTPInstallServer
	pluginRegistryService  *services.PluginRegistryService
	gitAuthService         *services.GitAuthService
	backupService          *services.BackupService
	pluginMigrationService *services.PluginMigrationService
	parquetService         *services.ParquetService
	delimitedFileService   *services.DelimitedFileService
	tableFileService       *services.TableFileService
	updateCheckService     *services.UpdateCheckService
	ready                  chan bool
	logFilePath            string
	appVersion             string
	initialized            chan struct{}
}

func NewApp() *App {
	return &App{
		ready:       make(chan bool),
		initialized: make(chan struct{}),
	}
}

func (a *App) SetApplication(wailsApp *application.App) {
	a.wailsApp = wailsApp
}

func (a *App) SetMainWindow(window *application.WebviewWindow) {
	a.mainWindow = window
}

func (a *App) SetVersion(version string) {
	a.appVersion = version
}

func (a *App) emitEvent(name string, data interface{}) {
	if a.wailsApp != nil && a.wailsApp.Event != nil {
		a.wailsApp.Event.Emit(name, data)
	}
}

func (a *App) Initialize() {
	log.Println("[App.Initialize] Starting application...")
	// GetPluginsV2 (the frontend's backend-ready gate) blocks on <-a.ready; close it on every exit path, including early returns and panics.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[App.Initialize] PANIC: %v\n%s", r, debug.Stack())
		}
		close(a.ready)
		close(a.initialized)
	}()

	// Fixed-delay heuristic (not a real readiness signal) giving the webview time to register its event listeners before Initialize emits any.
	time.Sleep(1 * time.Second)

	log.Println("[App.Initialize] Initializing database...")
	userDataPath, err := getUserDataPath()
	if err != nil {
		log.Printf("[App.Initialize] ERROR: Failed to get user data path: %v\n", err)
		return
	}

	db, err := services.NewDatabaseServiceV3(userDataPath)
	if err != nil {
		log.Printf("[App.Initialize] ERROR: Failed to initialize database: %v\n", err)
		fmt.Printf("[App.Initialize] ERROR: Failed to initialize database: %v\n", err)
		return
	}
	a.db = db
	log.Println("[App.Initialize] Database initialized successfully")

	log.Println("[App.Initialize] Initializing services...")
	a.settings = services.NewSettingsServiceV3(db)
	a.fileService = services.NewFileServiceV3(a.wailsApp)
	a.envService = services.NewEnvironmentServiceV3(db, a.settings, services.NewProgressNotifierV3(a.wailsApp))
	a.portableEnvService = services.NewPortableEnvServiceV3(a.fileService, a.wailsApp)
	a.uvService = services.NewUvServiceV3(a.fileService, db, a.settings, a.wailsApp)
	a.pluginMigrationService = services.NewPluginMigrationService(db)
	tableProgressNotifier := services.NewProgressNotifierV3(a.wailsApp)
	a.parquetService = services.NewParquetService(tableProgressNotifier)
	a.delimitedFileService = services.NewDelimitedFileService(tableProgressNotifier)
	a.tableFileService = services.NewTableFileService(a.parquetService, a.delimitedFileService)
	a.updateCheckService = services.NewUpdateCheckService(a.appVersion)
	a.rPortableService, err = services.NewRPortableServiceV3(a.wailsApp)
	if err != nil {
		log.Printf("[App.Initialize] ERROR: Failed to initialize R portable service: %v\n", err)
		return
	}

	log.Println("[App.Initialize] Initializing job queue...")
	a.jobQueue = services.NewJobQueueServiceV3(db, a.wailsApp)
	log.Println("[App.Initialize] Setting job queue runners...")

	log.Println("[App.Initialize] Initializing script executor...")
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

		a.emitEvent("job:update", job)
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

		a.emitEvent("job:output", map[string]interface{}{
			"jobId":  jobID,
			"output": line,
		})
	})

	log.Println("[App.Initialize] Initializing plugin service...")
	a.pluginService = services.NewPluginService()

	log.Println("[App.Initialize] Initializing Docker image builder...")
	dockerImageBuilder := services.NewDockerImageBuilder(a.db)
	if err := dockerImageBuilder.CheckDockerAvailable(); err != nil {
		log.Printf("[App.Initialize] Warning: Docker daemon not available. Docker-based plugins will not work: %v", err)
	} else {
		log.Println("[App.Initialize] Docker is available")
	}

	log.Println("[App.Initialize] Initializing plugin system V2...")
	a.pluginLoaderV2 = services.NewPluginLoaderV2("", a.db, dockerImageBuilder)
	if err := a.pluginLoaderV2.LoadPlugins(); err != nil {
		log.Printf("[App.Initialize] Failed to load plugins: %v", err)
	}
	a.pluginExecutor = services.NewPluginExecutor()

	log.Println("[App.Initialize] Wiring up job queue with script executor and plugin loader...")
	a.scriptExecutor.SetPluginLoader(a.pluginLoaderV2)
	a.jobQueue.SetScriptExecutor(a.scriptExecutor)
	a.jobQueue.SetPluginLoader(a.pluginLoaderV2)

	log.Println("[App.Initialize] Initializing Git authentication service...")
	a.gitAuthService = services.NewGitAuthService(a.db)
	log.Println("[App.Initialize] Git authentication service initialized")

	a.backupService = services.NewBackupService(a.db)

	exePath, _ := os.Executable()
	pluginsDir := filepath.Join(filepath.Dir(exePath), "plugins")
	a.pluginInstaller = services.NewPluginInstallerV3(pluginsDir, a.db, a.pluginLoaderV2, a.gitAuthService, a.wailsApp)
	log.Println("[App.Initialize] Plugin installer initialized")

	log.Println("[App.Initialize] Initializing plugin registry service...")
	a.pluginRegistryService = services.NewPluginRegistryServiceV3(a.settings, a.gitAuthService)
	log.Println("[App.Initialize] Plugin registry service initialized")

	log.Println("[App.Initialize] Initializing protocol handler...")
	a.protocolHandler = services.NewProtocolHandlerV3(a.pluginInstaller, a.wailsApp)
	if err := a.protocolHandler.RegisterProtocol(); err != nil {
		log.Printf("[App.Initialize] Warning: Failed to register protocol handler: %v", err)
	}
	log.Println("[App.Initialize] Protocol handler initialized")

	log.Println("[App.Initialize] Starting HTTP install server...")
	a.httpInstallServer = services.NewHTTPInstallServer(a.protocolHandler)
	a.httpInstallServer.SetWailsApp(a.wailsApp)
	ctx := context.Background()
	if err := a.httpInstallServer.Start(ctx); err != nil {
		log.Printf("[App.Initialize] Warning: Failed to start HTTP install server: %v", err)
	} else {
		log.Printf("[App.Initialize] HTTP install server ready at http://localhost:%d", a.httpInstallServer.GetPort())
	}

	log.Println("[App.Initialize] Checking for protocol URL...")
	a.handleProtocolURL()

	log.Println("[App.Initialize] Plugin system V2 initialized")

	log.Println("[App.Initialize] Checking for unfinished jobs...")
	go a.checkUnfinishedJobs()

	log.Println("[App.Initialize] Application initialization complete!")
}

func getUserDataPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dataPath := filepath.Join(configDir, "cauldron")
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return "", err
	}
	return dataPath, nil
}

func (a *App) Shutdown() {
	log.Println("[App.Shutdown] Starting shutdown...")

	if a.httpInstallServer != nil {
		log.Println("[App.Shutdown] Stopping HTTP install server...")
		if err := a.httpInstallServer.Stop(); err != nil {
			log.Printf("[App.Shutdown] Error stopping HTTP install server: %v", err)
		}
	}
	if a.jobQueue != nil {
		a.jobQueue.Shutdown()
	}
	if a.db != nil {
		a.db.Close()
	}

	log.Println("[App.Shutdown] Shutdown complete")
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

// CreateSettingsBackup writes settings + installed-plugin metadata to path. includeSecrets also backs up custom environment variables, which may hold plugin API keys in cleartext.
func (a *App) CreateSettingsBackup(path string, includeSecrets bool) (*services.BackupSummary, error) {
	data, err := a.backupService.CreateBackup(includeSecrets)
	if err != nil {
		return nil, err
	}
	if err := services.WriteBackupFile(path, data); err != nil {
		return nil, err
	}
	summary := data.Summary()
	return &summary, nil
}

// PreviewSettingsBackup reads a backup file's counts (never secret values) so the UI can confirm before restoring.
func (a *App) PreviewSettingsBackup(path string) (*services.BackupSummary, error) {
	data, err := services.ReadBackupFile(path)
	if err != nil {
		return nil, err
	}
	summary := data.Summary()
	return &summary, nil
}

// RestoreSettingsBackup applies settings, reinstalls missing remote plugins, and restores enabled state + env vars from a backup file.
func (a *App) RestoreSettingsBackup(path string) (*services.RestoreResult, error) {
	data, err := services.ReadBackupFile(path)
	if err != nil {
		return nil, err
	}
	return a.backupService.RestoreBackup(data, a.pluginInstaller, nil)
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

func (a *App) GetPythonVersion() (string, error) {
	cfg := a.settings.GetConfig()
	cmd := exec.Command(cfg.PythonPath, "--version")
	hideConsoleWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (a *App) GetRVersion() (string, error) {
	cfg := a.settings.GetConfig()
	cmd := exec.Command(cfg.RPath, "--version")
	hideConsoleWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}
	return strings.TrimSpace(string(output)), nil
}

func (a *App) CheckDockerVersion() (string, error) {
	cmd := exec.Command("docker", "--version")
	hideConsoleWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker not available: %w", err)
	}
	version := strings.TrimSpace(string(output))
	version = strings.TrimPrefix(version, "Docker version ")
	if idx := strings.Index(version, ","); idx != -1 {
		version = version[:idx]
	}
	return version, nil
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) DetectPythonEnvironments() ([]services.PythonEnvironment, error) {
	envs, err := a.envService.DetectPythonEnvironments()
	log.Printf("[App] DetectPythonEnvironments called - returning %d environments, error: %v", len(envs), err)
	for i, env := range envs {
		log.Printf("[App] [%d] Name=%s, Path=%s, Type=%s, IsVirtual=%v", i, env.Name, env.Path, env.Type, env.IsVirtual)
	}
	return envs, err
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

// resolvePluginFolderPath looks up a plugin's folder path by its string or numeric ID, logging (not failing) on a miss.
func (a *App) resolvePluginFolderPath(pluginID, logPrefix string) string {
	if pluginID == "" {
		return ""
	}

	plugin, err := a.pluginLoaderV2.GetPluginByStringID(pluginID)
	if err != nil {
		if id, convErr := strconv.ParseUint(pluginID, 10, 64); convErr == nil {
			plugin, err = a.pluginLoaderV2.GetPlugin(uint(id))
		}
	}

	if err != nil {
		log.Printf("[App] %s: Failed to get plugin folder path: %v", logPrefix, err)
		return ""
	}

	log.Printf("[App] %s: Found plugin folder path: %s", logPrefix, plugin.FolderPath)
	return plugin.FolderPath
}

func (a *App) CreatePythonVirtualEnv(basePythonPath string, venvPath string, pluginID string) error {
	pluginFolderPath := a.resolvePluginFolderPath(pluginID, "CreatePythonVirtualEnv")
	return a.envService.CreatePythonVirtualEnv(basePythonPath, venvPath, pluginID, pluginFolderPath)
}

func (a *App) GetVirtualEnvironments() ([]services.VirtualEnvironment, error) {
	venvs, err := a.envService.GetVirtualEnvironments()
	log.Printf("[App] GetVirtualEnvironments called - returning %d venvs, error: %v", len(venvs), err)
	return venvs, err
}

func (a *App) GetDefaultVenvPath(pluginID string) (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %w", err)
	}

	venvBaseDir := filepath.Join(userConfigDir, "cauldron", "venvs")
	if err := os.MkdirAll(venvBaseDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create venv base directory: %w", err)
	}

	venvPath := filepath.Join(venvBaseDir, fmt.Sprintf("venv-%s", pluginID))
	return venvPath, nil
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

// resolvePluginMigrationTargets loads the registry row and the loaded plugin's current schemaVersion for pluginID.
func (a *App) resolvePluginMigrationTargets(pluginID uint) (*models.PluginRegistry, int, error) {
	registry, err := a.db.GetPluginRegistryByID(pluginID)
	if err != nil {
		return nil, 0, err
	}
	if registry == nil {
		return nil, 0, fmt.Errorf("plugin not found with ID: %d", pluginID)
	}
	plugin, err := a.pluginLoaderV2.GetPlugin(pluginID)
	if err != nil {
		return nil, 0, err
	}
	return registry, plugin.Definition.Plugin.SchemaVersion, nil
}

// GetPendingEnvVarMigration is a read-only check for which saved custom env var keys a plugin update renamed or removed.
func (a *App) GetPendingEnvVarMigration(pluginID uint) (*services.PendingEnvVarMigration, error) {
	registry, currentSchemaVersion, err := a.resolvePluginMigrationTargets(pluginID)
	if err != nil {
		return nil, err
	}
	return a.pluginMigrationService.DetectPendingEnvVarMigration(registry, currentSchemaVersion)
}

// ApplyPendingEnvVarMigration reconciles saved custom env var keys, only in response to an explicit user action; confirmedLarge must be true to apply an unusually large pending migration.
func (a *App) ApplyPendingEnvVarMigration(pluginID uint, confirmedLarge bool) error {
	registry, currentSchemaVersion, err := a.resolvePluginMigrationTargets(pluginID)
	if err != nil {
		return err
	}
	return a.pluginMigrationService.ApplyPendingEnvVarMigration(registry, currentSchemaVersion, confirmedLarge)
}

// OpenTableFileDialog shows a native file picker restricted to .parquet/.csv/.tsv files.
func (a *App) OpenTableFileDialog() (string, error) {
	return a.fileService.OpenFileDialog("Select File", []application.FileFilter{
		{DisplayName: "Table Files (*.parquet, *.csv, *.tsv)", Pattern: "*.parquet;*.csv;*.tsv"},
		{DisplayName: "Parquet Files (*.parquet)", Pattern: "*.parquet"},
		{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
		{DisplayName: "TSV Files (*.tsv)", Pattern: "*.tsv"},
	})
}

func (a *App) GetTableFileInfo(path string) (*services.DataFileInfo, error) {
	return a.tableFileService.OpenFile(path)
}

func (a *App) GetTableFilePage(path string, offset int, limit int) ([]map[string]interface{}, error) {
	return a.tableFileService.ReadPage(path, offset, limit)
}

// SaveTableExportDialog shows a native save-file picker for the CSV/TSV export destination.
func (a *App) SaveTableExportDialog(defaultName string) (string, error) {
	return a.fileService.SaveFileDialog("Export Table Data", defaultName)
}

func (a *App) ExportTableFile(path string, outputPath string, columns []string, delimiter string) error {
	d := ','
	if delimiter != "" {
		d = rune(delimiter[0])
	}
	return a.tableFileService.ExportFile(path, outputPath, columns, d)
}

func (a *App) CloseTableFile(path string) error {
	return a.tableFileService.CloseFile(path)
}

// GetAppVersion returns the running app's version, embedded at build time.
func (a *App) GetAppVersion() string {
	return a.appVersion
}

// CheckForUpdate checks GitHub for a newer release than the running app version.
func (a *App) CheckForUpdate() (*services.UpdateInfo, error) {
	return a.updateCheckService.CheckForUpdate()
}

type GitAuthConfigResponse struct {
	ID            uint   `json:"id"`
	RepositoryURL string `json:"repositoryURL"`
	SSHKeyPath    string `json:"sshKeyPath"`
	HasPassphrase bool   `json:"hasPassphrase"`
	CreatedAt     int64  `json:"createdAt"`
	UpdatedAt     int64  `json:"updatedAt"`
}

func (a *App) SaveGitAuthConfig(repoURL string, sshKeyPath string, passphrase string) error {
	return a.gitAuthService.SaveGitAuthConfig(repoURL, sshKeyPath, passphrase)
}

func (a *App) GetGitAuthConfig(repoURL string) (*GitAuthConfigResponse, error) {
	config, err := a.gitAuthService.GetGitAuthConfig(repoURL)
	if err != nil {
		return nil, err
	}

	return &GitAuthConfigResponse{
		ID:            config.ID,
		RepositoryURL: config.RepositoryURL,
		SSHKeyPath:    config.SSHKeyPath,
		HasPassphrase: config.SSHKeyPassphrase != "",
		CreatedAt:     config.CreatedAt,
		UpdatedAt:     config.UpdatedAt,
	}, nil
}

func (a *App) GetAllGitAuthConfigs() ([]GitAuthConfigResponse, error) {
	configs, err := a.gitAuthService.GetAllGitAuthConfigs()
	if err != nil {
		return nil, err
	}

	responses := make([]GitAuthConfigResponse, len(configs))
	for i, config := range configs {
		responses[i] = GitAuthConfigResponse{
			ID:            config.ID,
			RepositoryURL: config.RepositoryURL,
			SSHKeyPath:    config.SSHKeyPath,
			HasPassphrase: config.SSHKeyPassphrase != "",
			CreatedAt:     config.CreatedAt,
			UpdatedAt:     config.UpdatedAt,
		}
	}

	return responses, nil
}

func (a *App) DeleteGitAuthConfig(repoURL string) error {
	return a.gitAuthService.DeleteGitAuthConfig(repoURL)
}

func (a *App) ValidateSSHKey(keyPath string, passphrase string) error {
	return a.gitAuthService.ValidateSSHKey(keyPath, passphrase)
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

func (a *App) GetPluginExampleFilePath(pluginID string, filePath string) (string, error) {
	plugin, err := a.pluginLoaderV2.GetPluginByStringID(pluginID)
	if err == nil {
		examplePath := filepath.Join(plugin.FolderPath, filePath)
		if _, err := os.Stat(examplePath); err == nil {
			log.Printf("[GetPluginExampleFilePath] Found in plugin folder: %s", examplePath)
			return examplePath, nil
		}
		log.Printf("[GetPluginExampleFilePath] Not found in plugin folder: %s", examplePath)
	} else {
		log.Printf("[GetPluginExampleFilePath] Plugin not found: %s", pluginID)
	}

	normalizedPath := filepath.ToSlash(filePath)
	parts := strings.Split(normalizedPath, "/")
	if len(parts) >= 2 {
		exampleType := parts[0]
		fileName := filepath.Join(parts[1:]...)
		log.Printf("[GetPluginExampleFilePath] Trying global examples: %s/%s", exampleType, fileName)

		globalPath, err := a.envService.GetExampleFilePath(exampleType, fileName)
		if err == nil {
			log.Printf("[GetPluginExampleFilePath] Found in global examples: %s", globalPath)
			return globalPath, nil
		}
		log.Printf("[GetPluginExampleFilePath] Not found in global examples: %v", err)
	}

	return "", fmt.Errorf("example file not found: %s", filePath)
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

	a.emitEvent("file:imported", importedFile)

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

func (a *App) GetPortableEnvironmentURL(platform, arch, version, environment string) (string, error) {
	return a.portableEnvService.GetPortableEnvironmentURL(platform, arch, version, environment)
}

func (a *App) DownloadPortableEnvironment(url, environment string) error {
	return a.portableEnvService.DownloadPortableEnvironment(url, environment)
}

func (a *App) GetPortableEnvironmentPath(environment string) (string, error) {
	return a.portableEnvService.GetPortableEnvironmentPath(environment)
}

func (a *App) GetUvPath() (string, error) {
	return a.uvService.ResolveUvPath()
}

func (a *App) DownloadUv() error {
	return a.uvService.DownloadUv()
}

func (a *App) IsUvAvailable() bool {
	return a.uvService.IsUvAvailable()
}

func (a *App) ListUvManagedPythons() ([]services.UvPythonVersion, error) {
	return a.uvService.ListUvManagedPythons()
}

func (a *App) InstallUvPythonVersion(version string) error {
	return a.uvService.InstallUvPythonVersion(version)
}

func (a *App) CreateUvVirtualEnv(pythonVersion string, venvPath string, pluginID string) error {
	pluginFolderPath := a.resolvePluginFolderPath(pluginID, "CreateUvVirtualEnv")
	return a.uvService.CreateUvVirtualEnv(pythonVersion, venvPath, pluginID, pluginFolderPath)
}

func (a *App) InstallUvPackages(venvPythonPath string, packages []string) error {
	return a.uvService.InstallUvPackages(venvPythonPath, packages)
}

func (a *App) InstallUvRequirements(venvPythonPath string, requirementsPath string) error {
	return a.uvService.InstallUvRequirements(venvPythonPath, requirementsPath)
}

func (a *App) ListUvPackages(venvPythonPath string) ([]string, error) {
	return a.uvService.ListUvPackages(venvPythonPath)
}

func (a *App) ListAvailableRVersions() ([]rversion.Release, error) {
	return a.rPortableService.ListAvailableRVersions()
}

func (a *App) ListInstalledRVersions() ([]string, error) {
	return a.rPortableService.ListInstalledRVersions()
}

func (a *App) InstallRVersion(version string) error {
	return a.rPortableService.InstallRVersion(version)
}

func (a *App) UninstallRVersion(version string) error {
	return a.rPortableService.UninstallRVersion(version)
}

func (a *App) GetRPortablePath(version string) (string, error) {
	return a.rPortableService.GetRPath(version)
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
		jobID, err = a.jobQueue.CreateJobWithParameters("plugin", jobName, "python", args, parameters, "", "")
		if err != nil {
			return "", err
		}

		go func() {
			ctx := context.Background()
			err := a.scriptExecutor.ExecutePythonScript(ctx, jobID, services.ScriptConfig{
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
		jobID, err = a.jobQueue.CreateJobWithParameters("plugin", jobName, "r", args, parameters, "", "")
		if err != nil {
			return "", err
		}

		go func() {
			ctx := context.Background()
			err := a.scriptExecutor.ExecuteRScript(ctx, jobID, services.ScriptConfig{
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
		jobID, err = a.jobQueue.CreateJobWithParameters("plugin", jobName, "python", args, parameters, "", "")
		if err != nil {
			return "", err
		}

		cfg := a.settings.GetConfig()
		if cfg.RPath != "" {
			args = append(args, "--r_home", cfg.RPath)
		}

		go func() {
			ctx := context.Background()
			err := a.scriptExecutor.ExecutePythonScript(ctx, jobID, services.ScriptConfig{
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
	<-a.ready
	if a.pluginLoaderV2 == nil {
		return nil
	}
	return a.pluginLoaderV2.GetAllPlugins()
}

func (a *App) GetPluginV2(id uint) (*models.PluginV2, error) {
	return a.pluginLoaderV2.GetPlugin(id)
}

func (a *App) SetPluginEnabled(id uint, enabled bool) error {
	return a.pluginLoaderV2.SetPluginEnabled(id, enabled)
}

func (a *App) EnableAllPlugins() error {
	log.Println("[App] Enabling all plugins")
	return a.db.GetDB().Model(&models.PluginRegistry{}).Update("enabled", true).Error
}

func (a *App) ExecutePluginV2(req models.PluginExecutionRequestV2) (string, error) {
	log.Printf("[ExecutePluginV2] Starting execution for plugin ID: %d", req.PluginID)
	plugin, err := a.pluginLoaderV2.GetPlugin(req.PluginID)
	if err != nil {
		log.Printf("[ExecutePluginV2] Failed to get plugin: %v", err)
		return "", err
	}
	log.Printf("[ExecutePluginV2] Plugin loaded: %s [ID:%d] (%s), Environments: %v",
		plugin.Definition.Plugin.Name, plugin.ID, plugin.Definition.Plugin.ID, plugin.Definition.Runtime.GetEnvironments())

	jobID, err := executePluginJob(a.pluginExecutor, a.jobQueue, a.settings, plugin, req.Parameters)
	if err != nil {
		return "", err
	}

	log.Printf("[ExecutePluginV2] Created job %s - will be processed by job queue worker", jobID)
	return jobID, nil
}

// executePluginJob validates, builds args, and enqueues a job; shared between App.ExecutePluginV2 (GUI) and the CLI's "job run" so both stay identical.
func executePluginJob(pluginExecutor *services.PluginExecutor, jobQueue *services.JobQueueService, settings *services.SettingsService, plugin *models.PluginV2, parameters map[string]interface{}) (string, error) {
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

	outputDir := services.GenerateJobOutputDir(baseOutputDir, plugin.Definition.Plugin.ID)
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

	return jobQueue.CreateJobWithParameters(
		plugin.Definition.Plugin.ID,
		plugin.Definition.Plugin.Name,
		runtimeTypeForJob,
		args,
		params,
		plugin.Definition.Plugin.Version,
		plugin.CommitHash,
	)
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

func (a *App) SetLogFilePath(path string) {
	a.logFilePath = path
}

func (a *App) GetLogFilePath() (string, error) {
	log.Printf("[GetLogFilePath] Stored log file path: '%s'", a.logFilePath)

	if a.logFilePath != "" {
		log.Printf("[GetLogFilePath] Returning stored path: %s", a.logFilePath)
		return a.logFilePath, nil
	}

	log.Printf("[GetLogFilePath] No stored path, searching for latest log file")

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("[GetLogFilePath] Failed to get user config dir: %v", err)
		return "", fmt.Errorf("failed to get user config dir: %v", err)
	}

	logDir := filepath.Join(userConfigDir, "cauldron")
	log.Printf("[GetLogFilePath] Log directory: %s", logDir)

	entries, err := os.ReadDir(logDir)
	if err != nil {
		log.Printf("[GetLogFilePath] Failed to read log directory: %v", err)
		return "", fmt.Errorf("failed to read log directory: %v", err)
	}

	var logFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "cauldron-") && strings.HasSuffix(entry.Name(), ".log") {
			logFiles = append(logFiles, entry.Name())
		}
	}

	log.Printf("[GetLogFilePath] Found %d log files: %v", len(logFiles), logFiles)

	if len(logFiles) == 0 {
		log.Printf("[GetLogFilePath] No log files found")
		return "", fmt.Errorf("no log files found in %s", logDir)
	}

	sort.Strings(logFiles)
	latestLogFile := logFiles[len(logFiles)-1]
	logFilePath := filepath.Join(logDir, latestLogFile)

	log.Printf("[GetLogFilePath] Latest log file: %s", logFilePath)

	return logFilePath, nil
}

func (a *App) OpenLogFile() error {
	log.Printf("[OpenLogFile] Starting to open log file")

	logFilePath, err := a.GetLogFilePath()
	if err != nil {
		log.Printf("[OpenLogFile] Error getting log file path: %v", err)
		return err
	}

	log.Printf("[OpenLogFile] Log file path: %s", logFilePath)

	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		log.Printf("[OpenLogFile] Log file does not exist: %s", logFilePath)
		return fmt.Errorf("log file does not exist: %s", logFilePath)
	}

	log.Printf("[OpenLogFile] Log file exists, opening...")

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("notepad.exe", logFilePath)
		log.Printf("[OpenLogFile] Windows command: notepad.exe \"%s\"", logFilePath)
	case "darwin":
		cmd = exec.Command("open", logFilePath)
		log.Printf("[OpenLogFile] macOS command: open \"%s\"", logFilePath)
	case "linux":
		cmd = exec.Command("xdg-open", logFilePath)
		log.Printf("[OpenLogFile] Linux command: xdg-open \"%s\"", logFilePath)
	default:
		err := fmt.Errorf("unsupported operating system: %s", goruntime.GOOS)
		log.Printf("[OpenLogFile] %v", err)
		return err
	}

	if err := cmd.Start(); err != nil {
		log.Printf("[OpenLogFile] Failed to execute command: %v", err)
		return fmt.Errorf("failed to open log file: %v", err)
	}

	log.Printf("[OpenLogFile] Successfully opened log file: %s", logFilePath)
	return nil
}

type LicenseInfo struct {
	Name       string  `json:"name"`
	Version    string  `json:"version"`
	License    string  `json:"license"`
	Repository *string `json:"repository,omitempty"`
}

type LicenseData struct {
	Go  []LicenseInfo `json:"go"`
	NPM []LicenseInfo `json:"npm"`
}

func (a *App) GetLicenseInfo() (LicenseData, error) {
	var result LicenseData

	goLicenses, err := a.getGoLicenses()
	if err != nil {
		log.Printf("[GetLicenseInfo] Failed to get Go licenses: %v", err)
	} else {
		result.Go = goLicenses
	}

	npmLicenses, err := a.getNPMLicenses()
	if err != nil {
		log.Printf("[GetLicenseInfo] Failed to get NPM licenses: %v", err)
	} else {
		result.NPM = npmLicenses
	}

	return result, nil
}

func (a *App) getGoLicenses() ([]LicenseInfo, error) {
	var licenses []LicenseInfo
	if err := json.Unmarshal(goLicensesJSON, &licenses); err != nil {
		log.Printf("[getGoLicenses] Failed to unmarshal Go licenses: %v", err)
		return []LicenseInfo{}, err
	}

	return licenses, nil
}

func (a *App) getNPMLicenses() ([]LicenseInfo, error) {
	var licenses []LicenseInfo
	if err := json.Unmarshal(npmLicensesJSON, &licenses); err != nil {
		log.Printf("[getNPMLicenses] Failed to unmarshal NPM licenses: %v", err)
		return []LicenseInfo{}, err
	}

	return licenses, nil
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
	if a.HasInProgressJobs() {
		log.Println("[HandleQuit] Jobs in progress, stopping...")
		if a.jobQueue != nil {
			a.jobQueue.StopQueueImmediate()
		}
	}
	if a.wailsApp != nil {
		a.wailsApp.Quit()
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

func (a *App) ProcessPendingJobs() error {
	if a.jobQueue == nil {
		return fmt.Errorf("job queue not initialized")
	}
	return a.jobQueue.ProcessPendingJobs()
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
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[checkUnfinishedJobs] PANIC: %v\n%s", r, debug.Stack())
		}
	}()

	// Same fixed-delay heuristic as Initialize's leading sleep, for the webview to register its "unfinished-jobs-found" listener first.
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
	a.emitEvent("unfinished-jobs-found", map[string]interface{}{
		"count": len(unfinishedJobs),
	})
}

type PluginInstallResult struct {
	PluginID string `json:"pluginId"`
}

func (a *App) InstallPluginFromRepo(repoURL string, commitHash string) (*PluginInstallResult, error) {
	log.Printf("[App] Installing plugin from repository: %s (ref: %s)", repoURL, commitHash)
	pluginID, err := a.pluginInstaller.InstallPlugin(repoURL, commitHash, nil, func(status string) {
		a.emitEvent("plugin:install:progress", map[string]string{
			"repo":   repoURL,
			"status": status,
		})
	})
	if err != nil {
		return nil, err
	}
	return &PluginInstallResult{PluginID: pluginID}, nil
}

func (a *App) UpdatePluginFromRepo(repoURL string) error {
	log.Printf("[App] Updating plugin from repository: %s", repoURL)
	return a.pluginInstaller.UpdatePlugin(repoURL)
}

func (a *App) UpdatePluginFromRepoForce(repoURL string, force bool) error {
	log.Printf("[App] Updating plugin from repository: %s (force: %v)", repoURL, force)
	return a.pluginInstaller.UpdatePluginWithForce(repoURL, force)
}

func (a *App) UpdatePluginToCommit(repoURL string, commitHash string) error {
	log.Printf("[App] Updating plugin from repository %s to commit: %s", repoURL, commitHash)
	return a.pluginInstaller.UpdatePluginToCommit(repoURL, commitHash)
}

func (a *App) UpdatePluginToCommitForce(repoURL string, commitHash string, force bool) error {
	log.Printf("[App] Updating plugin from repository %s to commit: %s (force: %v)", repoURL, commitHash, force)
	return a.pluginInstaller.UpdatePluginToCommitWithForce(repoURL, commitHash, force)
}

func (a *App) ReinstallPlugin(repoURL string) error {
	log.Printf("[App] Reinstalling plugin from repository: %s", repoURL)
	return a.pluginInstaller.ReinstallPlugin(repoURL)
}

func (a *App) UpdateAllRemotePlugins() error {
	log.Printf("[App] Updating all remote plugins")

	var registries []models.PluginRegistry
	if err := a.db.GetDB().Where("install_source = ?", "remote").Find(&registries).Error; err != nil {
		return fmt.Errorf("failed to fetch remote plugins: %w", err)
	}

	if len(registries) == 0 {
		log.Printf("[App] No remote plugins found to update")
		return fmt.Errorf("no external plugins installed")
	}

	log.Printf("[App] Found %d remote plugin(s) to update", len(registries))

	var errors []string
	successCount := 0

	for _, registry := range registries {
		if registry.Repository == "" {
			log.Printf("[App] Skipping plugin %s: no repository URL", registry.PluginID)
			continue
		}

		log.Printf("[App] Updating plugin %s from %s", registry.PluginID, registry.Repository)
		if err := a.pluginInstaller.UpdatePlugin(registry.Repository); err != nil {
			errMsg := fmt.Sprintf("Failed to update %s: %v", registry.PluginID, err)
			log.Printf("[App] %s", errMsg)
			errors = append(errors, errMsg)
		} else {
			successCount++
			log.Printf("[App] Successfully updated %s", registry.PluginID)
		}
	}

	log.Printf("[App] Updated %d/%d remote plugins", successCount, len(registries))

	if len(errors) > 0 {
		return fmt.Errorf("some plugins failed to update: %s", strings.Join(errors, "; "))
	}

	return nil
}

func (a *App) ForceUpdateAllRemotePlugins() error {
	log.Printf("[App] Force updating all remote plugins to latest")

	var registries []models.PluginRegistry
	if err := a.db.GetDB().Where("install_source = ?", "remote").Find(&registries).Error; err != nil {
		return fmt.Errorf("failed to fetch remote plugins: %w", err)
	}

	if len(registries) == 0 {
		log.Printf("[App] No remote plugins found to update")
		return fmt.Errorf("no external plugins installed")
	}

	log.Printf("[App] Found %d remote plugin(s) to force update", len(registries))

	var errors []string
	successCount := 0

	for _, registry := range registries {
		if registry.Repository == "" {
			log.Printf("[App] Skipping plugin %s: no repository URL", registry.PluginID)
			continue
		}

		log.Printf("[App] Force updating plugin %s from %s", registry.PluginID, registry.Repository)
		if err := a.pluginInstaller.UpdatePluginWithForce(registry.Repository, true); err != nil {
			errMsg := fmt.Sprintf("Failed to update %s: %v", registry.PluginID, err)
			log.Printf("[App] %s", errMsg)
			errors = append(errors, errMsg)
		} else {
			successCount++
			log.Printf("[App] Successfully force updated %s", registry.PluginID)
		}
	}

	log.Printf("[App] Force updated %d/%d remote plugins", successCount, len(registries))

	if len(errors) > 0 {
		return fmt.Errorf("some plugins failed to update: %s", strings.Join(errors, "; "))
	}

	return nil
}

func (a *App) GetRemotePlugins() ([]models.PluginRegistry, error) {
	log.Printf("[App] Getting all remote plugins")

	var registries []models.PluginRegistry
	if err := a.db.GetDB().Where("install_source = ?", "remote").Find(&registries).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch remote plugins: %w", err)
	}

	return registries, nil
}

func (a *App) ForceUpdateRemotePlugin(pluginID string) error {
	log.Printf("[App] Force updating remote plugin by ID: %s", pluginID)

	var registry models.PluginRegistry
	if err := a.db.GetDB().Where("plugin_id = ? AND install_source = ?", pluginID, "remote").First(&registry).Error; err != nil {
		return fmt.Errorf("remote plugin not found: %w", err)
	}

	if registry.Repository == "" {
		return fmt.Errorf("plugin %s: no repository URL", registry.PluginID)
	}

	log.Printf("[App] Force updating plugin %s from %s", registry.PluginID, registry.Repository)
	if err := a.pluginInstaller.UpdatePluginWithForce(registry.Repository, true); err != nil {
		log.Printf("[App] Failed to update %s: %v", registry.PluginID, err)
		return fmt.Errorf("failed to update %s: %w", registry.PluginID, err)
	}

	log.Printf("[App] Successfully force updated %s", registry.PluginID)
	return nil
}

func (a *App) UninstallPluginFromRepo(repoURL string, removeGitAuth bool, deleteJobHistory bool, deleteEnvironments bool) error {
	log.Printf("[App] Uninstalling plugin from repository: %s (removeGitAuth=%v, deleteJobHistory=%v, deleteEnvironments=%v)", repoURL, removeGitAuth, deleteJobHistory, deleteEnvironments)
	options := services.UninstallOptions{
		RemoveGitAuth:      removeGitAuth,
		DeleteJobHistory:   deleteJobHistory,
		DeleteEnvironments: deleteEnvironments,
	}
	err := a.pluginInstaller.UninstallPlugin(repoURL, options)
	if err != nil {
		return err
	}

	log.Printf("[App] Plugin uninstalled successfully from: %s", repoURL)
	a.emitEvent("plugin:uninstall:success", map[string]interface{}{
		"repo": repoURL,
	})

	return nil
}

func (a *App) GetPluginJobCount(pluginID string) (int64, error) {
	return a.pluginInstaller.GetPluginJobCount(pluginID)
}

func (a *App) GetPluginEnvironmentCount(pluginID string) (int64, error) {
	return a.pluginInstaller.GetPluginEnvironmentCount(pluginID)
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
	return a.ConfirmPluginInstallationWithRegistry(repoURL, commitHash, nil)
}

func (a *App) ConfirmPluginInstallationWithRegistry(repoURL string, commitHash string, registrySource *string) error {
	log.Printf("[App] User confirmed plugin installation from: %s (ref: %s)", repoURL, commitHash)
	if registrySource != nil {
		log.Printf("[App] Registry source: %s", *registrySource)
	}

	a.emitEvent("plugin:install:start", map[string]interface{}{
		"repo": repoURL,
		"ref":  commitHash,
	})

	go func() {
		_, err := a.pluginInstaller.InstallPlugin(repoURL, commitHash, registrySource, func(status string) {
			a.emitEvent("plugin:install:progress", map[string]string{
				"repo":   repoURL,
				"status": status,
			})
		})
		if err != nil {
			log.Printf("[App] Plugin installation failed: %v", err)
			a.emitEvent("plugin:install:error", map[string]interface{}{
				"repo":  repoURL,
				"error": err.Error(),
			})
			return
		}

		log.Printf("[App] Plugin installed successfully from: %s", repoURL)
		a.emitEvent("plugin:install:success", map[string]interface{}{
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
					a.emitEvent("protocol:error", map[string]string{
						"url":   url,
						"error": err.Error(),
					})
				} else {
					a.emitEvent("protocol:success", map[string]string{
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

	a.emitEvent("plugin:installed", nil)
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

	config := a.settings.GetConfig()
	var registrySource *string
	if config.PluginRegistryURL != "" {
		registrySource = &config.PluginRegistryURL
	}

	return a.ConfirmPluginInstallationWithRegistry(plugin.Repository, commitHash, registrySource)
}

func (a *App) CheckPluginUpdate(repoURL string, currentCommit string, registrySource *string) (interface{}, error) {
	log.Printf("[App] Checking update for plugin: %s (current: %s)", repoURL, currentCommit)

	var pluginRegistry models.PluginRegistry
	if err := a.db.GetDB().Where("repository = ?", repoURL).First(&pluginRegistry).Error; err != nil {
		return nil, fmt.Errorf("plugin not found in registry")
	}

	if pluginRegistry.UpdatePolicy == "manual" {
		log.Printf("[App] Plugin %s has manual update policy, skipping check", pluginRegistry.Name)
		return map[string]interface{}{
			"has_update":     false,
			"update_policy":  "manual",
			"current_commit": currentCommit,
		}, nil
	}

	if pluginRegistry.PinnedVersion != nil {
		log.Printf("[App] Plugin %s is pinned to version %s", pluginRegistry.Name, *pluginRegistry.PinnedVersion)
		return map[string]interface{}{
			"has_update":     false,
			"update_policy":  pluginRegistry.UpdatePolicy,
			"current_commit": currentCommit,
			"pinned_version": *pluginRegistry.PinnedVersion,
		}, nil
	}

	if pluginRegistry.RegistrySource != nil && *pluginRegistry.RegistrySource != "" {
		log.Printf("[App] Checking update from registry: %s", *pluginRegistry.RegistrySource)
		updateInfo, err := a.pluginRegistryService.CheckUpdate(pluginRegistry.PluginID)
		if err != nil {
			log.Printf("[App] Failed to check update from registry: %v", err)
			return nil, err
		}
		return updateInfo, nil
	}

	log.Printf("[App] Direct repository update check for: %s", repoURL)
	updateInfo, err := a.pluginInstaller.CheckRepositoryUpdate(repoURL, currentCommit)
	if err != nil {
		log.Printf("[App] Failed to check repository update: %v", err)
		return nil, err
	}

	return updateInfo, nil
}

func (a *App) SetPluginUpdatePolicy(repoURL string, policy string) error {
	log.Printf("[App] Setting update policy for %s to %s", repoURL, policy)

	var pluginRegistry models.PluginRegistry
	if err := a.db.GetDB().Where("repository = ?", repoURL).First(&pluginRegistry).Error; err != nil {
		return fmt.Errorf("plugin not found in registry")
	}

	pluginRegistry.UpdatePolicy = policy
	return a.db.GetDB().Save(&pluginRegistry).Error
}

func (a *App) PinPluginVersion(repoURL string, version string) error {
	log.Printf("[App] Pinning plugin %s to version %s", repoURL, version)

	var pluginRegistry models.PluginRegistry
	if err := a.db.GetDB().Where("repository = ?", repoURL).First(&pluginRegistry).Error; err != nil {
		return fmt.Errorf("plugin not found in registry")
	}

	pluginRegistry.PinnedVersion = &version
	return a.db.GetDB().Save(&pluginRegistry).Error
}

func (a *App) UnpinPluginVersion(repoURL string) error {
	log.Printf("[App] Unpinning plugin version for %s", repoURL)

	var pluginRegistry models.PluginRegistry
	if err := a.db.GetDB().Where("repository = ?", repoURL).First(&pluginRegistry).Error; err != nil {
		return fmt.Errorf("plugin not found in registry")
	}

	pluginRegistry.PinnedVersion = nil
	return a.db.GetDB().Save(&pluginRegistry).Error
}

type PluginRequirementsInfo struct {
	PluginID               string   `json:"pluginId"`
	PluginName             string   `json:"pluginName"`
	RuntimeEnvironments    []string `json:"runtimeEnvironments"`
	PythonRequirementsFile string   `json:"pythonRequirementsFile,omitempty"`
	RPackagesFile          string   `json:"rPackagesFile,omitempty"`
	PythonPackages         []string `json:"pythonPackages,omitempty"`
	RPackages              []string `json:"rPackages,omitempty"`
	RequirementsExist      bool     `json:"requirementsExist"`
}

func (a *App) GetPluginRequirements(pluginID string) (*PluginRequirementsInfo, error) {
	log.Printf("[App] Getting requirements for plugin: %s", pluginID)

	plugin, err := a.pluginLoaderV2.GetPluginByStringID(pluginID)
	if err != nil {
		return nil, fmt.Errorf("plugin not found: %w", err)
	}

	info := &PluginRequirementsInfo{
		PluginID:            plugin.Definition.Plugin.ID,
		PluginName:          plugin.Definition.Plugin.Name,
		RuntimeEnvironments: plugin.Definition.Runtime.GetEnvironments(),
	}

	if plugin.Definition.Execution.Requirements.PythonRequirementsFile != "" {
		info.PythonRequirementsFile = plugin.Definition.Execution.Requirements.PythonRequirementsFile
		reqPath := filepath.Join(plugin.FolderPath, info.PythonRequirementsFile)
		if content, err := os.ReadFile(reqPath); err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					info.PythonPackages = append(info.PythonPackages, line)
				}
			}
		}
	}

	if plugin.Definition.Execution.Requirements.RPackagesFile != "" {
		info.RPackagesFile = plugin.Definition.Execution.Requirements.RPackagesFile
		reqPath := filepath.Join(plugin.FolderPath, info.RPackagesFile)
		if content, err := os.ReadFile(reqPath); err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					info.RPackages = append(info.RPackages, line)
				}
			}
		}
	}

	info.RequirementsExist = len(info.PythonPackages) > 0 || len(info.RPackages) > 0

	return info, nil
}

func (a *App) FetchPluginDependencies(repoURL string) (map[string]interface{}, error) {
	log.Printf("[App] Fetching plugin dependencies from repo: %s", repoURL)

	pluginInfo, err := a.pluginInstaller.FetchPluginInfo(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch plugin info: %w", err)
	}

	hasPythonDeps := false
	hasRDeps := false

	hasPythonDeps = pluginInfo.Execution.Requirements.PythonRequirementsFile != "" ||
		(len(pluginInfo.Execution.Requirements.Packages) > 0 && pluginInfo.Runtime.HasEnvironment("python"))
	hasRDeps = pluginInfo.Execution.Requirements.RPackagesFile != "" ||
		(pluginInfo.Execution.Requirements.R != "" && pluginInfo.Runtime.HasEnvironment("r"))

	result := map[string]interface{}{
		"hasPythonDeps":       hasPythonDeps,
		"hasRDeps":            hasRDeps,
		"runtimeEnvironments": pluginInfo.Runtime.Environments,
		"name":                pluginInfo.Plugin.Name,
		"id":                  pluginInfo.Plugin.ID,
		"version":             pluginInfo.Plugin.Version,
		"author":              pluginInfo.Plugin.Author,
		"description":         pluginInfo.Plugin.Description,
		"category":            pluginInfo.Plugin.Category,
	}

	return result, nil
}

func (a *App) InstallPluginRequirements(pluginID string) error {
	log.Printf("[App] Installing requirements for plugin: %s", pluginID)

	plugin, err := a.pluginLoaderV2.GetPluginByStringID(pluginID)
	if err != nil {
		return fmt.Errorf("plugin not found: %w", err)
	}

	config := a.settings.GetConfig()

	if plugin.Definition.Execution.Requirements.PythonRequirementsFile != "" && plugin.Definition.Runtime.HasEnvironment("python") {
		reqPath := filepath.Join(plugin.FolderPath, plugin.Definition.Execution.Requirements.PythonRequirementsFile)
		if _, err := os.Stat(reqPath); err == nil {
			pythonPath := config.PythonPath

			pythonBinding, err := a.db.GetPluginEnvironmentBinding(pluginID, "python")
			if err == nil && pythonBinding != nil {
				pythonPath = pythonBinding.EnvironmentPath
				log.Printf("[App] Using bound Python environment: %s", pythonPath)
			} else {
				log.Printf("[App] No Python binding found, using global Python: %s", pythonPath)
			}

			if pythonPath == "" {
				return fmt.Errorf("Python path not configured")
			}

			log.Printf("[App] Installing Python requirements from: %s", reqPath)
			if err := a.envService.InstallPythonRequirements(pythonPath, reqPath); err != nil {
				return fmt.Errorf("failed to install Python requirements: %w", err)
			}
		}
	}

	if plugin.Definition.Execution.Requirements.RPackagesFile != "" && plugin.Definition.Runtime.HasEnvironment("r") {
		reqPath := filepath.Join(plugin.FolderPath, plugin.Definition.Execution.Requirements.RPackagesFile)
		if _, err := os.Stat(reqPath); err == nil {
			rPath := config.RPath

			rBinding, err := a.db.GetPluginEnvironmentBinding(pluginID, "r")
			if err == nil && rBinding != nil {
				rPath = rBinding.EnvironmentPath
				log.Printf("[App] Using bound R environment: %s", rPath)
			} else {
				log.Printf("[App] No R binding found, using global R: %s", rPath)
			}

			if rPath == "" {
				return fmt.Errorf("R path not configured")
			}

			content, err := os.ReadFile(reqPath)
			if err != nil {
				return fmt.Errorf("failed to read R packages file: %w", err)
			}

			var packages []string
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					packages = append(packages, line)
				}
			}

			if len(packages) > 0 {
				log.Printf("[App] Installing R packages: %v", packages)
				if err := a.envService.InstallRPackages(rPath, packages); err != nil {
					return fmt.Errorf("failed to install R packages: %w", err)
				}
			}
		}
	}

	return nil
}
