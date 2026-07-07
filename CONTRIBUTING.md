# Contributing To OpenAgentsGate

OpenAgentsGate is a policy, approval, audit, replay, and revocation layer for
AI agent actions. Contributions should strengthen that control layer without
turning the project into a model provider, agent runtime, IDE, or hosted-only
service.

## Public Source Of Truth

Contributors should be able to work from public project artifacts:

- `README.md` for product scope and quick start.
- `ROADMAP.md` for current direction.
- `SECURITY.md` for security reporting and security expectations.
- `docs/rfcs/` for proposed design changes.
- `docs/adr/` for accepted architecture decisions.
- GitHub issues for concrete work with acceptance criteria.

Maintainers may keep private drafts, but private drafts must not be required to
understand a public issue, API, roadmap commitment, or review rule.

## Before You Start

Use an issue before starting substantial work. Good issues describe:

- the user or operator problem
- the current behavior
- the proposed behavior
- security impact
- compatibility impact
- expected tests or verification

Small fixes, typos, and narrow test improvements can go straight to a pull
request.

## Development Setup

Requirements:

- Go 1.22 or newer
- Git
- Make

Build and test:

```bash
go test ./...
go vet ./...
make build
./bin/openagentsgate version
```

Run the CLI from source:

```bash
go run ./cmd/openagentsgate version
go run ./cmd/openagentsgate config doctor
go run ./cmd/openagentsgate check -action github.create_pr -agent codex -resource repo
```

## Architecture Boundaries

Keep the core protocol-neutral.

- `internal/action`: normalized action request model.
- `internal/risk`: risk classification.
- `internal/policy`: deterministic policy evaluation.
- `internal/gateway`: orchestration across risk, revocation, policy, approvals,
  and audit.
- `internal/audit`: append-only receipt storage and replay support.
- `internal/approval`: approval lifecycle.
- `internal/revocation`: kill switches.
- `internal/server`: HTTP transport only.
- `cmd/openagentsgate`: CLI and command surfaces.

Adapters should normalize external tool behavior into action requests, call the
decision engine, enforce the returned effect, and record the result. Adapters
should not become independent policy engines.

## Design Changes

Open an RFC for changes that alter:

- public schemas or wire formats
- decision semantics
- policy behavior
- approval or revocation behavior
- audit receipt structure
- adapter contracts
- storage source of truth
- security model
- compatibility guarantees

Accepted decisions should become ADRs in `docs/adr/`.

## Security Expectations

This project controls actions that may touch code, credentials, deployment
paths, private data, messages, APIs, and money. Treat every boundary as hostile.

Hard rules:

- default deny unless policy explicitly allows, dry-runs, or requires approval
- redact secrets before persistence or response output
- never log tokens, passwords, private keys, or API keys
- bind approvals to the exact action shape that was reviewed
- let revocation override allow rules
- validate external inputs at boundaries
- use argv APIs for commands instead of shell interpolation
- keep business rules out of transport handlers

Security-sensitive changes need tests for abuse cases, not just happy paths.

## Code Style

- Prefer the Go standard library unless a dependency clearly reduces risk or
  complexity.
- Keep state explicit.
- Keep failure modes explicit.
- Keep files under 700 lines.
- Keep public errors useful without leaking internal details.
- Keep CLI output script-friendly where practical.
- Add comments only where they clarify non-obvious behavior.

## Pull Request Checklist

Before opening a pull request, run:

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
make build
```

For docs-only changes, run:

```bash
git diff --check
```

Your pull request should explain:

- what changed
- why it is needed
- security impact
- compatibility impact
- verification performed
- any tradeoffs or known limits

## Licensing Note

This repository does not currently include a public license file. Maintainers
should choose and publish a license before accepting broad external
contributions. Until then, contributors should confirm licensing expectations
with a maintainer before submitting large work.
