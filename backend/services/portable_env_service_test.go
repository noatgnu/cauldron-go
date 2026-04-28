package services

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ulikunitz/xz"
)

func TestCalculateSHA256(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	testContent := []byte("Hello, World!")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash, err := calculateSHA256(testFile)
	if err != nil {
		t.Fatalf("calculateSHA256 failed: %v", err)
	}

	expectedHash := sha256.Sum256(testContent)
	expectedHashStr := hex.EncodeToString(expectedHash[:])

	if hash != expectedHashStr {
		t.Errorf("Hash mismatch: expected %s, got %s", expectedHashStr, hash)
	}

	if len(hash) != 64 {
		t.Errorf("Hash length should be 64, got %d", len(hash))
	}
}

func TestCalculateSHA256_NonExistentFile(t *testing.T) {
	_, err := calculateSHA256("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestCalculateSHA256_LargeFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "large.bin")

	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	data := make([]byte, 2*1024*1024) // 2MB
	for i := range data {
		data[i] = byte(i % 256)
	}

	if _, err := file.Write(data); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	file.Close()

	hash, err := calculateSHA256(testFile)
	if err != nil {
		t.Fatalf("calculateSHA256 failed: %v", err)
	}

	expectedHash := sha256.Sum256(data)
	expectedHashStr := hex.EncodeToString(expectedHash[:])

	if hash != expectedHashStr {
		t.Errorf("Hash mismatch for large file")
	}
}

func TestTranslatePlatform(t *testing.T) {
	cases := []struct {
		goos string
		want string
	}{
		{"windows", "win"},
		{"darwin", "darwin"},
		{"linux", "linux"},
		{"freebsd", "freebsd"},
	}
	for _, tc := range cases {
		got := translatePlatform(tc.goos)
		if got != tc.want {
			t.Errorf("translatePlatform(%q) = %q, want %q", tc.goos, got, tc.want)
		}
	}
}

func TestGetAppDataFolder(t *testing.T) {
	folder, err := getAppDataFolder()
	if err != nil {
		t.Fatalf("getAppDataFolder() error: %v", err)
	}
	if folder == "" {
		t.Error("expected non-empty folder path")
	}
	if !filepath.IsAbs(folder) {
		t.Errorf("expected absolute path, got %q", folder)
	}
}

func TestGetPortableEnvironmentPath_NotInstalled(t *testing.T) {
	svc := &PortableEnvService{}

	_, err := svc.GetPortableEnvironmentPath("python")
	if err == nil {
		t.Error("expected error when portable python is not installed, got nil")
	}

	_, err = svc.GetPortableEnvironmentPath("r-portable")
	if err == nil {
		t.Error("expected error when portable R is not installed, got nil")
	}
}

// setAppDataEnv injects a temp directory as the platform-specific app data root,
// returning the resolved cauldron app folder. Skips on darwin where no env override is available.
func setAppDataEnv(t *testing.T, base string) string {
	t.Helper()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("LOCALAPPDATA", base)
		return filepath.Join(base, "cauldron")
	case "darwin":
		t.Skip("path override not supported on darwin in this test")
		return ""
	default:
		t.Setenv("XDG_DATA_HOME", base)
		return filepath.Join(base, "cauldron")
	}
}

func TestGetPortableEnvironmentPath_Installed(t *testing.T) {
	platform := translatePlatform(runtime.GOOS)
	svc := &PortableEnvService{}

	t.Run("python", func(t *testing.T) {
		base := t.TempDir()
		appDataDir := setAppDataEnv(t, base)

		var exePath string
		if runtime.GOOS == "windows" {
			exePath = filepath.Join(appDataDir, "bin", platform, "python", "python.exe")
		} else {
			exePath = filepath.Join(appDataDir, "bin", platform, "python", "bin", "python")
		}
		if err := os.MkdirAll(filepath.Dir(exePath), 0755); err != nil {
			t.Fatalf("failed to create dirs: %v", err)
		}
		if err := os.WriteFile(exePath, []byte("fake python"), 0755); err != nil {
			t.Fatalf("failed to create fake python: %v", err)
		}

		got, err := svc.GetPortableEnvironmentPath("python")
		if err != nil {
			t.Fatalf("GetPortableEnvironmentPath(python) error: %v", err)
		}
		if got != exePath {
			t.Errorf("got %q, want %q", got, exePath)
		}
	})

	t.Run("r-portable", func(t *testing.T) {
		base := t.TempDir()
		appDataDir := setAppDataEnv(t, base)

		var exePath string
		if runtime.GOOS == "windows" {
			exePath = filepath.Join(appDataDir, "bin", platform, "R-Portable", "bin", "Rscript.exe")
		} else {
			exePath = filepath.Join(appDataDir, "bin", platform, "R-Portable", "bin", "Rscript")
		}
		if err := os.MkdirAll(filepath.Dir(exePath), 0755); err != nil {
			t.Fatalf("failed to create dirs: %v", err)
		}
		if err := os.WriteFile(exePath, []byte("fake Rscript"), 0755); err != nil {
			t.Fatalf("failed to create fake Rscript: %v", err)
		}

		got, err := svc.GetPortableEnvironmentPath("r-portable")
		if err != nil {
			t.Fatalf("GetPortableEnvironmentPath(r-portable) error: %v", err)
		}
		if got != exePath {
			t.Errorf("got %q, want %q", got, exePath)
		}
	})
}

