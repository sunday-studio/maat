# CLI, TUI, And Future UI

Maat currently has a CLI and a Bubble Tea TUI. A local web UI and MCP adapter are planned.

- CLI: command surface for agents and power users.
- TUI: Bubble Tea terminal dashboard.
- Future web UI: local browser dashboard.
- Future MCP: typed tool adapter for agents.

All interfaces should use the same core operations.

## CLI

The CLI binary is `maat`.

It should be easy to install on a new machine, link to a Git-controlled storage directory, register a project repo, rebuild the local index, and start querying.

### Setup

```sh
maat setup
maat setup --storage /absolute/path/to/maat-state
maat index rebuild
```

### Project Commands

```sh
maat projects
maat project show maat
maat project link
maat project link /absolute/path/to/source-repo
maat project link /absolute/path/to/source-repo --key maat --name "Maat"
```

`maat project link` detects the current Git repository when run from inside a source repo.

### Goal Commands

Current goal commands:

```sh
maat goal create maat "Ship first deploy"
maat goal create "Ship first deploy"
```

When run from inside a repo linked with `maat project link`, create commands can infer the project key.

Future goal commands:

- list goals
- show a goal
- update goal status with evidence

### Ticket Commands

Current ticket commands:

```sh
maat ticket create maat "Fix deploy doc link"
maat ticket create maat "Verify installer" --goal G-20260525-190533-a7f3
maat ticket create "Fix deploy doc link"
maat ticket list --project maat
maat ticket show T-20260525-190700-b91c
maat ticket claim T-20260525-190700-b91c --project maat --agent codex --ttl 2h
maat ticket comment T-20260525-190700-b91c "Found issue in launchd path." --project maat
maat ticket complete T-20260525-190700-b91c --evidence "smoke test passed" --project maat
```

Tickets may stand alone or belong to a goal.

Future ticket commands may add direct status transitions for blocked or waiting work.

### Query Commands

Current query commands:

```sh
maat status
maat projects
maat project show maat
maat ticket list --project maat
maat ticket show T-20260525-190700-b91c --project maat
maat search "agent health"
```

Future query commands:

- active work views
- blocked work views
- stale work views
- timeline views
- report generation

Current query commands support `--json` where a structured final result is useful. Project-scoped commands also support `--project <project>` where needed:

```sh
--json
--project <project>
```

### Output Modes

The default CLI output is for humans. It should use concise progress states such as `[run]`, `[ok]`, and `[warn]`, plus ANSI color when supported by the terminal. Color can be forced with `MAAT_COLOR=always` or disabled with `MAAT_COLOR=never` or `NO_COLOR=1`.

`--json` returns the command's final structured result and should not include progress text.

`--agent-use` is for agents that need progress without human prose. It emits newline-delimited JSON updates:

```json
{"type":"maat.update","step":"sync.start","status":"running","message":"checking git storage"}
{"type":"maat.update","step":"sync.ready","status":"ok","message":"sync complete","data":{}}
```

`--agent-use` cannot be combined with `--json`.

When a write command updates Markdown successfully but the local search index cannot be refreshed, the command should not ask the agent to repeat the write. Human output should show a warning that the index is stale. `--agent-use` should emit a warning update with the index failure so the agent can run `maat index rebuild` later without duplicating project history.

### Sync Commands

```sh
maat sync
maat sync --push
maat validate
maat index rebuild
```

Future sync shorthands may split pull and push into direct commands.

The normal agent write path should validate and commit. Push can be opt-in or configured by policy.

Sync and index commands operate on the local checkout. SQLite is not a coordination service; agents coordinate by syncing the Git-backed Markdown state.

## TUI

The TUI launches with:

```sh
maat tui
```

Use Charmbracelet Bubble Tea for the application model, Bubbles for common components, and Lip Gloss for styling.

### TUI Views

- Projects
- Active tickets
- Blocked tickets
- Stale work
- Timeline
- Search
- Decisions
- Agent activity
- Sync/index status

### TUI Interaction

The TUI should optimize for scanning and quick navigation:

- fuzzy project picker
- filterable ticket list
- split-pane detail view
- search overlay
- status badges
- keyboard-first navigation
- one-key copy of object IDs

The TUI should not become the only way to perform actions. Any TUI mutation should map to a CLI/core operation.

## Future Web UI

The first web UI version can be local-only and should launch from the `maat` binary when it lands.

### Web Views

- project overview
- project detail
- goal detail
- ticket detail
- global timeline
- search page
- blocked/stale queue
- decisions
- reports
- agent activity

The web UI should read primarily from SQLite for speed and ask the core layer to perform writes.

If the SQLite cache is stale, missing, or being rebuilt, the UI should surface that state and offer a rebuild path rather than treating it as lost project data.

## Future MCP Adapter

MCP should expose the same safe operations agents need:

```text
maat_list_projects
maat_get_project
maat_create_goal
maat_create_ticket
maat_claim_ticket
maat_comment_ticket
maat_complete_ticket
maat_search
maat_list_blocked
maat_sync
```

The MCP adapter should not bypass validation, event creation, or Git sync rules.
