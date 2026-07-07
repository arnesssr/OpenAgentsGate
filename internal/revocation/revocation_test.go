package revocation

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
)

func TestStoreMatchesAndRestoresRevocation(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "revocations.jsonl"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	_, err = store.Revoke(TargetAgent, "agent-a", "compromised", "admin", time.Unix(10, 0))
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}

	active, ok, err := store.Match(action.Request{AgentID: "agent-a", Action: "email.send"})
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if !ok {
		t.Fatal("expected revocation match")
	}
	if active.TargetID != "agent-a" {
		t.Fatalf("target = %q", active.TargetID)
	}

	if err := store.Restore(TargetAgent, "agent-a", "fixed", "admin", time.Unix(11, 0)); err != nil {
		t.Fatalf("restore: %v", err)
	}
	_, ok, err = store.Match(action.Request{AgentID: "agent-a", Action: "email.send"})
	if err != nil {
		t.Fatalf("match after restore: %v", err)
	}
	if ok {
		t.Fatal("revocation still active after restore")
	}
}

func TestStoreMatchesEveryTargetType(t *testing.T) {
	req := action.Request{
		AgentID:         "agent-a",
		AgentInstanceID: "instance-a",
		UserID:          "user-a",
		SessionID:       "session-a",
		Action:          "git.push",
	}
	cases := []struct {
		targetType TargetType
		targetID   string
	}{
		{TargetAll, ""},
		{TargetAgent, req.AgentID},
		{TargetAgentInstance, req.AgentInstanceID},
		{TargetUser, req.UserID},
		{TargetSession, req.SessionID},
		{TargetAction, req.Action},
	}
	for i, tc := range cases {
		store, err := NewStore(filepath.Join(t.TempDir(), "revocations.jsonl"))
		if err != nil {
			t.Fatalf("new store %s: %v", tc.targetType, err)
		}
		if _, err := store.Revoke(tc.targetType, tc.targetID, "stop", "admin", time.Unix(int64(i+1), 0)); err != nil {
			t.Fatalf("revoke %s: %v", tc.targetType, err)
		}
		active, ok, err := store.Match(req)
		if err != nil {
			t.Fatalf("match %s: %v", tc.targetType, err)
		}
		if !ok {
			t.Fatalf("expected match for %s", tc.targetType)
		}
		if active.TargetType != tc.targetType {
			t.Fatalf("active target = %s, want %s", active.TargetType, tc.targetType)
		}
	}
}

func TestStoreRejectsInvalidTargets(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "revocations.jsonl"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if _, err := store.Revoke("bad", "id", "bad", "admin", time.Unix(10, 0)); err == nil {
		t.Fatal("expected invalid target type error")
	}
	if _, err := store.Revoke(TargetAgent, " ", "bad", "admin", time.Unix(10, 0)); err == nil {
		t.Fatal("expected missing target id error")
	}
	if err := store.Restore(TargetSession, "", "bad", "admin", time.Unix(10, 0)); err == nil {
		t.Fatal("expected missing target id restore error")
	}
}

func TestStoreLatestEventWinsForSameTarget(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "revocations.jsonl"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if _, err := store.Revoke(TargetAgent, "agent-a", "stop", "admin", time.Unix(10, 0)); err != nil {
		t.Fatalf("revoke first: %v", err)
	}
	if err := store.Restore(TargetAgent, "agent-a", "resume", "admin", time.Unix(11, 0)); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if _, err := store.Revoke(TargetAgent, "agent-a", "stop again", "admin", time.Unix(12, 0)); err != nil {
		t.Fatalf("revoke second: %v", err)
	}

	active, ok, err := store.Match(action.Request{AgentID: "agent-a", Action: "email.send"})
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if !ok {
		t.Fatal("expected active revocation")
	}
	if active.Reason != "stop again" {
		t.Fatalf("reason = %q, want latest reason", active.Reason)
	}
}
