# Workflows

## Start A New Project

1. Run `matt project link` from the source repo or provide a path.
2. Create `projects/<project-key>/project.md`.
3. Record repository metadata under `projects/<project-key>/repos/`.
4. Create a `project.created` event file.
5. Validate, index, commit, and sync.

## Add A Goal

1. Create `projects/<project-key>/goals/<goal-id>.md`.
2. Create a `goal.created` event file.
3. Validate, index, commit, and sync.

## Add A Ticket

1. Create `projects/<project-key>/tickets/<ticket-id>.md`.
2. Attach a goal ID if the ticket belongs to a goal.
3. Create a `ticket.created` event file.
4. Validate, index, commit, and sync.

## Claim A Ticket

1. Create a `ticket.claimed` event with an expiration time.
2. Do not block other agents after the claim expires.
3. Validate, index, commit, and sync.

## Comment On A Ticket

1. Create a `ticket.commented` event.
2. Include useful progress, findings, or handoff context.
3. Validate, index, commit, and sync.

## Complete A Ticket

1. Create a `ticket.completed` event.
2. Include evidence.
3. Update computed state through the index.
4. Validate, index, commit, and sync.

## Record A Blocker

1. Create a `blocker.added` event for the affected object.
2. Include the reason and required unblock action.
3. Let current state compute `waiting` where appropriate.
4. Validate, index, commit, and sync.

## Clear A Blocker

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
