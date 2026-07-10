#!/usr/bin/env bash
set -euo pipefail

# 这个固定 helper 只服务 runner deadline 行为测试；它不能接受任意命令或仓库内路径。
if [[ $# -ne 1 ]]; then
  exit 64
fi
readonly MARKER_PATH="$1"
readonly MARKER_ROOT="${MARKER_PATH%/*}"
readonly MARKER_SUFFIX="${MARKER_ROOT#/tmp/yamc-runner-contract.}"
if [[ "${MARKER_PATH}" != "${MARKER_ROOT}/helper.pid" || "${MARKER_ROOT}" == "${MARKER_PATH}" || \
      "${MARKER_ROOT}" != /tmp/yamc-runner-contract.* || ! "${MARKER_SUFFIX}" =~ ^[[:alnum:]]+$ || \
      ! -d "${MARKER_ROOT}" || -L "${MARKER_ROOT}" ]]; then
  exit 64
fi

cleanup_marker() {
  trap - HUP INT TERM EXIT
  /bin/rm -f -- "${MARKER_PATH}"
}

handle_signal() {
  local exit_code="$1"
  cleanup_marker
  exit "${exit_code}"
}

trap cleanup_marker EXIT
trap 'handle_signal 129' HUP
trap 'handle_signal 130' INT
trap 'handle_signal 143' TERM

# PID marker 让测试在 watchdog 返回后同时验证 helper 已退出且 marker 已清除。
umask 077
printf '%s\n' "$$" >"${MARKER_PATH}"
/bin/sleep 30
