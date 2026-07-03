package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
	"github.com/arnesssr/OpenAgentsGate/internal/approval"
	"github.com/arnesssr/OpenAgentsGate/internal/gateway"
	"github.com/arnesssr/OpenAgentsGate/internal/revocation"
)

const maxRequestBytes = 1 << 20

type Server struct {
	gateway    *gateway.Service
	adminToken string
	clock      func() time.Time
}

func New(service *gateway.Service, adminToken string) (*Server, error) {
	if service == nil {
		return nil, errors.New("gateway service is required")
	}
	return &Server{
		gateway:    service,
		adminToken: adminToken,
		clock:      func() time.Time { return time.Now().UTC() },
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /v1/actions/decide", s.decide)
	mux.HandleFunc("GET /v1/approvals", s.listApprovals)
	mux.HandleFunc("GET /v1/approvals/{id}", s.getApproval)
	mux.HandleFunc("POST /v1/approvals/{id}/resolve", s.resolveApproval)
	mux.HandleFunc("GET /v1/revocations", s.listRevocations)
	mux.HandleFunc("POST /v1/revocations", s.createRevocation)
	mux.HandleFunc("DELETE /v1/revocations/{target_type}/{target_id}", s.restoreRevocation)
	mux.HandleFunc("GET /v1/audit", s.listAudit)
	mux.HandleFunc("GET /v1/audit/{id}", s.getAudit)
	mux.HandleFunc("POST /v1/audit/{id}/replay", s.replayAudit)
	return s.withAuth(mux)
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

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || s.adminToken == "" {
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+s.adminToken {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) decide(w http.ResponseWriter, r *http.Request) {
	var req action.Request
	if !readJSON(w, r, &req) {
		return
	}
	result, err := s.gateway.Decide(req, s.clock())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listApprovals(w http.ResponseWriter, r *http.Request) {
	status := approval.Status(strings.TrimSpace(r.URL.Query().Get("status")))
	records, err := s.gateway.ListApprovals(status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list approvals")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"approvals": records})
}

func (s *Server) getApproval(w http.ResponseWriter, r *http.Request) {
	record, err := s.gateway.GetApproval(r.PathValue("id"))
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "approval not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get approval")
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) resolveApproval(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status approval.Status `json:"status"`
		By     string          `json:"by"`
		Reason string          `json:"reason"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	record, err := s.gateway.ResolveApproval(r.PathValue("id"), req.Status, req.By, req.Reason, s.clock())
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "approval not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) listRevocations(w http.ResponseWriter, _ *http.Request) {
	items, err := s.gateway.ListRevocations()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list revocations")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revocations": items})
}

func (s *Server) createRevocation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetType revocation.TargetType `json:"target_type"`
		TargetID   string                `json:"target_id"`
		Reason     string                `json:"reason"`
		By         string                `json:"by"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	item, err := s.gateway.Revoke(req.TargetType, req.TargetID, req.Reason, req.By, s.clock())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) restoreRevocation(w http.ResponseWriter, r *http.Request) {
	targetType := revocation.TargetType(r.PathValue("target_type"))
	targetID := r.PathValue("target_id")
	if err := s.gateway.Restore(targetType, targetID, "", "", s.clock()); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}

func (s *Server) listAudit(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"))
	receipts, err := s.gateway.ListAudit(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list audit receipts")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"receipts": receipts})
}

func (s *Server) getAudit(w http.ResponseWriter, r *http.Request) {
	receipt, err := s.gateway.GetAudit(r.PathValue("id"))
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "audit receipt not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get audit receipt")
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

func (s *Server) replayAudit(w http.ResponseWriter, r *http.Request) {
	result, err := s.gateway.Replay(r.PathValue("id"), s.clock())
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "audit receipt not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func readJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return false
	}
	return true
}

func parseLimit(value string) int {
	if strings.TrimSpace(value) == "" {
		return 100
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 0 {
		return 100
	}
	if limit > 1000 {
		return 1000
	}
	return limit
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
