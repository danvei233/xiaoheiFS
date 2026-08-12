#!/bin/sh
# Sync image-bundled plugins into the persistent plugins directory without
# silently overwriting admin-managed installations.
#
# Ownership model (m046 M4 rework):
#   - Each bundled plugin that the image places into plugins/ is recorded in a
#     state file keyed by plugin path -> SHA-256 of its content.
#   - On a later start a plugin whose recorded hash still matches its current
#     content is an untouched image-managed copy. When the image ships a newer
#     bundle we replace it, so image upgrades deliver security fixes.
#   - Any difference from the recorded hash, absence from the state, or a
#     deleted plugin directory means the admin installed/modified/removed it:
#     we leave it alone and drop it from the state.
#   - We never delete anything from plugins/.
#
# Atomicity & failure semantics:
#   - Every copy is staged to a temp dir and verified before the swap, so a
#     failed/partial copy never destroys the previous working version.
#   - The swap uses two same-filesystem renames (dst -> backup, then staged ->
#     dst) so a crash between them leaves the old plugin recoverable (see the
#     stale-backup recovery pass below) instead of a half-written directory.
#   - If ANY managed plugin cannot be applied, the script exits non-zero so the
#     entrypoint can fail closed instead of starting a stale, possibly
#     vulnerable, managed plugin.
#
# --verify mode re-checks that every recorded image-managed plugin is still at
# the bundle's content baseline; the entrypoint runs it after a successful sync
# so a partially-written plugin directory never reaches the running server.
#
# Env overrides allow tests to run against a scratch layout:
#   BUNDLED_PLUGINS_DIR, PLUGINS_DIR, PLUGIN_BUNDLE_STATE
set -eu

BUNDLE_DIR="${BUNDLED_PLUGINS_DIR:-/app/bundled-plugins}"
PLUGINS_DIR="${PLUGINS_DIR:-/app/plugins}"
STATE_FILE="${PLUGIN_BUNDLE_STATE:-/app/config/plugin-bundle-state.tsv}"

log() { echo "sync-plugins: $*" >&2; }

# The plugin volume is writable by the container user.  Never allow a
# symlink in a path we are about to remove or rename: a path such as
# plugins/payment -> /outside would otherwise turn an image upgrade into an
# arbitrary write/delete primitive.  Keep this deliberately POSIX so the
# guard also works in the Alpine image.
path_chain_safe() {
  pcs_path="$1"
  case "$pcs_path" in /*) ;; *) return 1 ;; esac
  pcs_rest="${pcs_path#/}"
  pcs_cur="/"
  while [ -n "$pcs_rest" ]; do
    case "$pcs_rest" in
      */*) pcs_part="${pcs_rest%%/*}"; pcs_rest="${pcs_rest#*/}" ;;
      *) pcs_part="$pcs_rest"; pcs_rest="" ;;
    esac
    [ -n "$pcs_part" ] || continue
    [ "$pcs_part" != "." ] && [ "$pcs_part" != ".." ] || return 1
    [ "$pcs_cur" = "/" ] && pcs_cur="/$pcs_part" || pcs_cur="$pcs_cur/$pcs_part"
    [ ! -L "$pcs_cur" ] || return 1
  done
}

