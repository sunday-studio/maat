#!/bin/sh
set -eu

binary_name="${MAAT_BINARY_NAME:-maat}"
dist_dir="${DIST_DIR:-dist}"
version="${VERSION:-}"
commit="${COMMIT:-}"
date_value="${DATE:-}"

if [ -z "$version" ]; then
  version=$(git describe --tags --always --dirty 2>/dev/null || printf 'dev')
fi
if [ -z "$commit" ]; then
  commit=$(git rev-parse --short HEAD 2>/dev/null || printf 'unknown')
fi
if [ -z "$date_value" ]; then
  date_value=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
fi

module="github.com/sunday-studio/maat/internal/version"
ldflags="-s -w -X ${module}.Version=${version} -X ${module}.Commit=${commit} -X ${module}.Date=${date_value}"

targets="${TARGETS:-darwin/amd64 darwin/arm64 linux/amd64 linux/arm64}"

mkdir -p "$dist_dir"

for target in $targets; do
  goos=${target%/*}
  goarch=${target#*/}
  artifact="${binary_name}-${version}-${goos}-${goarch}"
  archive="${artifact}.tar.gz"
  printf 'building %s/%s...\n' "$goos" "$goarch"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath -ldflags "$ldflags" -o "$dist_dir/$artifact" ./cmd/matt
  tar -C "$dist_dir" -czf "$dist_dir/$archive" "$artifact"
done

(
  cd "$dist_dir"
  shasum -a 256 "${binary_name}-${version}-"*.tar.gz > "checksums-${version}.txt"
)

printf 'release artifacts written to %s\n' "$dist_dir"
