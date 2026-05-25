# Maat

Maat is a Git-native project memory for agent-run work.

It is intentionally plain Markdown. Agents update project files, append transactional events to the ledger, and commit every meaningful change to Git. Humans can read the repo directly; agent systems can integrate by cloning it, following `AGENTS.md`, and writing normal commits.

## What This Is For

Use Maat when you have multiple projects moving through multiple agents and you need one central, durable place to answer:

- What projects exist?
- What is each project trying to achieve?
- What goals and tasks are active, blocked, or done?
- Who changed what, when, and why?
- What happened across all projects in chronological order?

## Core Idea

Maat separates current state from history.

- `projects/` contains the current readable state of each project.
- `ledger/` contains the append-only transactional history.
- `agents/` contains agent identity notes and integration expectations.
- `reports/` contains periodic summaries produced by agents.
- `decisions/` contains durable decisions that should outlive task chatter.

Every agent action should update current state and append a ledger event in the same commit.

## Repository Layout

```text
.
├── AGENTS.md
├── README.md
├── agents/
├── decisions/
├── docs/
├── ledger/
├── projects/
└── reports/
```

## Minimum Agent Workflow

1. Pull the latest Git state.
2. Read `AGENTS.md`.
3. Find or create the relevant project in `projects/`.
4. Make the smallest useful state update.
5. Append an event to the monthly ledger in `ledger/`.
6. Commit with a clear message.
7. Push or otherwise sync the commit back to the central remote.

## Status Vocabulary

Use these statuses consistently:

- `proposed`: captured but not started.
- `active`: work is in progress.
- `waiting`: blocked by a person, system, dependency, or decision.
- `paused`: intentionally not moving now.
- `done`: finished for the stated scope.
- `archived`: no longer active, retained for history.

## Event Vocabulary

Use these event names in the ledger:

- `project.created`
- `project.updated`
- `goal.added`
- `goal.updated`
- `goal.completed`
- `task.added`
- `task.updated`
- `task.completed`
- `blocker.added`
- `blocker.cleared`
- `decision.recorded`
- `report.created`
- `handoff.created`

## First Projects To Register

Nearby workspaces suggest these projects may be good initial entries:

- `aether`: human-facing personal/productivity app with docs around journal, tasks, goals, settings, sync, and updater.
- `orion`: self-hosted monitoring app with Agent, Core, Console, incidents, monitors, and deployment docs.
- `neptune`: photo management system with local indexing, public API, and blog frontend.

Agents should create these project files only when they have enough current context to avoid inventing stale state.
