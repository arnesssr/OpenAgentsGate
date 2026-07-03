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
