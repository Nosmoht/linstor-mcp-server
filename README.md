# LINSTOR MCP Server

[![CI](https://github.com/Nosmoht/linstor-mcp-server/actions/workflows/ci.yml/badge.svg)](https://github.com/Nosmoht/linstor-mcp-server/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Nosmoht/linstor-mcp-server?sort=semver)](https://github.com/Nosmoht/linstor-mcp-server/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/ntbc/linstor-mcp-server.svg)](https://pkg.go.dev/github.com/ntbc/linstor-mcp-server)
[![codecov](https://codecov.io/gh/Nosmoht/linstor-mcp-server/graph/badge.svg)](https://codecov.io/gh/Nosmoht/linstor-mcp-server)
[![Go Report Card](https://goreportcard.com/badge/github.com/ntbc/linstor-mcp-server)](https://goreportcard.com/report/github.com/ntbc/linstor-mcp-server)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/Nosmoht/linstor-mcp-server/badge)](https://scorecard.dev/viewer/?uri=github.com/Nosmoht/linstor-mcp-server)
[![License](https://img.shields.io/github/license/Nosmoht/linstor-mcp-server)](LICENSE)

`linstor-mcp-server` is a Go MCP server for LINSTOR and the Piraeus operator. It exposes canonical inventory reads plus a staged cluster-configuration write flow designed for stateful infrastructure: plan first, review the diff, then apply only while the target state is still fresh.

Desired state comes from `piraeus.io/v1` CRDs. Runtime state comes from the LINSTOR controller API.

## Installation

**Via npm** (no Go required, Linux/macOS, amd64/arm64):

```bash
npx linstor-mcp
```

Note: npm and binary installs become available only after the automated release pipeline has published the first tagged release.

**Download binary** (Linux/macOS, amd64/arm64):

Download the latest release from [GitHub Releases](https://github.com/Nosmoht/linstor-mcp-server/releases), extract it, and place `linstor-mcp-server` in your `$PATH`.

**Build from source** (requires Go `1.26.2`):

```bash
git clone https://github.com/Nosmoht/linstor-mcp-server
cd linstor-mcp-server
make build
```

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

## Configuration

The built-in `homelab` profile is the default. Configuration precedence is:

`flags > env > config file > built-in profile defaults`

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
state_dir = "~/.local/state/linstor-mcp-server"
http_addr = ""
enable_http_beta = false
log_format = "text"
```

Environment overrides:

| Variable | Default | Description |
|---|---|---|
| `LINSTOR_MCP_CONFIG` | unset | Path to `config.toml` |
| `LINSTOR_MCP_PROFILE` | `homelab` | Configuration profile name |
| `LINSTOR_MCP_KUBE_CONTEXT` | from profile | Kubernetes context override |
| `LINSTOR_MCP_LINSTOR_CLUSTER` | from profile | LINSTOR cluster name override |
| `LINSTOR_MCP_CONTROLLER_MODE` | `port-forward` | Controller access mode |
| `LINSTOR_MCP_CONTROLLER_URL` | unset | Direct controller URL override |
| `LINSTOR_MCP_CONTROLLER_SERVICE` | `linstor-controller` | Controller service name |
| `LINSTOR_MCP_CONTROLLER_NAMESPACE` | `piraeus-datastore` | Controller namespace |
| `LINSTOR_MCP_TLS_CA_SECRET` | `linstor-api-tls` | Secret containing controller CA trust |
| `LINSTOR_MCP_TLS_CLIENT_SECRET` | `linstor-client-tls` | Secret containing client cert/key |
| `LINSTOR_MCP_STATE_DIR` | `~/.local/state/linstor-mcp-server` | State directory for plans and jobs |
| `LINSTOR_MCP_HTTP_ADDR` | unset | Beta Streamable HTTP listen address |
| `LINSTOR_MCP_ENABLE_HTTP_BETA` | `false` | Enable beta HTTP transport |
| `LINSTOR_MCP_LOG_FORMAT` | `text` | Log format: `text` or `json` |

Important `homelab` defaults:

- the server opens its own Kubernetes port-forward to `svc/linstor-controller:3371`
- controller TLS trust is loaded from `linstor-api-tls`
- client certs are loaded from `linstor-client-tls`

## Client Setup

### Claude Code

Add to your project's `.mcp.json`:

```json
{
  "mcpServers": {
    "linstor": {
      "command": "npx",
      "args": ["-y", "linstor-mcp"]
    }
  }
}
```

If you prefer a local binary, replace `"command": "npx"` with the path to `linstor-mcp-server`.

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "linstor": {
      "command": "npx",
      "args": ["-y", "linstor-mcp"]
    }
  }
}
```

### OpenAI Codex

Add to `.codex/config.toml` (project) or `~/.codex/config.toml` (global):

```toml
[mcp_servers.linstor]
command = "npx"
args = ["-y", "linstor-mcp"]

[mcp_servers.linstor.env]
LINSTOR_MCP_CONFIG = "/path/to/config.toml"
```

### Generic MCP client

The server speaks the [MCP protocol](https://modelcontextprotocol.io) over stdio:

```bash
./linstor-mcp-server
```

## Tools

### Read-only

| Tool | Description |
|---|---|
| `validate_environment` | Validate Kubernetes access, LINSTOR controller connectivity, default storage class, and core homelab assumptions. |
| `inventory_list` | List canonical inventory across clusters, satellite configurations, node connections, nodes, storage pools, resource definitions, and resources. |
| `inventory_get` | Read one canonical inventory object by semantic kind and ID. |
| `job_get` | Fetch the current status of a previously started apply job. |

### Staged writes

| Tool | Description | Guards |
|---|---|---|
| `plan_cluster_config` | Create a 5-minute plan for `LinstorCluster`, `LinstorSatelliteConfiguration`, or `LinstorNodeConnection` without mutating the cluster. | no live mutation; returns diff, summary, and preconditions |
| `apply_plan` | Apply a previously created plan. | requires `plan_id`, `idempotency_key`, fresh plan, and stale-state revalidation |
| `job_cancel` | Cancel a pending or running apply job if it has not already completed. | no effect once terminal |

## Safety Model

### Trust Boundaries

```text
MCP Client (Claude Code / Codex)
        |  stdio / JSON-RPC
        v
  linstor-mcp-server
        |  Kubernetes API + LINSTOR controller API
        v
Piraeus operator CRDs + LINSTOR runtime state
```

### Safety Mechanisms

| Mechanism | How it works |
|---|---|
| Staged writes | Every write is two-step: `plan_cluster_config` then `apply_plan` |
| Plan expiry | Plans expire after 5 minutes |
| Fresh-state refusal | `apply_plan` revalidates kube context plus target identity/version before mutating anything |
| Idempotency | `apply_plan` requires an `idempotency_key` and reuses the same job record on retries |
| Scoped writes | Mutations are limited to `LinstorCluster`, `LinstorSatelliteConfiguration`, and `LinstorNodeConnection` |
| Protected production resources | Existing CSI-backed production resources are treated as read-only |

### Intentionally Unsupported in GA

- destructive operations
- snapshots, backups, remotes, and schedules
- failover, rebalance, and evacuation
- direct mutation of existing CSI-backed production resources
- mutation of `LinstorSatellite`
- broad HTTP deployment and OAuth flows

### HTTP transport status

Streamable HTTP exists behind `--enable-http-beta` and `LINSTOR_MCP_ENABLE_HTTP_BETA=true`. It is beta-only and currently intended for trusted environments; the repository does not yet define a public auth story for HTTP mode.

## Verifying Downloads

### Checksums (integrity)

Each release includes a `linstor-mcp-server_<version>_checksums.txt` file with SHA-256 hashes of all archives. Verify the binary after downloading:

```bash
curl -LO https://github.com/Nosmoht/linstor-mcp-server/releases/download/v<version>/linstor-mcp-server_<version>_linux_amd64.tar.gz
curl -LO https://github.com/Nosmoht/linstor-mcp-server/releases/download/v<version>/linstor-mcp-server_<version>_checksums.txt

sha256sum --check --ignore-missing linstor-mcp-server_<version>_checksums.txt
```

### GitHub artifact attestations

Each release includes GitHub-native build provenance:

```bash
gh attestation verify linstor-mcp-server_<version>_linux_amd64.tar.gz \
  --repo Nosmoht/linstor-mcp-server
```

### npm package provenance

The npm packages are published with provenance attestation:

```bash
npm audit signatures
```

## Release Automation

This repository is configured for automated semantic versioning and releases.

- merge conventional changes to `main`
- the auto-tag workflow creates the next `v*` tag
- the release workflow publishes GitHub artifacts, attestations, and npm packages

Manual tag creation is not the intended release path.

## Claude Code Helpers

This repo includes project-local Claude Code helpers under `.claude/`.

- `linstor-mcp-guardrails`
  - background knowledge skill
  - not intended for direct invocation
  - keeps Claude aligned with this repo's safety rules, authority boundaries, and validation workflow
- `/plan-linstor-mcp-change`
  - planning workflow for MCP surface, planner/apply, transport, config, and contributor workflow changes
- `/validate-linstor-mcp`
  - validation workflow that prefers repo checks first and only uses read-only homelab checks when live validation is needed

Project subagents:

- `linstor-safety-reviewer`
- `mcp-contract-reviewer`
- `upstream-doc-researcher`

## Development

```bash
make build
make check
make check-full
make coverage
make run
```

Contributor guidance lives in [CONTRIBUTING.md](/Users/ntbc/workspace/linstor-mcp-server/CONTRIBUTING.md).

## License

[MIT](LICENSE)
- The LINSTOR controller API is the runtime-state source of truth.
- `internal.linstor.linbit.com` resources are diagnostics only.

## Reliability Notes

- plans and jobs are persisted in SQLite under the local state directory
- repeated `apply_plan` calls with the same `idempotency_key` reuse the existing job
- stale plans fail closed rather than attempting a best-effort write
- parser-heavy paths have unit and fuzz smoke coverage for cursor and resource URI handling
