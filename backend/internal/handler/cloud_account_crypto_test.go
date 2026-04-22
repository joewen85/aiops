package handler

import (
	"testing"

	"devops-system/backend/internal/config"
	"devops-system/backend/internal/models"
)

func TestCloudCredentialEncryptDecryptRoundTrip(t *testing.T) {
	h := &Handler{
		Config: config.Config{
			JWTSecret: "unit-test-jwt-secret",
		},
	}
	encrypted, err := h.encryptCloudCredential("AKID_TEST_123456")
	if err != nil {
		t.Fatalf("encrypt cloud credential failed: %v", err)
	}
	if encrypted == "" || encrypted == "AKID_TEST_123456" {
		t.Fatalf("expected encrypted text, got %q", encrypted)
	}
	decrypted, err := h.decryptCloudCredential(encrypted)
	if err != nil {
		t.Fatalf("decrypt cloud credential failed: %v", err)
	}
	if decrypted != "AKID_TEST_123456" {
		t.Fatalf("expected decrypted credential match, got %q", decrypted)
	}
}

func TestCloudCredentialsDecryptLegacyPlainText(t *testing.T) {
	h := &Handler{
		Config: config.Config{
			JWTSecret: "unit-test-jwt-secret",
		},
	}
	cred, err := h.cloudCredentials(models.CloudAccount{
		AccessKey: "legacy-ak",
		SecretKey: "legacy-sk",
		Region:    "ap-guangzhou",
	})
	if err != nil {
		t.Fatalf("cloud credentials from legacy plain text failed: %v", err)
	}
	if cred.AccessKey != "legacy-ak" || cred.SecretKey != "legacy-sk" {
		t.Fatalf("unexpected legacy credentials: %+v", cred)
	}
}
