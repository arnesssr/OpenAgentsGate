package risk

import (
	"fmt"
	"strings"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
)

const Unknown = "unknown"

type Rule struct {
	ID    string `json:"id"`
	Match Match  `json:"match"`
	Risk  string `json:"risk"`
}

type Match struct {
	Actions   []string `json:"actions,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Origins   []string `json:"origins,omitempty"`
}

type Classifier struct {
	rules []Rule
}

func NewClassifier(rules []Rule) (*Classifier, error) {
	if len(rules) == 0 {
		rules = DefaultRules()
	}
	for i, rule := range rules {
		if strings.TrimSpace(rule.ID) == "" {
			return nil, fmt.Errorf("risk_rules[%d].id is required", i)
		}
		if strings.TrimSpace(rule.Risk) == "" {
			return nil, fmt.Errorf("risk_rules[%d].risk is required", i)
		}
		if rule.Match.empty() {
			return nil, fmt.Errorf("risk_rules[%d].match must include at least one condition", i)
		}
	}
	return &Classifier{rules: rules}, nil
}

func DefaultRules() []Rule {
	return []Rule{
		{ID: "payment-actions", Match: Match{Actions: []string{"payment.*", "money.*"}}, Risk: "financial_side_effect"},
		{ID: "external-email", Match: Match{Actions: []string{"email.*"}, Resources: []string{"external*"}}, Risk: "external_side_effect"},
		{ID: "destructive-files", Match: Match{Actions: []string{"file.delete", "file.write"}}, Risk: "local_mutation"},
		{ID: "shell-execution", Match: Match{Actions: []string{"shell.*", "process.*"}}, Risk: "code_execution"},
		{ID: "network-calls", Match: Match{Actions: []string{"http.*", "api.*"}}, Risk: "external_network"},
		{ID: "agent-delegation", Match: Match{Actions: []string{"agent.*", "a2a.*"}}, Risk: "agent_delegation"},
	}
}

func (c *Classifier) Classify(req action.Request) string {
	if c == nil {
		return Unknown
	}
	for _, rule := range c.rules {
		if rule.Match.matches(req) {
			return rule.Risk
		}
	}
	return Unknown
}

func (m Match) empty() bool {
	return len(m.Actions) == 0 && len(m.Resources) == 0 && len(m.Origins) == 0
}

func (m Match) matches(req action.Request) bool {
	return matchAny(m.Actions, req.Action) &&
		matchAny(m.Resources, req.Resource) &&
		matchAny(m.Origins, req.Origin)
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
