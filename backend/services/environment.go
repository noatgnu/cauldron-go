package services

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type PythonEnvironment struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Version     string `json:"version"`
	IsVirtual   bool   `json:"isVirtual"`
	HasPackages bool   `json:"hasPackages"`
}

type REnvironment struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Version     string `json:"version"`
	LibPath     string `json:"libPath"`
	HasPackages bool   `json:"hasPackages"`
	IsDefault   bool   `json:"isDefault"`
}

type EnvironmentService struct {
	ctx              context.Context
	db               *DatabaseService
	progressNotifier *ProgressNotifier
}

func NewEnvironmentService(ctx context.Context, db *DatabaseService, progressNotifier *ProgressNotifier) *EnvironmentService {
	return &EnvironmentService{
		ctx:              ctx,
		db:               db,
		progressNotifier: progressNotifier,
	}
}

func (e *EnvironmentService) DetectPythonEnvironments() ([]PythonEnvironment, error) {
	log.Println("[DetectPythonEnvironments] Starting...")

	portablePython, err := e.detectPortablePython()
	if err == nil {
		log.Printf("[DetectPythonEnvironments] Found portable Python: %s\n", portablePython.Path)
		if err := e.db.SavePythonEnvironment(portablePython); err != nil {
			log.Printf("[DetectPythonEnvironments] Failed to save portable Python: %v\n", err)
		}
	} else {
		log.Printf("[DetectPythonEnvironments] No portable Python found: %v\n", err)
	}

	systemPython, err := e.detectSystemPython()
	if err == nil {
		log.Printf("[DetectPythonEnvironments] Found system Python: %s\n", systemPython.Path)
		if err := e.db.SavePythonEnvironment(systemPython); err != nil {
			log.Printf("[DetectPythonEnvironments] Failed to save system Python: %v\n", err)
		}
	} else {
		log.Printf("[DetectPythonEnvironments] No system Python found: %v\n", err)
	}

	environments, err := e.db.GetPythonEnvironments()
	if err != nil {
		log.Printf("[DetectPythonEnvironments] Failed to get environments from DB: %v\n", err)
		return []PythonEnvironment{}, err
	}

	log.Printf("[DetectPythonEnvironments] Complete, found %d environments in database\n", len(environments))
	return environments, nil
}

func (e *EnvironmentService) detectPortablePython() (PythonEnvironment, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return PythonEnvironment{}, err
	}

	var appFolder string
	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			appFolder = filepath.Join(localAppData, "cauldron")
		} else {
			appFolder = filepath.Join(homeDir, "AppData", "Local", "cauldron")
		}
	case "darwin":
		appFolder = filepath.Join(homeDir, "Library", "Application Support", "cauldron")
	case "linux":
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome != "" {
			appFolder = filepath.Join(xdgDataHome, "cauldron")
		} else {
			appFolder = filepath.Join(homeDir, ".local", "share", "cauldron")
		}
	default:
		appFolder = filepath.Join(homeDir, ".cauldron")
	}

	platformName := "win"
	if runtime.GOOS == "darwin" {
		platformName = "darwin"
	} else if runtime.GOOS == "linux" {
		platformName = "linux"
	}

	var pythonPath string
	if runtime.GOOS == "windows" {
		pythonPath = filepath.Join(appFolder, "bin", platformName, "python", "python.exe")
	} else {
		pythonPath = filepath.Join(appFolder, "bin", platformName, "python", "bin", "python3")
	}

	if _, err := os.Stat(pythonPath); err != nil {
		return PythonEnvironment{}, err
	}

	version := e.getPythonVersion(pythonPath)

	return PythonEnvironment{
		Name:      "Portable Python",
		Path:      pythonPath,
		Type:      "portable",
		Version:   version,
		IsVirtual: false,
	}, nil
}

func (e *EnvironmentService) detectSystemPython() (PythonEnvironment, error) {
	pythonCmd := "python3"
	if runtime.GOOS == "windows" {
		pythonCmd = "python"
	}

	path, err := exec.LookPath(pythonCmd)
	if err != nil {
		return PythonEnvironment{}, err
	}

	version := e.getPythonVersion(path)

	return PythonEnvironment{
		Name:      "System Python",
		Path:      path,
		Type:      "system",
		Version:   version,
		IsVirtual: false,
	}, nil
}

