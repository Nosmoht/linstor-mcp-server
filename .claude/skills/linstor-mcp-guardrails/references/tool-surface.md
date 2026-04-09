---
name: tool-surface
description: Current public MCP surface and transport expectations for linstor-mcp-server.
last_refreshed: 2026-04-09
---

- Primary transport is `stdio`.
- Streamable HTTP exists behind `--enable-http-beta`.
- GA tools are `validate_environment`, `inventory_list`, `inventory_get`, `plan_cluster_config`, `apply_plan`, `job_get`, and `job_cancel`.
- Public write scope is limited to `LinstorCluster`, `LinstorSatelliteConfiguration`, and `LinstorNodeConnection`.
- `apply_plan` requires an `idempotency_key` and revalidates target identity before mutating anything.
