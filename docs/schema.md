# Markdown Schema

Maat uses plain Markdown conventions instead of a database.

The schema is designed to be readable by humans and reliable enough for agents.

## Project File

Path:

```text
projects/<project-id>.md
```

Required shape:

```markdown
# Project: <Name>

| Field | Value |
|---|---|
| ID | <project-id> |
| Status | active |
| Owner | agents |
| Updated | 2026-05-25 |
| Tags | #example #tag |

## Current

Short current-state summary.

## Goals

### G-001: Goal name

| Field | Value |
|---|---|
| Status | active |
| Updated | 2026-05-25 |
| Tags | #example |

#### Tasks

- [ ] T-001: Task name
- [x] T-002: Completed task name

## Blockers

- None.

## Decisions

- None.

## Links

- None.
```

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

Add new tags to `tags.md` when they become reusable across projects.

## Ledger Event

Ledger events live in monthly files:

```text
ledger/YYYY-MM.md
```

Required shape:

```markdown
### EVT-YYYYMMDD-HHMMSS-<short-project>-<short-action>

| Field | Value |
|---|---|
| Time | 2026-05-25T19:20:00+02:00 |
| Actor | codex |
| Project | example |
| Event | task.completed |
| Files | projects/example.md |
| Summary | Marked T-001 complete after verification. |

#### Details

- Evidence: tests passed locally.
- Follow-up: none.
```

## Report File

Reports are agent-written summaries. They do not replace ledger events.

Path:

```text
reports/YYYY-MM-DD-<scope>.md
```

Use reports for:

- Daily or weekly cross-project status.
- Handoffs between agents.
- Risk summaries.
- Human-readable digest views.

## Decision File

Use `decisions/` for durable choices that affect multiple projects or the Maat system itself.

Path:

```text
decisions/D-YYYYMMDD-<slug>.md
```
