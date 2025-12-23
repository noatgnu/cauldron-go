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

func (s *ScriptExecutor) SetUpdateCallback(callback func(string, models.Job)) {
	s.updateCallback = callback
}

func (s *ScriptExecutor) SetOutputCallback(callback func(string, string)) {
	s.outputCallback = callback
}

type ScriptConfig struct {
	Type        string
	RuntimeType string
	ScriptName  string
	Args        []string
	OutputDir   string
}

func (s *ScriptExecutor) ExecutePythonScript(ctx context.Context, jobID string, config ScriptConfig) error {
	log.Printf("[ExecutePythonScript] Called for job %s with RuntimeType: '%s'", jobID, config.RuntimeType)
	cfg := s.settingsService.GetConfig()
	if cfg.PythonPath == "" {
		return fmt.Errorf("python path not configured")
	}

	pythonPath := cfg.PythonPath

	scriptPath := filepath.Join("plugins", config.Type, config.ScriptName)
	args := append([]string{scriptPath}, config.Args...)

	cmd := exec.CommandContext(ctx, pythonPath, args...)
	hideConsoleWindow(cmd)

	exePath, err := os.Executable()
	if err == nil {
		cmd.Dir = filepath.Dir(exePath)
	}

	if config.RuntimeType == "pythonWithR" {
		if cfg.RPath == "" {
			return fmt.Errorf("R path not configured for pythonWithR runtime")
		}

		rBinPath := filepath.Dir(cfg.RPath)

		if filepath.Base(rBinPath) == "x64" || filepath.Base(rBinPath) == "i386" {
			rBinPath = filepath.Dir(rBinPath)
		}

		rHomePath := filepath.Dir(rBinPath)

		env := os.Environ()
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
		cmd.Env = env
	}

	return s.executeCommand(ctx, jobID, cmd, config.OutputDir)
}

func (s *ScriptExecutor) ExecuteRScript(ctx context.Context, jobID string, config ScriptConfig) error {
	cfg := s.settingsService.GetConfig()
	if cfg.RPath == "" {
		return fmt.Errorf("R path not configured")
	}

	rPath := cfg.RPath
	var rLibPath string

	binding, err := s.db.GetPluginEnvironmentBinding(config.Type, "r")
	if err == nil && binding != nil {
		log.Printf("[ExecuteRScript] Found renv binding for plugin %s: %s", config.Type, binding.EnvironmentPath)

		renvEnv, err := s.db.GetRenvEnvironmentByID(binding.EnvironmentID)
		if err == nil {
			renvLibPath := filepath.Join(renvEnv.ProjectPath, "renv", "library")
			dirs, err := os.ReadDir(renvLibPath)
			if err == nil && len(dirs) > 0 {
				platformDir := filepath.Join(renvLibPath, dirs[0].Name())
				if _, err := os.Stat(platformDir); err == nil {
					rLibPath = platformDir
					log.Printf("[ExecuteRScript] Using renv library path: %s", rLibPath)
				}
			}
		}
	}

	scriptPath := filepath.Join("plugins", config.Type, config.ScriptName)
	scriptPath = strings.ReplaceAll(scriptPath, "\\", "/")
	log.Printf("[ExecuteRScript] Script path: %s", scriptPath)
	log.Printf("[ExecuteRScript] R path: %s", rPath)

	args := []string{"--vanilla", "--slave", scriptPath, "--args"}
	args = append(args, config.Args...)
	log.Printf("[ExecuteRScript] Full command: %s %v", rPath, args)

	cmd := exec.CommandContext(ctx, rPath, args...)
	hideConsoleWindow(cmd)

	env := os.Environ()
	env = append(env, "R_DEFAULT_DEVICE=null")

	if rLibPath != "" {
		rLibPath = strings.ReplaceAll(rLibPath, "\\", "/")
		env = append(env, fmt.Sprintf("R_LIBS_USER=%s", rLibPath))
		env = append(env, fmt.Sprintf("R_LIBS=%s", rLibPath))
		log.Printf("[ExecuteRScript] Set R_LIBS to: %s", rLibPath)
	}

	cmd.Env = env

	exePath, err := os.Executable()
	if err == nil {
		cmd.Dir = filepath.Dir(exePath)
		log.Printf("[ExecuteRScript] Working directory: %s", cmd.Dir)
	}

	return s.executeCommand(ctx, jobID, cmd, config.OutputDir)
}

func (s *ScriptExecutor) executeCommand(ctx context.Context, jobID string, cmd *exec.Cmd, outputDir string) error {
	s.mu.Lock()
	s.runningJobs[jobID] = cmd
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.runningJobs, jobID)
		s.mu.Unlock()
	}()

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

	go s.streamOutput(jobID, stdout, &wg, false)
	go s.streamOutput(jobID, stderr, &wg, true)

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

func (s *ScriptExecutor) streamOutput(jobID string, reader io.Reader, wg *sync.WaitGroup, isError bool) {
	defer wg.Done()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if isError {
			log.Printf("[ScriptExecutor][STDERR][%s] %s", jobID, line)
		} else {
			log.Printf("[ScriptExecutor][STDOUT][%s] %s", jobID, line)
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