path_inside_root() {
  pir_root="$1"
  pir_path="$2"
  pir_allow_missing="${3:-0}"
  path_chain_safe "$pir_root" || return 1
  path_chain_safe "$pir_path" || return 1
  [ ! -L "$pir_root" ] || return 1
  [ -d "$pir_root" ] || return 1
  pir_root_real="$(realpath "$pir_root" 2>/dev/null)" || return 1
  if [ -e "$pir_path" ]; then
    pir_target_real="$(realpath "$pir_path" 2>/dev/null)" || return 1
  elif [ "$pir_allow_missing" = "1" ]; then
    pir_target_real="$(realpath "$(dirname "$pir_path")" 2>/dev/null)" || return 1
  else
    return 1
  fi
  case "$pir_root_real" in /) ;; *) pir_root_real="${pir_root_real%/}" ;; esac
  case "$pir_target_real" in
    "$pir_root_real"|"$pir_root_real"/*) return 0 ;;
    *) return 1 ;;
  esac
}

require_path() {
  rp_root="$1"
  rp_path="$2"
  rp_allow_missing="${3:-0}"
  if ! path_inside_root "$rp_root" "$rp_path" "$rp_allow_missing"; then
    log "error: unsafe path (symlink, traversal, or outside root): $rp_path"
    return 1
  fi
}

require_root() {
  root="$1"
  path_chain_safe "$root" || {
    log "error: unsafe root: $root"
    return 1
  }
  if [ -L "$root" ] || { [ -e "$root" ] && [ ! -d "$root" ]; }; then
    log "error: unsafe root: $root"
    return 1
  fi
  [ -e "$root" ] || return 0
  require_path "$root" "$root"
}

# hash_dir <dir> -> stable sha256 of file names + contents.
hash_dir() {
  dir="$1"
  [ -d "$dir" ] && [ ! -L "$dir" ] || return 1
  # cp -a preserves links.  Refuse a bundle containing one rather than
  # allowing a later hash/copy operation to interpret it differently.
  [ -z "$(find "$dir" -type l -print -quit 2>/dev/null)" ] || return 1
  ( cd "$dir" && find . -type f -print0 | sort -z | xargs -0 sha256sum 2>/dev/null | sha256sum | cut -d' ' -f1 )
}

if ! require_root "$PLUGINS_DIR"; then exit 1; fi
if ! require_root "$BUNDLE_DIR"; then exit 1; fi
state_dir="$(dirname "$STATE_FILE")"
if ! require_root "$state_dir"; then exit 1; fi
if [ -e "$STATE_FILE" ] || [ -L "$STATE_FILE" ]; then
  if ! require_path "$state_dir" "$STATE_FILE"; then exit 1; fi
fi

# --- --verify mode ---------------------------------------------------------
# For every plugin recorded in the state file, the installed content hash must
# still equal the bundle hash it was synced from. A mismatch means the managed
# plugin is no longer at the security baseline: fail closed.
if [ "${1:-}" = "--verify" ]; then
  if [ ! -f "$STATE_FILE" ]; then
    exit 0
  fi
  bad=0
  while IFS="$(printf '\t')" read -r rel bundle_hash installed_hash; do
    [ -n "$rel" ] || continue
    case "$rel" in \#*) continue ;; esac
    case "$rel" in
      */*) category="${rel%%/*}"; plugin="${rel#*/}" ;;
      *) log "verify: malformed state path $rel"; bad=1; continue ;;
    esac
    [ -n "$category" ] && [ -n "$plugin" ] && [ "$plugin" != '*/*' ] || { log "verify: malformed state path $rel"; bad=1; continue; }
    dst="$PLUGINS_DIR/$rel"
    if ! require_path "$PLUGINS_DIR" "$dst"; then bad=1; continue; fi
    if [ ! -d "$dst" ]; then
      log "verify: managed plugin $rel is missing (bundle baseline required)"
      bad=1
      continue
    fi
    cur="$(hash_dir "$dst")"
    if [ "$cur" != "${installed_hash:-$bundle_hash}" ] || [ "$cur" != "$bundle_hash" ]; then
      log "verify: managed plugin $rel is not at the bundle baseline"
      bad=1
    fi
  done < "$STATE_FILE"
  exit $bad
fi

[ -d "$BUNDLE_DIR" ] || exit 0
if [ ! -d "$PLUGINS_DIR" ]; then
  log "error: plugins dir missing: $PLUGINS_DIR"
  exit 1
fi

mkdir -p "$state_dir" 2>/dev/null || { log "error: cannot create state dir $state_dir"; exit 1; }
require_root "$state_dir" || exit 1
state_tmp="${STATE_FILE}.tmp.$$"
require_path "$state_dir" "$state_tmp" 1 || exit 1
: > "$state_tmp" 2>/dev/null || { log "error: cannot create state file $state_tmp"; exit 1; }

