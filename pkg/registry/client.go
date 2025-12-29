package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Author struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Runtime struct {
	ID     int    `json:"id"`
	Plugin string `json:"plugin"`
	Type   string `json:"type"`
	Script string `json:"script"`
}

type Input struct {
	ID          int     `json:"id"`
	Plugin      string  `json:"plugin"`
	Name        string  `json:"name"`
	Label       string  `json:"label"`
	Type        string  `json:"type"`
	Required    bool    `json:"required"`
	Default     string  `json:"default,omitempty"`
	Description string  `json:"description,omitempty"`
	Placeholder string  `json:"placeholder,omitempty"`
	Accept      string  `json:"accept,omitempty"`
	Multiple    bool    `json:"multiple"`
	SourceFile  string  `json:"sourceFile,omitempty"`
	Min         float64 `json:"min,omitempty"`
	Max         float64 `json:"max,omitempty"`
	Step        float64 `json:"step,omitempty"`
}

type Output struct {
	ID          int    `json:"id"`
	Plugin      string `json:"plugin"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Format      string `json:"format,omitempty"`
}

type EnvVariable struct {
	ID          int     `json:"id"`
	Plugin      string  `json:"plugin"`
	Name        string  `json:"name"`
	Label       string  `json:"label"`
	Type        string  `json:"type"`
	Required    bool    `json:"required"`
	Default     string  `json:"default,omitempty"`
	Description string  `json:"description,omitempty"`
	Placeholder string  `json:"placeholder,omitempty"`
	Accept      string  `json:"accept,omitempty"`
	Multiple    bool    `json:"multiple"`
	SourceFile  string  `json:"sourceFile,omitempty"`
	Min         float64 `json:"min,omitempty"`
	Max         float64 `json:"max,omitempty"`
	Step        float64 `json:"step,omitempty"`
}

type Plugin struct {
	ID                     string        `json:"id"`
	Name                   string        `json:"name"`
	Description            string        `json:"description"`
	Version                string        `json:"version"`
	Author                 *Author       `json:"author"`
	Category               *Category     `json:"category"`
	Icon                   string        `json:"icon,omitempty"`
	Repository             string        `json:"repository,omitempty"`
	CommitHash             string        `json:"commit_hash,omitempty"`
	RecommendedCommit      *string       `json:"recommended_commit,omitempty"`
	LatestStableTag        *string       `json:"latest_stable_tag,omitempty"`
	RequiresAuthentication bool          `json:"requires_authentication"`
	CreatedAt              string        `json:"created_at"`
	UpdatedAt              string        `json:"updated_at"`
	Tags                   []Tag         `json:"tags,omitempty"`
	Runtime                *Runtime      `json:"runtime"`
	Inputs                 []Input       `json:"inputs"`
	Outputs                []Output      `json:"outputs"`
	EnvVariables           []EnvVariable `json:"env_variables"`
	Readme                 string        `json:"readme,omitempty"`
}

type UpdateInfo struct {
	PluginID          string  `json:"plugin_id"`
	CurrentCommit     string  `json:"current_commit"`
	LatestCommit      string  `json:"latest_commit"`
	RecommendedCommit string  `json:"recommended_commit"`
	LatestStableTag   *string `json:"latest_stable_tag"`
	HasUpdate         bool    `json:"has_update"`
	ChangelogURL      *string `json:"changelog_url"`
}

type PluginListResponse struct {
	Count    int      `json:"count"`
	Next     *string  `json:"next"`
	Previous *string  `json:"previous"`
	Results  []Plugin `json:"results"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) ListPlugins(params map[string]string) (*PluginListResponse, error) {
	endpoint := fmt.Sprintf("%s/api/plugins/", c.BaseURL)

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	for key, value := range params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var result PluginListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) GetPlugin(pluginID string) (*Plugin, error) {
	endpoint := fmt.Sprintf("%s/api/plugins/%s/", c.BaseURL, url.PathEscape(pluginID))

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var plugin Plugin
	if err := json.NewDecoder(resp.Body).Decode(&plugin); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &plugin, nil
}

func (c *Client) SearchPlugins(searchQuery string, limit int, offset int) (*PluginListResponse, error) {
	params := map[string]string{
		"search": searchQuery,
	}

	if limit > 0 {
		params["limit"] = fmt.Sprintf("%d", limit)
	}

	if offset > 0 {
		params["offset"] = fmt.Sprintf("%d", offset)
	}

	return c.ListPlugins(params)
}

func (c *Client) FilterByCategory(categoryName string) (*PluginListResponse, error) {
	params := map[string]string{
		"category__name": categoryName,
	}
	return c.ListPlugins(params)
}

func (c *Client) FilterByAuthor(authorName string) (*PluginListResponse, error) {
	params := map[string]string{
		"author__name": authorName,
	}
	return c.ListPlugins(params)
}

type CategoryListResponse struct {
	Count    int        `json:"count"`
	Next     *string    `json:"next"`
	Previous *string    `json:"previous"`
	Results  []Category `json:"results"`
}

func (c *Client) ListCategories() (*CategoryListResponse, error) {
	endpoint := fmt.Sprintf("%s/api/categories/", c.BaseURL)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var categories []Category
	if err := json.Unmarshal(body, &categories); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &CategoryListResponse{
		Count:    len(categories),
		Next:     nil,
		Previous: nil,
		Results:  categories,
	}, nil
}

func (c *Client) CheckUpdate(pluginID string) (*UpdateInfo, error) {
	endpoint := fmt.Sprintf("%s/api/plugins/%s/check_update/", c.BaseURL, url.PathEscape(pluginID))

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var updateInfo UpdateInfo
	if err := json.NewDecoder(resp.Body).Decode(&updateInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &updateInfo, nil
}
