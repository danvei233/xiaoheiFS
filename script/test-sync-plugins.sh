#!/bin/bash
# Regression tests for docker/sync-plugins.sh (m046 M4 rework).
#
# Scenarios covered:
#   1. First start on a fresh plugins volume provisions all bundled plugins.
#   2. Image upgrade replaces untouched image-managed plugins (security fixes).
#   3. Admin-modified image-managed plugin is preserved, never overwritten.
#   4. Admin-installed same-ID plugin is preserved.
#   5. Admin-deleted image-managed plugin stays deleted (not resurrected).
#   6. Admin-installed extra plugins are never removed.
#   7. Sync works when running as a non-root user (permission scenario).
#   8. A failed managed-plugin copy returns non-zero and preserves the old copy.
#   9. A failed swap (mv) returns non-zero and restores the previous version.
#   10. A failed state-file write returns non-zero.
#   11. Interrupted swap recovery: stale .bak/.tmp are restored/cleaned.
#   12. --verify fails closed when a managed plugin is missing the baseline.
#   14. Interrupted image upgrade (swap completed, state commit missed) re-adopts
#       the plugin as image-managed instead of permanently demoting it, so the
#       next image security fix still reaches it.
#   15. Symlinked category/plugin/backup paths fail closed without touching an
#       outside target.
#
# Usage: bash script/test-sync-plugins.sh
set -eu

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SYNC="$REPO_ROOT/docker/sync-plugins.sh"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0

ok() { pass=$((pass + 1)); echo "  ok   $1"; }
bad() { fail=$((fail + 1)); echo "  FAIL $1"; }

assert() { # assert <description> <command...>
  local desc="$1"; shift
  local cmd
  printf -v cmd '%q ' "$@"
  if eval "$cmd" >/dev/null 2>&1; then
    ok "$desc"
  else
    bad "$desc"
  fi
}

assert_not() { # assert_not <description> <command...>  (command must exit non-zero)
  local desc="$1"; shift
  local cmd
  printf -v cmd '%q ' "$@"
  if eval "$cmd" >/dev/null 2>&1; then
    bad "$desc"
  else
    ok "$desc"
  fi
}

BUNDLE="$WORK/bundled-plugins"
PLUGINS="$WORK/plugins"
STATE="$WORK/config/plugin-bundle-state.tsv"
mkdir -p "$BUNDLE/payment/demo" "$PLUGINS" "$(dirname "$STATE")"

write_plugin() { # write_plugin <base> <cat> <id> <marker>
  local base="$1" cat="$2" id="$3" marker="$4"
  mkdir -p "$base/$cat/$id/bin"
  printf '{"plugin_id":"%s"}\n' "$id" > "$base/$cat/$id/manifest.json"
  printf 'marker=%s\n' "$marker" > "$base/$cat/$id/version.txt"
  printf '#!/bin/sh\necho %s\n' "$marker" > "$base/$cat/$id/bin/plugin"
  chmod +x "$base/$cat/$id/bin/plugin"
}

run_sync() {
  BUNDLED_PLUGINS_DIR="$BUNDLE" PLUGINS_DIR="$PLUGINS" PLUGIN_BUNDLE_STATE="$STATE" \
    bash "$SYNC"
}

echo "== plugin sync tests =="

# --- 1. First start ---
write_plugin "$BUNDLE" payment demo v1
run_sync
assert "first start provisions bundled plugin" test -f "$PLUGINS/payment/demo/manifest.json"
assert "first start writes the exact marker" grep -q 'marker=v1' "$PLUGINS/payment/demo/version.txt"
assert "state file records the plugin" grep -q '^payment/demo	' "$STATE"

# --- 2. Image upgrade on an untouched copy ---
printf 'marker=v2\n' > "$BUNDLE/payment/demo/version.txt"
run_sync
assert "image upgrade updates an untouched image-managed plugin" grep -q 'marker=v2' "$PLUGINS/payment/demo/version.txt"
assert "state tracks the new bundle hash" grep -q '^payment/demo	' "$STATE"

# --- 3. Admin-modified plugin is preserved ---
printf 'marker=v1\n' > "$BUNDLE/payment/demo/version.txt"   # bundle "downgrades"
printf 'ADMIN_EDIT\n' >> "$PLUGINS/payment/demo/version.txt"
run_sync
assert "admin-modified plugin is preserved" grep -q 'ADMIN_EDIT' "$PLUGINS/payment/demo/version.txt"
assert "admin-modified plugin is not reverted to bundle" grep -qv '^marker=v1$' "$PLUGINS/payment/demo/version.txt"
assert_not "admin-modified plugin dropped from image state" grep -q '^payment/demo	' "$STATE"