# --- test-only fault injection hooks (never set in production) -------------
fail_cp()   { [ "${_SYNC_TEST_CP_FAIL:-0}" = "1" ]; }
fail_mv()   { [ "${_SYNC_TEST_MV_FAIL:-0}" = "1" ]; }
fail_state(){ [ "${_SYNC_TEST_STATE_FAIL:-0}" = "1" ]; }

# copy_plugin <src> <dst>: stage to temp, verify, then swap atomically.
# On failure the previous dst (if any) is left intact in a recoverable backup.
copy_plugin() {
  src="$1"
  dst="$2"
  tmp="${dst}.tmp.$$"
  bak="${dst}.bak.$$"
  require_path "$BUNDLE_DIR" "$src" || return 1
  require_path "$PLUGINS_DIR" "$(dirname "$dst")" 1 || return 1
  require_path "$PLUGINS_DIR" "$dst" 1 || return 1
  require_path "$PLUGINS_DIR" "$tmp" 1 || return 1
  require_path "$PLUGINS_DIR" "$bak" 1 || return 1
  rm -rf "$tmp" "$bak"
  if fail_cp || ! cp -a "$src" "$tmp"; then
    rm -rf "$tmp" "$bak"
    log "error: staging copy failed for $dst"
    return 1
  fi
  require_path "$PLUGINS_DIR" "$tmp" || { rm -rf "$tmp"; return 1; }
  if [ "$(hash_dir "$tmp")" != "$(hash_dir "$src")" ]; then
    rm -rf "$tmp" "$bak"
    log "error: staged copy hash mismatch for $dst"
    return 1
  fi
  # Move the current version aside (same filesystem, atomic rename).
  if [ -e "$dst" ]; then
    if fail_mv || ! mv "$dst" "$bak"; then
      rm -rf "$tmp" "$bak"
      log "error: cannot move current $dst aside"
      return 1
    fi
  fi
  require_path "$PLUGINS_DIR" "$bak" 1 || { rm -rf "$tmp"; return 1; }
  # Install the staged copy atomically.
  if fail_mv || ! mv "$tmp" "$dst"; then
    if [ -e "$bak" ] && require_path "$PLUGINS_DIR" "$bak"; then
      mv "$bak" "$dst" 2>/dev/null || true
    fi
    rm -rf "$tmp"
    log "error: cannot install $dst"
    return 1
  fi
  rm -rf "$bak"
  return 0
}

# --- recover from an interrupted swap --------------------------------------
# A crash between the two renames above leaves <plugin>.bak.* present (the old
# version) and <plugin> possibly missing. Restore it so the previous working
# plugin is never lost, then drop stale staging dirs.
recover_interrupted_swaps() {
  find "$PLUGINS_DIR" \( -name '*.bak.*' -o -name '*.tmp.*' \) -print 2>/dev/null | while read -r p; do
    require_path "$PLUGINS_DIR" "$p" || exit 1
    case "$p" in
      *.bak.*)
        base="${p%.bak.*}"
        require_path "$PLUGINS_DIR" "$base" 1 || exit 1
        if [ -d "$base" ]; then
          rm -rf "$p"
        else
          log "recover: restoring $p -> $base"
          mv "$p" "$base" 2>/dev/null || true
        fi
        ;;
      *.tmp.*)
        rm -rf "$p"
        ;;
    esac
  done
}

# --- load previous state ---------------------------------------------------
existing_file="${state_tmp}.prev"
require_path "$state_dir" "$existing_file" 1 || exit 1
if [ -f "$STATE_FILE" ]; then
  cp "$STATE_FILE" "$existing_file" 2>/dev/null || { log "error: cannot read previous state"; exit 1; }
else
  : > "$existing_file"
fi

prev_hash_of() { # prev_hash_of <rel> -> recorded bundle hash (or empty)
  rel="$1"
  awk -F'\t' -v rel="$rel" '$1 == rel { print $2 }' "$existing_file" | head -n 1
}

failures=0
recorded=0

recover_interrupted_swaps || exit 1

