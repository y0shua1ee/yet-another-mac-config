---
phase: 01-safety-privacy-and-state-foundation
reviewed: 2026-07-11T01:09:49Z
depth: standard
files_reviewed: 59
files_reviewed_list:
  - safety/AGENTS.md
  - safety/CLAUDE.md
  - safety/README.md
  - safety/cmd/yamc-safety/main.go
  - safety/go.mod
  - safety/internal/artifact/canonical.go
  - safety/internal/artifact/envelope.go
  - safety/internal/artifact/kinds.go
  - safety/internal/artifact/lineage.go
  - safety/internal/artifact/store.go
  - safety/internal/artifact/store_fs.go
  - safety/internal/artifact/validate_test.go
  - safety/internal/contract/controlplane.go
  - safety/internal/contract/controlplane_test.go
  - safety/internal/contract/policy.go
  - safety/internal/contract/policy_test.go
  - safety/internal/e2e/artifact_cli_test.go
  - safety/internal/e2e/controlplane_cli_test.go
  - safety/internal/e2e/no_cleanup_cli_test.go
  - safety/internal/e2e/phase_e2e_test.go
  - safety/internal/e2e/privacy_cli_test.go
  - safety/internal/e2e/real_sentinel_cli_test.go
  - safety/internal/e2e/sentinel_cli_test.go
  - safety/internal/e2e/tier_cli_test.go
  - safety/internal/e2e/walking_skeleton_test.go
  - safety/internal/fixture/environment.go
  - safety/internal/fixture/fixture_test.go
  - safety/internal/fixture/network.go
  - safety/internal/fixture/network_test.go
  - safety/internal/fixture/retention.go
  - safety/internal/fixture/root.go
  - safety/internal/privacy/capture.go
  - safety/internal/privacy/capture_test.go
  - safety/internal/privacy/gate.go
  - safety/internal/privacy/gate_test.go
  - safety/internal/sentinel/manifest.go
  - safety/internal/sentinel/real.go
  - safety/internal/sentinel/real_test.go
  - safety/internal/sentinel/sentinel_test.go
  - safety/internal/sentinel/snapshot.go
  - safety/internal/sentinel/synthetic.go
  - safety/internal/sentinel/verdict.go
  - safety/internal/sentinel/verdict_test.go
  - safety/internal/workflow/synthetic.go
  - safety/manifests/network-tests.v1.json
  - safety/manifests/offline-suite.v1.json
  - safety/manifests/protected-surfaces.v1.json
  - safety/manifests/real-adapters.v1.json
  - safety/scripts/test.sh
  - safety/testdata/artifacts/kind-cases.json
  - safety/testdata/artifacts/lineage-cases.json
  - safety/testdata/blueprints/walking-skeleton/expected-report.json
  - safety/testdata/blueprints/walking-skeleton/input.json
  - safety/testdata/blueprints/walking-skeleton/protected-surfaces.json
  - safety/testdata/canaries/cases.json
  - safety/testdata/controlplane/cases.json
  - safety/testdata/raw/fake-adapter.json
  - safety/testdata/runner/block-helper.sh
  - README.md
findings:
  critical: 1
  warning: 4
  info: 0
  total: 5
status: issues_found
---

# Phase 01: Final Code Re-Review After Iteration 3 Fixes

**Reviewed:** 2026-07-11T01:09:49Z
**Depth:** standard
**Files Reviewed:** 59
**Status:** issues_found

## Review Scope

本轮在 iteration-3 四个修复 commit 之后重新审查当前 HEAD，不把 `01-REVIEW-FIX.md` 的结论当作代码证明。范围是前一轮 58 个 Phase 01 文件，加上 `b7e5519` 新增的 `safety/internal/artifact/store_fs.go`；重点复核 artifact store rooted filesystem、public runner watchdog、Git tracked-input proof、read-only freshness，并检查这些修复引入的 race、lifecycle 和 evidence-integrity 回归。

没有读取或修改 `.config/alma/`，没有执行 real snapshot、`launchctl`、真实 HOME/manager/service adapter、网络、Nix、Homebrew、mise、uv、rustup、安装、activation、switch、update 或 cleanup。所有行为复现仅使用仓库外临时进程树或临时 Git root，并已清理。tracked service proof 缺失时，current-host 路径继续在任何 adapter/workload 前返回 `manual-required` / `32` / `indeterminate` / zero-call；这是有意的安全边界，不作为 finding。

## Verification

最终串行验证结果：

