package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/pkg/registry"
)

func createTestSettingsService(registryURL string) *SettingsService {
	return &SettingsService{
		ctx: context.Background(),
		config: &models.Config{
			PluginRegistryURL: registryURL,
		},
	}
}

func createTestGitAuthService() *GitAuthService {
	return &GitAuthService{
		db: nil,
	}
}

func TestNewPluginRegistryService(t *testing.T) {
	ctx := context.Background()
	settingsService := createTestSettingsService("https://test.com")
	gitAuthService := createTestGitAuthService()

	service := NewPluginRegistryService(ctx, settingsService, gitAuthService)

	if service.ctx != ctx {
		t.Error("expected context to be set")
	}

	if service.configService == nil {
		t.Error("expected configService to be set")
	}

	if service.gitAuth == nil {
		t.Error("expected gitAuth to be set")
	}
}

func TestGetClient_Success(t *testing.T) {
	ctx := context.Background()
	testURL := "https://registry.example.com"
	settingsService := createTestSettingsService(testURL)
	gitAuthService := createTestGitAuthService()

	service := NewPluginRegistryService(ctx, settingsService, gitAuthService)

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

func TestGetClient_EmptyURL(t *testing.T) {
	ctx := context.Background()
	settingsService := createTestSettingsService("")
	gitAuthService := createTestGitAuthService()

	service := NewPluginRegistryService(ctx, settingsService, gitAuthService)

	_, err := service.getClient()
	if err == nil {
		t.Error("expected error when registry URL is empty")
	}

	expectedError := "plugin registry URL is not configured"
	if err.Error() != expectedError {
		t.Errorf("expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestGetClient_CachesClient(t *testing.T) {
	ctx := context.Background()
	settingsService := createTestSettingsService("https://test.com")
	gitAuthService := createTestGitAuthService()

	service := NewPluginRegistryService(ctx, settingsService, gitAuthService)

	client1, err1 := service.getClient()
	if err1 != nil {
		t.Fatalf("unexpected error: %v", err1)
	}

	client2, err2 := service.getClient()
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}

	if client1 != client2 {
		t.Error("expected getClient to return cached client instance")
	}
}

func TestListPlugins_Success(t *testing.T) {
	mockResponse := registry.PluginListResponse{
		Count: 3,
		Results: []registry.Plugin{
			{ID: "plugin-1", Name: "Plugin 1", Version: "1.0.0"},
			{ID: "plugin-2", Name: "Plugin 2", Version: "2.0.0"},
			{ID: "plugin-3", Name: "Plugin 3", Version: "3.0.0"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		searchQuery := r.URL.Query().Get("search")
		if searchQuery != "test" {
			t.Errorf("expected search query 'test', got '%s'", searchQuery)
		}

		category := r.URL.Query().Get("category__name")
		if category != "Analysis" {
			t.Errorf("expected category 'Analysis', got '%s'", category)
		}

		author := r.URL.Query().Get("author__name")
		if author != "John" {
			t.Errorf("expected author 'John', got '%s'", author)
		}

		limit := r.URL.Query().Get("limit")
		if limit != "10" {
			t.Errorf("expected limit 10, got %s", limit)
		}

		offset := r.URL.Query().Get("offset")
		if offset != "20" {
			t.Errorf("expected offset 20, got %s", offset)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	ctx := context.Background()
	settingsService := createTestSettingsService(server.URL)
	gitAuthService := createTestGitAuthService()

	service := NewPluginRegistryService(ctx, settingsService, gitAuthService)

	result, err := service.ListPlugins("test", "Analysis", "John", 10, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 3 {
		t.Errorf("expected count 3, got %d", result.Count)
	}

	if len(result.Results) != 3 {
		t.Errorf("expected 3 plugins, got %d", len(result.Results))
	}
}

func TestListPlugins_EmptyParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("search") != "" {
			t.Error("expected no search parameter")
		}

		if r.URL.Query().Get("category__name") != "" {
			t.Error("expected no category parameter")
		}

		if r.URL.Query().Get("author__name") != "" {
			t.Error("expected no author parameter")
		}

		if r.URL.Query().Get("limit") != "" {
			t.Error("expected no limit parameter")
		}

		if r.URL.Query().Get("offset") != "" {
			t.Error("expected no offset parameter")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(registry.PluginListResponse{})
	}))
	defer server.Close()

	ctx := context.Background()
	settingsService := createTestSettingsService(server.URL)
	gitAuthService := createTestGitAuthService()

	service := NewPluginRegistryService(ctx, settingsService, gitAuthService)

	_, err := service.ListPlugins("", "", "", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListPlugins_NoRegistryURL(t *testing.T) {
	ctx := context.Background()
	settingsService := createTestSettingsService("")
	gitAuthService := createTestGitAuthService()

	service := NewPluginRegistryService(ctx, settingsService, gitAuthService)

	_, err := service.ListPlugins("", "", "", 0, 0)
	if err == nil {
		t.Error("expected error when registry URL is not configured")
	}
}

func TestGetPlugin_Success(t *testing.T) {
	mockPlugin := registry.Plugin{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Description: "Test Description",
		Version:     "1.0.0",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/plugins/test-plugin/"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockPlugin)
	}))
	defer server.Close()

	ctx := context.Background()
	settingsService := createTestSettingsService(server.URL)
	gitAuthService := createTestGitAuthService()

	service := NewPluginRegistryService(ctx, settingsService, gitAuthService)

	result, err := service.GetPlugin("test-plugin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "test-plugin" {
		t.Errorf("expected ID 'test-plugin', got '%s'", result.ID)
	}

	if result.Name != "Test Plugin" {
		t.Errorf("expected name 'Test Plugin', got '%s'", result.Name)
	}
}

func TestGetPlugin_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))
	}))
	defer server.Close()

	ctx := context.Background()
	settingsService := createTestSettingsService(server.URL)
	gitAuthService := createTestGitAuthService()

	service := NewPluginRegistryService(ctx, settingsService, gitAuthService)

	_, err := service.GetPlugin("non-existent")
	if err == nil {
		t.Error("expected error for non-existent plugin")
	}
}

