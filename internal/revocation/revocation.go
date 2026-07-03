package revocation

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
	"github.com/arnesssr/OpenAgentsGate/internal/ids"
)

type TargetType string

const (
	TargetAll           TargetType = "all"
	TargetAgent         TargetType = "agent"
	TargetAgentInstance TargetType = "agent_instance"
	TargetUser          TargetType = "user"
	TargetSession       TargetType = "session"
	TargetAction        TargetType = "action"
)

type EventType string

const (
	EventRevoke  EventType = "revoke"
	EventRestore EventType = "restore"
)

type Event struct {
	ID         string     `json:"id"`
	Event      EventType  `json:"event"`
	TargetType TargetType `json:"target_type"`
	TargetID   string     `json:"target_id"`
	Reason     string     `json:"reason,omitempty"`
	CreatedBy  string     `json:"created_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Active struct {
	EventID    string     `json:"event_id"`
	TargetType TargetType `json:"target_type"`
	TargetID   string     `json:"target_id"`
	Reason     string     `json:"reason,omitempty"`
	CreatedBy  string     `json:"created_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("revocation log path is required")
	}
	return &Store{path: path}, nil
}

func (s *Store) Revoke(targetType TargetType, targetID, reason, createdBy string, now time.Time) (Active, error) {
	if err := validateTarget(targetType, targetID); err != nil {
		return Active{}, err
	}
	event := Event{
		ID:         ids.New(),
		Event:      EventRevoke,
		TargetType: targetType,
		TargetID:   normalizeTargetID(targetType, targetID),
		Reason:     strings.TrimSpace(reason),
		CreatedBy:  strings.TrimSpace(createdBy),
		CreatedAt:  now,
	}
	if err := s.append(event); err != nil {
		return Active{}, err
	}
	return activeFromEvent(event), nil
}

func (s *Store) Restore(targetType TargetType, targetID, reason, createdBy string, now time.Time) error {
	if err := validateTarget(targetType, targetID); err != nil {
		return err
	}
	return s.append(Event{
		ID:         ids.New(),
		Event:      EventRestore,
		TargetType: targetType,
		TargetID:   normalizeTargetID(targetType, targetID),
		Reason:     strings.TrimSpace(reason),
		CreatedBy:  strings.TrimSpace(createdBy),
		CreatedAt:  now,
	})
}

func (s *Store) Active() ([]Active, error) {
	snapshot, err := s.snapshot()
	if err != nil {
		return nil, err
	}
	out := make([]Active, 0, len(snapshot))
	for _, item := range snapshot {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (s *Store) Match(req action.Request) (Active, bool, error) {
	active, err := s.Active()
	if err != nil {
		return Active{}, false, err
	}
	for _, item := range active {
		if item.matches(req) {
			return item, true, nil
		}
	}
	return Active{}, false, nil
}

func (s *Store) append(event Event) error {
	if s == nil {
		return errors.New("revocation store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(event)
}

func (s *Store) snapshot() (map[string]Active, error) {
	if s == nil {
		return nil, errors.New("revocation store is nil")
	}
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]Active{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	out := map[string]Active{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, err
		}
		key := event.TargetType.key(event.TargetID)
		switch event.Event {
		case EventRevoke:
			out[key] = activeFromEvent(event)
		case EventRestore:
			delete(out, key)
		default:
			return nil, fmt.Errorf("unknown revocation event %q", event.Event)
		}
	}
	return out, scanner.Err()
}

func validateTarget(targetType TargetType, targetID string) error {
	switch targetType {
	case TargetAll, TargetAgent, TargetAgentInstance, TargetUser, TargetSession, TargetAction:
	default:
		return fmt.Errorf("invalid target_type %q", targetType)
	}
	if targetType == TargetAll {
		return nil
	}
	if strings.TrimSpace(targetID) == "" {
		return errors.New("target_id is required")
	}
	return nil
}

func normalizeTargetID(targetType TargetType, targetID string) string {
	if targetType == TargetAll {
		return "*"
	}
	return strings.TrimSpace(targetID)
}

func activeFromEvent(event Event) Active {
	return Active{
		EventID:    event.ID,
		TargetType: event.TargetType,
		TargetID:   event.TargetID,
		Reason:     event.Reason,
		CreatedBy:  event.CreatedBy,
		CreatedAt:  event.CreatedAt,
	}
}

func (t TargetType) key(id string) string {
	return string(t) + ":" + normalizeTargetID(t, id)
}

func (a Active) matches(req action.Request) bool {
	switch a.TargetType {
	case TargetAll:
		return true
	case TargetAgent:
		return a.TargetID == req.AgentID
	case TargetAgentInstance:
		return a.TargetID == req.AgentInstanceID
	case TargetUser:
		return a.TargetID == req.UserID
	case TargetSession:
		return a.TargetID == req.SessionID
	case TargetAction:
		return a.TargetID == req.Action
	default:
		return false
	}
}
