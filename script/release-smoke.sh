#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

SHELL_SCRIPTS=(
  script/build-linux.sh
  script/build-plugins.sh
  script/release-smoke.sh
  pingbot/deploy/systemd/install.sh
  pingbot/deploy/systemd/uninstall.sh
  docker/entrypoint.sh
)

for script in "${SHELL_SCRIPTS[@]}"; do
  [[ -f "$script" ]] || { echo "missing shell script: $script" >&2; exit 1; }
  bash -n "$script"
  [[ "$(od -An -tx1 -N3 "$script" | tr -d ' \n')" != "efbbbf" ]] || {
    echo "UTF-8 BOM before shebang: $script" >&2
    exit 1
  }
  [[ -x "$script" ]] || { echo "shell script is not executable: $script" >&2; exit 1; }
done

node script/release-smoke.mjs
MYSQL_ROOT_PASSWORD=smoke-root MYSQL_PASSWORD=smoke-app \
  docker compose -f docker/docker-compose.yml config --quiet

echo "release smoke: shell and Compose checks passed"
