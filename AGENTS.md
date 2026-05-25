# Agent Instructions

This repository is primarily for agents. Treat it as the source of truth for project state and history.

These conventions are adapted from the nearby Orion and Aether repositories so agents can move between Sunday Studio projects without switching habits.

## Repository Map

- `agents/`: agent identity notes, integration expectations, and future agent profile templates.
- `decisions/`: durable product and architecture decisions.
- `docs/`: architecture, workflows, storage model, CLI/TUI/UI design, and integration notes.
- `ledger/`: early append-only ledger templates and migration notes. The target architecture prefers per-event files to reduce merge conflicts.
- `projects/`: current project state and project templates.
- `reports/`: generated or agent-written summaries, handoffs, and status digests.
- `tags.md`: shared tag vocabulary.

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

Keep Maat accurate without requiring the human to manage it.

For any meaningful project-management action, update both the current state and transaction history.

The early repository uses `projects/<project-id>.md` and `ledger/YYYY-MM.md`. The target architecture moves toward conflict-resistant per-project directories and per-event files.

Then commit the change to Git.

## Before You Change Anything

1. Pull or inspect the latest repository state.
2. Read this file.
3. Check whether a project already exists in `projects/`.
4. If uncertain, prefer adding a small `waiting` note or handoff instead of overwriting status.

## Project Files

Legacy project files are plain Markdown using the template in `projects/_template.md`. Target architecture uses `projects/<project-key>/project.md` plus separate goal, ticket, and event files.

Legacy project fields:

- `ID`
- `Status`
- `Owner`
- `Updated`
- `Tags`

Use stable IDs:

- Projects: lowercase slugs, for example `orion`.
- Early docs may use readable IDs like `G-001` and `T-001`.
- Target storage should use collision-resistant IDs such as `G-20260525-190533-a7f3` and `T-20260525-190700-b91c`.
- Decisions use stable IDs such as `D-20260525-short-slug`.

## Event Rules

Events are append-only.

- Target architecture writes new event files under `projects/<project-key>/events/YYYY/MM/`.
- Legacy v0 docs may still use monthly files such as `ledger/2026-05.md`.
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

The preferred integration is:

1. Clone this repo.
2. Perform the project update.
3. Create the matching event.
4. Commit as the agent identity.
5. Push to the shared remote.

For systems that cannot push directly, write a complete handoff file in `reports/` and include the exact object and event changes another agent should apply.

## What Not To Do

- Do not use hidden databases as the source of truth.
- Do not store primary state outside Markdown.
- Do not delete history.
- Do not silently change the meaning of a status.
- Do not mark work done unless the completion evidence is clear.
- Do not create broad refactors while making architecture or documentation updates.
