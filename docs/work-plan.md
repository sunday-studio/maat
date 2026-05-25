# Work Plan

This is the grouped implementation backlog for Maat.

The plan assumes multiple agents can work in parallel. Each group should keep a clear file ownership boundary and commit its own coherent changes.

## Group 1: Storage And Indexing

Goal: make local query and search fast while keeping Git plus Markdown authoritative.

Tasks:

- Add SQLite database creation and migrations.
- Add FTS search over Markdown documents.
- Keep the existing JSON bootstrap index until SQLite is wired into the CLI.
- Track source file path, hash, title, type, and indexed text.
- Add tests for rebuild and search behavior.
- Later: add optional vector embeddings as a separate layer.

Initial owner:

- Worker A: SQLite/FTS implementation under `internal/maat`.

## Group 2: Target Storage Parser

Goal: support the conflict-resistant target layout from `docs/storage-model.md`.

Tasks:

- Parse `projects/<project-key>/project.md`.
- Parse `goals/*.md`.
- Parse `tickets/*.md`.
- Parse `events/YYYY/MM/*.md`.
- Validate required fields and object links.
- Keep compatibility with the legacy flat `projects/*.md` files until migration exists.

Initial owner:

- Worker B: target object parser under `internal/maat`.

## Group 3: Validation And Store Health

Goal: let agents and humans trust the store before writing or committing.

Tasks:

- Detect duplicate project IDs.
- Detect duplicate goal IDs within a project.
- Detect duplicate ticket IDs within a goal or project.
- Detect invalid status values.
- Detect missing required fields.
- Detect malformed object files.
- Expose a validation API for future `matt validate`.

Initial owner:

- Worker C: validation API and tests under `internal/maat`.

## Group 4: Write Path Foundations

Goal: prepare safe agent writes without rushing CLI commands.

Tasks:

- Generate collision-resistant IDs for goals, tickets, events, decisions, and repos.
- Generate event file paths.
- Render event Markdown from structured input.
- Add tests for ID shape, event paths, and Markdown output.
- Later: wire `matt goal create`, `matt ticket create`, `matt ticket comment`, and `matt ticket complete`.

Initial owner:

- Worker D: ID and event helpers under `internal/maat`.

## Group 5: CLI Commands

Goal: make the binary useful for agents and humans.

Tasks:

- Add `matt validate`.
- Wire SQLite-backed `matt search`.
- Add `matt ticket create`.
- Add `matt ticket comment`.
- Add `matt ticket complete`.
- Add `matt sync` with safe Git flow.
- Add JSON output for query commands.

Suggested owner:

- Main integration agent after Groups 1-4 land.

## Group 6: TUI

Goal: give the human a polished terminal dashboard.

Tasks:

- Add Bubble Tea dependencies.
- Add `matt tui`.
- Show projects, active tickets, blocked tickets, timeline, and search.
- Keep mutations routed through the same core operations as the CLI.

Suggested owner:

- A later UI-focused agent after CLI read/write APIs stabilize.

## Group 7: Local Web UI

Goal: provide a browser dashboard for browsing all project state.

Tasks:

- Add `matt ui`.
- Serve a local dashboard.
- Read from SQLite.
- Use core operations for mutations.
- Show project overview, ticket detail, timeline, search, reports, decisions, and agent activity.

Suggested owner:

- A later frontend/UI agent after the index and core APIs stabilize.

## Group 8: Install And Distribution

Goal: make Maat easy to install on a new machine.

Tasks:

- Add release build commands.
- Add install script design.
- Add config path documentation for macOS and Linux.
- Add storage linking flow.
- Add upgrade notes.

Suggested owner:

- A packaging-focused agent once the CLI shape settles.

## Integration Rules

- Keep storage files and generated indexes out of source commits unless explicitly needed.
- Prefer additive files for parallel work.
- Avoid editing `cmd/matt/main.go` from multiple agents at once.
- Run `GOCACHE=/private/tmp/maat-go-cache GOPATH=/private/tmp/maat-go-path go test ./...`.
- Commit each coherent change using the format in `AGENTS.md`.
