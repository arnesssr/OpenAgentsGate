package main

import (
	"flag"
	"testing"

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
