# Agent Integrations

Maat should work with any agent that can operate on a Git repository.

## Universal Contract

Agents need only four capabilities:

1. Read Markdown files.
2. Create Markdown object and event files.
3. Run `maat` commands when available.
4. Commit and sync with Git.

No agent-specific database or API is required for the core system.

## Minimal Project Snippet

Install this into each project repo's `AGENTS.md` or equivalent agent instructions:

```text
Use Maat as the canonical project memory for this repo. Before material work, run `maat sync` if available, then inspect state with `maat status`, `maat project show <project>`, or `maat search <query>`. Create or claim a ticket before working. Record meaningful progress with ticket comments or events. When finished, complete the ticket with evidence, validate Maat, and sync. Do not mark work done without evidence.
```

For a full agent onboarding document, run:

```sh
maat initialize --project <project-key> --storage /absolute/path/to/maat-state
```

Use it when an agent needs instructions for linking storage, saving the Maat rule into Codex, Claude Code, Cursor, cloud agent instructions, or a generic skill file, and following the Maat command loop.

## Current CLI Surface

Current commands suitable for integrations:

```sh
maat init [storage-path]
maat storage link <storage-path>
maat index rebuild [--storage <path>]
maat status [--storage <path>] [--json]
maat projects [--storage <path>] [--json]
maat project show <project-id> [--storage <path>]
maat validate [--storage <path>] [--json]
maat search <query> [--storage <path>] [--json]
maat initialize [--project <project-key>] [--storage <path>] [--json]
maat tui [--storage <path>]
```

Write commands available for integrations:

```sh
maat project link
maat goal create <project> <title>
maat ticket create <project> [--goal <goal-id>] <title>
maat ticket claim <project> <ticket-id> --agent <agent-id> --ttl <duration>
maat ticket comment <project> <ticket-id> <comment>
maat ticket complete <project> <ticket-id> --evidence <evidence>
maat sync [--push]
```

## Codex

Codex can use Maat directly:

1. Open the Maat repo as a workspace.
2. Read `AGENTS.md`.
3. Prefer `maat` commands when the binary exists.
4. Otherwise create object and event files following the docs.
5. Commit and push.

Recommended instruction:

```text
Before and after material work, update Maat according to AGENTS.md. Prefer the `maat` CLI. Use current read commands for discovery and validation. When write commands are unavailable, create target-layout object and event files directly. Create or claim a ticket before work, record useful progress, complete work only with evidence, and sync afterward.
```

## Claude

Claude-style agents can use the same contract when they have filesystem and Git access.

Recommended instruction:

```text
Use the Maat repository as the canonical project tracker. Prefer the `maat` CLI. Use current read commands for discovery and validation. When write commands are unavailable, create target-layout object and event files directly. Create or claim a ticket before work, record useful progress, complete work only with evidence, and sync afterward.
```

## Agents Without Git Push Access

If an agent cannot push:

1. Write a complete handoff report in `state/reports/`.
2. Include proposed object changes.
3. Include proposed event files.
4. Ask a Git-capable agent to apply and commit the update.

## Future Adapter Ideas

The Markdown and Git core should remain the source of truth. Optional adapters can be layered on top:

- MCP server exposing project, goal, ticket, and event operations.
- CLI wrapper that validates templates before commit.
- GitHub Action that checks state consistency.
- Static dashboard generated from Markdown.
- Local menu bar watcher for unread reports and blockers.
