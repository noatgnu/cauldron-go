package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/noatgnu/cauldron-go/backend/models"
)

type ScriptExecutor struct {
	settingsService *SettingsService
	db              *DatabaseService
	pluginLoader    *PluginLoaderV2
	runningJobs     map[string]*exec.Cmd
	mu              sync.RWMutex
	updateCallback  func(string, models.Job)
	outputCallback  func(string, string)
}

func NewScriptExecutor(settingsService *SettingsService, db *DatabaseService) *ScriptExecutor {
	return &ScriptExecutor{
		settingsService: settingsService,
		db:              db,
		runningJobs:     make(map[string]*exec.Cmd),
	}
}

func (s *ScriptExecutor) SetPluginLoader(pluginLoader *PluginLoaderV2) {
	s.pluginLoader = pluginLoader
}

func (s *ScriptExecutor) SetUpdateCallback(callback func(string, models.Job)) {
	s.updateCallback = callback
}

func (s *ScriptExecutor) SetOutputCallback(callback func(string, string)) {
	s.outputCallback = callback
}

func (s *ScriptExecutor) prepareEnv(pluginID uint) []string {
	env := os.Environ()

	// 1. Load global custom env vars (pluginID 0)
	globals, err := s.db.GetGlobalCustomEnvVars()
	if err == nil {
		for _, v := range globals {
			env = append(env, fmt.Sprintf("%s=%s", v.Key, v.Value))
		}
	}

	// 2. Load plugin-specific custom env vars (overrides global)
	if pluginID != 0 {
		locals, err := s.db.GetCustomEnvVars(pluginID)
		if err == nil {
			for _, v := range locals {
				env = append(env, fmt.Sprintf("%s=%s", v.Key, v.Value))
			}
		}
	}

	return env
}

type ScriptConfig struct {
	PluginID     uint
	Type         string
	Environments []string
	ScriptName   string
	Args         []string
	OutputDir    string
	FolderPath   string
}

