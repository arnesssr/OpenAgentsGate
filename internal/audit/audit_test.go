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
	}, time.Unix(11, 0))
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
