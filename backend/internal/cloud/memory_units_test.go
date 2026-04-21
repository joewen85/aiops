package cloud

import "testing"

func TestFormatMemoryGB(t *testing.T) {
	if got := formatMemoryGB(16); got != "16G" {
		t.Fatalf("expected 16G got=%s", got)
	}
	if got := formatMemoryGB(0); got != "" {
		t.Fatalf("expected empty memory for zero got=%s", got)
	}
}

func TestFormatMemoryMBToGB(t *testing.T) {
	if got := formatMemoryMBToGB(16384); got != "16G" {
		t.Fatalf("expected 16G got=%s", got)
	}
	if got := formatMemoryMBToGB(1536); got != "1.5G" {
		t.Fatalf("expected 1.5G got=%s", got)
	}
}
