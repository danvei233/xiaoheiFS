#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

OUT_DIR="build/linux"

echo "[1/4] Build frontend (public site)..."
cd frontend
npm ci
npm run build
cd ..

echo "[2/4] Reset $OUT_DIR and copy public static..."
rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR/static" "$OUT_DIR/static-admin"
cp -a frontend/dist/. "$OUT_DIR/static/"

echo "[3/4] Build adminweb (admin SPA)..."
cd adminweb
# adminweb is a pnpm project; enable the pinned version when corepack is available.
corepack enable >/dev/null 2>&1 || true
pnpm install --frozen-lockfile
VITE_BUILD_OUT_DIR="$REPO_ROOT/$OUT_DIR/static-admin" pnpm build
cd ..

echo "[4/4] Build backend (linux)..."
cd backend
go env -w CGO_ENABLED=0
go build -o "../$OUT_DIR/server" ./cmd/server
cd ..

echo "Done."
