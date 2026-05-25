#!/bin/sh
set -eu

binary_name="${MATT_BINARY_NAME:-matt}"
install_dir="${MATT_INSTALL_DIR:-}"
source_bin="${MATT_SOURCE_BIN:-}"

script_dir=$(CDPATH= cd "$(dirname "$0")" && pwd -P)
repo_dir=$(CDPATH= cd "$script_dir/.." && pwd -P)

usage() {
  cat <<EOF
Install the matt binary from this checkout.

Environment:
  MATT_INSTALL_DIR   Install directory. Defaults to /usr/local/bin when writable,
                     otherwise \$HOME/.local/bin.
  MATT_SOURCE_BIN    Explicit binary to install.
  MATT_BINARY_NAME   Installed binary name. Defaults to matt.

Examples:
  scripts/install.sh
  MATT_INSTALL_DIR="\$HOME/.local/bin" scripts/install.sh
  MATT_SOURCE_BIN="./dist/matt" scripts/install.sh
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

if [ -z "$install_dir" ]; then
  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    install_dir=/usr/local/bin
  else
    install_dir="$HOME/.local/bin"
  fi
fi

resolve_source() {
  if [ -n "$source_bin" ]; then
    if [ ! -f "$source_bin" ]; then
      printf 'matt install: MATT_SOURCE_BIN does not exist: %s\n' "$source_bin" >&2
      exit 1
    fi
    source_path="$source_bin"
    return
  fi

  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)

  for candidate in \
    "$repo_dir/dist/$binary_name-$os-$arch" \
    "$repo_dir/dist/$binary_name" \
    "$repo_dir/$binary_name"
  do
    if [ -f "$candidate" ] && [ -x "$candidate" ]; then
      source_path="$candidate"
      return
    fi
  done
}

source_path=""
resolve_source
tmp_dir=""

cleanup() {
  if [ -n "$tmp_dir" ] && [ -d "$tmp_dir" ]; then
    rm -rf "$tmp_dir"
  fi
}
trap cleanup EXIT INT HUP TERM

if [ -z "$source_path" ]; then
  if ! command -v go >/dev/null 2>&1; then
    printf 'matt install: no prebuilt binary found and Go is not installed.\n' >&2
    printf 'Place a binary at dist/%s or set MATT_SOURCE_BIN.\n' "$binary_name" >&2
    exit 1
  fi

  tmp_dir=$(mktemp -d)
  source_path="$tmp_dir/$binary_name"
  printf 'matt install: building %s from local checkout...\n' "$binary_name" >&2
  (
    cd "$repo_dir"
    GOPROXY=off go build -o "$source_path" ./cmd/matt
  )
fi

mkdir -p "$install_dir"

if [ ! -w "$install_dir" ]; then
  printf 'matt install: install directory is not writable: %s\n' "$install_dir" >&2
  printf 'Choose a writable directory, for example:\n' >&2
  printf '  MATT_INSTALL_DIR="$HOME/.local/bin" scripts/install.sh\n' >&2
  exit 1
fi

target_path="$install_dir/$binary_name"
cp "$source_path" "$target_path"
chmod 0755 "$target_path"

printf 'Installed %s to %s\n' "$binary_name" "$target_path"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *)
    printf 'Note: %s is not on PATH.\n' "$install_dir"
    printf 'Add it to your shell profile before running %s from anywhere.\n' "$binary_name"
    ;;
esac
