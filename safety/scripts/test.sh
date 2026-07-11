#!/usr/bin/env bash
set -euo pipefail

# 每个 runner 在解析参数、定位仓库或创建临时根之前先进入唯一 watchdog。
# 测试预算只允许在固定测试模式下缩短，不能扩大生产预算。
runner_watchdog_budget_ms=15000
runner_task_budget_ms=15000
runner_wave_budget_ms=47000
case "${1:-}" in
  wave)
    runner_watchdog_budget_ms=47000
    ;;
  phase)
    runner_watchdog_budget_ms=305000
    ;;
esac
if [[ "${YAMC_RUNNER_TEST_MODE:-}" == '1' && "${YAMC_RUNNER_TEST_BUDGET_MS:-}" =~ ^[1-9][0-9]{2,5}$ ]] && \
   (( YAMC_RUNNER_TEST_BUDGET_MS >= 500 && YAMC_RUNNER_TEST_BUDGET_MS <= runner_watchdog_budget_ms )); then
  runner_watchdog_budget_ms="${YAMC_RUNNER_TEST_BUDGET_MS}"
fi
if [[ "${YAMC_RUNNER_TEST_MODE:-}" == '1' && "${YAMC_RUNNER_TEST_TASK_BUDGET_MS:-}" =~ ^[1-9][0-9]{2,4}$ ]] && \
   (( YAMC_RUNNER_TEST_TASK_BUDGET_MS >= 500 && YAMC_RUNNER_TEST_TASK_BUDGET_MS <= runner_task_budget_ms )); then
  runner_task_budget_ms="${YAMC_RUNNER_TEST_TASK_BUDGET_MS}"
fi
if [[ "${YAMC_RUNNER_TEST_MODE:-}" == '1' && "${YAMC_RUNNER_TEST_WAVE_BUDGET_MS:-}" =~ ^[1-9][0-9]{2,4}$ ]] && \
   (( YAMC_RUNNER_TEST_WAVE_BUDGET_MS >= 500 && YAMC_RUNNER_TEST_WAVE_BUDGET_MS <= runner_wave_budget_ms )); then
  runner_wave_budget_ms="${YAMC_RUNNER_TEST_WAVE_BUDGET_MS}"
fi

readonly runner_caller_path="${PATH:-/usr/bin:/bin}"
readonly runner_test_mode="${YAMC_RUNNER_TEST_MODE:-}"
readonly runner_test_block_point="${YAMC_RUNNER_TEST_BLOCK:-}"
readonly runner_test_marker="${YAMC_RUNNER_TEST_MARKER:-}"

