package audit

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
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
	ID            string            `json:"id"`
	Request       action.Request    `json:"request"`
	Decision      decision.Decision `json:"decision"`
	ApprovalID    string            `json:"approval_id,omitempty"`
	PreviousHash  string            `json:"previous_hash,omitempty"`
	IntegrityHash string            `json:"integrity_hash"`
	RecordedAt    time.Time         `json:"recorded_at"`
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

func (r *Recorder) Record(req action.Request, dec decision.Decision, approvalID string, now time.Time) (Receipt, error) {
	if r == nil {
		return Receipt{}, errors.New("audit recorder is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return Receipt{}, err
	}
	previousHash, err := r.lastHashLocked()
	if err != nil {
		return Receipt{}, err
	}
	receipt := Receipt{
		ID:           newID(),
		Request:      sanitizeRequest(req),
		Decision:     dec,
		ApprovalID:   approvalID,
		PreviousHash: previousHash,
		RecordedAt:   now,
	}
	receipt.IntegrityHash = hashReceipt(receipt)

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

func (r *Recorder) List(limit int) ([]Receipt, error) {
	receipts, err := r.readAll()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit >= len(receipts) {
		return receipts, nil
	}
	return receipts[len(receipts)-limit:], nil
}

func (r *Recorder) Get(id string) (Receipt, error) {
	receipts, err := r.readAll()
	if err != nil {
		return Receipt{}, err
	}
	for _, receipt := range receipts {
		if receipt.ID == id {
			return receipt, nil
		}
	}
	return Receipt{}, os.ErrNotExist
}

func (r *Recorder) readAll() ([]Receipt, error) {
	if r == nil {
		return nil, errors.New("audit recorder is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	file, err := os.Open(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return []Receipt{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var receipts []Receipt
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var receipt Receipt
		if err := json.Unmarshal(scanner.Bytes(), &receipt); err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}
	return receipts, scanner.Err()
}

func (r *Recorder) lastHashLocked() (string, error) {
	file, err := os.Open(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer file.Close()

	var last Receipt
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var receipt Receipt
		if err := json.Unmarshal(scanner.Bytes(), &receipt); err != nil {
			return "", err
		}
		last = receipt
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return last.IntegrityHash, nil
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

func hashReceipt(receipt Receipt) string {
	receipt.IntegrityHash = ""
	data, err := json.Marshal(receipt)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
