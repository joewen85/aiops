package app

import (
	"testing"

	"devops-system/backend/internal/config"
)

func TestBuildCORSConfigIncludesABACSignatureHeaders(t *testing.T) {
	corsConfig := buildCORSConfig(config.Config{})

	if !containsValue(corsConfig.AllowHeaders, "X-ABAC-Timestamp") {
		t.Fatalf("missing CORS allow header X-ABAC-Timestamp")
	}
	if !containsValue(corsConfig.AllowHeaders, "X-ABAC-Signature") {
		t.Fatalf("missing CORS allow header X-ABAC-Signature")
	}
}

func containsValue(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