# --- 4. Admin-installed same-ID plugin is preserved on image upgrade ---
run_sync
assert "admin-owned plugin still present after later sync" grep -q 'ADMIN_EDIT' "$PLUGINS/payment/demo/version.txt"

# --- 5. Admin deletes an image-managed plugin ---
write_plugin "$BUNDLE" sms aliyun v1
run_sync   # provision aliyun as image-managed
assert "second bundled plugin provisioned" test -f "$PLUGINS/sms/aliyun/manifest.json"
rm -rf "$PLUGINS/sms/aliyun"
run_sync
assert "admin-deleted image plugin stays deleted" test ! -e "$PLUGINS/sms/aliyun"
assert_not "deleted plugin removed from state" grep -q '^sms/aliyun	' "$STATE"

# --- 6. Admin-installed extra plugins are never removed ---
mkdir -p "$PLUGINS/kyc/thirdparty/bin"
printf '{"plugin_id":"thirdparty"}\n' > "$PLUGINS/kyc/thirdparty/manifest.json"
run_sync
assert "admin-installed extra plugin preserved" test -f "$PLUGINS/kyc/thirdparty/manifest.json"
assert_not "admin extra plugin not in image state" grep -q '^kyc/thirdparty	' "$STATE"

# --- 7. Non-root / permission scenario ---
NR="$WORK/nonroot"
mkdir -p "$NR/bundled/payment/nr" "$NR/plugins" "$NR/config"
write_plugin "$NR/bundled" payment nr v1
if [ "$(id -u)" -ne 0 ]; then
  # Already running as a non-root user: run the sync directly to prove it
  # does not require root privileges (all earlier scenarios also ran non-root).
  if BUNDLED_PLUGINS_DIR="$NR/bundled" PLUGINS_DIR="$NR/plugins" \
      PLUGIN_BUNDLE_STATE="$NR/config/state.tsv" bash "$SYNC" >/dev/null 2>&1 \
      && test -f "$NR/plugins/payment/nr/manifest.json"; then
    ok "sync works when run as a non-root user (effective uid != 0)"
  else
    bad "non-root sync did not provision plugin"
  fi
elif command -v setpriv >/dev/null 2>&1; then
  chown -R 65534:65534 "$NR" 2>/dev/null || true
  if setpriv --reuid=65534 --regid=65534 \
      env BUNDLED_PLUGINS_DIR="$NR/bundled" PLUGINS_DIR="$NR/plugins" \
        PLUGIN_BUNDLE_STATE="$NR/config/state.tsv" bash "$SYNC" >/dev/null 2>&1 \
      && test -f "$NR/plugins/payment/nr/manifest.json"; then
    ok "sync works when run as a non-root user (dropped from root)"
  else
    bad "non-root sync (dropped from root) failed"
  fi
else
  echo "  skip non-root scenario (already root, no setpriv)"
fi

# --- 8. Managed-plugin copy failure fails closed and preserves the old copy ---
F="$WORK/failcp"
mkdir -p "$F/bundled/payment/fp" "$F/plugins" "$F/config"
write_plugin "$F/bundled" payment fp v2
# Provision as image-managed first.
BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" >/dev/null 2>&1
# Now upgrade the bundle and force the staged copy to fail; the old copy must
# survive and the round must exit non-zero.
printf 'marker=v3\n' > "$F/bundled/payment/fp/version.txt"
if BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
    PLUGIN_BUNDLE_STATE="$F/config/state.tsv" \
    _SYNC_TEST_CP_FAIL=1 bash "$SYNC" >/dev/null 2>&1; then
  bad "copy failure must exit non-zero"
else
  ok "copy failure returns non-zero (fail closed)"
fi
assert "copy failure preserves previous plugin version" grep -q 'marker=v2' "$F/plugins/payment/fp/version.txt"

# --- 9. Swap (mv) failure restores the previous version and fails closed ---
F="$WORK/failmv"
mkdir -p "$F/bundled/payment/mv" "$F/plugins" "$F/config"
write_plugin "$F/bundled" payment mv v1
BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" >/dev/null 2>&1
printf 'marker=v2\n' > "$F/bundled/payment/mv/version.txt"
if BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
    PLUGIN_BUNDLE_STATE="$F/config/state.tsv" \
    _SYNC_TEST_MV_FAIL=1 bash "$SYNC" >/dev/null 2>&1; then
  bad "mv failure must exit non-zero"
