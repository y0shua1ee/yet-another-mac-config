---
phase: 01-safety-privacy-and-state-foundation
reviewed: 2026-07-11T02:06:54Z
depth: standard
files_reviewed: 17
files_reviewed_list:
  - README.md
  - safety/CLAUDE.md
  - safety/README.md
  - safety/internal/artifact/store.go
  - safety/internal/artifact/store_fs.go
  - safety/internal/artifact/store_stabilization_test.go
  - safety/internal/artifact/validate_test.go
  - safety/internal/e2e/artifact_cli_test.go
  - safety/internal/e2e/no_cleanup_cli_test.go
  - safety/internal/e2e/phase_e2e_test.go
  - safety/internal/e2e/real_sentinel_cli_test.go
  - safety/internal/e2e/walking_skeleton_test.go
  - safety/internal/fixture/fixture_test.go
  - safety/internal/fixture/root.go
  - safety/internal/workflow/synthetic.go
  - safety/internal/workflow/tracked_snapshot_test.go
  - safety/scripts/test.sh
findings:
  critical: 0
  warning: 1
  info: 0
  total: 1
status: issues_found
---

# Phase 01: Exceptional Safety Stabilization Acceptance Review

**Reviewed:** 2026-07-11T02:06:54Z
**Depth:** standard
**Files Reviewed:** 17
**Status:** issues_found

## Review Scope

本轮是有意收窄的稳定化验收，不是 iteration 4，也没有重新展开 Phase 01 的全部攻击面。范围固定为：上一份 `01-REVIEW.md` 的 1 个 Critical 与 4 个 Warning、`01-STABILIZATION.md` 的关闭声明，以及 `e288599`、`a9c1039`、`4e8a201`、`76a5bfd` 直接引入的回归。

审查亲自核对四个 commit 的完整 diff、当前 HEAD 源码、对应 deterministic seams、五组专项测试、完整 phase 汇总、最终 integration wave、Bash/Go 静态门与 safety 树隐私扫描。没有读取 `.config/alma/`，没有运行 current-host snapshot、真实 `launchctl`、真实 HOME/manager/service adapter、网络、Nix、Homebrew、mise、uv、rustup、安装、激活、switch、update 或真实环境 cleanup。

tracked controlled-service proof 继续有意保持缺失。current-host 路径仍在任何 adapter/workload 调用前返回 `manual-required` / `32` / `indeterminate` / zero-call；这是已确认的安全边界，不是 finding。

## Acceptance Result

上一份报告的五项 finding 均已关闭：

1. **原 CR-01 — closed：** mutable Store 只能独占创建 fresh capability root；existing pathname 只能只读 reopen。object、transition 与 staging 均 append-only；没有 publish cleanup、graph rollback、lifecycle delete 或按 digest/name unlink。namespace drift 会 non-zero，但不会删除 replacement。
2. **原 WR-01 — closed：** public invocation 只建立一个 supervisor/PGID；wave/phase 不递归 public `test.sh`，源码只有一个 `setpgrp(0, 0)` 且无 `setsid`。nested-body deadline 后 body/helper 都达到 `ESRCH`，marker/root 清理完成，输出严格只有一个 deadline envelope，退出码严格为 `124`。
3. **原 WR-02 — closed：** worktree Git mode 只由 owner execute `0100` 映射；`100644 + 0655` 接受为 `100644`，`100755 + 0655` 作为 owner-execute removal 拒绝。
4. **原 WR-03 — closed：** 每个 workflow 只捕获一次 HEAD OID 与 bounded full stage-0 index snapshot；全部输入共享该 view，并在结束前复核 HEAD、完整 index bytes 与 root identity。
5. **原 WR-04 — closed：** tracked input 从 retained repository-root handle 开始逐组件 no-follow/rooted 读取；intermediate symlink、chained symlink、directory replacement、final-file replacement 与 FIFO replacement 均有界拒绝。

四项修复代码本身没有重新打开原先的 Store replacement-delete、runner orphan、Git owner-mode、cross-input commit mixing 或 intermediate symlink consumption 问题。不过，`76a5bfd` 的 single-supervisor 改造直接引入了下面一个新的时间边界 Warning。

## Critical Issues

None.

## Warnings

### WR-01: single-supervisor 聚合丢失 nested task/wave 的独立 hard deadline

**Files:** `safety/scripts/test.sh:371-412`, `safety/scripts/test.sh:998-1084`, `safety/scripts/test.sh:1087-1172`

**Issue:** public invocation 的唯一 watchdog 只根据最外层 argv 选择预算：task 为 15 秒、wave 为 47 秒、phase 为 305 秒。`run_wave_child` 在启动前只检查 wave 的剩余时间是否至少 15 秒，随后同步调用 `run_embedded_task_body`；embedded task 自身没有 15 秒 timer、deadline context 或向 outer supervisor 报告 child deadline 的受信通道。`run_phase_wave_child` 与 `run_phase_task_child` 同样只检查 phase 的全局剩余时间，再同步进入 embedded wave/task；embedded wave 内的 `run_wave_child` 还继续读取 phase 级 `RUNNER_BUDGET_SECONDS=305`，因此既没有 component-wave 的 47 秒 hard stop，也没有 nested task 的 15 秒 hard stop。

