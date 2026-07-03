package approval

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
	"github.com/arnesssr/OpenAgentsGate/internal/decision"
)

func TestStoreCreatesAndResolvesApproval(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "approvals.jsonl"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	req := action.Request{
		RequestID: "req-1",
		AgentID:   "support-agent",
		Action:    "email.send",
		Input:     map[string]any{"api_token": "secret-value"},
	}
	dec := decision.Decision{
		Effect:    decision.EffectApproval,
		Reason:    "approval required",
		DecidedAt: time.Unix(10, 0),
	}

	record, err := store.CreatePending(req, dec, time.Unix(11, 0))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if record.Status != StatusPending {
		t.Fatalf("status = %q, want pending", record.Status)
	}
	if record.Request.Input["api_token"] != "[REDACTED]" {
		t.Fatalf("secret was not redacted: %#v", record.Request.Input)
	}

	resolved, err := store.Resolve(record.ID, StatusApproved, "admin", "looks safe", time.Unix(12, 0))
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Status != StatusApproved {
		t.Fatalf("status = %q, want approved", resolved.Status)
	}
}

func TestStoreDeduplicatesPendingApprovalByRequestID(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "approvals.jsonl"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	req := action.Request{RequestID: "req-1", AgentID: "agent-a", Action: "email.send"}
	dec := decision.Decision{Effect: decision.EffectApproval, DecidedAt: time.Unix(10, 0)}

	first, err := store.CreatePending(req, dec, time.Unix(11, 0))
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	second, err := store.CreatePending(req, dec, time.Unix(12, 0))
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("ids differ: %q != %q", first.ID, second.ID)
	}
}
