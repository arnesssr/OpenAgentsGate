package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
	"github.com/arnesssr/OpenAgentsGate/internal/approval"
	"github.com/arnesssr/OpenAgentsGate/internal/audit"
	"github.com/arnesssr/OpenAgentsGate/internal/config"
	"github.com/arnesssr/OpenAgentsGate/internal/gateway"
	"github.com/arnesssr/OpenAgentsGate/internal/policy"
	"github.com/arnesssr/OpenAgentsGate/internal/revocation"
	"github.com/arnesssr/OpenAgentsGate/internal/risk"
	"github.com/arnesssr/OpenAgentsGate/internal/server"
)

const version = "0.1.0-dev"

type runtime struct {
	cfg     config.Config
	service *gateway.Service
}

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "run":
		run(os.Args[2:])
	case "decide":
		decide(os.Args[2:])
	case "approvals":
		approvals(os.Args[2:])
	case "revocations":
		revocations(os.Args[2:])
	case "audit":
		auditCommands(os.Args[2:])
	case "version":
		fmt.Println(version)
	default:
		usage()
		os.Exit(2)
	}
}

func run(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", "examples/openagentsgate.json", "path to config file")
	_ = fs.Parse(args)

	rt := mustRuntime(*configPath)
	adminToken, err := rt.cfg.AdminToken()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	srv, err := server.New(rt.service, adminToken)
	if err != nil {
		log.Fatalf("server: %v", err)
	}

	httpServer := srv.HTTPServer(rt.cfg.ListenAddr)
	log.Printf("openagentsgate listening on http://%s", rt.cfg.ListenAddr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}

func decide(args []string) {
	fs := flag.NewFlagSet("decide", flag.ExitOnError)
	configPath := fs.String("config", "examples/openagentsgate.json", "path to config file")
	_ = fs.Parse(args)

	rt := mustRuntime(*configPath)
	var req action.Request
	decoder := json.NewDecoder(os.Stdin)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		log.Fatalf("request: invalid JSON")
	}
	result, err := rt.service.Decide(req, time.Now().UTC())
	if err != nil {
		log.Fatalf("decide: %v", err)
	}
	printJSON(result)
}

func approvals(args []string) {
	if len(args) < 1 {
		approvalUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("approvals list", flag.ExitOnError)
		configPath := fs.String("config", "examples/openagentsgate.json", "path to config file")
		status := fs.String("status", "", "filter by approval status")
		_ = fs.Parse(args[1:])
		rt := mustRuntime(*configPath)
		records, err := rt.service.ListApprovals(approval.Status(*status))
		if err != nil {
			log.Fatalf("approvals: %v", err)
		}
		printJSON(map[string]any{"approvals": records})
	case "resolve":
		fs := flag.NewFlagSet("approvals resolve", flag.ExitOnError)
		configPath := fs.String("config", "examples/openagentsgate.json", "path to config file")
		id := fs.String("id", "", "approval id")
		status := fs.String("status", "", "approved or denied")
		by := fs.String("by", "", "resolver identity")
		reason := fs.String("reason", "", "resolution reason")
		_ = fs.Parse(args[1:])
		rt := mustRuntime(*configPath)
		record, err := rt.service.ResolveApproval(*id, approval.Status(*status), *by, *reason, time.Now().UTC())
		if err != nil {
			log.Fatalf("approvals: %v", err)
		}
		printJSON(record)
	default:
		approvalUsage()
		os.Exit(2)
	}
}

