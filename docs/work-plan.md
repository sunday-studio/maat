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

Status:

- Done in `51f57df feat(index): add sqlite search index`.

## Group 2: Target Storage Parser

Goal: support the conflict-resistant object layout from `docs/storage-model.md`.

Tasks:

- Parse `projects/<project-key>/project.md`.
- Parse `goals/*.md`.
- Parse `tickets/*.md`.
- Parse `events/YYYY/MM/*.md`.
- Validate required fields and object links.

Initial owner:

- Worker B: target object parser under `internal/maat`.

Status:

- Done in `6011fb7 feat(parser): load object storage layout`.

## Group 3: Validation And Store Health

Goal: let agents and humans trust the store before writing or committing.

Tasks:

- Detect duplicate project IDs.
- Detect duplicate goal IDs within a project.
- Detect duplicate ticket IDs within a goal or project.
- Detect invalid status values.
- Detect missing required fields.
- Detect malformed object files.
- Expose validation through `maat validate`.

Initial owner:

- Worker C: validation API and tests under `internal/maat`.

Status:

- Validation checks the object layout under `projects/<project-key>/`.

## Group 4: Write Path Foundations

Goal: prepare safe agent writes without rushing CLI commands.

Tasks:

- Generate collision-resistant IDs for goals, tickets, events, decisions, and repos.
- Generate event file paths.
- Render event Markdown from structured input.
- Add tests for ID shape, event paths, and Markdown output.
- Wire CLI write commands on top of these helpers.

Initial owner:

- Worker D: ID and event helpers under `internal/maat`.

Status:

- Done in `ef82244 feat(core): add event write helpers`.
- CLI write commands are now available.

## Group 5: CLI Commands

Goal: make the binary useful for agents and humans.

Tasks:

- Add `maat validate`.
- Wire SQLite-backed `maat search`.
- Add `maat ticket create`.
- Add `maat ticket comment`.
- Add `maat ticket complete`.
- Add `maat sync` with safe Git flow.
- Add JSON output for query commands.

Initial owner:

- Worker 1: read-path CLI wiring for validation, SQLite-backed search, index rebuild, and JSON output.
- Write-command integration follows the write-path core.

Status:

- Core read and write commands are available in the first release.

- Done in `84b1db1 feat(cli): wire read path commands`.
- Next write-command wiring assigned to Worker A for goal and ticket create/claim/comment/complete.
- Write-command wiring done in `43e7e47 feat(cli): add agent write commands`.
- Next sync command wiring assigned to Worker 1.
- Next write-command UX and JSON output assigned to Worker 3.
- Sync command wiring done in `1787757 feat(cli): wire sync command`.
- Write-command ergonomics coverage done in `fdd4341 test(cli): cover write command ergonomics`.
- Next project inference, JSON show, and ticket list/show work assigned to Worker 1.

## Group 6: TUI

Goal: give the human a polished terminal dashboard.

Tasks:

- Add Bubble Tea dependencies.
- Add `maat tui`.
- Show projects, active tickets, blocked tickets, timeline, and search.
- Keep mutations routed through the same core operations as the CLI.

Initial owner:

- Worker 5: Bubble Tea skeleton and callable TUI entrypoint.

Status:

- Done in `9965baf feat(tui): add bubble tea dashboard`.
- Next detail-pane and selection improvements assigned to Worker D.
- Detail-pane and selection improvements done in `b2c593d feat(tui): refine dashboard navigation`.
- Next ticket/search view improvements assigned to Worker 4.
- Ticket detail mode done in `bb890b2 feat(tui): add ticket detail mode`.
- Next TUI search/timeline/detail improvements assigned to Worker 3.

## Group 7: Local Web UI

Goal: provide a browser dashboard for browsing all project state.

Tasks:

- Add a local web UI command.
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
- Add setup flow for linking storage.
- Add upgrade notes.

Initial owner:

- Worker 6: local install script and install documentation.

Status:

- Done in `ea4ea32 feat(sync): add git primitives` and `da8372b docs(install): clarify offline installer`.
- Build and release setup assigned to Worker 4.

## Group 9: Git Sync Primitives

Goal: prepare safe sync flows without rushing user-facing commands.

Tasks:

- Detect whether the storage path is a Git repository.
- Read branch and remote metadata.
- Parse dirty status.
- Provide pull, commit, and push primitives.
- Use a fake command runner in tests to avoid network access.

Initial owner:

- Worker 3: Git sync core under `internal/maat`.

Status:

- Done in `ea4ea32 feat(sync): add git primitives`.
- Next orchestration API assigned to Worker B.
- Sync orchestration API done in `024f39b feat(sync): orchestrate store sync`.

## Group 10: Agent Protocol Packaging

Goal: make it easy to teach other repos and agents how to use Maat.

Tasks:

- Document current and next agent-facing commands.
- Define the copy/paste snippet for external project `AGENTS.md` files.
- Generate the full setup contract with `maat initialize`.
- Keep the snippet short enough that agents actually follow it.
- Add tests that pin the command names and evidence rule.

Initial owner:

- Worker E: agent protocol docs and optional snippet helper.

Status:

- Done in `7654ed2 docs(agent): define command protocol`.
- Superseded the old snippet helper with `maat initialize`; no separate `maat agent` namespace is kept.

## Group 12: README And Product Packaging

Goal: make Maat understandable and installable from a clean checkout.

Tasks:

- Rewrite README around current commands and workflows.
- Keep advanced architecture in docs.
- Add build and release commands.
- Add GitHub Actions release/check workflow.
- Keep install documentation aligned with scripts.

Initial owners:

- Worker 4: build and release setup.
- Worker 5: simple current README.

## Group 13: Search And State Hardening

Goal: improve query quality for agents and humans.

Tasks:

- Improve target-layout object indexing.
- Add stale/active/blocked internal query APIs.
- Add claim-expiration awareness.
- Keep search and state behavior covered with tests.

Initial owner:

- Worker 2: storage/search hardening.

Status:

- Object indexing is improved enough for the first release. Stale, active, and blocked specialty views remain future work.

## Integration Rules

- Keep storage files and generated indexes out of source commits unless explicitly needed.
- Prefer additive files for parallel work.
- Avoid editing `cmd/maat/main.go` from multiple agents at once.
- Run `GOCACHE=/private/tmp/maat-go-cache GOPATH=/private/tmp/maat-go-path go test ./...`.
- Commit each coherent change using the format in `AGENTS.md`.