func (s *ScriptExecutor) ExecutePythonScript(ctx context.Context, jobID string, config ScriptConfig) error {
	log.Printf("[ExecutePythonScript] Called for job %s with environments: %v", jobID, config.Environments)
	cfg := s.settingsService.GetConfig()

	var pythonPath string
	var envInfo string

	// Check for plugin-specific venv binding
	binding, err := s.db.GetPluginEnvironmentBinding(config.Type, "python")
	if err != nil {
		log.Printf("[ExecutePythonScript] No binding found for plugin %s (type: %s): %v", jobID, config.Type, err)
	} else {
		log.Printf("[ExecutePythonScript] Found binding for plugin %s: %s", jobID, binding.EnvironmentPath)
	}

	if binding != nil && binding.EnvironmentPath != "" {
		// Check if the bound path is already a file (the executable)
		fileInfo, err := os.Stat(binding.EnvironmentPath)
		if err == nil && !fileInfo.IsDir() {
			pythonPath = binding.EnvironmentPath
			envInfo = fmt.Sprintf("Python (Bound): %s", pythonPath)
		} else {
			// Assume it's a venv root directory
			venvPath := binding.EnvironmentPath

			// Construct Python executable path from venv
			if runtime.GOOS == "windows" {
				pythonPath = filepath.Join(venvPath, "Scripts", "python.exe")
			} else {
				pythonPath = filepath.Join(venvPath, "bin", "python")
			}
			envInfo = fmt.Sprintf("Python (Bound venv): %s", venvPath)
		}

		// Verify the Python executable exists
		if _, err := os.Stat(pythonPath); err != nil {
			log.Printf("[ExecutePythonScript] Warning: Bound Python not found at %s, falling back to global Python", pythonPath)
			pythonPath = ""
		}
	}

	// Fall back to global Python if no binding or binding failed
	if pythonPath == "" {
		if cfg.PythonPath == "" {
			return fmt.Errorf("python path not configured")
		}
		pythonPath = cfg.PythonPath
		envInfo = fmt.Sprintf("Python (Global): %s", pythonPath)
		log.Printf("[ExecutePythonScript] Using global Python: %s", pythonPath)
	}

	var pluginDir string
	if config.FolderPath != "" {
		pluginDir = config.FolderPath
	} else {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		exeDir := filepath.Dir(exePath)
		pluginDir = filepath.Join(exeDir, "plugins", config.Type)
	}

	args := append([]string{config.ScriptName}, config.Args...)

	cmd := exec.CommandContext(ctx, pythonPath, args...)
	hideConsoleWindow(cmd)

	cmd.Dir = pluginDir
	log.Printf("[ExecutePythonScript] Working directory: %s", cmd.Dir)
	log.Printf("[ExecutePythonScript] Script name: %s", config.ScriptName)

	env := s.prepareEnv(config.PluginID)
	env = append(env, "PYTHONUNBUFFERED=1")

	hasR := false
	for _, e := range config.Environments {
		if e == "r" {
			hasR = true
			break
		}
	}

	if hasR {
		if cfg.RPath == "" {
			return fmt.Errorf("R path not configured for python+R runtime")
		}

		rBinPath := filepath.Dir(cfg.RPath)

		if filepath.Base(rBinPath) == "x64" || filepath.Base(rBinPath) == "i386" {
			rBinPath = filepath.Dir(rBinPath)
		}

		rHomePath := filepath.Dir(rBinPath)

		rhomeSet := false
		for i, e := range env {
			if len(e) > 7 && e[:7] == "R_HOME=" {
				env[i] = fmt.Sprintf("R_HOME=%s", rHomePath)
				rhomeSet = true
				break
			}
		}
		if !rhomeSet {
			env = append(env, fmt.Sprintf("R_HOME=%s", rHomePath))
		}
	}
	cmd.Env = env

	return s.executeCommand(ctx, jobID, cmd, config.OutputDir, envInfo)
}

