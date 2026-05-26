# Install

Maat is distributed as a single Go binary named `maat`.

You do not need to clone the Maat source repository to use it. Install a release binary, point it at your own Git-backed storage repo, and let agents update that storage repo through the CLI.

## Requirements

- macOS or Linux.
- Git for the Maat storage repo.
- A writable install directory on `PATH`, such as `~/.local/bin`.

Go is only required when building Maat from source.

## Install

Run the installer:

```sh
curl -fsSL https://raw.githubusercontent.com/sunday-studio/maat/main/scripts/install.sh | sh
```

The installer:

- detects macOS or Linux
- detects `arm64` or `amd64`
- downloads the matching GitHub Release archive
- verifies the checksum when a checksum tool is available
- chooses a writable install directory, preferring one already on `PATH`
- installs the binary as `maat`
- adds the install directory to your shell profile if it is not already on `PATH`

Check that your shell can find it:

```sh
maat version
```

If `maat` is not found, add the install directory to `PATH` in your shell profile:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

## Install Options

Install a specific release:

```sh
curl -fsSL https://raw.githubusercontent.com/sunday-studio/maat/main/scripts/install.sh | MAAT_VERSION=v0.1.0 sh
```

Use a custom install directory:

```sh
curl -fsSL https://raw.githubusercontent.com/sunday-studio/maat/main/scripts/install.sh | MAAT_INSTALL_DIR="$HOME/bin" sh
```

Skip shell profile changes:

```sh
curl -fsSL https://raw.githubusercontent.com/sunday-studio/maat/main/scripts/install.sh | MAAT_UPDATE_PATH=never sh
```

Manual release downloads are available on the [releases page](https://github.com/sunday-studio/maat/releases).

## Prepare Storage

Maat storage is a normal Git repository containing Markdown files. It is separate from the Maat product repo.

Create a new local storage repo:

```sh
mkdir -p "$HOME/maat-state"
git init "$HOME/maat-state"
```

Or clone a shared storage repo:

```sh
git clone <your-maat-storage-remote> "$HOME/maat-state"
```

Run first-time setup:

```sh
maat setup
```

The prompt records:

- storage repo path
- default actor
- auto-pull before reads
- auto-commit after writes
- auto-push after commits

Agents and scripts should use the non-interactive form:

```sh
maat setup --storage "$HOME/maat-state"
```

You can also pass storage explicitly:

```sh
maat status --storage "$HOME/maat-state"
maat projects --storage "$HOME/maat-state"
maat search "blocked" --storage "$HOME/maat-state"
```

## Start Using Maat

Register a project from inside that project repo:

```sh
cd /absolute/path/to/source-repo
maat initialize
```

`maat initialize` links the current repo and prints instructions for the agent working in that repo. Add those instructions to `AGENTS.md`, `CLAUDE.md`, Cursor rules, or the closest equivalent.

Inspect state:

```sh
maat status
maat projects
maat search "agent handoff"
maat tui
```

## Update And Uninstall

`maat update` checks GitHub Releases, downloads the matching archive for the current OS and CPU architecture, verifies checksums when available, and replaces the installed binary.

```sh
maat update
maat update --install-dir "$HOME/.local/bin"
```

Remove the installed binary:

```sh
maat uninstall
maat uninstall --install-dir "$HOME/.local/bin"
```

By default, uninstall removes only the binary and keeps Maat config. Remove local config explicitly with:

```sh
maat uninstall --purge-config
```

## Local Paths

These paths are local machine state and should not be treated as authoritative project data.

Config:

```text
macOS: ~/Library/Application Support/maat/config.json
Linux: ~/.config/maat/config.json
```

Storage:

```text
user-selected Git repo, for example ~/maat-state
```

Rebuildable indexes:

```text
<storage>/.maat/index.json
<storage>/.maat/index.sqlite
```

Deleting the index must not delete project state. Rebuild it with:

```sh
maat index rebuild
```

## Build From Source

Clone the source repo only if you want to contribute or build Maat yourself:

```sh
git clone https://github.com/sunday-studio/maat.git
cd maat
make build
dist/maat version
```

The checkout installer is for contributors:

```sh
scripts/install.sh
```

It copies an existing local binary when available or builds from the checkout with Go.

## Release Artifacts

GitHub Releases publish:

```text
maat-<version>-darwin-amd64.tar.gz
maat-<version>-darwin-arm64.tar.gz
maat-<version>-linux-amd64.tar.gz
maat-<version>-linux-arm64.tar.gz
checksums-<version>.txt
```

GitHub Actions builds these artifacts on `v*` tag pushes and marks tag-published releases as latest. Manual workflow dispatch uploads artifacts for inspection without publishing a release.
