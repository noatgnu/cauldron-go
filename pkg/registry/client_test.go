package registry

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

func TestListPlugins_Success(t *testing.T) {
	mockResponse := PluginListResponse{
		Count: 2,
		Results: []Plugin{
			{
				ID:          "plugin-1",
				Name:        "Test Plugin 1",
				Description: "Description 1",
				Version:     "1.0.0",
			},
			{
				ID:          "plugin-2",
				Name:        "Test Plugin 2",
				Description: "Description 2",
				Version:     "2.0.0",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/plugins/" {
			t.Errorf("expected path /api/plugins/, got %s", r.URL.Path)
		}

		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept header application/json, got %s", r.Header.Get("Accept"))
		}

		searchQuery := r.URL.Query().Get("search")
		if searchQuery != "test" {
			t.Errorf("expected search query 'test', got '%s'", searchQuery)
		}

		limit := r.URL.Query().Get("limit")
		if limit != "10" {
			t.Errorf("expected limit 10, got %s", limit)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	params := map[string]string{
		"search": "test",
		"limit":  "10",
	}

	result, err := client.ListPlugins(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 2 {
		t.Errorf("expected count 2, got %d", result.Count)
	}

	if len(result.Results) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(result.Results))
	}

	if result.Results[0].Name != "Test Plugin 1" {
		t.Errorf("expected plugin name 'Test Plugin 1', got '%s'", result.Results[0].Name)
	}
}

func TestListPlugins_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.ListPlugins(map[string]string{})

	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestListPlugins_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.ListPlugins(map[string]string{})

	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetPlugin_Success(t *testing.T) {
	mockPlugin := Plugin{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Description: "Test Description",
		Version:     "1.0.0",
		Repository:  "https://github.com/test/plugin",
		CommitHash:  "abc123",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/plugins/test-plugin/"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept header application/json, got %s", r.Header.Get("Accept"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockPlugin)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.GetPlugin("test-plugin")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "test-plugin" {
		t.Errorf("expected ID 'test-plugin', got '%s'", result.ID)
	}

	if result.Name != "Test Plugin" {
		t.Errorf("expected name 'Test Plugin', got '%s'", result.Name)
	}

	if result.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", result.Version)
	}
}

func TestGetPlugin_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Plugin not found"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetPlugin("non-existent")

	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetPlugin_SpecialCharactersInID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockPlugin := Plugin{
			ID:   "test/plugin",
			Name: "Test",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockPlugin)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.GetPlugin("test/plugin")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "test/plugin" {
		t.Errorf("expected ID 'test/plugin', got '%s'", result.ID)
	}
}

func TestSearchPlugins_Success(t *testing.T) {
	mockResponse := PluginListResponse{
		Count: 1,
		Results: []Plugin{
			{
				ID:          "search-result",
				Name:        "Search Result",
				Description: "Found by search",
				Version:     "1.0.0",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		searchQuery := r.URL.Query().Get("search")
		if searchQuery != "test query" {
			t.Errorf("expected search query 'test query', got '%s'", searchQuery)
		}

		limit := r.URL.Query().Get("limit")
		if limit != "5" {
			t.Errorf("expected limit 5, got %s", limit)
		}

		offset := r.URL.Query().Get("offset")
		if offset != "10" {
			t.Errorf("expected offset 10, got %s", offset)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.SearchPlugins("test query", 5, 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 1 {
		t.Errorf("expected count 1, got %d", result.Count)
	}

	if len(result.Results) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(result.Results))
	}
}

func TestSearchPlugins_NoLimitOrOffset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := r.URL.Query().Get("limit")
		if limit != "" {
			t.Errorf("expected no limit parameter, got %s", limit)
		}

		offset := r.URL.Query().Get("offset")
		if offset != "" {
			t.Errorf("expected no offset parameter, got %s", offset)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PluginListResponse{})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.SearchPlugins("test", 0, 0)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFilterByCategory_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		category := r.URL.Query().Get("category__name")
		if category != "Analysis" {
			t.Errorf("expected category 'Analysis', got '%s'", category)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PluginListResponse{Count: 3})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.FilterByCategory("Analysis")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 3 {
		t.Errorf("expected count 3, got %d", result.Count)
	}
}

func TestFilterByAuthor_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		author := r.URL.Query().Get("author__name")
		if author != "John Doe" {
			t.Errorf("expected author 'John Doe', got '%s'", author)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PluginListResponse{Count: 2})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.FilterByAuthor("John Doe")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 2 {
		t.Errorf("expected count 2, got %d", result.Count)
	}
}

func TestListCategories_Success(t *testing.T) {
	mockCategories := []Category{
		{ID: 1, Name: "Analysis", Description: "Analysis tools"},
		{ID: 2, Name: "Visualization", Description: "Visualization tools"},
		{ID: 3, Name: "Utilities", Description: "Utility tools"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/categories/" {
			t.Errorf("expected path /api/categories/, got %s", r.URL.Path)
		}

		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept header application/json, got %s", r.Header.Get("Accept"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockCategories)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.ListCategories()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 3 {
		t.Errorf("expected count 3, got %d", result.Count)
	}

	if len(result.Results) != 3 {
		t.Errorf("expected 3 categories, got %d", len(result.Results))
	}

	if result.Results[0].Name != "Analysis" {
		t.Errorf("expected category name 'Analysis', got '%s'", result.Results[0].Name)
	}
}

func TestListCategories_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service unavailable"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.ListCategories()

	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestCheckUpdate_Success(t *testing.T) {
	latestTag := "v2.0.0"
	changelogURL := "https://github.com/test/plugin/changelog"

	mockUpdateInfo := UpdateInfo{
		PluginID:          "test-plugin",
		CurrentCommit:     "abc123",
		LatestCommit:      "def456",
		RecommendedCommit: "def456",
		LatestStableTag:   &latestTag,
		HasUpdate:         true,
		ChangelogURL:      &changelogURL,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/plugins/test-plugin/check_update/"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept header application/json, got %s", r.Header.Get("Accept"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockUpdateInfo)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.CheckUpdate("test-plugin")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PluginID != "test-plugin" {
		t.Errorf("expected plugin ID 'test-plugin', got '%s'", result.PluginID)
	}

	if !result.HasUpdate {
		t.Error("expected HasUpdate to be true")
	}

	if result.CurrentCommit != "abc123" {
		t.Errorf("expected current commit 'abc123', got '%s'", result.CurrentCommit)
	}

	if result.LatestCommit != "def456" {
		t.Errorf("expected latest commit 'def456', got '%s'", result.LatestCommit)
	}

	if result.LatestStableTag == nil || *result.LatestStableTag != "v2.0.0" {
		t.Error("expected latest stable tag 'v2.0.0'")
	}
}

func TestCheckUpdate_NoUpdate(t *testing.T) {
	mockUpdateInfo := UpdateInfo{
		PluginID:          "up-to-date-plugin",
		CurrentCommit:     "abc123",
		LatestCommit:      "abc123",
		RecommendedCommit: "abc123",
		HasUpdate:         false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockUpdateInfo)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.CheckUpdate("up-to-date-plugin")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.HasUpdate {
		t.Error("expected HasUpdate to be false")
	}
}

func TestCheckUpdate_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Plugin not found"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.CheckUpdate("non-existent")

	if err == nil {
		t.Error("expected error, got nil")
	}
}
