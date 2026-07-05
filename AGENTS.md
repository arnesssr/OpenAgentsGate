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

## CLI Contract

- Installed users should be able to run `openagentsgate` without being inside
  the source repository.
- `-config` is optional. The CLI first looks for `.openagentsgate/config.json`
  in the current directory or a parent, then falls back to the user config at
  `$XDG_CONFIG_HOME/openagentsgate/config.json` or
  `~/.config/openagentsgate/config.json`.
- Commands that need a runtime may create the starter user config when no
  project or user config exists. Explicit `-config` paths must already exist
  unless the user is running `openagentsgate init -config <path>`.
- Default user runtime state belongs under `$XDG_STATE_HOME/openagentsgate` or
  `~/.local/state/openagentsgate`.
- `openagentsgate wrap -- <command>` authorizes and audits launching that
  process and passes OpenAgentsGate env vars into it. It does not intercept
  hidden child-tool actions unless that agent or adapter calls OpenAgentsGate.

## Implementation Rules

- Prefer Go standard library until a dependency clearly reduces risk or complexity.
- Keep business rules out of HTTP handlers and command parsing.
- Do not add Postgres or Redis for v0; JSON policy plus append-only JSONL stores
  are the current source of truth.
- Do not shell-interpolate user input. Supervised commands must use argv with
  `os/exec`.
- Keep runtime logs under user state, project `.openagentsgate/state`, `tmp/`,
  or another ignored path.
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
go run ./cmd/openagentsgate config doctor
go run ./cmd/openagentsgate check -action github.create_pr -agent codex -resource repo
go run ./cmd/openagentsgate tool git -- status --short
go run ./cmd/openagentsgate wrap -- /bin/echo wrapped
```

For HTTP changes, also start:

```bash
go run ./cmd/openagentsgate run
```

Then exercise the changed endpoint with a local request.
