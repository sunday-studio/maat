# Implementation Plan

This plan keeps implementation small while preserving the target architecture.

## Phase 0: Architecture And Conventions

- Record repository and commit conventions in `AGENTS.md`.
- Document architecture, storage, search, CLI/TUI/UI, and agent protocol.
- Keep Git plus Markdown as the source of truth.

## Phase 1: Core Parser And Validator

- Parse the target Markdown object layout.
- Validate required fields, statuses, timestamps, and object links.
- Detect duplicate IDs.
- Detect malformed events.
- Support the existing flat v0 files enough to migrate or inspect them.

Current bootstrap status:

- legacy flat project parsing exists for `state/projects/*.md`
- known status validation exists for project and goal state
- status totals can be computed from parsed projects
- target object layout parsing remains future work

## Phase 2: CLI Read Path

- `matt init`
- `matt storage link`
- `matt index rebuild`
- `matt projects`
- `matt project show`
- `matt status`
- `matt search` with SQLite FTS

Current bootstrap status:

- `matt init` and `matt storage link` write local config
- `matt index rebuild` writes `.maat/index.json` as a temporary rebuildable index
- `matt projects`, `matt project show`, `matt status`, and direct Markdown `matt search` work
- SQLite FTS remains future work

## Phase 3: CLI Write Path

- `matt project link`
- `matt goal create`
- `matt ticket create`
- `matt ticket claim`
- `matt ticket comment`
- `matt ticket complete`
- `matt sync`

Each write creates Markdown objects/events, validates, indexes, and commits.

## Phase 4: Bubble Tea TUI

- Project list.
- Ticket list.
- Ticket detail.
- Search view.
- Timeline view.
- Sync/index status view.

The TUI should call the same core operations as the CLI.

## Phase 5: Local Web UI

- Launch with `matt ui`.
- Read from SQLite.
- Use core operations for mutations.
- Include projects, tickets, timeline, blocked/stale views, decisions, and search.

## Phase 6: MCP Adapter

- Expose typed agent tools.
- Use the same validation and write flow as the CLI.
- Keep the Markdown store authoritative.

## Phase 7: Advanced Search

- Add optional embeddings.
- Store vector metadata in SQLite.
- Support semantic search fallback to FTS.
- Add result ranking tuned for agent retrieval.

## First Useful Version

The smallest useful build is:

- installable `matt` binary
- storage link
- index rebuild
- project list/show
- ticket create/comment/complete
- search
- sync

The TUI and web UI can follow after the data model and CLI write path feel stable.
