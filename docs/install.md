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

Query a storage repo:

```sh
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
