package reciperegistry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	baseURL := "https://example.com"
	client := NewClient(baseURL)

	if client.BaseURL != baseURL {
		t.Errorf("expected BaseURL %s, got %s", baseURL, client.BaseURL)
	}
	if client.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
	if client.HTTPClient.Timeout == 0 {
		t.Error("HTTPClient should have a timeout set")
	}
}

func TestListRecipes_Success(t *testing.T) {
	mockResponse := RecipeListResponse{
		Count: 2,
		Results: []Recipe{
			{ID: "recipe-1", Label: "Recipe One", Status: "approved"},
			{ID: "recipe-2", Label: "Recipe Two", Status: "approved"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/recipes/" {
			t.Errorf("expected path /api/recipes/, got %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept header application/json, got %s", r.Header.Get("Accept"))
		}
		if got := r.URL.Query().Get("search"); got != "test" {
			t.Errorf("expected search query 'test', got '%s'", got)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("expected limit 10, got %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.ListRecipes(map[string]string{"search": "test", "limit": "10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("expected count 2, got %d", result.Count)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 recipes, got %d", len(result.Results))
	}
	if result.Results[0].Label != "Recipe One" {
		t.Errorf("expected label 'Recipe One', got '%s'", result.Results[0].Label)
	}
}

func TestListRecipes_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	if _, err := client.ListRecipes(nil); err == nil {
		t.Fatal("expected an error for a non-200 response, got nil")
	}
}

func TestListRecipes_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	if _, err := client.ListRecipes(nil); err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}

func TestGetRecipe_Success(t *testing.T) {
	mockRecipe := Recipe{
		ID:    "recipe-1",
		Label: "Recipe One",
		LatestVersion: &Version{
			Revision: 3,
			Data:     map[string]interface{}{"version": float64(1), "label": "Recipe One"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/recipes/recipe-1/" {
			t.Errorf("expected path /api/recipes/recipe-1/, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(mockRecipe)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.GetRecipe("recipe-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Label != "Recipe One" {
		t.Errorf("expected label 'Recipe One', got '%s'", result.Label)
	}
	if result.LatestVersion == nil || result.LatestVersion.Revision != 3 {
		t.Errorf("expected latest version revision 3, got %+v", result.LatestVersion)
	}
}

func TestGetRecipe_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	if _, err := client.GetRecipe("does-not-exist"); err == nil {
		t.Fatal("expected an error for a 404 response, got nil")
	}
}

func TestGetRecipe_SpecialCharactersInID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Recipe{ID: "test/recipe"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.GetRecipe("test/recipe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "test/recipe" {
		t.Errorf("expected ID 'test/recipe', got '%s'", result.ID)
	}
}

func TestSearchRecipes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("search"); got != "melt" {
			t.Errorf("expected search 'melt', got '%s'", got)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("expected limit 5, got %s", got)
		}
		if got := r.URL.Query().Get("offset"); got != "10" {
			t.Errorf("expected offset 10, got %s", got)
		}
		json.NewEncoder(w).Encode(RecipeListResponse{})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	if _, err := client.SearchRecipes("melt", 5, 10); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchRecipes_NoLimitOrOffset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("limit") {
			t.Error("expected no limit param when limit <= 0")
		}
		if r.URL.Query().Has("offset") {
			t.Error("expected no offset param when offset <= 0")
		}
		json.NewEncoder(w).Encode(RecipeListResponse{})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	if _, err := client.SearchRecipes("melt", 0, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFilterByCategory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("category__name"); got != "preprocessing" {
			t.Errorf("expected category__name 'preprocessing', got '%s'", got)
		}
		json.NewEncoder(w).Encode(RecipeListResponse{})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	if _, err := client.FilterByCategory("preprocessing"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFilterByAuthor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("author__name"); got != "Jane" {
			t.Errorf("expected author__name 'Jane', got '%s'", got)
		}
		json.NewEncoder(w).Encode(RecipeListResponse{})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	if _, err := client.FilterByAuthor("Jane"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListFilterOptions_Success(t *testing.T) {
	mockOptions := FilterOptions{
		Categories: []string{"preprocessing"},
		Authors:    []string{"Jane"},
		Tags:       []string{"proteomics"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/recipes/filter_options/" {
			t.Errorf("expected path /api/recipes/filter_options/, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(mockOptions)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.ListFilterOptions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Categories) != 1 || result.Categories[0] != "preprocessing" {
		t.Errorf("unexpected categories: %v", result.Categories)
	}
}

func TestListFilterOptions_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	if _, err := client.ListFilterOptions(); err == nil {
		t.Fatal("expected an error, got nil")
	}
}
