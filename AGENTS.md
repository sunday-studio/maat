# Agent Instructions

This repository contains the Maat product source. Treat code, product docs, and tests here as canonical.

These conventions define how agents should work inside Maat.

## Repository Map

- `cmd/`: the `maat` CLI entrypoint.
- `docs/`: architecture, workflows, storage model, CLI/TUI/UI design, and integration notes.
- `internal/`: Go packages for storage, indexing, sync, validation, and TUI logic.
- `scripts/`: local install and release helper scripts.

`state/` is ignored in this repository. Use it only for local smoke data, or keep real Maat state in a separate Git-controlled storage repo.

## Naming Rules

- New files and folders must use lowercase kebab-case.
- Keep names descriptive and short enough to scan.
- Use stable object IDs for project state, but do not rely on sequential IDs when multiple agents may create objects concurrently.
- Do not rename existing files or folders just for style unless the task asks for it.

## Commit Message Format

Use this format for small commits:

```txt
conventional-commit-type(scope): one liner
```

Use this format only when the one-liner does not cover the change:

```txt
conventional-commit-type(scope): one liner

- key point if any exists;
- another key point if any exists;
```

- The first line must use a conventional commit type and a meaningful scope.
- Use scopes such as `docs`, `repo`, `agent`, `architecture`, `storage`, `cli`, `tui`, `ui`, `search`, or a project ID.
- Prefer a one-line commit message when the subject fully explains a small change.
- Add bullet points only when they add meaningful context beyond the one-liner.
- When bullets are needed, add one blank line after the subject.
- Bullet points must start with `- ` followed by text, with one space after the dash.
- End each bullet with `;`.
- Do not put blank lines between bullet points.

## Prime Directive

Keep the product repo clean. Do not commit local Maat state to this repository.

For project-management actions, update the configured external Maat storage repo. For product changes here, update code or docs, run relevant checks, commit, and push.

When you finish a change, always commit it before handing the work back unless the user explicitly asks you not to commit.

## Before You Change Anything

1. Pull or inspect the latest repository state.
2. Read this file.
3. Inspect the relevant code and docs before editing.
4. If project state is needed, use the configured external storage repo instead of adding tracked Markdown state here.

## Project Files

Legacy project files are plain Markdown in a Maat storage repo. Current object-layout storage uses `projects/<project-key>/project.md` plus separate goal, ticket, and event files inside that storage repo.

Legacy project fields:

- `ID`
- `Status`
- `Owner`
- `Updated`
- `Tags`

Use stable IDs:

- Projects: lowercase slugs, for example `maat`.
- Early docs may use readable IDs like `G-001` and `T-001`.
- Target storage should use collision-resistant IDs such as `G-20260525-190533-a7f3` and `T-20260525-190700-b91c`.
- Decisions use stable IDs such as `D-20260525-short-slug`.

## Event Rules

Events are append-only.

- Current object-layout storage writes new event files under `projects/<project-key>/events/YYYY/MM/` in the storage repo.
- Legacy v0 storage may still use monthly files such as `ledger/2026-05.md`.
- Do not rewrite old events unless you are correcting your own uncommitted mistake.
- If a previous event was wrong, append a new correction event.
- Every event must reference the project and changed object.

Use the event shape in `docs/schema.md` for new architecture work.

## Conflict Handling

If two agents edit the same project:

1. Preserve both agents' factual updates.
2. Resolve checkboxes and statuses conservatively.
3. Add an event explaining the merge if the resulting state changed.

Prefer storage patterns that avoid conflicts in the first place:

- Add new files for comments, events, reports, and tickets when possible.
- Avoid shared append targets for high-frequency agent writes.
- Treat generated summaries and rendered ledgers as rebuildable views unless explicitly committed.
- Pull before writing, validate before committing, and sync after committing.

## Integration Pattern

Any agent system can participate if it can:

- Read and write files.
- Run Git commands.
- Follow Markdown templates.

The preferred product-repo integration is:

1. Clone this repo.
2. Perform the code or documentation update.
3. Run the relevant checks.
4. Commit as the agent identity.
5. Push to the shared remote.

For systems that cannot push directly, write a complete handoff in the external Maat storage repo or in the conversation.

## What Not To Do

- Do not use hidden databases as the source of truth.
- Do not commit repo-local `state/` content.
- Do not delete history.
- Do not silently change the meaning of a status.
- Do not mark work done unless the completion evidence is clear.
- Do not create broad refactors while making architecture or documentation updates.
