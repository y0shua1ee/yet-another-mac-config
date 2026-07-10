---
phase: 01-safety-privacy-and-state-foundation
reviewed: 2026-07-10T21:57:34Z
depth: standard
files_reviewed: 56
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
findings:
  critical: 2
  warning: 4
  info: 0
  total: 6
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-07-10T21:57:34Z
**Depth:** standard
**Files Reviewed:** 56
**Status:** issues_found

## Review Scope

审查范围严格限定为 `safety/` 下的 56 个 Phase 01 文件，包括本地指导与文档、Go module、CLI、artifact/privacy/fixture/sentinel/control-plane/workflow 实现、测试、runner、tracked manifests 与 synthetic testdata。未检查或修改 `.config/alma/`，未修改既有 dirty `CLAUDE.md` 或 `.ai/` 文件，也未把当前 Mac 当作测试 fixture。

## Methods and Verification

按 standard depth 逐文件阅读，并针对以下边界做了跨文件调用链核对：artifact canonical digest/lineage/lifecycle/atomic store、共享 privacy gate、fixture ownership/retention、synthetic 与 real sentinel evidence/claim、real zero-call proof gate、runner task/wave/phase deadline 与 aggregation、report-only control plane。

仅通过仓库允许的离线隔离入口运行了以下验证；四项均返回 `synthetic-sentinel-passed`：

```text
./safety/scripts/test.sh task phase-e2e
./safety/scripts/test.sh task privacy-boundary
./safety/scripts/test.sh task artifact-lineage
./safety/scripts/test.sh task real-sentinel-envelope
```

测试通过不消除下列问题：其中两项是现有测试将 checked-in expectation 当作证明或仅做结构文字检查而产生的覆盖缺口。

## Narrative Findings (AI reviewer)

### Critical Issues

#### CR-01: `report` 从 expected fixture 复制 scoped claim，而不是从本次外层 real evidence 推导

**Severity:** Critical
**Files:** `safety/internal/workflow/synthetic.go:487-556`, `safety/internal/workflow/synthetic.go:651-688`, `safety/cmd/yamc-safety/main.go:137-158`, `safety/internal/e2e/phase_e2e_test.go:127-142`, `safety/internal/e2e/phase_e2e_test.go:189-200`, `safety/testdata/blueprints/walking-skeleton/expected-report.json:15-77`

**Issue:** `BuildPhaseReport` 校验 tracked suite、expected-report 文件和 synthetic artifact graph 后，直接把 `OuterSequence`、`Verdict`、`Claim` 与六组 `SurfaceEvidence` 从 `expected` template 复制到输出。该路径没有接收 `sentinel.Evidence`，没有调用 `RunRealEnvelope` / `Evaluate` / `RequestClaim`，也没有把 report digest 绑定到本次外层 evidence digest。expected fixture 本身硬编码了 `passed`、`covered-surfaces-unchanged-for-run` 和六组相等的假 HMAC token，因此它只是预期值，不能证明一次 observation window 确实发生过。

Phase E2E 在调用 `report` 前递归执行一次独立的 `TestRealSentinelEnvelope`，但 `assertIsolatedRealEnvelopeSuite` 只观察子 `go test` 的退出状态；其 evidence、window、tokens 和 digest 全部未传给 report。`report` CLI 也可对一个有效 synthetic summary/store 单独调用，因此无需同一命令或同一 evidence window 就能输出 top-level `passed` 和 scoped claim；同时 report 内的 `current_host.claim_eligible` 仍为 `false`。这破坏了 claim ceiling 的核心信任边界，并使通过的 phase report 可以陈述未被本次运行证明的安全结论。

**Fix:** 不要从 expected fixture 读取 verdict、claim 或 surface tokens。将 report 构造放进同一次受控 `RunRealEnvelope` 生命周期，在 process-only `realBinding` 尚有效时以实际 `Evidence` + `Evaluation` 调用 `sentinel.RequestClaim`，并把实际 evidence digest、window ID、manifest/suite digests 与 surface evidence 写入 report lineage。若 `report` 必须保持独立/可重放，则序列化后的 evidence 按当前设计已失去 claim capability，独立命令必须输出 claim-ineligible 的 bounded synthetic/report status，而不能恢复 scoped claim。E2E 应直接消费该生产路径的 envelope/report，不应以另一次单元测试的退出码替代 evidence binding。

#### CR-02: Privacy gate 按字段名过滤，允许秘密或真实身份藏在合法字符串字段中并落盘

