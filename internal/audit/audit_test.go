package audit

import (
	"encoding/json"
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

func TestRecorderVerifyValidChain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	recorder, err := NewRecorder(path)
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}
	if _, err := recorder.Record(action.Request{
		AgentID: "agent-a",
		Action:  "github.create_pr",
	}, decision.Decision{Effect: decision.EffectAllow}, "", time.Unix(10, 0)); err != nil {
		t.Fatalf("record first: %v", err)
	}
	second, err := recorder.Record(action.Request{
		AgentID: "agent-a",
		Action:  "git.status",
	}, decision.Decision{Effect: decision.EffectAllow}, "", time.Unix(11, 0))
	if err != nil {
		t.Fatalf("record second: %v", err)
	}

	result, err := recorder.Verify()
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !result.Valid {
		t.Fatalf("valid = false, failures = %#v", result.Failures)
	}
	if result.ReceiptCount != 2 {
		t.Fatalf("receipt count = %d, want 2", result.ReceiptCount)
	}
	if result.LastHash != second.IntegrityHash {
		t.Fatalf("last hash = %q, want %q", result.LastHash, second.IntegrityHash)
	}
}

func TestRecorderVerifyDetectsTamperedReceipt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	recorder, err := NewRecorder(path)
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}
	if _, err := recorder.Record(action.Request{
		AgentID: "agent-a",
		Action:  "github.create_pr",
	}, decision.Decision{Effect: decision.EffectAllow}, "", time.Unix(10, 0)); err != nil {
		t.Fatalf("record: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	tampered := strings.Replace(string(data), "github.create_pr", "github.delete_repo", 1)
	if err := os.WriteFile(path, []byte(tampered), 0o600); err != nil {
		t.Fatalf("write tampered audit: %v", err)
	}

	result, err := recorder.Verify()
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if result.Valid {
		t.Fatal("valid = true, want tamper failure")
	}
	if len(result.Failures) != 1 || result.Failures[0].Reason != "integrity hash mismatch" {
		t.Fatalf("failures = %#v", result.Failures)
	}
}

func TestRecorderVerifyDetectsBrokenPreviousHash(t *testing.T) {
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
		Action:  "git.status",
	}, decision.Decision{Effect: decision.EffectAllow}, "", time.Unix(11, 0))
	if err != nil {
		t.Fatalf("record second: %v", err)
	}
	second.PreviousHash = "wrong"
	second.IntegrityHash = hashReceipt(second)
	writeReceipts(t, path, first, second)

	result, err := recorder.Verify()
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if result.Valid {
		t.Fatal("valid = true, want broken chain failure")
	}
	if len(result.Failures) != 1 || result.Failures[0].Reason != "previous hash mismatch" {
		t.Fatalf("failures = %#v", result.Failures)
	}
}

func writeReceipts(t *testing.T, path string, receipts ...Receipt) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open audit: %v", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	for _, receipt := range receipts {
		if err := encoder.Encode(receipt); err != nil {
			t.Fatalf("encode receipt: %v", err)
		}
	}
}
