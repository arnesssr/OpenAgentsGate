package policy

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
	"github.com/arnesssr/OpenAgentsGate/internal/decision"
)

type Policy struct {
	DefaultEffect decision.Effect `json:"default_effect"`
	Rules         []Rule          `json:"rules"`
}

type Rule struct {
	ID          string          `json:"id"`
	Description string          `json:"description,omitempty"`
	Match       Match           `json:"match"`
	Effect      decision.Effect `json:"effect"`
	Reason      string          `json:"reason,omitempty"`
}

type Match struct {
	Actions   []string `json:"actions,omitempty"`
	Agents    []string `json:"agents,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Origins   []string `json:"origins,omitempty"`
	Risks     []string `json:"risks,omitempty"`
}

type Evaluator struct {
	policy Policy
}

func NewEvaluator(p Policy) (*Evaluator, error) {
	if p.DefaultEffect == "" {
		p.DefaultEffect = decision.EffectDeny
	}
	if !p.DefaultEffect.Valid() {
		return nil, fmt.Errorf("invalid default_effect %q", p.DefaultEffect)
	}
	for i, rule := range p.Rules {
		if strings.TrimSpace(rule.ID) == "" {
			return nil, fmt.Errorf("rules[%d].id is required", i)
		}
		if !rule.Effect.Valid() {
			return nil, fmt.Errorf("rules[%d].effect is invalid", i)
		}
		if rule.Match.empty() {
			return nil, fmt.Errorf("rules[%d].match must include at least one condition", i)
		}
	}
	return &Evaluator{policy: p}, nil
}

func (e *Evaluator) Decide(req action.Request, now time.Time) (decision.Decision, error) {
	if e == nil {
		return decision.Decision{}, errors.New("policy evaluator is nil")
	}
	if err := req.Validate(); err != nil {
		return decision.Decision{}, err
	}
	for _, rule := range e.policy.Rules {
		if rule.Match.matches(req) {
			return decision.Decision{
				Effect:    rule.Effect,
				RuleID:    rule.ID,
				Reason:    reasonOrDefault(rule.Reason, rule.Effect),
				DecidedAt: now,
			}, nil
		}
	}
	return decision.Decision{
		Effect:    e.policy.DefaultEffect,
		Reason:    reasonOrDefault("", e.policy.DefaultEffect),
		DecidedAt: now,
	}, nil
}

func (m Match) empty() bool {
	return len(m.Actions) == 0 &&
		len(m.Agents) == 0 &&
		len(m.Resources) == 0 &&
		len(m.Origins) == 0 &&
		len(m.Risks) == 0
}

func (m Match) matches(req action.Request) bool {
	return matchAny(m.Actions, req.Action) &&
		matchAny(m.Agents, req.AgentID) &&
		matchAny(m.Resources, req.Resource) &&
		matchAny(m.Origins, req.Origin) &&
		matchAny(m.Risks, req.Risk)
}

func matchAny(patterns []string, value string) bool {
	if len(patterns) == 0 {
		return true
	}
	value = strings.TrimSpace(value)
	for _, pattern := range patterns {
		if matchPattern(pattern, value) {
			return true
		}
	}
	return false
}

func matchPattern(pattern string, value string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(value, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == value
}

func reasonOrDefault(reason string, effect decision.Effect) string {
	if strings.TrimSpace(reason) != "" {
		return reason
	}
	switch effect {
	case decision.EffectAllow:
		return "allowed by policy"
	case decision.EffectApproval:
		return "approval required by policy"
	case decision.EffectDryRun:
		return "dry-run required by policy"
	default:
		return "denied by policy"
	}
}
