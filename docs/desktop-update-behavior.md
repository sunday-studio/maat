# Desktop Update Behavior

This document defines how the macOS desktop app updates itself and the `maat`
CLI it uses. It expands the update model in
[macOS App Architecture](./macos-app-architecture.md) so implementation can keep
desktop app updates, app-private CLI refreshes, and terminal CLI updates clear.

The desktop app remains a thin shell over the CLI. The app must not reimplement
Maat storage, indexing, validation, sync, or project writes, but it is
responsible for selecting a trusted CLI binary and showing the user which binary
is active.

## Goals

- Let a desktop app update replace the bundled `maat` binary.
- Refresh the app-private installed CLI from the bundled binary after an app
  update.
- Record and display the active CLI path and version.
- Keep terminal-facing CLI installs separate from the app-private CLI.
- Handle failed refreshes without leaving the app without a usable CLI.

## Non-Goals

- The desktop app does not silently replace the user's terminal `maat` binary.
- The desktop app does not download arbitrary binaries outside the signed app
  update path in the first release.
- The app does not store project state or update storage files directly.

## Binary Locations

The app has two possible CLI sources:

| Source | Example path | Purpose |
| --- | --- | --- |
| Bundled CLI | `<Maat.app>/Contents/Resources/bin/maat` | Trusted source shipped with the signed app bundle. |
| App-private CLI | `~/Library/Application Support/maat/bin/maat` | Writable binary the app runs for normal commands. |

The bundled binary is read-only application content. The app-private binary is
the active runtime dependency for the desktop app whenever it is installed and
verified. The app may run the bundled binary only as a bootstrap or fallback
source, such as when the app-private binary is missing, stale, or failed
verification.

The optional terminal install remains separate, for example:

```text
~/.local/bin/maat
```

Installing or updating the app-private CLI must not overwrite that terminal
binary unless the user chooses a separate "Install command line tool" action.

## Startup Selection

On each launch, the CLI manager should resolve the active binary in this order:

1. Locate the bundled CLI inside the current app bundle.
2. Run the bundled CLI with `maat version --json`.
3. Locate the app-private CLI from app settings or the default app support path.
4. Run the app-private CLI with `maat version --json` when the file exists.
5. Refresh the app-private CLI from the bundled CLI when it is missing,
   unverifiable, older than the bundled CLI, or explicitly marked unhealthy.
6. Record the selected active path and version.
7. Use the recorded active path for all command runner operations.

The app should prefer the app-private CLI after a successful refresh because it
lives in a writable app support directory. This avoids executing mutable update
operations against the signed app bundle.

If the app-private CLI is unusable and refresh fails, the app may temporarily run
the bundled CLI for read-only commands and setup recovery. Write actions should
remain disabled until the app has either verified the bundled CLI as compatible
or repaired the app-private install.

## Version Comparison

The app should compare versions using `maat version --json`, which returns:

```json
{
  "version": "v1.2.3",
  "commit": "abc123",
  "date": "2026-05-27T00:00:00Z"
}
```

Version comparison rules:

- Normalize a leading `v` before comparison.
- Treat `MAJOR.MINOR.PATCH` release versions as semantic versions.
- Treat versions with extra prerelease metadata as lower than the matching final
  release.
- Treat `dev`, `unknown`, empty versions, or unparsable versions as not safe to
  declare newer than a release build.
- Use `commit` and `date` only for display and diagnostics, not as the primary
  ordering key.

Refresh decisions:

| Installed app-private CLI | Bundled CLI | Action |
| --- | --- | --- |
| Missing | Any valid bundled version | Install bundled CLI into app support. |
| Older release | Newer release | Replace app-private CLI from bundled CLI. |
| Same release | Same release | Keep app-private CLI after verification. |
| Newer release | Older release | Keep app-private CLI unless compatibility checks fail. |
| `dev` or `unknown` | Release | Replace app-private CLI unless user enabled a developer override. |
| Release | `dev` or `unknown` | Keep app-private CLI unless it is missing or broken. |

The current CLI `maat update` command only checks remote release equality for
the terminal update path. The desktop app should implement its own bundled versus
installed comparison before deciding whether to call the CLI installer path.

## App-Private Refresh

Refreshing the app-private CLI should use the existing CLI installer behavior:

```sh
<bundled-maat> update \
  --source <bundled-maat> \
  --install-dir "$HOME/Library/Application Support/maat/bin" \
  --binary-name maat \
  --json
```

Expected JSON fields include `action`, `binary_name`, `install_dir`,
`target_path`, `source_path`, `install_recorded`, and `config_path`. The app
should persist its own desktop setting for the active binary path and version
after verification, even though the CLI also records installed binary metadata in
the local Maat config for uninstall support.

Refresh flow:

