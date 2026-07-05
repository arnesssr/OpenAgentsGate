package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/arnesssr/OpenAgentsGate/internal/config"
	"github.com/arnesssr/OpenAgentsGate/internal/decision"
)

func TestRequestFlagsPopulateActionRequest(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	spec := requestFlags(fs)
	err := fs.Parse([]string{
		"-action", "github.create_pr",
		"-agent", "codex",
		"-resource", "repo",
		"-input-json", `{"path":"README.md"}`,
	})
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	req := spec.actionRequest()
	if req.Action != "github.create_pr" {
		t.Fatalf("action = %q", req.Action)
	}
	if req.AgentID != "codex" {
		t.Fatalf("agent = %q", req.AgentID)
	}
	if req.Input["path"] != "README.md" {
		t.Fatalf("input = %#v", req.Input)
	}
}

func TestExitCodeForEffect(t *testing.T) {
	cases := map[decision.Effect]int{
		decision.EffectAllow:    0,
		decision.EffectDryRun:   exitDryRun,
		decision.EffectApproval: exitApproval,
		decision.EffectDeny:     exitDeny,
	}
	for effect, want := range cases {
		if got := exitCodeForEffect(effect); got != want {
			t.Fatalf("exit code for %s = %d, want %d", effect, got, want)
		}
	}
}

func TestResolveConfigPathCreatesUserDefault(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, "state"))
	chdir(t, tmp)

	loc, err := resolveConfigPath("", true)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if loc.Source != "user" || !loc.Created {
		t.Fatalf("location = %#v", loc)
	}
	if !fileExists(loc.Path) {
		t.Fatalf("config was not created at %s", loc.Path)
	}
	cfg, err := config.Load(loc.Path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	wantAudit := filepath.Join(tmp, "state", "openagentsgate", "audit.jsonl")
	if cfg.AuditLog != wantAudit {
		t.Fatalf("audit log = %q, want %q", cfg.AuditLog, wantAudit)
	}
}

func TestLocateConfigFindsProjectConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, "state"))
	projectConfig := filepath.Join(tmp, ".openagentsgate", "config.json")
	if err := config.WriteDefault(projectConfig, filepath.Join(tmp, ".openagentsgate", "state"), false); err != nil {
		t.Fatalf("write config: %v", err)
	}
	subdir := filepath.Join(tmp, "child")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	chdir(t, subdir)

	loc, err := resolveConfigPath("", false)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if loc.Source != "project" || loc.Path != projectConfig {
		t.Fatalf("location = %#v", loc)
	}
}

func TestRedactArgv(t *testing.T) {
	got := redactArgv([]string{"tool", "--token", "abc", "--api-key=def", "--safe", "value"})
	want := []string{"tool", "--token", "[REDACTED]", "--api-key=[REDACTED]", "--safe", "value"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("argv[%d] = %q, want %q; full=%#v", i, got[i], want[i], got)
		}
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
}