func (e *EnvironmentService) detectPortableR() (REnvironment, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return REnvironment{}, err
	}

	var appFolder string
	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			appFolder = filepath.Join(localAppData, "cauldron")
		} else {
			appFolder = filepath.Join(homeDir, "AppData", "Local", "cauldron")
		}
	case "darwin":
		appFolder = filepath.Join(homeDir, "Library", "Application Support", "cauldron")
	case "linux":
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome != "" {
			appFolder = filepath.Join(xdgDataHome, "cauldron")
		} else {
			appFolder = filepath.Join(homeDir, ".local", "share", "cauldron")
		}
	default:
		appFolder = filepath.Join(homeDir, ".cauldron")
	}

	platformName := "win"
	if runtime.GOOS == "darwin" {
		platformName = "darwin"
	} else if runtime.GOOS == "linux" {
		platformName = "linux"
	}

	var rPath string
	if runtime.GOOS == "windows" {
		rPath = filepath.Join(appFolder, "bin", platformName, "R-Portable", "bin", "Rscript.exe")
	} else {
		rPath = filepath.Join(appFolder, "bin", platformName, "R-Portable", "bin", "Rscript")
	}

	if _, err := os.Stat(rPath); err != nil {
		return REnvironment{}, err
	}

	version := e.getRVersion(rPath)
	libPath := e.getRLibPath(rPath)

	return REnvironment{
		Name:      "Portable R",
		Path:      rPath,
		Type:      "portable",
		Version:   version,
		LibPath:   libPath,
		IsDefault: false,
	}, nil
}

func (e *EnvironmentService) detectCondaEnvironments() ([]PythonEnvironment, error) {
	var environments []PythonEnvironment

	condaCmd := "conda"
	_, err := exec.LookPath(condaCmd)
	if err != nil {
		return environments, err
	}

	cmd := exec.Command(condaCmd, "env", "list")
	output, err := cmd.Output()
	if err != nil {
		return environments, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			envName := parts[0]
			envPath := parts[len(parts)-1]

			pythonPath := e.getPythonPathInEnv(envPath, true)
			if pythonPath != "" {
				version := e.getPythonVersion(pythonPath)
				environments = append(environments, PythonEnvironment{
					Name:      envName,
					Path:      pythonPath,
					Type:      "conda",
					Version:   version,
					IsVirtual: true,
				})
			}
		}
	}

	return environments, nil
}

func (e *EnvironmentService) detectVirtualEnvs() []PythonEnvironment {
	var environments []PythonEnvironment

	commonLocations := []string{
		filepath.Join(os.Getenv("HOME"), ".virtualenvs"),
		filepath.Join(os.Getenv("HOME"), "venvs"),
		filepath.Join(os.Getenv("USERPROFILE"), ".virtualenvs"),
		filepath.Join(os.Getenv("USERPROFILE"), "venvs"),
	}

	for _, location := range commonLocations {
		if _, err := os.Stat(location); err == nil {
			dirs, _ := os.ReadDir(location)
			for _, dir := range dirs {
				if dir.IsDir() {
					envPath := filepath.Join(location, dir.Name())
					pythonPath := e.getPythonPathInEnv(envPath, false)
					if pythonPath != "" {
						version := e.getPythonVersion(pythonPath)
						environments = append(environments, PythonEnvironment{
							Name:      dir.Name(),
							Path:      pythonPath,
							Type:      "venv",
							Version:   version,
							IsVirtual: true,
						})
					}
				}
			}
		}
	}

	return environments
}

func (e *EnvironmentService) detectPoetryEnvironments() []PythonEnvironment {
	var environments []PythonEnvironment

	poetryCmd := "poetry"
	_, err := exec.LookPath(poetryCmd)
	if err != nil {
		return environments
	}

	cacheDir := filepath.Join(os.Getenv("HOME"), ".cache", "pypoetry", "virtualenvs")
	if runtime.GOOS == "windows" {
		cacheDir = filepath.Join(os.Getenv("APPDATA"), "pypoetry", "Cache", "virtualenvs")
	}

	if _, err := os.Stat(cacheDir); err == nil {
		dirs, _ := os.ReadDir(cacheDir)
		for _, dir := range dirs {
			if dir.IsDir() {
				envPath := filepath.Join(cacheDir, dir.Name())
				pythonPath := e.getPythonPathInEnv(envPath, false)
				if pythonPath != "" {
					version := e.getPythonVersion(pythonPath)
					environments = append(environments, PythonEnvironment{
						Name:      dir.Name(),
						Path:      pythonPath,
						Type:      "poetry",
						Version:   version,
						IsVirtual: true,
					})
				}
			}
		}
	}

	return environments
}

