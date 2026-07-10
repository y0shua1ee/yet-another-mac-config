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

run_with_deadline() {
  local seconds="$1"
  shift
  /usr/bin/perl -MPOSIX=':sys_wait_h' -e '
    use strict;
    use warnings;
    my $seconds = shift @ARGV;
    my $pid = fork();
    exit 70 unless defined $pid;
    if ($pid == 0) {
      exit 70 unless setpgrp(0, 0);
      exec @ARGV;
      exit 127;
    }
    my $timed_out = 0;
    my $signal_exit = 0;
    my $stop_group = sub {
      kill "TERM", -$pid;
      kill "TERM", $pid;
      select undef, undef, undef, 0.2;
      kill "KILL", -$pid;
      kill "KILL", $pid;
    };
    my $stop_descendants = sub {
      return unless kill 0, -$pid;
      kill "TERM", -$pid;
      select undef, undef, undef, 0.2;
      kill "KILL", -$pid if kill 0, -$pid;
    };
    $SIG{ALRM} = sub { $timed_out = 1; $stop_group->(); };
    $SIG{HUP} = sub { $signal_exit = 129; $stop_group->(); };
    $SIG{INT} = sub { $signal_exit = 130; $stop_group->(); };
    $SIG{TERM} = sub { $signal_exit = 143; $stop_group->(); };
    alarm $seconds;
    waitpid($pid, 0);
    my $status = $?;
    alarm 0;
    $stop_descendants->();
    exit 124 if $timed_out;
    exit $signal_exit if $signal_exit;
    exit WEXITSTATUS($status) if WIFEXITED($status);
    exit 128 + WTERMSIG($status) if WIFSIGNALED($status);
    exit 70;
  ' "${seconds}" "$@"
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

# task 与 wave 各自共享一个 wall deadline；fresh cache 编译、list、测试及子 runner 都计入同一预算。
readonly RUNNER_STARTED_SECONDS="${SECONDS}"
if [[ "${SCOPE}" == 'task' ]]; then
  readonly RUNNER_BUDGET_SECONDS=15
elif [[ "${SCOPE}" == 'wave' ]]; then
  readonly RUNNER_BUDGET_SECONDS=47
else
  readonly RUNNER_BUDGET_SECONDS=305
fi

run_with_runner_deadline() {
  local elapsed=$((SECONDS - RUNNER_STARTED_SECONDS))
  local remaining=$((RUNNER_BUDGET_SECONDS - elapsed))
  if [[ "${remaining}" -le 0 ]]; then
    return 124
  fi
  run_with_deadline "${remaining}" "$@"
}

print_runner_deadline() {
  local suite_name="$1"
  printf '{"status":"harness-error","reason":"runner-deadline-exceeded","suite":"%s"}\n' "${suite_name}" >&2
}

unsupported_suite() {
  printf '%s\n' '{"status":"harness-error","reason":"unsupported-suite"}' >&2
  exit 64
}

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

handle_test_signal() {
  local exit_code="$1"
  trap - HUP INT TERM
  cleanup_test_root
  exit "${exit_code}"
}

trap cleanup_test_root EXIT
trap 'handle_test_signal 129' HUP
trap 'handle_test_signal 130' INT
trap 'handle_test_signal 143' TERM

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

run_wave_child() {
  local suite_name="$1"
  local output=''
  local child_status=0
  local elapsed=$((SECONDS - RUNNER_STARTED_SECONDS))
  local remaining=$((RUNNER_BUDGET_SECONDS - elapsed))

  if [[ "${remaining}" -lt 15 ]]; then
    print_runner_deadline "${suite_name}"
    return 124
  fi

  # 子 task 自己拥有 hard deadline；wave 不再叠加新的 process group，避免超时后遗留孙进程。
  output="$(/bin/bash "${SCRIPT_DIR}/test.sh" task "${suite_name}" 2>&1)" || child_status=$?
  # 任何已观察到的 deadline 都必须先原样传播，不能被输出上限改写成其他状态。
  if [[ "${child_status}" -eq 124 ]]; then
    print_runner_deadline "${suite_name}"
    return 124
  fi
  if (( ${#output} > 65536 )); then
    printf '%s\n' '{"status":"harness-error","reason":"bounded-output-exceeded"}' >&2
    return 70
  fi
  if [[ "${child_status}" -ne 0 ]]; then
    printf '{"status":"harness-error","reason":"wave-child-failed","suite":"%s"}\n' "${suite_name}" >&2
    return 1
  fi
  if [[ "${output}" != "{\"status\":\"synthetic-sentinel-passed\",\"suite\":\"${suite_name}\"}" ]]; then
    printf '{"status":"harness-error","reason":"wave-child-output-invalid","suite":"%s"}\n' "${suite_name}" >&2
    return 70
  fi

  elapsed=$((SECONDS - RUNNER_STARTED_SECONDS))
  if [[ "${elapsed}" -ge "${RUNNER_BUDGET_SECONDS}" ]]; then
    print_runner_deadline "${suite_name}"
    return 124
  fi
}

run_go_suite() {
  local package_path="$1"
  local test_pattern="$2"
  local output=''
  local status=0

  output="$(cd -- "${SAFETY_ROOT}" && run_with_runner_deadline "${OFFLINE_ENV[@]}" "${GO_BIN}" test -count=1 -timeout=30s -run "${test_pattern}" "${package_path}" 2>&1)" || status=$?

  # deadline 的退出码优先级高于输出上限，确保所有组合层都能精确传播 124。
  if [[ "${status}" -eq 124 ]]; then
    TEST_OUTPUT=''
    TEST_STATUS=124
    return 124
  fi

  if (( ${#output} > 65536 )); then
    printf '%s\n' '{"status":"harness-error","reason":"bounded-output-exceeded"}' >&2
    TEST_STATUS=70
    return 70
  fi

  TEST_OUTPUT="${output}"
  TEST_STATUS="${status}"
  return "${status}"
}

run_compiled_go_suite() {
  local package_directory="$1"
  local test_binary="$2"
  local test_pattern="$3"
  local output=''
  local status=0

  output="$(cd -- "${package_directory}" && run_with_runner_deadline "${OFFLINE_ENV[@]}" "${test_binary}" -test.count=1 -test.timeout=30s -test.run "${test_pattern}" 2>&1)" || status=$?

  # deadline 必须在输出上限之前传播，避免大输出把 124 改写为 70。
  if [[ "${status}" -eq 124 ]]; then
    TEST_OUTPUT=''
    TEST_STATUS=124
    return 124
  fi
  if (( ${#output} > 65536 )); then
    printf '%s\n' '{"status":"harness-error","reason":"bounded-output-exceeded"}' >&2
    TEST_STATUS=70
    return 70
  fi

  TEST_OUTPUT="${output}"
  TEST_STATUS="${status}"
  return "${status}"
}

run_exact_go_suite() {
  local package_path="$1"
  local test_pattern="$2"
  local test_name="$3"
  local suite_name="$4"
  local red_marker="$5"
  local package_directory=''
  local test_binary=''
  local build_output=''
  local build_status=0
  local listing=''
  local listing_status=0
  local selected=0

  if [[ "${package_path}" != ./* || "${suite_name}" =~ [^a-z0-9-] ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"test-selection-failed"}' >&2
    return 70
  fi
  package_directory="${SAFETY_ROOT}/${package_path#./}"
  test_binary="${ISOLATED_TMP}/${suite_name}.test"

  # 同一 exact suite 只编译一次，再用同一个二进制完成 list 与 behavior，保留 fresh cache 同时给 15 秒 task 留出稳定裕量。
  build_output="$(cd -- "${SAFETY_ROOT}" && run_with_runner_deadline "${OFFLINE_ENV[@]}" "${GO_BIN}" test -c -o "${test_binary}" "${package_path}" 2>&1)" || build_status=$?
  if [[ "${build_status}" -eq 124 ]]; then
    print_runner_deadline "${suite_name}"
    return 124
  fi
  if (( ${#build_output} > 65536 )); then
    printf '%s\n' '{"status":"harness-error","reason":"bounded-output-exceeded"}' >&2
    return 70
  fi
  if [[ "${build_status}" -ne 0 || ! -x "${test_binary}" ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"test-build-failed"}' >&2
    return 70
  fi

  listing="$(cd -- "${package_directory}" && run_with_runner_deadline "${OFFLINE_ENV[@]}" "${test_binary}" -test.timeout=30s -test.list "${test_pattern}" 2>&1)" || listing_status=$?
  if [[ "${listing_status}" -eq 124 ]]; then
    print_runner_deadline "${suite_name}"
    return 124
  fi
  if [[ "${listing_status}" -ne 0 ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"test-selection-failed"}' >&2
    return 70
  fi
  if (( ${#listing} > 65536 )); then
    printf '%s\n' '{"status":"harness-error","reason":"bounded-output-exceeded"}' >&2
    return 70
  fi
  selected="$(printf '%s\n' "${listing}" | /usr/bin/grep -Ec "^${test_name}$" || true)"
  if [[ "${selected}" -ne 1 ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"test-selection-not-exact"}' >&2
    return 70
  fi

  if run_compiled_go_suite "${package_directory}" "${test_binary}" "${test_pattern}"; then
    printf '{"status":"synthetic-sentinel-passed","suite":"%s"}\n' "${suite_name}"
    return 0
  fi
  if [[ "${TEST_STATUS:-0}" -eq 124 ]]; then
    print_runner_deadline "${suite_name}"
    return 124
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
  if [[ "${TEST_STATUS:-0}" -eq 124 ]]; then
    print_runner_deadline 'walking-skeleton'
    return 124
  fi
  printf '%s\n' '{"status":"harness-error","reason":"walking-skeleton-contract-failed"}' >&2
  return 1
}

run_red_walking_skeleton() {
  if run_go_suite './internal/e2e' '^TestWalkingSkeletonContract$'; then
    printf '%s\n' '{"status":"harness-error","reason":"red-contract-unexpectedly-passed"}' >&2
    return 1
  fi

  if [[ "${TEST_STATUS:-0}" -eq 124 ]]; then
    print_runner_deadline 'walking-skeleton-red'
    return 124
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
  run_wave_child artifact-kinds
  run_wave_child artifact-lineage
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

  if [[ "${privacy_status}" -eq 124 || "${e2e_status}" -eq 124 ]]; then
    print_runner_deadline 'bounded-capture'
    return 124
  fi
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
  run_wave_child privacy-boundary
  run_wave_child bounded-capture
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

  if [[ "${fixture_status}" -eq 124 || "${e2e_status}" -eq 124 ]]; then
    print_runner_deadline 'tier-network-policy'
    return 124
  fi
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
  run_wave_child fixture-lifecycle
  run_wave_child tier-network-policy
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"fixture-policy"}'
}

run_sentinel_manifest() {
  run_exact_go_suite \
    './internal/sentinel' \
    '^TestSentinelManifest$' \
    'TestSentinelManifest' \
    'sentinel-manifest' \
    'EXPECTED_RED: sentinel-manifest-behavior-missing'
}

run_sentinel_verdicts() {
  local sentinel_status=0
  local e2e_status=0
  local sentinel_output=''
  local e2e_output=''

  run_exact_go_suite \
    './internal/sentinel' \
    '^TestSentinelVerdicts$' \
    'TestSentinelVerdicts' \
    'sentinel-verdicts-unit' \
    'EXPECTED_RED: sentinel-verdicts-behavior-missing' >/dev/null 2>&1 || sentinel_status=$?
  sentinel_output="${TEST_OUTPUT:-}"

  run_exact_go_suite \
    './internal/e2e' \
    '^TestSentinelCLI$' \
    'TestSentinelCLI' \
    'sentinel-verdicts-e2e' \
    'EXPECTED_RED: sentinel-verdicts-behavior-missing' >/dev/null 2>&1 || e2e_status=$?
  e2e_output="${TEST_OUTPUT:-}"

  if [[ "${sentinel_status}" -eq 124 || "${e2e_status}" -eq 124 ]]; then
    print_runner_deadline 'sentinel-verdicts'
    return 124
  fi
  if [[ "${sentinel_status}" -eq 70 || "${e2e_status}" -eq 70 ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"sentinel-verdicts-selection-failed"}' >&2
    return 70
  fi
  if [[ "${sentinel_status}" -ne 0 && "${sentinel_output}" != *'EXPECTED_RED: sentinel-verdicts-behavior-missing'* ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"sentinel-verdicts-unit-contract-failed"}' >&2
    return 1
  fi
  if [[ "${e2e_status}" -ne 0 && "${e2e_output}" != *'EXPECTED_RED: sentinel-verdicts-behavior-missing'* ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"sentinel-verdicts-e2e-contract-failed"}' >&2
    return 1
  fi
  if [[ "${sentinel_status}" -ne 0 || "${e2e_status}" -ne 0 ]]; then
    printf '%s\n' '{"status":"expected-red-observed","suite":"sentinel-verdicts"}' >&2
    return 1
  fi
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"sentinel-verdicts"}'
}

run_real_sentinel_envelope() {
  local sentinel_status=0
  local e2e_status=0
  local sentinel_output=''
  local e2e_output=''

  run_exact_go_suite \
    './internal/sentinel' \
    '^TestRealSentinelEnvelope$' \
    'TestRealSentinelEnvelope' \
    'real-sentinel-envelope-unit' \
    'EXPECTED_RED: real-sentinel-envelope-behavior-missing' >/dev/null 2>&1 || sentinel_status=$?
  sentinel_output="${TEST_OUTPUT:-}"

  run_exact_go_suite \
    './internal/e2e' \
    '^TestRealSentinelCLI$' \
    'TestRealSentinelCLI' \
    'real-sentinel-envelope-e2e' \
    'EXPECTED_RED: real-sentinel-envelope-behavior-missing' >/dev/null 2>&1 || e2e_status=$?
  e2e_output="${TEST_OUTPUT:-}"

  if [[ "${sentinel_status}" -eq 124 || "${e2e_status}" -eq 124 ]]; then
    print_runner_deadline 'real-sentinel-envelope'
    return 124
  fi
  if [[ "${sentinel_status}" -eq 70 || "${e2e_status}" -eq 70 ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"real-sentinel-envelope-selection-failed"}' >&2
    return 70
  fi
  if [[ "${sentinel_status}" -ne 0 && "${sentinel_output}" != *'EXPECTED_RED: real-sentinel-envelope-behavior-missing'* ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"real-sentinel-envelope-unit-contract-failed"}' >&2
    return 1
  fi
  if [[ "${e2e_status}" -ne 0 && "${e2e_output}" != *'EXPECTED_RED: real-sentinel-envelope-behavior-missing'* ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"real-sentinel-envelope-e2e-contract-failed"}' >&2
    return 1
  fi
  if [[ "${sentinel_status}" -ne 0 || "${e2e_status}" -ne 0 ]]; then
    printf '%s\n' '{"status":"expected-red-observed","suite":"real-sentinel-envelope"}' >&2
    return 1
  fi
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"real-sentinel-envelope"}'
}

run_sentinels_wave() {
  # 三个已完成 handler 各自启动新的子 runner，外部根与 HMAC key 不复用。
  run_wave_child sentinel-manifest
  run_wave_child sentinel-verdicts
  run_wave_child real-sentinel-envelope
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"sentinels"}'
}

run_controlplane_contract() {
  local contract_status=0
  local e2e_status=0
  local contract_output=''
  local e2e_output=''

  run_exact_go_suite \
    './internal/contract' \
    '^TestControlPlaneContract$' \
    'TestControlPlaneContract' \
    'controlplane-contract-unit' \
    'EXPECTED_RED: controlplane-ownership-behavior-missing' >/dev/null 2>&1 || contract_status=$?
  contract_output="${TEST_OUTPUT:-}"

  run_exact_go_suite \
    './internal/e2e' \
    '^TestControlPlaneCLI$' \
    'TestControlPlaneCLI' \
    'controlplane-contract-e2e' \
    'EXPECTED_RED: controlplane-ownership-behavior-missing' >/dev/null 2>&1 || e2e_status=$?
  e2e_output="${TEST_OUTPUT:-}"

  if [[ "${contract_status}" -eq 124 || "${e2e_status}" -eq 124 ]]; then
    print_runner_deadline 'controlplane-contract'
    return 124
  fi
  if [[ "${contract_status}" -eq 70 || "${e2e_status}" -eq 70 ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"controlplane-contract-selection-failed"}' >&2
    return 70
  fi
  if [[ "${contract_status}" -ne 0 && "${contract_output}" != *'EXPECTED_RED: controlplane-ownership-behavior-missing'* ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"controlplane-contract-unit-failed"}' >&2
    return 1
  fi
  if [[ "${e2e_status}" -ne 0 && "${e2e_output}" != *'EXPECTED_RED: controlplane-ownership-behavior-missing'* ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"controlplane-contract-e2e-failed"}' >&2
    return 1
  fi
  if [[ "${contract_status}" -ne 0 || "${e2e_status}" -ne 0 ]]; then
    printf '%s\n' '{"status":"expected-red-observed","suite":"controlplane-contract"}' >&2
    return 1
  fi
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"controlplane-contract"}'
}

run_no_destructive_defaults() {
  local contract_status=0
  local e2e_status=0
  local contract_output=''
  local e2e_output=''

  run_exact_go_suite \
    './internal/contract' \
    '^TestNoDestructiveDefaults$' \
    'TestNoDestructiveDefaults' \
    'no-destructive-defaults-unit' \
    'EXPECTED_RED: destructive-policy-behavior-missing' >/dev/null 2>&1 || contract_status=$?
  contract_output="${TEST_OUTPUT:-}"

  run_exact_go_suite \
    './internal/e2e' \
    '^TestNoCleanupCLI$' \
    'TestNoCleanupCLI' \
    'no-destructive-defaults-e2e' \
    'EXPECTED_RED: destructive-policy-behavior-missing' >/dev/null 2>&1 || e2e_status=$?
  e2e_output="${TEST_OUTPUT:-}"

  if [[ "${contract_status}" -eq 124 || "${e2e_status}" -eq 124 ]]; then
    print_runner_deadline 'no-destructive-defaults'
    return 124
  fi
  if [[ "${contract_status}" -eq 70 || "${e2e_status}" -eq 70 ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"no-destructive-defaults-selection-failed"}' >&2
    return 70
  fi
  if [[ "${contract_status}" -ne 0 && "${contract_output}" != *'EXPECTED_RED: destructive-policy-behavior-missing'* ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"no-destructive-defaults-unit-failed"}' >&2
    return 1
  fi
  if [[ "${e2e_status}" -ne 0 && "${e2e_output}" != *'EXPECTED_RED: destructive-policy-behavior-missing'* ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"no-destructive-defaults-e2e-failed"}' >&2
    return 1
  fi
  if [[ "${contract_status}" -ne 0 || "${e2e_status}" -ne 0 ]]; then
    printf '%s\n' '{"status":"expected-red-observed","suite":"no-destructive-defaults"}' >&2
    return 1
  fi
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"no-destructive-defaults"}'
}

run_controlplane_wave() {
  # 两个已完成 handler 分别使用新的外部根，不共享 fixture、store 或 cache。
  run_wave_child controlplane-contract
  run_wave_child no-destructive-defaults
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"controlplane"}'
}

run_phase_e2e() {
  run_exact_go_suite \
    './internal/e2e' \
    '^TestPhaseE2E$' \
    'TestPhaseE2E' \
    'phase-e2e' \
    'EXPECTED_RED: phase-integration-behavior-missing'
}

docs_gate_error() {
  local reason="$1"
  printf '{"status":"harness-error","reason":"%s"}\n' "${reason}" >&2
}

run_docs_and_phase_gate() {
  # 文档 gate 只检查仓库拥有的固定路径与固定文字，不接受调用方提供路径、命令或匹配式。
  if [[ ! -f "${SAFETY_ROOT}/README.md" || -L "${SAFETY_ROOT}/README.md" ]]; then
    docs_gate_error 'safety-readme-invalid'
    return 1
  fi
  if [[ ! -f "${SAFETY_ROOT}/CLAUDE.md" || -L "${SAFETY_ROOT}/CLAUDE.md" ]]; then
    docs_gate_error 'safety-guidance-invalid'
    return 1
  fi
  if [[ ! -L "${SAFETY_ROOT}/AGENTS.md" || "$(/usr/bin/readlink "${SAFETY_ROOT}/AGENTS.md")" != 'CLAUDE.md' ]]; then
    docs_gate_error 'safety-agents-symlink-invalid'
    return 1
  fi
  if [[ ! -f "${SAFETY_ROOT}/../README.md" || -L "${SAFETY_ROOT}/../README.md" ]]; then
    docs_gate_error 'root-readme-invalid'
    return 1
  fi

  /usr/bin/grep -Fq 'covered-surfaces-unchanged-for-run' "${SAFETY_ROOT}/README.md" || {
    docs_gate_error 'safety-readme-claim-boundary-missing'
    return 1
  }
  /usr/bin/grep -Fq '15 / 47 / 305' "${SAFETY_ROOT}/README.md" || {
    docs_gate_error 'safety-readme-deadlines-missing'
    return 1
  }
  /usr/bin/grep -Fq './safety/scripts/test.sh task docs-and-phase-gate' "${SAFETY_ROOT}/CLAUDE.md" || {
    docs_gate_error 'safety-guidance-test-contract-missing'
    return 1
  }
  /usr/bin/grep -Fq '禁止执行真实激活、安装、更新或清理命令' "${SAFETY_ROOT}/CLAUDE.md" || {
    docs_gate_error 'safety-guidance-live-boundary-missing'
    return 1
  }
  /usr/bin/grep -Fq '| `safety/` |' "${SAFETY_ROOT}/../README.md" || {
    docs_gate_error 'root-readme-table-entry-missing'
    return 1
  }
  /usr/bin/grep -Fq './safety/scripts/test.sh phase' "${SAFETY_ROOT}/../README.md" || {
    docs_gate_error 'root-readme-phase-command-missing'
    return 1
  }
  /usr/bin/grep -Fq '仓库外' "${SAFETY_ROOT}/../README.md" || {
    docs_gate_error 'root-readme-local-state-boundary-missing'
    return 1
  }

  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"docs-and-phase-gate"}'
}

run_phase_integration_wave() {
  # 最终 wave 只串行聚合两个 Phase 7 task；完整 phase gate 必须由操作者另行运行。
  run_wave_child phase-e2e
  run_wave_child docs-and-phase-gate
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"phase-integration"}'
}

run_phase_wave_child() {
  local suite_name="$1"
  local expected_suite="${suite_name}"
  local output=''
  local child_status=0
  local elapsed=$((SECONDS - RUNNER_STARTED_SECONDS))
  local remaining=$((RUNNER_BUDGET_SECONDS - elapsed))

  if [[ "${suite_name}" == 'skeleton' ]]; then
    expected_suite='walking-skeleton'
  fi

  if [[ "${remaining}" -lt 47 ]]; then
    print_runner_deadline "${suite_name}"
    return 124
  fi

  # 组件 wave 自己拥有 47 秒预算；phase 只做启动前保留与完成后校验。
  output="$(/bin/bash "${SCRIPT_DIR}/test.sh" wave "${suite_name}" 2>&1)" || child_status=$?
  if [[ "${child_status}" -eq 124 ]]; then
    print_runner_deadline "${suite_name}"
    return 124
  fi
  if (( ${#output} > 65536 )); then
    printf '%s\n' '{"status":"harness-error","reason":"bounded-output-exceeded"}' >&2
    return 70
  fi
  if [[ "${child_status}" -ne 0 ]]; then
    printf '{"status":"harness-error","reason":"phase-child-failed","suite":"%s"}\n' "${suite_name}" >&2
    return 1
  fi
  if [[ "${output}" != "{\"status\":\"synthetic-sentinel-passed\",\"suite\":\"${expected_suite}\"}" ]]; then
    printf '{"status":"harness-error","reason":"phase-child-output-invalid","suite":"%s"}\n' "${suite_name}" >&2
    return 70
  fi

  elapsed=$((SECONDS - RUNNER_STARTED_SECONDS))
  if [[ "${elapsed}" -ge "${RUNNER_BUDGET_SECONDS}" ]]; then
    print_runner_deadline "${suite_name}"
    return 124
  fi
}

run_phase_task_child() {
  local suite_name="$1"
  local output=''
  local child_status=0
  local elapsed=$((SECONDS - RUNNER_STARTED_SECONDS))
  local remaining=$((RUNNER_BUDGET_SECONDS - elapsed))

  if [[ "${remaining}" -lt 15 ]]; then
    print_runner_deadline "${suite_name}"
    return 124
  fi

  # 最终 task 自己拥有 15 秒 hard deadline；phase 不叠加 process group。
  output="$(/bin/bash "${SCRIPT_DIR}/test.sh" task "${suite_name}" 2>&1)" || child_status=$?
  if [[ "${child_status}" -eq 124 ]]; then
    print_runner_deadline "${suite_name}"
    return 124
  fi
  if (( ${#output} > 65536 )); then
    printf '%s\n' '{"status":"harness-error","reason":"bounded-output-exceeded"}' >&2
    return 70
  fi
  if [[ "${child_status}" -ne 0 ]]; then
    printf '{"status":"harness-error","reason":"phase-child-failed","suite":"%s"}\n' "${suite_name}" >&2
    return 1
  fi
  if [[ "${output}" != "{\"status\":\"synthetic-sentinel-passed\",\"suite\":\"${suite_name}\"}" ]]; then
    printf '{"status":"harness-error","reason":"phase-child-output-invalid","suite":"%s"}\n' "${suite_name}" >&2
    return 70
  fi

  elapsed=$((SECONDS - RUNNER_STARTED_SECONDS))
  if [[ "${elapsed}" -ge "${RUNNER_BUDGET_SECONDS}" ]]; then
    print_runner_deadline "${suite_name}"
    return 124
  fi
}

run_phase_gate() {
  # 六个固定组件 wave 后只运行 phase-e2e；完整预算为 6*47+15+8=305 秒。
  run_phase_wave_child skeleton
  run_phase_wave_child artifact-contracts
  run_phase_wave_child privacy
  run_phase_wave_child fixture-policy
  run_phase_wave_child sentinels
  run_phase_wave_child controlplane
  run_phase_task_child phase-e2e
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"phase"}'
}

if [[ "${SCOPE}" != 'phase' ]]; then
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
  task:sentinel-manifest)
    run_sentinel_manifest
    ;;
  task:sentinel-verdicts)
    run_sentinel_verdicts
    ;;
  task:real-sentinel-envelope)
    run_real_sentinel_envelope
    ;;
  task:controlplane-contract)
    run_controlplane_contract
    ;;
  task:no-destructive-defaults)
    run_no_destructive_defaults
    ;;
  task:phase-e2e)
    run_phase_e2e
    ;;
  task:docs-and-phase-gate)
    run_docs_and_phase_gate
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
  wave:sentinels)
    run_sentinels_wave
    ;;
  wave:controlplane)
    run_controlplane_wave
    ;;
  wave:phase-integration)
    run_phase_integration_wave
    ;;
  *)
    unsupported_suite
    ;;
esac
exit 0
fi

case "${SCOPE}:${SUITE}" in
  phase:phase)
    run_phase_gate
    ;;
  *)
    unsupported_suite
    ;;
esac
