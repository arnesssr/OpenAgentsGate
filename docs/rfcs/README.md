# RFCs

RFCs are public design proposals for changes that affect project direction,
public contracts, security behavior, or contributor expectations.

Use an RFC before changing:

- public schemas or wire formats
- decision effect semantics
- policy behavior
- approval or revocation behavior
- audit receipt structure
- adapter contracts
- storage source of truth
- security model
- compatibility guarantees

Small bug fixes and narrow tests usually do not need an RFC.

## Process

1. Open an issue describing the problem.
2. Add an RFC file under this directory.
3. Discuss tradeoffs in the pull request.
4. Update the RFC until the direction is clear.
5. If accepted, implement the change and record lasting decisions as ADRs.

## Naming

Use:

```text
NNNN-short-title.md
```

Example:

```text
0001-action-request-schema.md
```

## Template

Copy `0000-template.md` for new RFCs.
