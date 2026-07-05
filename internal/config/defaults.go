package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/arnesssr/OpenAgentsGate/internal/decision"
	"github.com/arnesssr/OpenAgentsGate/internal/policy"
	"github.com/arnesssr/OpenAgentsGate/internal/risk"
)

func Default(stateDir string) Config {
	stateDir = strings.TrimSpace(stateDir)
	if stateDir == "" {
		stateDir = "."
	}
	return Config{
		ListenAddr:    "127.0.0.1:17671",
		AuditLog:      filepath.Join(stateDir, "audit.jsonl"),
		ApprovalLog:   filepath.Join(stateDir, "approvals.jsonl"),
		RevocationLog: filepath.Join(stateDir, "revocations.jsonl"),
		RiskRules: []risk.Rule{
			{
				ID: "external-email",
				Match: risk.Match{
					Actions:   []string{"email.*"},
					Resources: []string{"external*"},
				},
				Risk: "external_side_effect",
			},
			{
				ID:    "shell-execution",
				Match: risk.Match{Actions: []string{"shell.*", "process.*"}},
				Risk:  "code_execution",
			},
			{
				ID:    "payment-actions",
				Match: risk.Match{Actions: []string{"payment.*", "money.*"}},
				Risk:  "financial_side_effect",
			},
			{
				ID:    "agent-delegation",
				Match: risk.Match{Actions: []string{"agent.*", "a2a.*"}},
				Risk:  "agent_delegation",
			},
			{
				ID:    "git-publish",
				Match: risk.Match{Actions: []string{"git.push", "git.tag"}},
				Risk:  "external_side_effect",
			},
		},
		Policy: policy.Policy{
			DefaultEffect: decision.EffectDeny,
			Rules: []policy.Rule{
				{
					ID:          "allow-wrapper-launch",
					Description: "Launching a wrapped local agent process is allowed and audited.",
					Match:       policy.Match{Actions: []string{"agent.wrap"}},
					Effect:      decision.EffectAllow,
					Reason:      "wrapped process launch is allowed",
				},
				{
					ID:          "external-email-needs-approval",
					Description: "External email can create real-world side effects.",
					Match:       policy.Match{Actions: []string{"email.*"}, Risks: []string{"external_side_effect"}},
					Effect:      decision.EffectApproval,
					Reason:      "external email requires approval",
				},
				{
					ID:          "money-needs-approval",
					Description: "Financial actions must never run without explicit approval.",
					Match:       policy.Match{Risks: []string{"financial_side_effect"}},
					Effect:      decision.EffectApproval,
					Reason:      "financial side effects require approval",
				},
				{
					ID:          "shell-is-dry-run",
					Description: "Code execution starts in dry-run mode.",
					Match:       policy.Match{Risks: []string{"code_execution"}},
					Effect:      decision.EffectDryRun,
					Reason:      "shell execution requires dry-run review",
				},
				{
					ID:          "agent-delegation-needs-approval",
					Description: "Delegating to another agent changes authority boundaries.",
					Match:       policy.Match{Risks: []string{"agent_delegation"}},
					Effect:      decision.EffectApproval,
					Reason:      "agent delegation requires approval",
				},
				{
					ID:          "deny-file-delete",
					Description: "Destructive local file operations are blocked by default.",
					Match:       policy.Match{Actions: []string{"file.delete"}},
					Effect:      decision.EffectDeny,
					Reason:      "file deletion is blocked",
				},
				{
					ID:          "allow-readonly-git",
					Description: "Read-only git inspection is allowed.",
					Match:       policy.Match{Actions: []string{"git.status", "git.diff", "git.log", "git.show", "git.branch"}},
					Effect:      decision.EffectAllow,
					Reason:      "read-only git inspection is allowed",
				},
				{
					ID:          "git-publish-needs-approval",
					Description: "Publishing git refs crosses a repo boundary.",
					Match:       policy.Match{Actions: []string{"git.push", "git.tag"}},
					Effect:      decision.EffectApproval,
					Reason:      "publishing git refs requires approval",
				},
				{
					ID:          "allow-github-prs",
					Description: "Creating pull requests is allowed in the starter policy.",
					Match:       policy.Match{Actions: []string{"github.create_pr"}},
					Effect:      decision.EffectAllow,
					Reason:      "pull request creation is allowed",
				},
			},
		},
	}
}

func WriteDefault(path, stateDir string, overwrite bool) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("config path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(Default(stateDir), "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	flags := os.O_WRONLY | os.O_CREATE
	if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(path, flags, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return err
	}
	return file.Chmod(0o600)
}
