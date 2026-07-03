package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/arnesssr/OpenAgentsGate/internal/policy"
	"github.com/arnesssr/OpenAgentsGate/internal/risk"
)

type Config struct {
	ListenAddr    string        `json:"listen_addr"`
	AdminTokenEnv string        `json:"admin_token_env,omitempty"`
	AuditLog      string        `json:"audit_log"`
	ApprovalLog   string        `json:"approval_log"`
	RevocationLog string        `json:"revocation_log"`
	RiskRules     []risk.Rule   `json:"risk_rules,omitempty"`
	Policy        policy.Policy `json:"policy"`
}

func Load(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		return Config{}, errors.New("config path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:17671"
	}
	if cfg.AuditLog == "" {
		cfg.AuditLog = "openagentsgate.audit.jsonl"
	}
	if cfg.ApprovalLog == "" {
		cfg.ApprovalLog = "openagentsgate.approvals.jsonl"
	}
	if cfg.RevocationLog == "" {
		cfg.RevocationLog = "openagentsgate.revocations.jsonl"
	}
	if err := cfg.validateListenSecurity(); err != nil {
		return Config{}, err
	}
	if _, err := policy.NewEvaluator(cfg.Policy); err != nil {
		return Config{}, fmt.Errorf("policy: %w", err)
	}
	if _, err := risk.NewClassifier(cfg.RiskRules); err != nil {
		return Config{}, fmt.Errorf("risk: %w", err)
	}
	return cfg, nil
}

func (c Config) AdminToken() (string, error) {
	if strings.TrimSpace(c.AdminTokenEnv) == "" {
		return "", nil
	}
	token := os.Getenv(c.AdminTokenEnv)
	if token == "" {
		return "", fmt.Errorf("admin token env %s is not set", c.AdminTokenEnv)
	}
	return token, nil
}

func (c Config) validateListenSecurity() error {
	if strings.TrimSpace(c.AdminTokenEnv) != "" {
		return nil
	}
	host, _, err := net.SplitHostPort(c.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen_addr: %w", err)
	}
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return nil
	}
	return errors.New("admin_token_env is required when listen_addr is not loopback")
}
