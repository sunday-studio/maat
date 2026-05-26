# Workflows

For all workflows, Markdown and Git are authoritative. SQLite is a local search cache.

Agents should use the `maat` CLI.

## Start A Project

From the source repo:

```sh
maat project link
```

Or link a specific repo:

```sh
maat project link /absolute/path/to/source-repo --key <project-key> --name "<display-name>"
```

Maat creates or updates `projects/<project-key>/project.md`, validates the store, refreshes indexes, and syncs when configured.

## Add A Goal

```sh
maat goal create <project-key> "<goal title>" --outcome "the concrete outcome this goal should achieve"
```

Maat creates:

- `projects/<project-key>/goals/<goal-id>.md`
- a `goal.created` event

## Add A Ticket

Standalone ticket:

```sh
maat ticket create <project-key> "<ticket title>" --description "the concrete work another agent should do" --acceptance "clear completion condition"
```

Ticket under a goal:

```sh
maat ticket create <project-key> "<ticket title>" --goal <goal-id> --description "the concrete work another agent should do" --acceptance "clear completion condition"
```

Goals require `--outcome`. Tickets require `--description` and at least one `--acceptance` value.

Maat creates:

- `projects/<project-key>/tickets/<ticket-id>.md`
- a `ticket.created` event

## Claim A Ticket

```sh
maat ticket claim <ticket-id> --project <project-key> --agent <agent-id> --ttl 2h
```

Claims are soft leases recorded as events. Expired claims should not block other agents.

## Comment On A Ticket

```sh
maat ticket comment <ticket-id> "short factual progress note" --project <project-key>
```

Use comments for meaningful progress, blockers, findings, and handoffs.

## Complete A Ticket

```sh
maat ticket complete <ticket-id> --project <project-key> --evidence "go test ./... passed"
```

Completion always requires evidence.

## Sync

```sh
maat sync --status
maat validate
maat index rebuild
maat sync --message "status(<project-key>): update maat" --push
```

If index refresh fails after a Markdown write, keep the write and rebuild the index later. Do not repeat the same write just because SQLite was busy.
