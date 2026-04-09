# AGENTS.md

This file is the canonical project instruction set for coding agents working in
this repository. Keep it concise, current, and reviewable. If another project
document conflicts with this file, follow this file.

## Purpose

- Build and maintain a safe, production-oriented Go MCP server for LINSTOR and
  the Piraeus operator.
- Prefer reliable, reviewable changes over broad speculative implementation.
- Treat this repository as infrastructure software: safety, reproducibility, and
  clear operator-facing behavior matter more than speed.

## Primary Sources Of Truth

- Read these first when they are relevant:
  - `README.md` for supported scope and user-facing behavior
  - `CONTRIBUTING.md` for development and validation expectations
  - `Makefile` for standard commands
  - `go.mod` and `go.sum` for pinned dependencies
- For volatile or current facts, do not rely on memory alone. Verify with
  official documentation, primary sources, or the live environment before
  stating or encoding assumptions.

## Project Facts

- Language: Go
- Primary transport: `stdio`
- Beta transport: Streamable HTTP behind feature flag
- Acceptance cluster: `homelab`
- Default kube context: `admin@homelab`
- Desired-state source of truth: `piraeus.io/v1` CRDs
- Runtime-state source of truth: LINSTOR controller API
- Diagnostic-only source: `internal.linstor.linbit.com`

## Hard Safety Rules

- Never mutate `LinstorSatellite`.
- Do not treat `internal.linstor.linbit.com` resources as authoritative write
  targets.
- Do not expose hashed internal resource names as the public contract.
- Protect existing CSI-backed production resources by default.
- Do not run destructive or disruptive live-cluster operations on `homelab`
  unless the user explicitly requests it and the action is scoped to a safe,
  reversible fixture.
- Prefer synthetic fixtures such as `mcp-e2e-*` for live write validation.
- Fail closed on stale state, missing preconditions, or unclear ownership.

## Required Delivery Workflow

For anything beyond a trivial edit, follow this lifecycle by default:

1. Deconstruct
   - Restate the task in concrete technical terms.
   - Identify constraints, risks, affected components, and unknowns.
   - Check existing code and docs before proposing changes.
2. Spec
   - Produce a concise implementation spec before coding.
   - Include scope, non-goals, interfaces, data flow, failure modes, and
     acceptance criteria.
   - For small tasks, this can be a compact internal or user-visible summary.
3. Spec review
   - If the change is high-risk, ambiguous, cross-cutting, destructive, or
     architecture-shaping, get user confirmation before implementation.
   - If the user explicitly asks to proceed immediately and the risk is low,
     continue after doing the spec step internally.
4. Plan
   - Break the work into concrete execution steps.
   - Include validation strategy before editing files.
5. Implement
   - Make the smallest coherent change set that satisfies the spec.
   - Preserve unrelated user changes.
   - Prefer incremental, reviewable patches over wide refactors.
6. Review and fix loop
   - Review your own diff critically for correctness, regressions, safety,
     clarity, and missing tests.
   - Fix problems before finalizing.
7. Test
   - Run targeted tests first, then broader checks when justified.
   - Do not claim success without reporting what was actually validated.
8. Document
   - Update user and contributor docs when behavior, workflow, scope, setup, or
     support policy changes.
9. Final review
   - Review the complete change as a cohesive deliverable, not just file-level
     edits.
   - Confirm the result matches the original spec and acceptance criteria.
10. Commit
   - When a commit is requested or part of the task, use scoped Conventional
     Commits.

## Spec Requirements

When writing or presenting a spec, include:

- Problem statement
- Scope
- Explicit non-goals
- User-visible behavior changes
- Internal design or interface changes
- Safety and rollback considerations
- Validation plan
- Acceptance criteria

## Planning Requirements

- Name the files and subsystems likely to change.
- Prefer one clear path over many options unless tradeoffs are material.
- Call out blockers early.
- If using multiple agents or subagents, keep ownership boundaries explicit and
  non-overlapping.

## Implementation Rules

- Use ASCII unless the file already requires Unicode.
- Follow existing repository patterns before introducing new abstractions.
- Add comments only when they explain intent or non-obvious behavior.
- Avoid speculative helpers and one-off abstractions.
- Prefer simple, explicit code over cleverness.
- Keep APIs self-describing; avoid boolean flags that make call sites opaque.
- Do not silently broaden scope.

## Testing And Validation

- Standard commands:
  - `make check`
  - `make check-full`
  - `make build`
  - `make test`
  - `make test-race`
  - `make coverage`
  - `make fuzz-smoke`
  - `make verify`
- Minimum expectation for code changes:
  - relevant unit or regression tests
  - `make check` passing
- Prefer this validation order:
  1. focused unit tests
  2. package-level tests
  3. repo-wide checks
  4. live read-only smoke validation
  5. live write validation only for explicit safe fixtures
- If tests cannot be run, say exactly why.

## Documentation Rules

Update `README.md` when changing:

- supported scope
- setup steps
- tool surface
- safety guarantees
- supported versions

Update `CONTRIBUTING.md` when changing:

- development workflow
- validation expectations
- repository policy
- contribution safety constraints

## Research Rules

- For anything described as current, latest, recommended, supported, or best
  practice, verify with current official sources first.
- Prefer primary sources:
  - OpenAI docs and official OpenAI engineering posts for Codex behavior
  - Anthropic / Claude Code docs for Claude behavior
  - official upstream docs, specs, or source repositories for dependencies
- Make date-sensitive statements with concrete versions or dates when possible.

## Agent Tooling Guidance

- Codex uses `AGENTS.md` as project guidance. Keep this file practical:
  commands, safety rules, workflow, architecture constraints, and review
  expectations.
- Claude Code uses `CLAUDE.md` project memory, not `AGENTS.md`. Keep
  `CLAUDE.md` aligned with this file so both tools follow the same workflow.
- Keep persistent instruction files concise. Claude recommends targeting under
  200 lines for `CLAUDE.md`; avoid bloated agent instructions.

## Review Standard

Every substantial change should be reviewed for:

- correctness
- safety
- ownership boundary violations
- regressions
- missing or weak tests
- documentation drift
- unnecessary complexity

When asked for a review, findings come first, ordered by severity, with file
references and concrete remediation.

## Commit Policy

Use scoped Conventional Commits when committing:

- Format: `<type>(<scope>): <description>`
- Examples:
  - `feat(mcp): add apply-plan idempotency checks`
  - `fix(kube): handle missing default storage class`
  - `docs(readme): clarify homelab validation flow`
  - `test(app): add stale-plan regression coverage`
  - `refactor(inventory): simplify canonical URI parsing`

Commit rules:

- Prefer multiple small coherent commits over one mixed commit when the work has
  separable intent.
- Use the commit body for rationale, risk, or validation when useful.
- Do not hide breaking behavior; mark it explicitly.
- Do not commit unrelated changes.

## Done Criteria

A change is complete only when all of these are true:

- the implementation matches the spec
- the diff has been self-reviewed
- appropriate tests or checks were run
- documentation was updated if needed
- risks, limitations, and unvalidated areas are disclosed
- if committing, the history uses scoped Conventional Commits
