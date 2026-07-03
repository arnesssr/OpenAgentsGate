package decision

import "time"

type Effect string

const (
	EffectAllow    Effect = "allow"
	EffectDeny     Effect = "deny"
	EffectApproval Effect = "approval_required"
	EffectDryRun   Effect = "dry_run"
)

type Decision struct {
	Effect    Effect    `json:"effect"`
	RuleID    string    `json:"rule_id,omitempty"`
	Reason    string    `json:"reason"`
	DecidedAt time.Time `json:"decided_at"`
}

func (e Effect) Valid() bool {
	switch e {
	case EffectAllow, EffectDeny, EffectApproval, EffectDryRun:
		return true
	default:
		return false
	}
}