# public script 不接受任何 caller-provided re-exec guard；每次调用都无条件建立唯一 supervisor。
# supervisor 通过 fork 前创建的匿名 socket 管理固定 nested deadline，public argv/env 无法伪造或延长。
exec /usr/bin/env -i \
  'PATH=/usr/bin:/bin' \
  'HOME=/var/empty' \
  'LC_ALL=C' \
  'LANG=C' \
  /usr/bin/perl -MPOSIX=':sys_wait_h' -MTime::HiRes='clock_gettime,CLOCK_MONOTONIC,sleep' -MIO::Select -MSocket='AF_UNIX,SOCK_STREAM,PF_UNSPEC' -MFcntl='F_SETFD' -MErrno='EINTR' -e '
    use strict;
    use warnings;
    my $budget_ms = shift @ARGV;
    my $task_budget_ms = shift @ARGV;
    my $wave_budget_ms = shift @ARGV;
    my $script = shift @ARGV;
    my $caller_path = shift @ARGV;
    my $test_mode = shift @ARGV;
    my $test_block = shift @ARGV;
    my $test_marker = shift @ARGV;
    exit 70 unless $budget_ms =~ /^[1-9][0-9]{2,5}$/ && $budget_ms >= 500 && $budget_ms <= 305000;
    exit 70 unless $task_budget_ms =~ /^[1-9][0-9]{2,4}$/ && $task_budget_ms >= 500 && $task_budget_ms <= 15000;
    exit 70 unless $wave_budget_ms =~ /^[1-9][0-9]{2,4}$/ && $wave_budget_ms >= 500 && $wave_budget_ms <= 47000;
    my $grace_for = sub { return $_[0] <= 2000 ? 0.20 : 0.50; };
    my $now = sub { return clock_gettime(CLOCK_MONOTONIC); };
    my $public_deadline = $now->() + ($budget_ms / 1000.0);
    socketpair(my $control_parent, my $control_child, AF_UNIX, SOCK_STREAM, PF_UNSPEC) or exit 70;
    open my $random, "<:raw", "/dev/urandom" or exit 70;
    my $random_bytes = "";
    while (length($random_bytes) < 32) {
      my $count = sysread($random, my $chunk, 32 - length($random_bytes));
      exit 70 unless defined $count && $count > 0;
      $random_bytes .= $chunk;
    }
    close $random;
    my $control_token = unpack("H*", $random_bytes);
    $random_bytes = "\0" x length($random_bytes);
    my $pid = fork();
    exit 70 unless defined $pid;
    if ($pid == 0) {
      close $control_parent;
      exit 70 unless setpgrp(0, 0);
      my $control_fd = fileno($control_child);
      exit 70 unless defined $control_fd;
      if ($control_fd != 9) {
        exit 70 unless POSIX::dup2($control_fd, 9) == 9;
        close $control_child;
      }
      open my $control_keep, ">&=9" or exit 70;
      exit 70 unless defined fcntl($control_keep, F_SETFD, 0);
      open my $source, "<:raw", $script or exit 70;
      my $body = "";
      my $found = 0;
      while (my $line = <$source>) {
        if ($found) {
          $body .= $line;
          exit 70 if length($body) > 262144;
        } elsif ($line eq ": __YAMC_RUNNER_BODY__\n") {
          $found = 1;
        }
      }
      close $source;
      exit 70 unless $found && length($body) > 0;
      %ENV = (
        PATH => $caller_path,
        HOME => "/var/empty",
        LC_ALL => "C",
        LANG => "C",
      );
      if ($test_mode eq "1") {
        $ENV{YAMC_RUNNER_TEST_MODE} = "1";
        $ENV{YAMC_RUNNER_TEST_BLOCK} = $test_block;
        $ENV{YAMC_RUNNER_TEST_MARKER} = $test_marker;
      }
      exec "/bin/bash", "-c", $body, $script, $control_token, @ARGV;
      exit 127;
    }
    close $control_child;
    $SIG{PIPE} = "IGNORE";
    my $timed_out = 0;
    my $signal_exit = 0;
    my $protocol_error = 0;
    my $stopping = 0;
    my $active_grace = $grace_for->($budget_ms);
    my $stop_group = sub {
      return if $stopping;
      $stopping = 1;
      kill "TERM", -$pid;
      kill "TERM", $pid;
      sleep $active_grace;
      kill "KILL", -$pid;
      kill "KILL", $pid;
    };
    my $stop_descendants = sub {
      return unless kill 0, -$pid;
      kill "TERM", -$pid;
      sleep $active_grace;
      kill "KILL", -$pid if kill 0, -$pid;
    };
    $SIG{HUP} = sub { $signal_exit = 129; };
    $SIG{INT} = sub { $signal_exit = 130; };
    $SIG{TERM} = sub { $signal_exit = 143; };
    my @deadline_stack = ({ kind => "public", id => "0", deadline => $public_deadline, budget_ms => $budget_ms });
    my $selector = IO::Select->new($control_parent);
    my $control_buffer = "";
    my $send_ack = sub {
      my ($message) = @_;
      my $offset = 0;
      while ($offset < length($message)) {
        my $written = syswrite($control_parent, $message, length($message) - $offset, $offset);
        return 0 unless defined $written && $written > 0;
        $offset += $written;
      }
      return 1;
    };
    my $process_control_line = sub {
      my ($line) = @_;
      return 0 if length($line) > 192 || $line =~ /[^A-Za-z0-9 :-]/;
      my @parts = split / /, $line;
      return 0 unless @parts == 4;
      my ($action, $token, $kind, $id) = @parts;
      return 0 unless $token eq $control_token && ($kind eq "task" || $kind eq "wave") && $id =~ /^[1-9][0-9]{0,8}$/;
      if ($action eq "BEGIN") {
        my $scope_budget_ms = $kind eq "task" ? $task_budget_ms : $wave_budget_ms;
        my $scope_deadline = $now->() + ($scope_budget_ms / 1000.0);
        if ($scope_deadline > $deadline_stack[-1]{deadline}) {
          $timed_out = 1;
          return 1;
        }
        push @deadline_stack, { kind => $kind, id => $id, deadline => $scope_deadline, budget_ms => $scope_budget_ms };
        $active_grace = $grace_for->($scope_budget_ms);
        return $send_ack->("ACK $control_token BEGIN $kind $id\n");
      }
      if ($action eq "END") {
        return 0 unless @deadline_stack > 1;
        my $active = $deadline_stack[-1];
        return 0 unless $active->{kind} eq $kind && $active->{id} eq $id;
        if ($now->() >= $active->{deadline} - $grace_for->($active->{budget_ms})) {
          $timed_out = 1;
          return 1;
        }
        pop @deadline_stack;
        $active_grace = $grace_for->($deadline_stack[-1]{budget_ms});
        return $send_ack->("ACK $control_token END $kind $id\n");
      }
      return 0;
    };
    my $status = 0;
    while (1) {
      if ($protocol_error) {
        $stop_group->();
        waitpid($pid, 0);
        $status = $?;
        last;
      }
      if ($signal_exit) {
        $stop_group->();
        waitpid($pid, 0);
        $status = $?;
        last;
      }
      my $active = $deadline_stack[-1];
      my $fire_at = $active->{deadline} - $grace_for->($active->{budget_ms});
      if ($now->() >= $fire_at) {
        $timed_out = 1;
        $stop_group->();
        waitpid($pid, 0);
        $status = $?;
        last;
      }
      my $waited = waitpid($pid, WNOHANG);
      if ($waited == $pid) {
        $status = $?;
        $protocol_error = 1 unless @deadline_stack == 1 && length($control_buffer) == 0;
        last;
      }
      if ($waited == -1) {
        $protocol_error = 1;
        last;
      }
      my $wait_for = $fire_at - $now->();
      $wait_for = 0.10 if $wait_for > 0.10;
      $wait_for = 0 if $wait_for < 0;
      my @ready = $selector->can_read($wait_for);
      next unless @ready;
      my $read = sysread($control_parent, my $chunk, 4096);
      if (!defined $read) {
        next if $! == EINTR;
        $protocol_error = 1;
        last;
      }
      if ($read == 0) {
        $selector->remove($control_parent);
        $protocol_error = 1 if @deadline_stack > 1 || length($control_buffer) > 0;
        next;
      }
      $control_buffer .= $chunk;
      if (length($control_buffer) > 4096) {
        $protocol_error = 1;
        last;
      }
      while ($control_buffer =~ s/\A([^\n]*)\n//) {
        unless ($process_control_line->($1)) {
          $protocol_error = 1 unless $timed_out;
          last;
        }
      }
      if ($timed_out || $protocol_error) {
        $stop_group->();
        waitpid($pid, 0);
        $status = $?;
        last;
      }
    }
    $stop_descendants->();
    if ($timed_out) {
      print STDERR qq({"status":"harness-error","reason":"runner-deadline-exceeded"}\n);
      exit 124;
    }
    exit $signal_exit if $signal_exit;
    if ($protocol_error) {
      print STDERR qq({"status":"harness-error","reason":"runner-deadline-protocol-error"}\n);
      exit 70;
    }
    exit WEXITSTATUS($status) if WIFEXITED($status);
    exit 128 + WTERMSIG($status) if WIFSIGNALED($status);
    exit 70;
  ' "${runner_watchdog_budget_ms}" "${runner_task_budget_ms}" "${runner_wave_budget_ms}" "${BASH_SOURCE[0]}" "${runner_caller_path}" "${runner_test_mode}" "${runner_test_block_point}" "${runner_test_marker}" "$@"

