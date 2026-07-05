package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/action"
	"github.com/arnesssr/OpenAgentsGate/internal/approval"
	"github.com/arnesssr/OpenAgentsGate/internal/audit"
	"github.com/arnesssr/OpenAgentsGate/internal/buildinfo"
	"github.com/arnesssr/OpenAgentsGate/internal/config"
	"github.com/arnesssr/OpenAgentsGate/internal/decision"
	"github.com/arnesssr/OpenAgentsGate/internal/gateway"
	"github.com/arnesssr/OpenAgentsGate/internal/policy"
	"github.com/arnesssr/OpenAgentsGate/internal/revocation"
	"github.com/arnesssr/OpenAgentsGate/internal/risk"
	"github.com/arnesssr/OpenAgentsGate/internal/server"
)

const (
	exitUsage    = 2
	exitDryRun   = 10
	exitApproval = 20
	exitDeny     = 30
)

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
	case "check":
		check(os.Args[2:])
	case "tool":
		tool(os.Args[2:])
	case "decide":
		decide(os.Args[2:])
	case "approvals":
		approvals(os.Args[2:])
	case "revocations":
		revocations(os.Args[2:])
	case "audit":
		auditCommands(os.Args[2:])
	case "version":
		printJSON(map[string]string{
			"version": buildinfo.Version,
			"commit":  buildinfo.Commit,
			"date":    buildinfo.Date,
		})
	default:
		usage()
		os.Exit(2)
	}
}

func check(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	spec := requestFlags(fs)
	configPath := fs.String("config", "examples/openagentsgate.json", "path to config file")
	requestPath := fs.String("request", "", "path to full action request JSON, or - for stdin")
	strictExit := fs.Bool("strict-exit", true, "exit non-zero unless decision is allow")
	_ = fs.Parse(args)

	rt := mustRuntime(*configPath)
	req := mustActionRequest(*requestPath, spec)
	result, err := rt.service.Decide(req, time.Now().UTC())
	if err != nil {
		log.Fatalf("check: %v", err)
	}
	printJSON(result)
	if *strictExit {
		os.Exit(exitCodeForEffect(result.Decision.Effect))
	}
}

func tool(args []string) {
	if len(args) < 1 {
		toolUsage()
		os.Exit(exitUsage)
	}
	switch args[0] {
	case "shell":
		toolCommand(args[1:], "shell.run", "openagentsgate.tool.shell", func(argv []string) *exec.Cmd {
			return exec.Command(argv[0], argv[1:]...)
		})
	case "git":
		toolGit(args[1:])
	default:
		toolUsage()
		os.Exit(exitUsage)
	}
}

func toolGit(args []string) {
	fs := flag.NewFlagSet("tool git", flag.ExitOnError)
	spec := requestFlags(fs)
	configPath := fs.String("config", "examples/openagentsgate.json", "path to config file")
	_ = fs.Parse(args)
	argv := fs.Args()
	if len(argv) == 0 {
		log.Fatalf("tool git: missing git arguments")
	}
	actionName := "git." + argv[0]
	runSupervisedCommand(*configPath, spec, actionName, "openagentsgate.tool.git", append([]string{"git"}, argv...), func(_ []string) *exec.Cmd {
		return exec.Command("git", argv...)
	})
}

func toolCommand(args []string, actionName, origin string, command func([]string) *exec.Cmd) {
	fs := flag.NewFlagSet("tool shell", flag.ExitOnError)
	spec := requestFlags(fs)
	configPath := fs.String("config", "examples/openagentsgate.json", "path to config file")
	_ = fs.Parse(args)
	argv := fs.Args()
	if len(argv) == 0 {
		log.Fatalf("tool shell: missing command")
	}
	runSupervisedCommand(*configPath, spec, actionName, origin, argv, command)
}

func runSupervisedCommand(configPath string, spec *requestSpec, actionName, origin string, argv []string, command func([]string) *exec.Cmd) {
	rt := mustRuntime(configPath)
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("tool: %v", err)
	}
	spec.action = valueOrDefault(spec.action, actionName)
	spec.resource = valueOrDefault(spec.resource, cwd)
	spec.origin = valueOrDefault(spec.origin, origin)
	spec.input = map[string]any{"argv": argv, "cwd": cwd}

	result, err := rt.service.Decide(spec.actionRequest(), time.Now().UTC())
	if err != nil {
		log.Fatalf("tool: %v", err)
	}
	if result.Decision.Effect != decision.EffectAllow {
		printJSON(result)
		os.Exit(exitCodeForEffect(result.Decision.Effect))
	}

	cmd := command(argv)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		log.Fatalf("tool: %v", err)
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

