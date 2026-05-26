# Markdown Schema

Maat uses plain Markdown conventions as the durable store.

The schema is designed to be readable by humans, reliable enough for agents, and easy to index into SQLite.

The current schema is directory-per-project and object-per-file. See [Storage Model](./storage-model.md) for the full rationale. Examples below use paths relative to the configured storage repo root.

## Project File

Path:

```text
projects/<project-key>/project.md
```

Required shape:

```markdown
# Project: <Name>

| Field | Value |
|---|---|
| Project Key | <project-key> |
| Display Name | <name> |
| Status | active |
| Created | 2026-05-25T19:05:00+02:00 |
| Updated | 2026-05-25T19:05:00+02:00 |
| Tags | #example #tag |

## Summary

Short current-state summary.
```

## Goal File

Path:

```text
projects/<project-key>/goals/<goal-id>.md
```

Required shape:

```markdown
# Goal: Goal name

| Field | Value |
|---|---|
| Goal ID | G-20260525-190533-a7f3 |
| Project | <project-key> |
| Status | active |
| Created | 2026-05-25T19:05:33+02:00 |
| Tags | #example |

## Outcome

The outcome this goal is trying to achieve.
```

## Ticket File

Path:

```text
projects/<project-key>/tickets/<ticket-id>.md
```

Required shape:

```markdown
# Ticket: Ticket name

| Field | Value |
|---|---|
| Ticket ID | T-20260525-190700-b91c |
| Project | <project-key> |
| Goal | G-20260525-190533-a7f3 |
| Status | active |
| Created | 2026-05-25T19:07:00+02:00 |
| Tags | #example |

## Description

The concrete work to do.

## Acceptance

- Clear completion condition.
```

Tickets may be standalone. If no goal is attached, use `Goal | none`.

## Status Values

Use only:

- `proposed`
- `active`
- `waiting`
- `paused`
- `done`
- `archived`

## Tags

Tags are lowercase Markdown hashtags.

Examples:

- `#agent-run`
- `#git-native`
- `#blocked`
- `#release`
- `#docs`
- `#frontend`
- `#backend`
- `#infra`

## Event File

Events live under the project they affect:

```text
projects/<project-key>/events/YYYY/MM/<event-id>.md
```

Required shape:

```markdown
# Event: ticket.completed

| Field | Value |
|---|---|
| Event ID | E-20260525-192000-codex-a1b2 |
| Time | 2026-05-25T19:20:00+02:00 |
| Actor | codex |
| Project | <project-key> |
| Type | ticket.completed |
| Object | T-20260525-190700-b91c |

## Summary

Marked T-20260525-190700-b91c complete after verification.

## Evidence

- Evidence: tests passed locally.
- Follow-up: none.
```

Events are append-only. Correction events should be used instead of rewriting committed history.
