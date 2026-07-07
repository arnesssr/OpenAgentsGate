# Roadmap

OpenAgentsGate is moving from a local wrapper and policy CLI toward a neutral
authorization control plane for agent actions.

The roadmap is directional, not a promise of dates. Security, correctness, and
compatibility take priority over speed.

## North Star

Before an agent touches a risky tool, OpenAgentsGate should be able to answer:

```text
Can this agent perform this action on this resource under this authority?
```

The decision should be deterministic, enforceable, auditable, replayable, and
revocable.

## Non-Goals

OpenAgentsGate is not trying to become:

- a model provider
- a general agent runtime
- an IDE
- a hosted-only cloud product
- an MCP-only firewall
- a replacement for sandboxing

MCP, IDEs, CI systems, shells, Git hosts, and custom agents are integration
targets. The core is the authorization, approval, audit, replay, and revocation
layer.

## Phase 1: Harden The Current Core

Status: complete.

Goal: make the existing local engine boring and trustworthy.

Work:

- strengthen policy, approval, revocation, and audit tests
- add regression tests for secret redaction
- add audit verification commands
- document JSONL state files and config discovery
- keep the HTTP layer transport-only
- keep the CLI installable outside the source repository

Completed evidence:

- policy, approval, revocation, audit, redaction, HTTP boundary, config, and CLI
  tests cover the current core behavior
- `openagentsgate audit verify` validates audit log hash chains
- `make verify` creates isolated temp state and verifies a fresh audit chain
- `docs/state-and-config.md` documents JSONL state files and config discovery
- HTTP handlers remain limited to auth, decoding, status mapping, and gateway
  delegation
- installability is verified with an installed binary executed outside the
  source checkout

Exit criteria:

- `go test ./...` passes
- `go vet ./...` passes
- `make build` passes
- audit receipts can be verified locally
- redaction behavior is tested against common secret shapes

## Phase 2: Define Versioned Protocol Contracts

Goal: make OpenAgentsGate bigger than one binary.

Work:

- add `spec/v0/`
- define canonical schemas for action requests and decisions
- define approval, revocation, audit receipt, agent identity, and capability
  grant schemas
- define decision effect semantics
- define error and compatibility rules
- add conformance fixtures

Exit criteria:

- another implementation can produce valid requests and decisions
- adapter authors can work from public contracts
- breaking changes require a version bump

## Phase 3: Build Enforcement Adapters

Goal: enforce decisions at real action boundaries.

Initial adapter targets:

- shell
- Git
- filesystem
- MCP gateway
- GitHub or GitLab
- HTTP/API proxy

Exit criteria:

- denied actions are blocked before execution
- dry-run actions do not mutate state
- approval-required actions pause clearly
- attempted actions are auditable
- adapters do not duplicate policy logic

## Phase 4: Signed Replayable Receipts

Goal: make audit useful for debugging, incident response, and enterprise trust.

Work:

- canonicalize receipt payloads
- hash-chain receipts
- support optional signing keys
- include policy version and adapter identity
- include authority source and redacted inputs
- include execution result where available
- verify and replay receipts

Exit criteria:

- receipt tampering is detectable
- replay behavior is documented
- verification failures are actionable

## Phase 5: SDKs And Conformance

Goal: make integration easy.

Likely SDK targets:

- TypeScript
- Go
- Python
- Rust

Work:

- conformance CLI
- shared fixtures
- adapter authoring guide
- minimal examples

Exit criteria:

- vendors can integrate without shelling out to the CLI
- adapter behavior can be tested against shared fixtures
- examples are small enough to copy into real projects

## Phase 6: Public Governance

Goal: make the project credible to outside contributors and larger adopters.

Work:

- publish compatibility policy
- publish release policy
- publish maintainer roles
- publish project decision process
- define conformance or certification criteria
- keep license, release, and compatibility guidance explicit

Exit criteria:

- contributors know how decisions are made
- users know what is stable
- security issues have a clear private reporting path
- external contributors understand licensing terms

## Current Contribution Priorities

The best near-term contributions are:

- RFCs for `spec/v0` schemas and adapter contracts
- conformance fixtures for action requests, decisions, approvals, revocations,
  and audit receipts
- small CLI usability fixes
- issue reproductions with exact commands
- focused tests for new protocol or adapter work

Avoid large rewrites without an issue or RFC first.
