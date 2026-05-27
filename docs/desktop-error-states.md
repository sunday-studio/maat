# Desktop Error States

The macOS desktop app is a UI over the `maat` CLI. It should not reinterpret
Markdown storage, Git sync, validation, or indexing as app-owned state. The app
should run CLI commands, keep the raw process diagnostics, parse successful JSON
when available, and map known failures or warnings into recoverable UI states.

This document expands the error-handling contract in
`docs/macos-app-architecture.md`.

## Command Runner Result

Every desktop command invocation should produce one normalized result before UI
code updates view state.

```ts
type CommandRunnerResult<T = unknown> = {
  commandId: string
  argv: string[]
  startedAt: string
  finishedAt: string
  exitCode: number | null
  timedOut: boolean
  stdout: string
  stderr: string
  json: T | null
  jsonParseError?: string
  classification: DesktopCommandClassification
}

type DesktopCommandClassification = {
  severity: "ok" | "warning" | "error"
  state: DesktopErrorStateName
  durableWrite: boolean
  retryable: boolean
  diagnostics: DesktopDiagnostics
  recovery: DesktopRecoveryAction[]
}

type DesktopDiagnostics = {
  summary: string
  command: string
  stdout: string
  stderr: string
  parsedJson: unknown | null
  parseError?: string
}

type DesktopRecoveryAction = {
  id: string
  label: string
  kind: "retry" | "open-setup" | "choose-path" | "run-command" | "open-folder" | "external"
  command?: string[]
  targetPath?: string
}
```

The runner should keep `stdout`, `stderr`, the exact argv, exit code, and parsed
JSON details for troubleshooting even when the UI shows a short recovery message.
The diagnostics disclosure should be available from every error or warning state
and should copy the exact command, stderr, stdout, and parsed JSON.

Use `--json` for one-shot desktop reads and writes. Use `--agent-use` only for
streaming progress views because it emits newline-delimited updates and cannot be
combined with `--json`.

## UI State Names

Known states should be explicit so product behavior remains testable.

| State | Severity | When detected | Primary UI | Recovery actions |
| --- | --- | --- | --- | --- |
| `cli.missing` | error | Spawning the configured `maat` binary fails with `ENOENT`, macOS reports the file missing, or `maat version --json` cannot be started. | Show setup-required banner in the app shell. | Install bundled CLI, then retry `maat version --json`; choose CLI path in advanced settings. |
| `setup.missing` | error | CLI reports missing config or `maat setup doctor --json` returns `storage_configured` with `status: "error"`. | Open setup assistant at storage selection. | Choose/create/clone storage repo; run `maat setup --storage <path> --actor <name> --json`; rerun doctor. |
| `storage.invalid` | error | JSON `storageAccessFailureResult` is emitted, setup rejects the path, doctor reports `storage_path`, `storage_writable`, `git_repository`, or `validation` as `error`, or read commands fail while opening storage. | Show storage repair view with selected path and reason. | Choose another path; create or clone a repo; fix permissions; run `maat setup doctor --storage <path> --fix --json` when a check is fixable. |
| `git.credentials` | error | Git stderr or CLI error contains credential/authentication failures during pull, push, fetch, or clone, such as `Permission denied (publickey)`, `Authentication failed`, `could not read Username`, or keychain/SSH agent denial. | Show Git credential repair view. | Open storage repo in Finder or terminal; retry the failed command after credentials are fixed; open setup doctor diagnostics. |
| `index.warning` | warning | A write JSON result has `index_refreshed: false` and non-empty `index_warning`, or an agent update has `step: "index.refresh"` and `status: "warning"`. | Keep the new or changed object visible and show a rebuild warning. | Run `maat index rebuild --storage <path>` or `maat setup doctor --storage <path> --fix --json`; do not retry the write. |
| `merge.conflict` | error | Git stderr or status indicates unresolved merge/rebase conflicts, such as `CONFLICT`, `needs merge`, `unmerged files`, `You are in the middle of a merge`, or porcelain index/worktree status `U`. | Stop write controls for the affected storage repo and show conflict resolution guidance. | Open storage repo path; resolve with Git; run `maat validate --json`; retry the original read or sync after Git is clean. |
| `cli.unclassified` | error | The command exits non-zero and none of the known states match. | Show generic command failure with diagnostics expanded one level. | Retry the command if it is read-only; run setup doctor; copy diagnostics. |
| `ok` | ok | Exit code is zero and no warning fields are present. | Update normal view state. | None. |