1. Verify the bundled source can run `version --json`.
2. Compare the bundled version with the app-private installed version.
3. Skip install when the app-private CLI is current and healthy.
4. Run the bundled CLI `update --source ... --json` when refresh is needed.
5. Run the refreshed target CLI with `version --json`.
6. Confirm the target path matches the expected app support path.
7. Persist active CLI metadata in desktop settings.

The CLI installer writes the replacement through a temporary file and atomic
rename. The desktop app should still treat refresh as incomplete until the
post-install `version --json` check succeeds.

## Bundle Trust

The bundled CLI is trusted because it is inside the signed and notarized app
bundle. Implementation should preserve that trust boundary:

- Resolve the bundled binary from the app resources directory, not from `PATH`.
- Confirm the resolved path is inside the current app bundle.
- Rely on macOS app signing and notarization for distribution integrity.
- Sign the bundled CLI as part of the app bundle.
- Do not accept a user-selected update source for automatic app-private refresh.
- Do not run a quarantined or unsigned external binary as the desktop app's
  active CLI.

The app-private CLI is trusted only when it was copied from a trusted bundled
binary or explicitly installed by a user action with clear UI. After every app
update, the new bundled binary is the source of truth for refreshing the
app-private CLI.

## Active CLI Display

The desktop UI should expose active CLI details in About, Settings, or a similar
diagnostic surface:

| Field | Source |
| --- | --- |
| Active path | Desktop CLI manager selected path. |
| Version | `maat version --json` from the active path. |
| Commit | `maat version --json` from the active path. |
| Build date | `maat version --json` from the active path. |
| Source | `app-private`, `bundled fallback`, or `terminal install`. |
| Last verification | Timestamp of the last successful `version --json` check. |

The same surface should show refresh failures with the attempted source path,
target path, error text, and the currently active fallback, if any.

Command runner errors should include the active CLI path in developer-facing
details so support reports can distinguish app-private, bundled, and terminal
installs.

## Rollback And Error Handling

The app should preserve the last known-good active CLI metadata until a new
binary passes verification. A refresh attempt should not update desktop settings
until the target binary exists, is executable, and returns valid
`version --json`.

| Failure | App behavior |
| --- | --- |
| Bundled CLI missing | Show app repair guidance; do not use `PATH` implicitly. |
| Bundled CLI fails `version --json` | Block refresh and show the app bundle as unhealthy. |
| App support directory cannot be created | Keep last known-good CLI active and show permission guidance. |
| Install command exits non-zero | Keep last known-good CLI active and show stderr plus JSON output if present. |
| Target binary fails verification | Do not record it; retry from bundled CLI or use bundled fallback for recovery. |
| New app-private CLI is incompatible with current app | Keep prior app-private CLI when available; otherwise use bundled fallback with write actions disabled. |
| Last known-good CLI is missing too | Open setup or repair mode and ask the user to reinstall or update the app. |

The CLI installer uses atomic rename for the binary copy, so a failed copy should
not normally corrupt the previous target. The app should still verify the active
path after every failed refresh because filesystem errors, permissions, and
external modifications can leave local state unexpected.

Rollback is therefore metadata based: keep pointing the command runner at the
previous verified path until the replacement is verified. The app should not
attempt to reconstruct old binaries from its own cache in the first release.

## Desktop App Updates

The app updater is responsible for delivering a new signed app bundle. After an
app update:

1. The app starts from the new bundle.
2. The CLI manager verifies the new bundled CLI.
3. The CLI manager compares the bundled CLI with the app-private CLI.
4. The CLI manager refreshes the app-private CLI when the bundled version is
   newer or the installed binary is unhealthy.
5. The app records the refreshed active path and version.

This makes app updates able to replace both the bundled CLI and the app-private
installed CLI while preserving the user's separate terminal install.

## CLI Self-Update

The existing terminal CLI supports:

```sh
maat update --json
```

Without `--source`, that command checks GitHub Releases, downloads the matching
archive, verifies checksums when available, installs the binary, and records the
install path in local Maat config.

The desktop app should defer advanced remote CLI self-update for the first
release. The desktop UI should not automatically run `maat update --json`
against GitHub Releases for the app-private CLI. The supported first-release
desktop path is:

```text
signed app update -> new bundled CLI -> app-private CLI refresh
```

A later advanced settings screen may expose remote CLI self-update when the
product has decided how to handle release channels, user consent, compatibility
checks, rollback, and interaction with terminal installs.

## Implementation Notes

- Store desktop CLI metadata in desktop app settings, separate from project
  storage and separate from the CLI's local config file.
- Keep the active CLI path absolute.
- Do not discover the active desktop CLI from `PATH` during normal operation.
- Include the active path and version in diagnostic exports or support bundles.
- Keep app-private refresh idempotent so it can run on every launch after an app
  update.
- Treat storage repo paths and project Markdown as user data; update behavior
  must not mutate project state.
