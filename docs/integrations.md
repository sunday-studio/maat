# Agent Integrations

Maat should work with any agent that can operate on a Git repository.

## Universal Contract

Agents need only four capabilities:

1. Read Markdown files.
2. Create Markdown object and event files.
3. Run `matt` commands when available.
4. Commit and sync with Git.

No agent-specific database or API is required for the core system.

## Minimal Project Snippet

Install this into each project repo's `AGENTS.md` or equivalent agent instructions:

```text
Use Maat as the canonical project memory for this repo. Before material work, run `matt sync` if available, then inspect state with `matt status`, `matt project show <project>`, or `matt search <query>`. Create or claim a ticket before working. Record meaningful progress with ticket comments or events. When finished, complete the ticket with evidence, validate Maat, and sync. Do not mark work done without evidence.
```

`matt agent instructions` prints this snippet so project setup can stay copy/pasteable and consistent. Use `--output <path>` to write it directly.

For a full agent onboarding document, run:

```sh
matt agent initialize --project <project-key> --storage /absolute/path/to/maat-state
```

`matt initialize` is a top-level alias for the same setup document. Use it when an agent needs instructions for linking storage, installing the Maat rule into Codex, Claude Code, cloud agent instructions, or a generic skill file, and following the Maat command loop.

## Current CLI Surface

Current commands suitable for integrations:

```sh
matt init [storage-path]
matt storage link <storage-path>
matt index rebuild [--storage <path>]
matt status [--storage <path>] [--json]
matt projects [--storage <path>] [--json]
matt project show <project-id> [--storage <path>]
matt validate [--storage <path>] [--json]
matt search <query> [--storage <path>] [--json]
matt initialize [--project <project-key>] [--storage <path>] [--output <path>] [--json]
matt agent initialize [--project <project-key>] [--storage <path>] [--output <path>] [--json]
matt agent instructions [--json] [--output <path>]
matt tui [--storage <path>]
```

Write commands available for integrations:

```sh
matt project link
matt goal create <project> <title>
matt ticket create <project> [--goal <goal-id>] <title>
matt ticket claim <project> <ticket-id> --agent <agent-id> --ttl <duration>
matt ticket comment <project> <ticket-id> <comment>
matt ticket complete <project> <ticket-id> --evidence <evidence>
matt sync [--push]
```

## Codex

Codex can use Maat directly:

1. Open the Maat repo as a workspace.
2. Read `AGENTS.md`.
3. Prefer `matt` commands when the binary exists.
4. Otherwise create object and event files following the docs.
5. Commit and push.

Recommended instruction:

```text
Before and after material work, update Maat according to AGENTS.md. Prefer the `matt` CLI. Use current read commands for discovery and validation. When write commands are unavailable, create target-layout object and event files directly. Create or claim a ticket before work, record useful progress, complete work only with evidence, and sync afterward.
```

## Claude

Claude-style agents can use the same contract when they have filesystem and Git access.

Recommended instruction:

```text
Use the Maat repository as the canonical project tracker. Prefer the `matt` CLI. Use current read commands for discovery and validation. When write commands are unavailable, create target-layout object and event files directly. Create or claim a ticket before work, record useful progress, complete work only with evidence, and sync afterward.
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
