#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <goos> <goarch> <output-directory>" >&2
  exit 2
fi

TARGET_OS="$1"
TARGET_ARCH="$2"
OUTPUT_DIR="$3"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$REPO_ROOT/backend"
MANIFEST_ROOT="$BACKEND_DIR/plugins"
SOURCE_MAP="$REPO_ROOT/script/plugin-sources.txt"
PLATFORM="${TARGET_OS}_${TARGET_ARCH}"

case "$TARGET_OS" in
  windows)
    BINARY_NAME="plugin.exe"
    ;;
  linux | freebsd | darwin)
    BINARY_NAME="plugin"
    ;;
  *)
    echo "unsupported target OS: $TARGET_OS" >&2
    exit 2
    ;;
esac

mkdir -p "$OUTPUT_DIR"
OUTPUT_DIR="$(cd "$OUTPUT_DIR" && pwd)"
if [[ "$OUTPUT_DIR" == "$MANIFEST_ROOT" ]]; then
  echo "output directory must not overwrite backend/plugins" >&2
  exit 2
fi

built=0
while IFS='|' read -r plugin_id package_path; do
  [[ -z "$plugin_id" || "$plugin_id" == \#* ]] && continue

  manifest_dir="$MANIFEST_ROOT/$plugin_id"
  manifest="$manifest_dir/manifest.json"
  expected_rel="bin/$PLATFORM/$BINARY_NAME"

  if [[ ! -f "$manifest" ]]; then
    echo "missing manifest for $plugin_id" >&2
    exit 1
  fi
  if [[ ! -d "$BACKEND_DIR/${package_path#./}" ]]; then
    echo "missing plugin source for $plugin_id: $package_path" >&2
    exit 1
  fi

  if ! grep -Eq "\"${PLATFORM}\"[[:space:]]*:[[:space:]]*\"${expected_rel}\"" "$manifest"; then
    echo "Skipping $plugin_id: manifest does not support $PLATFORM"
    continue
  fi

  plugin_output="$OUTPUT_DIR/$plugin_id"
  mkdir -p "$plugin_output"
  cp -a "$manifest_dir/." "$plugin_output/"
  rm -rf "$plugin_output/bin"
  mkdir -p "$plugin_output/bin/$PLATFORM"

  echo "Building $plugin_id for $PLATFORM..."
  (
    cd "$BACKEND_DIR"
    CGO_ENABLED=0 GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" \
      go build -buildvcs=false -trimpath -o "$plugin_output/$expected_rel" "$package_path"
  )
  built=$((built + 1))
done < "$SOURCE_MAP"

while IFS= read -r manifest; do
  plugin_dir="$(dirname "$manifest")"
  if [[ ! -f "$plugin_dir/bin/$PLATFORM/$BINARY_NAME" ]]; then
    echo "bundle contains a manifest without a $PLATFORM binary: $manifest" >&2
    exit 1
  fi
done < <(find "$OUTPUT_DIR" -type f -name manifest.json -print)

echo "Built $built plugin(s) for $PLATFORM into $OUTPUT_DIR"
