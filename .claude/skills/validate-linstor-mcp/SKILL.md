---
name: validate-linstor-mcp
description: Validate linstor-mcp-server changes with repo checks first and optional read-only homelab checks second. Use after edits to Go code, MCP surface, config logic, or docs in this repo.
argument-hint: "[package-or-focus]"
disable-model-invocation: true
allowed-tools: Bash, Read, Grep, Glob
---

# Validate LINSTOR MCP

You are a validation workflow for this repository. Your job is to run the smallest useful validation set first, expand only when needed, and keep live-cluster activity read-only.

Read `AGENTS.md`, `CONTRIBUTING.md`, `Makefile`, and `references/read-only-homelab-checks.md` before running commands.

## Workflow

### 1. Choose the smallest useful local checks

Default order:
1. targeted `go test` for the touched package
2. `go test ./...`
3. `make check`

If the change is limited to docs or Claude artifacts, use lightweight structural validation instead of code checks.

### 2. Match checks to change type

- For planner, store, config, inventory, or parsing logic: run the focused package tests first.
- For public tool or resource changes: run `go test ./...`, then `make check` unless the change is obviously documentation-only.
- For doc-only changes: verify referenced commands, paths, and names still exist.

### 3. Optional read-only homelab checks

Only if the user asks for live validation or the change directly affects live assumptions, run read-only checks such as:
- `kubectl config current-context`
- `kubectl get` on relevant Piraeus resources
- read-only MCP calls like `validate_environment`

Keep any live check scoped and read-only.

### 4. Report exactly what was validated

Return:
- commands run
- pass or fail status
- the first meaningful failure if something broke
- gaps that remain unvalidated

## Hard Rules

- Never run `apply_plan` or any other live cluster mutation from this skill.
- Never use this skill to commit changes.
- Do not claim `make check` passed unless it was actually run.
- Keep live homelab validation read-only.
