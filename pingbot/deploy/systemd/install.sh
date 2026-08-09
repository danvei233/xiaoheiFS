#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_SRC="${1:-$SCRIPT_DIR/pingbot}"
CFG_SRC="${2:-$SCRIPT_DIR/config.example.yaml}"
SERVICE_SRC="${3:-$SCRIPT_DIR/pingbot.service}"
CONFIG_EXPLICIT=false
if [[ $# -ge 2 ]]; then
  CONFIG_EXPLICIT=true
fi

install -d -m 0755 /opt/pingbot
install -d -m 0755 /etc/pingbot
install -m 0755 "$BIN_SRC" /opt/pingbot/pingbot
install -m 0600 "$CFG_SRC" /etc/pingbot/config.example.yaml
if [[ "$CONFIG_EXPLICIT" == true || ! -e /etc/pingbot/config.yaml ]]; then
  install -m 0600 "$CFG_SRC" /etc/pingbot/config.yaml
fi
install -m 0644 "$SERVICE_SRC" /etc/systemd/system/pingbot.service

systemctl daemon-reload
systemctl enable pingbot.service
systemctl restart pingbot.service
systemctl status --no-pager pingbot.service || true
