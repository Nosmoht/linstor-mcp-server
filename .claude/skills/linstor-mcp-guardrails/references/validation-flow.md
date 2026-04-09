---
name: validation-flow
description: Repo validation and documentation expectations for changes in linstor-mcp-server.
last_refreshed: 2026-04-09
---

- Standard commands live in `Makefile`: `make build`, `make test`, `make test-race`, `make coverage`, `make fuzz-smoke`, `make verify`, `make check`, `make check-full`.
- Minimum expectation for code changes is relevant unit or regression tests plus `make check` when justified by scope.
- Prefer focused tests before repo-wide checks.
- Update `README.md` for user-visible behavior, setup, scope, or safety changes.
- Update `CONTRIBUTING.md` for workflow, validation, or repo policy changes.