**Severity:** Critical
**Files:** `safety/internal/privacy/gate.go:420-510`, `safety/internal/artifact/envelope.go:179-206`, `safety/internal/artifact/kinds.go:291-333`, `safety/internal/artifact/store.go:100-126`, `safety/internal/privacy/gate_test.go:161-205`, `safety/testdata/canaries/cases.json:4-65`

**Issue:** `scanValue` 会拒绝 forbidden key、绝对引用、private network 和少数被标记为 logical field 的值，但普通允许字段中的任意字符串直接通过。Artifact contract 对 `run_id` / `tier` / `suite_id` 只检查非空，对多种 `state` 只检查非空，对 operation IDs 也只做很弱的结构约束。因此，把 API token、真实用户名、主机名或 provider item 放入 `run_id`、`suite_id`、`state`、`reason`、`status` 或 `operation_ids` 等合法字段时，只要它不是绝对路径/private IP/损坏的 registered-prefix ref，`artifact.New` 和 `privacy.Gate` 都会接受；`Store.Write` 随后会把 canonical bytes 写入 content-addressed store。

现有 canary 全部把敏感值放在 `secret_value`、`username`、`hostname`、`environment`、`raw_output` 等本来就被 forbidden-field 表捕获的 key 中，没有验证敏感值经允许字段进入的路径。这意味着共享 gate 并未落实“无法分类的数据在 stdout/stderr/落盘前硬失败”，且真实秘密可进入本地 artifacts，之后被复制进 Git 或报告。

**Fix:** 为每个 artifact/command-result schema 建立封闭的字段和值分类，而不是依赖 key-name denylist：logical ref 必须走 typed parser，digest/token/timestamp 必须走各自 validator，state/status/reason/tier/mode 必须是封闭 enum，run/suite/operation IDs 必须由可信 adapter 生成或满足严格 public-ID contract；任何未注册的自由字符串都拒绝。为每个仍允许的字符串字段加入 canary matrix，至少覆盖 `run_id`、`suite_id`、`state`、`reason`、`status`、`operation_ids`，并验证 CLI rendering 和 `Store.Write` 均在写入前失败且不回显原值或派生指纹。

### Warnings

#### WR-01: Fixture 初始化失败后没有回滚刚创建的 owned root

**Severity:** Warning
**File:** `safety/internal/fixture/root.go:123-163`

**Issue:** `Create` 在 `os.Mkdir(physicalRoot)` 成功后，若 `writeMarker` 或 `createFixtureDirectories` 失败就直接返回。此时调用方拿不到 `Root` / `Retention` handle，无法执行 `Finalize`；外部 retention base 下会遗留无 marker、半写 marker 或部分目录的 `fixture-<nonce>` 子树。磁盘满、I/O 错误或中途权限失败即可触发，违反“成功和失败 fixture 默认都删除”的生命周期保证，也会让重复测试累积不可管理状态。

**Fix:** 把初始化改为事务：`Mkdir` 后立即注册只针对该 exact freshly-created direct child 的 deferred rollback，并在 marker 与全部目录完成后才置 `committed=true`。回滚必须重复验证 base containment、nonce/direct-child identity，并在 marker 已写入时复用 marker/UID 校验；同时加入可注入的 marker-write 和 mid-directory-create failure 测试，断言只移除本次 child、绝不越过 retention base。

#### WR-02: Synthetic snapshot 在检查 `MaxBytes` 前先把整个 regular file 读入内存

**Severity:** Warning
**Files:** `safety/internal/sentinel/snapshot.go:233-299`, `safety/internal/sentinel/real.go:1220-1280`

**Issue:** `fingerprintSurface` 对 regular file 调用 `os.ReadFile(path)`，读完后才累计 `limits.bytes` 并检查 `bounds.MaxBytes`。时间窗口也只在进入 `WalkDir` callback 时检查，无法中断正在进行的完整读取。大文件或读取期间持续增长的文件因此能在最终返回 `ReasonOverflow` / `ReasonWindow` 前突破声明的内存、字节和 wall-time 边界。real adapter 已有 `OpenRoot`、size precheck、`io.LimitReader(max+1)` 和前后 identity recheck 的正确模式，但 synthetic path 没有复用。

**Fix:** 在打开文件前计算剩余 byte budget 并用 rooted handle 验证 size；最多读取 `remaining+1` bytes，超过即立即返回 `ReasonOverflow`，读取前后验证 inode/mode/size/mtime 并在结束时重查 deadline。最好抽出 synthetic/real 共用的 bounded rooted reader，加入“单个超大文件”和“读取期间增长”负例，证明不会先分配/读取完整文件。

