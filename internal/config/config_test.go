package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRejectsNonLoopbackWithoutAdminToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := `{
		"listen_addr": "0.0.0.0:17671",
		"audit_log": "audit.jsonl",
		"approval_log": "approvals.jsonl",
		"revocation_log": "revocations.jsonl",
		"policy": {"default_effect": "deny"}
	}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected config error")
	}
	if !strings.Contains(err.Error(), "admin_token_env") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadAllowsLoopbackWithoutAdminToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := `{
		"listen_addr": "127.0.0.1:17671",
		"audit_log": "audit.jsonl",
		"approval_log": "approvals.jsonl",
		"revocation_log": "revocations.jsonl",
		"policy": {"default_effect": "deny"}
	}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(path); err != nil {
		t.Fatalf("load: %v", err)
	}
}
