---
phase: 01-safety-privacy-and-state-foundation
status: stabilized
source_review: 01-REVIEW.md
findings_in_scope: 5
closed: 5
skipped: 0
review_iteration: null
created: 2026-07-11
---

# Phase 01 Exceptional Safety Stabilization

这是一份针对 `01-REVIEW.md` 最终 1 个 Critical 与 4 个 Warning 的例外稳定化记录，不是 iteration 4 code review，也不改写原 review 的历史结论。五项 finding 全部进入 controlled RED，随后按三个独立安全边界修复并 GREEN；没有跳过、降级或用现有一般回归代替专项证明。

## 提交

1. `e288599 test(01): expose residual safety races`
   - 只加入 deterministic race seams、专项 RED 和 structural canary。
   - 覆盖 Store 单 writer/delete/rollback、Git run-wide snapshot、owner execute、逐组件路径替换，以及 runner nested-body/single-supervisor contract。
2. `a9c1039 fix(01): make artifact storage append only`
   - 关闭 CR-01；确立 fresh capability-owned single writer、existing store read-only reopen 与 append-only artifact model。
3. `4e8a201 fix(01): freeze rooted repository inputs`
   - 同时关闭 WR-02、WR-03、WR-04；确立 owner `0100` mode、run-wide frozen Git view 与 rooted component walk。
4. `76a5bfd fix(01): keep runner under one supervisor`
   - 关闭 WR-01；public invocation 只保留一个 supervisor/PGID，聚合改用 fixed internal dispatcher。

全部提交均为英文 atomic commit，只 stage 计划内精确文件；每次提交前均完成 cached diff、targeted privacy 与 staged Gitleaks。没有 push。

## CR-01 — Closed by capability-owned append-only Store

### Controlled RED

`TestStoreStabilizationContract` 在旧实现上证明：

- 同一 store pathname 可以重新取得 mutable writer；
- lifecycle `Delete` 会物理删除 immutable object；
- publish post-link failure 的按名 rollback 能删除并发放入的 replacement；
- source 仍保留 publish cleanup、graph rollback 与 lifecycle delete 的 pathname unlink primitive。

这些是目标 finding 本身，不是 setup、toolchain 或 unrelated test failure。

### 修复模型

- mutable `Store` 只能独占创建一个此前不存在的 fresh root，并发布 private random capability marker；caller 预建目录或第二个 writer 都不能获得 mutation authority。
- Store lifetime 保留 parent → root → capability marker → `sha256` / `transitions` 的完整 handle 与 identity chain。existing store 只能 read-only reopen、校验和读取。
- object、transition 与 staging 都是 append-only。publish 使用 unpredictable staging、bounded write、`fsync` 与 no-replace hard link；成功或失败都不按 pathname 删除 staging/final entry。
- graph failure 只回滚内存可见性，不 unlink 已发布 object；`Store.Delete` 始终拒绝物理删除。snapshot expiry 只改变未 pin artifact 的读取资格。
- retained `os.Root` 的 authority 明确绑定最初创建的 fresh capability inode，而不是持续声称 pathname 永不移动。named chain 漂移会使 operation non-zero；即使 write 已在原 capability 内开始，也不会删除 replacement，亦不会把任意 existing directory 变成 cleanup target。
- 唯一物理回收仍是主 verdict 冻结之后，对整棵 marker-owned fresh fixture 做 ownership/UID/nonce/TTL/containment revalidation 后 teardown；artifact Store 自身没有逐文件清理 authority。

### GREEN

- fresh first writer accepted；second writer 与 caller-precreated root rejected；
- expiry 后 `Delete` rejected，原 object identity 保持；
- post-link replacement 在 injected failure 后保持 byte-identical；
- source canary 证明没有 Store pathname rollback/delete primitive；
- `artifact-kinds`、`artifact-lineage`、`privacy-boundary`、`bounded-capture`、`fixture-lifecycle` 与完整 phase 回归通过。

## WR-01 — Closed by one public supervisor

### Controlled RED

旧 wave/phase 会递归执行 public `test.sh task|wave`，形成 outer body PGID → inner watcher → inner body PGID。专项 structural/behavioral RED 要求在 nested body 已建立后由最外 deadline 收敛整个树；旧实现仍有 public recursion，且没有 `nested-body` orphan proof。

### 修复模型

- 每次 public `test.sh` invocation 只由最外层 Perl watchdog fork 一次受监控 body，并且脚本中只有一次 `setpgrp(0, 0)`；没有 descendant `setpgrp` 或 `setsid`。
- wave/phase 改用 closed `run_embedded_task_body` / `run_embedded_wave_body` dispatcher；public argv/env 没有 internal mode 或 bypass 入口，也不再递归执行 public script。
- 每个 internal child 仍建立自己的 fresh external root、HOME/XDG、Go cache 与 manager roots，但 Bash subshell 保持在同一个 outer PGID。
- 外层 supervisor 在 deadline 时 TERM/KILL 整个 group，并在 normal body exit 后清理 surviving descendants；所有组合层优先传播唯一 deadline envelope 与退出码 `124`。

### GREEN

`nested-body` test 在 inner body 与 blocking helper 都已发布不同 PID 后，以 800 ms 外层预算触发 deadline，并证明：

- 输出严格等于一个 `runner-deadline-exceeded` envelope；
- exit code 严格为 `124`；
- body/helper PID 均变为 `ESRCH`；
- helper/body marker 与本次所有 `/tmp/yamc-safety.*` root 均消失；
- forged、stale 与 self-consistent inherited PID/FD/nonce 仍不能关闭 watchdog。

