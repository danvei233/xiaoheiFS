#!/bin/sh
set -eu

cd /app
mkdir -p config data uploads plugins

if [ -d bundled-plugins ]; then
  cp -a bundled-plugins/. plugins/
fi

link_persisted_file() {
  name="$1"
  target="$2"
  if [ ! -e "$name" ] && [ ! -L "$name" ]; then
    ln -s "$target" "$name"
  fi
}

link_persisted_file app.config.yaml config/app.config.yaml
link_persisted_file install.lock config/install.lock
link_persisted_file init.lock data/init.lock

exec "$@"