func TestListCategories_Success(t *testing.T) {
	mockCategories := []registry.Category{
		{ID: 1, Name: "Analysis"},
		{ID: 2, Name: "Visualization"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/categories/" {
			t.Errorf("expected path /api/categories/, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockCategories)
	}))
	defer server.Close()

	ctx := context.Background()
	settingsService := createTestSettingsService(server.URL)
	gitAuthService := createTestGitAuthService()

	service := NewPluginRegistryService(ctx, settingsService, gitAuthService)

	result, err := service.ListCategories()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 2 {
		t.Errorf("expected count 2, got %d", result.Count)
	}

	if len(result.Results) != 2 {
		t.Errorf("expected 2 categories, got %d", len(result.Results))
	}
}

func TestInstallPluginFromRegistry_Success(t *testing.T) {
	mockPlugin := registry.Plugin{
		ID:         "test-plugin",
		Name:       "Test Plugin",
		Repository: "https://github.com/test/plugin",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockPlugin)
	}))
	defer server.Close()

	ctx := context.Background()
	settingsService := createTestSettingsService(server.URL)
	gitAuthService := createTestGitAuthService()

	service := NewPluginRegistryService(ctx, settingsService, gitAuthService)

	err := service.InstallPluginFromRegistry("test-plugin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallPluginFromRegistry_NoRepository(t *testing.T) {
	mockPlugin := registry.Plugin{
		ID:         "test-plugin",
		Name:       "Test Plugin",
		Repository: "",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockPlugin)
	}))
	defer server.Close()

	ctx := context.Background()
	settingsService := createTestSettingsService(server.URL)
	gitAuthService := createTestGitAuthService()

	service := NewPluginRegistryService(ctx, settingsService, gitAuthService)

	err := service.InstallPluginFromRegistry("test-plugin")
	if err == nil {
		t.Error("expected error when plugin has no repository")
	}

	expectedError := "plugin test-plugin does not have a repository URL"
	if err.Error() != expectedError {
		t.Errorf("expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestCheckUpdate_Success(t *testing.T) {
	latestTag := "v2.0.0"
	mockUpdateInfo := registry.UpdateInfo{
		PluginID:          "test-plugin",
		CurrentCommit:     "abc123",
		LatestCommit:      "def456",
		RecommendedCommit: "def456",
		LatestStableTag:   &latestTag,
		HasUpdate:         true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/plugins/test-plugin/check_update/"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockUpdateInfo)
	}))
	defer server.Close()

	ctx := context.Background()
	settingsService := createTestSettingsService(server.URL)
	gitAuthService := createTestGitAuthService()

	service := NewPluginRegistryService(ctx, settingsService, gitAuthService)

	result, err := service.CheckUpdate("test-plugin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
}
