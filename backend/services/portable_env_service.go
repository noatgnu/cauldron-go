package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
)

type PortableEnvService struct {
	ctx              context.Context
	fileService      *FileService
	progressNotifier *ProgressNotifier
}

type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	URL                string `json:"url"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func NewPortableEnvService(ctx context.Context, fileService *FileService) *PortableEnvService {
	return &PortableEnvService{
		ctx:              ctx,
		fileService:      fileService,
		progressNotifier: NewProgressNotifier(ctx),
	}
}

func (p *PortableEnvService) GetPortableEnvironmentURL(platform, arch, version, environment string) (string, error) {
	url := "https://api.github.com/repos/noatgnu/cauldron-go/releases"

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var releases []GitHubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", err
	}

	// If version is empty or "latest", search through all releases for first match
	// Otherwise search for specific version
	for _, release := range releases {
		if version != "" && version != "latest" && release.TagName != version {
			continue
		}

		for _, asset := range release.Assets {
			// Special handling for Windows - check it contains platform and environment but NOT darwin
			if platform == "win" {
				if strings.Contains(asset.Name, platform) &&
					strings.Contains(asset.Name, environment) &&
					!strings.Contains(asset.Name, "darwin") {
					// Use browser_download_url for direct download
					if asset.BrowserDownloadURL != "" {
						return asset.BrowserDownloadURL, nil
					}
					return asset.URL, nil
				}
			}

			// For other platforms, check platform, arch, and environment are all in the name
			if strings.Contains(asset.Name, platform) &&
				strings.Contains(asset.Name, arch) &&
				strings.Contains(asset.Name, environment) {
				// Use browser_download_url for direct download
				if asset.BrowserDownloadURL != "" {
					return asset.BrowserDownloadURL, nil
				}
				return asset.URL, nil
			}
		}

		// If we found a matching release but no matching asset, continue to next release
		// If version was specified and we checked it, break
		if version != "" && version != "latest" {
			break
		}
	}

	return "", fmt.Errorf("no matching release asset found for %s/%s/%s in environment %s", platform, arch, version, environment)
}

func (p *PortableEnvService) DownloadPortableEnvironment(url, environment string) error {
	log.Printf("[DownloadPortableEnvironment] Starting download for %s from %s", environment, url)

	appFolder, err := getAppDataFolder()
	if err != nil {
		return err
	}

	tempFolder := filepath.Join(appFolder, "temp")
	binFolder := filepath.Join(appFolder, "bin")

	if err := os.MkdirAll(tempFolder, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(binFolder, 0755); err != nil {
		return err
	}

	var fileName string
	if environment == "python" {
		fileName = "python.tar.gz"
	} else {
		fileName = "r-portable.tar.gz"
	}

	tempFilePath := filepath.Join(tempFolder, fileName)
	log.Printf("[DownloadPortableEnvironment] Downloading to: %s", tempFilePath)

	p.progressNotifier.EmitStart(ProgressTypeDownload, fileName, fmt.Sprintf("Downloading %s", fileName))

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("[DownloadPortableEnvironment] Failed to create request: %v", err)
		return err
	}
	req.Header.Set("Accept", "application/octet-stream")

	log.Printf("[DownloadPortableEnvironment] Sending HTTP request...")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[DownloadPortableEnvironment] HTTP request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	log.Printf("[DownloadPortableEnvironment] Response status: %d, ContentLength: %d", resp.StatusCode, resp.ContentLength)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(tempFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	totalSize := resp.ContentLength
	downloadedSize := int64(0)

	buf := make([]byte, 32*1024)
	lastEmitSize := int64(0)
	emitThreshold := int64(100 * 1024)
	progressCounter := 0

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			downloadedSize += int64(n)

			shouldEmit := (downloadedSize - lastEmitSize) >= emitThreshold
			if shouldEmit {
				progressCounter++
				var percentage float64
				if totalSize > 0 {
					percentage = float64(downloadedSize) / float64(totalSize) * 100
				} else {
					percentage = 0
				}

				log.Printf("[Download Progress #%d] Downloaded: %d / %d (%.1f%%)", progressCounter, downloadedSize, totalSize, percentage)

				p.progressNotifier.EmitWithData(ProgressTypeDownload, fileName,
					fmt.Sprintf("Downloading %s", fileName), percentage, map[string]interface{}{
						"downloaded": downloadedSize,
						"total":      totalSize,
					})
				lastEmitSize = downloadedSize
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	// Final progress update
	log.Printf("[DownloadPortableEnvironment] Download complete: %d bytes", downloadedSize)
	p.progressNotifier.EmitComplete(ProgressTypeDownload, fileName, fmt.Sprintf("Downloaded %s", fileName))

	log.Printf("[DownloadPortableEnvironment] Starting extraction...")

	platformName := translatePlatform(goruntime.GOOS)

	if err := p.fileService.ExtractTarGz(tempFilePath, tempFolder); err != nil {
		return err
	}

	p.progressNotifier.EmitStart(ProgressTypeInstall, "move-"+fileName, "Moving new environment")

	var srcPath, destPath string
	if environment == "python" {
		srcPath = filepath.Join(tempFolder, "resources", "bin", platformName, "python")
		destPath = filepath.Join(binFolder, platformName, "python")
	} else {
		srcPath = filepath.Join(tempFolder, "resources", "bin", platformName, "R-Portable")
		destPath = filepath.Join(binFolder, platformName, "R-Portable")
	}

	if _, err := os.Stat(destPath); err == nil {
		if err := os.RemoveAll(destPath); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	if err := p.copyDirWithProgress(srcPath, destPath, "move-"+fileName); err != nil {
		return err
	}

	log.Printf("[DownloadPortableEnvironment] Cleaning up temporary files...")
	os.RemoveAll(filepath.Join(tempFolder, "resources"))
	os.Remove(tempFilePath)

	p.progressNotifier.EmitComplete(ProgressTypeInstall, fileName, "Installation completed")

	log.Printf("[DownloadPortableEnvironment] Installation completed successfully")

	return nil
}

func (p *PortableEnvService) GetPortableEnvironmentPath(environment string) (string, error) {
	appFolder, err := getAppDataFolder()
	if err != nil {
		return "", err
	}

	platformName := translatePlatform(goruntime.GOOS)
	binFolder := filepath.Join(appFolder, "bin", platformName)

	var exePath string
	if environment == "python" {
		if goruntime.GOOS == "windows" {
			exePath = filepath.Join(binFolder, "python", "python.exe")
		} else {
			exePath = filepath.Join(binFolder, "python", "bin", "python")
		}
	} else {
		if goruntime.GOOS == "windows" {
			exePath = filepath.Join(binFolder, "R-Portable", "bin", "Rscript.exe")
		} else {
			exePath = filepath.Join(binFolder, "R-Portable", "bin", "Rscript")
		}
	}

	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return "", fmt.Errorf("portable environment not found at %s", exePath)
	}

	return exePath, nil
}

func translatePlatform(goos string) string {
	switch goos {
	case "windows":
		return "win"
	case "darwin":
		return "darwin"
	case "linux":
		return "linux"
	default:
		return goos
	}
}

func getAppDataFolder() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch goruntime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			return filepath.Join(localAppData, "cauldron"), nil
		}
		return filepath.Join(homeDir, "AppData", "Local", "cauldron"), nil

	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "cauldron"), nil

	case "linux":
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome != "" {
			return filepath.Join(xdgDataHome, "cauldron"), nil
		}
		return filepath.Join(homeDir, ".local", "share", "cauldron"), nil

	default:
		return filepath.Join(homeDir, ".cauldron"), nil
	}
}

func (p *PortableEnvService) copyDirWithProgress(src, dst string, id string) error {
	p.progressNotifier.EmitProgress(ProgressTypeInstall, id, "Moving new environment", 0)

	var totalFiles int
	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalFiles++
		}
		return nil
	})

	if totalFiles == 0 {
		p.progressNotifier.EmitComplete(ProgressTypeInstall, id, "Environment moved")
		return nil
	}

	copiedFiles := 0
	lastEmitCount := 0
	emitThreshold := totalFiles / 20
	if emitThreshold < 1 {
		emitThreshold = 1
	}

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		if err := copyFile(path, dstPath); err != nil {
			return err
		}

		copiedFiles++

		if copiedFiles-lastEmitCount >= emitThreshold || copiedFiles == totalFiles {
			percentage := float64(copiedFiles) / float64(totalFiles) * 100
			if percentage > 100 {
				percentage = 100
			}

			p.progressNotifier.EmitProgress(ProgressTypeInstall, id, "Moving new environment", percentage)

			lastEmitCount = copiedFiles
		}

		return nil
	})

	if err == nil {
		p.progressNotifier.EmitComplete(ProgressTypeInstall, id, "Environment moved")
	}

	return err
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