func makeReleasesJSON(assets []GitHubAsset) []byte {
	releases := []GitHubRelease{
		{TagName: "v1.0.0", Assets: assets},
	}
	data, _ := json.Marshal(releases)
	return data
}

func TestGetPortableEnvironmentURL_Match(t *testing.T) {
	cases := []struct {
		name        string
		platform    string
		arch        string
		environment string
		assets      []GitHubAsset
		wantURL     string
	}{
		{
			name:        "python linux x86_64",
			platform:    "linux",
			arch:        "x86_64",
			environment: "python",
			assets: []GitHubAsset{
				{Name: "python-linux-x86_64.tar.xz", BrowserDownloadURL: "http://example.com/python-linux.tar.xz"},
				{Name: "r-portable-linux-x86_64.tar.xz", BrowserDownloadURL: "http://example.com/r.tar.xz"},
			},
			wantURL: "http://example.com/python-linux.tar.xz",
		},
		{
			name:        "r-portable linux x86_64",
			platform:    "linux",
			arch:        "x86_64",
			environment: "r-portable",
			assets: []GitHubAsset{
				{Name: "python-linux-x86_64.tar.xz", BrowserDownloadURL: "http://example.com/python.tar.xz"},
				{Name: "r-portable-linux-x86_64.tar.xz", BrowserDownloadURL: "http://example.com/r-portable-linux.tar.xz"},
			},
			wantURL: "http://example.com/r-portable-linux.tar.xz",
		},
		{
			name:        "python win x86_64 skips darwin asset",
			platform:    "win",
			arch:        "x86_64",
			environment: "python",
			assets: []GitHubAsset{
				{Name: "python-darwin-arm64.tar.xz", BrowserDownloadURL: "http://example.com/python-darwin.tar.xz"},
				{Name: "python-win-x86_64.tar.xz", BrowserDownloadURL: "http://example.com/python-win.tar.xz"},
			},
			wantURL: "http://example.com/python-win.tar.xz",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write(makeReleasesJSON(tc.assets))
			}))
			defer server.Close()

			svc := &PortableEnvService{releasesURL: server.URL}
			got, err := svc.GetPortableEnvironmentURL(tc.platform, tc.arch, "latest", tc.environment)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantURL {
				t.Errorf("got %q, want %q", got, tc.wantURL)
			}
		})
	}
}

func TestGetPortableEnvironmentURL_RealAPI_Win(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real network test")
	}
	svc := &PortableEnvService{}
	url, err := svc.GetPortableEnvironmentURL("win", "x86_64", "latest", "python")
	if err != nil {
		t.Fatalf("GetPortableEnvironmentURL(win/x86_64/latest/python) error: %v", err)
	}
	if url == "" {
		t.Error("expected non-empty URL")
	}
	t.Logf("Found URL: %s", url)
}

func TestGetPortableEnvironmentURL_RealAPI_AllCombinations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real network test")
	}
	cases := []struct {
		platform    string
		arch        string
		environment string
	}{
		{"win", "x86_64", "python"},
		{"win", "x86_64", "r-portable"},
		{"linux", "x86_64", "python"},
		{"linux", "x86_64", "r-portable"},
		{"darwin", "arm64", "python"},
		{"darwin", "arm64", "r-portable"},
	}
	svc := &PortableEnvService{}
	for _, tc := range cases {
		t.Run(tc.platform+"_"+tc.arch+"_"+tc.environment, func(t *testing.T) {
			url, err := svc.GetPortableEnvironmentURL(tc.platform, tc.arch, "latest", tc.environment)
			if err != nil {
				t.Errorf("error: %v", err)
				return
			}
			if url == "" {
				t.Error("expected non-empty URL")
				return
			}
			t.Logf("URL: %s", url)
		})
	}
}

func TestGetPortableEnvironmentURL_NoMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(makeReleasesJSON([]GitHubAsset{
			{Name: "cauldron-windows-amd64.zip", BrowserDownloadURL: "http://example.com/cauldron.zip"},
		}))
	}))
	defer server.Close()

	svc := &PortableEnvService{releasesURL: server.URL}
	_, err := svc.GetPortableEnvironmentURL("linux", "x86_64", "latest", "python")
	if err == nil {
		t.Error("expected error when no matching asset, got nil")
	}
}