func (e *EnvironmentService) getPythonPathInEnv(envPath string, isConda bool) string {
	var pythonPath string

	if runtime.GOOS == "windows" {
		if isConda {
			pythonPath = filepath.Join(envPath, "python.exe")
		} else {
			pythonPath = filepath.Join(envPath, "Scripts", "python.exe")
		}
	} else {
		pythonPath = filepath.Join(envPath, "bin", "python")
		if _, err := os.Stat(pythonPath); err != nil {
			pythonPath = filepath.Join(envPath, "bin", "python3")
		}
	}

	if _, err := os.Stat(pythonPath); err == nil {
		return pythonPath
	}

	return ""
}

func (e *EnvironmentService) getPythonVersion(pythonPath string) string {
	cmd := exec.Command(pythonPath, "--version")
	hideConsoleWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}

	version := strings.TrimSpace(string(output))
	version = strings.TrimPrefix(version, "Python ")
	return version
}

func (e *EnvironmentService) InstallPythonPackages(pythonPath string, packages []string) error {
	args := append([]string{"-m", "pip", "install"}, packages...)
	cmd := exec.Command(pythonPath, args...)
	hideConsoleWindow(cmd)
	return cmd.Run()
}

func (e *EnvironmentService) InstallPythonRequirements(pythonPath string, requirementsPath string) error {
	e.progressNotifier.EmitStart(ProgressTypeInstall, "python-requirements", "Installing Python packages...")

	cmd := exec.Command(pythonPath, "-m", "pip", "install", "-r", requirementsPath)
	hideConsoleWindow(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		e.progressNotifier.EmitError(ProgressTypeInstall, "python-requirements", "Failed to create stdout pipe", err.Error())
		return err
	}

	if err := cmd.Start(); err != nil {
		e.progressNotifier.EmitError(ProgressTypeInstall, "python-requirements", "Failed to start pip install", err.Error())
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("[PIP] %s\n", line)

		if strings.Contains(line, "Collecting") || strings.Contains(line, "Downloading") || strings.Contains(line, "Installing") {
			e.progressNotifier.EmitProgress(ProgressTypeInstall, "python-requirements", line, 50)
		}
	}

	if err := cmd.Wait(); err != nil {
		e.progressNotifier.EmitError(ProgressTypeInstall, "python-requirements", "Failed to install packages", err.Error())
		return err
	}

	e.progressNotifier.EmitComplete(ProgressTypeInstall, "python-requirements", "Python packages installed successfully")
	return nil
}

func (e *EnvironmentService) DetectREnvironments() ([]REnvironment, error) {
	log.Println("[DetectREnvironments] Starting...")

	portableR, err := e.detectPortableR()
	if err == nil {
		log.Printf("[DetectREnvironments] Found portable R: %s\n", portableR.Path)
		if err := e.db.SaveREnvironment(portableR); err != nil {
			log.Printf("[DetectREnvironments] Failed to save portable R: %v\n", err)
		}
	} else {
		log.Printf("[DetectREnvironments] No portable R found: %v\n", err)
	}

	defaultR, err := e.detectDefaultR()
	if err == nil {
		log.Printf("[DetectREnvironments] Found system R: %s\n", defaultR.Path)
		if err := e.db.SaveREnvironment(defaultR); err != nil {
			log.Printf("[DetectREnvironments] Failed to save system R: %v\n", err)
		}
	} else {
		log.Printf("[DetectREnvironments] No system R found: %v\n", err)
	}

	environments, err := e.db.GetREnvironments()
	if err != nil {
		log.Printf("[DetectREnvironments] Failed to get environments from DB: %v\n", err)
		return []REnvironment{}, err
	}

	log.Printf("[DetectREnvironments] Complete, found %d environments in database\n", len(environments))
	return environments, nil
}

