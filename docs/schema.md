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

## Terminal App Catalog Files

Terminal app catalog objects live inside a project so observations can point back to goals and tickets without leaving Markdown:

```text
projects/<project-key>/catalog/apps/<slug>.md
projects/<project-key>/catalog/patterns/<slug>.md
projects/<project-key>/catalog/decisions/<decision-id>.md
projects/<project-key>/catalog/opportunities/<opportunity-id>.md
projects/<project-key>/catalog/events/YYYY/MM/<event-id>.md
```

Catalog IDs should be stable and collision-resistant. Slugs must match their file names without `.md`.

### Catalog App File

Required shape:

```markdown
# Catalog App: lazygit

| Field | Value |
|---|---|
| App ID | CA-20260527-lazygit |
| Project | maat |
| Slug | lazygit |
| Name | lazygit |
| Summary | Terminal UI for Git workflows. |
| Source URL | https://github.com/jesseduffield/lazygit |
| Website URL | unknown |
| Stars | unknown |
| Language | Go |
| License | unknown |
| Category | git |
| Last Reviewed | 2026-05-27 |
| Tags | #terminal-app #git #keyboard-first |

## Screens

- unknown

## Notes

Use `unknown` for metadata that has not been verified.
```

### Catalog Pattern File

Required shape:

```markdown
# Catalog Pattern: Focused Detail Pane

| Field | Value |
|---|---|
| Pattern ID | CP-20260527-focused-detail-pane |
| Project | maat |
| Slug | focused-detail-pane |
| Title | Focused detail pane |
| Category | inspection/detail panes |
| Tags | #tui #detail |

## Problem

List views hide important object context.

## Observed In

- lazygit
- gh-dash

## Maat Relevance

Maat should let users inspect the selected project or ticket without leaving the terminal flow.

## Implementation Notes

Keep the detail pane structured so agents can extract metadata and next actions.

## Related Goals

- G-20260527-104618-2535

## Related Tickets

- T-20260527-104802-f29d
```

Observed apps must reference app slugs in the same catalog. Related goals and tickets are optional, but when present they must point at existing project objects.

### Catalog Decision File

Required shape:

```markdown
# Catalog Decision: Adopt Focused Detail Pane

| Field | Value |
|---|---|
| Decision ID | CD-20260527-focused-detail-pane |
| Project | maat |
| State | adopt |
| Pattern | focused-detail-pane |
| Date | 2026-05-27 |
| Related Goal | G-20260527-104618-2535 |
| Related Ticket | T-20260527-104802-f29d |

## Rationale

Focused detail keeps reading close to navigation.

## Evidence

- Pattern appears in catalog apps and maps to the current TUI board/detail flow.
```

Decision states are:

- `adopt`
- `adopt later`
- `reject`
- `needs research`

### Catalog Opportunity File

Required shape:

```markdown
# Catalog Opportunity: Project Board Detail Flow

| Field | Value |
|---|---|
| Opportunity ID | CO-20260527-project-board-detail-flow |
| Project | maat |
| Status | ticketed |
| Source Pattern | focused-detail-pane |
| Area | tui |
| Effort | medium |
| Risk | low |
| Suggested Goal | G-20260527-104618-2535 |
| Suggested Ticket | T-20260527-104802-f29d |

## Description

Make project list, board navigation, and item detail feel like one terminal workflow.
```

Opportunity statuses are:

- `proposed`
- `ticketed`
- `in progress`
- `verified`
- `declined`

### Catalog Event File

Catalog review events are append-only and separate from project lifecycle events:

```markdown
# Catalog Event: catalog.app.reviewed

| Field | Value |
|---|---|
| Event ID | CE-20260527-105500-codex-a1b2 |
| Time | 2026-05-27T10:55:00+02:00 |
| Actor | codex |
| Project | maat |
| Type | catalog.app.reviewed |
| Object | lazygit |

## Summary

Reviewed lazygit as a terminal app catalog seed.

## Evidence

- Seed object validates from Markdown.
```
