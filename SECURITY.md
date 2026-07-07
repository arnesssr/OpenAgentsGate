# Security Policy

OpenAgentsGate is security-sensitive software. It makes authorization,
approval, audit, replay, and revocation decisions for agent actions that may
touch source code, secrets, deployment paths, APIs, private data, messages, or
money.

## Reporting A Vulnerability

Do not open a public issue with exploit details, secrets, private logs, tokens,
or customer data.

Preferred reporting path:

1. Use GitHub private vulnerability reporting if it is enabled for this
   repository.
2. If private reporting is unavailable, contact the maintainer privately.
3. If no private channel is available, open a minimal public issue that says
   you have a security report, but do not include exploit details.

Include:

- affected version or commit
- affected command, API, adapter, or policy path
- impact
- reproduction steps
- whether secrets or private systems are involved
- suggested fix if known

## Supported Versions

This project is early. Security fixes are expected to target `main` first.
Tagged releases may receive fixes when the affected behavior exists in a
published release.

## Security Model

OpenAgentsGate assumes:

- agents can be confused by prompt injection
- tool output can be hostile
- credentials can leak
- users can approve the wrong thing
- integrations can be overpowered
- local state can be inspected by the user running the tool
- logs can become evidence in an incident

The core safety model is:

```text
normalized action request -> risk classification -> revocation check
-> policy decision -> approval if needed -> enforcement -> audit receipt
```

Revocation must override allow rules. Approval must be tied to the exact action
shape that was reviewed. Audit receipts must avoid secrets and private payloads
that are not needed for investigation.

## High-Risk Areas

Treat these changes as security-sensitive:

- policy evaluation
- action request parsing
- approval creation and resolution
- revocation matching
- audit receipt persistence and replay
- secret redaction
- HTTP admin API behavior
- command execution
- filesystem access
- Git and deployment adapters
- MCP or tool gateway adapters
- environment variable handling
- config discovery and default paths

## Secure Development Rules

- Default deny.
- Validate inputs at boundaries.
- Redact secrets before persistence.
- Do not log secrets.
- Do not shell-interpolate user-controlled input.
- Use least privilege for adapter actions.
- Keep authorization server-side or engine-side.
- Make policy version and decision reasons observable.
- Preserve auditability for denied and approval-required actions.
- Avoid broad wildcard permissions unless the policy file makes the risk clear.

## Out Of Scope For Public Reports

The following are usually not security vulnerabilities by themselves:

- a local user reading files they already own
- denial of service against a local development process without privilege gain
- missing hosted enterprise features that are not implemented
- social engineering without a product behavior flaw

When in doubt, report privately.
