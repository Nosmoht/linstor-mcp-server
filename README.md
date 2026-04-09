# LINSTOR MCP Server

`linstor-mcp-server` is a Go MCP server for LINSTOR and the Piraeus operator.

## Status

This repository contains the first usable v1 MVP.

What that means:

- `stdio` transport is the primary mode.
- Streamable HTTP exists behind `--enable-http-beta`.
- Safe GA tools only:
  - `validate_environment`
  - `inventory_list`
  - `inventory_get`
  - `plan_cluster_config`
  - `apply_plan`
  - `job_get`
  - `job_cancel`
- Cluster mutations are limited to `LinstorCluster`, `LinstorSatelliteConfiguration`, and `LinstorNodeConnection`.
- Existing CSI-backed production resources are treated as read-only.

## Supported Versions

Pinned and currently validated:

- Go `1.26.2`
- Go toolchain `go1.26.2`
- MCP Go SDK `v1.5.0`
- Kubernetes client libraries `v0.35.3`
- SQLite driver `modernc.org/sqlite v1.48.1`
- Acceptance cluster:
  - Kubernetes `v1.35.0`
  - Piraeus operator `v2.10.4`
  - LINSTOR server `v1.33.1`
  - LINSTOR API `1.27.0`
- Agent clients validated during development:
  - Codex CLI `0.118.0`
  - Claude Code `2.1.97`

## Quickstart

Build:

```bash
make build
```

Run with the built-in `homelab` profile:

```bash
make run
```

Add to Codex CLI:

```bash
codex mcp add linstor -- ./bin/linstor-mcp-server
```

Add to Claude Code:

```bash
claude mcp add linstor -- ./bin/linstor-mcp-server
```

## Claude Code Helpers

This repo includes project-local Claude Code helpers under `.claude/`.

Skills:

- `linstor-mcp-guardrails`
  - background knowledge skill
  - not intended for direct invocation
  - keeps Claude aligned with this repo's safety rules, authority boundaries,
    and validation workflow
- `/plan-linstor-mcp-change`
  - manual planning workflow for MCP surface, planner/apply, transport, config,
    and contributor-workflow changes
  - pulls repo context first and uses current upstream facts when needed
- `/validate-linstor-mcp`
  - manual validation workflow
  - prefers repo checks first and only uses read-only homelab checks when live
    validation is needed

Project subagents:

- `linstor-safety-reviewer`
  - reviews ownership boundaries, stale-plan protections, and mutation safety
- `mcp-contract-reviewer`
  - reviews tool contracts, resource URIs, schema drift, and transport behavior
- `upstream-doc-researcher`
  - checks `kb-server` first, then official upstream docs for current facts

Typical usage:

```text
/plan-linstor-mcp-change add a new inventory filter for storage pools
/validate-linstor-mcp internal/app
Use linstor-safety-reviewer to review this planner change
Use mcp-contract-reviewer to review this HTTP transport change
```

Safety boundary:

- these helpers are for planning, review, and validation
- live cluster activity stays read-only by default
- they must not be used to mutate `LinstorSatellite`

## What You Can Do Now

- Validate the active kube context and LINSTOR controller connection.
- List canonical cluster, satellite configuration, node connection, node, storage-pool, resource-definition, and resource inventory.
- Read single canonical objects using semantic IDs.
- Create a safe `plan_cluster_config` plan for:
  - `cluster`
  - `satellite_configuration`
  - `node_connection`
- Apply a fresh plan with `apply_plan`.
- Track jobs with `job_get` and `job_cancel`.

## What Is Intentionally Not In GA Yet

- destructive operations
- snapshots, backups, remotes, schedules
- failover, rebalance, evacuation
- direct mutation of live CSI-backed production resources
- broad HTTP deployment and OAuth flows

## Tool Reference

### `validate_environment`

Returns:

- active kube context
- LINSTOR cluster name
- controller/API version
- default storage class
- basic health checks

### `inventory_list`

Arguments:

- `kind`
- `name_prefix`
- `node`
- `owner`
- `limit`
- `cursor`

### `inventory_get`

Arguments:

- `kind`
- `id`

### `plan_cluster_config`

Arguments:

- `kind`: `cluster`, `satellite_configuration`, `node_connection`
- `name`
- `operation`: `create`, `update`, `reconcile`
- `spec`

Returns a `plan_id`, diff, summary, preconditions, and expiry timestamp.

### `apply_plan`

Arguments:

- `plan_id`
- `idempotency_key`

The apply step revalidates the kube context and target object identity/version before mutating anything.

### `job_get`, `job_cancel`

Inspect or cancel apply jobs.

## Config

Optional `config.toml`:

```toml
profile = "homelab"
kube_context = "admin@homelab"
linstor_cluster = "homelab"
controller_mode = "port-forward"
controller_service = "linstor-controller"
controller_namespace = "piraeus-datastore"
tls_ca_secret = "linstor-api-tls"
tls_client_secret = "linstor-client-tls"
```

Precedence is `flags > env > config file > built-in profile defaults`.

Important default for `homelab`:

- the server opens its own Kubernetes port-forward to `svc/linstor-controller:3371`
- controller TLS trust is loaded from `linstor-api-tls`
- client certs are loaded from `linstor-client-tls`

## Development Workflow

Standard commands:

```bash
make check
make check-full
make test
make test-race
make coverage
make fuzz-smoke
```

Contributor guidance lives in [CONTRIBUTING.md](/Users/ntbc/workspace/linstor-mcp-server/CONTRIBUTING.md).

## Repository Policy

- dependency versions are pinned in `go.mod` and `go.sum`
- the exact local Go toolchain is pinned with `toolchain go1.26.2`
- `make check` runs formatting, vetting, module verification, tests, and build
- `make check-full` adds race, coverage, and fuzz smoke coverage
- this repository is licensed under MIT; see [LICENSE](/Users/ntbc/workspace/linstor-mcp-server/LICENSE)
- collaboration expectations are documented in [CODE_OF_CONDUCT.md](/Users/ntbc/workspace/linstor-mcp-server/CODE_OF_CONDUCT.md)

## Safety Model

- Every write is two-step: `plan_cluster_config` then `apply_plan`.
- Plans expire after 5 minutes.
- `apply_plan` requires an `idempotency_key`.
- Apply-time revalidation checks kube context plus object identity/version.
- Deletes, live evacuations, failover, and mutations of existing CSI-backed resources are intentionally out of GA scope.
- `piraeus.io/v1` CRDs are the desired-state source of truth.
- The LINSTOR controller API is the runtime-state source of truth.
- `internal.linstor.linbit.com` resources are diagnostics only.

## Reliability Notes

- plans and jobs are persisted in SQLite under the local state directory
- repeated `apply_plan` calls with the same `idempotency_key` reuse the existing job
- stale plans fail closed rather than attempting a best-effort write
- parser-heavy paths have unit and fuzz smoke coverage for cursor and resource URI handling
