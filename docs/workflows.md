# Workflows

## Concurrency Baseline

For all workflows, Markdown and Git are authoritative. SQLite is a local search cache.

Agents should prefer the `maat` CLI. The file paths below describe what the commands create in the storage repo.

Agents should:

1. Pull or sync before writing.
2. Use a `maat` write command for the action.
3. Validate the Markdown state.
4. Rebuild or refresh the local index when possible.
5. Commit and push according to policy.

If the index refresh fails after the Markdown write, keep the write and warn. The agent should not retry the same write just because SQLite was busy; it can run `maat index rebuild` later.

## Start A New Project

1. Run `maat project link` from the source repo or provide a path.
2. Maat creates or updates `projects/<project-key>/project.md`.
3. Maat records repository metadata under `projects/<project-key>/repos/` when available.
4. Validate, index, commit, and sync.

## Add A Goal

1. Run `maat goal create <project-key> "<goal title>"`.
2. Maat creates `projects/<project-key>/goals/<goal-id>.md`.
3. Maat creates a `goal.created` event file.
4. Validate, index, commit, and sync.

## Add A Ticket

1. Run `maat ticket create <project-key> "<ticket title>"`, with `--goal <goal-id>` if needed.
2. Maat creates `projects/<project-key>/tickets/<ticket-id>.md`.
3. Maat creates a `ticket.created` event file.
4. Validate, index, commit, and sync.

## Claim A Ticket

1. Run `maat ticket claim <ticket-id> --project <project-key> --agent <agent-id> --ttl <duration>`.
2. Maat creates a `ticket.claimed` event with an expiration time.
3. Do not block other agents after the claim expires.
4. Validate, index, commit, and sync.

## Comment On A Ticket

1. Run `maat ticket comment <ticket-id> "<comment>" --project <project-key>`.
2. Include useful progress, findings, or handoff context.
3. Maat creates a `ticket.commented` event.
4. Validate, index, commit, and sync.

## Complete A Ticket

1. Run `maat ticket complete <ticket-id> --project <project-key> --evidence "<evidence>"`.
2. Include concrete evidence.
3. Maat creates a `ticket.completed` event.
4. Validate, index, commit, and sync.

## Record A Blocker

Direct blocker commands are future work. Until then, record blockers as events when a workflow needs durable blocker history.

1. Create a `blocker.added` event for the affected object.
2. Include the reason and required unblock action.
3. Let current state compute `waiting` where appropriate.
4. Validate, index, commit, and sync.

## Clear A Blocker

Direct blocker commands are future work. Until then, clear blockers with correction or blocker-clear events when needed.

1. Create a `blocker.cleared` event.
2. Include evidence that work can continue.
3. Validate, index, commit, and sync.

## Record A Decision

1. Create a decision file in `decisions/` or inside the project.
2. Create a `decision.recorded` event.
3. Validate, index, commit, and sync.

## Create A Cross-Project Report

1. Read all relevant project files.
2. Write `reports/YYYY-MM-DD-<scope>.md`.
3. Create a `report.created` event when the report is part of durable history.
4. Validate, index, commit, and sync.
