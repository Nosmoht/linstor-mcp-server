---
name: linstor-mcp-guardrails
description: Repository-specific guardrails and domain context for linstor-mcp-server. Use when working on LINSTOR MCP tools, Piraeus CRDs, inventory or planner logic, transport behavior, homelab validation, or contributor workflow updates in this repo.
user-invocable: false
allowed-tools: Read, Grep, Glob
---

# LINSTOR MCP Guardrails

You are a repository guardrails skill that supplies the non-obvious rules for this codebase. Your job is to keep changes aligned with the repo's safety model, authority boundaries, and validation workflow before implementation starts.

## When To Apply This Skill

Apply this context when the task touches:
- MCP tool or resource behavior
- Kubernetes or LINSTOR client logic
- planner, apply, stale-plan, or idempotency semantics
- Streamable HTTP behavior
- contributor docs, safety rules, or validation workflow

Load supporting references only when needed:
- `references/repo-safety.md` for authority boundaries and mutation constraints
- `references/tool-surface.md` for current GA scope and transport/tool expectations
- `references/validation-flow.md` for test and documentation expectations

## Workflow

### 1. Re-anchor on repository truth

Read `AGENTS.md` first for the canonical shared contract.

Then read the smallest relevant subset of:
- `README.md` for user-visible scope and current tool surface
- `CONTRIBUTING.md` for validation expectations
- `Makefile` for standard commands
- the touched Go package files

### 2. Enforce the authority split

Before proposing or making changes, classify the target behavior:
- desired state belongs to `piraeus.io/v1` resources
- runtime state belongs to the LINSTOR controller API
- `internal.linstor.linbit.com` resources are diagnostics only

If a task crosses those boundaries, keep the public contract anchored to the authoritative source and treat mirror-only data as informational.

### 3. Preserve the safety model

Keep these invariants intact:
- never mutate `LinstorSatellite`
- do not treat hashed internal resource names as the user-facing contract
- protect existing CSI-backed production resources by default
- keep write flows as `plan_cluster_config` then `apply_plan`
- stale or changed targets fail closed instead of applying best-effort writes

### 4. Match validation effort to blast radius

For code changes:
- add focused tests for logic changes
- run targeted tests first, then broader checks when justified
- update docs if scope, setup, safety guarantees, or workflow changes

For current or version-sensitive claims:
- prefer `kb-server` first if available
- then verify with current official sources only

## Hard Rules

- Do not normalize repo safety rules away for convenience.
- Do not broaden live-cluster mutation scope unless the user explicitly asks for a safe, reversible fixture workflow.
- Do not claim validation you did not actually run.
- Do not duplicate long-lived guardrails into `CLAUDE.md` when a skill or repo doc can carry them more cleanly.