func (e *EnvironmentService) detectDefaultR() (REnvironment, error) {
	rCmd := "Rscript"
	path, err := exec.LookPath(rCmd)
	if err != nil {
		return REnvironment{}, err
	}

	version := e.getRVersion(path)
	libPath := e.getRLibPath(path)

	return REnvironment{
		Name:      "System R",
		Path:      path,
		Type:      "system",
		Version:   version,
		LibPath:   libPath,
		IsDefault: true,
	}, nil
}

func (e *EnvironmentService) detectRenvEnvironments() []REnvironment {
	var environments []REnvironment

	return environments
}

func (e *EnvironmentService) getRVersion(rPath string) string {
	cmd := exec.Command(rPath, "--version")
	hideConsoleWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}

	return "Unknown"
}

func (e *EnvironmentService) getRLibPath(rPath string) string {
	cmd := exec.Command(rPath, "-e", ".libPaths()[1]")
	hideConsoleWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "[1]") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "[1]"))
			path = strings.Trim(path, "\"")
			return path
		}
	}

	return ""
}

func (e *EnvironmentService) InstallRPackages(rPath string, packages []string) error {
	e.progressNotifier.EmitStart(ProgressTypeInstall, "r-packages", "Checking BiocManager...")

	biocManagerInstalled, err := e.checkBiocManagerInstalled(rPath)
	if err != nil {
		e.progressNotifier.EmitError(ProgressTypeInstall, "r-packages", "Failed to check BiocManager", err.Error())
		return err
	}

	if !biocManagerInstalled {
		e.progressNotifier.EmitProgress(ProgressTypeInstall, "r-packages", "Installing BiocManager...", 5)
		installCmd := "install.packages('BiocManager', repos='https://cloud.r-project.org')"
		cmd := exec.Command(rPath, "-e", installCmd)
		hideConsoleWindow(cmd)
		if err := cmd.Run(); err != nil {
			e.progressNotifier.EmitError(ProgressTypeInstall, "r-packages", "Failed to install BiocManager", err.Error())
			return err
		}
	}

	totalPackages := 0
	for _, pkg := range packages {
		if pkg != "BiocManager" {
			totalPackages++
		}
	}

	installed := 0
	for _, pkg := range packages {
		if pkg == "BiocManager" {
			continue
		}

		installed++
		percentage := float64(installed) / float64(totalPackages) * 95.0
		e.progressNotifier.EmitProgress(ProgressTypeInstall, "r-packages",
			fmt.Sprintf("Installing %s (%d/%d)", pkg, installed, totalPackages), percentage)

		installCmd := "BiocManager::install('" + pkg + "')"
		cmd := exec.Command(rPath, "-e", installCmd)
		hideConsoleWindow(cmd)
		if err := cmd.Run(); err != nil {
			e.progressNotifier.EmitError(ProgressTypeInstall, "r-packages",
				fmt.Sprintf("Failed to install %s", pkg), err.Error())
			return err
		}
	}

	e.progressNotifier.EmitComplete(ProgressTypeInstall, "r-packages", "R packages installed successfully")
	return nil
}

func (e *EnvironmentService) GetBundledRequirementsPath(requirementType string) (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	var fileName string
	if requirementType == "python" {
		fileName = "requirements.txt"
	} else if requirementType == "r" {
		fileName = "r_requirements.txt"
	} else {
		return "", fmt.Errorf("unknown requirement type: %s", requirementType)
	}

	requirementsPath := filepath.Join(filepath.Dir(execPath), "scripts", fileName)
	if _, err := os.Stat(requirementsPath); os.IsNotExist(err) {
		requirementsPath = filepath.Join("scripts", fileName)
	}

	if _, err := os.Stat(requirementsPath); os.IsNotExist(err) {
		return "", fmt.Errorf("requirements file not found: %s", fileName)
	}

	return requirementsPath, nil
}

