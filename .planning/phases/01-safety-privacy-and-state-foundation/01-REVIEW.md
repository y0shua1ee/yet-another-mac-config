---
phase: 01-safety-privacy-and-state-foundation
reviewed: 2026-07-11T00:01:26Z
depth: standard
files_reviewed: 58
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
  warning: 3
  info: 0
  total: 4
status: issues_found
---

# Phase 01: Code Re-Review Report (Iteration 3)

**Reviewed:** 2026-07-11T00:01:26Z
**Depth:** standard
**Files Reviewed:** 58
**Status:** issues_found

## Review Scope

本轮从 iteration-2 修复后的当前源码重新审查，不以 `01-REVIEW-FIX.md` 的结论替代代码验证。范围仍是 frontmatter 中列出的 58 个 Phase 01 文件，覆盖 artifact schema/canonicalization/lineage/store、privacy/capture、fixture/retention/network、synthetic 与 real sentinel、one-shot claim/report binding、control plane、CLI、runner、manifests/testdata 和两层文档。

没有读取或修改 `.config/alma/`，没有执行 real snapshot、`launchctl`、真实 HOME/manager/service adapter、网络、Nix、Homebrew、mise、uv、rustup、安装或 activation。所有动态复现仅写入仓库外 fresh temporary roots，并在结束时删除。current-host 因 tracked `launchctl` isolated-negative proof 缺失而在任何 adapter/workload 前返回 `manual-required` / `32` / `indeterminate` / zero-call，是已知安全边界，不作为 finding。

## Verification

最终串行验证结果：

```text
/bin/bash -n safety/scripts/test.sh
  -> exit 0

gofmt -d <all safety/*.go>
  -> no diff

isolated offline go vet ./...
  -> exit 0

./safety/scripts/test.sh wave skeleton
./safety/scripts/test.sh wave artifact-contracts
./safety/scripts/test.sh wave privacy
./safety/scripts/test.sh wave fixture-policy
./safety/scripts/test.sh wave sentinels
./safety/scripts/test.sh wave controlplane
./safety/scripts/test.sh task phase-e2e
  -> all exit 0 with synthetic-sentinel-passed

./safety/scripts/test.sh phase
  -> exit 0; {"status":"synthetic-sentinel-passed","suite":"phase"}

./safety/scripts/test.sh task docs-and-phase-gate
  -> exit 0; {"status":"synthetic-sentinel-passed","suite":"docs-and-phase-gate"}

gitleaks detect --no-git --source safety --redact
  -> no leaks found

git diff --check -- safety README.md .planning/phases/01-safety-privacy-and-state-foundation/01-REVIEW.md
  -> exit 0 after report rewrite
```

第一次完整 phase 在与审查复现并行时到达 runner deadline 并按契约返回唯一 envelope + `124`；随后六个 wave、`phase-e2e` 和完整 phase 均在无并发审查负载的串行运行中通过，因此不把第一次资源竞争视为产品 finding。GREEN 证明现有 suite 通过，但 suite 没有覆盖以下四条可复现或可直接构造的路径。

## Narrative Findings (AI reviewer)

iteration-2 的七项修复均已在代码层重新验证：公开 legacy fixture root 已移除，identity/command-result contract 已收紧，manager-tree 内部 symlink escape 已拒绝，marker 使用 temp+fsync+no-replace hard-link 原子发布，real sentinel key 已内部生成并清零，tracked input 已绑定 frozen HEAD/index/blob，runner 也加入 PID/FD/nonce 握手。

仍有一个阻塞安全边界：artifact store 只 canonicalize 顶层 root，没有把 `sha256`/`transitions` 子目录纳入同一 no-symlink/regular-file identity contract。隔离复现证明预置 `sha256` symlink 能把七个 immutable objects 写到选定 store root 外。其余三个 warning 分别是 watchdog 握手仍可由 direct caller 自洽伪造、Git proof 没有比较当前 worktree executable bit，以及 read-only evidence 的 compact state 未证明存在于其 exact observed object 中。

## Critical Issues

### CR-01: pre-existing store child symlink 可把 immutable artifacts 写到所选 external root 外

**Files:** `safety/internal/artifact/store.go:59-76`, `safety/internal/artifact/store.go:145-195`, `safety/internal/artifact/store.go:688-720`, `safety/internal/artifact/store.go:744-799`, `safety/internal/artifact/store.go:835-848`

**Issue:** `NewStoreWithClock` 只通过 `ValidateExternalRoot` canonicalize 顶层 root，然后 `rebuildMetadata` 直接 `os.ReadDir(root/sha256)`。它没有对 `root/sha256` 或 `root/transitions` 做 `Lstat`、no-symlink、same-directory identity 或 resolved containment 验证。后续 `write` / `writeTransition` 的 `MkdirAll`、`CreateTemp` 和 `Link` 都会跟随预先存在的 directory symlink。

