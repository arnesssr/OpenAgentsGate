## Summary

What changed?

## Why

What problem does this solve?

## Security Impact

Does this affect policy, approvals, revocation, audit, command execution,
config, secrets, HTTP APIs, adapters, or stored state?

## Compatibility Impact

Does this change CLI output, config, schemas, HTTP APIs, audit receipts, or
adapter behavior?

## Verification

Commands run:

```bash
go test ./...
go vet ./...
make build
```

Docs-only changes:

```bash
git diff --check
```

## Tradeoffs

What was deliberately not done?
