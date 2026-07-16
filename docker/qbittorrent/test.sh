#!/usr/bin/env bash
# 使用临时目录和假 Web API 验证完成钩子的成功与失败路径，不接触真实下载。

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
hook="$script_dir/auto_remove.sh"
fixture="$(mktemp -d "${TMPDIR:-/tmp}/qbittorrent-hook.XXXXXX")"
trap 'rm -rf "$fixture"' EXIT

mkdir -p "$fixture/bin" "$fixture/downloads" "$fixture/gdrive" "$fixture/config"

cat > "$fixture/bin/curl" <<'EOF'
#!/bin/sh
printf '%s\n' "$*" >> "$QBIT_TEST_CURL_LOG"
EOF
chmod 0755 "$fixture/bin/curl"

export QBIT_DEST_DIR="$fixture/gdrive"
export QBIT_LOG_FILE="$fixture/config/auto_remove.log"
export QBIT_REQUIRE_MOUNT=0
export QBIT_TEST_CURL_LOG="$fixture/curl.log"
export PATH="$fixture/bin:$PATH"

item="$fixture/downloads/completed.bin"
printf 'fixture' > "$item"

"$hook" "$item" '0123456789abcdef0123456789abcdef01234567' "$fixture/downloads"

[[ ! -e "$item" ]]
[[ -f "$fixture/gdrive/completed.bin" ]]
grep -F -- 'deleteFiles=false' "$fixture/curl.log" >/dev/null
grep -F -- '/api/v2/torrents/delete' "$fixture/curl.log" >/dev/null

blocked_item="$fixture/downloads/collision.bin"
printf 'source' > "$blocked_item"
printf 'destination' > "$fixture/gdrive/collision.bin"
curl_calls_before="$(wc -l < "$fixture/curl.log")"

if "$hook" "$blocked_item" 'fedcba9876543210fedcba9876543210fedcba98' "$fixture/downloads"; then
  printf 'expected collision path to fail\n' >&2
  exit 1
fi

[[ -f "$blocked_item" ]]
[[ "$(wc -l < "$fixture/curl.log")" == "$curl_calls_before" ]]

printf 'qBittorrent hook fixture tests passed.\n'
