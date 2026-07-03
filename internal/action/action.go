package action

import (
	"errors"
	"strings"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/ids"
)

// Request is the protocol-neutral action shape evaluated by OpenAgentsGate.
type Request struct {
	RequestID       string         `json:"request_id,omitempty"`
	AgentID         string         `json:"agent_id"`
	AgentInstanceID string         `json:"agent_instance_id,omitempty"`
	UserID          string         `json:"user_id,omitempty"`
	SessionID       string         `json:"session_id,omitempty"`
	Action          string         `json:"action"`
	Resource        string         `json:"resource,omitempty"`
	Origin          string         `json:"origin,omitempty"`
	Risk            string         `json:"risk,omitempty"`
	Input           map[string]any `json:"input,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	RequestedAt     time.Time      `json:"requested_at,omitempty"`
}

func (r Request) Validate() error {
	if strings.TrimSpace(r.AgentID) == "" {
		return errors.New("agent_id is required")
	}
	if strings.TrimSpace(r.Action) == "" {
		return errors.New("action is required")
	}
	return nil
}

func (r Request) WithDefaults(now time.Time) Request {
	r.RequestID = strings.TrimSpace(r.RequestID)
	r.AgentID = strings.TrimSpace(r.AgentID)
	r.AgentInstanceID = strings.TrimSpace(r.AgentInstanceID)
	r.UserID = strings.TrimSpace(r.UserID)
	r.SessionID = strings.TrimSpace(r.SessionID)
	r.Action = strings.TrimSpace(r.Action)
	r.Resource = strings.TrimSpace(r.Resource)
	r.Origin = strings.TrimSpace(r.Origin)
	r.Risk = strings.TrimSpace(r.Risk)
	if r.RequestedAt.IsZero() {
		r.RequestedAt = now
	}
	if r.RequestID == "" {
		r.RequestID = ids.New()
	}
	return r
}
