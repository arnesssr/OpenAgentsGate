package approval

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
	"github.com/arnesssr/OpenAgentsGate/internal/decision"
	"github.com/arnesssr/OpenAgentsGate/internal/ids"
	"github.com/arnesssr/OpenAgentsGate/internal/redact"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusDenied   Status = "denied"
)

type EventType string

const (
	EventCreated  EventType = "created"
	EventResolved EventType = "resolved"
)

type Record struct {
	ID               string            `json:"id"`
	RequestID        string            `json:"request_id"`
	Status           Status            `json:"status"`
	Request          action.Request    `json:"request"`
	Decision         decision.Decision `json:"decision"`
	Reason           string            `json:"reason,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	ResolvedAt       *time.Time        `json:"resolved_at,omitempty"`
	ResolvedBy       string            `json:"resolved_by,omitempty"`
	ResolutionReason string            `json:"resolution_reason,omitempty"`
}

type Event struct {
	EventID          string            `json:"event_id"`
	Event            EventType         `json:"event"`
	ApprovalID       string            `json:"approval_id"`
	RequestID        string            `json:"request_id"`
	Status           Status            `json:"status"`
	Request          action.Request    `json:"request,omitempty"`
	Decision         decision.Decision `json:"decision,omitempty"`
	Reason           string            `json:"reason,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	ResolvedAt       time.Time         `json:"resolved_at,omitempty"`
	ResolvedBy       string            `json:"resolved_by,omitempty"`
	ResolutionReason string            `json:"resolution_reason,omitempty"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("approval log path is required")
	}
	return &Store{path: path}, nil
}

func (s *Store) CreatePending(req action.Request, dec decision.Decision, now time.Time) (Record, error) {
	if req.RequestID == "" {
		return Record{}, errors.New("request_id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.snapshotLocked()
	if err != nil {
		return Record{}, err
	}
	for _, record := range records {
		if record.RequestID == req.RequestID && record.Status == StatusPending {
			return record, nil
		}
	}

	record := Record{
		ID:        ids.New(),
		RequestID: req.RequestID,
		Status:    StatusPending,
		Request:   sanitizeRequest(req),
		Decision:  dec,
		Reason:    dec.Reason,
		CreatedAt: now,
	}
	event := Event{
		EventID:    ids.New(),
		Event:      EventCreated,
		ApprovalID: record.ID,
		RequestID:  record.RequestID,
		Status:     record.Status,
		Request:    record.Request,
		Decision:   record.Decision,
		Reason:     record.Reason,
		CreatedAt:  now,
	}
	if err := s.appendLocked(event); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s *Store) Resolve(id string, status Status, resolvedBy, reason string, now time.Time) (Record, error) {
	if status != StatusApproved && status != StatusDenied {
		return Record{}, errors.New("status must be approved or denied")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Record{}, errors.New("approval id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.snapshotLocked()
	if err != nil {
		return Record{}, err
	}
	record, ok := records[id]
	if !ok {
		return Record{}, os.ErrNotExist
	}
	if record.Status != StatusPending {
		return Record{}, fmt.Errorf("approval %s is already %s", id, record.Status)
	}
	event := Event{
		EventID:          ids.New(),
		Event:            EventResolved,
		ApprovalID:       id,
		RequestID:        record.RequestID,
		Status:           status,
		ResolvedAt:       now,
		ResolvedBy:       strings.TrimSpace(resolvedBy),
		ResolutionReason: strings.TrimSpace(reason),
		CreatedAt:        now,
	}
	if err := s.appendLocked(event); err != nil {
		return Record{}, err
	}
	record.Status = status
	record.ResolvedAt = &now
	record.ResolvedBy = event.ResolvedBy
	record.ResolutionReason = event.ResolutionReason
	return record, nil
}

func (s *Store) List(status Status) ([]Record, error) {
	records, err := s.snapshot()
	if err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(records))
	for _, record := range records {
		if status == "" || record.Status == status {
			out = append(out, record)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (s *Store) Get(id string) (Record, error) {
	records, err := s.snapshot()
	if err != nil {
		return Record{}, err
	}
	record, ok := records[strings.TrimSpace(id)]
	if !ok {
		return Record{}, os.ErrNotExist
	}
	return record, nil
}

func (s *Store) snapshot() (map[string]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

func (s *Store) snapshotLocked() (map[string]Record, error) {
	if s == nil {
		return nil, errors.New("approval store is nil")
	}
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]Record{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	records := map[string]Record{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, err
		}
		switch event.Event {
		case EventCreated:
			records[event.ApprovalID] = Record{
				ID:        event.ApprovalID,
				RequestID: event.RequestID,
				Status:    event.Status,
				Request:   event.Request,
				Decision:  event.Decision,
				Reason:    event.Reason,
				CreatedAt: event.CreatedAt,
			}
		case EventResolved:
			record := records[event.ApprovalID]
			record.Status = event.Status
			resolvedAt := event.ResolvedAt
			record.ResolvedAt = &resolvedAt
			record.ResolvedBy = event.ResolvedBy
			record.ResolutionReason = event.ResolutionReason
			records[event.ApprovalID] = record
		default:
			return nil, fmt.Errorf("unknown approval event %q", event.Event)
		}
	}
	return records, scanner.Err()
}

func (s *Store) appendLocked(event Event) error {
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

func sanitizeRequest(req action.Request) action.Request {
	req.Input = redact.Map(req.Input)
	req.Metadata = redact.Map(req.Metadata)
	return req
}
