# ADR 0001: OpenAgentsGate Is A Policy Control Plane

Status: accepted
Date: 2026-07-06

## Context

Agent systems are gaining the ability to call tools, edit files, run commands,
open pull requests, deploy code, access APIs, and act through adapters. These
actions need policy, approval, audit, replay, and revocation outside the model.

The project started as a local CLI and service with launch wrapping,
supervised commands, policy decisions, approvals, revocations, and JSONL audit
receipts.

Launch wrapping is useful, but it cannot govern hidden child actions unless the
agent or adapter asks OpenAgentsGate before each action.

## Decision

OpenAgentsGate is a protocol-neutral authorization control plane for agent
actions.

It is not an agent runtime, model provider, IDE, MCP server, or hosted-only
cloud product. Those systems are integration targets.

The core project should focus on:

- normalized action requests
- deterministic decisions
- explicit authority and capability grants
- approval workflows for high-impact actions
- revocation before policy allow
- audit receipts and replay
- adapters that enforce decisions at action boundaries

## Consequences

Contributors should prefer public contracts, schemas, adapters, and tests over
framework-specific behavior in the core.

Adapters may know about shells, Git, MCP, GitHub, filesystems, deployments, or
other tools. The core engine should remain protocol-neutral.

Public design changes that alter contracts, semantics, storage, or the
security model should start as RFCs and result in ADRs when accepted.

## Tradeoffs

This direction is slower than building one deep integration first. It is also
more durable because it lets different agent systems share the same control
surface without surrendering their own product architecture.
