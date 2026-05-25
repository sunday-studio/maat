# Agent Protocol

Agents should use Maat as their shared project memory.

The preferred path is through the `matt` CLI or future MCP tools. Direct Markdown edits are acceptable during early development, but they should follow the same object and event rules.

## Agent Lifecycle

Before work:

1. Sync Maat.
2. Query project state.
3. Check active goals and open tickets.
4. Claim or create a ticket.

During work:

1. Comment on meaningful discoveries.
2. Record blockers quickly.
3. Keep claims renewed if the work continues.

After work:

1. Complete or update the ticket.
2. Attach evidence.
3. Record decisions if the work changed direction.
4. Sync Maat.

## Agent Commands

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

This is the future snippet Maat can install into project repos:

```text
Use Maat as the canonical project memory. Before starting material work, run `matt sync` and inspect the relevant project and tickets. Create or claim a ticket before working. Record meaningful progress as comments or events. When finished, complete the ticket with evidence and sync Maat. Do not mark work done without evidence.
```
