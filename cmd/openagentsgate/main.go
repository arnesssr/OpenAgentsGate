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
	"github.com/arnesssr/OpenAgentsGate/internal/audit"
	"github.com/arnesssr/OpenAgentsGate/internal/config"
	"github.com/arnesssr/OpenAgentsGate/internal/policy"
	"github.com/arnesssr/OpenAgentsGate/internal/server"
)

const version = "0.1.0-dev"

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

	cfg, evaluator, recorder := mustRuntime(*configPath)
	srv, err := server.New(evaluator, recorder)
	if err != nil {
		log.Fatalf("server: %v", err)
	}

	httpServer := srv.HTTPServer(cfg.ListenAddr)
	log.Printf("openagentsgate listening on http://%s", cfg.ListenAddr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}

func decide(args []string) {
	fs := flag.NewFlagSet("decide", flag.ExitOnError)
	configPath := fs.String("config", "examples/openagentsgate.json", "path to config file")
	_ = fs.Parse(args)

	_, evaluator, recorder := mustRuntime(*configPath)
	var req action.Request
	decoder := json.NewDecoder(os.Stdin)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		log.Fatalf("request: invalid JSON")
	}

	now := time.Now().UTC()
	req = req.WithDefaults(now)
	dec, err := evaluator.Decide(req, now)
	if err != nil {
		log.Fatalf("request: %v", err)
	}
	receipt, err := recorder.Record(req, dec, now)
	if err != nil {
		log.Fatalf("audit: %v", err)
	}
	response := map[string]any{
		"decision": dec,
		"receipt":  receipt.ID,
	}
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		log.Fatalf("response: %v", err)
	}
}

func mustRuntime(path string) (config.Config, *policy.Evaluator, *audit.Recorder) {
	cfg, err := config.Load(path)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	evaluator, err := policy.NewEvaluator(cfg.Policy)
	if err != nil {
		log.Fatalf("policy: %v", err)
	}
	recorder, err := audit.NewRecorder(cfg.AuditLog)
	if err != nil {
		log.Fatalf("audit: %v", err)
	}
	return cfg, evaluator, recorder
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: openagentsgate <run|decide|version> [flags]")
}
