#!/usr/bin/env bash
set -euo pipefail

# P2-65 escape hatch for CI PR builds only: --allow-unsigned skips signing (and
# the final signature-presence gate) so an image can be built on an untrusted
# pull request without a signing key. Release/push builds MUST NOT use it — a
# missing key is a hard failure so an unsigned/dev-signed bundle is never
# produced as if it were a release.
ALLOW_UNSIGNED=0
ARGS=()
for arg in "$@"; do
  if [[ "$arg" == "--allow-unsigned" ]]; then
    ALLOW_UNSIGNED=1
  else
    ARGS+=("$arg")
  fi
done
set -- "${ARGS[@]}"

if [[ $# -ne 3 ]]; then
  echo "usage: $0 [--allow-unsigned] <goos> <goarch> <output-directory>" >&2
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
if [[ "$OUTPUT_DIR" == "$MANIFEST_ROOT" || "$OUTPUT_DIR" == "$MANIFEST_ROOT"/* ]]; then
  echo "output directory must not overwrite backend/plugins" >&2
  exit 2
fi

# P2-65 fail-closed signing: the private key is fed to pluginsign via the
# protected PLUGIN_SIGN_PRIVATE_KEY env source (never an argv flag). A missing
# key is a hard failure — a dev/dev-signed bundle must never be produced as if it
# were a release.
#
# The key can also be supplied through a BuildKit secret mount
# (/run/secrets/plugin_sign_key, id=plugin_sign_key) so Docker image builds do
# not need the key as a build ARG (which would leak into the image history).
if [[ -z "${PLUGIN_SIGN_PRIVATE_KEY:-}" && -z "${PLUGIN_SIGNING_KEY:-}" && -f /run/secrets/plugin_sign_key ]]; then
  export PLUGIN_SIGN_PRIVATE_KEY="$(cat /run/secrets/plugin_sign_key)"
fi

sign_plugin() {
  local dir="$1"
  if [[ "$ALLOW_UNSIGNED" -eq 1 ]]; then
    # CI PR escape hatch only: produce the bundle unsigned. The host refuses to
    # import new unsigned plugins (bootstrap.go logs-and-skips), so this is never
    # a release artifact.
    echo "Skipping signing of $dir (--allow-unsigned; CI PR builds only)" >&2
    return 0
  fi
  if [[ -z "${PLUGIN_SIGN_PRIVATE_KEY:-}" && -z "${PLUGIN_SIGNING_KEY:-}" ]]; then
    echo "ERROR: PLUGIN_SIGN_PRIVATE_KEY (or legacy PLUGIN_SIGNING_KEY) is not set; refusing to build an unsigned/dev-signed plugin bundle (P2-65)" >&2
    exit 1
  fi
  if ! (cd "$BACKEND_DIR" && PLUGIN_SIGN_PRIVATE_KEY="${PLUGIN_SIGN_PRIVATE_KEY:-$PLUGIN_SIGNING_KEY}" go run ./cmd/tools/pluginsign -dir "$dir" >/dev/null 2>&1); then
    echo "signing $dir failed" >&2
    exit 1
  fi
  if [[ ! -f "$dir/checksums.json" || ! -f "$dir/signature.sig" ]]; then
    echo "signing $dir produced no checksums.json/signature.sig" >&2
    exit 1
  fi
}

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

  # test_only 插件（如 mockpay）禁止进入生产发布包。
  if grep -q '"test_only"[[:space:]]*:[[:space:]]*true' "$manifest"; then
    echo "Skipping $plugin_id: test-only plugin must not be released"
    continue
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
  # P2-65: every bundled plugin must ship verifiable checksums.json + a signed
  # signature.sig so the host can import it on image upgrade (bootstrap.go skips
  # new unsigned plugins). The key comes from PLUGIN_SIGN_PRIVATE_KEY (base64
  # Ed25519 private); a missing key fails the build (fail-closed).
  sign_plugin "$plugin_output"
  built=$((built + 1))
done < "$SOURCE_MAP"

if [[ "$ALLOW_UNSIGNED" -eq 1 ]]; then
  echo "Built $built plugin(s) for $PLATFORM into $OUTPUT_DIR (unsigned; --allow-unsigned CI PR escape hatch)"
  exit 0
fi

while IFS= read -r manifest; do
  plugin_dir="$(dirname "$manifest")"
  if [[ ! -f "$plugin_dir/bin/$PLATFORM/$BINARY_NAME" ]]; then
    echo "bundle contains a manifest without a $PLATFORM binary: $manifest" >&2
    exit 1
  fi
  if [[ ! -f "$plugin_dir/checksums.json" || ! -f "$plugin_dir/signature.sig" ]]; then
    echo "bundle contains an unsigned plugin: $plugin_dir" >&2
    exit 1
  fi
done < <(find "$OUTPUT_DIR" -type f -name manifest.json -print)

echo "Built and signed $built plugin(s) for $PLATFORM into $OUTPUT_DIR"
