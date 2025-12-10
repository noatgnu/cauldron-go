package services

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type DirectRunner struct {
	binDir string
}

func NewDirectRunner() *DirectRunner {
	execPath, _ := os.Executable()
	binDir := filepath.Join(filepath.Dir(execPath), "bin", "external")

	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		binDir = "bin/external"
	}

	return &DirectRunner{
		binDir: binDir,
	}
}

func (d *DirectRunner) ExecuteProgram(programPath string, args []string, workingDir string, outputCallback func(line string)) error {
	var executablePath string

	if filepath.IsAbs(programPath) {
		executablePath = programPath
	} else if filepath.Ext(programPath) != "" {
		executablePath = filepath.Join(d.binDir, programPath)
	} else {
		var err error
		executablePath, err = exec.LookPath(programPath)
		if err != nil {
			executablePath = filepath.Join(d.binDir, programPath)
		}
	}

	if !filepath.IsAbs(executablePath) {
		absPath, err := filepath.Abs(executablePath)
		if err == nil {
			executablePath = absPath
		}
	}

	cmd := exec.Command(executablePath, args...)
	hideConsoleWindow(cmd)

	if workingDir != "" {
		cmd.Dir = workingDir
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
		return fmt.Errorf("failed to start program %s: %v", programPath, err)
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

func (d *DirectRunner) ValidateProgram(programName string) error {
	programPath, err := exec.LookPath(programName)
	if err != nil {
		testPath := filepath.Join(d.binDir, programName)
		if _, err := os.Stat(testPath); os.IsNotExist(err) {
			return fmt.Errorf("program not found in PATH or bin directory: %s", programName)
		}
		programPath = testPath
	}

	cmd := exec.Command(programPath, "--version")
	hideConsoleWindow(cmd)
	if err := cmd.Run(); err != nil {
		cmd = exec.Command(programPath, "-version")
		hideConsoleWindow(cmd)
		if err := cmd.Run(); err != nil {
			cmd = exec.Command(programPath, "-h")
			hideConsoleWindow(cmd)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("program validation failed: %v", err)
			}
		}
	}
	return nil
}

func (d *DirectRunner) GetProgramVersion(programName string) (string, error) {
	programPath, err := exec.LookPath(programName)
	if err != nil {
		testPath := filepath.Join(d.binDir, programName)
		if _, err := os.Stat(testPath); os.IsNotExist(err) {
			return "", fmt.Errorf("program not found: %s", programName)
		}
		programPath = testPath
	}

	cmd := exec.Command(programPath, "--version")
	hideConsoleWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		cmd = exec.Command(programPath, "-version")
		hideConsoleWindow(cmd)
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("could not get version: %v", err)
		}
	}
	return string(output), nil
}
