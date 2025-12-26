package services

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/noatgnu/cauldron-go/backend/models"
	"gopkg.in/yaml.v3"
)

type PluginInstaller struct {
	pluginsDir   string
	db           *DatabaseService
	pluginLoader *PluginLoaderV2
}

func NewPluginInstaller(pluginsDir string, db *DatabaseService, loader *PluginLoaderV2) *PluginInstaller {
	return &PluginInstaller{
		pluginsDir:   pluginsDir,
		db:           db,
		pluginLoader: loader,
	}
}

func (pi *PluginInstaller) IsPluginInstalled(repoURL string) (bool, error) {
	var count int64
	err := pi.db.GetDB().Model(&models.PluginRegistry{}).Where("repository = ?", repoURL).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (pi *PluginInstaller) FetchPluginInfo(repoURL string) (*models.PluginDefinition, error) {
	log.Printf("[PluginInstaller] Fetching plugin info from: %s", repoURL)

	tempDir := filepath.Join(pi.pluginsDir, ".temp-info-"+fmt.Sprintf("%d", time.Now().Unix()))
	defer os.RemoveAll(tempDir)

	_, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: io.Discard,
		Depth:    1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	pluginDef, err := pi.readPluginDefinition(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin definition: %w", err)
	}

	return pluginDef, nil
}

func (pi *PluginInstaller) InstallPlugin(repoURL string, commitHash string, progressCallback func(string)) error {
	log.Printf("[PluginInstaller] Installing plugin from: %s (ref: %s)", repoURL, commitHash)
	if progressCallback != nil {
		progressCallback("Checking existing installation...")
	}

	var existing models.PluginRegistry
	if err := pi.db.GetDB().Where("repository = ?", repoURL).First(&existing).Error; err == nil {
		return fmt.Errorf("plugin from this repository already installed [ID:%d]", existing.ID)
	}

	tempDir := filepath.Join(pi.pluginsDir, ".temp-"+fmt.Sprintf("%d", time.Now().Unix()))
	defer os.RemoveAll(tempDir)

	if progressCallback != nil {
		progressCallback("Cloning repository...")
	}

	cloneOptions := &git.CloneOptions{
		URL:      repoURL,
		Progress: io.Discard,
	}

	repo, err := git.PlainClone(tempDir, false, cloneOptions)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	if commitHash != "" {
		if progressCallback != nil {
			progressCallback(fmt.Sprintf("Checking out reference: %s...", commitHash))
		}
		w, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}

		err = w.Checkout(&git.CheckoutOptions{
			Hash:   plumbing.NewHash(commitHash),
			Create: false,
			Force:  true,
		})
		if err != nil {
			err = w.Checkout(&git.CheckoutOptions{
				Branch: plumbing.NewBranchReferenceName(commitHash),
				Force:  true,
			})
			if err != nil {
				return fmt.Errorf("failed to checkout ref '%s': %w", commitHash, err)
			}
		}
	}

	ref, err := repo.Head()
	actualHash := ""
	if err == nil {
		actualHash = ref.Hash().String()
	}

	if progressCallback != nil {
		progressCallback("Reading plugin configuration...")
	}

	pluginDef, err := pi.readPluginDefinition(tempDir)
	if err != nil {
		return fmt.Errorf("failed to read plugin definition: %w", err)
	}

	if progressCallback != nil {
		progressCallback("Registering plugin...")
	}

	registry := models.PluginRegistry{
		PluginID:      pluginDef.Plugin.ID,
		Name:          pluginDef.Plugin.Name,
		Version:       pluginDef.Plugin.Version,
		Repository:    repoURL,
		CommitHash:    actualHash,
		InstallSource: "remote",
		InstalledAt:   time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := pi.db.GetDB().Create(&registry).Error; err != nil {
		return fmt.Errorf("failed to create registry entry: %w", err)
	}

	finalDir := filepath.Join(pi.pluginsDir, fmt.Sprintf("%s-%d", pluginDef.Plugin.ID, registry.ID))
	registry.FolderPath = finalDir

	if progressCallback != nil {
		progressCallback("Finalizing installation...")
	}

	if err := os.Rename(tempDir, finalDir); err != nil {
		pi.db.GetDB().Delete(&registry)
		return fmt.Errorf("failed to move plugin to final location: %w", err)
	}

	if err := pi.db.GetDB().Save(&registry).Error; err != nil {
		log.Printf("[PluginInstaller] Warning: failed to update folder path: %v", err)
	}

	log.Printf("[PluginInstaller] Successfully installed plugin [ID:%d] to: %s", registry.ID, finalDir)

	if progressCallback != nil {
		progressCallback("Reloading plugins...")
	}

	if err := pi.pluginLoader.ReloadPlugins(); err != nil {
		log.Printf("[PluginInstaller] Warning: failed to reload plugins: %v", err)
	}

	return nil
}

func (pi *PluginInstaller) readPluginDefinition(pluginDir string) (*models.PluginDefinition, error) {
	configPath := filepath.Join(pluginDir, "plugin.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = filepath.Join(pluginDir, "plugin.yml")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var def models.PluginDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, err
	}

	return &def, nil
}

func (pi *PluginInstaller) UpdatePlugin(repoURL string) error {
	log.Printf("[PluginInstaller] Updating plugin from: %s", repoURL)

	var registry models.PluginRegistry
	if err := pi.db.GetDB().Where("repository = ?", repoURL).First(&registry).Error; err != nil {
		return fmt.Errorf("plugin not installed: %w", err)
	}

	repo, err := git.PlainOpen(registry.FolderPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Progress:   io.Discard,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull updates: %w", err)
	}

	if err == git.NoErrAlreadyUpToDate {
		log.Printf("[PluginInstaller] Plugin already up to date")
	} else {
		registry.UpdatedAt = time.Now()
		pi.db.GetDB().Save(&registry)
		log.Printf("[PluginInstaller] Successfully updated plugin")

		if err := pi.pluginLoader.ReloadPlugins(); err != nil {
			log.Printf("[PluginInstaller] Warning: failed to reload plugins: %v", err)
		}
	}

	return nil
}

func (pi *PluginInstaller) UninstallPlugin(repoURL string) error {
	log.Printf("[PluginInstaller] Uninstalling plugin from: %s", repoURL)

	var registry models.PluginRegistry
	if err := pi.db.GetDB().Where("repository = ?", repoURL).First(&registry).Error; err != nil {
		return fmt.Errorf("plugin not installed: %w", err)
	}

	if err := os.RemoveAll(registry.FolderPath); err != nil {
		return fmt.Errorf("failed to remove plugin directory: %w", err)
	}

	if err := pi.db.GetDB().Delete(&registry).Error; err != nil {
		log.Printf("[PluginInstaller] Warning: failed to delete registry entry: %v", err)
	}

	log.Printf("[PluginInstaller] Successfully uninstalled plugin [ID:%d]", registry.ID)

	if err := pi.pluginLoader.ReloadPlugins(); err != nil {
		log.Printf("[PluginInstaller] Warning: failed to reload plugins: %v", err)
	}

	return nil
}

func (pi *PluginInstaller) GetInstalledVersion(repoURL string) (string, error) {
	var registry models.PluginRegistry
	if err := pi.db.GetDB().Where("repository = ?", repoURL).First(&registry).Error; err != nil {
		return "", fmt.Errorf("plugin not installed: %w", err)
	}

	repo, err := git.PlainOpen(registry.FolderPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	return ref.Hash().String()[:7], nil
}

func EncodeRepoURL(repoURL string) string {
	return base64.URLEncoding.EncodeToString([]byte(repoURL))
}

func DecodeRepoURL(encoded string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
