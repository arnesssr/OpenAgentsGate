package gateway

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
	"github.com/arnesssr/OpenAgentsGate/internal/approval"
	"github.com/arnesssr/OpenAgentsGate/internal/audit"
	"github.com/arnesssr/OpenAgentsGate/internal/decision"
	"github.com/arnesssr/OpenAgentsGate/internal/policy"
	"github.com/arnesssr/OpenAgentsGate/internal/redact"
	"github.com/arnesssr/OpenAgentsGate/internal/revocation"
	"github.com/arnesssr/OpenAgentsGate/internal/risk"
)

const revocationRuleID = "revocation"

type Service struct {
	evaluator   *policy.Evaluator
	classifier  *risk.Classifier
	audit       *audit.Recorder
	approvals   *approval.Store
	revocations *revocation.Store
}

type Stores struct {
	Audit       *audit.Recorder
	Approvals   *approval.Store
	Revocations *revocation.Store
}

type Result struct {
	Request    action.Request     `json:"request"`
	Decision   decision.Decision  `json:"decision"`
	ReceiptID  string             `json:"receipt_id"`
	ApprovalID string             `json:"approval_id,omitempty"`
	Revocation *revocation.Active `json:"revocation,omitempty"`
}

type ReplayResult struct {
	Receipt         audit.Receipt      `json:"receipt"`
	CurrentRequest  action.Request     `json:"current_request"`
	CurrentDecision decision.Decision  `json:"current_decision"`
	Revocation      *revocation.Active `json:"revocation,omitempty"`
}

func New(evaluator *policy.Evaluator, classifier *risk.Classifier, stores Stores) (*Service, error) {
	if evaluator == nil {
		return nil, errors.New("policy evaluator is required")
	}
	if classifier == nil {
		return nil, errors.New("risk classifier is required")
	}
	if stores.Audit == nil {
		return nil, errors.New("audit recorder is required")
	}
	if stores.Approvals == nil {
		return nil, errors.New("approval store is required")
	}
	if stores.Revocations == nil {
		return nil, errors.New("revocation store is required")
	}
	return &Service{
		evaluator:   evaluator,
		classifier:  classifier,
		audit:       stores.Audit,
		approvals:   stores.Approvals,
		revocations: stores.Revocations,
	}, nil
}

func (s *Service) Decide(req action.Request, now time.Time) (Result, error) {
	req, dec, active, err := s.evaluate(req, now)
	if err != nil {
		return Result{}, err
	}

	var approvalID string
	if dec.Effect == decision.EffectApproval {
		record, err := s.approvals.CreatePending(req, dec, now)
		if err != nil {
			return Result{}, fmt.Errorf("approval: %w", err)
		}
		approvalID = record.ID
	}

	receipt, err := s.audit.Record(req, dec, approvalID, now)
	if err != nil {
		return Result{}, fmt.Errorf("audit: %w", err)
	}
	return Result{
		Request:    sanitizeRequest(req),
		Decision:   dec,
		ReceiptID:  receipt.ID,
		ApprovalID: approvalID,
		Revocation: active,
	}, nil
}

func (s *Service) Replay(receiptID string, now time.Time) (ReplayResult, error) {
	receipt, err := s.audit.Get(receiptID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ReplayResult{}, err
		}
		return ReplayResult{}, fmt.Errorf("audit: %w", err)
	}
	req, dec, active, err := s.evaluate(receipt.Request, now)
	if err != nil {
		return ReplayResult{}, err
	}
	return ReplayResult{
		Receipt:         receipt,
		CurrentRequest:  req,
		CurrentDecision: dec,
		Revocation:      active,
	}, nil
}

func (s *Service) ListAudit(limit int) ([]audit.Receipt, error) {
	return s.audit.List(limit)
}

func (s *Service) GetAudit(id string) (audit.Receipt, error) {
	return s.audit.Get(id)
}

func (s *Service) VerifyAudit() (audit.VerificationResult, error) {
	return s.audit.Verify()
}

func (s *Service) ListApprovals(status approval.Status) ([]approval.Record, error) {
	return s.approvals.List(status)
}

func (s *Service) GetApproval(id string) (approval.Record, error) {
	return s.approvals.Get(id)
}

func (s *Service) ResolveApproval(id string, status approval.Status, by, reason string, now time.Time) (approval.Record, error) {
	return s.approvals.Resolve(id, status, by, reason, now)
}

func (s *Service) ListRevocations() ([]revocation.Active, error) {
	return s.revocations.Active()
}

func (s *Service) Revoke(targetType revocation.TargetType, targetID, reason, by string, now time.Time) (revocation.Active, error) {
	return s.revocations.Revoke(targetType, targetID, reason, by, now)
}

func (s *Service) Restore(targetType revocation.TargetType, targetID, reason, by string, now time.Time) error {
	return s.revocations.Restore(targetType, targetID, reason, by, now)
}

func (s *Service) evaluate(req action.Request, now time.Time) (action.Request, decision.Decision, *revocation.Active, error) {
	if s == nil {
		return action.Request{}, decision.Decision{}, nil, errors.New("gateway service is nil")
	}
	req = req.WithDefaults(now)
	if req.Risk == "" {
		req.Risk = s.classifier.Classify(req)
	}
	if err := req.Validate(); err != nil {
		return action.Request{}, decision.Decision{}, nil, err
	}

	active, revoked, err := s.revocations.Match(req)
	if err != nil {
		return action.Request{}, decision.Decision{}, nil, fmt.Errorf("revocation: %w", err)
	}
	if revoked {
		dec := decision.Decision{
			Effect:    decision.EffectDeny,
			RuleID:    revocationRuleID,
			Reason:    fmt.Sprintf("blocked by revocation %s:%s", active.TargetType, active.TargetID),
			DecidedAt: now,
		}
		return req, dec, &active, nil
	}

	dec, err := s.evaluator.Decide(req, now)
	if err != nil {
		return action.Request{}, decision.Decision{}, nil, err
	}
	return req, dec, nil, nil
}

func sanitizeRequest(req action.Request) action.Request {
	req.Input = redact.Map(req.Input)
	req.Metadata = redact.Map(req.Metadata)
	return req
}