这不是原 WR-01 的 orphan 复发：最外层 deadline 仍能终止同一 PGID 中的所有 body/helper，并保持唯一 envelope + exit `124`。问题是 `76a5bfd` 在移除 recursive public watchdog 时，同时移除了原本由 child invocation 提供的局部 deadline，却仍在 `01-VALIDATION.md`、PLAN、README 与 local guidance 中声明每个 task child 15 秒、每个 component wave 47 秒。

**Deterministic evidence:** 在仓库外 marker root 中，以 `YAMC_RUNNER_TEST_BLOCK=nested-body` 让 `wave artifact-contracts` 的第一个 embedded task 阻塞，并把 outer test-only wave budget 设为 20 秒。命令严格返回 exit `124` 和唯一 `runner-deadline-exceeded` envelope，但实测在 **20 秒**才返回，而不是在 nested task 的 15 秒 ceiling 到期时返回。现有 800 ms nested-body test 只证明 outer watchdog/PGID 收敛，不区分 outer wave deadline 与 child task deadline，因此无法捕捉该回归。

**Impact:** 一个在 wave 中阻塞的 task 可持续到最外 wave 的 47 秒；一个在 full phase 内阻塞的 component wave/task 可持续到 phase 的 305 秒。外层隔离、无 orphan 和 current-host zero-call 仍成立，所以本项不是 Critical；但 15/47/305 的分层 Nyquist feedback ceiling 与资源边界不再真实，文档和测试会给维护者错误保证。

**Fix:** 保留唯一 outer supervisor/PGID，但让它同时执行受信的 active-child deadline：task 超过 15 秒或 component wave 超过 47 秒时，终止整次 public invocation 的同一 PGID并返回同一个 bounded envelope + `124`，无需恢复 recursive public `test.sh` 或建立 descendant PGID。新增 deterministic regression 时，outer budget 必须显著大于 child budget，并分别证明 nested task 在 15 秒 ceiling、phase component wave 在 47 秒 ceiling 停止；同时继续断言 body/helper `ESRCH`、唯一 envelope、marker-owned roots 全部清理。

## Verification Evidence

### Five focused acceptance tests

全部在 fresh external HOME/XDG/TMP/cache 下，以 `GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`、`GOENV=off`、`GOWORK=off`、`CGO_ENABLED=0` 运行并通过：

```text
go test ./internal/artifact -run '^TestStoreStabilizationContract$'
  -> PASS

go test ./internal/workflow -run '^TestTrackedRepositorySnapshot$'
  -> PASS

go test ./internal/e2e -run '^TestWalkingSkeletonContract$'
  -> PASS

go test ./internal/e2e -run '^TestPhaseE2E$'
  -> PASS

go test ./internal/e2e -run '^TestRealSentinelCLI$'
  -> PASS
```

这些专项分别覆盖 fresh single-writer/append-only/replacement preservation、one frozen HEAD+index/rooted component reads、owner-`0100` Git semantics、single-PGID nested-body ESRCH/unique-124，以及 current-host proof-missing exit-32/zero-call。

### Aggregate stabilization evidence

```text
./safety/scripts/test.sh phase
  -> exit 0 in 77s
  -> {"status":"synthetic-sentinel-passed","suite":"phase"}

./safety/scripts/test.sh wave phase-integration
  -> exit 0 in 9s
  -> {"status":"synthetic-sentinel-passed","suite":"phase-integration"}
```

两条命令合计重新覆盖稳定化报告记录的 14 个固定 task、7 个固定 wave 与独立 full phase 聚合；这证明当前正常路径无一般功能回归，但不消除上面的 nested child deadline 反例。

### Static, privacy, and commit checks

```text
/bin/bash -n safety/scripts/test.sh
  -> PASS

gofmt -d <all safety/**/*.go>
  -> no diff

isolated offline go vet ./...
  -> PASS

gitleaks detect --no-git --source safety --redact --no-banner
  -> scanned approximately 786.22 KB; no leaks found

test -L safety/AGENTS.md && test "$(readlink safety/AGENTS.md)" = CLAUDE.md
  -> PASS

git show --check e288599 a9c1039 4e8a201 76a5bfd
git diff --check e288599^..76a5bfd -- safety README.md
  -> PASS
```

## Boundaries Preserved

- Store 物理回收仍只随 marker-owned whole-fixture teardown 发生；artifact Store 没有逐文件 unlink 权限。
- Runner 的 prior orphan/bypass finding 已关闭；新 Warning 仅涉及 nested budget 的触发时间，不涉及额外 PGID、残留 helper 或 live capability。
- Git proof 仍绑定 single frozen HEAD/index、exact tracked blob、owner-execute mode 与 rooted consumed bytes。
- `launchctl print` isolated negative proof仍缺失；current-host 继续 `manual-required` / `32` / `indeterminate` / zero-call。
- passing offline phase 只证明 isolated synthetic/proof-double contracts，不证明当前 Mac、whole Mac、recovery readiness、multi-host consistency 或 fresh installation。

---

_Reviewer: gsd-code-reviewer_
_Depth: standard, fixed stabilization-acceptance scope_
