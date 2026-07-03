package risk

import (
	"testing"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
)

func TestClassifierUsesConfiguredRules(t *testing.T) {
	classifier, err := NewClassifier([]Rule{
		{
			ID:    "external-email",
			Match: Match{Actions: []string{"email.*"}, Resources: []string{"external*"}},
			Risk:  "external_side_effect",
		},
	})
	if err != nil {
		t.Fatalf("new classifier: %v", err)
	}

	got := classifier.Classify(action.Request{
		Action:   "email.send",
		Resource: "external_email",
	})
	if got != "external_side_effect" {
		t.Fatalf("risk = %q", got)
	}
}
