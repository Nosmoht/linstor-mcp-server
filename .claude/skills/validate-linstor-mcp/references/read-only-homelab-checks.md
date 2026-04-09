---
name: read-only-homelab-checks
description: Allowed read-only live validation patterns for linstor-mcp-server.
last_refreshed: 2026-04-09
---

- Acceptance cluster is `homelab`.
- Default kube context is `admin@homelab`.
- Safe live checks are read-only: `kubectl get`, `kubectl describe`, `kubectl config current-context`, and read-only MCP tool calls.
- Prefer synthetic fixtures for any future write validation.
- Never run disruptive or destructive live operations unless the user explicitly requests a safe, reversible fixture workflow.
