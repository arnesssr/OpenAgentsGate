# OpenAgentsGate Agent Notes

## Product Direction

OpenAgentsGate is a portable governance layer for agent actions.

It is not another agent framework, a generic approval prompt, or an MCP-only
firewall. MCP is an adapter target. The core product is policy, audit, replay,
and revocation across agentic developer tools such as Codex, Claude Code, CI
agents, custom scripts, MCP tools, and future SDK integrations.

The primary user-facing wedge is:

```text
agent/tool/script -> normalized action request -> OpenAgentsGate -> decision
```

## Core Invariants

- Default-deny unless policy explicitly allows, dry-runs, or requires approval.
- Every decision path must be auditable.
- Secrets must not be persisted or echoed in responses.
- Revocation must override policy allow rules.
- Replay must compare a historical receipt against current policy state.
- The core must stay protocol-neutral; adapter code translates external tool
  shapes into the internal action model.
- Keep every source and documentation file under 700 lines.

## Architecture Map

- `internal/action`: protocol-neutral action request model.
- `internal/risk`: risk classification before policy evaluation.
- `internal/policy`: deterministic policy evaluator.
- `internal/gateway`: orchestration of risk, revocation, policy, approvals, and audit.
- `internal/audit`: append-only, hash-chained audit receipts.
- `internal/approval`: append-only approval lifecycle events.
- `internal/revocation`: kill switches for actors, sessions, and actions.
- `internal/server`: HTTP transport only; no business rules here.
- `cmd/openagentsgate`: CLI surfaces for checks, supervised tools, and admin commands.

## Implementation Rules

- Prefer Go standard library until a dependency clearly reduces risk or complexity.
- Keep business rules out of HTTP handlers and command parsing.
- Do not add Postgres or Redis for v0; JSON policy plus append-only JSONL stores
  are the current source of truth.
- Do not shell-interpolate user input. Supervised commands must use argv with
  `os/exec`.
- Keep runtime logs under `tmp/` or another ignored path.
- Distribution is local-first: `go install`, static release binaries, and GitHub
  Releases. Do not introduce a required OpenAgentsGate cloud server.

## Verification

Run these before claiming implementation is complete:

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
make build
./bin/openagentsgate version
go run ./cmd/openagentsgate check -config examples/openagentsgate.json -action github.create_pr -agent codex -resource repo
go run ./cmd/openagentsgate tool git -config examples/openagentsgate.json -- status --short
```

For HTTP changes, also start:

```bash
go run ./cmd/openagentsgate run -config examples/openagentsgate.json
```

Then exercise the changed endpoint with a local request.
