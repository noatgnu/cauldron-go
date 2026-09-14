package services

import "testing"

func TestParseGitHubOwnerRepo(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantOK    bool
	}{
		{"https plain", "https://github.com/noatgnu/cauldron-go", "noatgnu", "cauldron-go", true},
		{"https with .git", "https://github.com/noatgnu/cauldron-go.git", "noatgnu", "cauldron-go", true},
		{"https trailing slash", "https://github.com/noatgnu/cauldron-go/", "noatgnu", "cauldron-go", true},
		{"http scheme", "http://github.com/noatgnu/cauldron-go", "noatgnu", "cauldron-go", true},
		{"ssh form", "git@github.com:noatgnu/cauldron-go.git", "noatgnu", "cauldron-go", true},
		{"ssh form no .git", "git@github.com:noatgnu/cauldron-go", "noatgnu", "cauldron-go", true},
		{"non-github host", "https://gitlab.com/noatgnu/cauldron-go", "", "", false},
		{"self-hosted git", "https://git.example.com/noatgnu/cauldron-go", "", "", false},
		{"not a url", "not-a-url", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, ok := parseGitHubOwnerRepo(tt.url)
			if ok != tt.wantOK || owner != tt.wantOwner || repo != tt.wantRepo {
				t.Errorf("parseGitHubOwnerRepo(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.url, owner, repo, ok, tt.wantOwner, tt.wantRepo, tt.wantOK)
			}
		})
	}
}

func TestFetchPluginYAMLFromGitHubRaw_NonGitHubHostReturnsFalse(t *testing.T) {
	_, ok := fetchPluginYAMLFromGitHubRaw("https://gitlab.com/someone/some-plugin")
	if ok {
		t.Error("expected a non-GitHub host to return ok=false immediately, without any network call")
	}
}

func TestFetchPluginYAMLFromGitHubRaw_MalformedURLReturnsFalse(t *testing.T) {
	_, ok := fetchPluginYAMLFromGitHubRaw("not a url at all")
	if ok {
		t.Error("expected a malformed URL to return ok=false")
	}
}
