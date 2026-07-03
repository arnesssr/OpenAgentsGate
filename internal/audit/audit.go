package audit

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
	"github.com/arnesssr/OpenAgentsGate/internal/decision"
	"github.com/arnesssr/OpenAgentsGate/internal/redact"
)

type Receipt struct {
	ID         string            `json:"id"`
	Request    action.Request    `json:"request"`
	Decision   decision.Decision `json:"decision"`
	RecordedAt time.Time         `json:"recorded_at"`
}

type Recorder struct {
	path string
	mu   sync.Mutex
}

func NewRecorder(path string) (*Recorder, error) {
	if path == "" {
		return nil, errors.New("audit log path is required")
	}
	return &Recorder{path: path}, nil
}

func (r *Recorder) Record(req action.Request, dec decision.Decision, now time.Time) (Receipt, error) {
	if r == nil {
		return Receipt{}, errors.New("audit recorder is nil")
	}
	receipt := Receipt{
		ID:         newID(),
		Request:    sanitizeRequest(req),
		Decision:   dec,
		RecordedAt: now,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return Receipt{}, err
	}
	file, err := os.OpenFile(r.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return Receipt{}, err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(receipt); err != nil {
		return Receipt{}, err
	}
	return receipt, nil
}

func sanitizeRequest(req action.Request) action.Request {
	req.Input = redact.Map(req.Input)
	req.Metadata = redact.Map(req.Metadata)
	return req
}

func newID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buf[:])
}