func revocations(args []string) {
	if len(args) < 1 {
		revocationUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("revocations list", flag.ExitOnError)
		configPath := fs.String("config", "examples/openagentsgate.json", "path to config file")
		_ = fs.Parse(args[1:])
		rt := mustRuntime(*configPath)
		items, err := rt.service.ListRevocations()
		if err != nil {
			log.Fatalf("revocations: %v", err)
		}
		printJSON(map[string]any{"revocations": items})
	case "add":
		fs := flag.NewFlagSet("revocations add", flag.ExitOnError)
		configPath := fs.String("config", "examples/openagentsgate.json", "path to config file")
		targetType := fs.String("type", "", "target type")
		targetID := fs.String("id", "", "target id")
		reason := fs.String("reason", "", "revocation reason")
		by := fs.String("by", "", "creator identity")
		_ = fs.Parse(args[1:])
		rt := mustRuntime(*configPath)
		item, err := rt.service.Revoke(revocation.TargetType(*targetType), *targetID, *reason, *by, time.Now().UTC())
		if err != nil {
			log.Fatalf("revocations: %v", err)
		}
		printJSON(item)
	case "remove":
		fs := flag.NewFlagSet("revocations remove", flag.ExitOnError)
		configPath := fs.String("config", "examples/openagentsgate.json", "path to config file")
		targetType := fs.String("type", "", "target type")
		targetID := fs.String("id", "", "target id")
		reason := fs.String("reason", "", "restore reason")
		by := fs.String("by", "", "resolver identity")
		_ = fs.Parse(args[1:])
		rt := mustRuntime(*configPath)
		if err := rt.service.Restore(revocation.TargetType(*targetType), *targetID, *reason, *by, time.Now().UTC()); err != nil {
			log.Fatalf("revocations: %v", err)
		}
		printJSON(map[string]string{"status": "restored"})
	default:
		revocationUsage()
		os.Exit(2)
	}
}

func auditCommands(args []string) {
	if len(args) < 1 {
		auditUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("audit list", flag.ExitOnError)
		configPath := fs.String("config", "examples/openagentsgate.json", "path to config file")
		limit := fs.Int("limit", 100, "maximum receipts to return")
		_ = fs.Parse(args[1:])
		rt := mustRuntime(*configPath)
		receipts, err := rt.service.ListAudit(*limit)
		if err != nil {
			log.Fatalf("audit: %v", err)
		}
		printJSON(map[string]any{"receipts": receipts})
	case "get":
		fs := flag.NewFlagSet("audit get", flag.ExitOnError)
		configPath := fs.String("config", "examples/openagentsgate.json", "path to config file")
		id := fs.String("id", "", "receipt id")
		_ = fs.Parse(args[1:])
		rt := mustRuntime(*configPath)
		receipt, err := rt.service.GetAudit(*id)
		if err != nil {
			log.Fatalf("audit: %v", err)
		}
		printJSON(receipt)
	case "replay":
		fs := flag.NewFlagSet("audit replay", flag.ExitOnError)
		configPath := fs.String("config", "examples/openagentsgate.json", "path to config file")
		id := fs.String("id", "", "receipt id")
		_ = fs.Parse(args[1:])
		rt := mustRuntime(*configPath)
		result, err := rt.service.Replay(*id, time.Now().UTC())
		if err != nil {
			log.Fatalf("audit: %v", err)
		}
		printJSON(result)
	default:
		auditUsage()
		os.Exit(2)
	}
}

func mustRuntime(path string) runtime {
	cfg, err := config.Load(path)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	evaluator, err := policy.NewEvaluator(cfg.Policy)
	if err != nil {
		log.Fatalf("policy: %v", err)
	}
	classifier, err := risk.NewClassifier(cfg.RiskRules)
	if err != nil {
		log.Fatalf("risk: %v", err)
	}
	recorder, err := audit.NewRecorder(cfg.AuditLog)
	if err != nil {
		log.Fatalf("audit: %v", err)
	}
	approvalStore, err := approval.NewStore(cfg.ApprovalLog)
	if err != nil {
		log.Fatalf("approvals: %v", err)
	}
	revocationStore, err := revocation.NewStore(cfg.RevocationLog)
	if err != nil {
		log.Fatalf("revocations: %v", err)
	}
	service, err := gateway.New(evaluator, classifier, gateway.Stores{
		Audit:       recorder,
		Approvals:   approvalStore,
		Revocations: revocationStore,
	})
	if err != nil {
		log.Fatalf("gateway: %v", err)
	}
	return runtime{cfg: cfg, service: service}
}

func printJSON(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		log.Fatalf("json: %v", err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: openagentsgate <run|decide|approvals|revocations|audit|version> [flags]")
}

func approvalUsage() {
	fmt.Fprintln(os.Stderr, "usage: openagentsgate approvals <list|resolve> [flags]")
}

func revocationUsage() {
	fmt.Fprintln(os.Stderr, "usage: openagentsgate revocations <list|add|remove> [flags]")
}

func auditUsage() {
	fmt.Fprintln(os.Stderr, "usage: openagentsgate audit <list|get|replay> [flags]")
}
