package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestUpdateCheckService(t *testing.T, currentVersion string, handler http.HandlerFunc) *UpdateCheckService {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	svc := NewUpdateCheckService(currentVersion)
	svc.releasesURL = server.URL
	return svc
}

func TestCheckForUpdate_ReportsUpdateAvailable(t *testing.T) {
	svc := newTestUpdateCheckService(t, "v0.0.9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name":"v0.1.0","html_url":"https://github.com/noatgnu/cauldron-go/releases/tag/v0.1.0","body":"New stuff","published_at":"2026-09-04T00:00:00Z"}`))
	})

	info, err := svc.CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate failed: %v", err)
	}
	if !info.Available {
		t.Error("expected an update to be available")
	}
	if info.CurrentVersion != "v0.0.9" {
		t.Errorf("expected current version v0.0.9, got %q", info.CurrentVersion)
	}
	if info.LatestVersion != "v0.1.0" {
		t.Errorf("expected latest version v0.1.0, got %q", info.LatestVersion)
	}
	if info.ReleaseURL == "" || info.ReleaseNotes != "New stuff" {
		t.Errorf("expected release URL/notes to be populated, got %+v", info)
	}
}

func TestCheckForUpdate_AlreadyUpToDate(t *testing.T) {
	svc := newTestUpdateCheckService(t, "v0.1.0", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v0.1.0"}`))
	})

	info, err := svc.CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate failed: %v", err)
	}
	if info.Available {
		t.Error("expected no update to be available when already on the latest version")
	}
}

func TestCheckForUpdate_CurrentVersionNewerThanLatest(t *testing.T) {
	svc := newTestUpdateCheckService(t, "v0.2.0", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v0.1.0"}`))
	})

	info, err := svc.CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate failed: %v", err)
	}
	if info.Available {
		t.Error("expected no update to be available when the current version is newer")
	}
}

func TestCheckForUpdate_DoubleDigitPatchVersionsCompareNumerically(t *testing.T) {
	svc := newTestUpdateCheckService(t, "v0.0.9", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v0.0.10"}`))
	})

	info, err := svc.CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate failed: %v", err)
	}
	if !info.Available {
		t.Error("expected v0.0.10 to be recognized as newer than v0.0.9 (not a lexicographic comparison)")
	}
}

func TestCheckForUpdate_HTTPErrorStatus(t *testing.T) {
	svc := newTestUpdateCheckService(t, "v0.0.9", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	if _, err := svc.CheckForUpdate(); err == nil {
		t.Fatal("expected an error for a non-200 response, got nil")
	}
}

func TestCheckForUpdate_MalformedJSON(t *testing.T) {
	svc := newTestUpdateCheckService(t, "v0.0.9", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	})

	if _, err := svc.CheckForUpdate(); err == nil {
		t.Fatal("expected an error for malformed JSON, got nil")
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v0.0.9", "v0.0.9", 0},
		{"v0.0.9", "v0.0.10", -1},
		{"v0.0.10", "v0.0.9", 1},
		{"v1.0.0", "v0.9.9", 1},
		{"v0.1", "v0.1.0", 0},
		{"0.0.9", "v0.0.9", 0},
	}
	for _, c := range cases {
		got := compareVersions(c.a, c.b)
		if got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
