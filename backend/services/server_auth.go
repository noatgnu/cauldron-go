package services

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"os"
)

const serverAuthTokenSettingKey = "serverAuthToken"

// ServerAuthEnvVar overrides the persisted server auth token when set.
const ServerAuthEnvVar = "CAULDRON_SERVER_TOKEN"

// ServerAuthService holds the single shared token that gates Cauldron's server-mode HTTP surface.
type ServerAuthService struct {
	token string
}

// NewServerAuthService resolves the active token from ServerAuthEnvVar if set, else a persisted
// DB value, generating and persisting a new random one on first run.
func NewServerAuthService(db *DatabaseService) (*ServerAuthService, error) {
	token := os.Getenv(ServerAuthEnvVar)

	if token == "" {
		stored, err := db.GetSetting(serverAuthTokenSettingKey)
		if err != nil {
			return nil, fmt.Errorf("failed to read server auth token: %w", err)
		}
		token = stored
	}

	if token == "" {
		generated, err := randomToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate server auth token: %w", err)
		}
		token = generated
	}

	if err := db.SaveSetting(serverAuthTokenSettingKey, token); err != nil {
		return nil, fmt.Errorf("failed to persist server auth token: %w", err)
	}

	return &ServerAuthService{token: token}, nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Token returns the active shared token, for startup logging only.
func (s *ServerAuthService) Token() string {
	return s.token
}

// Verify reports whether candidate matches the configured token, in constant time.
func (s *ServerAuthService) Verify(candidate string) bool {
	if candidate == "" || s.token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(s.token)) == 1
}