```text
/bin/bash -n safety/scripts/test.sh
  -> exit 0

gofmt -d <all safety/**/*.go>
  -> no diff

isolated offline go vet ./...
  -> exit 0

14 fixed task routes
  -> 14/14 exit 0 with synthetic-sentinel-passed

7 fixed waves
  -> 7/7 exit 0 with synthetic-sentinel-passed

./safety/scripts/test.sh phase
  -> exit 0 in 73s; {"status":"synthetic-sentinel-passed","suite":"phase"}

gitleaks detect --no-git --source safety --redact --no-banner
  -> no leaks found

test -L safety/AGENTS.md && test "$(readlink safety/AGENTS.md)" = CLAUDE.md
  -> exit 0

git diff --check -- safety README.md
  -> exit 0 before report rewrite
```

专项隔离验证还确认：自洽 caller PID/FD/nonce 已不能关闭 public watchdog；read-only `FreshObserved.State` 修复在 `ValidateLineage`、`Store.WriteGraph` 和 CLI pre-store 三层生效；Git chmod 基线用例通过。GREEN 证明现有 suite 没有一般回归，但没有覆盖下面五条 race/consistency 路径。

## Narrative Findings (AI reviewer)

iteration-3 的四个目标修复中，`2b8f368` 的 read-only freshness 修复完整关闭；`b7e5519` 关闭了预置 `sha256`/`transitions` symlink、object/transition FIFO/symlink、已发生的 child replacement 和 unbounded special-file read；`d4a58f1` 关闭了 caller-controlled watchdog re-exec bypass；`47c84ad` 开始绑定实际 worktree mode。

但 artifact store 仍在“precheck 通过后发生 rename”时先执行 root 外 mutation，再靠 postcheck 报错，而且 rollback/delete 按名字删除而不绑定 exact inode。这直接重新打开 Phase 01 的零 root-outside-write / no-uncertain-delete 阻塞边界。其余四个 warning 是：nested public watchdog 的 process-group tree 仍可留下 orphan；Git executable-mode 映射不符合 Git 的 owner-execute 规则；HEAD 只在每个 input 内临时读取而非 run-wide frozen；tracked input 的 `EvalSymlinks` 与绝对路径重开之间仍有 intermediate-directory substitution 窗口。

## Critical Issues

### CR-01: retained `os.Root` 在 rename-out 后仍会写入 moved store，并可能按名字删除替换对象

**Files:** `safety/internal/artifact/store_fs.go:17-21`, `safety/internal/artifact/store_fs.go:181-201`, `safety/internal/artifact/store_fs.go:262-319`, `safety/internal/artifact/store_fs.go:330-355`, `safety/internal/artifact/store.go:209-225`

**Issue:** `storeDirectory` 长期保留 `sha256`/`transitions` 的 child `*os.Root`，但较高层 `rootHandle` 在初始化返回时关闭。安装中的 Go 1.26 `os.Root` 契约明确说明：目录被移动后，Root 继续引用原目录在其新位置。`verifyStoreDirectory` 只在离散时间点以 pathname identity 检查 containment；`publishStoreFile` 在第 275 行 precheck 后才创建 temp、写入、`Sync`、hard-link 最终 digest，直到第 316 行才再次检查。

因此在 precheck 后把 `root/sha256`、`root/transitions` 或整个 store root rename 到所选 root 外，retained handle 仍会在 moved directory 中创建 `.pending-*` 和最终 digest/transition。postcheck 只能在 mutation 已发生后返回错误；若进程在 defer 之前终止或 cleanup 失败，文件会持久留在所选 root 外。原 finding 的低门槛 pre-existing symlink 已关闭，但同一“零 root 外写入”保证仍可被 namespace race 打破。

删除路径还不具备 exact-object authority：publish cleanup 在第 286-289、307-311 行按 filename 盲删；`removeStoreFile` 在 `Lstat(name)` 与 `Remove(name)` 之间没有将待删 entry 绑定到同一 inode；graph rollback 的 `removeCreatedObjectFile` 连 directory revalidation 都没有。并发方若先 rename-away 本次对象、再放入同名替换对象，错误路径会删除替换对象并忽略 cleanup error。目录在 precheck 后移出时，删除也会先发生在 moved directory，之后才 postcheck 失败。

**Impact:** 这同时违反 `safety/README.md` 与 `safety/CLAUDE.md` 的 race fail-closed、exact direct-child identity、零 selected-root 外写入和不确定时零删除承诺。它可能把 canonical object/transition 写到 caller 没有选择的目录，或在 rollback/lifecycle delete 中删除不是当前 operation 创建/验证的对象；对 Phase 01 唯一允许的 mutable local-state boundary 属于阻塞问题。