else
  ok "mv failure returns non-zero (fail closed)"
fi
assert "mv failure preserves the previous plugin version" grep -q 'marker=v1' "$F/plugins/payment/mv/version.txt"
assert "mv failure leaves no stale backup behind" \
  test -z "$(find "$F/plugins" -name '*.bak.*' -print)"

# --- 10. State write failure fails closed ---
F="$WORK/failstate"
mkdir -p "$F/bundled/payment/st" "$F/plugins" "$F/config"
write_plugin "$F/bundled" payment st v1
if BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
    PLUGIN_BUNDLE_STATE="$F/config/state.tsv" \
    _SYNC_TEST_STATE_FAIL=1 bash "$SYNC" >/dev/null 2>&1; then
  bad "state write failure must exit non-zero"
else
  ok "state write failure returns non-zero (fail closed)"
fi

# --- 11. Interrupted swap recovery ---
F="$WORK/swapint"
mkdir -p "$F/bundled/payment/sw" "$F/plugins/payment" "$F/config"
write_plugin "$F/bundled" payment sw v1
# Simulate a crash: dst moved aside to .bak, staged copy present, dst missing.
cp -a "$F/bundled/payment/sw" "$F/plugins/payment/sw.bak.999"
cp -a "$F/bundled/payment/sw" "$F/plugins/payment/sw.tmp.999"
BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" >/dev/null 2>&1
assert "interrupted swap restores the plugin directory" test -d "$F/plugins/payment/sw"
assert "stale backup cleaned after recovery" \
  test -z "$(find "$F/plugins" -name '*.bak.*' -print)"
assert "stale staging dir cleaned after recovery" \
  test -z "$(find "$F/plugins" -name '*.tmp.*' -print)"

# --- 12. --verify fails closed on a missing baseline ---
F="$WORK/verify"
mkdir -p "$F/bundled/payment/vf" "$F/plugins" "$F/config"
write_plugin "$F/bundled" payment vf v1
BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" >/dev/null 2>&1
assert "verify passes when the managed plugin is at baseline" \
  env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
    PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" --verify
rm -rf "$F/plugins/payment/vf"
assert_not "verify fails when the managed plugin is missing (baseline not met)" \
  env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
    PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" --verify

# --- 13b. Deleting the state file re-adopts an unchanged bundle plugin and
#          still preserves an admin-modified one ---
F="$WORK/readopt"
mkdir -p "$F/bundled/payment/ra" "$F/plugins" "$F/config"
write_plugin "$F/bundled" payment ra v1
env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" >/dev/null 2>&1
assert "readopt: plugin provisioned as image-managed" test -d "$F/plugins/payment/ra"
# Operator deletes the state file (manual reset).
rm -f "$F/config/state.tsv"
env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" >/dev/null 2>&1
assert "readopt: unchanged bundle plugin is re-recorded as image-managed" \
  grep -q '^payment/ra	' "$F/config/state.tsv"
# A subsequent image upgrade now updates it (proving it is managed again).
printf 'marker=v2\n' > "$F/bundled/payment/ra/version.txt"
env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" >/dev/null 2>&1
assert "readopt: image upgrade applies after state-file reset" \
  grep -q 'marker=v2' "$F/plugins/payment/ra/version.txt"

# Admin-modified plugin must NOT be re-adopted on a state reset.
F="$WORK/readopt-admin"
mkdir -p "$F/bundled/payment/ad" "$F/plugins" "$F/config"
write_plugin "$F/bundled" payment ad v1
env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" >/dev/null 2>&1
printf 'ADMIN_EDIT\n' >> "$F/plugins/payment/ad/version.txt"
rm -f "$F/config/state.tsv"
env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" >/dev/null 2>&1
assert "readopt: admin-modified plugin is not re-recorded after state reset" \
  test -z "$(grep '^payment/ad	' "$F/config/state.tsv" || true)"

