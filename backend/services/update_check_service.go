package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const defaultReleasesURL = "https://api.github.com/repos/noatgnu/cauldron-go/releases/latest"

type githubReleaseDetail struct {
	TagName     string `json:"tag_name"`
	HTMLURL     string `json:"html_url"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
}

// UpdateInfo describes the result of comparing the running app version against the latest GitHub release.
type UpdateInfo struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	Available      bool   `json:"available"`
	ReleaseURL     string `json:"releaseUrl"`
	ReleaseNotes   string `json:"releaseNotes"`
	PublishedAt    string `json:"publishedAt"`
}

// UpdateCheckService checks GitHub for a newer release than the currently running app version.
type UpdateCheckService struct {
	currentVersion string
	releasesURL    string
}

func NewUpdateCheckService(currentVersion string) *UpdateCheckService {
	return &UpdateCheckService{currentVersion: currentVersion}
}

// compareVersions numerically compares two "vMAJOR.MINOR.PATCH" strings, returning -1/0/1, padding missing components with 0.
func compareVersions(a, b string) int {
	aParts := strings.Split(strings.TrimPrefix(strings.TrimSpace(a), "v"), ".")
	bParts := strings.Split(strings.TrimPrefix(strings.TrimSpace(b), "v"), ".")

	max := len(aParts)
	if len(bParts) > max {
		max = len(bParts)
	}

	for i := 0; i < max; i++ {
		var av, bv int
		if i < len(aParts) {
			av, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bv, _ = strconv.Atoi(bParts[i])
		}
		if av != bv {
			if av < bv {
				return -1
			}
			return 1
		}
	}
	return 0
}

// CheckForUpdate fetches the latest GitHub release and reports whether it's newer than currentVersion.
func (s *UpdateCheckService) CheckForUpdate() (*UpdateInfo, error) {
	url := s.releasesURL
	if url == "" {
		url = defaultReleasesURL
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var release githubReleaseDetail
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, err
	}

	info := &UpdateInfo{
		CurrentVersion: s.currentVersion,
		LatestVersion:  release.TagName,
		ReleaseURL:     release.HTMLURL,
		ReleaseNotes:   release.Body,
		PublishedAt:    release.PublishedAt,
	}
	info.Available = release.TagName != "" && compareVersions(release.TagName, s.currentVersion) > 0

	return info, nil
}
