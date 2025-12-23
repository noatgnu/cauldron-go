package services

import (
	"context"
	"fmt"
	"log"

	"github.com/noatgnu/cauldron-go/pkg/registry"
)

type PluginRegistryService struct {
	ctx           context.Context
	configService *SettingsService
	client        *registry.Client
}

func NewPluginRegistryService(ctx context.Context, configService *SettingsService) *PluginRegistryService {
	return &PluginRegistryService{
		ctx:           ctx,
		configService: configService,
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

	return plugin, nil
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
