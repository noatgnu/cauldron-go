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

	"strconv"

	"github.com/noatgnu/cauldron-go/backend/models"
	"gopkg.in/yaml.v3"
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

type packageCacheEntry struct {
	packages  []string
	timestamp time.Time
}

type EnvironmentService struct {
	ctx              context.Context
	db               *DatabaseService
	settingsService  *SettingsService
	progressNotifier *ProgressNotifier
	packageCache     map[string]*packageCacheEntry
	cacheTTL         time.Duration
}

func NewEnvironmentService(ctx context.Context, db *DatabaseService, settingsService *SettingsService, progressNotifier *ProgressNotifier) *EnvironmentService {
	return &EnvironmentService{
		ctx:              ctx,
		db:               db,
		settingsService:  settingsService,
		progressNotifier: progressNotifier,
		packageCache:     make(map[string]*packageCacheEntry),
		cacheTTL:         5 * time.Minute,
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

	venvs, err := e.GetVirtualEnvironments()
	if err != nil {
		log.Printf("[DetectPythonEnvironments] Failed to get virtual environments: %v\n", err)
	} else {
		log.Printf("[DetectPythonEnvironments] Found %d virtual environments, adding to list\n", len(venvs))
		for _, venv := range venvs {
			version := e.getPythonVersion(venv.Path)
			environments = append(environments, PythonEnvironment{
				Name:      venv.Name,
				Path:      venv.Path,
				Type:      "venv",
				Version:   version,
				IsVirtual: true,
			})
		}
	}

	log.Printf("[DetectPythonEnvironments] Complete, found %d total environments (system + venvs)\n", len(environments))
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
		},
		nil
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
		},
		nil
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
		},
		nil
}

func (e *EnvironmentService) detectCondaEnvironments() ([]PythonEnvironment, error) {
	var environments []PythonEnvironment

	condaCmd := "conda"
	_, err := exec.LookPath(condaCmd)
	if err != nil {
		return environments, err
	}

	cmd := exec.Command(condaCmd, "env", "list")
	hideConsoleWindow(cmd)
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

	cacheKey := "python:" + pythonPath
	delete(e.packageCache, cacheKey)
	log.Printf("[InstallPythonRequirements] Invalidated package cache for %s\n", pythonPath)

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
		},
		nil
}

func (e *EnvironmentService) detectRenvEnvironments() []RenvEnvironment {
	var environments []RenvEnvironment

	var renvDir string
	globalConfig := e.settingsService.GetConfig()
	if globalConfig.RenvStoragePath != "" {
		renvDir = globalConfig.RenvStoragePath
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return environments
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

		renvDir = filepath.Join(appFolder, "renv-projects")
	}

	if _, err := os.Stat(renvDir); err == nil {
		dirs, _ := os.ReadDir(renvDir)
		for _, dir := range dirs {
			if dir.IsDir() {
				projectPath := filepath.Join(renvDir, dir.Name())
				lockfilePath := filepath.Join(projectPath, "renv.lock")
				if _, err := os.Stat(lockfilePath); err == nil {
					env := e.getRenvEnvironmentInfo(projectPath, dir.Name())
					if env != nil {
						environments = append(environments, *env)
					}
				}
			}
		}
	}

	return environments
}

func (e *EnvironmentService) getRenvEnvironmentInfo(projectPath string, name string) *RenvEnvironment {
	rPath, err := e.getActiveRPath()
	if err != nil {
		return nil
	}

	renvDir := filepath.Join(projectPath, "renv")
	if _, err := os.Stat(renvDir); err != nil {
		return nil
	}

	return &RenvEnvironment{
		Name:        name,
		Path:        projectPath,
		ProjectPath: projectPath,
		BaseRPath:   rPath,
		CreatedAt:   time.Now().Unix(),
	}
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

	cacheKey := "r:" + rPath
	delete(e.packageCache, cacheKey)
	log.Printf("[InstallRPackages] Invalidated package cache for %s\n", rPath)

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

	cacheKey := "python:" + pythonPath
	if cached, ok := e.packageCache[cacheKey]; ok {
		if time.Since(cached.timestamp) < e.cacheTTL {
			log.Printf("[ListPythonPackages] Using cached result (%d packages)\n", len(cached.packages))
			return cached.packages, nil
		}
	}

	log.Printf("[ListPythonPackages] Cache miss, fetching from pip...\n")
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

	e.packageCache[cacheKey] = &packageCacheEntry{
		packages:  packages,
		timestamp: time.Now(),
	}

	log.Printf("[ListPythonPackages] Found %d packages (cached for %v)\n", len(packages), e.cacheTTL)
	return packages, nil
}

