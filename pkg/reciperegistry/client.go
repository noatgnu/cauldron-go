package reciperegistry

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

type Version struct {
	Revision  int                    `json:"revision"`
	Data      map[string]interface{} `json:"data"`
	Changelog string                 `json:"changelog,omitempty"`
	CreatedAt string                 `json:"created_at"`
}

type Recipe struct {
	ID            string    `json:"id"`
	Label         string    `json:"label"`
	Description   string    `json:"description,omitempty"`
	Author        *Author   `json:"author"`
	Category      *Category `json:"category"`
	Tags          []Tag     `json:"tags,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     string    `json:"created_at"`
	UpdatedAt     string    `json:"updated_at"`
	LatestVersion *Version  `json:"latest_version"`
}

type RecipeListResponse struct {
	Count    int      `json:"count"`
	Next     *string  `json:"next"`
	Previous *string  `json:"previous"`
	Results  []Recipe `json:"results"`
}

type FilterOptions struct {
	Categories []string `json:"categories"`
	Authors    []string `json:"authors"`
	Tags       []string `json:"tags"`
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

func (c *Client) ListRecipes(params map[string]string) (*RecipeListResponse, error) {
	endpoint := fmt.Sprintf("%s/api/recipes/", c.BaseURL)

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

	var result RecipeListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) GetRecipe(recipeID string) (*Recipe, error) {
	endpoint := fmt.Sprintf("%s/api/recipes/%s/", c.BaseURL, url.PathEscape(recipeID))

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

	var recipe Recipe
	if err := json.NewDecoder(resp.Body).Decode(&recipe); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &recipe, nil
}

func (c *Client) SearchRecipes(searchQuery string, limit int, offset int) (*RecipeListResponse, error) {
	params := map[string]string{
		"search": searchQuery,
	}
	if limit > 0 {
		params["limit"] = fmt.Sprintf("%d", limit)
	}
	if offset > 0 {
		params["offset"] = fmt.Sprintf("%d", offset)
	}
	return c.ListRecipes(params)
}

func (c *Client) FilterByCategory(categoryName string) (*RecipeListResponse, error) {
	return c.ListRecipes(map[string]string{"category__name": categoryName})
}

func (c *Client) FilterByAuthor(authorName string) (*RecipeListResponse, error) {
	return c.ListRecipes(map[string]string{"author__name": authorName})
}

func (c *Client) ListFilterOptions() (*FilterOptions, error) {
	endpoint := fmt.Sprintf("%s/api/recipes/filter_options/", c.BaseURL)

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

	var result FilterOptions
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
