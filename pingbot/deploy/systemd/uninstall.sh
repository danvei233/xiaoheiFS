#!/usr/bin/env bash
set -euo pipefail

systemctl disable --now pingbot.service || true
rm -f /etc/systemd/system/pingbot.service
systemctl daemon-reload

echo "pingbot service removed. data under /etc/pingbot and /opt/pingbot kept."
