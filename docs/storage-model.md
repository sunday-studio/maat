# Storage Model

Maat stores canonical state as Markdown files in a Git repository.

The current storage model is object-oriented and conflict-resistant. Most agent actions create new files instead of editing shared files.

## Principles

- Git plus Markdown is authoritative.
- SQLite is a rebuildable cache.
- High-frequency writes create new files.
- Current state is computed from object files and event files.
- Generated summaries are views, not primary state.
- File names use collision-resistant IDs, not shared counters.

## Current Layout

```text
maat-state/
├── setup.md
├── projects/
│   └── maat/
│       ├── project.md
│       ├── goals/
│       │   └── G-20260525-190533-a7f3.md
│       ├── tickets/
│       │   └── T-20260525-190700-b91c.md
│       └── events/
│           └── 2026/
│               └── 05/
│                   └── E-20260525-190812-codex-4c9a.md
└── .maat/
    ├── index.json
    └── index.sqlite
```

The product repository ignores `state/` so local smoke data and nested storage experiments do not clutter the source tree. Storage repos use the root layout above.

## Storage Setup File

Path:

```text
setup.md
```

The storage root may include `setup.md` with default agent rules for that storage repo. `maat setup --storage <path>` creates and commits it when missing or blank, and `maat setup rules --storage <path>` can backfill an existing storage repo. The CLI tells the user they can edit the file to change the default rules agents should follow in that storage repo.

Use it for durable coordination rules, including the expectation that agents store useful plans as ticket comments when those plans matter for handoff or future work. Do not use it for private scratch reasoning or generated summaries that can be rebuilt.

## Local Cache Layout

Maat may create local cache data under `.maat/`, including `index.json` and `index.sqlite`.

That cache is rebuildable from Markdown and should normally stay ignored:

- product repos should ignore `.maat/`
- local storage checkouts should ignore `.maat/` unless the storage repo deliberately chooses otherwise
- agents should not commit `.maat/` as primary state
- cache deletion should only require `maat index rebuild`

For concurrency, each agent, process, or machine can have its own cache. Shared state is the Markdown files plus Git history, not a shared SQLite database.

## Project Identity

A project needs a stable identity even if the source repo moves.

Use:

- `project_key`: stable directory-safe key inside Maat.
- `display_name`: human-readable name.
- `Primary Repo`: local source repo path when linked.
- `Remote`: Git remote URL when available.

If a project has a Git remote, the remote is the strongest identity signal. If it has no remote, Maat uses the requested or inferred project key.

## Project File

Path:

```text
projects/<project-key>/project.md
```

Shape:

```markdown
# Project: Maat

| Field | Value |
|---|---|
| Project Key | maat |
| Display Name | Maat |
| Status | active |
| Created | 2026-05-25T19:05:00+02:00 |
| Updated | 2026-05-25T19:05:00+02:00 |
| Tags | #product #agent-run |

## Summary

Git-native project memory for agent-managed work.

## Identity

| Field | Value |
|---|---|
| Primary Repo | /Users/casprine/Desktop/vendor/sunday-studio/maat |
| Remote | git@github.com:sunday-studio/maat.git |
```

Project files should change rarely. Frequent updates belong in event files.

## Goal File

Path:

```text
projects/<project-key>/goals/<goal-id>.md
```

Shape:

```markdown
# Goal: Improve Agent Health Clarity

| Field | Value |
|---|---|
| Goal ID | G-20260525-190533-a7f3 |
| Project | maat |
| Status | active |
| Created | 2026-05-25T19:05:33+02:00 |
| Tags | #backend #frontend |

## Outcome

Agent workflow docs should make the state model and handoff expectations clear.
```

## Ticket File

Path:

```text
projects/<project-key>/tickets/<ticket-id>.md
```

Shape:

```markdown
# Ticket: Separate Project State From Product Examples

| Field | Value |
|---|---|
| Ticket ID | T-20260525-190700-b91c |
| Project | maat |
| Goal | G-20260525-190533-a7f3 |
| Status | active |
| Created | 2026-05-25T19:07:00+02:00 |
| Tags | #backend |

## Description

Keep Maat's own repository focused on Maat state while product examples remain generic.

## Acceptance

- The repository only contains Maat project state.
- Public docs use Maat or generic examples.
- Cross-project state lives in the user's chosen Maat storage repo, not the product repo.
```

Tickets may be goal-linked or standalone.

## Event File

Path:

```text
projects/<project-key>/events/YYYY/MM/<event-id>.md
```

Shape:

```markdown
# Event: ticket.completed

| Field | Value |
|---|---|
| Event ID | E-20260525-191100-codex-4c9a |
| Time | 2026-05-25T19:11:00+02:00 |
| Actor | codex |
| Project | maat |
| Type | ticket.completed |
| Object | T-20260525-190700-b91c |
| Commit | abc1234 |

## Summary

Completed the ticket after backend tests passed.

## Evidence

- `go test ./...` passed in `apps/core`.
```

Events are append-only. If an event is wrong, write a correction event.

## Comments

Comments should be events:

```text
Type: ticket.commented
Object: T-20260525-190700-b91c
```

This prevents many agents from editing one ticket file just to add notes.

## Claims

Claims should be lease events:

```text
Type: ticket.claimed
Object: T-20260525-190700-b91c
Expires: 2026-05-25T21:07:00+02:00
```

Claims expire automatically. An expired claim should not block another agent.

## Merge Conflict Reduction

This storage model reduces conflicts because most operations add files:

- two agents creating tickets create two different files
- two agents commenting create two different event files
- two agents recording progress create two different event files
- status history is event-based instead of one shared append target

The remaining conflict risk is low-frequency metadata editing, such as project summaries or ticket descriptions. Those edits should be small and validated.

Index rebuild conflicts are not state conflicts. If SQLite is locked, stale, or missing, agents should keep the Markdown changes and rebuild the cache later.
