package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/pkg/reciperegistry"
)

func createTestSettingsServiceForRecipes(registryURL string) *SettingsService {
	return &SettingsService{
		config: &models.Config{
			RecipeRegistryURL: registryURL,
		},
	}
}

func TestNewRecipeRegistryServiceV3(t *testing.T) {
	settingsService := createTestSettingsServiceForRecipes("https://test.com")
	service := NewRecipeRegistryServiceV3(settingsService)

	if service.configService == nil {
		t.Error("expected configService to be set")
	}
}

func TestRecipeRegistry_GetClient_Success(t *testing.T) {
	testURL := "https://registry.example.com"
	service := NewRecipeRegistryServiceV3(createTestSettingsServiceForRecipes(testURL))

	client, err := service.getClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}
	if client.BaseURL != testURL {
		t.Errorf("expected BaseURL %s, got %s", testURL, client.BaseURL)
	}
}

func TestRecipeRegistry_GetClient_EmptyURL(t *testing.T) {
	service := NewRecipeRegistryServiceV3(createTestSettingsServiceForRecipes(""))

	_, err := service.getClient()
	if err == nil {
		t.Fatal("expected error when registry URL is empty")
	}
	expected := "recipe registry URL is not configured"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestRecipeRegistry_GetClient_CachesClient(t *testing.T) {
	service := NewRecipeRegistryServiceV3(createTestSettingsServiceForRecipes("https://test.com"))

	client1, err1 := service.getClient()
	if err1 != nil {
		t.Fatalf("unexpected error: %v", err1)
	}
	client2, err2 := service.getClient()
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	if client1 != client2 {
		t.Error("expected getClient to return a cached client instance")
	}
}

func TestRecipeRegistry_ListRecipes_Success(t *testing.T) {
	mockResponse := reciperegistry.RecipeListResponse{
		Count: 2,
		Results: []reciperegistry.Recipe{
			{ID: "recipe-1", Label: "Recipe 1"},
			{ID: "recipe-2", Label: "Recipe 2"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("search"); got != "test" {
			t.Errorf("expected search 'test', got '%s'", got)
		}
		if got := r.URL.Query().Get("category__name"); got != "preprocessing" {
			t.Errorf("expected category__name 'preprocessing', got '%s'", got)
		}
		if got := r.URL.Query().Get("tag"); got != "proteomics" {
			t.Errorf("expected tag 'proteomics', got '%s'", got)
		}
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	service := NewRecipeRegistryServiceV3(createTestSettingsServiceForRecipes(server.URL))
	result, err := service.ListRecipes("test", "preprocessing", "", "proteomics", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("expected count 2, got %d", result.Count)
	}
}

func TestRecipeRegistry_ListRecipes_NoRegistryURL(t *testing.T) {
	service := NewRecipeRegistryServiceV3(createTestSettingsServiceForRecipes(""))
	if _, err := service.ListRecipes("", "", "", "", 0, 0); err == nil {
		t.Fatal("expected error when registry URL is not configured")
	}
}

func TestRecipeRegistry_GetRecipe_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/recipes/recipe-1/" {
			t.Errorf("expected path /api/recipes/recipe-1/, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(reciperegistry.Recipe{ID: "recipe-1", Label: "Recipe 1"})
	}))
	defer server.Close()

	service := NewRecipeRegistryServiceV3(createTestSettingsServiceForRecipes(server.URL))
	recipe, err := service.GetRecipe("recipe-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recipe.Label != "Recipe 1" {
		t.Errorf("expected label 'Recipe 1', got '%s'", recipe.Label)
	}
}

func TestRecipeRegistry_GetRecipe_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	service := NewRecipeRegistryServiceV3(createTestSettingsServiceForRecipes(server.URL))
	if _, err := service.GetRecipe("does-not-exist"); err == nil {
		t.Fatal("expected error for a 404 response")
	}
}

func TestRecipeRegistry_ListFilterOptions_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(reciperegistry.FilterOptions{Categories: []string{"preprocessing"}})
	}))
	defer server.Close()

	service := NewRecipeRegistryServiceV3(createTestSettingsServiceForRecipes(server.URL))
	result, err := service.ListFilterOptions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Categories) != 1 {
		t.Errorf("expected 1 category, got %d", len(result.Categories))
	}
}
