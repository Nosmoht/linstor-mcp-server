# Contributing

## Scope

This repository targets a safe v1 MCP server for LINSTOR and Piraeus.

Current GA scope:

- `stdio` transport
- inventory reads
- `LinstorCluster`, `LinstorSatelliteConfiguration`, and `LinstorNodeConnection` planning/apply
- non-destructive, operator-safe behavior

Out of GA scope unless explicitly planned:

- destructive actions
- snapshots, backups, remotes, schedules
- failover, evacuation, rebalance
- direct mutation of existing CSI-backed production resources

## Requirements

- Go `1.26.2`
- `kubectl` configured for the target cluster if you want live validation
- default acceptance environment:
  - Kubernetes `v1.35.0`
  - Piraeus operator `v2.10.4`
  - LINSTOR server `v1.33.1`
  - LINSTOR API `1.27.0`

## Development Workflow

Run the standard checks:

```bash
make check
```

Useful targets:

```bash
make build
make test
make test-race
make coverage
make fuzz-smoke
make verify
make check-full
make run
```

## Claude Code Support

Project-local Claude Code helpers live under `.claude/`.
Use [README.md](/Users/ntbc/workspace/linstor-mcp-server/README.md) for the
discoverable inventory and usage examples.

When changing these helpers, keep them aligned with `AGENTS.md` and preserve
their read-only live-cluster boundary unless the user explicitly requests a
safe, reversible fixture workflow.

## Safety Rules

- Keep `piraeus.io/v1` CRDs as the source of truth for desired operator state.
- Keep the LINSTOR controller API as the source of truth for runtime LINSTOR state.
- Treat `internal.linstor.linbit.com` resources as diagnostics only.
- Never expose hashed internal resource names in the public MCP contract.
- Never mutate `LinstorSatellite`.
- Protect existing CSI-backed resources unless the change is explicitly scoped to synthetic e2e fixtures.
- Preserve the two-step `plan_cluster_config` -> `apply_plan` flow and apply-time stale-plan revalidation.
- Keep repo-local Claude workflows read-only against the live cluster unless the
  user explicitly requests a safe, reversible fixture workflow.

## Tests

Minimum expectations for changes:

- unit tests for any new pure logic
- regression tests for safety checks
- fuzz coverage for parser-heavy or user-input-heavy logic when practical
- `make check` passing locally

For live-cluster validation, prefer read-only tests unless the change is explicitly scoped to synthetic reversible fixtures.

## Documentation

Update the end-user `README.md` whenever you change:

- the supported scope
- setup steps
- the tool surface
- safety guarantees
- supported versions
