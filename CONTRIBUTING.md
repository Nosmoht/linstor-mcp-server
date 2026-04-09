# Contributing to linstor-mcp-server

## Quick start

```bash
git clone https://github.com/Nosmoht/linstor-mcp-server
cd linstor-mcp-server
make check
```

## Prerequisites

- Go `1.26.2`
- [golangci-lint](https://golangci-lint.run/welcome/install/) for `make lint` and `make check`
- `kubectl` configured for the target cluster if you want live validation

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

Default acceptance environment:

- Kubernetes `v1.35.0`
- Piraeus operator `v2.10.4`
- LINSTOR server `v1.33.1`
- LINSTOR API `1.27.0`

## Development

```bash
make build       # build the binary with version metadata
make test        # run package tests
make test-race   # run tests with the race detector
make lint        # run golangci-lint
make coverage    # generate coverage output
make fuzz-smoke  # short fuzz smoke tests
make check       # full CI parity: fmt + vet + verify + lint + test + build
make check-full  # extended validation
```

## Scope and safety

This repository targets a safe MCP server for LINSTOR and Piraeus.

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

Hard safety rules:

- Keep `piraeus.io/v1` CRDs as the desired-state source of truth.
- Keep the LINSTOR controller API as the runtime-state source of truth.
- Treat `internal.linstor.linbit.com` resources as diagnostics only.
- Never expose hashed internal resource names in the public MCP contract.
- Never mutate `LinstorSatellite`.
- Protect existing CSI-backed resources unless the change is explicitly scoped to synthetic fixtures.
- Preserve the two-step `plan_cluster_config` -> `apply_plan` flow and apply-time stale-plan revalidation.

## Tests

Minimum expectations for changes:

- unit tests for new pure logic
- regression tests for safety checks
- fuzz coverage for parser-heavy or user-input-heavy logic when practical
- `make check` passing locally

For live-cluster validation, prefer read-only checks unless the change is explicitly scoped to safe reversible fixtures.

## Commit messages

This project uses [conventional commits](https://www.conventionalcommits.org/). Use scoped prefixes where practical:

- `feat(scope):`
- `fix(scope):`
- `docs(scope):`
- `ci(scope):`
- `chore(scope):`
- `refactor(scope):`
- `test(scope):`

## Pull requests

1. Fork the repo and create a branch from `main`
2. Ensure `make check` passes locally
3. Fill in the PR template
4. Keep each PR to one logical change

## Claude Code Support

Project-local Claude Code helpers live under `.claude/`.
Use [README.md](/Users/ntbc/workspace/linstor-mcp-server/README.md) for the public tool surface and setup examples.

When changing these helpers, keep them aligned with `AGENTS.md` and preserve their read-only live-cluster boundary unless the user explicitly requests a safe, reversible fixture workflow.

## Security vulnerabilities

Do not open public issues for security bugs. Use [GitHub Private Vulnerability Reporting](https://github.com/Nosmoht/linstor-mcp-server/security/advisories/new) instead. See [SECURITY.md](/Users/ntbc/workspace/linstor-mcp-server/SECURITY.md) for details.

## License

By contributing, you agree your contributions will be licensed under the [MIT License](LICENSE).
