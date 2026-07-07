package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arnesssr/OpenAgentsGate/internal/approval"
	"github.com/arnesssr/OpenAgentsGate/internal/audit"
	"github.com/arnesssr/OpenAgentsGate/internal/decision"
	"github.com/arnesssr/OpenAgentsGate/internal/gateway"
	"github.com/arnesssr/OpenAgentsGate/internal/policy"
	"github.com/arnesssr/OpenAgentsGate/internal/revocation"
	"github.com/arnesssr/OpenAgentsGate/internal/risk"
)

func TestDecideEndpointRecordsDecision(t *testing.T) {
	srv := newTestServer(t, "")

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

func TestAdminTokenProtectsAPIRoutesButNotHealth(t *testing.T) {
	srv := newTestServer(t, "secret-token")
	handler := srv.Handler()

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d", health.Code)
	}

	unauthorized := httptest.NewRecorder()
	body := strings.NewReader(`{"agent_id":"codex","action":"github.create_pr"}`)
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, "/v1/actions/decide", body))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, body = %s", unauthorized.Code, unauthorized.Body.String())
	}

	authorized := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/actions/decide", strings.NewReader(`{"agent_id":"codex","action":"github.create_pr"}`))
	req.Header.Set("Authorization", "Bearer secret-token")
	handler.ServeHTTP(authorized, req)
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized status = %d, body = %s", authorized.Code, authorized.Body.String())
	}
}

func TestDecideEndpointRejectsUnknownFields(t *testing.T) {
	srv := newTestServer(t, "")
	body := strings.NewReader(`{
		"agent_id":"support-agent",
		"action":"email.send",
		"unexpected":"field"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/actions/decide", body)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"error":"invalid JSON request"`)) {
		t.Fatalf("unexpected error body: %s", rr.Body.String())
	}
}

func newTestServer(t *testing.T, adminToken string) *Server {
	t.Helper()
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
	dir := t.TempDir()
	recorder, err := audit.NewRecorder(filepath.Join(dir, "audit.jsonl"))
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}
	approvals, err := approval.NewStore(filepath.Join(dir, "approvals.jsonl"))
	if err != nil {
		t.Fatalf("new approvals: %v", err)
	}
	revocations, err := revocation.NewStore(filepath.Join(dir, "revocations.jsonl"))
	if err != nil {
		t.Fatalf("new revocations: %v", err)
	}
	classifier, err := risk.NewClassifier(nil)
	if err != nil {
		t.Fatalf("new classifier: %v", err)
	}
	service, err := gateway.New(evaluator, classifier, gateway.Stores{
		Audit:       recorder,
		Approvals:   approvals,
		Revocations: revocations,
	})
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	srv, err := New(service, adminToken)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return srv
}
