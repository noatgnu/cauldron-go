package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func generateTestSSHKey(t *testing.T, withPassphrase bool) (privateKeyPath string, publicKeyPath string, passphrase string) {
	tmpDir := t.TempDir()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateKeyPath = filepath.Join(tmpDir, "id_rsa")
	privateKeyFile, err := os.Create(privateKeyPath)
	if err != nil {
		t.Fatalf("failed to create private key file: %v", err)
	}
	defer privateKeyFile.Close()

	if withPassphrase {
		passphrase = "test-passphrase-123"
		encryptedPEM, err := x509.EncryptPEMBlock(rand.Reader, privateKeyPEM.Type, privateKeyPEM.Bytes, []byte(passphrase), x509.PEMCipherAES256)
		if err != nil {
			t.Fatalf("failed to encrypt PEM: %v", err)
		}
		if err := pem.Encode(privateKeyFile, encryptedPEM); err != nil {
			t.Fatalf("failed to encode encrypted PEM: %v", err)
		}
	} else {
		if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
			t.Fatalf("failed to encode PEM: %v", err)
		}
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to generate SSH public key: %v", err)
	}

	publicKeyPath = filepath.Join(tmpDir, "id_rsa.pub")
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	if err := os.WriteFile(publicKeyPath, publicKeyBytes, 0644); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	return privateKeyPath, publicKeyPath, passphrase
}

func createTestDatabaseForGitAuth(t *testing.T) *DatabaseService {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=DELETE"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&GitAuthConfig{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	service := &DatabaseService{
		ctx: context.Background(),
		db:  db,
	}

	t.Cleanup(func() {
		sqlDB, closeErr := service.db.DB()
		if closeErr != nil {
			t.Logf("Error getting underlying DB for cleanup: %v", closeErr)
			return
		}
		if sqlDB != nil {
			if runtime.GOOS == "windows" {
				service.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
			}
			if closeErr := sqlDB.Close(); closeErr != nil {
				t.Logf("Error closing database in cleanup: %v", closeErr)
			}
		}
		if runtime.GOOS == "windows" {
			time.Sleep(500 * time.Millisecond)
		}
	})

	return service
}

func TestNewGitAuthService(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	if service == nil {
		t.Fatal("expected non-nil service")
	}

	if service.db != dbService {
		t.Error("expected db to be set")
	}
}

