# Agent Protocol

Agents should use Maat as their shared project memory.

The preferred path is through the `maat` CLI or future MCP tools. Direct Markdown edits are a fallback for agents that cannot run the CLI, and they should follow the same object and event rules in `docs/storage-model.md`.

Git plus Markdown is canonical. SQLite, TUI screens, and generated views are rebuildable.

## Agent Lifecycle

Before work:

1. Sync or pull the Maat storage repo.
2. Run `maat validate`.
3. Query project state with `maat status`, `maat project show`, or `maat search`.
4. Check active goals and open tickets.
5. Claim or create a ticket when write commands are available.

During work:

1. Comment on meaningful discoveries.
2. Record blockers quickly.
3. Keep claims renewed if the work continues.

After work:

1. Complete or update the ticket.
2. Attach evidence.
3. Record decisions if the work changed direction.
4. Run `maat validate`.
5. Sync and commit Maat changes.

## Current Agent Commands

These commands are available now and are safe for agents to rely on:

```sh
maat setup --storage <absolute-git-repo-path>
maat index rebuild [--storage <path>]
maat status [--storage <path>] [--json]
maat projects [--storage <path>] [--json]
maat project show <project-id> [--storage <path>]
maat validate [--storage <path>] [--json]
maat search <query> [--storage <path>] [--json]
maat project link [source-path] [--storage <path>] [--key <project-key>] [--name <display-name>] [--json]
maat goal create [project-key] <title> [--storage <path>] [--json]
maat ticket create [project-key] <title> [--goal <goal-id>] [--storage <path>] [--json]
maat ticket list [--project <project-key>] [--storage <path>] [--json]
maat ticket show <ticket-id> [--project <project-key>] [--storage <path>] [--json]
maat ticket claim <ticket-id> [--agent <agent>] [--ttl <duration>] [--project <project-key>] [--storage <path>] [--json]
maat ticket comment <ticket-id> <comment> [--project <project-key>] [--storage <path>] [--json]
maat ticket complete <ticket-id> --evidence <text> [--project <project-key>] [--storage <path>] [--json]
maat sync [--storage <path>] [--message <msg>] [--push] [--status] [--json]
maat tui [--storage <path>]
```

Use JSON output when another tool or agent needs to parse results.

## Write Command Flow

These commands define the current agent write protocol.

Typical start:

```sh
maat sync
maat project show maat
maat ticket list --project maat
maat ticket claim T-20260525-190700-b91c --project maat --agent codex --ttl 2h
```

New work:

```sh
maat goal create maat "Improve agent handoff clarity"
maat ticket create maat "Separate project state from product examples" --goal G-20260525-190533-a7f3
```

Progress:

```sh
maat ticket comment T-20260525-190700-b91c "Status rollup combines monitor failures with agent liveness." --project maat
```

Completion:

```sh
maat ticket complete T-20260525-190700-b91c --evidence "go test ./... passed" --project maat
maat sync
```

Migration and setup:

```sh
maat project link
maat migrate plan --storage <path>
maat migrate apply --storage <path> --dest <path>
```

Future status update commands may add direct transitions without completing tickets.

Sync:

```sh
maat sync
maat sync --push
```

## Evidence Rules

Agents should not mark work done without evidence.

Evidence can be:

- test command and result
- build command and result
- file path to changed docs
- linked commit
- linked PR
- screenshot path
- explicit human confirmation

## Comments

Comments are for useful state, not stream-of-consciousness.

Good comments:

- found root cause
- narrowed scope
- discovered blocker
- explained tradeoff
- left handoff details

Avoid comments that only say an agent is "working on it" unless the comment includes useful context.

## Claims

Claims are soft leases.

They should include:

- agent ID
- ticket ID
- claimed time
- expiration time

Other agents should avoid claimed tickets unless the claim expired or the user asks them to take over.

## Conflict Behavior

When an agent sees conflicting state:

1. Preserve factual updates from all agents.
2. Prefer appending a corrective event over rewriting history.
3. Do not mark done unless the done state has evidence.
4. If status is ambiguous, use `waiting` or `needs-review`.

## Instruction Snippet

This is the minimal snippet Maat should install into project repos. It is intentionally short enough to paste into an existing `AGENTS.md` without taking over the whole file:

```text
Use Maat as the canonical project memory for this repo. Before material work, run `maat sync` if available, then inspect state with `maat status`, `maat project show <project>`, or `maat search <query>`. Create or claim a ticket before working. Record meaningful progress with ticket comments or events. When finished, complete the ticket with evidence, validate Maat, and sync. Do not mark work done without evidence.
```

## Agent Setup Document

Use this when handing Maat to a new agent, a hosted agent, or a skill/instruction system that needs the full operating protocol:

```sh
maat initialize --storage /absolute/path/to/maat-state
```

The setup document explains how to link storage, tells the agent to save the Maat rule into `AGENTS.md` or the equivalent instruction surface it reads, and lists the commands to run before planning, during work, and when finishing with evidence.
