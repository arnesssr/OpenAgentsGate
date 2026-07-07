package main

import (
	"fmt"
	"io"
	"os"
)

func usage(out io.Writer) {
	fmt.Fprintln(out, "usage: openagentsgate <init|run|check|wrap|tool|decide|approvals|revocations|audit|config|version> [flags]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "first run:")
	fmt.Fprintln(out, "  openagentsgate init")
	fmt.Fprintln(out, "  openagentsgate run")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "agent launch:")
	fmt.Fprintln(out, "  openagentsgate wrap -- codex")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "inspect config:")
	fmt.Fprintln(out, "  openagentsgate config path")
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
	fmt.Fprintln(os.Stderr, "usage: openagentsgate audit <list|get|replay|verify> [flags]")
}

func configUsage() {
	fmt.Fprintln(os.Stderr, "usage: openagentsgate config <path|doctor> [flags]")
}
