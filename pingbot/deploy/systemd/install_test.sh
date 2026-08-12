#!/usr/bin/env bash
# Shell tests for install.sh. They run the script against a DESTDIR sandbox with
# fake system-management commands on PATH, so no host state is touched.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_SCRIPT="$SCRIPT_DIR/install.sh"

fails=0
fail() { echo "FAIL: $*" >&2; fails=$((fails + 1)); }
ok() { echo "ok: $*"; }

# --- sandbox ---------------------------------------------------------------
setup() {
  SANDBOX="$(mktemp -d)"
  SRC="$SANDBOX/src"
  FAKEBIN="$SANDBOX/fakebin"
  mkdir -p "$SRC" "$FAKEBIN"

  # Source files to install (also serve as SCRIPT_DIR defaults when install.sh
  # is copied next to them).
  printf '#!/bin/sh\necho fake-pingbot\n' > "$SRC/pingbot"
  chmod +x "$SRC/pingbot"
  printf 'enroll_token: new-secret\nprobe_secret: new-probe\n' > "$SRC/config.example.yaml"
  printf '[Unit]\nDescription=fake pingbot\n' > "$SRC/pingbot.service"
  cp "$INSTALL_SCRIPT" "$SRC/install.sh"

  # Fake system-management commands.
  cat > "$FAKEBIN/id" <<'EOF'
#!/usr/bin/env bash
if [[ "${1:-}" == "-u" && "${2:-}" == "pingbot" ]]; then echo 1000; exit 0; fi
exec /usr/bin/id "$@"
EOF
  for cmd in useradd usermod chown; do
    printf '#!/usr/bin/env bash\nexit 0\n' > "$FAKEBIN/$cmd"
    chmod +x "$FAKEBIN/$cmd"
  done
  cat > "$FAKEBIN/systemctl" <<'EOF'
#!/usr/bin/env bash
echo "systemctl $*" >> "${PINGBOT_TEST_SYSTEMD_LOG:-/dev/null}"
exit 0
EOF
  # Fake install that can inject a staged-config failure for the rollback test.
  cat > "$FAKEBIN/install" <<'EOF'
#!/usr/bin/env bash
if [[ "${PINGBOT_TEST_FAIL_INSTALL:-}" == "1" ]]; then
  for a in "$@"; do
    if [[ "$a" == *"/config.yaml.tmp."* ]]; then
      echo "fake install: simulated staged-config failure" >&2
      exit 1
    fi
  done
fi
exec /usr/bin/install "$@"
EOF
  chmod +x "$FAKEBIN/id" "$FAKEBIN/systemctl" "$FAKEBIN/install"

  export PATH="$FAKEBIN:$PATH"
  export PINGBOT_TEST_SYSTEMD_LOG="$SANDBOX/systemd.log"
  : > "$PINGBOT_TEST_SYSTEMD_LOG"

  ROOT="$SANDBOX/root"
  export DESTDIR="$ROOT"
  # Fresh sandbox: no existing runtime config; systemd dir mirrors a real host.
  mkdir -p "$ROOT/etc/pingbot" "$ROOT/etc/systemd/system"
}

teardown() {
  rm -rf "$SANDBOX"
}

run_install() {
  # Runs the copy of install.sh whose SCRIPT_DIR is the sandbox src dir.
  (cd "$SRC" && DESTDIR="$ROOT" bash install.sh "$@")
}

seed_config() {
  printf 'enroll_token: old-secret\nprobe_secret: old-probe\n' > "$ROOT/etc/pingbot/config.yaml"
}

config_content() {
  cat "$ROOT/etc/pingbot/config.yaml"
}

# --- tests -----------------------------------------------------------------

test_fresh_install() {
  run_install --replace-config >/dev/null || fail "fresh install exited non-zero"
  [[ -f "$ROOT/etc/pingbot/config.yaml" ]] || fail "fresh install did not create config.yaml"
  grep -q "new-secret" "$ROOT/etc/pingbot/config.yaml" || fail "fresh install wrote example config"
  [[ -x "$ROOT/opt/pingbot/pingbot" ]] || fail "fresh install did not place binary"
  ok "fresh install"
}

test_noarg_upgrade_keeps_config() {
  seed_config
  run_install >/dev/null 2>&1 || fail "no-arg upgrade exited non-zero"
  grep -q "old-secret" "$ROOT/etc/pingbot/config.yaml" || fail "no-arg upgrade overwrote existing config"
  ! grep -q "new-secret" "$ROOT/etc/pingbot/config.yaml" || fail "no-arg upgrade replaced config with example"
  ok "no-arg upgrade keeps existing config"
}

test_three_path_upgrade_keeps_config() {
  seed_config
  run_install "$SRC/pingbot" "$SRC/config.example.yaml" "$SRC/pingbot.service" >/dev/null 2>&1 \
    || fail "three-path upgrade exited non-zero"
  grep -q "old-secret" "$ROOT/etc/pingbot/config.yaml" || fail "three-path upgrade overwrote existing config"
  ok "three-path upgrade keeps existing config"
}

test_explicit_replace_backs_up_and_swaps() {
  seed_config
  run_install --replace-config "$SRC/pingbot" "$SRC/config.example.yaml" "$SRC/pingbot.service" >/dev/null \
    || fail "explicit replace exited non-zero"
  grep -q "new-secret" "$ROOT/etc/pingbot/config.yaml" || fail "explicit replace did not swap config"
  # A timestamped backup holding the old secret must exist.
  backups=("$ROOT"/etc/pingbot/config.yaml.bak.*)
  (( ${#backups[@]} >= 1 )) || fail "explicit replace did not create a backup"
  grep -q "old-secret" "${backups[0]}" || fail "backup does not hold the previous config"
  ok "explicit replace swaps config and keeps a backup"
}

test_failed_replace_restores_backup() {
  seed_config
  set +e
  PINGBOT_TEST_FAIL_INSTALL=1 run_install --replace-config >/dev/null 2>&1
  code=$?
  set -e
  [[ $code -ne 0 ]] || fail "failed replace should exit non-zero"
  grep -q "old-secret" "$ROOT/etc/pingbot/config.yaml" || fail "failed replace did not restore previous config"
  ok "failed replace restores previous config"
}

setup
test_fresh_install
test_noarg_upgrade_keeps_config
test_three_path_upgrade_keeps_config
test_explicit_replace_backs_up_and_swaps
test_failed_replace_restores_backup
teardown

if [[ $fails -gt 0 ]]; then
  echo "$fails test(s) failed" >&2
  exit 1
fi
echo "all install.sh tests passed"
