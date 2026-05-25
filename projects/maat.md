# Project: Maat

| Field | Value |
|---|---|
| ID | maat |
| Status | active |
| Owner | agents |
| Updated | 2026-05-25 |
| Tags | #agent-run #git-native #product |

## Current

Maat is being established as a Git-native Markdown project management system for agent-run work. The first version defines the repository structure, agent operating instructions, current-state project files, and an append-only history model.

The system is intentionally file-first: agents should be able to clone the repo, update Markdown, create events, and commit without needing a separate authoritative database or hosted app.

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

### G-002: Design the installable Maat architecture

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-25 |
| Tags | #architecture #cli #search #docs |

#### Tasks

- [x] T-001: Capture repository and commit conventions from Orion and Aether.
- [x] T-002: Document the durable Git and Markdown storage architecture.
- [x] T-003: Document the SQLite search and indexing architecture.
- [x] T-004: Document the CLI, Bubble Tea TUI, local web UI, and agent protocol.
- [x] T-005: Document the phased implementation plan.

### G-003: Build the first usable `matt` CLI slice

| Field | Value |
|---|---|
| Status | active |
| Updated | 2026-05-25 |
| Tags | #cli #search #go |

#### Tasks

- [x] T-001: Add the Go module and `cmd/matt` entrypoint.
- [x] T-002: Parse and validate legacy flat project Markdown files.
- [x] T-003: Add `projects`, `project show`, `status`, and `search` read commands.
- [x] T-004: Add a rebuildable bootstrap index command.
- [ ] T-005: Replace the bootstrap JSON index with SQLite FTS.
- [ ] T-006: Add target object layout parsing for project directories, goals, tickets, and event files.

## Blockers

- None.

## Decisions

- Use Markdown files as the source of truth rather than a database.
- Use Git commits plus append-only event files for transactional history.
- Keep optional adapters, dashboards, MCP servers, and CLIs layered on top of the Markdown core.

## Links

- [Agent instructions](../AGENTS.md)
- [Schema](../docs/schema.md)
- [Workflows](../docs/workflows.md)
- [Development](../docs/development.md)
