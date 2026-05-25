# Agent Operating Instructions

This repository is primarily for agents. Treat it as the source of truth for project state and history.

## Prime Directive

Keep Maat accurate without requiring the human to manage it.

For any meaningful project-management action, update both:

1. The current project state in `projects/<project-id>.md`.
2. The chronological ledger in `ledger/YYYY-MM.md`.

Then commit the change to Git.

## Before You Change Anything

1. Pull or inspect the latest repository state.
2. Read this file.
3. Check whether a project already exists in `projects/`.
4. If uncertain, prefer adding a small `waiting` note or handoff instead of overwriting status.

## Project Files

Each project file is plain Markdown using the template in `projects/_template.md`.

Required project fields:

- `ID`
- `Status`
- `Owner`
- `Updated`
- `Tags`

Use stable IDs:

- Projects: lowercase slugs, for example `orion`.
- Goals: `G-001`, `G-002`, increasing inside a project.
- Tasks: `T-001`, `T-002`, increasing inside a project.
- Decisions: `D-001`, `D-002`, increasing inside a project or decision file.

## Ledger Rules

The ledger is append-only.

- Append new events to the current month file, for example `ledger/2026-05.md`.
- Do not rewrite old ledger events unless you are correcting your own uncommitted mistake.
- If a previous event was wrong, append a new correction event.
- Every ledger event must reference the project and the changed file.

Use the event template in `ledger/_template.md`.

## Commit Rules

Commit each coherent update.

Good commit messages:

- `feat(orion): add first deployment goal`
- `status(aether): mark sync checklist waiting`
- `ledger(neptune): record API handoff`

Avoid vague messages like `update`, `changes`, or `notes`.

## Conflict Handling

If two agents edit the same project:

1. Preserve both agents' factual updates.
2. Resolve checkboxes and statuses conservatively.
3. Add a ledger event explaining the merge if the resulting state changed.

## Integration Pattern

Any agent system can participate if it can:

- Read and write files.
- Run Git commands.
- Follow Markdown templates.

The preferred integration is:

1. Clone this repo.
2. Perform the project update.
3. Append the ledger event.
4. Commit as the agent identity.
5. Push to the shared remote.

For systems that cannot push directly, write a complete handoff file in `reports/` and include the exact project and ledger changes another agent should apply.

## What Not To Do

- Do not use hidden databases as the source of truth.
- Do not store primary state outside Markdown.
- Do not delete history.
- Do not silently change the meaning of a status.
- Do not mark work done unless the completion evidence is clear.
