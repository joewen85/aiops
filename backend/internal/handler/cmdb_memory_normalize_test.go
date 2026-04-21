package handler

import "testing"

func TestNormalizeCloudResourceMetadataMemoryMB(t *testing.T) {
	metadata := map[string]interface{}{
		"memoryMB": 16384,
	}
	got := normalizeCloudResourceMetadata(metadata)
	if got["memory"] != "16G" {
		t.Fatalf("expected memory=16G got=%v", got["memory"])
	}
}

func TestNormalizeCloudResourceMetadataMemoryWithoutUnit(t *testing.T) {
	metadata := map[string]interface{}{
		"memory": 16,
	}
	got := normalizeCloudResourceMetadata(metadata)
	if got["memory"] != "16G" {
		t.Fatalf("expected memory=16G got=%v", got["memory"])
	}
}

func TestNormalizeCloudResourceMetadataMemoryKeepUnit(t *testing.T) {
	metadata := map[string]interface{}{
		"memory": "16Gi",
	}
	got := normalizeCloudResourceMetadata(metadata)
	if got["memory"] != "16Gi" {
		t.Fatalf("expected memory=16Gi got=%v", got["memory"])
	}
}