for cat_dir in "$BUNDLE_DIR"/*/; do
  [ -d "$cat_dir" ] || continue
  require_path "$BUNDLE_DIR" "$cat_dir" || exit 1
  cat_name="$(basename "$cat_dir")"
  case "$cat_name" in .*) continue ;; esac
  for plugin_dir in "$cat_dir"*/; do
    [ -d "$plugin_dir" ] || continue
    require_path "$BUNDLE_DIR" "$plugin_dir" || exit 1
    plugin_id="$(basename "$plugin_dir")"
    case "$plugin_id" in .*) continue ;; esac
    rel="${cat_name}/${plugin_id}"
    dst="$PLUGINS_DIR/$rel"
    require_path "$PLUGINS_DIR" "$dst" 1 || { failures=$((failures+1)); continue; }
    bundle_hash="$(hash_dir "$plugin_dir")"
    [ -n "$bundle_hash" ] || continue

    if [ ! -e "$dst" ]; then
      # Missing target. Provision it only if it was never image-managed
      # (fresh volume / first start). If it WAS image-managed and is now
      # missing, the admin deleted it: respect the deletion.
      prev="$(prev_hash_of "$rel")"
      if [ -z "$prev" ]; then
        mkdir -p "$PLUGINS_DIR/$cat_name" 2>/dev/null || { log "error: cannot mkdir $PLUGINS_DIR/$cat_name"; failures=$((failures+1)); continue; }
        require_path "$PLUGINS_DIR" "$PLUGINS_DIR/$cat_name" || { failures=$((failures+1)); continue; }
        if copy_plugin "$plugin_dir" "$dst"; then
          installed="$(hash_dir "$dst")"
          printf '%s\t%s\t%s\n' "$rel" "$bundle_hash" "$installed" >> "$state_tmp"
          recorded=$((recorded+1))
        else
          failures=$((failures+1))
        fi
      fi
      continue
    fi

    prev="$(prev_hash_of "$rel")"
    cur="$(hash_dir "$dst")"
    if [ -n "$prev" ] && [ "$prev" = "$cur" ]; then
      # Untouched image-managed copy -> upgrade to the new bundle.
      if copy_plugin "$plugin_dir" "$dst"; then
        installed="$(hash_dir "$dst")"
        printf '%s\t%s\t%s\n' "$rel" "$bundle_hash" "$installed" >> "$state_tmp"
        recorded=$((recorded+1))
      else
        # Upgrade failed: keep the OLD record so the plugin remains
        # image-managed and the next start retries the upgrade. The round
        # already exits non-zero, so the entrypoint fails closed.
        failures=$((failures+1))
        if [ -n "$prev" ]; then
          printf '%s\t%s\t%s\n' "$rel" "$prev" "$cur" >> "$state_tmp"
        fi
      fi
    else
      # Admin-owned or modified, OR the record is stale from an interrupted
      # image upgrade (the new bundle was already swapped in but the state file
      # still holds the previous hash). Preserve the directory either way, but
      # when the installed content is byte-identical to the CURRENT bundle we
      # re-adopt it as image-managed, so a crash between the swap and the state
      # commit never permanently demotes a security-baseline copy to admin-owned
      # (and later image upgrades keep delivering fixes). A genuinely
      # admin-modified copy differs from the bundle and stays admin-owned.
      if [ "$cur" = "$bundle_hash" ]; then
        printf '%s\t%s\t%s\n' "$rel" "$bundle_hash" "$cur" >> "$state_tmp"
        recorded=$((recorded+1))
      fi
    fi
  done
done

# Persist atomically; a failed write means the managed set is not reliably
# recorded, so the round must fail closed too.
require_path "$state_dir" "$STATE_FILE" 1 || { rm -f "$state_tmp" "$existing_file"; exit 1; }
if fail_state || ! mv "$state_tmp" "$STATE_FILE" 2>/dev/null; then
  log "error: cannot persist plugin bundle state"
  rm -f "$state_tmp" "$existing_file"
  exit 1
fi
sync 2>/dev/null || true
rm -f "$existing_file"

if [ "$failures" -gt 0 ]; then
  log "error: $failures managed plugin(s) could not be synced; refusing to report success"
  exit 1
fi
[ "$recorded" -gt 0 ] && log "synced $recorded managed plugin(s)"
exit 0
