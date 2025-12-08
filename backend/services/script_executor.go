package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/noatgnu/cauldron-go/backend/models"
)

type ScriptExecutor struct {
	settingsService *SettingsService
	runningJobs     map[string]*exec.Cmd
	mu              sync.RWMutex
	updateCallback  func(string, models.Job)
}

func NewScriptExecutor(settingsService *SettingsService) *ScriptExecutor {
	return &ScriptExecutor{
		settingsService: settingsService,
		runningJobs:     make(map[string]*exec.Cmd),
	}
}

func (s *ScriptExecutor) SetUpdateCallback(callback func(string, models.Job)) {
	s.updateCallback = callback
}

type ScriptConfig struct {
	Type       string
	ScriptName string
	Args       []string
	OutputDir  string
}

func (s *ScriptExecutor) ExecutePythonScript(ctx context.Context, jobID string, config ScriptConfig) error {
	cfg := s.settingsService.GetConfig()
	if cfg.PythonPath == "" {
		return fmt.Errorf("python path not configured")
	}

	pythonPath := cfg.PythonPath

	scriptPath := filepath.Join("scripts", "python", config.ScriptName)
	args := append([]string{scriptPath}, config.Args...)

	cmd := exec.CommandContext(ctx, pythonPath, args...)

	return s.executeCommand(ctx, jobID, cmd, config.OutputDir)
}

func (s *ScriptExecutor) ExecuteRScript(ctx context.Context, jobID string, config ScriptConfig) error {
	cfg := s.settingsService.GetConfig()
	if cfg.RPath == "" {
		return fmt.Errorf("R path not configured")
	}

	rPath := cfg.RPath

	scriptPath := filepath.Join("scripts", "r", config.ScriptName)

	args := []string{"--vanilla", "-f", scriptPath, "--args"}
	args = append(args, config.Args...)

	cmd := exec.CommandContext(ctx, rPath, args...)

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

	var wg sync.WaitGroup
	wg.Add(2)

	go s.streamOutput(jobID, stdout, &wg, false)
	go s.streamOutput(jobID, stderr, &wg, true)

	wg.Wait()

	if err := cmd.Wait(); err != nil {
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
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[ScriptExecutor] Error reading output for job %s: %v", jobID, err)
	}
}

func (s *ScriptExecutor) CancelJob(jobID string) error {
	s.mu.RLock()
	cmd, exists := s.runningJobs[jobID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job %s is not running", jobID)
	}

	if cmd.Process != nil {
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	if s.updateCallback != nil {
		s.updateCallback(jobID, models.Job{
			Status: "failed",
			Error:  "Cancelled by user",
		})
	}

	return nil
}