func (e *EnvironmentService) ListRPackages(rPath string) ([]string, error) {
	log.Printf("[ListRPackages] Listing packages for: %s\n", rPath)

	cacheKey := "r:" + rPath
	if cached, ok := e.packageCache[cacheKey]; ok {
		if time.Since(cached.timestamp) < e.cacheTTL {
			log.Printf("[ListRPackages] Using cached result (%d packages)\n", len(cached.packages))
			return cached.packages, nil
		}
	}

	log.Printf("[ListRPackages] Cache miss, fetching from R...\n")
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

	e.packageCache[cacheKey] = &packageCacheEntry{
		packages:  packages,
		timestamp: time.Now(),
	}

	log.Printf("[ListRPackages] Found %d packages (cached for %v)\n", len(packages), e.cacheTTL)
	return packages, nil
}

func (e *EnvironmentService) loadPluginDefinition(pluginYamlPath string) (*models.PluginDefinition, error) {
	data, err := os.ReadFile(pluginYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin.yaml: %w", err)
	}

	var pluginDef models.PluginDefinition
	if err := yaml.Unmarshal(data, &pluginDef); err != nil {
		return nil, fmt.Errorf("failed to parse plugin.yaml: %w", err)
	}

	return &pluginDef, nil
}

func (e *EnvironmentService) installPythonPackagesList(pythonPath string, packages []string) error {
	if len(packages) == 0 {
		return nil
	}

	e.progressNotifier.EmitStart(ProgressTypeInstall, "python-packages", "Installing Python packages...")

	args := append([]string{"-m", "pip", "install"}, packages...)
	cmd := exec.Command(pythonPath, args...)
	hideConsoleWindow(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		e.progressNotifier.EmitError(ProgressTypeInstall, "python-packages", "Failed to create stdout pipe", err.Error())
		return err
	}

	if err := cmd.Start(); err != nil {
		e.progressNotifier.EmitError(ProgressTypeInstall, "python-packages", "Failed to start pip install", err.Error())
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("[PIP] %s\n", line)
		e.progressNotifier.EmitProgress(ProgressTypeInstall, "python-packages", line, 50)
	}

	if err := cmd.Wait(); err != nil {
		e.progressNotifier.EmitError(ProgressTypeInstall, "python-packages", "Failed to install packages", err.Error())
		return err
	}

	e.progressNotifier.EmitComplete(ProgressTypeInstall, "python-packages", "Packages installed successfully")
	return nil
}

