#!/usr/bin/env bash
set -euo pipefail

# 此入口只运行固定的离线 Go 测试，不接受任意命令或隐式更高权限模式。
readonly SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
readonly SAFETY_ROOT="$(CDPATH='' cd -- "${SCRIPT_DIR}/.." && pwd -P)"

usage() {
  printf '%s\n' 'usage: ./safety/scripts/test.sh task <suite> | wave <suite> | phase' >&2
}

manual_required() {
  printf '%s\n' '{"status":"manual-required","reason":"local-go-unavailable"}' >&2
  exit 32
}

if [[ $# -lt 1 ]]; then
  usage
  exit 64
fi

readonly SCOPE="$1"
shift

case "${SCOPE}" in
  task|wave)
    if [[ $# -ne 1 ]]; then
      usage
      exit 64
    fi
    readonly SUITE="$1"
    ;;
  phase)
    if [[ $# -ne 0 ]]; then
      usage
      exit 64
    fi
    readonly SUITE='phase'
    ;;
  *)
    usage
    exit 64
    ;;
esac

if ! command -v go >/dev/null 2>&1; then
  manual_required
fi

readonly GO_BIN="$(command -v go)"
if [[ "${GO_BIN}" == */* ]]; then
  readonly GO_DIR="${GO_BIN%/*}"
else
  manual_required
fi

# 每次运行都使用新的系统临时根；真实 HOME、XDG、缓存和管理器状态不会被继承。
readonly TEST_ROOT="$(/usr/bin/mktemp -d '/tmp/yamc-safety.XXXXXXXX')"
readonly TEST_MARKER="${TEST_ROOT}/.yamc-owned-test-root"
/usr/bin/touch "${TEST_MARKER}"

cleanup_test_root() {
  # 仅删除本次创建且带 marker 的外部临时子目录，任何不确定性都会保留现场。
  if [[ -n "${TEST_ROOT:-}" && "${TEST_ROOT}" == /tmp/yamc-safety.* && ! -L "${TEST_ROOT}" && -f "${TEST_MARKER}" ]]; then
    /bin/rm -rf -- "${TEST_ROOT}"
  fi
}
trap cleanup_test_root EXIT HUP INT TERM

readonly ISOLATED_HOME="${TEST_ROOT}/home"
readonly XDG_CONFIG_HOME="${TEST_ROOT}/xdg/config"
readonly XDG_DATA_HOME="${TEST_ROOT}/xdg/data"
readonly XDG_CACHE_HOME="${TEST_ROOT}/xdg/cache"
readonly XDG_STATE_HOME="${TEST_ROOT}/xdg/state"
readonly XDG_RUNTIME_DIR="${TEST_ROOT}/xdg/runtime"
readonly ISOLATED_TMP="${TEST_ROOT}/tmp"
readonly GOCACHE_ROOT="${TEST_ROOT}/go/build-cache"
readonly GOMODCACHE_ROOT="${TEST_ROOT}/go/module-cache"
readonly GOPATH_ROOT="${TEST_ROOT}/go/path"
readonly MANAGER_ROOT="${TEST_ROOT}/managers"

/bin/mkdir -p \
  "${ISOLATED_HOME}" \
  "${XDG_CONFIG_HOME}" \
  "${XDG_DATA_HOME}" \
  "${XDG_CACHE_HOME}" \
  "${XDG_STATE_HOME}" \
  "${XDG_RUNTIME_DIR}" \
  "${ISOLATED_TMP}" \
  "${GOCACHE_ROOT}" \
  "${GOMODCACHE_ROOT}" \
  "${GOPATH_ROOT}" \
  "${MANAGER_ROOT}/mise/config" \
  "${MANAGER_ROOT}/mise/data" \
  "${MANAGER_ROOT}/mise/cache" \
  "${MANAGER_ROOT}/uv/cache" \
  "${MANAGER_ROOT}/uv/python" \
  "${MANAGER_ROOT}/rustup" \
  "${MANAGER_ROOT}/cargo" \
  "${MANAGER_ROOT}/nix" \
  "${MANAGER_ROOT}/homebrew/cache" \
  "${MANAGER_ROOT}/homebrew/logs"

# 从空环境构造固定 allowlist，显式关闭 Go 自动工具链和依赖网络访问。
readonly -a OFFLINE_ENV=(
  /usr/bin/env -i
  "PATH=${GO_DIR}:/usr/bin:/bin"
  "HOME=${ISOLATED_HOME}"
  "XDG_CONFIG_HOME=${XDG_CONFIG_HOME}"
  "XDG_DATA_HOME=${XDG_DATA_HOME}"
  "XDG_CACHE_HOME=${XDG_CACHE_HOME}"
  "XDG_STATE_HOME=${XDG_STATE_HOME}"
  "XDG_RUNTIME_DIR=${XDG_RUNTIME_DIR}"
  "TMPDIR=${ISOLATED_TMP}"
  "GOTOOLCHAIN=local"
  "GOPROXY=off"
  "GOSUMDB=off"
  "GOENV=off"
  "GOWORK=off"
  "CGO_ENABLED=0"
  "GOCACHE=${GOCACHE_ROOT}"
  "GOMODCACHE=${GOMODCACHE_ROOT}"
  "GOPATH=${GOPATH_ROOT}"
  "MISE_CONFIG_DIR=${MANAGER_ROOT}/mise/config"
  "MISE_DATA_DIR=${MANAGER_ROOT}/mise/data"
  "MISE_CACHE_DIR=${MANAGER_ROOT}/mise/cache"
  "UV_CACHE_DIR=${MANAGER_ROOT}/uv/cache"
  "UV_PYTHON_INSTALL_DIR=${MANAGER_ROOT}/uv/python"
  "RUSTUP_HOME=${MANAGER_ROOT}/rustup"
  "CARGO_HOME=${MANAGER_ROOT}/cargo"
  "NIX_STATE_DIR=${MANAGER_ROOT}/nix"
  "HOMEBREW_CACHE=${MANAGER_ROOT}/homebrew/cache"
  "HOMEBREW_LOGS=${MANAGER_ROOT}/homebrew/logs"
  "YAMC_TEST_EXTERNAL_ROOT=${TEST_ROOT}"
  "YAMC_TEST_TIER=offline-static"
  "YAMC_NETWORK=deny"
)

run_go_suite() {
  local package_path="$1"
  local test_pattern="$2"
  local output=''
  local status=0

  output="$(cd -- "${SAFETY_ROOT}" && "${OFFLINE_ENV[@]}" "${GO_BIN}" test -count=1 -run "${test_pattern}" "${package_path}" 2>&1)" || status=$?

  if (( ${#output} > 65536 )); then
    printf '%s\n' '{"status":"harness-error","reason":"bounded-output-exceeded"}' >&2
    return 70
  fi

  TEST_OUTPUT="${output}"
  return "${status}"
}

run_exact_go_suite() {
  local package_path="$1"
  local test_pattern="$2"
  local test_name="$3"
  local suite_name="$4"
  local red_marker="$5"
  local listing=''
  local selected=0

  listing="$(cd -- "${SAFETY_ROOT}" && "${OFFLINE_ENV[@]}" "${GO_BIN}" test -list "${test_pattern}" "${package_path}" 2>&1)" || {
    printf '%s\n' '{"status":"harness-error","reason":"test-selection-failed"}' >&2
    return 70
  }
  if (( ${#listing} > 65536 )); then
    printf '%s\n' '{"status":"harness-error","reason":"bounded-output-exceeded"}' >&2
    return 70
  fi
  selected="$(printf '%s\n' "${listing}" | /usr/bin/grep -Ec "^${test_name}$" || true)"
  if [[ "${selected}" -ne 1 ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"test-selection-not-exact"}' >&2
    return 70
  fi

  if run_go_suite "${package_path}" "${test_pattern}"; then
    printf '{"status":"synthetic-sentinel-passed","suite":"%s"}\n' "${suite_name}"
    return 0
  fi
  if [[ -n "${red_marker}" && "${TEST_OUTPUT}" == *"${red_marker}"* ]]; then
    printf '{"status":"expected-red-observed","suite":"%s"}\n' "${suite_name}" >&2
    return 1
  fi
  printf '{"status":"harness-error","reason":"%s-contract-failed"}\n' "${suite_name}" >&2
  return 1
}

run_green_walking_skeleton() {
  if run_go_suite './internal/e2e' '^TestWalkingSkeletonContract$'; then
    printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"walking-skeleton"}'
    return 0
  fi
  printf '%s\n' '{"status":"harness-error","reason":"walking-skeleton-contract-failed"}' >&2
  return 1
}

run_red_walking_skeleton() {
  if run_go_suite './internal/e2e' '^TestWalkingSkeletonContract$'; then
    printf '%s\n' '{"status":"harness-error","reason":"red-contract-unexpectedly-passed"}' >&2
    return 1
  fi

  if [[ "${TEST_OUTPUT}" != *'EXPECTED_RED: round-trip-capability-missing'* ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"unexpected-red-cause"}' >&2
    return 1
  fi

  if [[ "${TEST_OUTPUT}" == *'SETUP_FAILURE:'* || "${TEST_OUTPUT}" == *'TOOLCHAIN_FAILURE:'* || "${TEST_OUTPUT}" == *'OVERCLAIM_ACCEPTED:'* ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"unsafe-red-cause"}' >&2
    return 1
  fi

  printf '%s\n' '{"status":"expected-red-observed","reason":"round-trip-capability-missing"}'
}

run_artifact_kinds() {
  run_exact_go_suite \
    './internal/artifact' \
    '^TestArtifactKinds$' \
    'TestArtifactKinds' \
    'artifact-kinds' \
    'EXPECTED_RED: artifact-kind-behavior-missing'
}

run_artifact_lineage() {
  run_exact_go_suite \
    './internal/e2e' \
    '^TestArtifactLineage$' \
    'TestArtifactLineage' \
    'artifact-lineage' \
    'EXPECTED_RED: artifact-lineage-behavior-missing'
}

run_artifact_contracts_wave() {
  # 每个 task 由新的子 runner 建立独立外部根，禁止复用 fixture 或 store。
  /bin/bash "${SCRIPT_DIR}/test.sh" task artifact-kinds >/dev/null
  /bin/bash "${SCRIPT_DIR}/test.sh" task artifact-lineage >/dev/null
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"artifact-contracts"}'
}

run_privacy_boundary() {
  run_exact_go_suite \
    './internal/privacy' \
    '^TestPrivacyBoundary$' \
    'TestPrivacyBoundary' \
    'privacy-boundary' \
    'EXPECTED_RED: privacy-boundary-behavior-missing'
}

run_bounded_capture() {
  local privacy_status=0
  local e2e_status=0
  local privacy_output=''
  local e2e_output=''

  run_exact_go_suite \
    './internal/privacy' \
    '^TestBoundedCapture$' \
    'TestBoundedCapture' \
    'bounded-capture-privacy' \
    'EXPECTED_RED: bounded-capture-behavior-missing' >/dev/null 2>&1 || privacy_status=$?
  privacy_output="${TEST_OUTPUT:-}"

  run_exact_go_suite \
    './internal/e2e' \
    '^TestPrivacyCLI$' \
    'TestPrivacyCLI' \
    'bounded-capture-e2e' \
    'EXPECTED_RED: bounded-capture-behavior-missing' >/dev/null 2>&1 || e2e_status=$?
  e2e_output="${TEST_OUTPUT:-}"

  if [[ "${privacy_status}" -eq 70 || "${e2e_status}" -eq 70 ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"bounded-capture-selection-failed"}' >&2
    return 70
  fi
  if [[ "${privacy_status}" -ne 0 && "${privacy_output}" != *'EXPECTED_RED: bounded-capture-behavior-missing'* ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"bounded-capture-privacy-contract-failed"}' >&2
    return 1
  fi
  if [[ "${e2e_status}" -ne 0 && "${e2e_output}" != *'EXPECTED_RED: bounded-capture-behavior-missing'* ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"bounded-capture-e2e-contract-failed"}' >&2
    return 1
  fi
  if [[ "${privacy_status}" -ne 0 || "${e2e_status}" -ne 0 ]]; then
    printf '%s\n' '{"status":"expected-red-observed","suite":"bounded-capture"}' >&2
    return 1
  fi
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"bounded-capture"}'
}

run_privacy_wave() {
  # 两个已完成 handler 各自启动新的子 runner，绝不复用外部根或 store。
  /bin/bash "${SCRIPT_DIR}/test.sh" task privacy-boundary >/dev/null
  /bin/bash "${SCRIPT_DIR}/test.sh" task bounded-capture >/dev/null
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"privacy"}'
}

run_fixture_lifecycle() {
  run_exact_go_suite \
    './internal/fixture' \
    '^TestFixtureLifecycle$' \
    'TestFixtureLifecycle' \
    'fixture-lifecycle' \
    'EXPECTED_RED: fixture-lifecycle-behavior-missing'
}

run_tier_network_policy() {
  local fixture_status=0
  local e2e_status=0
  local fixture_output=''
  local e2e_output=''

  run_exact_go_suite \
    './internal/fixture' \
    '^TestTierNetworkPolicy$' \
    'TestTierNetworkPolicy' \
    'tier-network-policy-fixture' \
    'EXPECTED_RED: tier-network-policy-behavior-missing' >/dev/null 2>&1 || fixture_status=$?
  fixture_output="${TEST_OUTPUT:-}"

  run_exact_go_suite \
    './internal/e2e' \
    '^TestTierCLI$' \
    'TestTierCLI' \
    'tier-network-policy-e2e' \
    'EXPECTED_RED: tier-network-policy-behavior-missing' >/dev/null 2>&1 || e2e_status=$?
  e2e_output="${TEST_OUTPUT:-}"

  if [[ "${fixture_status}" -eq 70 || "${e2e_status}" -eq 70 ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"tier-network-policy-selection-failed"}' >&2
    return 70
  fi
  if [[ "${fixture_status}" -ne 0 && "${fixture_output}" != *'EXPECTED_RED: tier-network-policy-behavior-missing'* ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"tier-network-policy-fixture-contract-failed"}' >&2
    return 1
  fi
  if [[ "${e2e_status}" -ne 0 && "${e2e_output}" != *'EXPECTED_RED: tier-network-policy-behavior-missing'* ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"tier-network-policy-e2e-contract-failed"}' >&2
    return 1
  fi
  if [[ "${fixture_status}" -ne 0 || "${e2e_status}" -ne 0 ]]; then
    printf '%s\n' '{"status":"expected-red-observed","suite":"tier-network-policy"}' >&2
    return 1
  fi
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"tier-network-policy"}'
}

run_fixture_policy_wave() {
  # 两个已完成 handler 各自启动新的子 runner，保持 tier、fixture 与 cache 互不复用。
  /bin/bash "${SCRIPT_DIR}/test.sh" task fixture-lifecycle >/dev/null
  /bin/bash "${SCRIPT_DIR}/test.sh" task tier-network-policy >/dev/null
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"fixture-policy"}'
}

case "${SCOPE}:${SUITE}" in
  task:walking-skeleton-red)
    run_red_walking_skeleton
    ;;
  task:walking-skeleton)
    run_green_walking_skeleton
    ;;
  task:artifact-kinds)
    run_artifact_kinds
    ;;
  task:artifact-lineage)
    run_artifact_lineage
    ;;
  task:privacy-boundary)
    run_privacy_boundary
    ;;
  task:bounded-capture)
    run_bounded_capture
    ;;
  task:fixture-lifecycle)
    run_fixture_lifecycle
    ;;
  task:tier-network-policy)
    run_tier_network_policy
    ;;
  wave:skeleton)
    run_green_walking_skeleton
    ;;
  wave:artifact-contracts)
    run_artifact_contracts_wave
    ;;
  wave:privacy)
    run_privacy_wave
    ;;
  wave:fixture-policy)
    run_fixture_policy_wave
    ;;
  *)
    printf '%s\n' '{"status":"harness-error","reason":"unsupported-suite"}' >&2
    exit 64
    ;;
esac
