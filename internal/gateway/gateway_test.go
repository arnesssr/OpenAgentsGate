package gateway

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
	"github.com/arnesssr/OpenAgentsGate/internal/approval"
	"github.com/arnesssr/OpenAgentsGate/internal/audit"
	"github.com/arnesssr/OpenAgentsGate/internal/decision"
	"github.com/arnesssr/OpenAgentsGate/internal/policy"
	"github.com/arnesssr/OpenAgentsGate/internal/revocation"
	"github.com/arnesssr/OpenAgentsGate/internal/risk"
)

func TestServiceCreatesApprovalAndReceipt(t *testing.T) {
	service := newTestService(t)
	result, err := service.Decide(action.Request{
		RequestID: "req-1",
		AgentID:   "support-agent",
		Action:    "email.send",
		Resource:  "external_email",
		Input:     map[string]any{"api_token": "secret-value"},
	}, time.Unix(10, 0))
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if result.Decision.Effect != decision.EffectApproval {
		t.Fatalf("effect = %q", result.Decision.Effect)
	}
	if result.ApprovalID == "" {
		t.Fatal("missing approval id")
	}
	if result.ReceiptID == "" {
		t.Fatal("missing receipt id")
	}
	if result.Request.Input["api_token"] != "[REDACTED]" {
		t.Fatalf("secret was not redacted in response: %#v", result.Request.Input)
	}
}

func TestServiceRevocationOverridesPolicyAllow(t *testing.T) {
	service := newTestService(t)
	_, err := service.Revoke(revocation.TargetAgent, "support-agent", "compromised", "admin", time.Unix(9, 0))
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}

	result, err := service.Decide(action.Request{
		RequestID: "req-2",
		AgentID:   "support-agent",
		Action:    "github.create_pr",
	}, time.Unix(10, 0))
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if result.Decision.Effect != decision.EffectDeny {
		t.Fatalf("effect = %q", result.Decision.Effect)
	}
	if result.Decision.RuleID != revocationRuleID {
		t.Fatalf("rule = %q", result.Decision.RuleID)
	}
}

func TestServiceReplayUsesCurrentPolicyState(t *testing.T) {
	service := newTestService(t)
	result, err := service.Decide(action.Request{
		RequestID: "req-3",
		AgentID:   "support-agent",
		Action:    "github.create_pr",
	}, time.Unix(10, 0))
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	_, err = service.Revoke(revocation.TargetAction, "github.create_pr", "freeze code changes", "admin", time.Unix(11, 0))
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}

	replay, err := service.Replay(result.ReceiptID, time.Unix(12, 0))
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if replay.Receipt.Decision.Effect != decision.EffectAllow {
		t.Fatalf("original effect = %q", replay.Receipt.Decision.Effect)
	}
	if replay.CurrentDecision.Effect != decision.EffectDeny {
		t.Fatalf("current effect = %q", replay.CurrentDecision.Effect)
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	evaluator, err := policy.NewEvaluator(policy.Policy{
		DefaultEffect: decision.EffectDeny,
		Rules: []policy.Rule{
			{
				ID:     "external-email-needs-approval",
				Match:  policy.Match{Actions: []string{"email.*"}, Risks: []string{"external_side_effect"}},
				Effect: decision.EffectApproval,
			},
			{
				ID:     "allow-prs",
				Match:  policy.Match{Actions: []string{"github.create_pr"}},
				Effect: decision.EffectAllow,
			},
		},
	})
	if err != nil {
		t.Fatalf("new evaluator: %v", err)
	}
	classifier, err := risk.NewClassifier(nil)
	if err != nil {
		t.Fatalf("new classifier: %v", err)
	}
	dir := t.TempDir()
	recorder, err := audit.NewRecorder(filepath.Join(dir, "audit.jsonl"))
	if err != nil {
		t.Fatalf("new audit: %v", err)
	}
	approvals, err := approval.NewStore(filepath.Join(dir, "approvals.jsonl"))
	if err != nil {
		t.Fatalf("new approvals: %v", err)
	}
	revocations, err := revocation.NewStore(filepath.Join(dir, "revocations.jsonl"))
	if err != nil {
		t.Fatalf("new revocations: %v", err)
	}
	service, err := New(evaluator, classifier, Stores{
		Audit:       recorder,
		Approvals:   approvals,
		Revocations: revocations,
	})
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	return service
}