func (e *EnvironmentService) CreatePythonVirtualEnv(basePythonPath string, venvPath string, pluginID string, pluginFolderPath string) error {
	log.Printf("[CreatePythonVirtualEnv] Creating virtual environment at %s using %s for plugin %s (folder: %s)\n", venvPath, basePythonPath, pluginID, pluginFolderPath)

	venvName := filepath.Base(venvPath)
	e.progressNotifier.EmitStart(ProgressTypeInstall, "python-venv", fmt.Sprintf("Creating virtual environment '%s'...", venvName))

	if _, err := os.Stat(venvPath); err == nil {
		log.Printf("[CreatePythonVirtualEnv] Virtual environment directory already exists at %s, will recreate with --clear flag", venvPath)
	}

	e.progressNotifier.EmitProgress(ProgressTypeInstall, "python-venv", "Setting up virtual environment structure...", 20)
	cmd := exec.Command(basePythonPath, "-m", "venv", "--clear", venvPath)
	hideConsoleWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[CreatePythonVirtualEnv] ERROR: %v\nOutput: %s\n", err, string(output))
		e.progressNotifier.EmitError(ProgressTypeInstall, "python-venv", "Failed to create virtual environment", err.Error())
		return fmt.Errorf("failed to create virtual environment: %w", err)
	}

	pythonExe := filepath.Join(venvPath, "Scripts", "python.exe")
	if runtime.GOOS != "windows" {
		pythonExe = filepath.Join(venvPath, "bin", "python")
	}

	// Auto-install plugin requirements if pluginID provided
	if pluginID != "" && pluginFolderPath != "" {
		e.progressNotifier.EmitProgress(ProgressTypeInstall, "python-venv", "Installing plugin requirements...", 40)

		// Try to load plugin definition to get inline packages
		pluginYamlPath := filepath.Join(pluginFolderPath, "plugin.yaml")
		var inlinePackages []string
		var requirementsFile string

		log.Printf("[CreatePythonVirtualEnv] Looking for plugin.yaml at: %s", pluginYamlPath)
		if pluginDef, err := e.loadPluginDefinition(pluginYamlPath); err == nil {
			inlinePackages = pluginDef.Execution.Requirements.Packages
			requirementsFile = pluginDef.Execution.Requirements.PythonRequirementsFile
			log.Printf("[CreatePythonVirtualEnv] Found plugin definition - inline packages: %v, requirements file: %s", inlinePackages, requirementsFile)
		} else {
			log.Printf("[CreatePythonVirtualEnv] Failed to load plugin definition from %s: %v", pluginYamlPath, err)
		}

		// Use inline packages if available
		if len(inlinePackages) > 0 {
			log.Printf("[CreatePythonVirtualEnv] Installing inline packages from plugin.yaml: %v\n", inlinePackages)
			if err := e.installPythonPackagesList(pythonExe, inlinePackages); err != nil {
				log.Printf("[CreatePythonVirtualEnv] Warning: Failed to install inline packages: %v\n", err)
			}
		} else if requirementsFile != "" {
			// Use explicit pythonRequirementsFile from plugin.yaml
			requirementsPath := filepath.Join(pluginFolderPath, requirementsFile)
			log.Printf("[CreatePythonVirtualEnv] Looking for requirements file at: %s", requirementsPath)
			if _, err := os.Stat(requirementsPath); err == nil {
				log.Printf("[CreatePythonVirtualEnv] Installing requirements from %s\n", requirementsPath)
				if err := e.InstallPythonRequirements(pythonExe, requirementsPath); err != nil {
					log.Printf("[CreatePythonVirtualEnv] Warning: Failed to install plugin requirements: %v\n", err)
				}
			} else {
				log.Printf("[CreatePythonVirtualEnv] Warning: Specified pythonRequirementsFile '%s' not found at %s\n", requirementsFile, requirementsPath)
			}
		} else {
			log.Printf("[CreatePythonVirtualEnv] No package requirements specified for plugin %s\n", pluginID)
		}
	} else if pluginID != "" && pluginFolderPath == "" {
		log.Printf("[CreatePythonVirtualEnv] Plugin ID provided but no folder path - skipping package installation for %s\n", pluginID)
	}

	e.progressNotifier.EmitProgress(ProgressTypeInstall, "python-venv", "Saving environment configuration...", 90)

	var existingVenv VirtualEnvironment
	err = e.db.GetDB().Where("path = ?", pythonExe).First(&existingVenv).Error
	if err == nil {
		log.Printf("[CreatePythonVirtualEnv] Found existing venv with path %s (ID: %d), deleting old record", pythonExe, existingVenv.ID)
		if err := e.db.GetDB().Delete(&existingVenv).Error; err != nil {
			log.Printf("[CreatePythonVirtualEnv] Warning: Failed to delete existing venv record: %v", err)
		}
	}

	venv := VirtualEnvironment{
		Name:           venvName,
		Path:           pythonExe,
		BasePythonPath: basePythonPath,
		CreatedAt:      time.Now().Unix(),
	}

	log.Printf("[CreatePythonVirtualEnv] Attempting to save venv to database - Name: %s, Path: %s, BasePythonPath: %s", venvName, pythonExe, basePythonPath)
	if err := e.db.GetDB().Create(&venv).Error; err != nil {
		log.Printf("[CreatePythonVirtualEnv] ERROR: Failed to save to database: %v\n", err)
		e.progressNotifier.EmitError(ProgressTypeInstall, "python-venv", "Failed to save virtual environment to database", err.Error())
		return fmt.Errorf("failed to save virtual environment to database: %w", err)
	}

	log.Printf("[CreatePythonVirtualEnv] Successfully created and saved virtual environment with ID: %d\n", venv.ID)
	e.progressNotifier.EmitComplete(ProgressTypeInstall, "python-venv", fmt.Sprintf("Virtual environment '%s' created successfully", venvName))
	return nil
}

