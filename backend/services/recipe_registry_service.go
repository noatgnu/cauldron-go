package services

import (
	"fmt"
	"log"

	"github.com/noatgnu/cauldron-go/pkg/reciperegistry"
)

type RecipeRegistryService struct {
	configService *SettingsService
	client        *reciperegistry.Client
}

func NewRecipeRegistryServiceV3(configService *SettingsService) *RecipeRegistryService {
	return &RecipeRegistryService{configService: configService}
}

func (s *RecipeRegistryService) getClient() (*reciperegistry.Client, error) {
	if s.client != nil {
		return s.client, nil
	}

	config := s.configService.GetConfig()
	if config.RecipeRegistryURL == "" {
		return nil, fmt.Errorf("recipe registry URL is not configured")
	}

	s.client = reciperegistry.NewClient(config.RecipeRegistryURL)
	return s.client, nil
}

func (s *RecipeRegistryService) ListRecipes(searchQuery string, categoryName string, authorName string, tag string, limit int, offset int) (*reciperegistry.RecipeListResponse, error) {
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
	if tag != "" {
		params["tag"] = tag
	}
	if limit > 0 {
		params["limit"] = fmt.Sprintf("%d", limit)
	}
	if offset > 0 {
		params["offset"] = fmt.Sprintf("%d", offset)
	}

	log.Printf("[RecipeRegistryService] Fetching recipes with params: %v", params)

	result, err := client.ListRecipes(params)
	if err != nil {
		log.Printf("[RecipeRegistryService] Failed to list recipes: %v", err)
		return nil, err
	}

	log.Printf("[RecipeRegistryService] Found %d recipes", result.Count)
	return result, nil
}

func (s *RecipeRegistryService) GetRecipe(recipeID string) (*reciperegistry.Recipe, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}

	log.Printf("[RecipeRegistryService] Fetching recipe: %s", recipeID)

	recipe, err := client.GetRecipe(recipeID)
	if err != nil {
		log.Printf("[RecipeRegistryService] Failed to get recipe %s: %v", recipeID, err)
		return nil, err
	}

	return recipe, nil
}

func (s *RecipeRegistryService) ListFilterOptions() (*reciperegistry.FilterOptions, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}

	log.Printf("[RecipeRegistryService] Fetching filter options")

	result, err := client.ListFilterOptions()
	if err != nil {
		log.Printf("[RecipeRegistryService] Failed to list filter options: %v", err)
		return nil, err
	}

	return result, nil
}