These states are mutually exclusive for the main surface. If multiple conditions
are present, prefer the state closest to the user's next safe action:
`merge.conflict`, `git.credentials`, `setup.missing`, `storage.invalid`,
`cli.missing`, `index.warning`, `cli.unclassified`, then `ok`.

## Condition Detection

Detection should combine process-level facts, structured CLI JSON, and conservative
stderr matching.

### Missing CLI

Detect before storage logic runs:

- configured binary path is empty;
- spawn fails because the file does not exist or is not executable;
- `maat version --json` cannot be started.

The CLI manager should install the bundled binary into app support and rerun
`maat version --json`. If install fails, keep the user in `cli.missing` with the
install diagnostics disclosed.

### Missing Setup

The setup assistant should run:

```sh
maat setup doctor --json
```

If JSON includes a check with `id: "storage_configured"` and `status: "error"`,
the app should enter `setup.missing`. The UI should not ask the user to repair
Markdown files yet because there is no selected storage repo. It should ask for a
storage repo path and then run:

```sh
maat setup --storage <path> --actor <name> --json
maat setup doctor --storage <path> --fix --json
```

### Invalid Storage

Read and setup commands may emit structured storage failures:

```json
{
  "action": "status.load",
  "ok": false,
  "error": {
    "storage_path": "/path/to/storage",
    "operation": "read status",
    "error": "stat /path/to/storage: permission denied",
    "remediation": "Choose a readable and writable Maat storage repo, grant the agent access to this path, or pass --storage <path>.",
    "retry": "retry after storage permissions or path selection are fixed"
  }
}
```

The app should map this to `storage.invalid`, display the path and operation, and
offer path selection or permission repair. It should also map doctor checks with
`status: "error"` for `storage_path`, `storage_writable`, `git_repository`, or
`validation` to the same state, using each check's `message`, `path`,
`can_fix`, and `suggestion`.

### Git Credential Failures

Credential failures usually come from Git stderr wrapped in CLI stderr. Detect
these on failed Git operations, including read auto-pull warnings and write
auto-sync warnings. Match case-insensitively against:

- `Permission denied (publickey)`;
- `Authentication failed`;
- `could not read Username`;
- `terminal prompts disabled`;
- `repository not found` when the remote URL is present;
- `sign_and_send_pubkey`;
- `keychain` or `ssh-agent` failures paired with `permission denied`.

Map these to `git.credentials`, show the failed Git action, and offer retry only
after the user has repaired credentials. The app should not store Git
credentials; Git should continue to use the user's credential helper, SSH agent,
or keychain.

### Index Failures

Write commands return a successful JSON result after the Markdown write and event
write persist. If index refresh fails, the result still has a successful action,
for example `goal.created` or `ticket.created`, with:

```json
{
  "action": "goal.created",
  "project_key": "maat",
  "goal_id": "G-...",
  "event_id": "E-...",
  "index_refreshed": false,
  "index_warning": "index refresh failed after state write persisted: ..."
}
```

The app must map this to `index.warning`, not a failed write state. The object ID
and event ID are durable evidence. Keep the new state in the UI and offer:

```sh
maat index rebuild --storage <path>
```

or doctor fix:

```sh
maat setup doctor --storage <path> --fix --json
```

The retry button for the original write must be hidden or disabled in this state.

### Merge Conflicts

