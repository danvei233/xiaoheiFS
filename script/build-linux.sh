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

echo "[3/3] Build backend (linux)..."
cd backend
go build -o ../build/linux/server ./cmd/server
cd ..

echo "Done."