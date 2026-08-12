#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# The runtime config at /etc/pingbot/config.yaml is written by the agent and
# holds enroll_token/probe_secret. It must NEVER be silently overwritten by an
# upgrade script that happens to pass all three source paths, because that would
# wipe enrollment credentials. It is only replaced on a fresh install (no config
# yet) or when the operator explicitly passes --replace-config.
REPLACE_CONFIG=false
positional=()
for arg in "$@"; do
  case "$arg" in
    --replace-config)
      REPLACE_CONFIG=true
      ;;
    -h|--help)
      echo "usage: install.sh [--replace-config] [BIN] [CONFIG] [SERVICE]"
      echo
      echo "Installs pingbot. --replace-config atomically replaces an existing"
      echo "/etc/pingbot/config.yaml (with a timestamped backup); without it an"
      echo "existing runtime config is left untouched."
      exit 0
      ;;
    -*)
      echo "install.sh: unknown option $arg" >&2
      exit 2
      ;;
    *)
      positional+=("$arg")
      ;;
  esac
done

BIN_SRC="${positional[0]:-$SCRIPT_DIR/pingbot}"
CFG_SRC="${positional[1]:-$SCRIPT_DIR/config.example.yaml}"
SERVICE_SRC="${positional[2]:-$SCRIPT_DIR/pingbot.service}"
# DESTDIR prefixes the install root (packaging/staging convention); it is empty
# on a real host so the paths below are the production ones.
DESTDIR="${DESTDIR:-}"
PREFIX="$DESTDIR/opt/pingbot"
ETC="$DESTDIR/etc/pingbot"
SYSTEMD="$DESTDIR/etc/systemd/system"
CONFIG_PATH="$ETC/config.yaml"

install -d -m 0755 "$PREFIX"
install -d -m 0755 "$ETC"
install -m 0755 "$BIN_SRC" "$PREFIX/pingbot"
install -m 0600 "$CFG_SRC" "$ETC/config.example.yaml"

# Install the runtime config only when it does not exist yet (fresh install) or
# the operator explicitly asked to replace it.
if [[ -e "$CONFIG_PATH" && "$REPLACE_CONFIG" != true ]]; then
  echo "keep existing $CONFIG_PATH (use --replace-config to replace)"
elif [[ -e "$CONFIG_PATH" ]]; then
  # Explicit replace: refuse non-regular paths, then back up atomically before
  # swapping so a failed install can be rolled back.
  if [[ ! -f "$CONFIG_PATH" ]]; then
    echo "install.sh: refusing to replace non-regular file $CONFIG_PATH" >&2
    exit 1
  fi
  BACKUP_PATH="$CONFIG_PATH.bak.$(date +%s.%N)"
  if ! cp -p "$CONFIG_PATH" "$BACKUP_PATH"; then
    echo "install.sh: failed to back up existing config before replace" >&2
    exit 1
  fi
  chmod 0600 "$BACKUP_PATH"

  TMP_CONFIG="$CONFIG_PATH.tmp.$$"
  if ! install -m 0600 "$CFG_SRC" "$TMP_CONFIG"; then
    rm -f "$TMP_CONFIG"
    echo "install.sh: failed to stage new config; previous config kept" >&2
    exit 1
  fi
  if ! mv -f "$TMP_CONFIG" "$CONFIG_PATH"; then
    rm -f "$TMP_CONFIG"
    # Roll back: the previous config is restored from the backup.
    mv -f "$BACKUP_PATH" "$CONFIG_PATH" 2>/dev/null || true
    echo "install.sh: failed to swap config; restored previous config" >&2
    exit 1
  fi
  echo "replaced $CONFIG_PATH (backup: $BACKUP_PATH)"
else
  install -m 0600 "$CFG_SRC" "$CONFIG_PATH"
fi
install -m 0644 "$SERVICE_SRC" "$SYSTEMD/pingbot.service"

# Run the agent as a dedicated least-privilege system user instead of root.
if ! id -u pingbot >/dev/null 2>&1; then
  useradd --system --home-dir /opt/pingbot --shell /usr/sbin/nologin pingbot
fi
# Best-effort: adm/systemd-journal grant read access to the systemd journal
# for journal: log sources. Never fail the install over group membership.
usermod -aG adm,systemd-journal pingbot >/dev/null 2>&1 || usermod -aG adm pingbot >/dev/null 2>&1 || true

# The agent persists probe credentials back into $CONFIG_PATH and may keep
# local logs under $PREFIX; both must be owned by the service user for
# enrollment and log writes to succeed.
chown -R pingbot:pingbot "$PREFIX" "$ETC"

systemctl daemon-reload
systemctl enable pingbot.service
systemctl restart pingbot.service
systemctl status --no-pager pingbot.service || true
