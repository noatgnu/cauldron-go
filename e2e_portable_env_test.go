package main

import (
	"os"
	"testing"
	"time"
)

func TestE2EPortableEnvDownload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping portable env download test in short mode (requires real network + ~230MB download)")
	}

	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	t.Run("resolves portable Python URL from real GitHub releases", func(t *testing.T) {
		url, err := app.GetPortableEnvironmentURL("linux", "x86_64", "latest", "python")
		if err != nil {
			t.Fatalf("URL resolution error: %v", err)
		}
		if url == "" {
			t.Fatal("Expected non-empty URL")
		}
		t.Logf("Resolved Python URL: %s", url)
	})

	t.Run("resolves portable R URL from real GitHub releases", func(t *testing.T) {
		url, err := app.GetPortableEnvironmentURL("linux", "x86_64", "latest", "r-portable")
		if err != nil {
			t.Fatalf("URL resolution error: %v", err)
		}
		if url == "" {
			t.Fatal("Expected non-empty URL")
		}
		t.Logf("Resolved R URL: %s", url)
	})

	t.Run("downloads and installs portable Python", func(t *testing.T) {
		downloadURL, err := app.GetPortableEnvironmentURL("linux", "x86_64", "latest", "python")
		if err != nil {
			t.Fatalf("Failed to get download URL: %v", err)
		}
		if downloadURL == "" {
			t.Fatal("No download URL returned")
		}
		t.Logf("Starting download from: %s", downloadURL)

		downloadDone := make(chan error, 1)
		go func() {
			downloadDone <- app.DownloadPortableEnvironment(downloadURL, "python")
		}()

		timeout := time.After(25 * time.Minute)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case err := <-downloadDone:
				if err != nil {
					t.Fatalf("Download failed: %v", err)
				}
				path, err := app.GetPortableEnvironmentPath("python")
				if err != nil || path == "" {
					t.Fatalf("Download reported success but path is unavailable: %v", err)
				}
				t.Logf("Portable Python installed at: %s", path)
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("Installed path does not exist on disk: %s", path)
				}
				return

			case <-ticker.C:
				path, err := app.GetPortableEnvironmentPath("python")
				t.Logf("Installation status: installed=%v path=%s", err == nil, path)

			case <-timeout:
				t.Fatal("Timed out waiting for portable Python installation after 25 minutes")
			}
		}
	})
}