func TestNormalizeRepoURL(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "SSH format with git@",
			input:    "git@github.com:user/repo.git",
			expected: "ssh://git@github.com/user/repo.git",
		},
		{
			name:     "HTTPS URL unchanged",
			input:    "https://github.com/user/repo.git",
			expected: "https://github.com/user/repo.git",
		},
		{
			name:     "SSH URL unchanged",
			input:    "ssh://git@github.com/user/repo.git",
			expected: "ssh://git@github.com/user/repo.git",
		},
		{
			name:     "git@ without path separator",
			input:    "git@github.com",
			expected: "git@github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.normalizeRepoURL(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidateSSHKey_WithoutPassphrase(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath, _, _ := generateTestSSHKey(t, false)

	err := service.ValidateSSHKey(privateKeyPath, "")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateSSHKey_WithPassphrase(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath, _, passphrase := generateTestSSHKey(t, true)

	err := service.ValidateSSHKey(privateKeyPath, passphrase)
	if err != nil {
		t.Errorf("expected no error with correct passphrase, got: %v", err)
	}
}

func TestValidateSSHKey_WrongPassphrase(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath, _, _ := generateTestSSHKey(t, true)

	err := service.ValidateSSHKey(privateKeyPath, "wrong-passphrase")
	if err == nil {
		t.Error("expected error with wrong passphrase")
	}
}

func TestValidateSSHKey_MissingPassphrase(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath, _, _ := generateTestSSHKey(t, true)

	err := service.ValidateSSHKey(privateKeyPath, "")
	if err == nil {
		t.Error("expected error when passphrase required but not provided")
	}
}

func TestValidateSSHKey_NonExistentFile(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	err := service.ValidateSSHKey("/non/existent/path", "")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestValidateSSHKey_InvalidKeyFormat(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	tmpDir := t.TempDir()
	invalidKeyPath := filepath.Join(tmpDir, "invalid_key")
	if err := os.WriteFile(invalidKeyPath, []byte("not a valid ssh key"), 0600); err != nil {
		t.Fatalf("failed to create invalid key file: %v", err)
	}

	err := service.ValidateSSHKey(invalidKeyPath, "")
	if err == nil {
		t.Error("expected error for invalid key format")
	}
}

func TestCreateSSHAuth_WithoutPassphrase(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath, _, _ := generateTestSSHKey(t, false)

	auth, err := service.createSSHAuth(privateKeyPath, "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if auth == nil {
		t.Error("expected non-nil auth method")
	}
}

func TestCreateSSHAuth_WithPassphrase(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath, _, passphrase := generateTestSSHKey(t, true)

	auth, err := service.createSSHAuth(privateKeyPath, passphrase)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if auth == nil {
		t.Error("expected non-nil auth method")
	}
}

func TestCreateSSHAuth_WrongPassphrase(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath, _, _ := generateTestSSHKey(t, true)

	_, err := service.createSSHAuth(privateKeyPath, "wrong-passphrase")
	if err == nil {
		t.Error("expected error with wrong passphrase")
	}
}

func TestCreateSSHAuth_NonExistentFile(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	_, err := service.createSSHAuth("/non/existent/key", "")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestSaveGitAuthConfig_Success(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath, _, _ := generateTestSSHKey(t, false)
	repoURL := "git@github.com:user/repo.git"

	err := service.SaveGitAuthConfig(repoURL, privateKeyPath, "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	config, err := service.GetGitAuthConfig(repoURL)
	if err != nil {
		t.Fatalf("failed to retrieve saved config: %v", err)
	}

	if config.RepositoryURL != "ssh://git@github.com/user/repo.git" {
		t.Errorf("expected normalized URL, got: %s", config.RepositoryURL)
	}

	if config.SSHKeyPath != privateKeyPath {
		t.Errorf("expected key path %s, got: %s", privateKeyPath, config.SSHKeyPath)
	}
}

func TestSaveGitAuthConfig_WithPassphrase(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath, _, passphrase := generateTestSSHKey(t, true)
	repoURL := "https://github.com/user/repo.git"

	err := service.SaveGitAuthConfig(repoURL, privateKeyPath, passphrase)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	config, err := service.GetGitAuthConfig(repoURL)
	if err != nil {
		t.Fatalf("failed to retrieve saved config: %v", err)
	}

	if config.SSHKeyPassphrase != passphrase {
		t.Error("expected passphrase to be saved")
	}
}

func TestSaveGitAuthConfig_UpdateExisting(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath1, _, _ := generateTestSSHKey(t, false)
	privateKeyPath2, _, _ := generateTestSSHKey(t, false)
	repoURL := "git@github.com:user/repo.git"

	err := service.SaveGitAuthConfig(repoURL, privateKeyPath1, "")
	if err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	err = service.SaveGitAuthConfig(repoURL, privateKeyPath2, "")
	if err != nil {
		t.Fatalf("failed to update config: %v", err)
	}

	config, err := service.GetGitAuthConfig(repoURL)
	if err != nil {
		t.Fatalf("failed to retrieve updated config: %v", err)
	}

	if config.SSHKeyPath != privateKeyPath2 {
		t.Errorf("expected updated key path %s, got: %s", privateKeyPath2, config.SSHKeyPath)
	}

	configs, err := service.GetAllGitAuthConfigs()
	if err != nil {
		t.Fatalf("failed to get all configs: %v", err)
	}

	if len(configs) != 1 {
		t.Errorf("expected 1 config after update, got: %d", len(configs))
	}
}

func TestSaveGitAuthConfig_InvalidKey(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	tmpDir := t.TempDir()
	invalidKeyPath := filepath.Join(tmpDir, "invalid_key")
	if err := os.WriteFile(invalidKeyPath, []byte("not valid"), 0600); err != nil {
		t.Fatalf("failed to create invalid key: %v", err)
	}

	err := service.SaveGitAuthConfig("git@github.com:user/repo.git", invalidKeyPath, "")
	if err == nil {
		t.Error("expected error when saving invalid key")
	}
}

func TestGetGitAuthConfig_NotFound(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	_, err := service.GetGitAuthConfig("git@github.com:nonexistent/repo.git")
	if err == nil {
		t.Error("expected error when config not found")
	}
}

func TestGetAllGitAuthConfigs_Empty(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	configs, err := service.GetAllGitAuthConfigs()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(configs) != 0 {
		t.Errorf("expected 0 configs, got: %d", len(configs))
	}
}

func TestGetAllGitAuthConfigs_Multiple(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath, _, _ := generateTestSSHKey(t, false)

	repos := []string{
		"git@github.com:user/repo1.git",
		"git@github.com:user/repo2.git",
		"git@gitlab.com:user/repo3.git",
	}

	for _, repo := range repos {
		if err := service.SaveGitAuthConfig(repo, privateKeyPath, ""); err != nil {
			t.Fatalf("failed to save config for %s: %v", repo, err)
		}
	}

	configs, err := service.GetAllGitAuthConfigs()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(configs) != 3 {
		t.Errorf("expected 3 configs, got: %d", len(configs))
	}

	for i := 1; i < len(configs); i++ {
		if configs[i-1].RepositoryURL >= configs[i].RepositoryURL {
			t.Error("expected configs to be ordered by repository URL")
		}
	}
}

func TestDeleteGitAuthConfig_Success(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath, _, _ := generateTestSSHKey(t, false)
	repoURL := "git@github.com:user/repo.git"

	if err := service.SaveGitAuthConfig(repoURL, privateKeyPath, ""); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	err := service.DeleteGitAuthConfig(repoURL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	_, err = service.GetGitAuthConfig(repoURL)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteGitAuthConfig_NotFound(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	err := service.DeleteGitAuthConfig("git@github.com:nonexistent/repo.git")
	if err != nil {
		t.Errorf("expected no error when deleting non-existent config, got: %v", err)
	}
}

func TestGetAuthMethod_Found(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath, _, _ := generateTestSSHKey(t, false)
	repoURL := "git@github.com:user/repo.git"

	if err := service.SaveGitAuthConfig(repoURL, privateKeyPath, ""); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	auth, err := service.GetAuthMethod(repoURL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if auth == nil {
		t.Error("expected non-nil auth method")
	}
}

func TestGetAuthMethod_NotFound(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	auth, err := service.GetAuthMethod("git@github.com:nonexistent/repo.git")
	if err != nil {
		t.Errorf("expected no error when auth not found, got: %v", err)
	}

	if auth != nil {
		t.Error("expected nil auth method when not found")
	}
}

func TestGetAuthMethod_WithPassphrase(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath, _, passphrase := generateTestSSHKey(t, true)
	repoURL := "git@github.com:user/private-repo.git"

	if err := service.SaveGitAuthConfig(repoURL, privateKeyPath, passphrase); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	auth, err := service.GetAuthMethod(repoURL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if auth == nil {
		t.Error("expected non-nil auth method")
	}
}

func TestGetAuthMethod_NormalizedURL(t *testing.T) {
	dbService := createTestDatabaseForGitAuth(t)
	service := NewGitAuthService(dbService)

	privateKeyPath, _, _ := generateTestSSHKey(t, false)

	if err := service.SaveGitAuthConfig("git@github.com:user/repo.git", privateKeyPath, ""); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	auth, err := service.GetAuthMethod("ssh://git@github.com/user/repo.git")
	if err != nil {
		t.Fatalf("expected no error with normalized URL, got: %v", err)
	}

	if auth == nil {
		t.Error("expected to find auth with normalized URL")
	}
}
