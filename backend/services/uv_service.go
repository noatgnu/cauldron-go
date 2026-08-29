package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"
)

type UvService struct {
	ctx              context.Context
	fileService      *FileService
	db               *DatabaseService
	settingsService  *SettingsService
	progressNotifier *ProgressNotifier
	releasesURL      string
}

type UvPythonVersion struct {
	Version        string `json:"version"`
	Path           string `json:"path"`
	Implementation string `json:"implementation"`
}

func NewUvService(ctx context.Context, fileService *FileService, db *DatabaseService, settingsService *SettingsService) *UvService {
	return &UvService{
		ctx:              ctx,
		fileService:      fileService,
		db:               db,
		settingsService:  settingsService,
		progressNotifier: NewProgressNotifier(ctx),
	}
}

// GetUvReleaseAsset maps a Go OS/arch pair to the matching astral-sh/uv release asset name.
func GetUvReleaseAsset(goos, goarch string) (string, error) {
	switch {
	case goos == "linux" && goarch == "amd64":
		return "uv-x86_64-unknown-linux-gnu.tar.gz", nil
	case goos == "linux" && goarch == "arm64":
		return "uv-aarch64-unknown-linux-gnu.tar.gz", nil
	case goos == "darwin" && goarch == "amd64":
		return "uv-x86_64-apple-darwin.tar.gz", nil
	case goos == "darwin" && goarch == "arm64":
		return "uv-aarch64-apple-darwin.tar.gz", nil
	case goos == "windows" && goarch == "amd64":
		return "uv-x86_64-pc-windows-msvc.zip", nil
	default:
		return "", fmt.Errorf("no uv release asset known for %s/%s", goos, goarch)
	}
}

func (u *UvService) uvBinaryDir() (string, error) {
	appFolder, err := getAppDataFolder()
	if err != nil {
		return "", err
	}
	return filepath.Join(appFolder, "bin", translatePlatform(goruntime.GOOS), "uv"), nil
}

// GetUvPath returns the path to the managed uv binary, or an error if it isn't installed.
func (u *UvService) GetUvPath() (string, error) {
	binDir, err := u.uvBinaryDir()
	if err != nil {
		return "", err
	}

	uvBinaryName := "uv"
	if goruntime.GOOS == "windows" {
		uvBinaryName = "uv.exe"
	}

	uvPath := filepath.Join(binDir, uvBinaryName)
	if _, err := os.Stat(uvPath); os.IsNotExist(err) {
		return "", fmt.Errorf("uv is not installed at %s", uvPath)
	}

	return uvPath, nil
}

