# OpenAgentsGate

Framework-agnostic action control plane for AI agents.

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

The first target is an MCP-compatible policy gateway for local developers and
small teams:

- Wrap MCP servers behind a policy layer.
- Classify tool calls by risk.
- Allow, deny, dry-run, or require approval.
- Log every attempted and completed action.
- Expose a simple policy file for repeatable configuration.

## Security Posture

OpenAgentsGate assumes agents can be confused, prompts can be hostile, tools can
be overpowered, credentials can leak, and humans need useful defaults. Policy is
enforced outside the model, not by asking the model to behave.

## Status

Early project. The intended direction is a minimal, secure-by-default gateway
before adding broader framework adapters.