func TestGetPortableEnvironmentURL_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	svc := &PortableEnvService{releasesURL: server.URL}
	_, err := svc.GetPortableEnvironmentURL("linux", "x86_64", "latest", "python")
	if err == nil {
		t.Error("expected error on API failure, got nil")
	}
}

func TestGetPortableEnvironmentURL_FallbackToAssetURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		releases := []GitHubRelease{{
			TagName: "v1.0.0",
			Assets: []GitHubAsset{
				{Name: "python-linux-x86_64.tar.xz", URL: "http://example.com/api-url", BrowserDownloadURL: ""},
			},
		}}
		json.NewEncoder(w).Encode(releases)
	}))
	defer server.Close()

	svc := &PortableEnvService{releasesURL: server.URL}
	got, err := svc.GetPortableEnvironmentURL("linux", "x86_64", "latest", "python")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "http://example.com/api-url" {
		t.Errorf("expected fallback to asset URL, got %q", got)
	}
}

func makeTarXz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	xzWriter, err := xz.NewWriter(&buf)
	if err != nil {
		t.Fatalf("xz.NewWriter: %v", err)
	}
	tw := tar.NewWriter(xzWriter)

	dirs := map[string]bool{}
	for name := range files {
		parts := name
		for {
			dir := filepath.ToSlash(filepath.Dir(parts))
			if dir == "." || dirs[dir+"/"] {
				break
			}
			dirs[dir+"/"] = true
			hdr := &tar.Header{
				Name:     dir + "/",
				Typeflag: tar.TypeDir,
				Mode:     0755,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatalf("tar WriteHeader dir: %v", err)
			}
			parts = dir
		}
		body := []byte(files[name])
		hdr := &tar.Header{
			Name:     name,
			Typeflag: tar.TypeReg,
			Size:     int64(len(body)),
			Mode:     0755,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("tar WriteHeader file: %v", err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatalf("tar Write: %v", err)
		}
	}

	tw.Close()
	xzWriter.Close()
	return buf.Bytes()
}

func TestDownloadPortableEnvironment_Python(t *testing.T) {
	platform := translatePlatform(runtime.GOOS)

	var exeName string
	if runtime.GOOS == "windows" {
		exeName = "python.exe"
	} else {
		exeName = "bin/python"
	}

	archiveFiles := map[string]string{
		fmt.Sprintf("bin/%s/python/%s", platform, exeName):   "#!/bin/sh\necho python",
		fmt.Sprintf("bin/%s/python/lib/hello.txt", platform): "hello",
	}
	archiveData := makeTarXz(t, archiveFiles)

	hash := sha256.Sum256(archiveData)
	hashStr := hex.EncodeToString(hash[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/python.tar.xz":
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(archiveData)))
			w.Write(archiveData)
		case "/python.tar.xz.sha256":
			fmt.Fprintf(w, "%s  python.tar.xz\n", hashStr)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	base := t.TempDir()
	appDataDir := setAppDataEnv(t, base)

	fs := &FileService{}
	svc := &PortableEnvService{
		fileService:      fs,
		progressNotifier: &ProgressNotifier{},
	}

	err := svc.DownloadPortableEnvironment(server.URL+"/python.tar.xz", "python")
	if err != nil {
		t.Fatalf("DownloadPortableEnvironment error: %v", err)
	}

	var expectedExe string
	if runtime.GOOS == "windows" {
		expectedExe = filepath.Join(appDataDir, "bin", platform, "python", "python.exe")
	} else {
		expectedExe = filepath.Join(appDataDir, "bin", platform, "python", "bin", "python")
	}
	if _, err := os.Stat(expectedExe); os.IsNotExist(err) {
		t.Errorf("expected executable at %s, not found", expectedExe)
	}
}

func TestCopyFileBuffered(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	content := []byte("copy me")

	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	buf := make([]byte, 4096)
	if err := copyFileBuffered(src, dst, buf); err != nil {
		t.Fatalf("copyFileBuffered: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

func TestCopyFileBuffered_MissingSrc(t *testing.T) {
	dir := t.TempDir()
	buf := make([]byte, 4096)
	err := copyFileBuffered(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dst"), buf)
	if err == nil {
		t.Error("expected error for missing source, got nil")
	}
}

func TestCopyDirWithProgress(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")

	files := map[string]string{
		"a.txt":       "content a",
		"sub/b.txt":   "content b",
		"sub/c/d.txt": "content d",
	}
	for rel, body := range files {
		full := filepath.Join(src, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	svc := &PortableEnvService{progressNotifier: &ProgressNotifier{}}
	if err := svc.copyDirWithProgress(src, dst, "test"); err != nil {
		t.Fatalf("copyDirWithProgress: %v", err)
	}

	for rel, want := range files {
		got, err := os.ReadFile(filepath.Join(dst, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("missing file %s: %v", rel, err)
			continue
		}
		if string(got) != want {
			t.Errorf("file %s: got %q, want %q", rel, got, want)
		}
	}
}
