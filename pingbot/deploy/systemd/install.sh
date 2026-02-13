#!/usr/bin/env bash
set -euo pipefail

BIN_SRC="${1:-./pingbot}"
CFG_SRC="${2:-./config.yaml}"

install -d -m 0755 /opt/pingbot
install -d -m 0755 /etc/pingbot
install -m 0755 "$BIN_SRC" /opt/pingbot/pingbot
install -m 0600 "$CFG_SRC" /etc/pingbot/config.yaml
install -m 0644 ./pingbot.service /etc/systemd/system/pingbot.service

systemctl daemon-reload
systemctl enable --now pingbot.service
systemctl status --no-pager pingbot.service || true