: __YAMC_RUNNER_BODY__
set -euo pipefail

# 此入口只运行固定的离线 Go 测试，不接受任意命令或隐式更高权限模式。
readonly SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)"
readonly SAFETY_ROOT="$(CDPATH='' cd -- "${SCRIPT_DIR}/.." && pwd -P)"
if [[ $# -lt 1 || ! "${1}" =~ ^[0-9a-f]{64}$ ]]; then
  exit 70
fi
readonly RUNNER_DEADLINE_TOKEN="$1"
shift
RUNNER_DEADLINE_SEQUENCE=0
RUNNER_DEADLINE_LAST_ID=''

runner_deadline_exchange() {
  local action="$1"
  local kind="$2"
  local deadline_id="$3"
  local response=''

  case "${action}:${kind}" in
    BEGIN:task|BEGIN:wave|END:task|END:wave)
      ;;
    *)
      return 70
      ;;
  esac
  printf '%s %s %s %s\n' "${action}" "${RUNNER_DEADLINE_TOKEN}" "${kind}" "${deadline_id}" >&9 || return 70
  IFS= read -r response <&9 || return 70
  [[ "${response}" == "ACK ${RUNNER_DEADLINE_TOKEN} ${action} ${kind} ${deadline_id}" ]] || return 70
}