**Fix:** 不能只增加另一次 pathname pre/post check。先明确并强制 mutable store 的 namespace-ownership 模型：推荐仅允许当前 process 创建并独占的 fresh/private root，无法证明独占时拒绝 mutation。Store lifetime 应保留完整 parent→root→child authority，但也要处理 retained directory 被移出原 pathname 的语义。publish/rollback/delete 必须记录并验证本次 temp/published object 的 exact identity；删除采用 ownership-bound quarantine/rename protocol，任何 identity 或 namespace 不确定性都保留现场并 non-zero，不能按可预测 digest name 盲删。加入同步 race seam，覆盖 child/root rename-out、link 后 postcheck failure、temp/final entry replacement、graph rollback 与 lifecycle Delete，并断言 root 外零写入、替换对象零删除、失败有界。

## Warnings

### WR-01: nested public watchdog 在 inner watcher 不响应时可留下 orphan body

**Files:** `safety/scripts/test.sh:25-118`, `safety/scripts/test.sh:313-355`, `safety/scripts/test.sh:941-1030`, `safety/internal/e2e/phase_e2e_test.go:386-491`

**Issue:** 每次 public `test.sh` 都 fork 一个 body 并 `setpgrp`。wave/phase 又通过 `/bin/bash test.sh task|wave ...` 递归调用 public entry，所以进程树是 outer body group → inner watchdog → inner body group。outer watchdog timeout 只 TERM/KILL outer body group；正常 inner watcher 会转发信号，但若 inner watcher 已停止、卡死或在转发前被 KILL，inner body 位于另一个 PGID，不在 outer kill target 中。

仓库外隔离复现把 outer wave test budget 缩为 800ms，并在 inner body 建立后暂停 inner watcher。outer entry 正确返回唯一 deadline envelope + `124`，但 inner body 仍以 PPID 1 存活，需审查 harness 额外终止。未暂停 watcher 的控制组会正常级联清理，因此问题不是“所有 nested run 都 orphan”，而是 hard-deadline guarantee 依赖每一层 watcher 都能及时转发。现有 block-point matrix 在启动 nested child 之前阻塞，没有覆盖 inner watcher 已 fork body 后失去响应。

**Impact:** public caller 能看到正确 `124`，但 build/test/helper 仍可能继续占用 CPU、文件描述符和临时 root，并与 parent cleanup 竞争；这违反 15/47/305 秒 lifecycle ceiling 与 no-orphan contract。

**Fix:** 聚合层不要递归创建独立 public watchdog/PGID；让一个不可由 public argv/env 选择的固定内部 dispatcher 保持完整 invocation tree 在同一 supervisor authority 下，或让 outer supervisor可靠持有并终止所有 descendant groups。回归测试要在 nested body 建立后使 inner watcher 不响应，随后验证 outer timeout/TERM 后 inner body、helper、所有 marker-owned roots 均消失；不得恢复 caller-selectable bypass guard。

### WR-02: worktree mode 使用 `0111`，会接受 Git 认为已降级的 executable mode

**Files:** `safety/internal/workflow/synthetic.go:1021-1051`, `safety/internal/e2e/walking_skeleton_test.go:524-545`

**Issue:** `readBoundedNoSymlinkWithMode` 用 `before.Mode().Perm()&0o111 != 0` 映射 `100755`。Git 的 index/worktree executable mapping 只检查 owner execute bit `0100`。在隔离 temporary Git root 中，HEAD/index 为 `100755` 后将 worktree 改为 `0655`，Git 报告 `100755 => 100644` mode change；当前函数仍映射为 `100755`。bytes、index blob 和 HEAD blob 全相同时，`validateTrackedInput` 因而错误接受这项 chmod-only drift。现有测试只覆盖 `0700 ↔ 0600`，没有覆盖 partial execute bits。

**Impact:** repository input proof 接受 Git 自己可观察的一种 mode substitution；在 owner execute 被清除但 group/other execute 仍存在时，实际调用用户的执行语义也可能改变。这与文档宣称的 exact worktree mode binding 不一致。

**Fix:** 按 Git 规则只用 `mode.Perm()&0o100` 映射 `100755`，否则映射 `100644`。加入 HEAD/index `100755` + worktree `0655` 必须拒绝，以及 HEAD/index `100644` + 非-owner execute bits 仍按 `100644` 处理的隔离 Git cases；保留双向 `0700/0600` 和 bytes/index/symlink negatives。

### WR-03: “frozen HEAD” 只在每个 input 内读取，整个 run 可混合不同 commits

