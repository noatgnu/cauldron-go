package services

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	gossh "golang.org/x/crypto/ssh"
)

type GitAuthService struct {
	db *DatabaseService
}

func NewGitAuthService(db *DatabaseService) *GitAuthService {
	return &GitAuthService{db: db}
}

func (g *GitAuthService) GetAuthMethod(repoURL string) (transport.AuthMethod, error) {
	normalizedURL := g.normalizeRepoURL(repoURL)

	var config GitAuthConfig
	err := g.db.GetDB().Where("repository_url = ?", normalizedURL).First(&config).Error
	if err != nil {
		return nil, nil
	}

	return g.createSSHAuth(config.SSHKeyPath, config.SSHKeyPassphrase)
}

func (g *GitAuthService) createSSHAuth(keyPath, passphrase string) (transport.AuthMethod, error) {
	var auth transport.AuthMethod
	var err error

	if passphrase != "" {
		auth, err = ssh.NewPublicKeysFromFile("git", keyPath, passphrase)
	} else {
		auth, err = ssh.NewPublicKeysFromFile("git", keyPath, "")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load SSH key from %s: %w", keyPath, err)
	}

	return auth, nil
}

func (g *GitAuthService) normalizeRepoURL(repoURL string) string {
	if strings.HasPrefix(repoURL, "git@") {
		parts := strings.SplitN(repoURL, ":", 2)
		if len(parts) == 2 {
			return "ssh://" + parts[0] + "/" + parts[1]
		}
	}
	return repoURL
}

func (g *GitAuthService) ValidateSSHKey(keyPath, passphrase string) error {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	if passphrase != "" {
		_, err = gossh.ParsePrivateKeyWithPassphrase(data, []byte(passphrase))
	} else {
		_, err = gossh.ParsePrivateKey(data)
	}

	if err != nil {
		return fmt.Errorf("invalid SSH key: %w", err)
	}

	return nil
}

func (g *GitAuthService) SaveGitAuthConfig(repoURL, sshKeyPath, passphrase string) error {
	if err := g.ValidateSSHKey(sshKeyPath, passphrase); err != nil {
		return err
	}

	normalizedURL := g.normalizeRepoURL(repoURL)

	var existing GitAuthConfig
	result := g.db.GetDB().Where("repository_url = ?", normalizedURL).First(&existing)

	config := GitAuthConfig{
		RepositoryURL:    normalizedURL,
		SSHKeyPath:       sshKeyPath,
		SSHKeyPassphrase: passphrase,
	}

	if result.Error == nil {
		config.ID = existing.ID
		log.Printf("[GitAuth] Updating auth config for: %s", normalizedURL)
		return g.db.GetDB().Save(&config).Error
	}

	log.Printf("[GitAuth] Creating new auth config for: %s", normalizedURL)
	return g.db.GetDB().Create(&config).Error
}

func (g *GitAuthService) GetGitAuthConfig(repoURL string) (*GitAuthConfig, error) {
	normalizedURL := g.normalizeRepoURL(repoURL)

	var config GitAuthConfig
	err := g.db.GetDB().Where("repository_url = ?", normalizedURL).First(&config).Error
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (g *GitAuthService) GetAllGitAuthConfigs() ([]GitAuthConfig, error) {
	var configs []GitAuthConfig
	err := g.db.GetDB().Order("repository_url ASC").Find(&configs).Error
	return configs, err
}

func (g *GitAuthService) DeleteGitAuthConfig(repoURL string) error {
	normalizedURL := g.normalizeRepoURL(repoURL)
	log.Printf("[GitAuth] Deleting auth config for: %s", normalizedURL)
	return g.db.GetDB().Where("repository_url = ?", normalizedURL).Delete(&GitAuthConfig{}).Error
}