func (e *EnvironmentService) GetVirtualEnvironments() ([]VirtualEnvironment, error) {
	var venvs []VirtualEnvironment
	err := e.db.GetDB().Order("created_at DESC").Find(&venvs).Error
	if err != nil {
		log.Printf("[GetVirtualEnvironments] ERROR: Failed to query virtual environments: %v", err)
		return nil, err
	}
	log.Printf("[GetVirtualEnvironments] Found %d virtual environments", len(venvs))
	for i, venv := range venvs {
		log.Printf("[GetVirtualEnvironments] [%d] ID=%d, Name=%s, Path=%s", i, venv.ID, venv.Name, venv.Path)
	}
	return venvs, nil
}

func (e *EnvironmentService) DeleteVirtualEnvironment(id uint) error {
	log.Printf("[DeleteVirtualEnvironment] Deleting environment with ID: %d\n", id)

	e.db.GetDB().Where("environment_id = ? AND environment_type = ?", id, "python").Delete(&PluginEnvironmentBinding{})

	return e.db.GetDB().Delete(&VirtualEnvironment{}, id).Error
}

func (e *EnvironmentService) getActiveRPath() (string, error) {
	cfg := e.db.GetDB()
	var rEnv REnvironmentDB
	if err := cfg.Where("is_active = ?", true).First(&rEnv).Error; err != nil {
		return "", fmt.Errorf("no active R environment found")
	}
	return rEnv.Path, nil
}

