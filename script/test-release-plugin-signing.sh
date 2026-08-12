#!/usr/bin/env bash
# P2-65 release/upgrade E2E contract test.
#
# Proves the bundled-plugin release path is real, not a Docker-state hash:
#   1. builds a genuine signed plugin bundle via script/build-plugins.sh
#      (go build of every plugin for linux_amd64);
#   2. every bundled plugin ships a verifiable checksums.json + signature.sig;
#   3. the signature round-trips with the signing public key (pluginsign -verify,
#      mirroring the host-side VerifySignature);
#   4. tampering a binary flips verification to FAIL;
#   5. a bundle missing its signature is rejected (unsigned plugins never slip
#      into a release silently).
#
# The bootstrap import behaviour (signed new plugin auto-imported on image
# upgrade, unsigned logged-and-skipped) is pinned by the Go test
# backend/internal/adapter/plugins/core/bootstrap_upgrade_test.go
# (TestBootstrapFromDiskImageUpgrade), which this script's CI job also runs.
#
# Usage: bash script/test-release-plugin-signing.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BACKEND_DIR="$REPO_ROOT/backend"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0

ok() { pass=$((pass + 1)); echo "  ok   $1"; }
bad() { fail=$((fail + 1)); echo "  FAIL $1"; }

# --- Generate a fresh Ed25519 keypair via the signing tool (empty priv -> generate
# a keypair to a 0600 sibling file and refuse to sign, P2-65 fail-closed). ---
GEN="$WORK/gen"
mkdir -p "$GEN/plugin"
printf '{"plugin_id":"keygen"}\n' > "$GEN/plugin/manifest.json"
KEY_OUTPUT="$(cd "$BACKEND_DIR" && go run ./cmd/tools/pluginsign -dir "$GEN/plugin" 2>/dev/null || true)"
PUB="$(printf '%s\n' "$KEY_OUTPUT" | sed -n 's/^PublicKey(base64): //p')"
KEY_FILE="$GEN/plugin.ed25519.private.key"
if [[ ! -f "$KEY_FILE" ]]; then
  echo "signing tool did not write the generated private key to $KEY_FILE" >&2
  exit 1
fi
PRIV="$(cat "$KEY_FILE")"
if [[ -z "$PRIV" || -z "$PUB" ]]; then
  echo "failed to generate a signing keypair" >&2
  exit 1
fi
ok "generated a fresh Ed25519 signing keypair (private key at 0600 file)"

# --- P2-65 fail-closed: without any signing key the build must refuse. ---
NO_KEY_OUT="$WORK/nokey"
if PLUGIN_SIGN_PRIVATE_KEY= PLUGIN_SIGNING_KEY= bash "$REPO_ROOT/script/build-plugins.sh" linux amd64 "$NO_KEY_OUT" >/dev/null 2>&1; then
  bad "build-plugins.sh succeeded without a signing key (fail-closed broken)"
else
  ok "build-plugins.sh refuses to build without a signing key (fail-closed)"
fi

# --- P2-65 escape hatch: --allow-unsigned builds an unsigned bundle (CI PR
# builds only) without a key; the host still refuses to import it, but the image
# build itself succeeds. ---
ALLOW_OUT="$WORK/allow"
if PLUGIN_SIGN_PRIVATE_KEY= PLUGIN_SIGNING_KEY= bash "$REPO_ROOT/script/build-plugins.sh" --allow-unsigned linux amd64 "$ALLOW_OUT" >/dev/null 2>&1; then
  if [[ "$(find "$ALLOW_OUT" -type f -name manifest.json | wc -l)" -gt 0 ]]; then
    ok "--allow-unsigned builds an unsigned bundle without a key (CI PR escape hatch)"
  else
    bad "--allow-unsigned produced no plugin manifests"
  fi
else
  bad "--allow-unsigned build failed (CI PR escape hatch broken)"
fi

# --- Build a real signed bundle. ---
OUT="$WORK/release"
PLUGIN_SIGNING_KEY="$PRIV" bash "$REPO_ROOT/script/build-plugins.sh" linux amd64 "$OUT" >/dev/null 2>&1 \
  || { echo "build-plugins.sh failed" >&2; exit 1; }
ok "build-plugins.sh produced a signed linux_amd64 bundle"

BUNDLED=0
while IFS= read -r plugin_dir; do
  [[ -z "$plugin_dir" ]] && continue
  if [[ ! -f "$plugin_dir/checksums.json" || ! -f "$plugin_dir/signature.sig" ]]; then
    bad "bundled plugin is unsigned: $plugin_dir"
    continue
  fi
  if ! (cd "$BACKEND_DIR" && go run ./cmd/tools/pluginsign -dir "$plugin_dir" -verify -ed25519-pub "$PUB" >/dev/null 2>&1); then
    bad "signature round-trip failed for $plugin_dir"
    continue
  fi
  ok "verified signature for $(basename "$plugin_dir")"
  BUNDLED=$((BUNDLED + 1))
done < <(find "$OUT" -type f -name manifest.json -exec dirname {} \;)

if [[ "$BUNDLED" -eq 0 ]]; then
  bad "no bundled plugins were verified"
else
  ok "$BUNDLED bundled plugins all verify with the signing public key"
fi

# --- Tampering a binary must fail verification (installed bytes are bound). ---
FIRST_PLUGIN="$(find "$OUT" -type f -name manifest.json -print -quit | xargs dirname)"
if [[ -n "$FIRST_PLUGIN" ]]; then
  TAMPER="$WORK/tampered"
  cp -a "$FIRST_PLUGIN" "$TAMPER"
  # Find a real binary and corrupt it.
  BIN="$(find "$TAMPER/bin" -type f | head -1)"
  printf 'x' >> "$BIN"
  if (cd "$BACKEND_DIR" && go run ./cmd/tools/pluginsign -dir "$TAMPER" -verify -ed25519-pub "$PUB" >/dev/null 2>&1); then
    bad "tampered binary still verifies (installed bytes not bound)"
  else
    ok "tampered binary fails verification (checksums bind real bytes)"
  fi

  # Stripping the signature must be rejected (unsigned bundle never verified).
  UNSIGNED="$WORK/unsigned"
  cp -a "$FIRST_PLUGIN" "$UNSIGNED"
  rm -f "$UNSIGNED/signature.sig"
  if (cd "$BACKEND_DIR" && go run ./cmd/tools/pluginsign -dir "$UNSIGNED" -verify -ed25519-pub "$PUB" >/dev/null 2>&1); then
    bad "unsigned bundle passed verification"
  else
    ok "unsigned bundle is rejected by verification"
  fi
fi

echo
echo "$pass passed, $fail failed"
[ "$fail" -eq 0 ]
