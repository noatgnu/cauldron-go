package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/noatgnu/cauldron-go/pkg/registry"
)

type PluginRegistryService struct {
	ctx           context.Context
	configService *SettingsService
	gitAuth       *GitAuthService
	client        *registry.Client
}

func NewPluginRegistryService(ctx context.Context, configService *SettingsService, gitAuth *GitAuthService) *PluginRegistryService {
	return &PluginRegistryService{
		ctx:           ctx,
		configService: configService,
		gitAuth:       gitAuth,
	}
}

func (s *PluginRegistryService) getClient() (*registry.Client, error) {
	if s.client != nil {
		return s.client, nil
	}

	config := s.configService.GetConfig()
	if config.PluginRegistryURL == "" {
		return nil, fmt.Errorf("plugin registry URL is not configured")
	}

	s.client = registry.NewClient(config.PluginRegistryURL)
	return s.client, nil
}

func (s *PluginRegistryService) ListPlugins(searchQuery string, categoryName string, authorName string, limit int, offset int) (*registry.PluginListResponse, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}

	params := make(map[string]string)

	if searchQuery != "" {
		params["search"] = searchQuery
	}

	if categoryName != "" {
		params["category__name"] = categoryName
	}

	if authorName != "" {
		params["author__name"] = authorName
	}

	if limit > 0 {
		params["limit"] = fmt.Sprintf("%d", limit)
	}

	if offset > 0 {
		params["offset"] = fmt.Sprintf("%d", offset)
	}

	log.Printf("[PluginRegistryService] Fetching plugins with params: %v", params)

	result, err := client.ListPlugins(params)
	if err != nil {
		log.Printf("[PluginRegistryService] Failed to list plugins: %v", err)
		return nil, err
	}

	log.Printf("[PluginRegistryService] Found %d plugins", result.Count)
	return result, nil
}

func (s *PluginRegistryService) GetPlugin(pluginID string) (*registry.Plugin, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}

	log.Printf("[PluginRegistryService] Fetching plugin: %s", pluginID)

	plugin, err := client.GetPlugin(pluginID)
	if err != nil {
		log.Printf("[PluginRegistryService] Failed to get plugin %s: %v", pluginID, err)
		return nil, err
	}

	if plugin.Repository != "" && plugin.CommitHash != "" {
		log.Printf("[PluginRegistryService] Fetching README from repository: %s at %s", plugin.Repository, plugin.CommitHash)
		readme, err := s.FetchReadmeFromGit(plugin.Repository, plugin.CommitHash)
		if err != nil {
			log.Printf("[PluginRegistryService] Warning: Failed to fetch README: %v", err)
		} else {
			plugin.Readme = readme
			log.Printf("[PluginRegistryService] Successfully fetched README (%d bytes)", len(readme))
		}
	}

	return plugin, nil
}

func (s *PluginRegistryService) FetchReadmeFromGit(repoURL string, commitHash string) (string, error) {
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("cauldron-readme-%d", time.Now().UnixNano()))
	defer os.RemoveAll(tempDir)

	auth, err := s.gitAuth.GetAuthMethod(repoURL)
	if err != nil {
		log.Printf("[PluginRegistryService] Warning: failed to get auth method: %v", err)
	}

	cloneOptions := &git.CloneOptions{
		URL:      repoURL,
		Progress: io.Discard,
		Depth:    1,
		Auth:     auth,
	}

	repo, err := git.PlainClone(tempDir, false, cloneOptions)
	if err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	if commitHash != "" {
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
			return "", fmt.Errorf("failed to checkout commit %s: %w", commitHash, err)
		}
	}

	readmePath := filepath.Join(tempDir, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return "", fmt.Errorf("failed to read README.md: %w", err)
	}

	return string(content), nil
}

func (s *PluginRegistryService) ListCategories() (*registry.CategoryListResponse, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}

	log.Printf("[PluginRegistryService] Fetching categories")

	result, err := client.ListCategories()
	if err != nil {
		log.Printf("[PluginRegistryService] Failed to list categories: %v", err)
		return nil, err
	}

	log.Printf("[PluginRegistryService] Found %d categories", result.Count)
	return result, nil
}

func (s *PluginRegistryService) InstallPluginFromRegistry(pluginID string) error {
	plugin, err := s.GetPlugin(pluginID)
	if err != nil {
		return fmt.Errorf("failed to get plugin details: %w", err)
	}

	if plugin.Repository == "" {
		return fmt.Errorf("plugin %s does not have a repository URL", pluginID)
	}

	log.Printf("[PluginRegistryService] Installing plugin %s from repository: %s", plugin.Name, plugin.Repository)

	return nil
}

func (s *PluginRegistryService) CheckUpdate(pluginID string) (*registry.UpdateInfo, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}

	log.Printf("[PluginRegistryService] Checking update for plugin: %s", pluginID)

	updateInfo, err := client.CheckUpdate(pluginID)
	if err != nil {
		log.Printf("[PluginRegistryService] Failed to check update for %s: %v", pluginID, err)
		return nil, err
	}

	log.Printf("[PluginRegistryService] Update info for %s: has_update=%v", pluginID, updateInfo.HasUpdate)
	return updateInfo, nil
}