// resolveUvPath returns the app-managed uv path, falling back to system PATH; every uv-invoking method must use this (not GetUvPath) to match IsUvAvailable.
func (u *UvService) resolveUvPath() (string, error) {
	if path, err := u.GetUvPath(); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("uv"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("uv is not installed")
}

// IsUvAvailable reports whether a managed uv install exists, falling back to checking PATH.
func (u *UvService) IsUvAvailable() bool {
	_, err := u.resolveUvPath()
	return err == nil
}

// DownloadUv fetches, verifies, and installs the uv binary for the current platform.
func (u *UvService) DownloadUv() error {
	assetName, err := GetUvReleaseAsset(goruntime.GOOS, goruntime.GOARCH)
	if err != nil {
		return err
	}

	releaseURL := u.releasesURL
	if releaseURL == "" {
		releaseURL = "https://api.github.com/repos/astral-sh/uv/releases/latest"
	}

	log.Printf("[DownloadUv] Requesting %s for asset %s", releaseURL, assetName)

	release, err := fetchGitHubRelease(releaseURL)
	if err != nil {
		return err
	}

	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			if asset.BrowserDownloadURL != "" {
				downloadURL = asset.BrowserDownloadURL
			} else {
				downloadURL = asset.URL
			}
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("no matching uv release asset found for %s", assetName)
	}

	appFolder, err := getAppDataFolder()
	if err != nil {
		return err
	}

	tempFolder := filepath.Join(appFolder, "temp")
	binDir, err := u.uvBinaryDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(tempFolder, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	tempFilePath := filepath.Join(tempFolder, assetName)
	checksumFilePath := tempFilePath + ".sha256"

	u.progressNotifier.EmitStart(ProgressTypeDownload, assetName, fmt.Sprintf("Downloading %s", assetName))

	if err := downloadFileWithProgress(downloadURL, tempFilePath, u.progressNotifier, assetName); err != nil {
		return err
	}

	u.progressNotifier.EmitComplete(ProgressTypeDownload, assetName, fmt.Sprintf("Downloaded %s", assetName))

	checksumURL := downloadURL + ".sha256"
	expectedHash, err := downloadChecksum(checksumURL, checksumFilePath)
	if err != nil {
		log.Printf("[DownloadUv] Warning: failed to download checksum, proceeding without verification: %v", err)
	} else {
		u.progressNotifier.EmitStart(ProgressTypeInstall, "verify-"+assetName, "Verifying download integrity")

		actualHash, err := calculateSHA256(tempFilePath)
		if err != nil {
			u.progressNotifier.EmitError(ProgressTypeInstall, "verify-"+assetName, fmt.Sprintf("Hash calculation failed: %v", err), "")
			return fmt.Errorf("failed to calculate file hash: %w", err)
		}
		if actualHash != expectedHash {
			u.progressNotifier.EmitError(ProgressTypeInstall, "verify-"+assetName, "Hash verification failed", "")
			return fmt.Errorf("hash verification failed: expected %s, got %s", expectedHash, actualHash)
		}

		u.progressNotifier.EmitComplete(ProgressTypeInstall, "verify-"+assetName, "Integrity verified")
	}

	extractDir := filepath.Join(tempFolder, "uv-extract")
	os.RemoveAll(extractDir)
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return err
	}

	if strings.HasSuffix(assetName, ".zip") {
		if err := u.fileService.ExtractZip(tempFilePath, extractDir); err != nil {
			return err
		}
	} else {
		if err := u.fileService.ExtractTarGz(tempFilePath, extractDir); err != nil {
			return err
		}
	}

	// uv release archives nest their contents one directory deep, e.g. uv-x86_64-unknown-linux-gnu/uv
	sourceDir := extractDir
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return err
	}
	if len(entries) == 1 && entries[0].IsDir() {
		sourceDir = filepath.Join(extractDir, entries[0].Name())
	}

	uvBinaryName := "uv"
	if goruntime.GOOS == "windows" {
		uvBinaryName = "uv.exe"
	}

	srcBinary := filepath.Join(sourceDir, uvBinaryName)
	if _, err := os.Stat(srcBinary); err != nil {
		return fmt.Errorf("uv binary not found in extracted archive at %s: %w", srcBinary, err)
	}

	destBinary := filepath.Join(binDir, uvBinaryName)
	os.Remove(destBinary)
	if err := os.Rename(srcBinary, destBinary); err != nil {
		return err
	}
	if goruntime.GOOS != "windows" {
		if err := os.Chmod(destBinary, 0755); err != nil {
			return err
		}
	}

	os.RemoveAll(extractDir)
	os.Remove(tempFilePath)
	os.Remove(checksumFilePath)

	u.progressNotifier.EmitComplete(ProgressTypeInstall, "uv-install", "uv installed successfully")
	log.Printf("[DownloadUv] Installed uv at %s", destBinary)

	return nil
}

// ListUvManagedPythons returns the Python versions uv has already installed.
func (u *UvService) ListUvManagedPythons() ([]UvPythonVersion, error) {
	uvPath, err := u.resolveUvPath()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(uvPath, "python", "list", "--only-installed", "--output-format", "json")
	hideConsoleWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list uv-managed Python versions: %w", err)
	}

	type uvPythonListEntry struct {
		Version        string `json:"version"`
		Path           string `json:"path"`
		Implementation string `json:"implementation"`
	}
	var entries []uvPythonListEntry
	if err := json.Unmarshal(output, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse uv python list output: %w", err)
	}

	versions := make([]UvPythonVersion, 0, len(entries))
	for _, entry := range entries {
		versions = append(versions, UvPythonVersion{
			Version:        entry.Version,
			Path:           entry.Path,
			Implementation: entry.Implementation,
		})
	}

	return versions, nil
}

