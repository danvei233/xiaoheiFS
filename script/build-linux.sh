#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "[1/3] Build frontend..."
cd frontend
npm ci
npm run build
cd ..

echo "[2/3] Copy dist to build/linux/static..."
rm -rf build/linux
mkdir -p build/linux/static
cp -a frontend/dist/. build/linux/static/

echo "[3/4] Build backend (linux)..."
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -o ../build/linux/server ./cmd/server
cd ..

echo "[4/4] Build plugins (linux/amd64)..."
if [[ -z "${PLUGIN_SIGN_PRIVATE_KEY:-}" && -z "${PLUGIN_SIGNING_KEY:-}" ]]; then
  echo "ERROR: no plugin signing key is configured (P2-65)." >&2
  echo "  Bundled plugins must be signed so the host can import them on upgrade." >&2
  echo "  Generate a keypair (writes <dir>/plugin.ed25519.private.key):" >&2
  echo "    cd backend && go run ./cmd/tools/pluginsign -dir /tmp/keygen/plugin" >&2
  echo "  Then export the private key:" >&2
  echo "    export PLUGIN_SIGN_PRIVATE_KEY=\"\$(cat /tmp/keygen/plugin.ed25519.private.key)\"" >&2
  echo "  (or set the legacy PLUGIN_SIGNING_KEY)." >&2
  echo "  For a local UNSIGNED dev bundle only (not a release), pass --allow-unsigned:" >&2
  echo "    ./script/build-plugins.sh --allow-unsigned linux amd64 build/linux/plugins" >&2
  exit 1
fi
./script/build-plugins.sh linux amd64 build/linux/plugins

echo "Done."
