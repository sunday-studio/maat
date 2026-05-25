# CLI, TUI, And UI

Maat has three human-facing interfaces and one future agent-facing adapter.

- CLI: command surface for agents and power users.
- TUI: Bubble Tea terminal dashboard.
- Web UI: local browser dashboard.
- MCP: typed tool adapter for agents.

All interfaces should use the same core operations.

## CLI

The CLI binary is `matt`.

It should be easy to install on a new machine, link to a Git-controlled storage directory, rebuild the local index, and start querying.

### Setup

```sh
matt init
matt storage link /absolute/path/to/maat-state
matt index rebuild
```

### Project Commands

```sh
matt projects
matt project show orion
matt project link
matt project link /absolute/path/to/source-repo
matt project link /absolute/path/to/source-repo --key orion --name "Orion"
matt project status orion active
```

`matt project link` should detect the current Git repository when run from inside a source repo.

### Goal Commands

```sh
matt goal create orion "Ship first deploy"
matt goal create "Ship first deploy"
matt goal list orion
matt goal show G-20260525-190533-a7f3
matt goal status G-20260525-190533-a7f3 done --evidence "all tickets complete"
```

When run from inside a repo linked with `matt project link`, create commands can infer the project key.

### Ticket Commands

```sh
matt ticket create orion "Fix deploy doc link"
matt ticket create orion --goal G-20260525-190533-a7f3 "Verify installer"
matt ticket create "Fix deploy doc link"
matt ticket show T-20260525-190700-b91c
matt ticket claim T-20260525-190700-b91c --agent codex --ttl 2h
matt ticket comment T-20260525-190700-b91c "Found issue in launchd path."
matt ticket status T-20260525-190700-b91c waiting --reason "needs credentials"
matt ticket complete T-20260525-190700-b91c --evidence "smoke test passed"
```

Tickets may stand alone or belong to a goal.

### Query Commands

```sh
matt status
matt active
matt blocked
matt stale
matt timeline --today
matt search "agent health"
matt report daily
```

Most query commands should support:

```sh
--json
--project <project>
--since <time>
```

### Sync Commands

```sh
matt sync
matt pull
matt push
matt validate
matt index rebuild
```

The normal agent write path should validate and commit. Push can be opt-in or configured by policy.

## TUI

The TUI launches with:

```sh
matt tui
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
matt ui
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