func (e *EnvironmentService) CreateRenvEnvironment(name string, packages []string, pluginID string, useCache bool) error {
	log.Printf("[CreateRenvEnvironment] Creating renv environment: %s for plugin: %s (useCache: %v)\n", name, pluginID, useCache)

	rPath, err := e.getActiveRPath()
	if err != nil {
		return fmt.Errorf("failed to get active R path: %w", err)
	}

	cacheDir := ""
	globalConfig := e.settingsService.GetConfig()
	// Cache is only used if both Global and Per-Environment toggles are TRUE
	if globalConfig.UseRenvCache && useCache {
		cacheDir, _ = e.getRenvCachePath()
	}

	var renvDir string
	if globalConfig.RenvStoragePath != "" {
		renvDir = globalConfig.RenvStoragePath
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
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

		renvDir = filepath.Join(appFolder, "renv-projects")
	}

	if err := os.MkdirAll(renvDir, 0755); err != nil {
		return fmt.Errorf("failed to create renv projects directory: %w", err)
	}

	projectPath := filepath.Join(renvDir, name)
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	e.progressNotifier.EmitStart(ProgressTypeInstall, "renv-init", "Initializing renv...")

	// Attempt to resolve numeric ID to plugin string ID (folder name)
	resolvedPluginID := pluginID
	if id, err := strconv.ParseUint(pluginID, 10, 64); err == nil {
		var pluginRegistry models.PluginRegistry
		if err := e.db.GetDB().First(&pluginRegistry, id).Error; err == nil {
			resolvedPluginID = pluginRegistry.PluginID
		}
	}

	// 1. Check if the source plugin has a renv.lock we can use
	pluginLockPath := filepath.Join("plugins", resolvedPluginID, "renv.lock")
	targetLockPath := filepath.Join(projectPath, "renv.lock")
	hasExistingLock := false
	if _, err := os.Stat(pluginLockPath); err == nil {
		log.Printf("[CreateRenvEnvironment] Found existing renv.lock for plugin %s, copying...\n", resolvedPluginID)
		if lockData, err := os.ReadFile(pluginLockPath); err == nil {
			if err := os.WriteFile(targetLockPath, lockData, 0644); err == nil {
				hasExistingLock = true
			}
		}
	}

	initCmd := fmt.Sprintf("setwd('%s'); if (!requireNamespace('renv', quietly = TRUE)) install.packages('renv', repos='https://cloud.r-project.org'); renv::init(bare = TRUE)", strings.ReplaceAll(projectPath, "\\", "/"))
	if hasExistingLock {
		// If we have a lock file, we just need to activate renv
		initCmd = fmt.Sprintf("setwd('%s'); if (!requireNamespace('renv', quietly = TRUE)) install.packages('renv', repos='https://cloud.r-project.org'); renv::activate()", strings.ReplaceAll(projectPath, "\\", "/"))
	}

	cmd := exec.Command(rPath, "-e", initCmd)
	hideConsoleWindow(cmd)

	if cacheDir != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("RENV_PATHS_CACHE=%s", cacheDir))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		e.progressNotifier.EmitError(ProgressTypeInstall, "renv-init", "Failed to initialize renv", err.Error())
		log.Printf("[CreateRenvEnvironment] renv init failed: %v\nOutput: %s", err, string(output))
		return fmt.Errorf("failed to initialize renv: %w\n%s", err, string(output))
	}

	// 2. Use restore if lock file exists, otherwise proceed to manual install
	if hasExistingLock {
		e.progressNotifier.EmitProgress(ProgressTypeInstall, "renv-init", "Restoring environment from renv.lock...", 50)
		restoreCmd := fmt.Sprintf("Sys.setenv(RENV_PROJECT='%s'); source('%s/renv/activate.R'); renv::restore(confirm = FALSE)",
			strings.ReplaceAll(projectPath, "\\", "/"),
			strings.ReplaceAll(projectPath, "\\", "/"))
		cmdR := exec.Command(rPath, "-e", restoreCmd)
		hideConsoleWindow(cmdR)
		envRestore := os.Environ()
		if cacheDir != "" {
			envRestore = append(envRestore, fmt.Sprintf("RENV_PATHS_CACHE=%s", cacheDir))
		}
		envRestore = append(envRestore, fmt.Sprintf("RENV_PROJECT=%s", projectPath))
		cmdR.Env = envRestore
		if out, err := cmdR.CombinedOutput(); err != nil {
			log.Printf("[CreateRenvEnvironment] renv::restore failed: %v\nOutput: %s", err, string(out))
			// Fallback to manual installation if restore fails
		} else {
			log.Printf("[CreateRenvEnvironment] renv::restore completed successfully")
		}
	}

	e.progressNotifier.EmitComplete(ProgressTypeInstall, "renv-init", "renv initialized successfully")

	// Auto-install plugin requirements if no lock file was used or for extra packages
	var packagesToInstall []string
	if pluginID != "" {
		// Try to load plugin definition to get inline packages
		pluginYamlPath := filepath.Join("plugins", resolvedPluginID, "plugin.yaml")
		var inlinePackages []string
		var packagesFile string

		if pluginDef, err := e.loadPluginDefinition(pluginYamlPath); err == nil {
			inlinePackages = pluginDef.Execution.Requirements.Packages
			packagesFile = pluginDef.Execution.Requirements.RPackagesFile
		}

		// Use inline packages if available
		if len(inlinePackages) > 0 {
			log.Printf("[CreateRenvEnvironment] Loading inline packages from plugin.yaml: %v\n", inlinePackages)
			packagesToInstall = append(packagesToInstall, inlinePackages...)
		} else if packagesFile != "" {
			// Use explicit rPackagesFile from plugin.yaml
			packagesPath := filepath.Join("plugins", resolvedPluginID, packagesFile)
			if _, err := os.Stat(packagesPath); err == nil {
				log.Printf("[CreateRenvEnvironment] Loading packages from %s\n", packagesPath)
				pluginPackages, err := e.LoadRPackagesFromFile(packagesPath)
				if err != nil {
					log.Printf("[CreateRenvEnvironment] Warning: Failed to load plugin packages: %v\n", err)
				} else {
					packagesToInstall = append(packagesToInstall, pluginPackages...)
				}
			} else {
				log.Printf("[CreateRenvEnvironment] Warning: Specified rPackagesFile '%s' not found\n", packagesPath)
			}
		} else {
			log.Printf("[CreateRenvEnvironment] No package requirements specified for plugin %s (resolved from %s)\n", resolvedPluginID, pluginID)
		}
	}

	if len(packages) > 0 {
		packagesToInstall = append(packagesToInstall, packages...)
	}

	if len(packagesToInstall) > 0 {
		if err := e.InstallRenvPackages(projectPath, rPath, packagesToInstall, useCache); err != nil {
			return fmt.Errorf("failed to install packages: %w", err)
		}
	}

	renvEnv := RenvEnvironment{
		Name:           name,
		Path:           projectPath,
		ProjectPath:    projectPath,
		BaseRPath:      rPath,
		UseGlobalCache: useCache,
		CreatedAt:      time.Now().Unix(),
	}

	if err := e.db.SaveRenvEnvironment(renvEnv); err != nil {
		log.Printf("[CreateRenvEnvironment] Warning: Failed to save to database: %v\n", err)
	}

	log.Printf("[CreateRenvEnvironment] Successfully created renv environment at %s\n", projectPath)
	return nil
}

