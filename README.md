# Maat

Maat is a Git-native project memory for agent-run work.

It is intentionally plain Markdown at rest. Agents update project state through the `matt` CLI or future MCP tools, and every meaningful change is recorded in Git. Humans can read the repo directly, query it from the terminal, or open a local UI.

## What This Is For

Use Maat when you have multiple projects moving through multiple agents and you need one central, durable place to answer:

- What projects exist?
- What is each project trying to achieve?
- What goals and tickets are active, blocked, or done?
- Who changed what, when, and why?
- What happened across all projects in chronological order?

## Core Idea

Maat separates durable state from fast local views.

- Git plus Markdown is the source of truth.
- SQLite is a rebuildable index for status, search, timelines, and UI queries.
- The CLI is the primary write interface for agents.
- A Bubble Tea TUI and local web UI provide browsable views.
- Events are append-only and should be stored as small files to reduce merge conflicts.

The early repository has flat project files and a monthly ledger. The target architecture moves to per-project directories, per-object Markdown files, and generated ledgers.

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
├── reports/
└── scripts/
```

## Architecture Docs

- [Architecture](docs/architecture.md)
- [Storage Model](docs/storage-model.md)
- [Search And Indexing](docs/search-index.md)
- [CLI, TUI, And UI](docs/cli-tui-ui.md)
- [Agent Protocol](docs/agent-protocol.md)
- [Implementation Plan](docs/implementation-plan.md)
- [Work Plan](docs/work-plan.md)
- [Development](docs/development.md)
- [Install](docs/install.md)
- [Markdown Schema](docs/schema.md)
- [Workflows](docs/workflows.md)
- [Integrations](docs/integrations.md)

## Install

The install foundation is local-first and does not publish or fetch artifacts:

```sh
scripts/install.sh
```

It installs an existing `matt` binary from the checkout when present, or builds `./cmd/matt` locally with Go in offline mode. See [Install](docs/install.md) for macOS/Linux paths, storage setup, and run commands.

## Current CLI

The first implementation is a Go CLI named `matt`.

Run it locally:

```sh
go run ./cmd/matt status --storage .
go run ./cmd/matt projects --storage .
go run ./cmd/matt project show orion --storage .
go run ./cmd/matt search "agent health" --storage .
go run ./cmd/matt index rebuild --storage .
```

The current index command writes a rebuildable bootstrap index to `.maat/index.json`. The target architecture still calls for SQLite FTS and optional vector search.

## Minimum Agent Workflow

1. Sync Maat.
2. Inspect the relevant project, goals, and tickets.
3. Claim or create a ticket.
4. Record progress as comments or events.
5. Complete or update the ticket with evidence.
6. Sync the Maat storage repo.

## Status Vocabulary

Use these statuses consistently:

- `proposed`: captured but not started.
- `active`: work is in progress.
- `waiting`: blocked by a person, system, dependency, or decision.
- `paused`: intentionally not moving now.
- `done`: finished for the stated scope.
- `archived`: no longer active, retained for history.

## Event Vocabulary

Use these event names for append-only events:

- `project.created`
- `project.updated`
- `project.linked`
- `goal.created`
- `goal.updated`
- `goal.completed`
- `ticket.created`
- `ticket.claimed`
- `ticket.commented`
- `ticket.updated`
- `ticket.completed`
- `blocker.added`
- `blocker.cleared`
- `decision.recorded`
- `report.created`
- `handoff.created`

## Existing Project Records

Initial project records exist for:

- `aether`: human-facing personal/productivity app with docs around journal, tasks, goals, settings, sync, and updater.
- `orion`: self-hosted monitoring app with Agent, Core, Console, incidents, monitors, and deployment docs.
- `neptune`: photo management system with local indexing, public API, and blog frontend.
- `maat`: this project.
