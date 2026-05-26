# Install

Maat is distributed as a single Go binary named `matt`.

The installed binary is only an interface. The durable source of truth is still the Git-controlled Markdown storage repo, and the local SQLite index can be rebuilt at any time.

## Requirements

- macOS or Linux.
- Git for syncing the storage repo.
- A `matt` binary, either prebuilt or built from this checkout.
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
dist/matt-<os>-<arch>
dist/matt
./matt
```

If no executable is found and Go is available, it builds:

```sh
go build -o <temp>/matt ./cmd/matt
```

The default install target is `/usr/local/bin` when writable. Otherwise it installs to:

```text
~/.local/bin
```

Use a custom target with:

```sh
MATT_INSTALL_DIR="$HOME/.local/bin" scripts/install.sh
```

Install a specific binary with:

```sh
MATT_SOURCE_BIN="./dist/matt" scripts/install.sh
```

Control ANSI progress color with:

```sh
MATT_COLOR=always scripts/install.sh
MATT_COLOR=never scripts/install.sh
```

During install, the script prints step-by-step progress for selecting the target directory, finding or building the binary, preparing the target, installing the executable, and checking whether the target directory is on `PATH`.

## Update And Uninstall

`matt update` checks the current binary version, reads the latest GitHub release for `sunday-studio/maat`, downloads the matching archive for the current OS and CPU architecture, verifies the checksum when the release provides one, extracts the binary, and replaces the installed binary.

```sh
matt update
matt update --install-dir "$HOME/.local/bin"
```

Use a local source only for development or smoke testing:

```sh
matt update --source ./dist/matt --install-dir "$HOME/.local/bin"
matt update --source /tmp/matt-new --install-dir /usr/local/bin --binary-name matt
```

When `--install-dir` is omitted, `matt update` tries to replace the currently running installed binary when that path is writable. Otherwise it uses the same default target as the installer.

Remove the installed binary with:

```sh
matt uninstall
matt uninstall --install-dir "$HOME/.local/bin"
```

By default, uninstall removes only the binary and keeps Maat config. Remove the local config explicitly with:

```sh
matt uninstall --purge-config
```

Use `--binary-name <name>` for test installs or renamed binaries.

## Test Install, Update, And Uninstall

Use a temporary install directory so the test does not touch your real system path:

```sh
GOCACHE=/private/tmp/maat-go-cache go build -o /tmp/matt ./cmd/matt

INSTALL_DIR=$(mktemp -d)
MATT_INSTALL_DIR="$INSTALL_DIR" MATT_SOURCE_BIN=/tmp/matt scripts/install.sh

"$INSTALL_DIR/matt" version
"$INSTALL_DIR/matt" update --source /tmp/matt --install-dir "$INSTALL_DIR"
"$INSTALL_DIR/matt" uninstall --install-dir "$INSTALL_DIR"

test ! -e "$INSTALL_DIR/matt"
```

Test config purge without touching your normal config:

```sh
CONFIG_FILE=$(mktemp)
printf '{}\n' > "$CONFIG_FILE"

MAAT_CONFIG="$CONFIG_FILE" /tmp/matt uninstall --install-dir "$INSTALL_DIR" --purge-config
test ! -e "$CONFIG_FILE"
```

## Build From Source

Build the local binary into `dist/matt`:

```sh
make build
```

The build stamps version metadata when Git is available. Check it with:

```sh
dist/matt version
```

Build release archives for macOS and Linux:

```sh
make release
```

This writes tarballs and checksums under `dist/`:

```text
dist/matt-<version>-darwin-amd64.tar.gz
dist/matt-<version>-darwin-arm64.tar.gz
dist/matt-<version>-linux-amd64.tar.gz
dist/matt-<version>-linux-arm64.tar.gz
dist/checksums-<version>.txt
```

No publish step is included. GitHub Actions can build and upload these artifacts on tag pushes or manual dispatch.

## Storage Repo

Maat storage is a normal Git repository containing Markdown files.

It can live anywhere, for example:

```text
/Users/casprine/maat-state
~/work/maat-state
~/Desktop/vendor/sunday-studio/maat
```

During the current implementation phase, pass it explicitly:

```sh
matt status --storage /absolute/path/to/maat-state
matt projects --storage /absolute/path/to/maat-state
matt search "blocked" --storage /absolute/path/to/maat-state
```

The target setup flow will persist this path with `matt init` or `matt storage link`.

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
matt --help
```

The installer ends with a `maat ready to use` banner and a short start-here command list:

```sh
matt version
matt --help
matt init /absolute/path/to/maat-state
matt index rebuild --storage /absolute/path/to/maat-state
matt status --storage /absolute/path/to/maat-state
matt tui --storage /absolute/path/to/maat-state
```

Query a storage repo:

```sh
matt version
matt status --storage /absolute/path/to/maat-state
matt projects --storage /absolute/path/to/maat-state
matt index rebuild --storage /absolute/path/to/maat-state
```

When the TUI lands, the expected command will be:

```sh
matt tui
```

When the local web UI lands, the expected command will be:

```sh
matt ui
```

## New Machine Flow

1. Clone the Maat storage repo.
2. Install `matt`.
3. Link or pass the storage path.
4. Rebuild the local index.
5. Query from the CLI, TUI, or UI.

Current commands:

```sh
git clone <storage-remote> /absolute/path/to/maat-state
scripts/install.sh
matt index rebuild --storage /absolute/path/to/maat-state
matt status --storage /absolute/path/to/maat-state
```