func (e *EnvironmentService) GetExampleFilePath(exampleType string, fileName string) (string, error) {
	log.Printf("[GetExampleFilePath] Requested: %s/%s", exampleType, fileName)
	execPath, err := os.Executable()
	if err != nil {
		log.Printf("[GetExampleFilePath] ERROR: failed to get executable path: %v", err)
		return "", err
	}
	log.Printf("[GetExampleFilePath] Executable path: %s", execPath)

	examplePath := filepath.Join(filepath.Dir(execPath), "examples", exampleType, fileName)
	log.Printf("[GetExampleFilePath] Trying path 1: %s", examplePath)
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		log.Printf("[GetExampleFilePath] Path 1 not found, trying fallback")
		examplePath = filepath.Join("examples", exampleType, fileName)
		log.Printf("[GetExampleFilePath] Trying path 2: %s", examplePath)
	}

	fileInfo, err := os.Stat(examplePath)
	if os.IsNotExist(err) {
		log.Printf("[GetExampleFilePath] ERROR: example file not found")
		return "", fmt.Errorf("example file not found: %s/%s", exampleType, fileName)
	}
	log.Printf("[GetExampleFilePath] Found file: %s (size: %d bytes)", examplePath, fileInfo.Size())

	return examplePath, nil
}

func (e *EnvironmentService) LoadRPackagesFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var packages []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				packageName := strings.Trim(parts[0], "\"")
				packages = append(packages, packageName)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return packages, nil
}

func (e *EnvironmentService) checkBiocManagerInstalled(rPath string) (bool, error) {
	checkCmd := "requireNamespace('BiocManager', quietly = TRUE)"
	cmd := exec.Command(rPath, "-e", checkCmd)
	hideConsoleWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	return strings.Contains(string(output), "[1] TRUE"), nil
}

func (e *EnvironmentService) ListPythonPackages(pythonPath string) ([]string, error) {
	log.Printf("[ListPythonPackages] Listing packages for: %s\n", pythonPath)
	cmd := exec.Command(pythonPath, "-m", "pip", "list", "--format=freeze")
	hideConsoleWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[ListPythonPackages] ERROR: %v\n", err)
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var packages []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			packages = append(packages, line)
		}
	}

	log.Printf("[ListPythonPackages] Found %d packages\n", len(packages))
	return packages, nil
}

func (e *EnvironmentService) ListRPackages(rPath string) ([]string, error) {
	log.Printf("[ListRPackages] Listing packages for: %s\n", rPath)
	listCmd := "cat(installed.packages()[,1], sep='\\n')"
	cmd := exec.Command(rPath, "-e", listCmd)
	hideConsoleWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[ListRPackages] ERROR: %v\n", err)
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var packages []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "[") {
			packages = append(packages, line)
		}
	}

	log.Printf("[ListRPackages] Found %d packages\n", len(packages))
	return packages, nil
}

func (e *EnvironmentService) CreatePythonVirtualEnv(basePythonPath string, venvPath string) error {
	log.Printf("[CreatePythonVirtualEnv] Creating virtual environment at %s using %s\n", venvPath, basePythonPath)

	cmd := exec.Command(basePythonPath, "-m", "venv", venvPath)
	hideConsoleWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[CreatePythonVirtualEnv] ERROR: %v\nOutput: %s\n", err, string(output))
		return fmt.Errorf("failed to create virtual environment: %w", err)
	}

	venvName := filepath.Base(venvPath)
	pythonExe := filepath.Join(venvPath, "Scripts", "python.exe")
	if runtime.GOOS != "windows" {
		pythonExe = filepath.Join(venvPath, "bin", "python")
	}

	venv := VirtualEnvironment{
		Name:           venvName,
		Path:           pythonExe,
		BasePythonPath: basePythonPath,
		CreatedAt:      time.Now().Unix(),
	}

	if err := e.db.GetDB().Create(&venv).Error; err != nil {
		log.Printf("[CreatePythonVirtualEnv] Warning: Failed to save to database: %v\n", err)
	}

	log.Printf("[CreatePythonVirtualEnv] Successfully created virtual environment\n")
	return nil
}

func (e *EnvironmentService) GetVirtualEnvironments() ([]VirtualEnvironment, error) {
	var venvs []VirtualEnvironment
	err := e.db.GetDB().Order("created_at DESC").Find(&venvs).Error
	return venvs, err
}

func (e *EnvironmentService) DeleteVirtualEnvironment(id uint) error {
	return e.db.GetDB().Delete(&VirtualEnvironment{}, id).Error
}
