package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
	"gopkg.in/yaml.v3"
)

var githubRawHTTPClient = &http.Client{Timeout: 5 * time.Second}

var githubHTTPSPattern = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+?)(?:\.git)?/?$`)
var githubSSHPattern = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(?:\.git)?$`)

// parseGitHubOwnerRepo extracts owner/repo from an https:// or git@ GitHub URL.
func parseGitHubOwnerRepo(repoURL string) (owner, repo string, ok bool) {
	if m := githubHTTPSPattern.FindStringSubmatch(repoURL); m != nil {
		return m[1], m[2], true
	}
	if m := githubSSHPattern.FindStringSubmatch(repoURL); m != nil {
		return m[1], m[2], true
	}
	return "", "", false
}

// fetchPluginYAMLFromGitHubRaw reads plugin.yaml straight from GitHub's raw-content API, skipping
// a full repository clone. Returns ok=false on any failure (non-GitHub host, private repo, network
// error) so the caller falls back to the slower but universally reliable git clone.
func fetchPluginYAMLFromGitHubRaw(repoURL string) (*models.PluginDefinition, bool) {
	owner, repo, matched := parseGitHubOwnerRepo(repoURL)
	if !matched {
		return nil, false
	}

	branch, err := fetchGitHubDefaultBranch(owner, repo)
	if err != nil {
		return nil, false
	}

	for _, name := range []string{"plugin.yaml", "plugin.yml"} {
		rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, branch, name)
		data, err := fetchGitHubURLBody(rawURL)
		if err != nil {
			continue
		}
		var def models.PluginDefinition
		if err := yaml.Unmarshal(data, &def); err != nil {
			continue
		}
		return &def, true
	}
	return nil, false
}

func fetchGitHubDefaultBranch(owner, repo string) (string, error) {
	data, err := fetchGitHubURLBody(fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo))
	if err != nil {
		return "", err
	}
	var info struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return "", err
	}
	if info.DefaultBranch == "" {
		return "", fmt.Errorf("no default branch reported for %s/%s", owner, repo)
	}
	return info.DefaultBranch, nil
}

func fetchGitHubURLBody(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "cauldron-go")

	resp, err := githubRawHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}
