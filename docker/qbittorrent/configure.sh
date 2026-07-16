#!/usr/bin/env bash
# 通过容器内 loopback Web API 应用并核对 qBittorrent 完成钩子设置。

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
compose_file="$script_dir/compose.yaml"
hook_source="$script_dir/auto_remove.sh"
container="${QBITTORRENT_CONTAINER:-qbittorrent}"
api_base="http://localhost:8081/api/v2"
autorun_program='/config/auto_remove.sh "%F" "%I" "%D"'

usage() {
  cat <<'EOF'
Usage: ./docker/qbittorrent/configure.sh --check|--apply

  --check  只读检查容器挂载、完成钩子与 qBittorrent 偏好
  --apply  备份 qBittorrent.conf，关闭 .torrent 导出并启用完成钩子
EOF
}

die() {
  printf 'qBittorrent configure: %s\n' "$1" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "missing command: $1"
}

query_preferences() {
  docker exec "$container" \
    curl -fsS --max-time 15 "$api_base/app/preferences"
}

verify_state() {
  local preferences mount_json mount_source expected_source

  preferences="$(query_preferences)"
  jq -e --arg program "$autorun_program" '
    .export_dir == ""
    and .export_dir_fin == ""
    and .autorun_enabled == true
    and .autorun_program == $program
  ' <<<"$preferences" >/dev/null \
    || die 'qBittorrent preferences do not match the repository policy'

  mount_json="$(docker inspect "$container" | jq -c '
    [.[0].Mounts[] | select(.Destination == "/config/auto_remove.sh")][0]
  ')"
  jq -e '.Type == "bind" and .RW == false' <<<"$mount_json" >/dev/null \
    || die 'completion hook is not mounted read-only from the repository'

  mount_source="$(jq -r '.Source' <<<"$mount_json")"
  expected_source="$(cd -- "$(dirname -- "$hook_source")" && pwd -P)/$(basename -- "$hook_source")"
  [[ "$mount_source" == "$expected_source" ]] \
    || die 'completion hook mount points to a different host file'

  docker exec "$container" test -x /config/auto_remove.sh \
    || die 'completion hook is not executable inside the container'

  printf 'qBittorrent configuration verified: exports disabled, hook enabled, read-only mount active.\n'
}

[[ $# -eq 1 ]] || {
  usage >&2
  exit 2
}

case "$1" in
  --check|--apply)
    mode=$1
    ;;
  -h|--help)
    usage
    exit 0
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac

require_command docker
require_command jq

[[ -f "$compose_file" ]] || die 'compose.yaml is missing'
[[ -x "$hook_source" ]] || die 'repository completion hook is not executable'
[[ "$(docker inspect --format '{{.State.Running}}' "$container" 2>/dev/null || true)" == true ]] \
  || die "container is not running: $container"

if [[ "$mode" == --apply ]]; then
  timestamp="$(date '+%Y%m%d-%H%M%S')"
  docker exec "$container" sh -c '
    source=/config/qBittorrent/config/qBittorrent.conf
    stamp=$1
    if [ -f "$source" ]; then
      cp -p "$source" "${source}.before-configure-${stamp}"
    fi
  ' sh "$timestamp"

  preference_json="$(jq -cn --arg program "$autorun_program" '{
    export_dir: "",
    export_dir_fin: "",
    autorun_enabled: true,
    autorun_program: $program
  }')"

  docker exec "$container" \
    curl -fsS --max-time 15 -X POST \
      --data-urlencode "json=$preference_json" \
      "$api_base/app/setPreferences" >/dev/null
fi

verify_state
