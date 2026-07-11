---
phase: 01-safety-privacy-and-state-foundation
reviewed: 2026-07-11T02:42:51Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - README.md
  - safety/CLAUDE.md
  - safety/README.md
  - safety/internal/e2e/phase_e2e_test.go
  - safety/scripts/test.sh
findings:
  critical: 0
  warning: 0
  info: 0
  total: 0
status: clean
---

# Phase 01: Layered Runner Deadline Follow-up Acceptance Review

**Reviewed:** 2026-07-11T02:42:51Z
**Depth:** standard
**Files Reviewed:** 5
**Status:** clean

## Review Scope

本轮只复验 `ca1d334` 对上一份报告唯一 Warning 的修复，不是新的全阶段攻击面审查。范围固定为私有 AF_UNIX 控制协议、monotonic deadline stack、15 / 47 / 305 秒分层预算、single-supervisor / single-PGID 约束，以及该 commit 新增的 deterministic deadline seams、phase E2E、integration、静态与隐私回归。

上一轮已经接受的 exceptional stabilization 结论继续成立：原先的 **1 个 Critical + 4 个 Warning** 已由 `e288599`、`a9c1039`、`4e8a201` 与 `76a5bfd` 关闭；本轮没有重新展开 Store replacement-delete、Git snapshot/rooted read 或 current-host adapter 的全部攻击面，只确认 `ca1d334` 没有直接重新打开这些边界。

没有读取 `.config/alma/`，也没有运行 current-host snapshot、真实 `launchctl`、真实 HOME/manager/service adapter、网络、Nix、Homebrew、mise、uv、rustup、安装、激活、switch、update 或真实环境 cleanup。所有动态验证仅使用 runner 创建的仓库外临时根与隔离 cache。

## Acceptance Result

上一份报告的唯一 follow-up Warning 已关闭，且本轮没有发现新的 Critical、Warning 或 Info finding。

### Prior WR-01 — closed: nested task/wave hard deadlines restored

`ca1d334` 在不恢复 recursive public runner、descendant supervisor 或额外 process group 的前提下，把 nested deadline authority 放回唯一 outer supervisor：

- supervisor 在 fork 前只创建一次匿名 AF_UNIX `socketpair`，随机 token 不从 public env/argv 输入；公开环境中没有可选择的 deadline FD、token、re-exec 或 bypass mode；
- fixed body 只能同步发送 `BEGIN` / `END`，消息不携带 duration；task 的 15 秒和 component wave 的 47 秒预算由 supervisor 端固定选择；
- test-only task/wave override 先受 production ceiling 校验，只能缩短到至少 500 ms，不能延长或关闭 deadline；
- deadline 使用 `CLOCK_MONOTONIC` 并按栈管理。`BEGIN` 若不能把 deadline 严格放在当前 parent deadline 内，会直接进入唯一 timeout 路径；`END` 必须匹配栈顶 kind/id 并保持严格 LIFO；
- active frame 或残留 partial control line 遇到 EOF 时会 fail closed 为 protocol error、终止同一 PGID 并返回 `70`。base frame 失去控制通道后，固定聚合器的下一个同步 `BEGIN` 会因 FD9 exchange 失败而在启动 child 前返回 `70`，而 public deadline 仍持续有效；因此 EOF 不能延长、弹出或关闭 deadline；
- actual embedded task body 在进入测试工作负载前关闭 FD9，不能取得 supervisor 控制能力；embedded wave 只为固定 nested task 聚合保留该通道；
- 当前源码仍只有一个 `setpgrp(0, 0)`，没有 `setsid`，没有 recursive public `test.sh task|wave`，也没有第二个 supervisor。timeout 继续由唯一 authority 终止同一 PGID 中的 body/helper，并只返回一个 bounded envelope 与退出码 `124`。

旧反例已被直接反转：`wave artifact-contracts` 的 outer test budget 设为 20 秒，同时尝试把 nested task test override 扩大到 20 秒，阻塞 `nested-body` 后仍在 **15.062 秒**返回，而不是等待 outer 20 秒；结果严格为 exit `124` 和唯一 `runner-deadline-exceeded` envelope。这同时证明 production 15 秒 ceiling 不能被 caller 的 test-only override 扩大。

## Critical Issues

None.

## Warnings

None.

## Verification Evidence

### New nested deadline seams and aggregate paths

```text
./safety/scripts/test.sh task phase-e2e
  -> exit 0
  -> {"status":"synthetic-sentinel-passed","suite":"phase-e2e"}

./safety/scripts/test.sh wave phase-integration
  -> exit 0
  -> {"status":"synthetic-sentinel-passed","suite":"phase-integration"}
```

`TestPhaseE2E` 内新增的三个 seam 均通过：

- nested task ceiling：outer wave 20 秒、task 800 ms；
- phase component-wave ceiling：outer phase 5 秒、wave 500 ms；
- final phase task ceiling：outer phase 5 秒、wave 2 秒、task 500 ms。

每个 seam 都同时断言 fixed helper 与 embedded body 在 deadline 后达到 `ESRCH`、二者保持同一 PGID、输出只有一个 deadline envelope、退出码为 `124`，并且 marker 与 runner 临时根没有残留。

### Direct regression for the prior Warning

```text
YAMC_RUNNER_TEST_MODE=1 \
YAMC_RUNNER_TEST_BUDGET_MS=20000 \
YAMC_RUNNER_TEST_TASK_BUDGET_MS=20000 \
YAMC_RUNNER_TEST_BLOCK=nested-body \
./safety/scripts/test.sh wave artifact-contracts
  -> exit 124 in 15.062s
  -> {"status":"harness-error","reason":"runner-deadline-exceeded"}
```

marker 使用仓库外 `/tmp/yamc-runner-contract.*` 临时根，命令结束后已清除。20 秒 task override 超过 production 15 秒 ceiling，因而被拒绝；实测时间落在 production nested-task deadline，而不是 outer wave deadline。

### Static and privacy checks

```text
/bin/bash -n safety/scripts/test.sh
  -> PASS

./safety/scripts/test.sh wave privacy
  -> exit 0
  -> {"status":"synthetic-sentinel-passed","suite":"privacy"}

git show --check ca1d334
git diff --check ca1d334^..ca1d334
  -> PASS

single socketpair / single setpgrp / no setsid / no recursive public entry /
no public YAMC_RUNNER_DEADLINE_FD or YAMC_RUNNER_DEADLINE_TOKEN assignment
  -> PASS
```

## Boundaries Preserved

- 原 1 Critical + 4 Warnings 的 stabilization acceptance 保持关闭；`ca1d334` 只修复其后发现的 layered deadline 回归。
- timeout 仍由一个 supervisor、一个 process group 和一个 bounded result authority 收敛；没有新增 live-host、manager、network 或 arbitrary-command capability。
- caller 最多通过显式 test mode 缩短测试预算，不能扩大、关闭或绕过 production budget。
- control protocol 的失败只会提前拒绝或终止执行，不能恢复更宽的 parent deadline 之外的执行窗口。
- passing offline E2E/integration 只证明 isolated synthetic contracts，不证明当前 Mac、whole Mac、recovery readiness、multi-host consistency 或 fresh installation。

---

_Reviewer: gsd-code-reviewer_
_Depth: standard, fixed layered-deadline follow-up scope_
