# Storage Model

Maat stores canonical state as Markdown files in a Git repository.

The target storage model is object-oriented and conflict-resistant. Most agent actions create new files instead of editing shared files.

## Principles

- Git plus Markdown is authoritative.
- SQLite is a rebuildable cache.
- High-frequency writes create new files.
- Current state is computed from object files and event files.
- Generated summaries are views, not primary state.
- File names use collision-resistant IDs, not shared counters.

## Target Layout

```text
maat-state/
├── maat.toml
├── state/
│   ├── agents/
│   ├── decisions/
│   │   └── D-20260525-architecture-direction.md
│   ├── projects/
│   │   └── maat/
│   │       ├── project.md
│   │       ├── repos/
│   │       │   └── R-20260525-190100-a31f.md
│   │       ├── goals/
│   │       │   └── G-20260525-190533-a7f3.md
│   │       ├── tickets/
│   │       │   └── T-20260525-190700-b91c.md
│   │       ├── reports/
│   │       └── events/
│   │           └── 2026/
│   │               └── 05/
│   │                   └── E-20260525-190812-codex-4c9a.md
│   ├── reports/
│   ├── templates/
│   └── tags.md
├── docs/
└── README.md
```

The current repository still has early flat files such as `state/projects/maat.md` and `state/ledger/2026-05.md`. Those are useful v0 documents. The architecture target is the directory-per-project layout above.

## Project Identity

A project needs a stable identity even if the source repo moves.

Use:

- `project_key`: stable directory-safe key inside Maat.
- `display_name`: human-readable name.
- `repo_fingerprint`: hash of the normalized remote URL when available.
- `created_at`: timestamp used when no remote exists.
- `path_aliases`: known local paths where the project has lived.

If a project has a Git remote, the remote should be the strongest identity signal. If it has no remote, Maat should generate a stable project key and later attach the remote when one appears.

## Project File

Path:

```text
state/projects/<project-key>/project.md
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
| Primary Repo | R-20260525-190100-a31f |
| Remote | git@github.com:sunday-studio/maat.git |
```

Project files should change rarely. Frequent updates belong in event files.

## Goal File

Path:

```text
state/projects/<project-key>/goals/<goal-id>.md
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

Goal status may be computed from events once event processing exists. Until then the file can carry a status field.

## Ticket File

Path:

```text
state/projects/<project-key>/tickets/<ticket-id>.md
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
state/projects/<project-key>/events/YYYY/MM/<event-id>.md
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

## Generated Views

These should be generated from object and event files:

- monthly ledger
- project summary
- open ticket list
- blocked ticket list
- agent activity feed
- daily report
- stale project report

Generated views may be committed only when there is a specific reason to preserve the rendered snapshot.

## Merge Conflict Reduction

This storage model reduces conflicts because most operations add files:

- two agents creating tickets create two different files
- two agents commenting create two different event files
- two agents recording progress create two different event files
- status history is event-based instead of one shared ledger append

The remaining conflict risk is low-frequency metadata editing, such as project summaries or ticket descriptions. Those edits should be small and validated.
