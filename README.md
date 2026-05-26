# Maat

Maat is a Git-backed project memory for agent-run work.

It gives agents one shared place to create goals, create and claim tickets, record progress, complete work with evidence, and sync project status through Git. Humans can query the same state from the terminal or browse it in a Bubble Tea TUI.

## Quick Start

### 1. Install Maat

Run the installer:

```sh
curl -fsSL https://raw.githubusercontent.com/sunday-studio/maat/main/scripts/install.sh | sh
```

The installer detects macOS/Linux and arm64/amd64, downloads the right release, chooses a writable install directory, installs the binary as `maat`, and adds the install directory to your shell profile if needed.

Check the binary:

```sh
maat version
```

You do not need to clone this repository to use Maat. Clone it only if you want to contribute or build from source.

### 2. Prepare A Storage Repo

Maat state lives in a normal Git repository full of Markdown files. That storage repo is the source of truth and is separate from the Maat product repo.

Create one locally:

```sh
mkdir -p "$HOME/maat-state"
git init "$HOME/maat-state"
```

Or clone an existing shared storage repo:

```sh
git clone <your-maat-storage-remote> "$HOME/maat-state"
```

Then run:

```sh
maat setup
```

The setup prompt asks for:

- the absolute path to the storage Git repo
- the default actor name
- whether Maat should auto-pull before reads
- whether Maat should auto-commit after writes
- whether Maat should auto-push after commits

For agents and scripts, use the non-interactive form:

```sh
maat setup --storage "$HOME/maat-state"
```

You can also pass storage explicitly on any command:

```sh
maat status --storage "$HOME/maat-state"
```

### 3. Register A Project Repo

Run this from inside the source repo you want Maat to track:

```sh
cd /absolute/path/to/source-repo
maat initialize
```

`maat initialize` links the current repo as a Maat project and prints an agent setup document. Save that document, or its short snippet, into `AGENTS.md`, `CLAUDE.md`, Cursor rules, or whatever instruction surface your agent reads before work.

To link without printing the full agent instructions:

```sh
maat project link
maat project show <project-key>
```

### 4. Inspect Current State

```sh
maat status
maat projects
maat project show <project-key>
maat search "blocked deploy"
maat validate
```

Human output is colored when the terminal supports it. Set `MAAT_COLOR=always` or `MAAT_COLOR=never` to force color behavior.

Agents that need parseable progress should use `--agent-use` instead of scraping human output:

```sh
maat status --agent-use
```

`--agent-use` emits newline-delimited JSON updates and cannot be combined with `--json`.

### 5. Create Goals And Tickets

```sh
maat goal create <project-key> "Ship first deploy"
maat ticket create <project-key> "Verify installer"
maat ticket create <project-key> "Fix deploy docs" --goal <goal-id>
```

Tickets can belong to a goal or stand on their own.

### 6. Work A Ticket

```sh
maat ticket claim <ticket-id> --project <project-key> --agent codex --ttl 2h
maat ticket comment <ticket-id> "Found the failing path." --project <project-key>
maat ticket complete <ticket-id> --project <project-key> --evidence "go test ./... passed"
```

Completion should always include clear evidence: tests, commits, PRs, screenshots, or exact verification notes.

### 7. Sync

If auto-commit and auto-push are enabled in setup, Maat will commit and push storage changes after successful writes.

You can still sync manually:

```sh
maat sync --status
maat sync --message "status(<project-key>): update state"
maat sync --push
```

## How Maat Stores Data

Markdown plus Git is canonical.

- Project state lives in the configured storage repo.
- Events are append-only Markdown files.
- `.maat/index.json` and `.maat/index.sqlite` are local, rebuildable caches.
- SQLite is for fast local search, not shared coordination.
- Agents should write through `maat` instead of hand-editing files.

For many-agent use, each agent, process, or machine can keep its own local `.maat` cache and rebuild it from Markdown whenever needed. Coordination happens through Git commits, pulls, pushes, and append-only state files.

If an index rebuild is stale or temporarily fails, the Markdown write still owns the truth. Human output warns that search may be stale, and `--agent-use` emits a machine-readable warning so the agent can rebuild the index later instead of retrying the write and creating duplicate history.

This product repository ignores `state/`; keep real Maat state in a separate Git-controlled storage repo or in local ignored smoke data.

## Update Or Uninstall

```sh
maat update
maat uninstall --install-dir "$HOME/.local/bin"
```

## TUI

Launch the terminal dashboard:

```sh
maat tui
```

The TUI currently shows projects, status totals, project detail, and ticket lists. Search and timeline views are planned next.

## Migration

Migrate legacy flat project files into the object layout:

```sh
maat migrate plan --json
maat migrate apply --dest /tmp/maat-migrated
```

## Agent Loop

Agents should follow this loop:

1. Run `maat sync --status` or otherwise pull the Maat storage repo.
2. Inspect state with `maat status`, `maat project show`, or `maat search`.
3. Create or claim a ticket before doing material work.
4. Record meaningful progress with `maat ticket comment`.
5. Complete tickets only with evidence.
6. Run `maat validate` and `maat sync --push`.

Generate a fresh setup handoff for an agent at any time:

```sh
maat initialize --storage /absolute/path/to/maat-state
```

## Useful Docs

- [Architecture](docs/architecture.md)
- [Storage Model](docs/storage-model.md)
- [CLI, TUI, And Future UI](docs/cli-tui-ui.md)
- [Agent Protocol](docs/agent-protocol.md)
- [Install](docs/install.md)
- [Development](docs/development.md)

## Build From Source

Source builds are for contributors:

```sh
git clone https://github.com/sunday-studio/maat.git
cd maat
go build -o maat ./cmd/maat
```