func (s *ScriptExecutor) ExecuteRScript(ctx context.Context, jobID string, config ScriptConfig) error {
	log.Printf("[ExecuteRScript] Starting execution for job %s with config.Type=%s", jobID, config.Type)

	cfg := s.settingsService.GetConfig()
	if cfg.RPath == "" {
		return fmt.Errorf("R path not configured")
	}

	rPath := cfg.RPath
	var envInfo string

	// Check for plugin-specific renv binding
	binding, err := s.db.GetPluginEnvironmentBinding(config.Type, "r")
	if err != nil {
		log.Printf("[ExecuteRScript] No binding found for plugin %s (type: %s): %v", jobID, config.Type, err)
	} else {
		log.Printf("[ExecuteRScript] Found binding for plugin %s: ID=%d", jobID, binding.EnvironmentID)
	}

	var renvProjectPath string
	if binding != nil && binding.EnvironmentPath != "" {
		renvEnv, err := s.db.GetRenvEnvironmentByID(binding.EnvironmentID)
		if err == nil {
			renvProjectPath = renvEnv.ProjectPath
			envInfo = fmt.Sprintf("R (Bound renv): %s", renvProjectPath)
		}
	}

	// Fall back to global R if no binding
	if envInfo == "" {
		envInfo = fmt.Sprintf("R (Global): %s", rPath)
		log.Printf("[ExecuteRScript] Using global R: %s", rPath)
	}

	var pluginDir string
	if config.FolderPath != "" {
		pluginDir = config.FolderPath
	} else {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		exeDir := filepath.Dir(exePath)
		pluginDir = filepath.Join(exeDir, "plugins", config.Type)
	}

	scriptName := strings.ReplaceAll(config.ScriptName, "\\", "/")
	log.Printf("[ExecuteRScript] Script name: %s", scriptName)
	log.Printf("[ExecuteRScript] R path: %s", rPath)
	log.Printf("[ExecuteRScript] Working directory: %s", pluginDir)

	var args []string
	var cmd *exec.Cmd

	// If using renv, create wrapper script to activate renv before running plugin script
	if renvProjectPath != "" {
		// Create a temporary wrapper script that activates renv and sources the plugin script
		renvProjectPathR := strings.ReplaceAll(renvProjectPath, "\\", "/")
		scriptNameR := strings.ReplaceAll(scriptName, "\\", "/")

		wrapperScript := fmt.Sprintf(`
Sys.setenv(RENV_PROJECT='%s')
source('%s/renv/activate.R')
cat('Library paths after activation:\n')
cat(.libPaths(), sep='\n')
source('%s')
`, renvProjectPathR, renvProjectPathR, scriptNameR)

		// Write wrapper to temp file
		tmpFile, err := os.CreateTemp("", "renv-wrapper-*.R")
		if err != nil {
			return fmt.Errorf("failed to create temp wrapper script: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(wrapperScript); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to write wrapper script: %w", err)
		}
		tmpFile.Close()

		args = []string{"--vanilla", "--slave", tmpFile.Name(), "--args"}
		args = append(args, config.Args...)
		log.Printf("[ExecuteRScript] Using renv wrapper script: %s", tmpFile.Name())
		log.Printf("[ExecuteRScript] Full command: %s %v", rPath, args)

		cmd = exec.CommandContext(ctx, rPath, args...)
		hideConsoleWindow(cmd)

		env := s.prepareEnv(config.PluginID)
		env = append(env, "R_DEFAULT_DEVICE=null")
		env = append(env, fmt.Sprintf("RENV_PROJECT=%s", renvProjectPath))
		cmd.Env = env
	} else {
		// No renv binding, run script directly
		args = []string{"--vanilla", "--slave", scriptName, "--args"}
		args = append(args, config.Args...)
		log.Printf("[ExecuteRScript] Full command: %s %v", rPath, args)

		cmd = exec.CommandContext(ctx, rPath, args...)
		hideConsoleWindow(cmd)

		env := s.prepareEnv(config.PluginID)
		env = append(env, "R_DEFAULT_DEVICE=null")
		cmd.Env = env
	}

	cmd.Dir = pluginDir
	log.Printf("[ExecuteRScript] Working directory: %s", cmd.Dir)

	return s.executeCommand(ctx, jobID, cmd, config.OutputDir, envInfo)
}

func (s *ScriptExecutor) ExecuteDirectScript(ctx context.Context, jobID string, config ScriptConfig) error {
	log.Printf("[ExecuteDirectScript] Called for job %s", jobID)

	var pluginDir string
	if config.FolderPath != "" {
		pluginDir = config.FolderPath
	} else {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		exeDir := filepath.Dir(exePath)
		pluginDir = filepath.Join(exeDir, "plugins", config.Type)
	}

	executablePath := filepath.Join(pluginDir, config.ScriptName)

	if !filepath.IsAbs(executablePath) {
		absPath, err := filepath.Abs(executablePath)
		if err == nil {
			executablePath = absPath
		}
	}

	log.Printf("[ExecuteDirectScript] Executable path: %s", executablePath)
	log.Printf("[ExecuteDirectScript] Working directory: %s", pluginDir)
	log.Printf("[ExecuteDirectScript] Args: %v", config.Args)

	cmd := exec.CommandContext(ctx, executablePath, config.Args...)
	hideConsoleWindow(cmd)

	cmd.Dir = pluginDir

	env := s.prepareEnv(config.PluginID)
	cmd.Env = env

	envInfo := fmt.Sprintf("Direct execution: %s", executablePath)

	return s.executeCommand(ctx, jobID, cmd, config.OutputDir, envInfo)
}

