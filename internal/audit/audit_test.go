package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
	"github.com/arnesssr/OpenAgentsGate/internal/decision"
)

func TestRecorderWritesRedactedReceipt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	recorder, err := NewRecorder(path)
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}

	_, err = recorder.Record(action.Request{
		AgentID: "agent-a",
		Action:  "email.send",
		Input: map[string]any{
			"to":        "person@example.com",
			"api_token": "secret-value",
		},
	}, decision.Decision{
		Effect:    decision.EffectApproval,
		Reason:    "approval required",
		DecidedAt: time.Unix(10, 0),
	}, "approval-1", time.Unix(11, 0))
	if err != nil {
		t.Fatalf("record: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "secret-value") {
		t.Fatalf("audit log leaked secret: %s", text)
	}
	if !strings.Contains(text, "[REDACTED]") {
		t.Fatalf("audit log did not redact secret: %s", text)
	}
}

func TestRecorderCanListAndGetReceipts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	recorder, err := NewRecorder(path)
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}
	receipt, err := recorder.Record(action.Request{
		AgentID: "agent-a",
		Action:  "github.create_pr",
	}, decision.Decision{
		Effect:    decision.EffectAllow,
		Reason:    "allowed",
		DecidedAt: time.Unix(10, 0),
	}, "", time.Unix(11, 0))
	if err != nil {
		t.Fatalf("record: %v", err)
	}

	list, err := recorder.List(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	got, err := recorder.Get(receipt.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != receipt.ID {
		t.Fatalf("id = %q, want %q", got.ID, receipt.ID)
	}
	if got.IntegrityHash == "" {
		t.Fatal("missing integrity hash")
	}
}

func TestRecorderChainsReceiptHashes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	recorder, err := NewRecorder(path)
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}
	first, err := recorder.Record(action.Request{
		AgentID: "agent-a",
		Action:  "github.create_pr",
	}, decision.Decision{Effect: decision.EffectAllow}, "", time.Unix(10, 0))
	if err != nil {
		t.Fatalf("record first: %v", err)
	}
	second, err := recorder.Record(action.Request{
		AgentID: "agent-a",
		Action:  "github.create_pr",
	}, decision.Decision{Effect: decision.EffectAllow}, "", time.Unix(11, 0))
	if err != nil {
		t.Fatalf("record second: %v", err)
	}
	if second.PreviousHash != first.IntegrityHash {
		t.Fatalf("previous hash = %q, want %q", second.PreviousHash, first.IntegrityHash)
	}
}