Merge conflicts are storage-level blockers. Detect them from failed `sync`,
auto-pull, auto-sync, or setup doctor Git inspection results when stderr contains
conflict language, and from Git status details when available. Treat porcelain
statuses containing `U` in either index or worktree position as conflicts.

While in `merge.conflict`, disable mutating actions for the storage repo. Reads
may continue only if they can read local Markdown without invoking Git. The UI
should show the storage repo path, explain that Git conflict resolution is needed
outside Maat, and keep diagnostics available.

## Durable Writes And Duplicate Prevention

A desktop write should be considered durable when the CLI exits zero and returns
a write result with a stable object or event ID:

- `goal.created`: `goal_id` and `event_id`;
- `ticket.created`: `ticket_id` and `event_id`;
- `ticket.claimed`, `ticket.commented`, `ticket.completed`: `ticket_id` and
  `event_id`;
- `project.linked` or setup-style project registration: returned project and
  event details when present.

When `index_refreshed` is false, the app should:

1. Commit the returned object/event IDs into frontend state.
2. Mark search and index-backed lists as stale.
3. Offer index rebuild as the only primary recovery.
4. Avoid resubmitting the original write command.
5. After rebuild succeeds, refresh reads with `maat status --json`, project or
   ticket detail commands, and search as needed.

This prevents duplicate ticket, goal, comment, claim, or completion events. The
index is a rebuildable cache; Markdown and Git remain authoritative.

If the CLI exits non-zero before returning durable IDs, the write result is not
known to be durable. The UI may offer retry, but diagnostics should stay visible
because a partial failure might still have written files. Implementation should
prefer future CLI idempotency keys for desktop-originated writes, but the first
desktop design must rely on returned IDs and the index warning contract.

## Diagnostics Disclosure

Every non-`ok` state should include a compact user message plus a diagnostics
disclosure. The disclosure should include:

- exact command argv, with secrets redacted if future commands accept them;
- exit code or timeout status;
- stderr exactly as captured;
- stdout exactly as captured;
- parsed JSON object, if any;
- JSON parse error, if stdout was not valid JSON;
- storage path and project or ticket ID when known.

For known states, show the recovery message first and keep raw CLI details one
click away. For `cli.unclassified`, open diagnostics by default because the UI
does not know a safe specific fix.

## Retry And Fix Actions

Recovery actions should run explicit commands and then reclassify the new result.

| State | Safe automatic action | User-confirmed action |
| --- | --- | --- |
| `cli.missing` | Install bundled CLI into app support and verify with `maat version --json`. | Choose alternate CLI path. |
| `setup.missing` | None until a path is selected. | Run `maat setup --storage <path> --actor <name> --json`. |
| `storage.invalid` | Rerun doctor after path selection. | Create directory, clone repo, fix permissions, or run doctor fix when `can_fix` is true. |
| `git.credentials` | None. | Retry failed pull, push, clone, or sync after external credential repair. |
| `index.warning` | Run `maat index rebuild --storage <path>`. | Run setup doctor fix instead. |
| `merge.conflict` | None. | Open folder or terminal for manual Git resolution, then run `maat validate --json` and retry sync. |
| `cli.unclassified` | Retry only if the command was read-only. | Copy diagnostics or run setup doctor. |

Mutating retries should require confidence that no durable IDs were returned. If
the result includes a durable object or event ID, the action should be a refresh
or rebuild, not a retry.

## Implementation Notes

- Store only the latest diagnostics needed for UI troubleshooting; do not make
  diagnostics a new source of product state.
- Use state names in telemetry and tests exactly as written in this document.
- Prefer structured JSON fields over stderr text whenever both are available.
- Treat doctor `warning` checks with `requires_approval: true` as user decisions,
  not automatic repairs.
- Keep write controls disabled while `merge.conflict` is active for a storage
  repo.
- After any recovery command succeeds, refresh the affected views from CLI JSON
  instead of trusting stale frontend state.
