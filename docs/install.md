# Install

Maat is distributed as a single Go binary named `maat`.

The installed binary is only an interface. The durable source of truth is still the Git-controlled Markdown storage repo, and the local SQLite index can be rebuilt at any time.

## Requirements

- macOS or Linux.
- Git for syncing the storage repo.
- A `maat` binary, either prebuilt or built from this checkout.
- Go only when building locally from source.

The installer does not require network access. If it needs to build from source, it runs Go with `GOPROXY=off`, so any module dependencies must already be available locally.

If local module dependencies are missing, build or download them before running the installer, then rerun `scripts/install.sh`. The installer should not be the step that reaches out to the network.

## Install From A Checkout

From the repository root:

```sh
scripts/install.sh
```

The installer looks for an existing executable in this order:

```text
dist/maat-<os>-<arch>
dist/maat
./maat
```

If no executable is found and Go is available, it builds:

```sh
go build -o <temp>/maat ./cmd/maat
```

The default install target is `/usr/local/bin` when writable. Otherwise it installs to:

```text
~/.local/bin
```

Use a custom target with:

```sh
MAAT_INSTALL_DIR="$HOME/.local/bin" scripts/install.sh
```

Install a specific binary with:

```sh
MAAT_SOURCE_BIN="./dist/maat" scripts/install.sh
```

Control ANSI progress color with:

```sh
MAAT_COLOR=always scripts/install.sh
MAAT_COLOR=never scripts/install.sh
```

During install, the script prints step-by-step progress for selecting the target directory, finding or building the binary, preparing the target, installing the executable, and checking whether the target directory is on `PATH`.

## Update And Uninstall

`maat update` checks the current binary version, reads the latest GitHub release for `sunday-studio/maat`, downloads the matching archive for the current OS and CPU architecture, verifies the checksum when the release provides one, extracts the binary, and replaces the installed binary.

```sh
maat update
maat update --install-dir "$HOME/.local/bin"
```

Use a local source only for development or smoke testing:

```sh
maat update --source ./dist/maat --install-dir "$HOME/.local/bin"
maat update --source /tmp/maat-new --install-dir /usr/local/bin --binary-name maat
```

When `--install-dir` is omitted, `maat update` tries to replace the currently running installed binary when that path is writable. Otherwise it uses the same default target as the installer.

Remove the installed binary with:

```sh
maat uninstall
maat uninstall --install-dir "$HOME/.local/bin"
```

By default, uninstall removes only the binary and keeps Maat config. Remove the local config explicitly with:

```sh
maat uninstall --purge-config
```

Use `--binary-name <name>` for test installs or renamed binaries.

## Test Install, Update, And Uninstall

Use a temporary install directory so the test does not touch your real system path:

```sh
GOCACHE=/private/tmp/maat-go-cache go build -o /tmp/maat ./cmd/maat

INSTALL_DIR=$(mktemp -d)
MAAT_INSTALL_DIR="$INSTALL_DIR" MAAT_SOURCE_BIN=/tmp/maat scripts/install.sh

"$INSTALL_DIR/maat" version
"$INSTALL_DIR/maat" update --source /tmp/maat --install-dir "$INSTALL_DIR"
"$INSTALL_DIR/maat" uninstall --install-dir "$INSTALL_DIR"

test ! -e "$INSTALL_DIR/maat"
```

Test config purge without touching your normal config:

```sh
CONFIG_FILE=$(mktemp)
printf '{}\n' > "$CONFIG_FILE"

MAAT_CONFIG="$CONFIG_FILE" /tmp/maat uninstall --install-dir "$INSTALL_DIR" --purge-config
test ! -e "$CONFIG_FILE"
```

## Build From Source

Build the local binary into `dist/maat`:

```sh
make build
```

The build stamps version metadata when Git is available. Check it with:

```sh
dist/maat version
```

Build release archives for macOS and Linux:

```sh
make release
```

This writes tarballs and checksums under `dist/`:

```text
dist/maat-<version>-darwin-amd64.tar.gz
dist/maat-<version>-darwin-arm64.tar.gz
dist/maat-<version>-linux-amd64.tar.gz
dist/maat-<version>-linux-arm64.tar.gz
dist/checksums-<version>.txt
```

GitHub Actions builds these same artifacts on `v*` tag pushes and publishes them to the matching GitHub Release. Manual dispatch builds and uploads the artifacts for inspection without publishing a release.

## Storage Repo

Maat storage is a normal Git repository containing Markdown files.

It can live anywhere, for example:

```text
/Users/casprine/maat-state
~/work/maat-state
~/Desktop/vendor/sunday-studio/maat
```

For one-time human setup, run:

```sh
maat setup
```

The prompt records the storage Git repo path, default actor, and auto pull/commit/push choices. Press Enter to accept the shown defaults. The storage path must still be absolute and point at a Git repository.

Agents and scripts should use the non-interactive form:

```sh
maat setup --storage /absolute/path/to/maat-state
```

You can also pass storage explicitly per command:

```sh
maat status --storage /absolute/path/to/maat-state
maat projects --storage /absolute/path/to/maat-state
maat search "blocked" --storage /absolute/path/to/maat-state
```

Both setup forms persist the selected path for later commands.

## Local Paths

These paths are local machine state and should not be treated as authoritative project data.

### Config

Target config paths:

```text
macOS: ~/Library/Application Support/maat/config.toml
Linux: ~/.config/maat/config.toml
```

The config records settings such as the storage repo path, default UI port, preferred editor, and agent identity defaults.

### Storage

The storage path is user-selected and should usually be a Git checkout:

```text
/absolute/path/to/maat-state
```

This is the durable state. It should be committed and synced.

### Index

Target index paths:

```text
macOS: ~/Library/Caches/maat/index.sqlite
Linux: ~/.cache/maat/index.sqlite
```

The current bootstrap implementation may also write inside the storage repo:

```text
<storage>/.maat/index.json
<storage>/.maat/index.sqlite
```

The index is rebuildable. Deleting it must not delete project state.

## Run After Install

Check the binary:

```sh
maat --help
```

The installer ends with a `maat ready to use` banner and a short start-here command list:

```sh
maat version
maat --help
maat setup
maat index rebuild --storage /absolute/path/to/maat-state
maat status --storage /absolute/path/to/maat-state
maat tui --storage /absolute/path/to/maat-state
```

Query a storage repo:

```sh
maat version
maat status --storage /absolute/path/to/maat-state
maat projects --storage /absolute/path/to/maat-state
maat index rebuild --storage /absolute/path/to/maat-state
```

When the TUI lands, the expected command will be:

```sh
maat tui
```

When the local web UI lands, the expected command will be:

```sh
maat ui
```

## New Machine Flow

1. Clone the Maat storage repo.
2. Install `maat`.
3. Link or pass the storage path.
4. Rebuild the local index.
5. Query from the CLI, TUI, or UI.

Current commands:

```sh
git clone <storage-remote> /absolute/path/to/maat-state
scripts/install.sh
maat index rebuild --storage /absolute/path/to/maat-state
maat status --storage /absolute/path/to/maat-state
```
