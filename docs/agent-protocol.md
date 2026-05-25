# Agent Protocol

Agents should use Maat as their shared project memory.

The preferred path is through the `matt` CLI or future MCP tools. Direct Markdown edits are acceptable during early development, but they should follow the same object and event rules in `docs/storage-model.md`.

Git plus Markdown is canonical. SQLite, TUI screens, and generated views are rebuildable.

## Agent Lifecycle

Before work:

1. Sync or pull the Maat storage repo.
2. Run `matt validate`.
3. Query project state with `matt status`, `matt project show`, or `matt search`.
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
4. Run `matt validate`.
5. Sync and commit Maat changes.

## Current Agent Commands

These commands are available now and are safe for agents to rely on:

```sh
matt init [storage-path]
matt storage link <storage-path>
matt index rebuild [--storage <path>]
matt status [--storage <path>] [--json]
matt projects [--storage <path>] [--json]
matt project show <project-id> [--storage <path>]
matt validate [--storage <path>] [--json]
matt search <query> [--storage <path>] [--json]
matt tui [--storage <path>]
```

Use JSON output when another tool or agent needs to parse results.

## Next Write Commands

These commands define the intended agent write protocol. Until they are wired into `cmd/matt`, agents should use the same object files and event rules directly in the Maat storage repo.

Typical start:

```sh
matt sync
matt project show orion
matt ticket list --project orion --status active
matt ticket claim T-20260525-190700-b91c --agent codex --ttl 2h
```

New work:

```sh
matt goal create orion "Improve agent health clarity"
matt ticket create orion --goal G-20260525-190533-a7f3 "Separate agent availability from monitor health"
```

Progress:

```sh
matt ticket comment T-20260525-190700-b91c "Status rollup combines monitor failures with agent liveness."
matt ticket status T-20260525-190700-b91c active
```

Completion:

```sh
matt ticket complete T-20260525-190700-b91c --evidence "go test ./... passed"
matt sync
```

Migration and setup:

```sh
matt project link
matt migrate plan --storage <path>
matt migrate apply --storage <path> --destination <path>
```

Sync:

```sh
matt sync
matt sync --push
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
Use Maat as the canonical project memory for this repo. Before material work, run `matt sync` if available, then inspect state with `matt status`, `matt project show <project>`, or `matt search <query>`. Create or claim a ticket before working. Record meaningful progress with ticket comments or events. When finished, complete the ticket with evidence, validate Maat, and sync. Do not mark work done without evidence.
```

CLI:

```sh
matt agent instructions
```

prints exactly this snippet. Use `matt agent instructions --output AGENTS.md` when creating a new project-level agent instruction file.
