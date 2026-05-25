# Project: Maat

| Field | Value |
|---|---|
| ID | maat |
| Status | active |
| Owner | agents |
| Updated | 2026-05-25 |
| Tags | #agent-run #git-native #product |

## Current

Maat is being established as a Git-native Markdown project management system for agent-run work. The first version defines the repository structure, agent operating instructions, current-state project files, and an append-only monthly ledger.

The system is intentionally file-first: agents should be able to clone the repo, update Markdown, append ledger events, and commit without needing a separate database or web app.

## Goals

### G-001: Define the agent-operable project system

| Field | Value |
|---|---|
| Status | active |
| Updated | 2026-05-25 |
| Tags | #agent-run #git-native #docs |

#### Tasks

- [x] T-001: Define the core repository structure.
- [x] T-002: Add instructions for agents.
- [x] T-003: Add project and ledger templates.
- [x] T-004: Register active external projects after reading their current state.
- [ ] T-005: Add validation or automation once the Markdown workflow stabilizes.

## Blockers

- None.

## Decisions

- Use Markdown files as the source of truth rather than a database.
- Use Git commits plus an append-only ledger for transactional history.
- Keep optional adapters, dashboards, MCP servers, and CLIs layered on top of the Markdown core.

## Links

- [Agent instructions](../AGENTS.md)
- [Schema](../docs/schema.md)
- [Workflows](../docs/workflows.md)
