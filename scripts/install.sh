#!/bin/sh
set -eu

binary_name="${MATT_BINARY_NAME:-matt}"
install_dir="${MATT_INSTALL_DIR:-}"
source_bin="${MATT_SOURCE_BIN:-}"
color_mode="${MATT_COLOR:-auto}"

script_dir=$(CDPATH= cd "$(dirname "$0")" && pwd -P)
repo_dir=$(CDPATH= cd "$script_dir/.." && pwd -P)

use_color=0
case "$color_mode" in
  always) use_color=1 ;;
  never) use_color=0 ;;
  auto)
    if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
      use_color=1
    fi
    ;;
  *)
    printf 'matt install: unsupported MATT_COLOR value: %s\n' "$color_mode" >&2
    printf 'Use MATT_COLOR=auto, MATT_COLOR=always, or MATT_COLOR=never.\n' >&2
    exit 1
    ;;
esac

if [ "$use_color" -eq 1 ]; then
  esc=$(printf '\033')
  reset="${esc}[0m"
  bold="${esc}[1m"
  dim="${esc}[2m"
  green="${esc}[32m"
  blue="${esc}[34m"
  cyan="${esc}[36m"
  yellow="${esc}[33m"
else
  reset=""
  bold=""
  dim=""
  green=""
  blue=""
  cyan=""
  yellow=""
fi

step_total=5
step_number=0

step() {
  step_number=$((step_number + 1))
  printf '%s[%d/%d]%s %s\n' "$blue" "$step_number" "$step_total" "$reset" "$1"
}

done_line() {
  printf '  %sdone%s %s\n' "$green" "$reset" "$1"
}

info_line() {
  printf '  %snote%s %s\n' "$yellow" "$reset" "$1"
}

fail_line() {
  printf '  failed %s\n' "$1" >&2
}

usage() {
  cat <<EOF
Install the matt binary from this checkout.

Environment:
  MATT_INSTALL_DIR   Install directory. Defaults to /usr/local/bin when writable,
                     otherwise \$HOME/.local/bin.
  MATT_SOURCE_BIN    Explicit binary to install.
  MATT_BINARY_NAME   Installed binary name. Defaults to matt.
  MATT_COLOR         Progress color mode: auto, always, or never.

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
  step "Choosing install directory"
  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    install_dir=/usr/local/bin
  else
    install_dir="$HOME/.local/bin"
  fi
  done_line "$install_dir"
else
  step "Using requested install directory"
  done_line "$install_dir"
fi

resolve_source() {
  if [ -n "$source_bin" ]; then
    if [ ! -f "$source_bin" ]; then
      fail_line "MATT_SOURCE_BIN does not exist: $source_bin"
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
step "Finding installable binary"
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
    fail_line "no prebuilt binary found and Go is not installed"
    printf 'Place a binary at dist/%s or set MATT_SOURCE_BIN.\n' "$binary_name" >&2
    exit 1
  fi

  tmp_dir=$(mktemp -d)
  source_path="$tmp_dir/$binary_name"
  info_line "no prebuilt binary found; building $binary_name from this checkout"
  (
    cd "$repo_dir"
    GOPROXY=off go build -o "$source_path" ./cmd/matt
  )
  done_line "built $binary_name"
else
  done_line "$source_path"
fi

step "Preparing install directory"
mkdir -p "$install_dir"

if [ ! -w "$install_dir" ]; then
  fail_line "install directory is not writable: $install_dir"
  printf 'Choose a writable directory, for example:\n' >&2
  printf '  MATT_INSTALL_DIR="$HOME/.local/bin" scripts/install.sh\n' >&2
  exit 1
fi
done_line "$install_dir is writable"

step "Installing $binary_name"
target_path="$install_dir/$binary_name"
cp "$source_path" "$target_path"
chmod 0755 "$target_path"
done_line "$target_path"

step "Checking shell path"
case ":$PATH:" in
  *":$install_dir:"*)
    done_line "$install_dir is on PATH"
    ;;
  *)
    info_line "$install_dir is not on PATH"
    info_line "add it to your shell profile before running $binary_name from anywhere"
    ;;
esac

printf '\n'
printf '%s+-------------------+%s\n' "$cyan" "$reset"
printf '%s| %smaat ready to use%s%s |%s\n' "$cyan" "$bold" "$reset" "$cyan" "$reset"
printf '%s+-------------------+%s\n' "$cyan" "$reset"
printf '\n'
printf '%sInstalled:%s %s\n' "$bold" "$reset" "$target_path"
printf '\n'
printf '%sStart with:%s\n' "$bold" "$reset"
printf '  %s version\n' "$binary_name"
printf '  %s --help\n' "$binary_name"
printf '  %s init /absolute/path/to/maat-state\n' "$binary_name"
printf '  %s index rebuild --storage /absolute/path/to/maat-state\n' "$binary_name"
printf '  %s status --storage /absolute/path/to/maat-state\n' "$binary_name"
printf '  %s tui --storage /absolute/path/to/maat-state\n' "$binary_name"
printf '\n'
printf '%sTip:%s run %s init once to save the storage path, then omit --storage.\n' "$dim" "$reset" "$binary_name"
