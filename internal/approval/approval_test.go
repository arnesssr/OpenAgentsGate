package approval

import (
	"encoding/json"
	"os"
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

func TestStoreAllowsNewPendingAfterResolution(t *testing.T) {
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
	if _, err := store.Resolve(first.ID, StatusDenied, "admin", "no", time.Unix(12, 0)); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	second, err := store.CreatePending(req, dec, time.Unix(13, 0))
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if second.ID == first.ID {
		t.Fatalf("second approval reused resolved approval id %q", second.ID)
	}
}

func TestStoreRejectsInvalidResolution(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "approvals.jsonl"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	req := action.Request{RequestID: "req-1", AgentID: "agent-a", Action: "email.send"}
	dec := decision.Decision{Effect: decision.EffectApproval, DecidedAt: time.Unix(10, 0)}
	record, err := store.CreatePending(req, dec, time.Unix(11, 0))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := store.Resolve(record.ID, StatusPending, "admin", "bad", time.Unix(12, 0)); err == nil {
		t.Fatal("expected invalid status error")
	}
	if _, err := store.Resolve("", StatusDenied, "admin", "bad", time.Unix(12, 0)); err == nil {
		t.Fatal("expected missing id error")
	}
	if _, err := store.Resolve("missing", StatusDenied, "admin", "bad", time.Unix(12, 0)); err == nil {
		t.Fatal("expected missing approval error")
	}
	if _, err := store.Resolve(record.ID, StatusApproved, "admin", "ok", time.Unix(13, 0)); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if _, err := store.Resolve(record.ID, StatusDenied, "admin", "again", time.Unix(14, 0)); err == nil {
		t.Fatal("expected already resolved error")
	}
}

func TestStoreListFiltersAndSortsApprovals(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "approvals.jsonl"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	dec := decision.Decision{Effect: decision.EffectApproval}
	late, err := store.CreatePending(action.Request{RequestID: "req-late", AgentID: "agent-a", Action: "email.send"}, dec, time.Unix(20, 0))
	if err != nil {
		t.Fatalf("late create: %v", err)
	}
	early, err := store.CreatePending(action.Request{RequestID: "req-early", AgentID: "agent-a", Action: "email.send"}, dec, time.Unix(10, 0))
	if err != nil {
		t.Fatalf("early create: %v", err)
	}
	if _, err := store.Resolve(late.ID, StatusApproved, "admin", "ok", time.Unix(30, 0)); err != nil {
		t.Fatalf("resolve late: %v", err)
	}

	pending, err := store.List(StatusPending)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 1 || pending[0].ID != early.ID {
		t.Fatalf("pending = %#v, want only early", pending)
	}
	all, err := store.List("")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 || all[0].ID != early.ID || all[1].ID != late.ID {
		t.Fatalf("all = %#v, want created-at sorted early then late", all)
	}
}

func TestStoreRejectsResolutionBeforeCreation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "approvals.jsonl")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	if err := json.NewEncoder(file).Encode(Event{
		Event:      EventResolved,
		ApprovalID: "approval-1",
		RequestID:  "req-1",
		Status:     StatusApproved,
		ResolvedAt: time.Unix(10, 0),
	}); err != nil {
		t.Fatalf("encode event: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close log: %v", err)
	}

	if _, err := store.List(""); err == nil {
		t.Fatal("expected corrupt lifecycle error")
	}
}
