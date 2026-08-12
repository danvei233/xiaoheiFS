#!/bin/sh
set -eu

cd /app
mkdir -p config data uploads plugins

# Sync image-bundled plugins into the persistent plugins directory. The sync
# distinguishes image-managed copies (upgraded on image upgrade so security
# fixes arrive) from admin-managed plugins (never silently overwritten).
#
# Fail-closed policy: if a managed plugin cannot be brought to the bundle's
# security baseline (or the baseline cannot be verified), the container must
# not start with a stale, potentially vulnerable plugin set — we abort. The
# previous working plugin layout is preserved by the sync script's atomic
# rename/backup strategy, so the failure is recoverable by fixing the
# underlying cause and restarting.
if [ -d bundled-plugins ]; then
  if [ ! -x /usr/local/bin/xiaohei-sync-plugins ]; then
    echo "error: xiaohei-sync-plugins is missing; cannot guarantee the plugin security baseline" >&2
    exit 1
  fi
  if ! /usr/local/bin/xiaohei-sync-plugins; then
    echo "error: plugin sync failed; refusing to start with a stale managed plugin set" >&2
    exit 1
  fi
  if ! /usr/local/bin/xiaohei-sync-plugins --verify; then
    echo "error: managed plugin baseline verification failed; refusing to start" >&2
    exit 1
  fi
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

# Run the server and its plugin children as the non-root app user. When the
# entrypoint starts as root (the default), hand the runtime directories to the
# app user first so both fresh and pre-existing volumes stay writable, then
# drop privileges before exec. Without this the web process and every plugin
# subprocess would run as root in the container.
APP_USER="${APP_USER:-xiaohei}"

if [ "$(id -u)" -eq 0 ]; then
  # Fail closed on the non-root control: if the configured APP_USER does not
  # exist, or no privilege-dropping helper is installed, refuse to start rather
  # than silently running the web process and every plugin subprocess as root.
  if ! id "$APP_USER" >/dev/null 2>&1; then
    echo "error: APP_USER '$APP_USER' does not exist in this image; refusing to run as root" >&2
    exit 1
  fi
  chown "$APP_USER":"$APP_USER" /app 2>/dev/null || true
  chown -R "$APP_USER":"$APP_USER" /app/config /app/data /app/uploads /app/plugins 2>/dev/null || true
  if command -v su-exec >/dev/null 2>&1; then
    exec su-exec "$APP_USER:$APP_USER" "$@"
  fi
  if command -v setpriv >/dev/null 2>&1; then
    exec setpriv --reuid="$APP_USER" --regid="$APP_USER" --clear-groups "$@"
  fi
  echo "error: no su-exec/setpriv available; refusing to run the server as root" >&2
  exit 1
fi

exec "$@"
