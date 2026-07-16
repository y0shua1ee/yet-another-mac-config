#!/bin/sh
# 下载完成后把当前任务的数据搬到 Google Drive；只有搬移成功才删除 qBittorrent 任务。

set -u
umask 077

ITEM=${1-}
HASH=${2-}
SAVE_PATH=${3-}

DEST_DIR=${QBIT_DEST_DIR:-/gdrive}
LOG_FILE=${QBIT_LOG_FILE:-/config/auto_remove.log}
API_URL=${QBIT_API_URL:-http://localhost:8081}
REQUIRE_MOUNT=${QBIT_REQUIRE_MOUNT:-1}

log_message() {
  level=$1
  message=$2
  timestamp=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
  printf '%s [%s] %s\n' "$timestamp" "$level" "$message" >> "$LOG_FILE"
}

fail() {
  message=$1
  log_message ERROR "$message" 2>/dev/null || true
  printf 'qBittorrent completion hook: %s\n' "$message" >&2
  exit 1
}

if ! : >> "$LOG_FILE" 2>/dev/null; then
  printf 'qBittorrent completion hook: cannot write log %s\n' "$LOG_FILE" >&2
  exit 1
fi

[ -n "$ITEM" ] || fail 'missing completed item path'
[ -n "$HASH" ] || fail 'missing torrent info hash'
[ -n "$SAVE_PATH" ] || fail 'missing torrent save path'
[ -e "$ITEM" ] || fail 'completed item no longer exists'
[ -d "$DEST_DIR" ] || fail 'destination directory is unavailable'
[ -w "$DEST_DIR" ] || fail 'destination directory is not writable'

case "$REQUIRE_MOUNT" in
  0)
    ;;
  1)
    [ -r /proc/mounts ] || fail 'cannot verify destination mount'
    awk -v target="$DEST_DIR" '
      $2 == target { found = 1 }
      END { exit(found ? 0 : 1) }
    ' /proc/mounts || fail 'destination is not a mounted volume'
    ;;
  *)
    fail 'invalid QBIT_REQUIRE_MOUNT value'
    ;;
esac

save_root=${SAVE_PATH%/}
[ -n "$save_root" ] || save_root=/
case "$ITEM" in
  "$save_root"|"$save_root"/*)
    ;;
  *)
    fail 'completed item is outside the declared save path'
    ;;
esac

item_name=$(basename "$ITEM")
[ -n "$item_name" ] || fail 'completed item has no basename'
target_path="$DEST_DIR/$item_name"

if [ -e "$target_path" ] || [ -L "$target_path" ]; then
  fail 'destination already contains an item with the same name'
fi

hash_tag=$(printf '%.12s' "$HASH")
log_message INFO "moving completed item hash=$hash_tag"

if ! mv -- "$ITEM" "$DEST_DIR/"; then
  fail "move failed; torrent task retained hash=$hash_tag"
fi

log_message INFO "move completed; deleting torrent task hash=$hash_tag"

if ! curl -fsS --max-time 15 -X POST \
  --data-urlencode "hashes=$HASH" \
  --data 'deleteFiles=false' \
  "$API_URL/api/v2/torrents/delete" > /dev/null; then
  fail "task deletion failed after move hash=$hash_tag"
fi

log_message INFO "torrent task deleted without deleting data hash=$hash_tag"
