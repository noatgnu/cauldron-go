package services

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	gitAuth      *GitAuthService
	wailsApp     interface{}
}

func NewPluginInstaller(pluginsDir string, db *DatabaseService, loader *PluginLoaderV2, gitAuth *GitAuthService) *PluginInstaller {
	return &PluginInstaller{
		pluginsDir:   pluginsDir,
		db:           db,
		pluginLoader: loader,
		gitAuth:      gitAuth,
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

	auth, err := pi.gitAuth.GetAuthMethod(repoURL)
	if err != nil {
		log.Printf("[PluginInstaller] Warning: failed to get auth method: %v", err)
	}

	_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: io.Discard,
		Depth:    1,
		Auth:     auth,
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

func (pi *PluginInstaller) InstallPlugin(repoURL string, commitHash string, registrySource *string, progressCallback func(string)) (string, error) {
	log.Printf("[PluginInstaller] Installing plugin from: %s (ref: %s)", repoURL, commitHash)
	if registrySource != nil {
		log.Printf("[PluginInstaller] Registry source: %s", *registrySource)
	}
	if progressCallback != nil {
		progressCallback("Checking existing installation...")
	}

	var existing models.PluginRegistry
	if err := pi.db.GetDB().Where("repository = ?", repoURL).First(&existing).Error; err == nil {
		if existing.FolderPath != "" {
			if _, statErr := os.Stat(existing.FolderPath); statErr == nil {
				return "", fmt.Errorf("plugin from this repository already installed [ID:%d]", existing.ID)
			}
		}
		log.Printf("[PluginInstaller] Stale registry entry found for %s (ID:%d, missing folder), removing", repoURL, existing.ID)
		pi.db.GetDB().Delete(&existing)
	}

	tempDir := filepath.Join(pi.pluginsDir, ".temp-"+fmt.Sprintf("%d", time.Now().Unix()))
	defer os.RemoveAll(tempDir)

	if progressCallback != nil {
		progressCallback("Cloning repository...")
	}

	auth, err := pi.gitAuth.GetAuthMethod(repoURL)
	if err != nil {
		log.Printf("[PluginInstaller] Warning: failed to get auth method: %v", err)
	}

	cloneOptions := &git.CloneOptions{
		URL:      repoURL,
		Progress: io.Discard,
		Auth:     auth,
	}

	repo, err := git.PlainClone(tempDir, false, cloneOptions)
	if err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	if commitHash != "" {
		if progressCallback != nil {
			progressCallback(fmt.Sprintf("Checking out reference: %s...", commitHash))
		}
		w, err := repo.Worktree()
		if err != nil {
			return "", fmt.Errorf("failed to get worktree: %w", err)
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
				return "", fmt.Errorf("failed to checkout ref '%s': %w", commitHash, err)
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
		return "", fmt.Errorf("failed to read plugin definition: %w", err)
	}

	if progressCallback != nil {
		progressCallback("Registering plugin...")
	}

	registry := models.PluginRegistry{
		PluginID:       pluginDef.Plugin.ID,
		Name:           pluginDef.Plugin.Name,
		Version:        pluginDef.Plugin.Version,
		Repository:     repoURL,
		CommitHash:     actualHash,
		InstallSource:  "remote",
		RegistrySource: registrySource,
		Enabled:        true,
		InstalledAt:    time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := pi.db.GetDB().Create(&registry).Error; err != nil {
		return "", fmt.Errorf("failed to create registry entry: %w", err)
	}

	finalDir := filepath.Join(pi.pluginsDir, fmt.Sprintf("%s-%d", pluginDef.Plugin.ID, registry.ID))
	registry.FolderPath = finalDir

	if progressCallback != nil {
		progressCallback("Finalizing installation...")
	}

	if err := os.Rename(tempDir, finalDir); err != nil {
		pi.db.GetDB().Delete(&registry)
		return "", fmt.Errorf("failed to move plugin to final location: %w", err)
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

	return pluginDef.Plugin.ID, nil
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
	return pi.UpdatePluginWithForce(repoURL, false)
}

func (pi *PluginInstaller) UpdatePluginWithForce(repoURL string, force bool) error {
	log.Printf("[PluginInstaller] Updating plugin from: %s (force: %v)", repoURL, force)

	var registry models.PluginRegistry
	if err := pi.db.GetDB().Where("repository = ?", repoURL).First(&registry).Error; err != nil {
		return fmt.Errorf("plugin not installed: %w", err)
	}

	repo, err := git.PlainOpen(registry.FolderPath)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "repository does not exist") {
			log.Printf("[PluginInstaller] Plugin folder missing, reinstalling from: %s", repoURL)
			return pi.ReinstallPlugin(repoURL)
		}
		return fmt.Errorf("failed to open repository: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("failed to check repository status: %w", err)
	}

	if !status.IsClean() && !force {
		return fmt.Errorf("LOCAL_MODIFICATIONS: plugin has local modifications - updating will discard these changes")
	}

	if !status.IsClean() {
		log.Printf("[PluginInstaller] Discarding local modifications (force update)")
		err = w.Reset(&git.ResetOptions{Mode: git.HardReset})
		if err != nil {
			return fmt.Errorf("failed to discard local changes: %w", err)
		}
	}

	auth, err := pi.gitAuth.GetAuthMethod(repoURL)
	if err != nil {
		log.Printf("[PluginInstaller] Warning: failed to get auth method: %v", err)
	}

	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Progress:   io.Discard,
		Auth:       auth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull updates: %w", err)
	}

	if err == git.NoErrAlreadyUpToDate {
		log.Printf("[PluginInstaller] Plugin already up to date")
	} else {
		ref, err := repo.Head()
		if err == nil {
			registry.CommitHash = ref.Hash().String()
		}
		registry.UpdatedAt = time.Now()
		pi.db.GetDB().Save(&registry)
		log.Printf("[PluginInstaller] Successfully updated plugin")

		if err := pi.pluginLoader.ReloadPlugins(); err != nil {
			log.Printf("[PluginInstaller] Warning: failed to reload plugins: %v", err)
		}
	}

	return nil
}

func (pi *PluginInstaller) UpdatePluginToCommit(repoURL string, commitHash string) error {
	return pi.UpdatePluginToCommitWithForce(repoURL, commitHash, false)
}

func (pi *PluginInstaller) UpdatePluginToCommitWithForce(repoURL string, commitHash string, force bool) error {
	log.Printf("[PluginInstaller] Updating plugin from %s to commit: %s (force: %v)", repoURL, commitHash, force)

	var registry models.PluginRegistry
	if err := pi.db.GetDB().Where("repository = ?", repoURL).First(&registry).Error; err != nil {
		return fmt.Errorf("plugin not installed: %w", err)
	}

	if registry.CommitHash == commitHash {
		log.Printf("[PluginInstaller] Plugin already at target commit: %s", commitHash)
		return nil
	}

	repo, err := git.PlainOpen(registry.FolderPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("failed to check repository status: %w", err)
	}

	if !status.IsClean() && !force {
		return fmt.Errorf("LOCAL_MODIFICATIONS: plugin has local modifications - updating will discard these changes")
	}

	if !status.IsClean() {
		log.Printf("[PluginInstaller] Discarding local modifications (force update)")
		err = w.Reset(&git.ResetOptions{Mode: git.HardReset})
		if err != nil {
			return fmt.Errorf("failed to discard local changes: %w", err)
		}
	}

	auth, err := pi.gitAuth.GetAuthMethod(repoURL)
	if err != nil {
		log.Printf("[PluginInstaller] Warning: failed to get auth method: %v", err)
	}

	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Progress:   io.Discard,
		Auth:       auth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		log.Printf("[PluginInstaller] Warning: fetch failed: %v", err)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash:  plumbing.NewHash(commitHash),
		Force: false,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout commit %s: %w", commitHash, err)
	}

	registry.CommitHash = commitHash
	registry.UpdatedAt = time.Now()
	if err := pi.db.GetDB().Save(&registry).Error; err != nil {
		return fmt.Errorf("failed to update registry: %w", err)
	}

	log.Printf("[PluginInstaller] Successfully updated plugin to commit: %s", commitHash)

	if err := pi.pluginLoader.ReloadPlugins(); err != nil {
		log.Printf("[PluginInstaller] Warning: failed to reload plugins: %v", err)
	}

	return nil
}

func (pi *PluginInstaller) ReinstallPlugin(repoURL string) error {
	log.Printf("[PluginInstaller] Reinstalling plugin from: %s", repoURL)

	var registry models.PluginRegistry
	if err := pi.db.GetDB().Where("repository = ?", repoURL).First(&registry).Error; err != nil {
		return fmt.Errorf("plugin not installed: %w", err)
	}

	commitHash := registry.CommitHash
	folderPath := registry.FolderPath

	if _, err := os.Stat(folderPath); err == nil {
		log.Printf("[PluginInstaller] Removing plugin folder: %s", folderPath)
		if err := os.RemoveAll(folderPath); err != nil {
			return fmt.Errorf("failed to remove plugin folder: %w", err)
		}
	} else {
		log.Printf("[PluginInstaller] Plugin folder does not exist, skipping deletion: %s", folderPath)
	}

	tempDir := filepath.Join(pi.pluginsDir, ".temp-reinstall-"+fmt.Sprintf("%d", time.Now().Unix()))
	defer os.RemoveAll(tempDir)

	log.Printf("[PluginInstaller] Cloning fresh copy from repository...")

	auth, err := pi.gitAuth.GetAuthMethod(repoURL)
	if err != nil {
		log.Printf("[PluginInstaller] Warning: failed to get auth method: %v", err)
	}

	repo, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: io.Discard,
		Auth:     auth,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	if commitHash != "" {
		log.Printf("[PluginInstaller] Checking out commit: %s", commitHash)
		w, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}

		err = w.Checkout(&git.CheckoutOptions{
			Hash:  plumbing.NewHash(commitHash),
			Force: true,
		})
		if err != nil {
			return fmt.Errorf("failed to checkout commit %s: %w", commitHash, err)
		}
	}

	parentDir := filepath.Dir(folderPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	if err := os.Rename(tempDir, folderPath); err != nil {
		return fmt.Errorf("failed to move plugin to final location: %w", err)
	}

	registry.UpdatedAt = time.Now()
	if err := pi.db.GetDB().Save(&registry).Error; err != nil {
		log.Printf("[PluginInstaller] Warning: failed to update registry: %v", err)
	}

	log.Printf("[PluginInstaller] Successfully reinstalled plugin [ID:%d]", registry.ID)

	if err := pi.pluginLoader.ReloadPlugins(); err != nil {
		log.Printf("[PluginInstaller] Warning: failed to reload plugins: %v", err)
	}

	return nil
}

type UninstallOptions struct {
	RemoveGitAuth      bool
	DeleteJobHistory   bool
	DeleteEnvironments bool
}

func (pi *PluginInstaller) UninstallPlugin(repoURL string, options UninstallOptions) error {
	log.Printf("[PluginInstaller] Uninstalling plugin from: %s", repoURL)

	var registry models.PluginRegistry
	if err := pi.db.GetDB().Where("repository = ?", repoURL).First(&registry).Error; err != nil {
		return fmt.Errorf("plugin not installed: %w", err)
	}

	pluginID := registry.PluginID

	log.Printf("[PluginInstaller] Cleaning up plugin data for: %s", pluginID)

	if err := pi.db.GetDB().Where("plugin_id = ?", pluginID).Delete(&PluginEnvironmentBinding{}).Error; err != nil {
		log.Printf("[PluginInstaller] Warning: failed to delete environment bindings: %v", err)
	}

	if err := pi.db.GetDB().Where("plugin_id = ?", registry.ID).Delete(&CustomEnvVar{}).Error; err != nil {
		log.Printf("[PluginInstaller] Warning: failed to delete custom env vars: %v", err)
	} else {
		log.Printf("[PluginInstaller] Deleted custom env vars for plugin registry ID: %d", registry.ID)
	}

	if options.RemoveGitAuth {
		if err := pi.gitAuth.DeleteGitAuthConfig(repoURL); err != nil {
			log.Printf("[PluginInstaller] Warning: failed to delete Git auth config: %v", err)
		} else {
			log.Printf("[PluginInstaller] Deleted Git authentication config for: %s", repoURL)
		}
	}

	if options.DeleteJobHistory {
		if err := pi.deletePluginJobs(pluginID); err != nil {
			log.Printf("[PluginInstaller] Warning: failed to delete job history: %v", err)
		} else {
			log.Printf("[PluginInstaller] Deleted job history for plugin: %s", pluginID)
		}
	}

	if options.DeleteEnvironments {
		if err := pi.deletePluginEnvironments(pluginID); err != nil {
			log.Printf("[PluginInstaller] Warning: failed to delete environments: %v", err)
		} else {
			log.Printf("[PluginInstaller] Deleted bound environments for plugin: %s", pluginID)
		}
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

func (pi *PluginInstaller) deletePluginJobs(pluginID string) error {
	var jobs []models.Job
	if err := pi.db.GetDB().Where("type = ?", pluginID).Find(&jobs).Error; err != nil {
		return fmt.Errorf("failed to query jobs: %w", err)
	}

	for _, job := range jobs {
		if job.OutputPath != "" {
			if err := os.RemoveAll(job.OutputPath); err != nil {
				log.Printf("[PluginInstaller] Warning: failed to delete job output directory %s: %v", job.OutputPath, err)
			}
		}
	}

	if err := pi.db.GetDB().Where("type = ?", pluginID).Delete(&models.Job{}).Error; err != nil {
		return fmt.Errorf("failed to delete jobs: %w", err)
	}

	return nil
}

func (pi *PluginInstaller) GetPluginJobCount(pluginID string) (int64, error) {
	var count int64
	if err := pi.db.GetDB().Model(&models.Job{}).Where("type = ?", pluginID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (pi *PluginInstaller) deletePluginEnvironments(pluginID string) error {
	var bindings []PluginEnvironmentBinding
	if err := pi.db.GetDB().Where("plugin_id = ?", pluginID).Find(&bindings).Error; err != nil {
		return fmt.Errorf("failed to query plugin bindings: %w", err)
	}

	for _, binding := range bindings {
		switch binding.EnvironmentType {
		case "python":
			var venv VirtualEnvironment
			if err := pi.db.GetDB().Where("id = ?", binding.EnvironmentID).First(&venv).Error; err == nil {
				if err := os.RemoveAll(venv.Path); err != nil {
					log.Printf("[PluginInstaller] Warning: failed to delete Python venv directory %s: %v", venv.Path, err)
				}
				if err := pi.db.GetDB().Delete(&venv).Error; err != nil {
					log.Printf("[PluginInstaller] Warning: failed to delete Python venv DB entry: %v", err)
				} else {
					log.Printf("[PluginInstaller] Deleted Python venv: %s", venv.Name)
				}
			}
		case "r":
			var renv RenvEnvironment
			if err := pi.db.GetDB().Where("id = ?", binding.EnvironmentID).First(&renv).Error; err == nil {
				if err := os.RemoveAll(renv.Path); err != nil {
					log.Printf("[PluginInstaller] Warning: failed to delete R renv directory %s: %v", renv.Path, err)
				}
				if err := pi.db.GetDB().Delete(&renv).Error; err != nil {
					log.Printf("[PluginInstaller] Warning: failed to delete R renv DB entry: %v", err)
				} else {
					log.Printf("[PluginInstaller] Deleted R renv: %s", renv.Name)
				}
			}
		}
	}

	return nil
}

func (pi *PluginInstaller) GetPluginEnvironmentCount(pluginID string) (int64, error) {
	var count int64
	if err := pi.db.GetDB().Model(&PluginEnvironmentBinding{}).Where("plugin_id = ?", pluginID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
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

type RepositoryUpdateInfo struct {
	PluginID          string  `json:"plugin_id"`
	CurrentCommit     string  `json:"current_commit"`
	LatestCommit      string  `json:"latest_commit"`
	RecommendedCommit string  `json:"recommended_commit"`
	HasUpdate         bool    `json:"has_update"`
	ChangelogURL      *string `json:"changelog_url"`
}

func (pi *PluginInstaller) CheckRepositoryUpdate(repoURL string, currentCommit string) (*RepositoryUpdateInfo, error) {
	log.Printf("[PluginInstaller] Checking repository update: %s (current: %s)", repoURL, currentCommit)

	tempDir := filepath.Join(pi.pluginsDir, ".temp-update-check-"+fmt.Sprintf("%d", time.Now().Unix()))
	defer os.RemoveAll(tempDir)

	auth, err := pi.gitAuth.GetAuthMethod(repoURL)
	if err != nil {
		log.Printf("[PluginInstaller] Warning: failed to get auth method: %v", err)
	}

	repo, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: io.Discard,
		Depth:    1,
		Auth:     auth,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	latestCommit := ref.Hash().String()
	hasUpdate := latestCommit != currentCommit

	var changelogURL *string
	if hasUpdate {
		changelog := fmt.Sprintf("%s/compare/%s...%s", repoURL, currentCommit, latestCommit)
		changelogURL = &changelog
	}

	var registryEntry models.PluginRegistry
	var pluginID string
	if err := pi.db.GetDB().Where("repository = ?", repoURL).First(&registryEntry).Error; err == nil {
		pluginID = registryEntry.PluginID
	} else {
		pluginID = repoURL
	}

	updateInfo := &RepositoryUpdateInfo{
		PluginID:          pluginID,
		CurrentCommit:     currentCommit,
		LatestCommit:      latestCommit,
		RecommendedCommit: latestCommit,
		HasUpdate:         hasUpdate,
		ChangelogURL:      changelogURL,
	}

	log.Printf("[PluginInstaller] Update check result: has_update=%v", hasUpdate)
	return updateInfo, nil
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