func (s *ScriptExecutor) ExecuteDockerScript(ctx context.Context, jobID string, config ScriptConfig) error {
	log.Printf("[ExecuteDockerScript] Starting execution for job %s", jobID)

	plugin, err := s.pluginLoader.GetPlugin(config.PluginID)
	if err != nil {
		return fmt.Errorf("failed to get plugin: %w", err)
	}

	if plugin.Definition.Runtime.Docker == nil {
		return fmt.Errorf("docker configuration not found for plugin")
	}

	imageName := plugin.Definition.Runtime.GetDockerImageName(plugin.Definition.Plugin.ID)
	if plugin.Definition.Runtime.Docker.Image != "" {
		imageName = plugin.Definition.Runtime.Docker.Image
	}

	checkCmd := exec.Command("docker", "image", "inspect", imageName)
	hideConsoleWindow(checkCmd)
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("docker image not found: %s (hint: plugin may need reinstallation)", imageName)
	}

	pluginDir := config.FolderPath
	if pluginDir == "" {
		return fmt.Errorf("plugin folder path is empty")
	}

	if config.OutputDir == "" {
		return fmt.Errorf("output directory is empty")
	}

	pluginDir = filepath.Clean(pluginDir)
	outputDir := filepath.Clean(config.OutputDir)

	pluginDir = filepath.ToSlash(pluginDir)
	outputDir = filepath.ToSlash(outputDir)

	args := []string{"run"}
	args = append(args, "--rm")

	args = append(args, "-v", fmt.Sprintf("%s:/output", outputDir))

	log.Printf("[ExecuteDockerScript] Volume mounts: /output <- %s", outputDir)

	env := s.prepareEnv(config.PluginID)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 && parts[0] != "" && !strings.HasPrefix(parts[0], "=") {
			args = append(args, "-e", fmt.Sprintf("%s=%s", parts[0], parts[1]))
		}
	}

	if plugin.Definition.Runtime.Docker.Platform != "" {
		args = append(args, "--platform", plugin.Definition.Runtime.Docker.Platform)
	}

	primaryEnv := config.Environments[0]
	if primaryEnv != "docker" {
		return fmt.Errorf("docker must be primary environment when using docker runtime")
	}

	remappedArgs, volumeMounts, err := s.prepareDockerInputFiles(config.Args, outputDir)
	if err != nil {
		return fmt.Errorf("failed to prepare input files: %w", err)
	}

	for _, mount := range volumeMounts {
		args = append(args, "-v", mount)
	}

	args = append(args, imageName)

	entrypointIsCommand := strings.Contains(config.ScriptName, " ")

	if entrypointIsCommand {
		cmdStr := config.ScriptName
		for _, arg := range remappedArgs {
			escapedArg := strings.ReplaceAll(arg, "'", "'\\''")
			cmdStr += " '" + escapedArg + "'"
		}
		args = append(args, "sh", "-c", cmdStr)
		log.Printf("[ExecuteDockerScript] Using sh -c wrapper for complex entrypoint")
	} else {
		args = append(args, config.ScriptName)
		args = append(args, remappedArgs...)
		log.Printf("[ExecuteDockerScript] Using direct entrypoint: %s", config.ScriptName)
	}

	log.Printf("[ExecuteDockerScript] Full docker command: docker %v", args)

	cmd := exec.CommandContext(ctx, "docker", args...)
	hideConsoleWindow(cmd)

	cmd.Dir = pluginDir
	cmd.Env = os.Environ()

	envInfo := fmt.Sprintf("Docker image: %s", imageName)

	return s.executeCommand(ctx, jobID, cmd, outputDir, envInfo)
}

