---
name: plan-linstor-mcp-change
description: Plan a linstor-mcp-server change before implementation. Use when scoping MCP tool changes, planner/apply behavior, inventory semantics, transport work, config changes, or contributor workflow updates in this repo.
argument-hint: "[change description]"
disable-model-invocation: true
allowed-tools: Read, Grep, Glob, Agent, WebFetch
---

# Plan LINSTOR MCP Change

You are a planning workflow that produces a concrete implementation plan for this repository before code changes begin. Your job is to tie the requested change to the current codebase, repo guardrails, current upstream facts, and the project review workflow.

## Workflow

### 1. Ground in the current repo

Read:
- `AGENTS.md`
- `README.md`
- `CONTRIBUTING.md`
- `Makefile`
- the most relevant touched package files

Load the `linstor-mcp-guardrails` skill context before drafting a plan.

### 2. Clarify the change against current behavior

State:
- what the repo does today
- what the requested change would alter
- which boundaries matter: desired state, runtime state, diagnostics, transport, or safety

Name the specific tools, resources, config keys, or packages involved.

### 3. Refresh volatile facts

If the plan depends on current upstream behavior:
- use `kb-server` first when it is available
- for missing or stale facts, verify with official sources only

Typical current-fact topics in this repo:
- Claude Code skill or subagent behavior
- MCP transport or security details
- Piraeus CRD semantics
- LINSTOR controller API behavior

### 4. Ask the repo reviewers for focused checks

When the change is non-trivial, call the project agents:
- `linstor-safety-reviewer` for safety and ownership boundaries
- `mcp-contract-reviewer` for public tool/resource/transport contract changes
- `upstream-doc-researcher` when the plan depends on current upstream facts

Use only the reviewers that materially reduce risk for the task.

### 5. Produce the implementation plan

The plan should include:
- problem statement
- exact scope and explicit non-goals
- interfaces or user-visible behavior changes
- failure modes and rollback considerations
- tests to add or update
- documentation updates required

## Hard Rules

- Do not skip repo inspection and jump straight to a plan from memory.
- Do not rely on `internal.linstor.linbit.com` diagnostics as the public contract.
- Do not propose direct `LinstorSatellite` mutation.
- Do not present a vague plan that leaves major decisions to the implementer.