// InstallUvPythonVersion downloads and installs a Python version via uv.
func (u *UvService) InstallUvPythonVersion(version string) error {
	uvPath, err := u.resolveUvPath()
	if err != nil {
		return err
	}

	id := "uv-python-install-" + version
	u.progressNotifier.EmitStart(ProgressTypeInstall, id, fmt.Sprintf("Installing Python %s via uv...", version))

	cmd := exec.Command(uvPath, "python", "install", version)
	hideConsoleWindow(cmd)

	if err := streamCommandProgress(cmd, u.progressNotifier, id, "UV"); err != nil {
		u.progressNotifier.EmitError(ProgressTypeInstall, id, "Failed to install Python version", err.Error())
		return err
	}

	u.progressNotifier.EmitComplete(ProgressTypeInstall, id, fmt.Sprintf("Python %s installed successfully", version))
	return nil
}

// CreateUvVirtualEnv creates a virtual environment with uv, optionally installing plugin requirements.
func (u *UvService) CreateUvVirtualEnv(pythonVersion, venvPath, pluginID, pluginFolderPath string) error {
	uvPath, err := u.resolveUvPath()
	if err != nil {
		return err
	}

	venvName := filepath.Base(venvPath)
	u.progressNotifier.EmitStart(ProgressTypeInstall, "uv-venv", fmt.Sprintf("Creating virtual environment '%s' with uv...", venvName))

	args := []string{"venv", venvPath}
	if pythonVersion != "" {
		args = append(args, "--python", pythonVersion)
	}

	cmd := exec.Command(uvPath, args...)
	hideConsoleWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		u.progressNotifier.EmitError(ProgressTypeInstall, "uv-venv", "Failed to create virtual environment", err.Error())
		return fmt.Errorf("failed to create uv virtual environment: %w (output: %s)", err, string(output))
	}

	pythonExe := filepath.Join(venvPath, "Scripts", "python.exe")
	if goruntime.GOOS != "windows" {
		pythonExe = filepath.Join(venvPath, "bin", "python")
	}

	if pluginID != "" && pluginFolderPath != "" {
		u.progressNotifier.EmitProgress(ProgressTypeInstall, "uv-venv", "Installing plugin requirements...", 40)

		pluginYamlPath := filepath.Join(pluginFolderPath, "plugin.yaml")
		if pluginDef, err := loadPluginDefinition(pluginYamlPath); err == nil {
			inlinePackages := pluginDef.Execution.Requirements.Packages
			requirementsFile := pluginDef.Execution.Requirements.PythonRequirementsFile

			if len(inlinePackages) > 0 {
				if err := u.InstallUvPackages(pythonExe, inlinePackages); err != nil {
					log.Printf("[CreateUvVirtualEnv] Warning: Failed to install inline packages: %v\n", err)
				}
			} else if requirementsFile != "" {
				requirementsPath := filepath.Join(pluginFolderPath, requirementsFile)
				if _, err := os.Stat(requirementsPath); err == nil {
					if err := u.InstallUvRequirements(pythonExe, requirementsPath); err != nil {
						log.Printf("[CreateUvVirtualEnv] Warning: Failed to install plugin requirements: %v\n", err)
					}
				}
			}
		}
	}

	u.progressNotifier.EmitProgress(ProgressTypeInstall, "uv-venv", "Saving environment configuration...", 90)

	var existingVenv VirtualEnvironment
	if err := u.db.GetDB().Where("path = ?", pythonExe).First(&existingVenv).Error; err == nil {
		u.db.GetDB().Delete(&existingVenv)
	}

	venv := VirtualEnvironment{
		Name:           venvName,
		Path:           pythonExe,
		BasePythonPath: pythonVersion,
		CreatedAt:      time.Now().Unix(),
		Source:         "uv",
	}
	if err := u.db.GetDB().Create(&venv).Error; err != nil {
		u.progressNotifier.EmitError(ProgressTypeInstall, "uv-venv", "Failed to save virtual environment to database", err.Error())
		return fmt.Errorf("failed to save virtual environment to database: %w", err)
	}

	u.progressNotifier.EmitComplete(ProgressTypeInstall, "uv-venv", fmt.Sprintf("Virtual environment '%s' created successfully", venvName))
	return nil
}

