# Security Policy

## Supported Versions

Only the latest release receives security fixes. Older versions are not maintained.

## Reporting a Vulnerability

Please use [GitHub Private Vulnerability Reporting](https://github.com/Nosmoht/linstor-mcp-server/security/advisories/new) to report security issues privately. Do not open a public issue.

**Response timeline:** Best effort. Expect an initial response within a week. This is an open-source project maintained by a single person in spare time, so response speed is not guaranteed.

## Scope

In scope:
- Bugs in `linstor-mcp-server` that bypass or weaken the staged `plan_cluster_config` -> `apply_plan` safety model
- Vulnerabilities in the Go code that could allow privilege escalation or unauthorized cluster mutation beyond the configured Kubernetes and LINSTOR credentials
- Supply chain issues involving release artifacts, npm packages, or the build pipeline

Out of scope:
- Vulnerabilities in LINSTOR, the Piraeus operator, Kubernetes, or upstream client libraries themselves
- Vulnerabilities in the MCP client (Claude Code, Codex, and others)
- Behavior that assumes the host running `linstor-mcp-server` is already compromised

## Security Model Summary

See [Safety Model](README.md#safety-model) in the README for the public description of trust boundaries, staged writes, stale-plan refusal, and intentionally unsupported operations.