func (s *ScriptExecutor) prepareDockerInputFiles(args []string, outputDir string) ([]string, []string, error) {
	remappedArgs := make([]string, len(args))
	copy(remappedArgs, args)

	var volumeMounts []string
	fileIndex := 0

	cleanOutputDir := filepath.Clean(outputDir)
	cleanOutputDir = filepath.ToSlash(cleanOutputDir)

	for i, arg := range remappedArgs {
		if !filepath.IsAbs(arg) {
			continue
		}

		cleanArg := filepath.Clean(arg)
		cleanArg = filepath.ToSlash(cleanArg)

		if cleanArg == cleanOutputDir {
			remappedArgs[i] = "/output"
			log.Printf("[prepareDockerInputFiles] Remapping output dir %s -> /output", arg)
			continue
		}

		fileInfo, err := os.Stat(arg)
		if err != nil {
			continue
		}

		if fileInfo.IsDir() {
			continue
		}

		ext := filepath.Ext(arg)
		containerPath := fmt.Sprintf("/input_%d%s", fileIndex, ext)

		volumeMount := fmt.Sprintf("%s:%s:ro", cleanArg, containerPath)
		volumeMounts = append(volumeMounts, volumeMount)
		log.Printf("[prepareDockerInputFiles] Mounting %s -> %s:ro", cleanArg, containerPath)

		remappedArgs[i] = containerPath
		fileIndex++
	}

	return remappedArgs, volumeMounts, nil
}

func (s *ScriptExecutor) executeCommand(ctx context.Context, jobID string, cmd *exec.Cmd, outputDir string, runtimeInfo string) error {
	s.mu.Lock()
	s.runningJobs[jobID] = cmd
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.runningJobs, jobID)
		s.mu.Unlock()
	}()

	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Printf("[ScriptExecutor] Warning: Failed to create output directory: %v", err)
		}
	}

	logFilePath := filepath.Join(outputDir, "execution.log")
	logFile, err := os.Create(logFilePath)
	if err != nil {
		log.Printf("[ScriptExecutor] Warning: Failed to create log file at %s: %v", logFilePath, err)
		logFile = nil
	}
	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()

	// Log the exact command and environment variables for reproducibility
	cmdInfo := fmt.Sprintf("=== COMMAND EXECUTION INFO ===\nRuntime Environment: %s\nWorking Directory: %s\nExecutable: %s\nArguments: %v\n",
		runtimeInfo, cmd.Dir, cmd.Path, cmd.Args[1:])
	log.Printf("[ScriptExecutor][%s] %s", jobID, cmdInfo)
	if logFile != nil {
		logFile.WriteString(cmdInfo + "\n")
	}

	// Log environment variables
	envInfo := "Environment Variables:\n"
	baseEnv := os.Environ()
	baseEnvMap := make(map[string]string)
	for _, e := range baseEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			baseEnvMap[parts[0]] = parts[1]
		}
	}

	// Log custom and modified environment variables
	for _, e := range cmd.Env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			key, value := parts[0], parts[1]
			// Only log if it's different from base environment or is a known custom variable
			if baseVal, exists := baseEnvMap[key]; !exists || baseVal != value {
				envInfo += fmt.Sprintf("  %s=%s\n", key, value)
			}
		}
	}
	log.Printf("[ScriptExecutor][%s] %s", jobID, envInfo)
	if logFile != nil {
		logFile.WriteString(envInfo + "\n")
	}

	// Create a reproducible command string
	reproducibleCmd := fmt.Sprintf("# Runtime: %s\ncd \"%s\" && ", runtimeInfo, cmd.Dir)
	// Add environment variables
	for _, e := range cmd.Env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			key, value := parts[0], parts[1]
			if baseVal, exists := baseEnvMap[key]; !exists || baseVal != value {
				reproducibleCmd += fmt.Sprintf("%s=\"%s\" ", key, value)
			}
		}
	}
	// Add the command
	reproducibleCmd += fmt.Sprintf("\"%s\"", cmd.Path)
	for _, arg := range cmd.Args[1:] {
		reproducibleCmd += fmt.Sprintf(" \"%s\"", arg)
	}
	reproducibleCmdInfo := fmt.Sprintf("Reproducible Command:\n%s\n=== END COMMAND INFO ===\n", reproducibleCmd)
	log.Printf("[ScriptExecutor][%s] %s", jobID, reproducibleCmdInfo)
	if logFile != nil {
		logFile.WriteString(reproducibleCmdInfo + "\n")
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go s.streamOutput(jobID, stdout, &wg, false, logFile)
	go s.streamOutput(jobID, stderr, &wg, true, logFile)

	select {
	case <-ctx.Done():
		log.Printf("[ScriptExecutor] Context cancelled for job %s, killing process tree", jobID)
		s.killProcessTree(cmd)
		wg.Wait()
		if s.updateCallback != nil {
			s.updateCallback(jobID, models.Job{
				Status:     "failed",
				Error:      "Job cancelled by user",
				OutputPath: outputDir,
			})
		}
		return fmt.Errorf("job cancelled")

	case err := <-done:
		wg.Wait()
		if err != nil {
			log.Printf("[ScriptExecutor] Command failed for job %s: %v", jobID, err)
			if s.updateCallback != nil {
				s.updateCallback(jobID, models.Job{
					Status:     "failed",
					Error:      err.Error(),
					OutputPath: outputDir,
				})
			}
			return err
		}

		log.Printf("[ScriptExecutor] Command completed successfully for job %s", jobID)
		if s.updateCallback != nil {
			s.updateCallback(jobID, models.Job{
				Status:     "completed",
				Progress:   100,
				OutputPath: outputDir,
			})
		}

		return nil
	}
}

