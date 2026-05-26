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

Current status:

- legacy flat project parsing exists for `projects/*.md`
- target object layout parsing exists for `projects/<project-key>/`
- validation checks required fields, known statuses, timestamps, duplicate IDs, object links, malformed tables, and event paths
- status totals can be computed from legacy and object-layout projects

## Phase 2: CLI Read Path

- `maat setup --storage <absolute-git-repo-path>`
- `maat index rebuild`
- `maat projects`
- `maat project show`
- `maat status`
- `maat search` with SQLite FTS

Current status:

- `maat setup` writes local config
- `maat index rebuild` writes rebuildable `.maat/index.json` and `.maat/index.sqlite` indexes
- `maat projects`, `maat project show`, `maat status`, and direct Markdown `maat search` work
- SQLite FTS is wired with direct Markdown search as a fallback

## Phase 3: CLI Write Path

- `maat project link`
- `maat goal create`
- `maat ticket create`
- `maat ticket claim`
- `maat ticket comment`
- `maat ticket complete`
- `maat sync`

Each write creates Markdown objects/events, validates, indexes, and commits.

## Phase 4: Bubble Tea TUI

- Project list.
- Ticket list.
- Ticket detail.
- Search view.
- Timeline view.
- Sync/index status view.

The TUI should call the same core operations as the CLI.

## Phase 5: Future Local Web UI

- Add a local dashboard launched from the `maat` binary.
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

- installable `maat` binary
- setup storage
- index rebuild
- project list/show
- ticket create/comment/complete
- search
- sync

The TUI and web UI can follow after the data model and CLI write path feel stable.
