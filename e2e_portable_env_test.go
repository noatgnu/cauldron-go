package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

const testAPIPort = 9246

func callPortableTestAPI(path, method string, body interface{}) (map[string]interface{}, error) {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:%d%s", testAPIPort, path), bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func waitForPortableTestAPI(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/test/health", testAPIPort))
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("TestAPI not ready after %v", timeout)
}

func TestE2EPortableEnvDownload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping portable env download test in short mode (requires real network + ~230MB download)")
	}

	os.Setenv("CAULDRON_TEST_MODE", "true")
	defer os.Unsetenv("CAULDRON_TEST_MODE")

	app := NewApp()
	app.Initialize()
	defer app.Shutdown()

	testAPI := NewTestAPI(app)
	testAPI.Start(testAPIPort)

	time.Sleep(500 * time.Millisecond)

	if err := waitForPortableTestAPI(10 * time.Second); err != nil {
		t.Fatalf("TestAPI did not start: %v", err)
	}
	t.Log("[TestAPI] ready on port", testAPIPort)

	t.Run("resolves portable Python URL from real GitHub releases", func(t *testing.T) {
		result, err := callPortableTestAPI("/test/portable-env-url?platform=linux&arch=x86_64&version=latest&type=python", "GET", nil)
		if err != nil {
			t.Fatalf("API call failed: %v", err)
		}
		if errMsg, ok := result["error"].(string); ok && errMsg != "" {
			t.Fatalf("URL resolution error: %s", errMsg)
		}
		url, _ := result["url"].(string)
		if url == "" {
			t.Fatalf("Expected URL in response, got: %v", result)
		}
		t.Logf("Resolved Python URL: %s", url)
	})

	t.Run("resolves portable R URL from real GitHub releases", func(t *testing.T) {
		result, err := callPortableTestAPI("/test/portable-env-url?platform=linux&arch=x86_64&version=latest&type=r-portable", "GET", nil)
		if err != nil {
			t.Fatalf("API call failed: %v", err)
		}
		if errMsg, ok := result["error"].(string); ok && errMsg != "" {
			t.Fatalf("URL resolution error: %s", errMsg)
		}
		url, _ := result["url"].(string)
		if url == "" {
			t.Fatalf("Expected URL in response, got: %v", result)
		}
		t.Logf("Resolved R URL: %s", url)
	})

	t.Run("downloads and installs portable Python via TestAPI", func(t *testing.T) {
		urlResult, err := callPortableTestAPI("/test/portable-env-url?platform=linux&arch=x86_64&version=latest&type=python", "GET", nil)
		if err != nil {
			t.Fatalf("Failed to get download URL: %v", err)
		}
		downloadURL, _ := urlResult["url"].(string)
		if downloadURL == "" {
			t.Fatalf("No download URL returned: %v", urlResult)
		}
		t.Logf("Starting download from: %s", downloadURL)

		downloadDone := make(chan error, 1)
		go func() {
			result, err := callPortableTestAPI("/test/download-portable-env", "POST", map[string]string{
				"url":         downloadURL,
				"environment": "python",
			})
			if err != nil {
				downloadDone <- fmt.Errorf("HTTP call failed: %v", err)
				return
			}
			if errMsg, ok := result["error"].(string); ok && errMsg != "" {
				downloadDone <- fmt.Errorf("download error: %s", errMsg)
				return
			}
			downloadDone <- nil
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
				pathResult, err := callPortableTestAPI("/test/portable-env-path?type=python", "GET", nil)
				if err != nil {
					t.Fatalf("Failed to check installed path: %v", err)
				}
				path, _ := pathResult["path"].(string)
				if path == "" {
					t.Fatal("Download reported success but path is empty")
				}
				t.Logf("Portable Python installed at: %s", path)
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("Installed path does not exist on disk: %s", path)
				}
				return

			case <-ticker.C:
				pathResult, err := callPortableTestAPI("/test/portable-env-path?type=python", "GET", nil)
				if err != nil {
					t.Logf("Poll error (continuing): %v", err)
					continue
				}
				installed, _ := pathResult["installed"].(bool)
				path, _ := pathResult["path"].(string)
				t.Logf("Installation status: installed=%v path=%s", installed, path)
				if installed && path != "" {
					t.Logf("Portable Python installed at: %s", path)
					return
				}

			case <-timeout:
				t.Fatal("Timed out waiting for portable Python installation after 25 minutes")
			}
		}
	})
}