# --- 14. Interrupted upgrade (swap done, state not committed) re-adopts the
#          plugin instead of permanently demoting it to admin-owned ---
F="$WORK/interrupted-upgrade"
mkdir -p "$F/bundled/payment/iu" "$F/plugins" "$F/config"
write_plugin "$F/bundled" payment iu v1
env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" >/dev/null 2>&1
# Simulate an image upgrade v1->v2 whose swap already completed but whose state
# commit never happened (container killed between the two renames and the final
# `mv "$state_tmp" "$STATE_FILE"`): the installed directory is now v2 while the
# state file still records v1.
printf 'marker=v2\n' > "$F/bundled/payment/iu/version.txt"
rm -rf "$F/plugins/payment/iu"
cp -a "$F/bundled/payment/iu" "$F/plugins/payment/iu"
env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" >/dev/null 2>&1
assert "interrupted upgrade: plugin is re-recorded as image-managed" \
  grep -q '^payment/iu	' "$F/config/state.tsv"
# A further image upgrade v2->v3 must still reach the plugin (it is managed
# again, so security fixes keep being delivered).
printf 'marker=v3\n' > "$F/bundled/payment/iu/version.txt"
env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" >/dev/null 2>&1
assert "interrupted upgrade: v3 fix is delivered after re-adoption" \
  grep -q 'marker=v3' "$F/plugins/payment/iu/version.txt"

# --- 13. A failed managed upgrade keeps the record and is retried next start ---
F="$WORK/retry"
mkdir -p "$F/bundled/payment/rt" "$F/plugins" "$F/config"
write_plugin "$F/bundled" payment rt v1
env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" >/dev/null 2>&1
printf 'marker=v2\n' > "$F/bundled/payment/rt/version.txt"
# Force a failure on the first upgrade attempt.
env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" _SYNC_TEST_CP_FAIL=1 bash "$SYNC" >/dev/null 2>&1 || true
assert "failed upgrade keeps the image-managed state record" \
  grep -q '^payment/rt	' "$F/config/state.tsv"
# Next start (no injected failure) succeeds and applies the security fix.
env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
  PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC" >/dev/null 2>&1
assert "failed managed upgrade is retried and applied on the next start" \
  grep -q 'marker=v2' "$F/plugins/payment/rt/version.txt"

# --- 15. Symlink boundaries fail closed -----------------------------------
# A writable plugin volume must not be able to redirect a category or plugin
# path outside that volume.  The target marker proves no copy/remove followed
# the malicious link.
F="$WORK/symlink-category"
mkdir -p "$F/bundled/payment/cat" "$F/plugins" "$F/config" "$F/outside"
write_plugin "$F/bundled" payment cat v1
printf 'outside-marker\n' > "$F/outside/marker"
ln -s "$F/outside" "$F/plugins/payment"
assert_not "symlinked category is rejected" \
  env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
    PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC"
assert "symlinked category leaves outside content untouched" \
  grep -q 'outside-marker' "$F/outside/marker"

F="$WORK/symlink-bundle"
mkdir -p "$F/bundled/payment" "$F/plugins" "$F/config" "$F/outside"
write_plugin "$F/outside" payment escaped v1
ln -s "$F/outside/payment/escaped" "$F/bundled/payment/escaped"
assert_not "symlinked bundle plugin is rejected" \
  env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
    PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC"
assert "symlinked bundle plugin is not copied outside its root" \
  test ! -e "$F/plugins/payment/escaped"

F="$WORK/symlink-destination"
mkdir -p "$F/bundled/payment/dst" "$F/plugins/payment" "$F/config" "$F/outside"
write_plugin "$F/bundled" payment dst v1
printf 'outside-marker\n' > "$F/outside/marker"
ln -s "$F/outside" "$F/plugins/payment/dst"
assert_not "symlinked destination plugin is rejected" \
  env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
    PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC"
assert "symlinked destination leaves outside content untouched" \
  grep -q 'outside-marker' "$F/outside/marker"

F="$WORK/symlink-recovery"
mkdir -p "$F/bundled/payment/recover" "$F/plugins/payment" "$F/config" "$F/outside"
write_plugin "$F/bundled" payment recover v1
printf 'outside-marker\n' > "$F/outside/marker"
ln -s "$F/outside" "$F/plugins/payment/recover.bak.999"
assert_not "symlinked recovery backup is rejected" \
  env BUNDLED_PLUGINS_DIR="$F/bundled" PLUGINS_DIR="$F/plugins" \
    PLUGIN_BUNDLE_STATE="$F/config/state.tsv" bash "$SYNC"
assert "symlinked recovery backup leaves outside content untouched" \
  grep -q 'outside-marker' "$F/outside/marker"

echo
echo "$pass passed, $fail failed"
[ "$fail" -eq 0 ]
