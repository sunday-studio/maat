# Agent Protocol

Agents use Maat as shared project memory.

Use the `maat` CLI. Git plus Markdown is canonical. SQLite, TUI screens, and generated views are rebuildable.

## Setup

Run setup once on the machine:

```sh
maat setup --storage /absolute/path/to/maat-state
```

From a project repo, teach the agent by running:

```sh
maat initialize --project <project-key> --storage /absolute/path/to/maat-state
```

Save the generated snippet into `AGENTS.md`, `CLAUDE.md`, Cursor rules, or the equivalent instruction surface the agent reads before work.

## Agent Loop

Before work:

```sh
maat sync --status
maat status
maat project show <project-key>
maat ticket list --project <project-key>
```

Create or claim work:

```sh
maat goal create <project-key> "<goal title>"
maat ticket create <project-key> "<ticket title>" --goal <goal-id>
maat ticket claim <ticket-id> --project <project-key> --agent "<agent-id>" --ttl 2h
```

Record useful progress:

```sh
maat ticket comment <ticket-id> "short factual progress note" --project <project-key>
maat search "<query>"
```

Finish with evidence:

```sh
maat ticket complete <ticket-id> --project <project-key> --evidence "tests, commit, PR, or exact verification"
maat validate
maat sync --message "status(<project-key>): update maat" --push
```

## Commands

Read commands:

```sh
maat status [--storage <path>] [--json]
maat projects [--storage <path>] [--json]
maat project show <project-key> [--storage <path>] [--json]
maat ticket list [--project <project-key>] [--storage <path>] [--json]
maat ticket show <ticket-id> [--project <project-key>] [--storage <path>] [--json]
maat search <query> [--storage <path>] [--json]
maat validate [--storage <path>] [--json]
```

Write commands:

```sh
maat project link [source-path] [--storage <path>] [--key <project-key>] [--name <display-name>] [--json]
maat goal create [project-key] <title> [--storage <path>] [--json]
maat ticket create [project-key] <title> [--goal <goal-id>] [--storage <path>] [--json]
maat ticket claim <ticket-id> [--agent <agent-id>] [--ttl <duration>] [--project <project-key>] [--storage <path>] [--json]
maat ticket comment <ticket-id> <comment> [--project <project-key>] [--storage <path>] [--json]
maat ticket complete <ticket-id> --evidence <text> [--project <project-key>] [--storage <path>] [--json]
maat sync [--storage <path>] [--message <msg>] [--push] [--status] [--json]
```

Use `--agent-use` when an agent needs newline-delimited progress updates instead of human-readable output.

## Rules

- Create or claim a ticket before material work.
- Add comments for meaningful progress, blockers, and handoffs.
- Complete tickets only with clear evidence.
- Do not retry a write just because index refresh failed; rebuild the index later.
- Commit finished product changes in the product repo.
- Commit and push Maat storage changes.
- Do not store primary project state outside Markdown.
