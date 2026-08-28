package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetUvReleaseAsset(t *testing.T) {
	cases := []struct {
		goos, goarch, want string
	}{
		{"linux", "amd64", "uv-x86_64-unknown-linux-gnu.tar.gz"},
		{"linux", "arm64", "uv-aarch64-unknown-linux-gnu.tar.gz"},
		{"darwin", "amd64", "uv-x86_64-apple-darwin.tar.gz"},
		{"darwin", "arm64", "uv-aarch64-apple-darwin.tar.gz"},
		{"windows", "amd64", "uv-x86_64-pc-windows-msvc.zip"},
	}

	for _, tc := range cases {
		got, err := GetUvReleaseAsset(tc.goos, tc.goarch)
		if err != nil {
			t.Errorf("GetUvReleaseAsset(%q, %q) unexpected error: %v", tc.goos, tc.goarch, err)
		}
		if got != tc.want {
			t.Errorf("GetUvReleaseAsset(%q, %q) = %q, want %q", tc.goos, tc.goarch, got, tc.want)
		}
	}
}

func TestGetUvReleaseAsset_Unknown(t *testing.T) {
	if _, err := GetUvReleaseAsset("plan9", "amd64"); err == nil {
		t.Error("expected error for unknown platform, got nil")
	}
}

func TestGetUvPath_NotInstalled(t *testing.T) {
	setAppDataEnv(t, t.TempDir())

	svc := &UvService{}
	if _, err := svc.GetUvPath(); err == nil {
		t.Error("expected error when uv is not installed, got nil")
	}
}

func makeUvTarGz(t *testing.T, nestedDir string, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	if err := tarWriter.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     nestedDir + "/",
		Mode:     0755,
	}); err != nil {
		t.Fatalf("failed to write tar directory header: %v", err)
	}

	for name, content := range files {
		fullName := nestedDir + "/" + name
		header := &tar.Header{
			Typeflag: tar.TypeReg,
			Name:     fullName,
			Mode:     0755,
			Size:     int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("failed to write tar header: %v", err)
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write tar content: %v", err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	return buf.Bytes()
}

func TestDownloadUv(t *testing.T) {
	assetName, err := GetUvReleaseAsset(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("no uv release asset known for this platform: %v", err)
	}

	uvBinaryName := "uv"
	if runtime.GOOS == "windows" {
		uvBinaryName = "uv.exe"
	}

	archiveData := makeUvTarGz(t, "uv-test-platform", map[string]string{
		uvBinaryName: "#!/bin/sh\necho uv",
	})

	hash := sha256.Sum256(archiveData)
	hashStr := hex.EncodeToString(hash[:])

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/release":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"tag_name":"0.0.0-test","assets":[{"name":%q,"browser_download_url":%q}]}`,
				assetName, server.URL+"/asset")
		case "/asset":
			w.Write(archiveData)
		case "/asset.sha256":
			fmt.Fprintf(w, "%s  %s\n", hashStr, assetName)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	appDataDir := setAppDataEnv(t, t.TempDir())

	svc := &UvService{
		fileService:      &FileService{},
		progressNotifier: &ProgressNotifier{},
		releasesURL:      server.URL + "/release",
	}

	if err := svc.DownloadUv(); err != nil {
		t.Fatalf("DownloadUv error: %v", err)
	}

	platform := translatePlatform(runtime.GOOS)
	expectedPath := filepath.Join(appDataDir, "bin", platform, "uv", uvBinaryName)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected uv binary at %s, not found", expectedPath)
	}

	gotPath, err := svc.GetUvPath()
	if err != nil {
		t.Fatalf("GetUvPath error after install: %v", err)
	}
	if gotPath != expectedPath {
		t.Errorf("GetUvPath() = %q, want %q", gotPath, expectedPath)
	}
}
