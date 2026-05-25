# Workflows

## Start A New Project

1. Create `projects/<project-id>.md` from `projects/_template.md`.
2. Fill in the project metadata table.
3. Add the first goal if known.
4. Append a `project.created` ledger event.
5. Commit the project file and ledger file together.

## Add A Goal

1. Open the project file.
2. Add the next goal ID under `## Goals`.
3. Set the goal status to `proposed`, `active`, or `waiting`.
4. Append a `goal.added` ledger event.
5. Commit.

## Add A Task

1. Find the relevant goal.
2. Add the next task ID under that goal's `#### Tasks`.
3. Append a `task.added` ledger event.
4. Commit.

## Complete A Task

1. Change `- [ ] T-###` to `- [x] T-###`.
2. Update the goal or project status if the completion changes the current state.
3. Append a `task.completed` ledger event with evidence.
4. Commit.

## Record A Blocker

1. Add the blocker under `## Blockers`.
2. Set the project or goal status to `waiting` if the blocker stops progress.
3. Append a `blocker.added` ledger event.
4. Commit.

## Clear A Blocker

1. Mark the blocker cleared or move it to the relevant decision/context note.
2. Restore project or goal status if work can continue.
3. Append a `blocker.cleared` ledger event.
4. Commit.

## Record A Decision

1. Add the decision to the project `## Decisions` section if local to a project.
2. Create a file in `decisions/` if it affects the system or multiple projects.
3. Append a `decision.recorded` ledger event.
4. Commit.

## Create A Cross-Project Report

1. Read all relevant project files.
2. Write `reports/YYYY-MM-DD-<scope>.md`.
3. Append a `report.created` ledger event for each project that materially changed, or one `report.created` event with `Project` set to `maat` if it is only a summary.
4. Commit.
