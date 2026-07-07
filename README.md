<p align="center">
  <img src="assets/openagentsgate-logo.png" alt="OpenAgentsGate logo" width="180">
</p>

# OpenAgentsGate

Portable policy, audit, replay, and revocation for AI agent actions.

Before an agent touches tools, files, APIs, money, email, code, or another
agent, who proves it is allowed, what exactly is it allowed to do, and how do
we audit, revoke, replay, or roll it back?

Think of OpenAgentsGate as:

```text
OAuth + sudo + firewall + audit log + sandbox policy for AI agents
```

## Why This Exists

Agent frameworks are getting good at planning, tool calling, delegation, and
workflow execution. That creates a harder problem: once agents can act, every
real system needs a durable way to control what they are allowed to do.

OpenAgentsGate sits between agentic tools and action surfaces. It is designed
to give Codex, Claude Code, CI agents, custom scripts, MCP tools, and future
adapters one shared policy and audit layer.

## Core Goals

- Enforce capability-based permissions before agent actions run.
- Require human approval for high-impact operations.
- Keep tamper-evident, replayable action receipts for audit and debugging.
- Contain prompt-injection risk from untrusted content.
- Provide a kill switch and revocation path for agents, tools, and policies.
- Work across agent frameworks instead of replacing them.

## Project Direction

OpenAgentsGate is becoming a neutral authorization control plane for agent
actions.

The project should stay protocol-neutral: agents, IDEs, CI systems, MCP
servers, custom scripts, and future SDKs should be able to ask the same
question before an action runs:

```text
Can this agent perform this action on this resource under this authority?
```

The core project answers with a deterministic decision:

```text
allow | deny | dry_run | require_approval
```

OpenAgentsGate is not an agent runtime, model provider, IDE, MCP server, or
hosted cloud requirement. It is the policy, approval, audit, replay, and
revocation layer those systems can share.

## Initial Scope

The first target is a protocol-neutral policy gateway for agentic developer
workflows:

- Accept normalized action requests from agents, apps, or adapters.
- Classify tool calls by risk.
- Allow, deny, dry-run, or require approval.
- Log every attempted and completed action.
- Expose a simple policy file for repeatable configuration.
- Provide CLI surfaces that other tools can call before acting.

MCP support belongs in an adapter. It is not a dependency of the core product.

## Backend Status

The backend is a Go service and CLI with no database dependency.

For v0, the source of truth is:

- A JSON policy file.
- Append-only JSONL audit receipts.
- Append-only JSONL approval events.
- Append-only JSONL revocation events.

SQLite is the likely next storage layer once approvals and audit history need
querying. Redis is not required for the MVP; it only becomes useful later for
distributed deployments, realtime fanout, or queueing.

Implemented v0 backend capabilities:

- Protocol-neutral action request model.
- Configurable risk classification.
- Default-deny policy evaluation.
- Allow, deny, approval-required, and dry-run decisions.
- Pending approval creation and approval resolution.
- Revocation kill switches for all agents, one agent, one agent instance, one
  user, one session, or one action.
- Secret redaction before approval or audit persistence.
- Append-only, hash-chained audit receipts.
- Audit receipt lookup and replay against current policy/revocation state.
- Local HTTP API and CLI commands.

## Quick Start

Install:

```bash
go install github.com/arnesssr/OpenAgentsGate/cmd/openagentsgate@latest
openagentsgate version
```

Build locally:

```bash
make build
./bin/openagentsgate version
```

Release binaries are published from Git tags. After the first tagged release,
the installer fetches the latest binary from GitHub Releases. Before that, it
falls back to `go install`:

```bash
curl -fsSL https://raw.githubusercontent.com/arnesssr/OpenAgentsGate/main/scripts/install.sh | sh
```

OpenAgentsGate is local-first. Updates come from GitHub Releases or `go install`;
there is no required OpenAgentsGate cloud server.

Create the default user config:

```bash
openagentsgate init
openagentsgate config doctor
```

Without `-config`, the CLI uses the nearest `.openagentsgate/config.json` above
the current directory. If none exists, it uses
`~/.config/openagentsgate/config.json` and stores local JSONL state under
`~/.local/state/openagentsgate`.

See [docs/state-and-config.md](docs/state-and-config.md) for the config
discovery order, JSONL state files, and recovery notes.

Check a single action from flags:

```bash
openagentsgate check -action github.create_pr -agent codex -resource repo
```

`check` exits `0` for allow, `10` for dry-run, `20` for approval required, and
`30` for deny.

Check a full action request from stdin:

```bash
openagentsgate check -request - -strict-exit=false < examples/action-request.json
```

Run a supervised read-only git command:

```bash
openagentsgate tool git -- status --short
```

Inspect a shell command without executing it when policy says dry-run:

```bash
openagentsgate tool shell -- npm test
```

Launch an existing agent CLI under an audited session:

```bash
openagentsgate wrap -- codex
```

`wrap` authorizes and audits the process launch, then passes
`OPENAGENTSGATE_CONFIG`, `OPENAGENTSGATE_SESSION`, and `OPENAGENTSGATE_AGENT`
into the child process. Per-action enforcement inside that child requires the
agent or an adapter to call OpenAgentsGate before it acts.

Start the local HTTP decision service:

```bash
openagentsgate run
```

Then send an action request:

```bash
curl -s http://127.0.0.1:17671/v1/actions/decide \
  -H 'Content-Type: application/json' \
  --data @examples/action-request.json
```

The gateway returns a deterministic decision and records an audit receipt.

List pending approvals:

```bash
openagentsgate approvals list -status pending
```

Resolve an approval:

```bash
openagentsgate approvals resolve \
  -id <approval-id> -status approved -by admin -reason "reviewed"
```

Revoke an agent:

```bash
openagentsgate revocations add \
  -type agent -id support-agent -by admin -reason "compromised"
```

Replay an audit receipt against current policy state:

```bash
openagentsgate audit replay -id <receipt-id>
```

Verify the audit log hash chain:

```bash
openagentsgate audit verify
```

HTTP API:

```text
POST   /v1/actions/decide
GET    /v1/approvals
GET    /v1/approvals/{id}
POST   /v1/approvals/{id}/resolve
GET    /v1/revocations
POST   /v1/revocations
DELETE /v1/revocations/{target_type}/{target_id}
GET    /v1/audit
GET    /v1/audit/{id}
POST   /v1/audit/{id}/replay
```

By default the service binds to `127.0.0.1`. If you bind it to a non-loopback
address, configure `admin_token_env` so HTTP API calls require
`Authorization: Bearer <token>`.

## Security Posture

OpenAgentsGate assumes agents can be confused, prompts can be hostile, tools can
be overpowered, credentials can leak, and humans need useful defaults. Policy is
enforced outside the model, not by asking the model to behave.

## Status

Early project. The intended direction is a minimal, secure-by-default gateway
before adding broader framework adapters.

Phase 1 core hardening is complete. See [ROADMAP.md](ROADMAP.md) for the
current public direction.

## Contributing

Contributions should make the project safer, clearer, or easier to integrate.
Start with [CONTRIBUTING.md](CONTRIBUTING.md), then check open issues and the
roadmap.

Larger changes should begin as an RFC under [docs/rfcs](docs/rfcs). Accepted
architecture decisions are recorded under [docs/adr](docs/adr).

Security-sensitive reports should follow [SECURITY.md](SECURITY.md). Do not
open public issues with exploit details, secrets, or private system data.

## License

OpenAgentsGate is licensed under the Apache License 2.0. See [LICENSE](LICENSE).
