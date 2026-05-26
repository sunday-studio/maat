#!/bin/sh
set -eu

binary_name="${MAAT_BINARY_NAME:-maat}"
install_dir="${MAAT_INSTALL_DIR:-}"
source_bin="${MAAT_SOURCE_BIN:-}"
color_mode="${MAAT_COLOR:-auto}"
maat_repo="${MAAT_REPO:-sunday-studio/maat}"
maat_version="${MAAT_VERSION:-}"
update_path="${MAAT_UPDATE_PATH:-auto}"

script_dir=$(CDPATH= cd "$(dirname "$0")" && pwd -P)
repo_dir=$(CDPATH= cd "$script_dir/.." && pwd -P)
checkout_mode=0
if [ -f "$repo_dir/go.mod" ] && [ -d "$repo_dir/cmd/maat" ]; then
  checkout_mode=1
fi

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
    printf 'maat install: unsupported MAAT_COLOR value: %s\n' "$color_mode" >&2
    printf 'Use MAAT_COLOR=auto, MAAT_COLOR=always, or MAAT_COLOR=never.\n' >&2
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
Install the maat binary.

Environment:
  MAAT_INSTALL_DIR   Install directory. Defaults to /usr/local/bin when writable,
                     otherwise \$HOME/.local/bin.
  MAAT_SOURCE_BIN    Explicit binary to install.
  MAAT_VERSION       Release version to install. Defaults to latest GitHub release.
  MAAT_REPO          GitHub repo to download from. Defaults to sunday-studio/maat.
  MAAT_BINARY_NAME   Installed binary name. Defaults to maat.
  MAAT_COLOR         Progress color mode: auto, always, or never.
  MAAT_UPDATE_PATH   Add install directory to shell profile when missing: auto, never.

Examples:
  curl -fsSL https://raw.githubusercontent.com/sunday-studio/maat/main/scripts/install.sh | sh
  curl -fsSL https://raw.githubusercontent.com/sunday-studio/maat/main/scripts/install.sh | MAAT_VERSION=v0.1.0 sh
  scripts/install.sh
  MAAT_INSTALL_DIR="\$HOME/.local/bin" scripts/install.sh
  MAAT_SOURCE_BIN="./dist/maat" scripts/install.sh
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

path_contains() {
  case ":$PATH:" in
    *":$1:"*) return 0 ;;
    *) return 1 ;;
  esac
}

choose_install_dir() {
  if command -v "$binary_name" >/dev/null 2>&1; then
    existing=$(command -v "$binary_name")
    existing_dir=$(dirname "$existing")
    if [ -w "$existing_dir" ]; then
      printf '%s\n' "$existing_dir"
      return
    fi
  fi

  for candidate in /usr/local/bin /opt/homebrew/bin "$HOME/.local/bin" "$HOME/bin"; do
    if [ -d "$candidate" ] && [ -w "$candidate" ] && path_contains "$candidate"; then
      printf '%s\n' "$candidate"
      return
    fi
  done

  old_ifs=$IFS
  IFS=:
  for candidate in $PATH; do
    if [ -n "$candidate" ] && [ -d "$candidate" ] && [ -w "$candidate" ]; then
      IFS=$old_ifs
      printf '%s\n' "$candidate"
      return
    fi
  done
  IFS=$old_ifs

  printf '%s\n' "$HOME/.local/bin"
}

if [ -z "$install_dir" ]; then
  step "Choosing install directory"
  install_dir=$(choose_install_dir)
  done_line "$install_dir"
else
  step "Using requested install directory"
  done_line "$install_dir"
fi

detect_target() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    darwin|linux) ;;
    *)
      fail_line "unsupported operating system: $os"
      exit 1
      ;;
  esac

  machine=$(uname -m)
  case "$machine" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *)
      fail_line "unsupported CPU architecture: $machine"
      exit 1
      ;;
  esac
}

latest_version() {
  if [ -n "$maat_version" ]; then
    printf '%s\n' "$maat_version"
    return
  fi
  if ! command -v curl >/dev/null 2>&1; then
    fail_line "curl is required to download the latest release"
    exit 1
  fi
  tag=$(curl -fsSL "https://api.github.com/repos/$maat_repo/releases/latest" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)
  if [ -z "$tag" ]; then
    fail_line "could not determine latest release for $maat_repo"
    exit 1
  fi
  printf '%s\n' "$tag"
}

verify_checksum() {
  checksum_file="$1"
  asset_name="$2"
  if ! grep "  $asset_name\$" "$checksum_file" > "$tmp_dir/checksum-selected.txt"; then
    info_line "checksum entry not found for $asset_name; skipping checksum verification"
    return
  fi
  if ! command -v shasum >/dev/null 2>&1 && ! command -v sha256sum >/dev/null 2>&1; then
    info_line "no checksum tool found; skipping checksum verification"
    return
  fi
  (
    cd "$tmp_dir"
    if command -v shasum >/dev/null 2>&1; then
      shasum -a 256 -c checksum-selected.txt >/dev/null
    else
      sha256sum -c checksum-selected.txt >/dev/null
    fi
  )
  done_line "verified checksum"
}

