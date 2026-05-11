package handler

import (
	"strings"
	"testing"
)

func TestBuildNodeJoinInventoryQuotesSensitiveValues(t *testing.T) {
	req := kubernetesNodeRegisterRequest{
		Hostname:    "worker-01",
		InternalIP:  "10.0.0.21",
		SSHUser:     "root",
		SSHPassword: `pa ss'"word`,
		SSHPort:     22,
	}
	inventory, err := buildNodeJoinInventory(req, "a2ViZWFkbSBqb2lu")
	if err != nil {
		t.Fatalf("buildNodeJoinInventory returned error: %v", err)
	}
	if !strings.Contains(inventory, `ansible_host="10.0.0.21"`) {
		t.Fatalf("expected ansible_host quoted, got: %s", inventory)
	}
	if !strings.Contains(inventory, `ansible_user="root"`) {
		t.Fatalf("expected ansible_user quoted, got: %s", inventory)
	}
	if !strings.Contains(inventory, `ansible_password="pa ss'\"word"`) {
		t.Fatalf("expected ansible_password quoted, got: %s", inventory)
	}
}

func TestBuildNodeJoinInventoryRejectsInvalidHost(t *testing.T) {
	req := kubernetesNodeRegisterRequest{
		Hostname: "worker 01",
		SSHUser:  "root",
		SSHPort:  22,
	}
	if _, err := buildNodeJoinInventory(req, "a2ViZWFkbSBqb2lu"); err == nil {
		t.Fatalf("expected invalid host error")
	}
}

func TestValidateNodeRegisterTaskInput(t *testing.T) {
	req := kubernetesNodeRegisterRequest{
		Hostname: "worker-01",
		SSHUser:  "root",
		SSHPort:  22,
	}
	if err := validateNodeRegisterTaskInput(req, "kubeadm join 10.0.0.10:6443 --token abcdef.0123456789abcdef"); err != nil {
		t.Fatalf("expected valid input, got error: %v", err)
	}
	if err := validateNodeRegisterTaskInput(req, "echo hello"); err == nil {
		t.Fatalf("expected invalid joinCommand error")
	}
}

func TestRedactNodeRegisterTaskOutput(t *testing.T) {
	raw := "kubeadm join 10.0.0.10:6443 --token abcdef.0123456789abcdef --discovery-token-ca-cert-hash sha256:123"
	redacted := redactNodeRegisterTaskOutput(raw)
	if strings.Contains(redacted, "abcdef.0123456789abcdef") {
		t.Fatalf("expected token redacted, got: %s", redacted)
	}
	if !strings.Contains(redacted, "--token ***REDACTED***") {
		t.Fatalf("expected token placeholder, got: %s", redacted)
	}
}
