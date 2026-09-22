package services

import (
	"testing"
)

func TestValidateRemoteRepoURL(t *testing.T) {
	tests := []struct {
		name         string
		repoURL      string
		allowedHosts []string
		wantErr      bool
	}{
		{"valid https public host", "https://github.com/owner/repo.git", nil, false},
		{"http scheme rejected", "http://github.com/owner/repo.git", nil, true},
		{"file scheme rejected", "file:///etc/passwd", nil, true},
		{"no scheme rejected", "github.com/owner/repo.git", nil, true},
		{"ssh scheme rejected", "ssh://git@github.com/owner/repo.git", nil, true},
		{"scp shorthand rejected", "git@github.com:owner/repo.git", nil, true},
		{"loopback ip rejected", "https://127.0.0.1/owner/repo.git", nil, true},
		{"loopback hostname rejected", "https://localhost/owner/repo.git", nil, true},
		{"link-local metadata ip rejected", "https://169.254.169.254/owner/repo.git", nil, true},
		{"private class a rejected", "https://10.0.0.5/owner/repo.git", nil, true},
		{"private class b rejected", "https://172.16.0.5/owner/repo.git", nil, true},
		{"private class c rejected", "https://192.168.1.5/owner/repo.git", nil, true},
		{"unspecified ip rejected", "https://0.0.0.0/owner/repo.git", nil, true},
		{"unresolvable host rejected", "https://this-host-does-not-exist.invalid/owner/repo.git", nil, true},
		{"unparseable url rejected", "https://[::1", nil, true},
		{"private host allowed via whitelist", "https://git.internal.example/owner/repo.git", []string{"git.internal.example"}, false},
		{"whitelist match is case-insensitive", "https://Git.Internal.Example/owner/repo.git", []string{"git.internal.example"}, false},
		{"host not on whitelist still rejected", "https://10.0.0.5/owner/repo.git", []string{"git.internal.example"}, true},
		{"whitelist does not bypass scheme check", "http://git.internal.example/owner/repo.git", []string{"git.internal.example"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRemoteRepoURL(tt.repoURL, tt.allowedHosts)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateRemoteRepoURL(%q, %v) = nil, want error", tt.repoURL, tt.allowedHosts)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateRemoteRepoURL(%q, %v) = %v, want nil", tt.repoURL, tt.allowedHosts, err)
			}
		})
	}
}
