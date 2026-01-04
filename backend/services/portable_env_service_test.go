package services

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
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