download_release() {
  detect_target
  version=$(latest_version)
  asset_name="$binary_name-$version-$os-$arch.tar.gz"
  base_url="https://github.com/$maat_repo/releases/download/$version"
  tmp_dir=$(mktemp -d)

  info_line "downloading $asset_name"
  if ! command -v curl >/dev/null 2>&1; then
    fail_line "curl is required to download release binaries"
    exit 1
  fi
  curl -fsSL "$base_url/$asset_name" -o "$tmp_dir/$asset_name"

  if curl -fsSL "$base_url/checksums-$version.txt" -o "$tmp_dir/checksums.txt"; then
    verify_checksum "$tmp_dir/checksums.txt" "$asset_name"
  else
    info_line "checksum file unavailable; continuing without checksum verification"
  fi

  tar -xzf "$tmp_dir/$asset_name" -C "$tmp_dir"
  extracted="$tmp_dir/${binary_name}-${version}-${os}-${arch}"
  if [ ! -f "$extracted" ]; then
    fail_line "release archive did not contain expected binary: ${binary_name}-${version}-${os}-${arch}"
    exit 1
  fi
  chmod 0755 "$extracted"
  source_path="$extracted"
}

resolve_source() {
  if [ -n "$source_bin" ]; then
    if [ ! -f "$source_bin" ]; then
      fail_line "MAAT_SOURCE_BIN does not exist: $source_bin"
      exit 1
    fi
    source_path="$source_bin"
    return
  fi

  if [ "$checkout_mode" -eq 1 ]; then
    detect_target
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
  fi
}

source_path=""
tmp_dir=""
step "Finding installable binary"
resolve_source

cleanup() {
  if [ -n "$tmp_dir" ] && [ -d "$tmp_dir" ]; then
    rm -rf "$tmp_dir"
  fi
}
trap cleanup EXIT INT HUP TERM

if [ -z "$source_path" ]; then
  if [ "$checkout_mode" -eq 1 ] && command -v go >/dev/null 2>&1; then
    tmp_dir=$(mktemp -d)
    source_path="$tmp_dir/$binary_name"
    info_line "no prebuilt binary found; building $binary_name from this checkout"
    (
      cd "$repo_dir"
      GOPROXY=off go build -o "$source_path" ./cmd/maat
    )
    done_line "built $binary_name"
  else
    download_release
    done_line "$source_path"
  fi
else
  done_line "$source_path"
fi

step "Preparing install directory"
mkdir -p "$install_dir"

if [ ! -w "$install_dir" ]; then
  fail_line "install directory is not writable: $install_dir"
  printf 'Choose a writable directory, for example:\n' >&2
  printf '  MAAT_INSTALL_DIR="$HOME/.local/bin" scripts/install.sh\n' >&2
  exit 1
fi
done_line "$install_dir is writable"

step "Installing $binary_name"
target_path="$install_dir/$binary_name"
cp "$source_path" "$target_path"
chmod 0755 "$target_path"
done_line "$target_path"

profile_path() {
  shell_name=$(basename "${SHELL:-}")
  case "$shell_name" in
    zsh)
      printf '%s\n' "$HOME/.zshrc"
      ;;
    bash)
      if [ "$(uname -s)" = "Darwin" ]; then
        printf '%s\n' "$HOME/.bash_profile"
      else
        printf '%s\n' "$HOME/.bashrc"
      fi
      ;;
    *)
      printf '%s\n' "$HOME/.profile"
      ;;
  esac
}

add_path_to_profile() {
  profile=$(profile_path)
  mkdir -p "$(dirname "$profile")"
  if [ -f "$profile" ] && grep -F "$install_dir" "$profile" >/dev/null 2>&1; then
    done_line "$install_dir is already configured in $profile"
    return
  fi
  {
    printf '\n'
    printf '# Added by Maat installer\n'
    printf 'export PATH="%s:$PATH"\n' "$install_dir"
  } >> "$profile"
  done_line "added $install_dir to $profile"
  info_line "restart your shell or run: export PATH=\"$install_dir:\$PATH\""
}

step "Configuring maat command"
case ":$PATH:" in
  *":$install_dir:"*)
    done_line "$install_dir is on PATH"
    ;;
  *)
    if [ "$update_path" = "never" ]; then
      info_line "$install_dir is not on PATH"
      info_line "run: export PATH=\"$install_dir:\$PATH\""
    else
      add_path_to_profile
    fi
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
printf '  %s setup --storage /absolute/path/to/maat-state\n' "$binary_name"
printf '  %s index rebuild --storage /absolute/path/to/maat-state\n' "$binary_name"
printf '  %s status --storage /absolute/path/to/maat-state\n' "$binary_name"
printf '  %s tui --storage /absolute/path/to/maat-state\n' "$binary_name"
printf '\n'
printf '%sTip:%s run %s setup --storage once to save the storage path, then omit --storage.\n' "$dim" "$reset" "$binary_name"
