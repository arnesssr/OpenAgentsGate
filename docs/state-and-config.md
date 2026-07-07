# State And Config

OpenAgentsGate v0 is local-first and file-backed. There is no required database
or cloud service.

## Source Of Truth

The v0 source of truth is:

- one JSON config file
- one append-only JSONL audit log
- one append-only JSONL approval event log
- one append-only JSONL revocation event log

These files are intentionally simple so a local operator can inspect, back up,
copy, diff, and recover them with normal tools.

## Config Discovery

Unless `-config` is provided, commands search in this order:

1. nearest `.openagentsgate/config.json` in the current directory or any parent
2. user config at `$XDG_CONFIG_HOME/openagentsgate/config.json`
3. user config at `~/.config/openagentsgate/config.json` when
   `$XDG_CONFIG_HOME` is unset

Commands that need runtime state may create the user config automatically.
Explicit `-config` paths must already exist unless the command is
`openagentsgate init -config <path>`.

Project config can be created with:

```bash
openagentsgate init -project
```

User config can be created with:

```bash
openagentsgate init
```

## State Directory

Default user state is stored under:

```text
$XDG_STATE_HOME/openagentsgate
```

If `$XDG_STATE_HOME` is unset, the fallback is:

```text
~/.local/state/openagentsgate
```

Project config created by `openagentsgate init -project` stores state under:

```text
.openagentsgate/state
```

Explicit config paths created by `openagentsgate init -config <path>` use the
current user state directory unless `XDG_STATE_HOME` is set for that command.

## State Files

Default config writes:

```text
audit.jsonl
approvals.jsonl
revocations.jsonl
```

The sample repo config writes:

```text
tmp/openagentsgate.audit.jsonl
tmp/openagentsgate.approvals.jsonl
tmp/openagentsgate.revocations.jsonl
```

The `tmp/` directory is ignored by Git.

## Audit Log

The audit log stores append-only receipts. Each receipt includes:

- normalized request
- decision
- optional approval id
- previous receipt hash
- integrity hash
- recorded timestamp

Verify the hash chain:

```bash
openagentsgate audit verify
```

Replay a receipt against current policy and revocation state:

```bash
openagentsgate audit replay -id <receipt-id>
```

Verification detects missing hashes, receipt tampering, and broken previous hash
links. It does not repair old logs automatically.

## Approval Log

The approval log stores append-only lifecycle events:

- `created`
- `resolved`

Approvals are reconstructed from events. A resolution before a creation is
treated as corrupt lifecycle state and returns an error instead of synthesizing
an approval record.

Pending approvals are deduplicated by `request_id`. Once an approval is
resolved, the same request id can create a new pending approval if the action is
submitted again.

## Revocation Log

The revocation log stores append-only events:

- `revoke`
- `restore`

Active revocations are reconstructed by applying events in order. Later events
for the same target override earlier events.

Supported targets:

- all agents
- agent id
- agent instance id
- user id
- session id
- action name

Revocation is evaluated before normal policy allow rules in the gateway.

## File Permissions

OpenAgentsGate writes config and JSONL state files with owner-only permissions
where it creates them:

```text
0600 files
0700 directories
```

State files may still contain sensitive operational metadata even after secret
redaction. Do not commit state files.

## Recovery Notes

Back up the config file and JSONL state files together.

If an audit log fails verification:

1. preserve the file as evidence
2. inspect the first reported line
3. compare against backups if available
4. rotate to a new state directory only after preserving the old one

If approval or revocation logs fail to load, preserve the original JSONL file
before editing. Manual repair should be explicit and reviewed because these
logs represent authority and incident-response state.