在仓库外隔离复现中，创建一个空 store root，并令 `store/sha256 -> escape-target`；`NewStore` 接受该 root，随后公开 CLI `store --mode apply` 成功返回，并把完整 graph 的七个 immutable objects 写入 `escape-target`，即选定 store root 之外。复现只使用 fresh temporary repositories/fixtures，结束后全部删除。

同一边界还存在 unbounded special-file read：`rebuildMetadata` 只拒绝 directory、symlink 和非法 digest name；一个以合法 digest 命名的 FIFO 会进入 `readBoundedFile`，而该 helper 直接 `os.Open`，可在有 writer 前无限阻塞。`loadExact` / transition read 也复用这个 reader。因此当前实现既没有兑现 exact external-root containment，也没有兑现 bounded artifact read。

**Impact:** 调用者只需选择一个已有或被替换过子目录的 external store，就可能在“content-addressed store”写入阶段修改 root 外的任意可写目录；恶意/损坏 store 还可以让 standalone CLI 无期限挂起。这违反 D-02、SAFE-04/05、Plan 01-01 的 outside-root-write blocking condition，以及文档对 immutable external local state 的边界说明。

**Fix:** 在打开 store 时验证或原子创建 exact `sha256` 和 `transitions` directories，拒绝 symlink、FIFO/device/socket/non-directory，记录 directory identity，并在每次 read/write/link 前后重新验证 identity 与 resolved containment。优先用 directory-relative no-follow operations；若保持 path API，至少要 `Lstat` + no-follow open + `Fstat`/`SameFile` + before/after directory identity，并拒绝任何不确定性。`readBoundedFile` 应只接受 regular file，以 nonblocking/no-follow 方式打开，做 size precheck、`limit+1` read 和 named/opened identity recheck。加入预置 `sha256` symlink、`transitions` symlink、digest-object symlink/FIFO、child replacement/race 负例，断言零 root 外写入、bounded non-zero 返回和无残留。

## Warnings

### WR-01: direct caller 可构造完整 PID/FD/nonce 握手并绕过 public runner watchdog

**Files:** `safety/scripts/test.sh:20-34`, `safety/scripts/test.sh:34-113`, `safety/internal/e2e/phase_e2e_test.go:382-451`, `safety/README.md:27`

**Issue:** 新握手比单一 ambient PID 更强，但所有认证材料仍由 direct parent 完全控制：脚本接受 `YAMC_RUNNER_WATCHDOG_PID == PPID`、任意继承 FD，以及该 FD 第一行等于 caller-provided 64-hex nonce。调用方可以自己建立 pipe、写入选择的 nonce、清除 read FD 的 `CLOEXEC`，fork/exec runner 并填写自己的 parent PID；脚本会把该组合判定为 authenticated，完全跳过 lines 34-109 的 watchdog。

隔离复现向固定 setup block helper 传入 500ms 测试预算。伪造 direct parent/FD/nonce 后，helper 已发布 marker，runner 在 1.5 秒后仍存活；外层审查 harness 随即终止独立 process group 并清理 temporary marker。现有 `forged` case 只设置正确 PID，`stale` case 只设置不一致的 PID/FD/nonce，均未覆盖自洽的 inherited pipe。

**Impact:** 文档承诺的 15/47/305 秒 public-entry hard ceiling 可由 ambient caller 绕过；阻塞 setup/build/test 或 child dispatch 可无限存活。它不会直接授予 live mutation 权限，但会破坏 Phase 01 的 bounded-resource 与 no-orphan 保证。

**Fix:** public entry 不应存在由 caller-controlled environment/FD 关闭 watchdog 的路径。最直接方案是每次 public invocation 都安装 watchdog，并把 wave/phase 的内部调度改为同一受监控进程内的固定 dispatcher；或者把内部 body 移到独立 helper，由不可从 public argv/env 选择的 wrapper 生命周期控制。无论结构如何，加入“self-consistent direct parent + live inherited pipe + matching nonce”负例，要求仍以唯一 deadline envelope + `124` 结束，并证明 helper、process group 与 marker-owned root 全部消失。

### WR-02: tracked-input proof 接受 chmod-only worktree mode drift

**Files:** `safety/internal/workflow/synthetic.go:425-470`, `safety/internal/workflow/synthetic.go:590-592`, `safety/internal/workflow/synthetic.go:1015-1031`, `safety/README.md:7`

**Issue:** `validateTrackedInput` 已正确证明 index entry 与 frozen HEAD tree 的 mode/blob 相等，并证明 `cat-file` blob bytes 等于本次 worktree bytes；但它从未把当前 worktree file mode 映射为 Git `100644`/`100755` 再与 index/tree mode 比较。`readBoundedNoSymlink` 只证明 read 前后 mode 没变化，不证明它等于 tracked mode。