func (e *EnvironmentService) InstallRenvPackages(projectPath string, rPath string, packages []string, useCache bool) error {
	log.Printf("[InstallRenvPackages] Installing %d packages in renv project: %s (useCache: %v)\n", len(packages), projectPath, useCache)

	cacheDir := ""
	globalConfig := e.settingsService.GetConfig()
	if globalConfig.UseRenvCache && useCache {
		cacheDir, _ = e.getRenvCachePath()
	}

	e.progressNotifier.EmitStart(ProgressTypeInstall, "renv-packages", "Initializing renv library...")

	// 1. Get base packages to avoid re-installing (matches original logic)
	basePkgCmd := "cat(rownames(installed.packages(priority='base')), sep='\\n')"
	cmdBase := exec.Command(rPath, "-e", basePkgCmd)
	hideConsoleWindow(cmdBase)
	baseOut, _ := cmdBase.Output()
	basePackages := strings.Split(string(baseOut), "\n")
	baseMap := make(map[string]bool)
	for _, p := range basePackages {
		baseMap[strings.TrimSpace(p)] = true
	}

	totalPackages := len(packages)
	for i, pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" || baseMap[pkg] {
			continue
		}

		percentage := float64(i+1) / float64(totalPackages) * 95.0
		e.progressNotifier.EmitProgress(ProgressTypeInstall, "renv-packages",
			fmt.Sprintf("Installing %s (%d/%d)", pkg, i+1, totalPackages), percentage)

		var installCmd string
		// Special case: Rmpi (from original logic)
		if pkg == "Rmpi" {
			installCmd = fmt.Sprintf("Sys.setenv(RENV_PROJECT='%s'); source('%s/renv/activate.R'); renv::install('Rmpi', configure.args='ORTED=prted'); cat('Library paths:', .libPaths(), sep='\\n')",
				strings.ReplaceAll(projectPath, "\\", "/"),
				strings.ReplaceAll(projectPath, "\\", "/"))
		} else {
			// Try standard renv::install first
			installCmd = fmt.Sprintf("Sys.setenv(RENV_PROJECT='%s'); source('%s/renv/activate.R'); renv::install('%s'); cat('Library paths:', .libPaths(), sep='\\n')",
				strings.ReplaceAll(projectPath, "\\", "/"),
				strings.ReplaceAll(projectPath, "\\", "/"),
				pkg)
		}

		cmd := exec.Command(rPath, "-e", installCmd)
		hideConsoleWindow(cmd)
		env := os.Environ()
		if cacheDir != "" {
			env = append(env, fmt.Sprintf("RENV_PATHS_CACHE=%s", cacheDir))
		}
		env = append(env, fmt.Sprintf("RENV_PROJECT=%s", projectPath))
		cmd.Env = env

		output, err := cmd.CombinedOutput()
		log.Printf("[InstallRenvPackages] Output for %s:\n%s", pkg, string(output))
		if err != nil {
			log.Printf("[InstallRenvPackages] Standard install failed for %s, trying bioc:: fallback...", pkg)
			// Fallback: Try explicit Bioconductor prefix
			biocCmd := fmt.Sprintf("Sys.setenv(RENV_PROJECT='%s'); source('%s/renv/activate.R'); renv::install('bioc::%s'); cat('Library paths:', .libPaths(), sep='\\n')",
				strings.ReplaceAll(projectPath, "\\", "/"),
				strings.ReplaceAll(projectPath, "\\", "/"),
				pkg)
			cmdBioc := exec.Command(rPath, "-e", biocCmd)
			hideConsoleWindow(cmdBioc)
			cmdBioc.Env = env
			output, err = cmdBioc.CombinedOutput()
			if err != nil {
				log.Printf("[InstallRenvPackages] Failed to install %s: %v\nOutput: %s", pkg, err, string(output))
				// We log and continue like the original script to ensure other packages get a chance
				e.progressNotifier.EmitProgress(ProgressTypeInstall, "renv-packages", fmt.Sprintf("Warning: Failed to install %s", pkg), percentage)
				continue
			}
		}
		log.Printf("[InstallRenvPackages] Installed %s successfully", pkg)
	}

	e.progressNotifier.EmitProgress(ProgressTypeInstall, "renv-packages", "Creating renv snapshot...", 97)
	snapshotCmd := fmt.Sprintf("Sys.setenv(RENV_PROJECT='%s'); source('%s/renv/activate.R'); renv::snapshot(confirm = FALSE)",
		strings.ReplaceAll(projectPath, "\\", "/"),
		strings.ReplaceAll(projectPath, "\\", "/"))
	cmd := exec.Command(rPath, "-e", snapshotCmd)
	hideConsoleWindow(cmd)
	envSnap := os.Environ()
	if cacheDir != "" {
		envSnap = append(envSnap, fmt.Sprintf("RENV_PATHS_CACHE=%s", cacheDir))
	}
	envSnap = append(envSnap, fmt.Sprintf("RENV_PROJECT=%s", projectPath))
	cmd.Env = envSnap
	if err := cmd.Run(); err != nil {
		log.Printf("[InstallRenvPackages] Warning: snapshot failed: %v", err)
	}

	e.progressNotifier.EmitComplete(ProgressTypeInstall, "renv-packages", "R packages installed successfully")
	log.Printf("[InstallRenvPackages] Package installation complete")
	return nil
}

