package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
	"github.com/arnesssr/OpenAgentsGate/internal/audit"
	"github.com/arnesssr/OpenAgentsGate/internal/policy"
)

const maxRequestBytes = 1 << 20

type Server struct {
	evaluator *policy.Evaluator
	recorder  *audit.Recorder
	clock     func() time.Time
}

func New(evaluator *policy.Evaluator, recorder *audit.Recorder) (*Server, error) {
	if evaluator == nil {
		return nil, errors.New("policy evaluator is required")
	}
	if recorder == nil {
		return nil, errors.New("audit recorder is required")
	}
	return &Server{
		evaluator: evaluator,
		recorder:  recorder,
		clock:     func() time.Time { return time.Now().UTC() },
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /v1/actions/decide", s.decide)
	return mux
}

func (s *Server) HTTPServer(addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) decide(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req action.Request
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid action request")
		return
	}

	now := s.clock()
	req = req.WithDefaults(now)
	dec, err := s.evaluator.Decide(req, now)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	receipt, err := s.recorder.Record(req, dec, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record audit receipt")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"decision": dec,
		"receipt":  receipt.ID,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		fmt.Fprintf(w, `{"error":"failed to encode response"}`)
	}
}
