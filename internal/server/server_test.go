package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arnesssr/OpenAgentsGate/internal/audit"
	"github.com/arnesssr/OpenAgentsGate/internal/decision"
	"github.com/arnesssr/OpenAgentsGate/internal/policy"
)

func TestDecideEndpointRecordsDecision(t *testing.T) {
	evaluator, err := policy.NewEvaluator(policy.Policy{
		DefaultEffect: decision.EffectDeny,
		Rules: []policy.Rule{
			{
				ID:     "external-email-needs-approval",
				Match:  policy.Match{Actions: []string{"email.*"}, Risks: []string{"external_side_effect"}},
				Effect: decision.EffectApproval,
				Reason: "external email requires approval",
			},
		},
	})
	if err != nil {
		t.Fatalf("new evaluator: %v", err)
	}
	recorder, err := audit.NewRecorder(filepath.Join(t.TempDir(), "audit.jsonl"))
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}
	srv, err := New(evaluator, recorder)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	body := strings.NewReader(`{
		"agent_id":"support-agent",
		"action":"email.send",
		"risk":"external_side_effect",
		"input":{"api_token":"secret-value"}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/actions/decide", body)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"effect":"approval_required"`)) {
		t.Fatalf("missing approval decision: %s", rr.Body.String())
	}
}
