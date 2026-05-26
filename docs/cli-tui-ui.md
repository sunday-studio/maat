# CLI, TUI, And UI

Maat has three human-facing interfaces and one future agent-facing adapter.

- CLI: command surface for agents and power users.
- TUI: Bubble Tea terminal dashboard.
- Web UI: local browser dashboard.
- MCP: typed tool adapter for agents.

All interfaces should use the same core operations.

## CLI

The CLI binary is `maat`.

It should be easy to install on a new machine, link to a Git-controlled storage directory, rebuild the local index, and start querying.

### Setup

```sh
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
maat project status maat active
```

`maat project link` should detect the current Git repository when run from inside a source repo.

### Goal Commands

```sh
maat goal create maat "Ship first deploy"
maat goal create "Ship first deploy"
maat goal list maat
maat goal show G-20260525-190533-a7f3
maat goal status G-20260525-190533-a7f3 done --evidence "all tickets complete"
```

When run from inside a repo linked with `maat project link`, create commands can infer the project key.

### Ticket Commands

```sh
maat ticket create maat "Fix deploy doc link"
maat ticket create maat --goal G-20260525-190533-a7f3 "Verify installer"
maat ticket create "Fix deploy doc link"
maat ticket show T-20260525-190700-b91c
maat ticket claim T-20260525-190700-b91c --agent codex --ttl 2h
maat ticket comment T-20260525-190700-b91c "Found issue in launchd path."
maat ticket status T-20260525-190700-b91c waiting --reason "needs credentials"
maat ticket complete T-20260525-190700-b91c --evidence "smoke test passed"
```

Tickets may stand alone or belong to a goal.

### Query Commands

```sh
maat status
maat active
maat blocked
maat stale
maat timeline --today
maat search "agent health"
maat report daily
```

Most query commands should support:

```sh
--json
--project <project>
--since <time>
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
maat pull
maat push
maat validate
maat index rebuild
```

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

## Web UI

The web UI launches with:

```sh
maat ui
```

The first version can be local-only.

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

## MCP Adapter

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
