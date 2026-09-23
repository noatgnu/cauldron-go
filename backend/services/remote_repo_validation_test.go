package services

import (
	"testing"
)

func TestValidateRemoteRepoURL(t *testing.T) {
	tests := []struct {
		name         string
		repoURL      string
		allowedHosts []string
		restrict     bool
		wantErr      bool
	}{
		{"valid https public host", "https://github.com/owner/repo.git", nil, false, false},
		{"http scheme rejected", "http://github.com/owner/repo.git", nil, false, true},
		{"file scheme rejected", "file:///etc/passwd", nil, false, true},
		{"no scheme rejected", "github.com/owner/repo.git", nil, false, true},
		{"ssh scheme rejected", "ssh://git@github.com/owner/repo.git", nil, false, true},
		{"scp shorthand rejected", "git@github.com:owner/repo.git", nil, false, true},
		{"loopback ip rejected", "https://127.0.0.1/owner/repo.git", nil, false, true},
		{"loopback hostname rejected", "https://localhost/owner/repo.git", nil, false, true},
		{"link-local metadata ip rejected", "https://169.254.169.254/owner/repo.git", nil, false, true},
		{"private class a rejected", "https://10.0.0.5/owner/repo.git", nil, false, true},
		{"private class b rejected", "https://172.16.0.5/owner/repo.git", nil, false, true},
		{"private class c rejected", "https://192.168.1.5/owner/repo.git", nil, false, true},
		{"unspecified ip rejected", "https://0.0.0.0/owner/repo.git", nil, false, true},
		{"unresolvable host rejected", "https://this-host-does-not-exist.invalid/owner/repo.git", nil, false, true},
		{"unparseable url rejected", "https://[::1", nil, false, true},
		{"private host allowed via whitelist", "https://git.internal.example/owner/repo.git", []string{"git.internal.example"}, false, false},
		{"whitelist match is case-insensitive", "https://Git.Internal.Example/owner/repo.git", []string{"git.internal.example"}, false, false},
		{"host not on whitelist still rejected", "https://10.0.0.5/owner/repo.git", []string{"git.internal.example"}, false, true},
		{"whitelist does not bypass scheme check", "http://git.internal.example/owner/repo.git", []string{"git.internal.example"}, false, true},
		{"strict mode allows a listed host", "https://git.internal.example/owner/repo.git", []string{"git.internal.example"}, true, false},
		{"strict mode rejects an unlisted public host", "https://github.com/owner/repo.git", []string{"git.internal.example"}, true, true},
		{"strict mode with empty allowlist rejects everything", "https://github.com/owner/repo.git", nil, true, true},
		{"strict mode still rejects http even for a listed host", "http://git.internal.example/owner/repo.git", []string{"git.internal.example"}, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRemoteRepoURL(tt.repoURL, tt.allowedHosts, tt.restrict)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateRemoteRepoURL(%q, %v, %v) = nil, want error", tt.repoURL, tt.allowedHosts, tt.restrict)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateRemoteRepoURL(%q, %v, %v) = %v, want nil", tt.repoURL, tt.allowedHosts, tt.restrict, err)
			}
		})
	}
}