type requestSpec struct {
	requestID       string
	agentID         string
	agentInstanceID string
	userID          string
	sessionID       string
	action          string
	resource        string
	origin          string
	risk            string
	input           map[string]any
	metadata        map[string]any
	inputJSON       string
	inputFile       string
	metadataJSON    string
	metadataFile    string
}

func requestFlags(fs *flag.FlagSet) *requestSpec {
	spec := &requestSpec{}
	fs.StringVar(&spec.requestID, "request-id", "", "request id")
	fs.StringVar(&spec.agentID, "agent", "", "agent id")
	fs.StringVar(&spec.agentInstanceID, "agent-instance", "", "agent instance id")
	fs.StringVar(&spec.userID, "user", "", "user id")
	fs.StringVar(&spec.sessionID, "session", "", "session id")
	fs.StringVar(&spec.action, "action", "", "action name")
	fs.StringVar(&spec.resource, "resource", "", "resource name")
	fs.StringVar(&spec.origin, "origin", "", "request origin")
	fs.StringVar(&spec.risk, "risk", "", "risk label")
	fs.StringVar(&spec.inputJSON, "input-json", "", "input JSON object")
	fs.StringVar(&spec.inputFile, "input-file", "", "path to input JSON object, or - for stdin")
	fs.StringVar(&spec.metadataJSON, "metadata-json", "", "metadata JSON object")
	fs.StringVar(&spec.metadataFile, "metadata-file", "", "path to metadata JSON object, or - for stdin")
	return spec
}

func mustActionRequest(path string, spec *requestSpec) action.Request {
	req := action.Request{}
	if path != "" {
		if err := readJSON(path, &req); err != nil {
			log.Fatalf("request: %v", err)
		}
	}
	if spec != nil {
		spec.applyTo(&req)
	}
	if req.AgentID == "" {
		req.AgentID = "cli-agent"
	}
	if req.Origin == "" {
		req.Origin = "openagentsgate.cli"
	}
	return req
}

func (s *requestSpec) actionRequest() action.Request {
	req := action.Request{}
	s.applyTo(&req)
	if req.AgentID == "" {
		req.AgentID = "cli-agent"
	}
	if req.Origin == "" {
		req.Origin = "openagentsgate.cli"
	}
	return req
}

func (s *requestSpec) applyTo(req *action.Request) {
	if s == nil {
		return
	}
	if s.requestID != "" {
		req.RequestID = s.requestID
	}
	if s.agentID != "" {
		req.AgentID = s.agentID
	}
	if s.agentInstanceID != "" {
		req.AgentInstanceID = s.agentInstanceID
	}
	if s.userID != "" {
		req.UserID = s.userID
	}
	if s.sessionID != "" {
		req.SessionID = s.sessionID
	}
	if s.action != "" {
		req.Action = s.action
	}
	if s.resource != "" {
		req.Resource = s.resource
	}
	if s.origin != "" {
		req.Origin = s.origin
	}
	if s.risk != "" {
		req.Risk = s.risk
	}
	if s.inputJSON != "" || s.inputFile != "" {
		req.Input = mustJSONMap(s.inputJSON, s.inputFile, "input")
	}
	if s.metadataJSON != "" || s.metadataFile != "" {
		req.Metadata = mustJSONMap(s.metadataJSON, s.metadataFile, "metadata")
	}
	if s.input != nil {
		req.Input = s.input
	}
	if s.metadata != nil {
		req.Metadata = s.metadata
	}
}

func mustJSONMap(raw, path, label string) map[string]any {
	if raw != "" && path != "" {
		log.Fatalf("%s: use JSON string or file, not both", label)
	}
	var data []byte
	var err error
	switch {
	case raw != "":
		data = []byte(raw)
	case path != "":
		data, err = readAll(path)
	default:
		return nil
	}
	if err != nil {
		log.Fatalf("%s: %v", label, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		log.Fatalf("%s: invalid JSON object", label)
	}
	return out
}

func readJSON(path string, target any) error {
	data, err := readAll(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytesReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func readAll(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func bytesReader(data []byte) io.Reader {
	return &byteReader{data: data}
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func valueOrDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func exitCodeForEffect(effect decision.Effect) int {
	switch effect {
	case decision.EffectAllow:
		return 0
	case decision.EffectDryRun:
		return exitDryRun
	case decision.EffectApproval:
		return exitApproval
	default:
		return exitDeny
	}
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
	fmt.Fprintln(os.Stderr, "usage: openagentsgate <run|check|tool|decide|approvals|revocations|audit|version> [flags]")
}

func toolUsage() {
	fmt.Fprintln(os.Stderr, "usage: openagentsgate tool <shell|git> [flags] -- <command>")
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
