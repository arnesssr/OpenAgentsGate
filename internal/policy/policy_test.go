package policy

import (
	"testing"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
	"github.com/arnesssr/OpenAgentsGate/internal/decision"
)

func TestEvaluatorDefaultDeny(t *testing.T) {
	evaluator, err := NewEvaluator(Policy{})
	if err != nil {
		t.Fatalf("new evaluator: %v", err)
	}

	got, err := evaluator.Decide(action.Request{
		AgentID: "agent-a",
		Action:  "email.send",
	}, time.Unix(10, 0))
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if got.Effect != decision.EffectDeny {
		t.Fatalf("effect = %q, want %q", got.Effect, decision.EffectDeny)
	}
}

func TestEvaluatorFirstMatchingRuleWins(t *testing.T) {
	evaluator, err := NewEvaluator(Policy{
		DefaultEffect: decision.EffectDeny,
		Rules: []Rule{
			{
				ID:     "external-email-needs-approval",
				Match:  Match{Actions: []string{"email.*"}, Risks: []string{"external_side_effect"}},
				Effect: decision.EffectApproval,
			},
			{
				ID:     "allow-email",
				Match:  Match{Actions: []string{"email.send"}},
				Effect: decision.EffectAllow,
			},
		},
	})
	if err != nil {
		t.Fatalf("new evaluator: %v", err)
	}

	got, err := evaluator.Decide(action.Request{
		AgentID: "agent-a",
		Action:  "email.send",
		Risk:    "external_side_effect",
	}, time.Unix(10, 0))
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if got.Effect != decision.EffectApproval {
		t.Fatalf("effect = %q, want %q", got.Effect, decision.EffectApproval)
	}
	if got.RuleID != "external-email-needs-approval" {
		t.Fatalf("rule = %q", got.RuleID)
	}
}

func TestEvaluatorValidatesRequiredFields(t *testing.T) {
	evaluator, err := NewEvaluator(Policy{})
	if err != nil {
		t.Fatalf("new evaluator: %v", err)
	}

	_, err = evaluator.Decide(action.Request{AgentID: "agent-a"}, time.Unix(10, 0))
	if err == nil {
		t.Fatal("expected validation error")
	}
}
