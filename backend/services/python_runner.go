package services

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type PythonRunner struct {
	pythonPath string
	scriptDir  string
}

func NewPythonRunner(settings *SettingsService) *PythonRunner {
	execPath, _ := os.Executable()
	scriptDir := filepath.Join(filepath.Dir(execPath), "scripts", "python")

	if _, err := os.Stat(scriptDir); os.IsNotExist(err) {
		scriptDir = "scripts/python"
	}

	return &PythonRunner{
		pythonPath: settings.config.PythonPath,
		scriptDir:  scriptDir,
	}
}

func (p *PythonRunner) ExecuteScript(scriptName string, args []string, outputCallback func(line string)) error {
	var scriptPath string
	if filepath.IsAbs(scriptName) {
		scriptPath = scriptName
	} else {
		scriptPath = filepath.Join(p.scriptDir, scriptName)
	}

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script not found: %s", scriptPath)
	}

	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.Command(p.pythonPath, cmdArgs...)
	hideConsoleWindow(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if outputCallback != nil {
				outputCallback(scanner.Text())
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			if outputCallback != nil {
				outputCallback("[ERROR] " + scanner.Text())
			}
		}
	}()

	return cmd.Wait()
}

func (p *PythonRunner) ValidatePythonInstallation() error {
	cmd := exec.Command(p.pythonPath, "--version")
	hideConsoleWindow(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("python installation not valid: %v", err)
	}
	return nil
}

func (p *PythonRunner) InstallRequirements() error {
	requirementsPath := filepath.Join(filepath.Dir(p.scriptDir), "requirements.txt")

	if _, err := os.Stat(requirementsPath); os.IsNotExist(err) {
		return fmt.Errorf("requirements.txt not found: %s", requirementsPath)
	}

	cmd := exec.Command(p.pythonPath, "-m", "pip", "install", "-r", requirementsPath)
	hideConsoleWindow(cmd)
	return cmd.Run()
}

func (p *PythonRunner) GetPythonVersion() (string, error) {
	if p.pythonPath == "" {
		return "", fmt.Errorf("python path not configured")
	}
	cmd := exec.Command(p.pythonPath, "--version")
	hideConsoleWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