**Files:** `safety/internal/workflow/synthetic.go:33-35`, `safety/internal/workflow/synthetic.go:364-390`, `safety/internal/workflow/synthetic.go:393-471`, `safety/internal/workflow/synthetic.go:679-705`, `safety/internal/workflow/synthetic.go:842-875`, `safety/internal/workflow/synthetic.go:919-928`

**Issue:** `trackedRepository` 只保存 root；`openTrackedRepository` 不捕获 HEAD OID。每次 `validateTrackedInput` 都重新运行 `rev-parse HEAD^{commit}`。`RunSynthetic` 连续验证 blueprint、surfaces、raw sample；`BuildPhaseReport` 验证 suite、expected report、四个 manifest 和 synthetic manifest。若在两个 validation 之间 checkout/update HEAD 和 index，第一个 `trackedInput.data` 可来自 commit A，后续 inputs 可各自合法绑定 commit B；最终 workflow 使用一组从未共同存在于任何单一 tree 的 bytes。

**Impact:** 每个单文件 proof 仍可自洽，但 suite/manifest/blueprint 组合不再是一个 immutable repository snapshot，削弱 source-of-truth evidence，并与 `safety/README.md` / `safety/CLAUDE.md` 的 frozen HEAD 表述冲突。

**Fix:** `openTrackedRepository` 一次捕获、验证并保存 exact HEAD commit OID；所有 `ls-tree`/blob checks 都使用该 OID。可在 workflow 返回前再 fail closed 检查 current HEAD 未改变，但不能以末尾检查替代最初 OID binding。加入 package-private scheduling seam：第一次 input validation 后切换到内容组合不同但各自合法的第二 commit，要求全部输入仍绑定初始 OID或整次运行拒绝。

### WR-04: `EvalSymlinks` 与绝对路径重开之间可替换 intermediate directory

**Files:** `safety/internal/workflow/synthetic.go:426-471`, `safety/internal/workflow/synthetic.go:1021-1051`

**Issue:** `validateTrackedInput` 先 path-based `filepath.EvalSymlinks(path)` 并要求结果等于 path，随后才调用另一个 path-based reader。reader 的 `Lstat` / `OpenFile(O_NOFOLLOW)` / `Lstat` 只拒绝 final-component symlink；`O_NOFOLLOW` 不阻止 intermediate components 被解析为 symlink。在两步之间把仓库内 intermediate directory 替换成指向仓库外、但含有相同 bytes/mode 文件的 symlink，before/opened/after 都会观察同一 external regular inode，Git relative blob/mode 也仍匹配，于是该物理来源被接受。

**Impact:** content digest 仍受 HEAD blob 限制，所以这不是任意内容注入；但“必须从 exact tracked worktree path、symlink substitution fail closed”的 provenance claim 被绕过，且后续代码无法证明消费的是仓库 root 下的 inode。

**Fix:** 从一次打开并 identity-bound 的 repository root handle 进行 relative rooted read，并明确拒绝每个 traversed component 的 symlink/replacement；不要在 `EvalSymlinks` 后按绝对 pathname 重新打开。加入同步 seam，在 initial path check 后替换 intermediate directory，断言 validation 拒绝且 external file 不被消费。

## Revalidated Fixes and Known Boundaries

- `2b8f368` 已关闭：apply/read-only 都要求 `FreshObserved.State` 存在于 exact typed observed facts；correct/absent/different-valid-state cases 覆盖 validator、Store 零首写和 CLI pre-store。
- `b7e5519` 已关闭前一轮的 pre-existing child symlink、object/transition symlink/FIFO/device-shaped read 和 operation-before replacement cases；CR-01 只针对 precheck 之后的 namespace race 与 name-only cleanup。
- `d4a58f1` 已关闭 caller-selectable environment/PID/FD/nonce watchdog bypass；WR-01 不恢复该旧入口，只指出 nested supervisor tree 的 failure mode。
- `47c84ad` 已绑定 consumed bytes 与大部分 worktree mode drift；WR-02 是 Git owner-execute bit 语义差异，WR-03/04 是 run-wide snapshot 与 path provenance 的剩余边界。
- iteration-2 的 marker atomic publish、manager-tree internal symlink target containment、ephemeral per-run sentinel key、closed public identity/command-result contract和 marker-owned public fixture entrypoint在本轮未发现回归。
- tracked controlled-service proof 缺失时，current-host 仍在任何 adapter/workload 调用前 fail closed；full offline phase、所有 task/wave、docs gate、format/vet 与 secret scan 通过，但不会自动消除上述未覆盖 race。

---

_Reviewer: gsd-code-reviewer_
_Depth: standard_
