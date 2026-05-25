# Maat

Maat is a Git-backed project memory for agent-run work.

It gives agents a shared place to create goals, create and claim tickets, record progress, complete work with evidence, and sync status through Git. Humans can query the same state from the terminal or browse it in a Bubble Tea TUI.

## How It Works

Maat keeps durable state in a normal Git repository full of Markdown files. That repo is the source of truth.

The `matt` CLI builds local indexes from that Markdown for faster search and dashboard views:

- Markdown in Git is canonical.
- `.maat/index.json` and `.maat/index.sqlite` are rebuildable.
- Agents should write through `matt` instead of hand-editing files.
- Events are stored as small object files to reduce merge conflicts.

## Install Or Build

From a checkout:

```sh
scripts/install.sh
```

Build directly:

```sh
go build -o matt ./cmd/matt
```

Run from source:

```sh
go run ./cmd/matt status --storage .
```

Link your storage repo once:

```sh
matt init /absolute/path/to/maat-state
```

Or pass it explicitly:

```sh
matt status --storage /absolute/path/to/maat-state
```

## Core Workflows

Set up and inspect:

```sh
matt init /absolute/path/to/maat-state
matt index rebuild
matt validate
matt status
matt projects
matt search "blocked deploy"
```

Link a source repo:

```sh
cd /absolute/path/to/source-repo
matt project link
matt project show <project-key>
```

Create work:

```sh
matt goal create <project-key> "Ship first deploy"
matt ticket create <project-key> "Verify installer"
matt ticket create <project-key> "Fix deploy docs" --goal <goal-id>
```

Work a ticket:

```sh
matt ticket claim <ticket-id> --agent codex --ttl 2h
matt ticket comment <ticket-id> "Found the failing path."
matt ticket complete <ticket-id> --evidence "go test ./... passed"
```

Sync changes:

```sh
matt sync --message "status(orion): complete installer ticket"
matt sync --push
matt sync --status --json
```

Migrate legacy flat project files into the object layout:

```sh
matt migrate plan --json
matt migrate apply --dest /tmp/maat-migrated
```

## Agent Workflow

Agents should follow this loop:

1. Run `matt sync` or otherwise pull the Maat storage repo.
2. Inspect state with `matt status`, `matt project show`, or `matt search`.
3. Create or claim a ticket before doing material work.
4. Record meaningful progress with `matt ticket comment`.
5. Complete tickets only with evidence.
6. Run `matt validate` and `matt sync`.

Install the project instruction snippet into another repo with:

```sh
matt agent instructions --output AGENTS.md
```

## TUI

Launch the terminal dashboard:

```sh
matt tui
```

The TUI currently shows projects, status totals, project detail, and ticket lists. Search and timeline views are planned next.

## Useful Docs

- [Architecture](docs/architecture.md)
- [Storage Model](docs/storage-model.md)
- [CLI, TUI, And UI](docs/cli-tui-ui.md)
- [Agent Protocol](docs/agent-protocol.md)
- [Install](docs/install.md)
- [Development](docs/development.md)
- [Migration Dogfood](docs/migration-dogfood.md)
