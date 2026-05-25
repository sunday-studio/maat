# Agent Integrations

Maat should work with any agent that can operate on a Git repository.

## Universal Contract

Agents need only four capabilities:

1. Read Markdown files.
2. Edit Markdown files.
3. Append a ledger event.
4. Commit and sync with Git.

No agent-specific database or API is required for the core system.

## Codex

Codex can use Maat directly:

1. Open the Maat repo as a workspace.
2. Read `AGENTS.md`.
3. Update project state and ledger files.
4. Commit and push.

Recommended instruction:

```text
Before and after material work, update the Maat repository according to AGENTS.md.
```

## Claude

Claude-style agents can use the same contract when they have filesystem and Git access.

Recommended instruction:

```text
Use the Maat repository as the canonical project tracker. Update Markdown project state and append ledger events for every meaningful project-management change.
```

## Agents Without Git Push Access

If an agent cannot push:

1. Write a complete handoff report in `reports/`.
2. Include proposed project-file changes.
3. Include proposed ledger events.
4. Ask a Git-capable agent to apply and commit the update.

## Future Adapter Ideas

The Markdown and Git core should remain the source of truth. Optional adapters can be layered on top:

- MCP server exposing project, goal, task, and ledger operations.
- CLI wrapper that validates templates before commit.
- GitHub Action that checks ledger/project consistency.
- Static dashboard generated from Markdown.
- Local menu bar watcher for unread reports and blockers.