func (e *EnvironmentService) GetRenvEnvironments() ([]RenvEnvironment, error) {
	log.Println("[GetRenvEnvironments] Starting...")

	detectedEnvs := e.detectRenvEnvironments()
	for _, env := range detectedEnvs {
		if err := e.db.SaveRenvEnvironment(env); err != nil {
			log.Printf("[GetRenvEnvironments] Failed to save environment %s: %v\n", env.Name, err)
		}
	}

	envs, err := e.db.GetRenvEnvironments()
	if err != nil {
		log.Printf("[GetRenvEnvironments] Failed to get environments from DB: %v\n", err)
		return []RenvEnvironment{}, err
	}

	log.Printf("[GetRenvEnvironments] Complete, found %d environments\n", len(envs))
	return envs, nil
}

func (e *EnvironmentService) DeleteRenvEnvironment(id uint) error {
	log.Printf("[DeleteRenvEnvironment] Deleting R environment with ID: %d\n", id)

	env, err := e.db.GetRenvEnvironmentByID(id)
	if err != nil {
		return err
	}

	e.db.GetDB().Where("environment_id = ? AND environment_type = ?", id, "r").Delete(&PluginEnvironmentBinding{})

	if err := os.RemoveAll(env.ProjectPath); err != nil {
		log.Printf("[DeleteRenvEnvironment] Warning: Failed to delete directory %s: %v", env.ProjectPath, err)
	}

	return e.db.DeleteRenvEnvironment(id)
}

func (e *EnvironmentService) GetRenvLibPath(projectPath string) string {
	renvLibPath := filepath.Join(projectPath, "renv", "library")

	dirs, err := os.ReadDir(renvLibPath)
	if err != nil || len(dirs) == 0 {
		return ""
	}

	platformDir := filepath.Join(renvLibPath, dirs[0].Name())
	if _, err := os.Stat(platformDir); err == nil {
		return platformDir
	}

	return renvLibPath
}

func (e *EnvironmentService) getDefaultAppFolder() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
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

	return appFolder, nil
}

func (e *EnvironmentService) getRenvCachePath() (string, error) {
	appFolder, err := e.getDefaultAppFolder()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(appFolder, "renv-cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	return cacheDir, nil
}

func (e *EnvironmentService) getRenvStorageDir() (string, error) {
	globalConfig := e.settingsService.GetConfig()
	if globalConfig.RenvStoragePath != "" {
		return globalConfig.RenvStoragePath, nil
	}

	appFolder, err := e.getDefaultAppFolder()
	if err != nil {
		return "", err
	}

	return filepath.Join(appFolder, "renv-projects"), nil
}

func (e *EnvironmentService) getVenvStorageDir() (string, error) {
	globalConfig := e.settingsService.GetConfig()
	if globalConfig.VenvStoragePath != "" {
		return globalConfig.VenvStoragePath, nil
	}

	appFolder, err := e.getDefaultAppFolder()
	if err != nil {
		return "", err
	}

	return filepath.Join(appFolder, "venv-projects"), nil
}
