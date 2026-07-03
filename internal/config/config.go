package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/arnesssr/OpenAgentsGate/internal/policy"
)

type Config struct {
	ListenAddr string        `json:"listen_addr"`
	AuditLog   string        `json:"audit_log"`
	Policy     policy.Policy `json:"policy"`
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
	if _, err := policy.NewEvaluator(cfg.Policy); err != nil {
		return Config{}, fmt.Errorf("policy: %w", err)
	}
	return cfg, nil
}