focused runner tasks、`phase-integration` 与完整 phase 全部通过。

## WR-02 — Closed by Git owner-execute semantics

### Controlled RED

旧 reader 使用 `mode.Perm() & 0111`。因此 HEAD/index `100644` + worktree `0655` 被错误拒绝，而 HEAD/index `100755` + worktree `0655` 被错误接受；这与 Git 只依据 owner execute bit 的 mapping 不一致。

### 修复与 GREEN

所有 tracked worktree mode mapping 统一为 `mode.Perm() & 0100`：只有 owner execute 产生 `100755`，否则产生 `100644`。专项 Git fixture 证明：

- HEAD/index `100644` + worktree `0655` 按 `100644` 接受；
- HEAD/index `100755` + worktree `0655` 作为 owner-execute removal 拒绝；
- 原有双向 chmod-only drift、byte/index substitution、untracked、ignored 与 symlink negatives 保持 GREEN。

## WR-03 — Closed by one frozen HEAD/index view

### Controlled RED

旧 `validateTrackedInput` 为每个 input 单独读取当前 HEAD；在第一个 input 后 checkout 到第二个合法 commit，可以把 commit A 与 commit B 的 inputs 混入同一次 workflow。

### 修复模型与 GREEN

- `openTrackedRepository` 在第一个 input 前只捕获一次 validated HEAD commit OID。
- 同时只执行一次 bounded full `git ls-files -z --stage`，解析为唯一 stage-0 map，并保存 exact index bytes。
- 所有 input 的 tree/blob/index checks 都绑定该同一 HEAD OID 与 frozen map；不再逐 input 读取 current HEAD/index entry。
- workflow 返回前重新核对 repository root identity、current HEAD 与完整 index bytes；任何 drift 都使整个 workflow fail closed。
- A/B commit scheduling test 证明第二个 commit 的 input 不能混入已打开的 repository view。

## WR-04 — Closed by rooted component-by-component reads

### Controlled RED

旧流程在 `EvalSymlinks` 完成后按绝对 pathname 重新打开文件；若在两步之间把 intermediate directory 替换成指向仓库外 byte-identical file 的 symlink，final-component `O_NOFOLLOW` 仍会消费外部 inode。

### 修复模型

- repository root 通过 `Lstat` + retained `os.Root` + `SameFile` 绑定初始 identity。
- relative path 的每个 intermediate component 都执行 `Lstat`（拒绝 symlink、要求 directory）→ `OpenRoot` → opened/named `SameFile` recheck，并保留整条 directory-handle chain 到读取结束。
- final file 先 `Lstat` 为 bounded regular file，再用 `O_NOFOLLOW | O_NONBLOCK` 打开，核对 before/opened/named identity；bounded `limit+1` read 后再次核对 identity、mode、size 与 mtime。
- 返回前再次复核每层 named/opened directory binding 与 repository root binding。

### GREEN

专项 tests 全部 bounded reject：single intermediate symlink swap、chained symlink、byte-identical directory replacement、byte-identical final-file replacement，以及 final FIFO replacement；canonical tracked file 继续通过。

## 最终独立回归

### 14 个固定 task

全部 exit `0` 并输出 `synthetic-sentinel-passed`：

`walking-skeleton`、`artifact-kinds`、`artifact-lineage`、`privacy-boundary`、`bounded-capture`、`fixture-lifecycle`、`tier-network-policy`、`sentinel-manifest`、`sentinel-verdicts`、`real-sentinel-envelope`、`controlplane-contract`、`no-destructive-defaults`、`phase-e2e`、`docs-and-phase-gate`。

### 7 个固定 wave

全部 exit `0` 并输出 `synthetic-sentinel-passed`：

`skeleton`、`artifact-contracts`、`privacy`、`fixture-policy`、`sentinels`、`controlplane`、`phase-integration`。

### 完整 phase 与静态门

- `./safety/scripts/test.sh phase`：exit `0`，`{"status":"synthetic-sentinel-passed","suite":"phase"}`。
- `/bin/bash -n safety/scripts/test.sh`：通过。
- `gofmt -d` over all `safety/**/*.go`：无 diff。
- fresh external HOME/cache/tmp、`GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`、`GOENV=off`、`GOWORK=off` 下 `go vet ./...`：通过。
- `gitleaks detect --no-git --source safety --redact --no-banner`：扫描约 786.22 KB，零泄漏。
- `safety/AGENTS.md -> CLAUDE.md` exact relative symlink：通过。
- `git diff --check -- safety README.md`：通过。
- `sentinel-manifest` 与完整 phase 的 manifest/digest gate：通过；本次没有修改 manifest-bound sentinel implementation/test sources，因此不需要改 manifest JSON digest。
- 结束后没有 `/tmp/yamc-safety.*`、`/tmp/yamc-runner-contract.*` 或本次 vet/tracked-test temp root 残留。

## 安全边界与未改变事项

- 全程没有读取 `.config/alma/`，也没有 stage 用户已有的根 `CLAUDE.md`、`.ai/` 或 `.config/alma/` 状态。
- 没有运行 current-host snapshot、真实 `launchctl`、HOME/manager/service adapter、网络、Nix、Homebrew、mise、uv、rustup、install、activation、switch、update 或真实环境 cleanup。
- `launchctl print` 的 tracked isolated negative proof 仍有意缺失；current-host 路径继续在任何 adapter/workload 前返回 `manual-required` / `32` / `indeterminate` / zero-call。
- 本报告只记录本次例外稳定化事实，保持未提交，供主线程与后续审查使用。
