# Agent Integrations

Maat should work with any agent that can operate on a Git repository.

## Universal Contract

Agents need only four capabilities:

1. Read Markdown files.
2. Create Markdown object and event files.
3. Run `matt` commands when available.
4. Commit and sync with Git.

No agent-specific database or API is required for the core system.

## Codex

Codex can use Maat directly:

1. Open the Maat repo as a workspace.
2. Read `AGENTS.md`.
3. Prefer `matt` commands when the binary exists.
4. Otherwise create object and event files following the docs.
4. Commit and push.

Recommended instruction:

```text
Before and after material work, update Maat according to AGENTS.md. Prefer the `matt` CLI. Create or claim a ticket before work, record useful progress, complete work only with evidence, and sync afterward.
```

## Claude

Claude-style agents can use the same contract when they have filesystem and Git access.

Recommended instruction:

```text
Use the Maat repository as the canonical project tracker. Prefer the `matt` CLI. Create or claim a ticket before work, record useful progress, complete work only with evidence, and sync afterward.
```

## Agents Without Git Push Access

If an agent cannot push:

1. Write a complete handoff report in `reports/`.
2. Include proposed object changes.
3. Include proposed event files.
4. Ask a Git-capable agent to apply and commit the update.

## Future Adapter Ideas

The Markdown and Git core should remain the source of truth. Optional adapters can be layered on top:

- MCP server exposing project, goal, ticket, and event operations.
- CLI wrapper that validates templates before commit.
- GitHub Action that checks ledger/project consistency.
- Static dashboard generated from Markdown.
- Local menu bar watcher for unread reports and blockers.
