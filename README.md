<p align="center">
  <img src="assets/openagentsgate-logo.png" alt="OpenAgentsGate logo" width="180">
</p>

# OpenAgentsGate

Framework-agnostic action control plane for AI agent actions.

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

OpenAgentsGate sits between agents and action surfaces. It is designed to
enforce explicit policy before tool calls, record what happened after every
action, and make dangerous autonomy auditable, revocable, and debuggable.

## Core Goals

- Enforce capability-based permissions before agent actions run.
- Require human approval for high-impact operations.
- Keep signed, replayable action receipts for audit and debugging.
- Contain prompt-injection risk from untrusted content.
- Provide a kill switch and revocation path for agents, tools, and policies.
- Work across agent frameworks instead of replacing them.

## Initial Scope

The first target is a protocol-neutral policy gateway for local developers and
small teams:

- Accept normalized action requests from agents, apps, or adapters.
- Classify tool calls by risk.
- Allow, deny, dry-run, or require approval.
- Log every attempted and completed action.
- Expose a simple policy file for repeatable configuration.

MCP support belongs in an adapter. It is not a dependency of the core product.

## Backend Status

The first backend slice is a Go service and CLI with no database dependency.

For v0, the source of truth is:

- A JSON policy file.
- Append-only JSONL audit receipts.

SQLite is the likely next storage layer once approvals and audit history need
querying. Redis is not required for the MVP; it only becomes useful later for
distributed deployments, realtime fanout, or queueing.

## Quick Start

Run a policy decision from stdin:

```bash
go run ./cmd/openagentsgate decide \
  -config examples/openagentsgate.json < examples/action-request.json
```

Start the local HTTP decision service:

```bash
go run ./cmd/openagentsgate run -config examples/openagentsgate.json
```

Then send an action request:

```bash
curl -s http://127.0.0.1:17671/v1/actions/decide \
  -H 'Content-Type: application/json' \
  --data @examples/action-request.json
```

The gateway returns a deterministic decision and records an audit receipt.

## Security Posture

OpenAgentsGate assumes agents can be confused, prompts can be hostile, tools can
be overpowered, credentials can leak, and humans need useful defaults. Policy is
enforced outside the model, not by asking the model to behave.

## Status

Early project. The intended direction is a minimal, secure-by-default gateway
before adding broader framework adapters.
