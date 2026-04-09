---
name: repo-safety
description: Safety and authority boundaries for linstor-mcp-server.
last_refreshed: 2026-04-09
---

- `piraeus.io/v1` CRDs are the desired-state source of truth.
- The LINSTOR controller API is the runtime-state source of truth.
- `internal.linstor.linbit.com` resources are diagnostic-only mirrors.
- Never mutate `LinstorSatellite`; the operator creates it by merging matching `LinstorSatelliteConfiguration` resources.
- Existing CSI-backed production resources are read-only by default.
- Fail closed on stale state, missing preconditions, or unclear ownership.
