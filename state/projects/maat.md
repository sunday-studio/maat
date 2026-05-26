# Project: Maat

| Field | Value |
|---|---|
| ID | maat |
| Status | done |
| Owner | agents |
| Updated | 2026-05-26 |
| Tags | #agent-run #git-native #product |

## Current

Maat is being established as a Git-native Markdown project management system for agent-run work. The first version defines the repository structure, agent operating instructions, the Maat project state file, and an append-only history model.

The system is intentionally file-first: agents should be able to clone the repo, update Markdown, create events, and commit without needing a separate authoritative database or hosted app.

The implementation now has a usable read CLI, validation, SQLite-backed search/indexing, target object parsing, write-path core operations, Git sync primitives, migration planning, a first Bubble Tea dashboard, and local install documentation.

The latest cleanup moved all Maat-managed Markdown state under `state/` so projects, ledger, decisions, reports, agents, tags, and templates are no longer scattered across the repository root.

The local installer now gives visible step-by-step progress, supports ANSI color control, and ends with a `maat ready to use` banner plus starter commands.

## Goals

### G-001: Define the agent-operable project system

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-25 |
| Tags | #agent-run #git-native #docs |

#### Tasks

- [x] T-001: Define the core repository structure.
- [x] T-002: Add instructions for agents.
- [x] T-003: Add project and ledger templates.
- [x] T-004: Keep this repository scoped to Maat project state only.
- [ ] T-005: Add validation or automation once the Markdown workflow stabilizes.

### G-002: Design the installable Maat architecture

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-25 |
| Tags | #architecture #cli #search #docs |

#### Tasks

- [x] T-001: Capture repository and commit conventions for Maat agents.
- [x] T-002: Document the durable Git and Markdown storage architecture.
- [x] T-003: Document the SQLite search and indexing architecture.
- [x] T-004: Document the CLI, Bubble Tea TUI, local web UI, and agent protocol.
- [x] T-005: Document the phased implementation plan.

### G-003: Build the first usable `matt` CLI slice

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-25 |
| Tags | #cli #search #go |

#### Tasks

- [x] T-001: Add the Go module and `cmd/matt` entrypoint.
- [x] T-002: Parse and validate legacy flat project Markdown files.
- [x] T-003: Add `projects`, `project show`, `status`, and `search` read commands.
- [x] T-004: Add a rebuildable bootstrap index command.
- [x] T-005: Replace the bootstrap JSON index with SQLite FTS.
- [x] T-006: Add target object layout parsing for project directories, goals, tickets, and event files.

### G-004: Parallelize the next implementation tracks

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-25 |
| Tags | #agent-run #planning #cli |

#### Tasks

- [x] T-001: Group the remaining implementation work into independent tracks.
- [x] T-002: Assign SQLite indexing to a worker agent.
- [x] T-003: Assign target storage parsing to a worker agent.
- [x] T-004: Assign validation improvements to a worker agent.
- [x] T-005: Assign ID and event helper foundations to a worker agent.
- [x] T-006: Integrate worker commits after review.

### G-005: Wire the next product slices

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-25 |
| Tags | #agent-run #cli #tui #sync |

#### Tasks

- [x] T-001: Wire validation, SQLite search, index rebuild, and JSON query output into the CLI.
- [x] T-002: Add write-path core operations for projects, goals, tickets, claims, comments, and completion events.
- [x] T-003: Add Git sync primitives for repository detection, status, pull, commit, and push.
- [x] T-004: Add migration core from legacy flat files to target object layout.
- [x] T-005: Add the first Bubble Tea TUI skeleton.
- [x] T-006: Add local install and distribution documentation.
- [x] T-007: Integrate and verify all worker commits.

### G-006: Wire agent-facing workflows

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-25 |
| Tags | #agent-run #cli #sync #tui |

#### Tasks

- [x] T-001: Add CLI write commands for goals and tickets.
- [x] T-002: Add sync orchestration internals for validate, index, commit, and optional push.
- [x] T-003: Add migration plan/apply CLI commands.
- [x] T-004: Improve the TUI with project selection and detail view.
- [x] T-005: Package the agent protocol snippet for other repos.
- [x] T-006: Integrate and verify the workflow worker commits.

### G-007: Finish first agent-operable CLI loop

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-25 |
| Tags | #agent-run #cli #tui #migration |