隔离复现从当前 commit 建立 local clone，把 tracked blueprint 从 `100644` 改为 `100755` 而不改 bytes；Git 明确报告 mode drift，但 `fixture run` 仍返回 `synthetic-sentinel-passed`。这与 README 的“index mode/blob 与 frozen HEAD tree 完全一致，实际消费 worktree bytes 也受同一 proof 约束”陈述不完全一致。

**Impact:** 当前 Phase 01 输入均为 JSON，因此 executable bit 不会改变 parser 语义；但 source-of-truth proof 仍接受一种 Git 可观察的 worktree substitution，也缺少修复报告所声称的 content-and-mode 完整性。

**Fix:** 让 bounded reader 返回经过 before/after identity 验证的 file mode；按 Git 规则将任何 executable bit 映射为 `100755`，否则为 `100644`，并要求它等于 index/tree mode。加入 `100644 -> 100755` 与反向 drift 负例，断言在 fixture/store 创建前 fail closed，同时保留 bytes-only、staged/index 和 symlink 负例。

### WR-03: read-only lineage 接受 observed facts 中不存在的 `FreshObserved.State`

**Files:** `safety/internal/artifact/lineage.go:159-189`, `safety/internal/artifact/kinds.go:276-288`, `safety/internal/e2e/artifact_cli_test.go:638-660`

**Issue:** apply lineage 在验证 compact `FreshObserved` descriptor 时调用 `observedContainsState`，要求 descriptor state 真正存在于 exact fresh observed artifact 的 facts。read-only 分支只比较 observed digest、content digest、scope 和空 receipt；任意合法 logical ref 都可作为 `FreshObserved.State`，即使 exact observed artifact 完全没有该 state。kind validation 只检查词法 `validLogicalRef`，无法补上语义绑定。

因此一个 read-only evidence/report bundle 可以在 digest、scope、provenance 和 report edges 全部有效时，持久化一个与其 exact observation 相矛盾的 state descriptor。现有 valid read-only fixture 使用匹配 state，但没有只替换 descriptor state 的 negative case。

**Impact:** 不会创建 applied receipt，也不能绕过 real-sentinel one-shot claim capability；但会削弱 read-only verification evidence 的诚实性，并使 apply/read-only 对同一 descriptor 的语义不一致。

**Fix:** 在 `validateReadOnlyEdges` 的 freshness predicate 中加入 `observedContainsState(observedPayload, evidencePayload.FreshObserved.State)`，与 apply path 保持一致。加入 absent-state、wrong-state-with-valid-logical-ref 和 correct-state 控制用例，并通过 `ValidateLineage`、`Store.WriteGraph` 与 CLI `store --mode read-only` 三层验证失败发生在首笔写入前。

## Revalidated Fixes and Known Boundaries

- iteration-2 CR-01 已关闭：公开 `fixture run` 只接受 external base，physical child/store path 不再作为 CLI 参数；fresh child、atomic marker、rollback/finalize 均绑定 direct-child inode/UID/nonce。
- iteration-2 CR-02 已关闭到当前 production surfaces：trusted run ID、fixed suite/operation registry、closed command-result field/type registry，以及 neutral identity/opaque/unknown-field canaries 均在 construction/render/store/CLI 前拒绝。
- iteration-2 CR-03 已关闭：manager-tree 会解析每个内部 symlink 的最终 target；relative、absolute 和 chained escape 都是 incomplete/no token/no claim。
- iteration-2 WR-01 已关闭：marker 使用同目录 temp + sync/close + no-replace hard-link 发布；partial marker rollback 只删除本次 fresh direct child，并保留 base/sibling。
- iteration-2 WR-03 已关闭：production real sentinel key 在 proof gate 之后内部生成，失败/成功路径 defer clear；caller 不再提供可复用 key。
- iteration-2 WR-04 的 blob/content 部分已关闭：Git plumbing 固定 `/usr/bin/git`，冻结 HEAD，要求 unique stage-0 index/tree object 一致，并将实际 bounded bytes 与 HEAD blob 比较；WR-02 仅指出遗漏的 current worktree mode 比较。
- tracked service proof 缺失时，current-host 路径仍在任何 adapter/workload 调用前返回 `manual-required` / `32` / `indeterminate` / zero-call；这是安全保守状态，不是失败实现或 finding。
- full offline phase、所有 component waves、phase-e2e、docs gate、Go formatting/vet 与 secret scan 均通过；这些 GREEN 结果不会自动消除上述未覆盖路径。

---

_Reviewer: gsd-code-reviewer_
_Depth: standard_
