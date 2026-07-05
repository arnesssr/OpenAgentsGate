package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/arnesssr/OpenAgentsGate/internal/decision"
	"github.com/arnesssr/OpenAgentsGate/internal/ids"
)

func wrapCommand(args []string) {
	fs := flag.NewFlagSet("wrap", flag.ExitOnError)
	spec := requestFlags(fs)
	configPath := configFlag(fs)
	_ = fs.Parse(args)
	argv := fs.Args()
	if len(argv) == 0 {
		log.Fatalf("wrap: missing command; usage: openagentsgate wrap [flags] -- <command> [args...]")
	}

	rt := mustRuntime(*configPath)
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("wrap: %v", err)
	}
	if spec.agentID == "" {
		spec.agentID = filepath.Base(argv[0])
	}
	if spec.sessionID == "" {
		spec.sessionID = ids.New()
	}
	spec.action = valueOrDefault(spec.action, "agent.wrap")
	spec.resource = valueOrDefault(spec.resource, cwd)
	spec.origin = valueOrDefault(spec.origin, "openagentsgate.wrap")
	spec.input = commandInput(argv, cwd)
	spec.metadata = map[string]any{
		"enforcement": "launch_audit",
	}

	result, err := rt.service.Decide(spec.actionRequest(), time.Now().UTC())
	if err != nil {
		log.Fatalf("wrap: %v", err)
	}
	if result.Decision.Effect != decision.EffectAllow {
		printJSON(result)
		os.Exit(exitCodeForEffect(result.Decision.Effect))
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"OPENAGENTSGATE_CONFIG="+rt.configPath,
		"OPENAGENTSGATE_SESSION="+result.Request.SessionID,
		"OPENAGENTSGATE_AGENT="+result.Request.AgentID,
	)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		log.Fatalf("wrap: %v", err)
	}
}
