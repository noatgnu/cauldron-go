package services

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type RRunner struct {
	rscriptPath string
	rLibPath    string
	scriptDir   string
}

func NewRRunner(settings *SettingsService) *RRunner {
	execPath, _ := os.Executable()
	scriptDir := filepath.Join(filepath.Dir(execPath), "scripts", "r")

	if _, err := os.Stat(scriptDir); os.IsNotExist(err) {
		scriptDir = "scripts/r"
	}

	return &RRunner{
		rscriptPath: settings.config.RPath,
		rLibPath:    settings.config.RLibPath,
		scriptDir:   scriptDir,
	}
}

func (r *RRunner) ExecuteScript(scriptName string, args []string, outputCallback func(line string)) error {
	var scriptPath string
	if filepath.IsAbs(scriptName) {
		scriptPath = scriptName
	} else {
		scriptPath = filepath.Join(r.scriptDir, scriptName)
	}

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script not found: %s", scriptPath)
	}

	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.Command(r.rscriptPath, cmdArgs...)
	hideConsoleWindow(cmd)

	if r.rLibPath != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("R_LIBS=%s", r.rLibPath))
	}

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

func (r *RRunner) ValidateRInstallation() error {
	cmd := exec.Command(r.rscriptPath, "--version")
	hideConsoleWindow(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("R installation not valid: %v", err)
	}
	return nil
}

func (r *RRunner) InstallPackages() error {
	installScript := filepath.Join(r.scriptDir, "install_packages.R")

	if _, err := os.Stat(installScript); os.IsNotExist(err) {
		return fmt.Errorf("install_packages.R not found: %s", installScript)
	}

	cmd := exec.Command(r.rscriptPath, installScript)
	hideConsoleWindow(cmd)
	if r.rLibPath != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("R_LIBS=%s", r.rLibPath))
	}

	return cmd.Run()
}

func (r *RRunner) GetRVersion() (string, error) {
	if r.rscriptPath == "" {
		return "", fmt.Errorf("R path not configured")
	}
	cmd := exec.Command(r.rscriptPath, "--version")
	hideConsoleWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
