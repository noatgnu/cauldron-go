package services

import "testing"

func TestNewServerAuthService_GeneratesTokenOnFirstRun(t *testing.T) {
	db := createTestDB(t)

	svc, err := NewServerAuthService(db)
	if err != nil {
		t.Fatalf("NewServerAuthService error: %v", err)
	}
	if svc.Token() == "" {
		t.Error("expected a non-empty generated token")
	}

	stored, err := db.GetSetting(serverAuthTokenSettingKey)
	if err != nil {
		t.Fatalf("GetSetting error: %v", err)
	}
	if stored != svc.Token() {
		t.Errorf("stored token = %q, want %q", stored, svc.Token())
	}
}

func TestNewServerAuthService_ReusesPersistedToken(t *testing.T) {
	db := createTestDB(t)
	if err := db.SaveSetting(serverAuthTokenSettingKey, "existing-token"); err != nil {
		t.Fatalf("SaveSetting: %v", err)
	}

	svc, err := NewServerAuthService(db)
	if err != nil {
		t.Fatalf("NewServerAuthService error: %v", err)
	}
	if svc.Token() != "existing-token" {
		t.Errorf("Token() = %q, want %q", svc.Token(), "existing-token")
	}
}

func TestNewServerAuthService_EnvVarOverridesPersistedToken(t *testing.T) {
	db := createTestDB(t)
	if err := db.SaveSetting(serverAuthTokenSettingKey, "db-token"); err != nil {
		t.Fatalf("SaveSetting: %v", err)
	}
	t.Setenv(ServerAuthEnvVar, "env-token")

	svc, err := NewServerAuthService(db)
	if err != nil {
		t.Fatalf("NewServerAuthService error: %v", err)
	}
	if svc.Token() != "env-token" {
		t.Errorf("Token() = %q, want %q", svc.Token(), "env-token")
	}

	stored, err := db.GetSetting(serverAuthTokenSettingKey)
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if stored != "env-token" {
		t.Errorf("expected env var token to be persisted, stored = %q", stored)
	}
}

func TestNewServerAuthService_TwoRunsProduceSameToken(t *testing.T) {
	db := createTestDB(t)

	first, err := NewServerAuthService(db)
	if err != nil {
		t.Fatalf("first NewServerAuthService error: %v", err)
	}
	second, err := NewServerAuthService(db)
	if err != nil {
		t.Fatalf("second NewServerAuthService error: %v", err)
	}
	if first.Token() != second.Token() {
		t.Errorf("expected stable token across runs, got %q then %q", first.Token(), second.Token())
	}
}

func TestServerAuthService_Verify(t *testing.T) {
	db := createTestDB(t)
	if err := db.SaveSetting(serverAuthTokenSettingKey, "correct-token"); err != nil {
		t.Fatalf("SaveSetting: %v", err)
	}
	svc, err := NewServerAuthService(db)
	if err != nil {
		t.Fatalf("NewServerAuthService error: %v", err)
	}

	if !svc.Verify("correct-token") {
		t.Error("expected the correct token to verify")
	}
	if svc.Verify("wrong-token") {
		t.Error("expected an incorrect token to fail verification")
	}
	if svc.Verify("") {
		t.Error("expected an empty candidate to fail verification")
	}
}

func TestServerAuthService_VerifyWithEmptyConfiguredTokenAlwaysFails(t *testing.T) {
	svc := &ServerAuthService{}
	if svc.Verify("anything") {
		t.Error("expected verification to fail when no token is configured")
	}
	if svc.Verify("") {
		t.Error("expected verification to fail for an empty candidate against an empty token")
	}
}
