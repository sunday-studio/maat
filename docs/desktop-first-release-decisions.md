# Desktop First Release Decisions

Status: accepted defaults for the first macOS desktop release.

This record resolves the open product decisions in
[macOS App Architecture](./macos-app-architecture.md). The defaults below keep
the desktop app thin over the existing `maat` CLI, preserve Markdown plus Git as
the source of truth, and keep terminal usage compatible without making it a
first-run requirement.

## Decision Summary

| Decision | First-release default |
| --- | --- |
| CLI installation | Install only the app-private CLI during first-run setup. Do not prompt for a terminal CLI install in the main setup path. |
| Storage setup | Support creating a new local storage repo and selecting an existing local storage repo. Do not build clone-from-remote setup into the first release. |
| Auto-push | Keep auto-push out of first-run setup. Expose it only in advanced settings after setup. |
| Terminal/TUI view | Focus on native desktop views backed by CLI JSON. Do not embed a read-only terminal or TUI view in the first release. |

## Architecture Anchors

These decisions rely on the macOS app architecture constraints:

- The app is a desktop interface over the existing `maat` CLI, not a new storage
  or sync implementation.
- The app should provide a first-run setup flow for users who do not want to
  start in a terminal.
- The CLI remains the product API through JSON commands.
- The desktop app must stay optional and must not change the storage model.
- The app does not replace the terminal CLI or TUI.
- First-run setup should perform conservative local actions by default, while
  adding remotes, pushing, or enabling auto-push requires explicit user choice.

## Decisions

### CLI Installation

Chosen default: first-run setup installs and uses only the app-private CLI at an
app-controlled path such as:

```text
~/Library/Application Support/maat/bin/maat
```

The first release should not ask every desktop user to install a terminal-facing
CLI into `PATH` during setup. Terminal installation should remain available as a
separate action after setup, using the same binary and the existing CLI install
location such as:

```text
~/.local/bin/maat
```

Rationale:

- The desktop app needs a reliable CLI path that is unaffected by shell profile
  configuration.
- The first-run flow is for users who may not want terminal setup decisions.
- Terminal users already have the documented installer and `maat update` path.
- Keeping terminal install out of the default setup reduces permission,
  symlink, and shell profile edge cases in the first release.

Deferred work:

- Add a post-setup "Install command line tool" action.
- Add terminal install health checks for path writability, symlink validity, and
  installed version drift.
- Add uninstall or repair behavior for the terminal-facing install.

Implementation implications:

- The CLI manager must install, verify, update, and record the app-private
  binary path before any desktop reads or writes.
- Setup should not block on `PATH` detection.
- Settings can show both the app-private CLI path and optional terminal CLI
  status.

### Storage Setup

Chosen default: first-run setup supports:

- creating a new local storage repo;
- selecting an existing local storage repo.

Clone-from-remote is deferred. Users who already have shared storage can clone it
with Git or another tool, then select the local checkout in the app.

Rationale:

- Creating or selecting local storage matches the conservative setup model.
- Built-in clone requires remote URL validation, Git credential handling,
  progress reporting, retries, and clearer recovery states.
- Selecting an already cloned repo still supports shared storage without making
  network and credential behavior part of the first setup milestone.
- This keeps the first release focused on the app-to-CLI contract and storage
  verification.

Deferred work:

- Add a clone-from-remote flow with Git credential guidance.
- Add remote validation, branch selection, progress, cancellation, and retry
  states.
- Add clearer recovery paths for authentication failure, existing non-empty
  target directories, and partial clones.

Implementation implications:

- The setup assistant should create a directory, run `git init` when creating new
  storage, or verify a selected path.
- The setup assistant should run `maat setup --storage <path> --actor <name>
  --json`, then `maat setup doctor --storage <path> --fix --json`.
- Remote configuration should not appear in the first-run happy path.

### Auto-Push

Chosen default: auto-push is not exposed in first-run setup and remains off by
default. The first release may expose auto-push only in advanced settings after
setup, with clear status and failure handling.

Rationale:

- Auto-push publishes project state automatically and needs a confirmed remote,
  credentials, and user intent.
- First-run setup should establish local correctness before enabling background
  publication behavior.
- The install documentation already says auto-push is off by default and should
  be enabled only when the storage remote is confirmed and the agent is allowed
  to publish automatically.

Deferred work:

- Add advanced settings for auto-pull, auto-commit, and auto-push together.
- Add a preflight check that reports remote, upstream, credentials, and
  ahead/behind status before auto-push can be enabled.
- Add per-write and background sync error states for push failures.

Implementation implications:

- First-run setup should not include an auto-push toggle.
- Default config written by desktop setup must leave auto-push disabled unless
  the user later enables it explicitly.
- The sync controller should expose explicit `maat sync --json` and, when
  configured, `maat sync --push --json` actions with visible results.

### Terminal/TUI View

Chosen default: first release focuses on native desktop views backed by CLI JSON
commands. It does not embed a read-only terminal or the Bubble Tea TUI.

Rationale:

- The architecture makes the CLI the product API and says the app does not
  replace the terminal CLI or TUI.
- Native views can map directly to parseable commands such as `maat status
  --json`, `maat projects --json`, `maat ticket list --json`, and `maat search
  <query> --json`.
- Embedding a terminal adds rendering, focus, keyboard, accessibility, color,
  copy/paste, and process lifecycle complexity without improving the core
  desktop contract.
- A read-only TUI view risks duplicating the existing TUI while still being less
  capable than running `maat tui` in a real terminal.

Deferred work:

- Add "Open in Terminal" actions for the selected project or storage repo.
- Consider a read-only command transcript or activity log for debugging CLI
  calls.
- Reconsider embedded terminal support only after native dashboard, ticket,
  search, validation, and sync workflows are stable.

Implementation implications:

- The frontend should build native dashboard, project, ticket, search, activity,
  validation, and sync views from CLI JSON output.
- The command runner should capture stdout, stderr, exit code, command metadata,
  and parsed JSON for error handling and optional diagnostics.
- No terminal emulator dependency is required for the first release.

## First Release Scope

The first release should therefore ship this setup and runtime shape:

1. Install and verify the app-private CLI.
2. Create new local storage or select existing local storage.
3. Run setup, doctor, and index rebuild through the CLI.
4. Show native read views from CLI JSON.
5. Support write actions through existing CLI commands.
6. Provide explicit sync controls and advanced settings for publication behavior.
7. Offer terminal CLI installation after setup, not as a first-run prompt.