// InstallUvPackages installs packages into a uv-managed Python environment.
func (u *UvService) InstallUvPackages(venvPythonPath string, packages []string) error {
	if len(packages) == 0 {
		return nil
	}

	uvPath, err := u.resolveUvPath()
	if err != nil {
		return err
	}

	id := "uv-packages"
	u.progressNotifier.EmitStart(ProgressTypeInstall, id, "Installing packages with uv...")

	args := append([]string{"pip", "install", "--python", venvPythonPath}, packages...)
	cmd := exec.Command(uvPath, args...)
	hideConsoleWindow(cmd)

	if err := streamCommandProgress(cmd, u.progressNotifier, id, "UV"); err != nil {
		u.progressNotifier.EmitError(ProgressTypeInstall, id, "Failed to install packages", err.Error())
		return err
	}

	u.progressNotifier.EmitComplete(ProgressTypeInstall, id, "Packages installed successfully")
	return nil
}

// InstallUvRequirements installs a requirements file into a uv-managed Python environment.
func (u *UvService) InstallUvRequirements(venvPythonPath, requirementsPath string) error {
	uvPath, err := u.resolveUvPath()
	if err != nil {
		return err
	}

	id := "uv-requirements"
	u.progressNotifier.EmitStart(ProgressTypeInstall, id, "Installing requirements with uv...")

	cmd := exec.Command(uvPath, "pip", "install", "--python", venvPythonPath, "-r", requirementsPath)
	hideConsoleWindow(cmd)

	if err := streamCommandProgress(cmd, u.progressNotifier, id, "UV"); err != nil {
		u.progressNotifier.EmitError(ProgressTypeInstall, id, "Failed to install requirements", err.Error())
		return err
	}

	u.progressNotifier.EmitComplete(ProgressTypeInstall, id, "Requirements installed successfully")
	return nil
}

// ListUvPackages lists installed packages in a uv-managed Python environment.
func (u *UvService) ListUvPackages(venvPythonPath string) ([]string, error) {
	uvPath, err := u.resolveUvPath()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(uvPath, "pip", "list", "--python", venvPythonPath, "--format=freeze")
	hideConsoleWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list uv-managed packages: %w", err)
	}

	var packages []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			packages = append(packages, line)
		}
	}

	return packages, nil
}

// streamCommandProgress runs cmd, streaming its combined output through the progress notifier.
func streamCommandProgress(cmd *exec.Cmd, notifier *ProgressNotifier, id, logPrefix string) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("[%s] %s\n", logPrefix, line)
		notifier.EmitProgress(ProgressTypeInstall, id, line, 50)
	}

	return cmd.Wait()
}

// fetchGitHubRelease fetches and parses a GitHub release (or releases-list) JSON response; when the URL returns an array, the first entry is used.
func fetchGitHubRelease(url string) (*GitHubRelease, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err == nil && release.TagName != "" {
		return &release, nil
	}

	var releases []GitHubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, err
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}

	return &releases[0], nil
}

// downloadFileWithProgress downloads url to destPath, emitting periodic progress notifications.
func downloadFileWithProgress(url, destPath string, notifier *ProgressNotifier, id string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	totalSize := resp.ContentLength
	downloadedSize := int64(0)
	lastEmitSize := int64(0)
	emitThreshold := int64(100 * 1024)

	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloadedSize += int64(n)

			if downloadedSize-lastEmitSize >= emitThreshold {
				var percentage float64
				if totalSize > 0 {
					percentage = float64(downloadedSize) / float64(totalSize) * 100
				}
				notifier.EmitWithData(ProgressTypeDownload, id, fmt.Sprintf("Downloading %s", id), percentage, map[string]interface{}{
					"downloaded": downloadedSize,
					"total":      totalSize,
				})
				lastEmitSize = downloadedSize
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	return nil
}