#### WR-03: 调用方可用未来 `created_at` 绕过 snapshot 的 24 小时 retention

**Severity:** Warning
**Files:** `safety/internal/artifact/kinds.go:139-152`, `safety/internal/artifact/kinds.go:179-194`, `safety/internal/artifact/store.go:100-109`, `safety/internal/artifact/store.go:651-656`

**Issue:** Lifecycle validation 只要求 `expires_at == created_at + 24h`，没有把 `created_at` 绑定到可信 store clock。`NewWithOptions` 允许调用方提供任意 RFC3339 时间；`Store.Validate` 只比较当前时间是否已到 caller-supplied `expires_at`。因此一个 `created_at=2100-...`、`expires_at=created_at+24h` 的 desired/observed snapshot 在今天是 structurally valid 且不会被判过期，实际 retention 可被延长数十年；若再被 evidence/report pin，清理语义会进一步偏离 24 小时合同。

**Fix:** 在 external store ingest 时用 `store.now()` 校验 snapshot `created_at` 不得晚于当前可信时间加一个明确且很小的 clock-skew allowance，并保证 `expires_at` 不得超过 store time + 24h + allowance；或者由 store 在原子写入时生成/签定 lifecycle，而不信任外部 artifact 时间。加入 future-created snapshot 的 Write/Reopen/Read/Delete 负例，并保留测试时钟注入以避免真实时间 flake。

#### WR-04: 15/47/305 秒“hard deadline”没有覆盖完整 runner 生命周期

**Severity:** Warning
**Files:** `safety/scripts/test.sh:92-109`, `safety/scripts/test.sh:121-223`, `safety/scripts/test.sh:225-262`, `safety/scripts/test.sh:789-837`, `safety/scripts/test.sh:847-925`, `safety/internal/e2e/phase_e2e_test.go:317-370`

**Issue:** `RUNNER_STARTED_SECONDS` 与 `run_with_runner_deadline` 只在 Go build/list/test 命令周围建立 process-group alarm。runner 启动阶段的 `command -v`、`mktemp`、`touch`、`mkdir`，docs task 的 `readlink`/`grep`，以及 wave/phase 对 child `test.sh` 的 shell invocation 都不在任何异步 deadline 内；父层只在 child 返回后检查 elapsed。任一这些命令或 child 在建立自己的第一个 Go deadline 前阻塞，就能无限超过宣称的 15/47/305 秒上限。`testPhaseRunnerContract` 仅搜索预算/remaining 字面量，并明确禁止 phase handler 使用 deadline wrapper，未进行真实 overrun/descendant-cleanup 行为验证，因此现有 GREEN 不证明 hard ceiling。

**Fix:** 让每个 task runner 从进程生命周期最早阶段就拥有一个单一 watchdog/process group，覆盖 setup、固定 shell checks、build/list/test 和 cleanup；wave/phase 仍可避免额外嵌套 group，但必须调用“从入口即自限时”的 child，并为 aggregator 自身的 setup/dispatch 建立不会产生 orphan 的绝对 deadline。加入使用固定 fake helper 阻塞 setup/docs/child-before-Go 的行为测试，断言 wall time bounded、exit 124、唯一 `runner-deadline-exceeded` envelope、所有 descendants 被回收且 marker-owned temp root 按策略清理。

## Known Non-Issues and Limitations

- tracked `launchctl-print-service-v1` 缺少 isolated negative proof，因此 current-host gate 返回 `manual-required` / exit 32 / `indeterminate` / claim-ineligible，是已声明的保守边界，不作为 finding。
- `sentinel verify --mode real` 当前在 CLI 中只装载 registry、评估 proof 并调用 `RequireControlledRealEnvelope`；proof 不足或尚无受控 runner 时不会调用 adapter/workload。审查未发现该 zero-call gate 绕过。
- control plane 仍是 report-only，未发现 apply、cleanup、uninstall、prune、trust 或任意 shell dispatch 路由。Fixture marker-owned teardown 不等于真实机器 convergence cleanup。
- 本次没有执行 full `./safety/scripts/test.sh phase`，也没有执行 real host、network、Nix/Home Manager、Homebrew、mise、uv、rustup、service 或 activation 命令；结论来自 56 文件审查与上述四个离线隔离 task。
- standard depth 不是形式化验证；并发 filesystem adversary、磁盘故障注入和真实 deadline overrun 需要修复后的专门行为测试补齐。

---

_Reviewed: 2026-07-10T21:57:34Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
