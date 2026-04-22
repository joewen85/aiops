package handler

import (
	"testing"

	"devops-system/backend/internal/cloud"
)

func TestCloudProviderByName(t *testing.T) {
	h := &Handler{
		CloudProviders: map[string]cloud.Provider{
			"aws": cloud.NewStubProvider("aws"),
		},
	}

	provider, normalized, err := h.cloudProviderByName(" AWS ")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if normalized != "aws" {
		t.Fatalf("expected normalized provider aws, got %q", normalized)
	}
	if provider.Name() != "aws" {
		t.Fatalf("expected aws provider, got %q", provider.Name())
	}
}

func TestCloudProviderResolveAppError(t *testing.T) {
	emptyErr := cloudProviderResolveAppError(errCloudProviderEmpty)
	if emptyErr.Code != 3001 {
		t.Fatalf("expected empty provider code=3001, got %d", emptyErr.Code)
	}

	unsupportedErr := cloudProviderResolveAppError(errCloudProviderUnsupported)
	if unsupportedErr.Code != 4003 {
		t.Fatalf("expected unsupported provider code=4003, got %d", unsupportedErr.Code)
	}
}

func TestValidateCloudCredentialInput(t *testing.T) {
	if err := validateCloudCredentialInput("aws", "mock***", "secret"); err == nil {
		t.Fatalf("expected masked credential validation error")
	}
	if err := validateCloudCredentialInput("tencent", "abc", "secret"); err == nil {
		t.Fatalf("expected tencent ak prefix validation error")
	}
	if err := validateCloudCredentialInput("tencent", "AKID_TEST", "secret"); err != nil {
		t.Fatalf("expected valid credential, got %v", err)
	}
}