#### Tasks

- [x] T-001: Wire the `matt sync` CLI command.
- [x] T-002: Wire the `matt agent instructions` CLI command.
- [x] T-003: Improve write command human and JSON output.
- [x] T-004: Improve the TUI ticket/search surface.
- [x] T-005: Dogfood migration into a temporary destination and document safety.
- [x] T-006: Integrate and verify this worker round.

### G-008: Reduce agent command friction

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-25 |
| Tags | #cli #agent-run #git |

#### Tasks

- [x] T-001: Add `matt project link` for source repo/path registration.
- [x] T-002: Infer project key and display name from Git remote or source directory.
- [x] T-003: Infer project automatically for write commands when run from a linked repo.
- [ ] T-004: Add source path aliases or repo identity records beyond project identity metadata.
- [x] T-005: Show target-layout linked projects with `matt project show`.

### G-009: Package the first usable release shape

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-25 |
| Tags | #release #cli #docs #tui |

#### Tasks

- [x] T-001: Fill remaining CLI query gaps, including ticket list/show and project JSON output.
- [x] T-002: Harden storage/search state queries.
- [x] T-003: Improve TUI read views.
- [x] T-004: Add build and release setup.
- [x] T-005: Rewrite README as a simple current product guide.
- [x] T-006: Integrate and verify this packaging worker round.

### G-010: Keep implementation surface tidy

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-26 |
| Tags | #cleanup #go |

#### Tasks

- [x] T-001: Remove unused Git wrapper helpers superseded by `GitSync` methods.
- [x] T-002: Remove the stale project ID prefix from generated ID validation.
- [x] T-003: Re-run tests and vet after cleanup.

### G-011: Scope this repository to Maat only

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-26 |
| Tags | #cleanup #storage #docs |

#### Tasks

- [x] T-001: Remove unrelated project files from `state/projects/`.
- [x] T-002: Remove cross-project registration events from the legacy ledger.
- [x] T-003: Replace public examples and test fixtures that used real project names.
- [x] T-004: Verify Maat now validates as a one-project store.

### G-012: Group reusable templates

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-26 |
| Tags | #cleanup #docs #templates |

#### Tasks

- [x] T-001: Move agent, project, and ledger event templates into `state/templates/`.
- [x] T-002: Add a template directory index.
- [x] T-003: Update agent, project, and ledger docs to point at the new template paths.

### G-013: Improve installer first-run feedback

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-26 |
| Tags | #release #cli #install |

#### Tasks

- [x] T-001: Add visible installer progress steps.
- [x] T-002: Add an ANSI completion banner that says `maat ready to use`.
- [x] T-003: Print practical starter commands after install.
- [x] T-004: Document installer color and post-install output.

### G-014: Consolidate Markdown state layout

| Field | Value |
|---|---|
| Status | done |
| Updated | 2026-05-26 |
| Tags | #cleanup #storage #docs |

#### Tasks

- [x] T-001: Move agents, decisions, ledger, projects, reports, tags, and templates under `state/`.
- [x] T-002: Keep product documentation in `docs/` and root onboarding files at the repo root.
- [x] T-003: Teach the core loaders and writers to use `state/` when present while preserving legacy root-layout compatibility.
- [x] T-004: Add tests for state-layout project loading and writes.

## Blockers

- None.

## Decisions

- Use Markdown files as the source of truth rather than a database.
- Use Git commits plus append-only event files for transactional history.
- Keep optional adapters, dashboards, MCP servers, and CLIs layered on top of the Markdown core.
- Keep the JSON bootstrap index temporarily while SQLite FTS lands behind the core API.
- Use SQLite-backed search as the CLI default, with direct Markdown search as a fallback.
- Keep migration apply non-destructive by writing target-layout files to a destination path instead of rewriting legacy files.
- Do not run in-place migration until generated IDs and target object ergonomics are reviewed after dogfood.
- Let `matt project link` be idempotent so agents can safely run it before project work.
- Let `goal create` and `ticket create` infer the project when the current directory is inside a linked source path.
- Keep release artifacts under `dist/` and build them with `make build` or `make release`.

## Links

- [Agent instructions](../../AGENTS.md)
- [Schema](../../docs/schema.md)
- [Workflows](../../docs/workflows.md)
- [Development](../../docs/development.md)
- [Work plan](../../docs/work-plan.md)