func (s *ScriptExecutor) streamOutput(jobID string, reader io.Reader, wg *sync.WaitGroup, isError bool, logFile *os.File) {
	defer wg.Done()

	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		if isError {
			log.Printf("[ScriptExecutor][STDERR][%s] %s", jobID, line)
		} else {
			log.Printf("[ScriptExecutor][STDOUT][%s] %s", jobID, line)
		}

		if logFile != nil {
			logFile.WriteString(line + "\n")
		}

		if s.outputCallback != nil {
			s.outputCallback(jobID, line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[ScriptExecutor] Error reading output for job %s: %v", jobID, err)
	}
}

func (s *ScriptExecutor) killProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pid := cmd.Process.Pid
	log.Printf("[ScriptExecutor] Killing process tree for PID %d", pid)

	if runtime.GOOS == "windows" {
		killCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
		if err := killCmd.Run(); err != nil {
			log.Printf("[ScriptExecutor] Failed to kill process tree with taskkill: %v", err)
			return cmd.Process.Kill()
		}
		log.Printf("[ScriptExecutor] Successfully killed process tree for PID %d", pid)
		return nil
	} else {
		return cmd.Process.Kill()
	}
}

func (s *ScriptExecutor) CancelJob(jobID string) error {
	s.mu.RLock()
	cmd, exists := s.runningJobs[jobID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job %s is not running", jobID)
	}

	if err := s.killProcessTree(cmd); err != nil {
		return fmt.Errorf("failed to kill process tree: %w", err)
	}

	if s.updateCallback != nil {
		s.updateCallback(jobID, models.Job{
			Status: "failed",
			Error:  "Cancelled by user",
		})
	}

	return nil
}

func (s *ScriptExecutor) KillAllJobs() {
	s.mu.RLock()
	jobIDs := make([]string, 0, len(s.runningJobs))
	for jobID := range s.runningJobs {
		jobIDs = append(jobIDs, jobID)
	}
	s.mu.RUnlock()

	log.Printf("[ScriptExecutor] Killing %d running jobs", len(jobIDs))
	for _, jobID := range jobIDs {
		s.mu.RLock()
		cmd, exists := s.runningJobs[jobID]
		s.mu.RUnlock()

		if exists {
			log.Printf("[ScriptExecutor] Force killing job %s", jobID)
			if err := s.killProcessTree(cmd); err != nil {
				log.Printf("[ScriptExecutor] Error killing job %s: %v", jobID, err)
			}
		}
	}
	log.Println("[ScriptExecutor] Finished killing all jobs")
}