runner_deadline_begin() {
  local kind="$1"
  RUNNER_DEADLINE_SEQUENCE=$((RUNNER_DEADLINE_SEQUENCE + 1))
  RUNNER_DEADLINE_LAST_ID="${RUNNER_DEADLINE_SEQUENCE}"
  runner_deadline_exchange BEGIN "${kind}" "${RUNNER_DEADLINE_LAST_ID}"
}

runner_deadline_end() {
  local kind="$1"
  local deadline_id="$2"
  runner_deadline_exchange END "${kind}" "${deadline_id}"
}

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

# task 与 wave 各自共享一个 wall deadline；fresh cache 编译、list、测试及子 runner 都计入同一预算。
readonly RUNNER_STARTED_SECONDS="${SECONDS}"
if [[ "${SCOPE}" == 'task' ]]; then
  readonly RUNNER_BUDGET_SECONDS=15
elif [[ "${SCOPE}" == 'wave' ]]; then
  readonly RUNNER_BUDGET_SECONDS=47
else
  readonly RUNNER_BUDGET_SECONDS=305
fi
readonly RUNNER_DEADLINE_ENVELOPE='{"status":"harness-error","reason":"runner-deadline-exceeded"}'

print_runner_deadline() {
  printf '%s\n' "${RUNNER_DEADLINE_ENVELOPE}" >&2
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

# public body 与每个固定内部 child 都创建自己的临时根。
# 内部 child 只是同一受监控 process group 中的 Bash subshell，不再递归 public entry。
TEST_ROOT=''
TEST_MARKER=''
ISOLATED_HOME=''
XDG_CONFIG_HOME=''
XDG_DATA_HOME=''
XDG_CACHE_HOME=''
XDG_STATE_HOME=''
XDG_RUNTIME_DIR=''
ISOLATED_TMP=''
GOCACHE_ROOT=''
GOMODCACHE_ROOT=''
GOPATH_ROOT=''
MANAGER_ROOT=''
RUNNER_TEST_BODY_MARKER=''
RUNNER_TEST_BODY_PID=''
OFFLINE_ENV=()

cleanup_runner_test_body_marker() {
  local observed_pid=''
  if [[ -z "${RUNNER_TEST_BODY_MARKER:-}" || -z "${RUNNER_TEST_BODY_PID:-}" || \
        "${RUNNER_TEST_BODY_MARKER}" != /tmp/yamc-runner-contract.*/body.pid || \
        -L "${RUNNER_TEST_BODY_MARKER}" || ! -f "${RUNNER_TEST_BODY_MARKER}" ]]; then
    return 0
  fi
  IFS= read -r observed_pid <"${RUNNER_TEST_BODY_MARKER}" || return 0
  if [[ "${observed_pid}" == "${RUNNER_TEST_BODY_PID}" ]]; then
    /bin/rm -f -- "${RUNNER_TEST_BODY_MARKER}"
  fi
}

cleanup_test_root() {
  # 仅删除本次创建且带 marker 的外部临时子目录，任何不确定性都会保留现场。
  if [[ -n "${TEST_ROOT:-}" && "${TEST_ROOT}" == /tmp/yamc-safety.* && ! -L "${TEST_ROOT}" && -f "${TEST_MARKER}" ]]; then
    /bin/rm -rf -- "${TEST_ROOT}"
  fi
  cleanup_runner_test_body_marker
}

handle_test_signal() {
  local exit_code="$1"
  trap - HUP INT TERM
  cleanup_test_root
  exit "${exit_code}"
}

runner_test_block() {
  local block_point="$1"
  local marker_path=''
  local marker_root=''
  local marker_suffix=''
  local body_marker=''
  local block_status=0

  if [[ "${YAMC_RUNNER_TEST_MODE:-}" != '1' || "${YAMC_RUNNER_TEST_BLOCK:-}" != "${block_point}" ]]; then
    return 0
  fi
  marker_path="${YAMC_RUNNER_TEST_MARKER:-}"
  marker_root="${marker_path%/*}"
  marker_suffix="${marker_root#/tmp/yamc-runner-contract.}"
  if [[ "${marker_path}" != "${marker_root}/helper.pid" || "${marker_root}" == "${marker_path}" || \
        "${marker_root}" != /tmp/yamc-runner-contract.* || ! "${marker_suffix}" =~ ^[[:alnum:]]+$ || \
        ! -d "${marker_root}" || -L "${marker_root}" ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"runner-test-marker-invalid"}' >&2
    return 70
  fi
  body_marker="${marker_root}/body.pid"
  if [[ -e "${body_marker}" || -L "${body_marker}" ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"runner-test-marker-invalid"}' >&2
    return 70
  fi
  RUNNER_TEST_BODY_MARKER="${body_marker}"
  # macOS 系统 Bash 没有 BASHPID；固定子 shell 只写入它的直接父 PID，也就是当前 embedded body。
  if ! /bin/sh -c 'umask 077; set -C; printf "%s\n" "$PPID" >"$1"' yamc-runner-body "${RUNNER_TEST_BODY_MARKER}" 2>/dev/null; then
    RUNNER_TEST_BODY_MARKER=''
    RUNNER_TEST_BODY_PID=''
    printf '%s\n' '{"status":"harness-error","reason":"runner-test-marker-invalid"}' >&2
    return 70
  fi
  IFS= read -r RUNNER_TEST_BODY_PID <"${RUNNER_TEST_BODY_MARKER}" || RUNNER_TEST_BODY_PID=''
  if [[ ! "${RUNNER_TEST_BODY_PID}" =~ ^[1-9][0-9]*$ ]]; then
    printf '%s\n' '{"status":"harness-error","reason":"runner-test-marker-invalid"}' >&2
    return 70
  fi
  { /bin/bash "${SAFETY_ROOT}/testdata/runner/block-helper.sh" "${marker_path}" 9>&-; } >/dev/null 2>&1 || block_status=$?
  cleanup_runner_test_body_marker
  return "${block_status}"
}

initialize_test_context() {
  # 每次 context 都使用新的系统临时根；真实 HOME、XDG、缓存和管理器状态不会被继承。
  TEST_ROOT="$(/usr/bin/mktemp -d '/tmp/yamc-safety.XXXXXXXX')"
  TEST_MARKER="${TEST_ROOT}/.yamc-owned-test-root"
  /usr/bin/touch "${TEST_MARKER}"
  trap cleanup_test_root EXIT
  trap 'handle_test_signal 129' HUP
  trap 'handle_test_signal 130' INT
  trap 'handle_test_signal 143' TERM

  # 固定阻塞点位于 marker-owned root 创建之后、其余 setup 之前，用来证明 watchdog 也覆盖清理。
  runner_test_block setup

  ISOLATED_HOME="${TEST_ROOT}/home"
  XDG_CONFIG_HOME="${TEST_ROOT}/xdg/config"
  XDG_DATA_HOME="${TEST_ROOT}/xdg/data"
  XDG_CACHE_HOME="${TEST_ROOT}/xdg/cache"
  XDG_STATE_HOME="${TEST_ROOT}/xdg/state"
  XDG_RUNTIME_DIR="${TEST_ROOT}/xdg/runtime"
  ISOLATED_TMP="${TEST_ROOT}/tmp"
  GOCACHE_ROOT="${TEST_ROOT}/go/build-cache"
  GOMODCACHE_ROOT="${TEST_ROOT}/go/module-cache"
  GOPATH_ROOT="${TEST_ROOT}/go/path"
  MANAGER_ROOT="${TEST_ROOT}/managers"

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
  OFFLINE_ENV=(
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
}

initialize_test_context

run_wave_child() {
  local suite_name="$1"
  local deadline_id=''
  local output=''
  local child_status=0
  local elapsed=$((SECONDS - RUNNER_STARTED_SECONDS))
  local remaining=$((RUNNER_BUDGET_SECONDS - elapsed))

  if [[ "${remaining}" -lt 15 ]]; then
    print_runner_deadline "${suite_name}"
    return 124
  fi

  # 唯一 supervisor 先确认 15 秒私有 deadline，再启动同 PGID 内的 fresh task body。
  runner_deadline_begin task || return 70
  deadline_id="${RUNNER_DEADLINE_LAST_ID}"
  output="$(run_embedded_task_body "${suite_name}" 2>&1)" || child_status=$?
  runner_deadline_end task "${deadline_id}" || return 70
  # 任何已观察到的 deadline 都必须先原样传播，不能被输出上限改写成其他状态。
  if [[ "${child_status}" -eq 124 ]]; then
    if [[ "${output}" != "${RUNNER_DEADLINE_ENVELOPE}" ]]; then
      printf '%s\n' '{"status":"harness-error","reason":"wave-child-deadline-invalid"}' >&2
      return 70
    fi
    printf '%s\n' "${RUNNER_DEADLINE_ENVELOPE}" >&2
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

  output="$(cd -- "${SAFETY_ROOT}" && "${OFFLINE_ENV[@]}" "${GO_BIN}" test -count=1 -timeout=30s -run "${test_pattern}" "${package_path}" 2>&1)" || status=$?

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

  output="$(cd -- "${package_directory}" && "${OFFLINE_ENV[@]}" "${test_binary}" -test.count=1 -test.timeout=30s -test.run "${test_pattern}" 2>&1)" || status=$?

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
  build_output="$(cd -- "${SAFETY_ROOT}" && "${OFFLINE_ENV[@]}" "${GO_BIN}" test -c -o "${test_binary}" "${package_path}" 2>&1)" || build_status=$?
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

  listing="$(cd -- "${package_directory}" && "${OFFLINE_ENV[@]}" "${test_binary}" -test.timeout=30s -test.list "${test_pattern}" 2>&1)" || listing_status=$?
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

run_skeleton_wave() {
  # 单 task wave 也必须经过同一 15 秒子 deadline，不能直接绕过聚合器。
  run_wave_child walking-skeleton
  printf '%s\n' '{"status":"synthetic-sentinel-passed","suite":"walking-skeleton"}'
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
  runner_test_block docs
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
  local deadline_id=''
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

  # phase 与组件 wave 共享唯一 supervisor/PGID，但 supervisor 会先压入 47 秒 hard ceiling。
  runner_deadline_begin wave || return 70
  deadline_id="${RUNNER_DEADLINE_LAST_ID}"
  output="$(run_embedded_wave_body "${suite_name}" 2>&1)" || child_status=$?
  runner_deadline_end wave "${deadline_id}" || return 70
  if [[ "${child_status}" -eq 124 ]]; then
    if [[ "${output}" != "${RUNNER_DEADLINE_ENVELOPE}" ]]; then
      printf '%s\n' '{"status":"harness-error","reason":"phase-child-deadline-invalid"}' >&2
      return 70
    fi
    printf '%s\n' "${RUNNER_DEADLINE_ENVELOPE}" >&2
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
  local deadline_id=''
  local output=''
  local child_status=0
  local elapsed=$((SECONDS - RUNNER_STARTED_SECONDS))
  local remaining=$((RUNNER_BUDGET_SECONDS - elapsed))

  if [[ "${remaining}" -lt 15 ]]; then
    print_runner_deadline "${suite_name}"
    return 124
  fi

  # 最终 task 由同一 supervisor 压入 15 秒 deadline，不产生第二个 watchdog 或 process group。
  runner_deadline_begin task || return 70
  deadline_id="${RUNNER_DEADLINE_LAST_ID}"
  output="$(run_embedded_task_body "${suite_name}" 2>&1)" || child_status=$?
  runner_deadline_end task "${deadline_id}" || return 70
  if [[ "${child_status}" -eq 124 ]]; then
    if [[ "${output}" != "${RUNNER_DEADLINE_ENVELOPE}" ]]; then
      printf '%s\n' '{"status":"harness-error","reason":"phase-child-deadline-invalid"}' >&2
      return 70
    fi
    printf '%s\n' "${RUNNER_DEADLINE_ENVELOPE}" >&2
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

run_embedded_task_body() (
  local suite_name="${1:-}"

  # 这是只能由已闭合聚合器调用的固定内部 task body；public argv/env 没有 internal mode。
  exec 9>&-
  initialize_test_context
  runner_test_block child
  runner_test_block nested-body
  case "${suite_name}" in
    walking-skeleton)
      run_green_walking_skeleton
      ;;
    artifact-kinds)
      run_artifact_kinds
      ;;
    artifact-lineage)
      run_artifact_lineage
      ;;
    privacy-boundary)
      run_privacy_boundary
      ;;
    bounded-capture)
      run_bounded_capture
      ;;
    fixture-lifecycle)
      run_fixture_lifecycle
      ;;
    tier-network-policy)
      run_tier_network_policy
      ;;
    sentinel-manifest)
      run_sentinel_manifest
      ;;
    sentinel-verdicts)
      run_sentinel_verdicts
      ;;
    real-sentinel-envelope)
      run_real_sentinel_envelope
      ;;
    controlplane-contract)
      run_controlplane_contract
      ;;
    no-destructive-defaults)
      run_no_destructive_defaults
      ;;
    phase-e2e)
      run_phase_e2e
      ;;
    docs-and-phase-gate)
      run_docs_and_phase_gate
      ;;
    *)
      printf '%s\n' '{"status":"harness-error","reason":"internal-task-dispatch-rejected"}' >&2
      return 70
      ;;
  esac
)

run_embedded_wave_body() (
  local suite_name="${1:-}"

  # 内部 wave body 与最外 body 共享唯一 PGID；它只创建 fresh context，不建立新 supervisor。
  initialize_test_context
  runner_test_block child
  runner_test_block nested-wave-body
  case "${suite_name}" in
    skeleton)
      run_skeleton_wave
      ;;
    artifact-contracts)
      run_artifact_contracts_wave
      ;;
    privacy)
      run_privacy_wave
      ;;
    fixture-policy)
      run_fixture_policy_wave
      ;;
    sentinels)
      run_sentinels_wave
      ;;
    controlplane)
      run_controlplane_wave
      ;;
    phase-integration)
      run_phase_integration_wave
      ;;
    *)
      printf '%s\n' '{"status":"harness-error","reason":"internal-wave-dispatch-rejected"}' >&2
      return 70
      ;;
  esac
)

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
    run_skeleton_wave
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
