package app

import (
	"testing"

	"devops-system/backend/internal/config"
)

func TestBuildCORSConfig_DefaultOrigins(t *testing.T) {
	cfg := config.Config{
		CORSAllowOrigins: "http://localhost:5173,http://127.0.0.1:5173",
	}
	c := buildCORSConfig(cfg)

	if c.AllowAllOrigins {
		t.Fatalf("expected specific origins, got allow all")
	}
	if len(c.AllowOrigins) != 2 {
		t.Fatalf("expected 2 origins, got %d (%v)", len(c.AllowOrigins), c.AllowOrigins)
	}
	if !contains(c.AllowHeaders, "Authorization") {
		t.Fatalf("cors allow headers should include Authorization")
	}
	if !contains(c.AllowHeaders, "X-Resource-Tag") || !contains(c.AllowHeaders, "X-Env") {
		t.Fatalf("cors allow headers should include ABAC headers")
	}
}

func TestBuildCORSConfig_Wildcard(t *testing.T) {
	cfg := config.Config{
		CORSAllowOrigins: "*",
	}
	c := buildCORSConfig(cfg)
	if !c.AllowAllOrigins {
		t.Fatalf("expected allow all origins for wildcard")
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
