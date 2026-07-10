---
phase: 01-safety-privacy-and-state-foundation
reviewed: 2026-07-10T22:51:18Z
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
  critical: 3
  warning: 4
  info: 0
  total: 7
status: issues_found
---

# Phase 01: Code Re-Review Report

**Reviewed:** 2026-07-10T22:51:18Z
**Depth:** standard
**Files Reviewed:** 58
**Status:** issues_found

## Review Scope

本次从修复后的当前源码重新审查，不采信 `01-REVIEW-FIX.md` 的结论。范围是在原 56 个文件基础上加入 `safety/testdata/runner/block-helper.sh` 与根 `README.md`，共 58 个文件；frontmatter 保存了完整清单。审查覆盖 artifact/canonical lineage/store、privacy/capture、fixture/retention/network、synthetic 与 real sentinel、one-shot claim/report binding、control plane、CLI、runner、manifests/testdata 和两层文档。

没有读取或修改 `.config/alma/`，没有执行 real snapshot、`launchctl`、真实 HOME/manager/service adapter、网络、Nix、Homebrew、mise、uv、rustup 或 activation。current-host 因 service proof 缺失而 `manual-required` / zero-call 是已知安全边界，不作为问题。

## Verification

仅使用仓库支持的 offline safety 入口；长命令保留同一执行会话并轮询到真实退出码：

```text
/bin/bash -n safety/scripts/test.sh
./safety/scripts/test.sh phase
  -> exit 0; {"status":"synthetic-sentinel-passed","suite":"phase"}
./safety/scripts/test.sh task docs-and-phase-gate
  -> exit 0; {"status":"synthetic-sentinel-passed","suite":"docs-and-phase-gate"}
```

GREEN 证明现有 positive/negative suite 通过，但没有覆盖以下可复现路径。

## Critical Issues

### CR-01: 公开的 legacy `fixture run` 可把已有任意仓库外目录当成 fixture 并覆盖内容

**Files:** `safety/cmd/yamc-safety/main.go:557-583`, `safety/internal/workflow/synthetic.go:169-213`, `safety/internal/workflow/synthetic.go:347-376`, `safety/internal/sentinel/synthetic.go:77-89`, `safety/internal/privacy/capture.go:105-151`

**Issue:** CLI 仍公开接受 `--fixture-root` + `--store-root` 的 legacy 组合。该分支只调用 `artifact.ValidateExternalRoot`，后者只证明路径不在 repository 内；它不要求 root 不存在、为空、由本次创建、带 ownership marker，也不拒绝真实 HOME、manager root 或其他已有仓库外目录。随后 `RunSynthetic` 对该 root 执行 `MkdirAll`，`PrepareSynthetic` 用 `os.WriteFile` 写入/覆盖 `protected/...`，`MaterializeFixtureAdapter` 再写入 `path/bin/...`。因此一个已存在的任意目录（包括真实 HOME）满足 parser 和 preflight，可能在“测试”开始前被写入；该路径也不使用 `fixture.Retention`，所以没有 marker-owned teardown。

这与 README 中“所有生成状态写入仓库外的新建临时根”以及 SAFE-04/SAFE-05 的不破坏现有环境边界直接冲突。现有测试总是传入 `t.TempDir()` 下尚不存在的 child，因此没有覆盖 existing-root 情况。

**Fix:** 删除公开 legacy 入口，或把它限制为不可从 CLI 触达的测试 helper。所有 CLI fixture run 都应只接受 external base，由 `fixture.Create` 原子创建 fresh direct child、写 marker、建立隔离路径并在冻结 verdict 后 finalize。若确需显式 root，必须用原子 `mkdir` 要求目标不存在，并把 exact root 纳入同一 ownership/rollback/retention 状态机；同时拒绝与 HOME、repository、protected roots、store 及彼此的重叠。加入 existing non-empty root、HOME-shaped root、fixture/store overlap 和 preexisting witness 负例，断言零写入。

### CR-02: 原 CR-02 仍未关闭；所谓 public-ID/default contract 仍接受中性身份、未知秘密与未注册字段

**Files:** `safety/internal/artifact/envelope.go:182-221`, `safety/internal/privacy/gate.go:422-474`, `safety/internal/privacy/gate.go:474-603`, `safety/internal/privacy/gate.go:618-645`, `safety/internal/privacy/gate_test.go:267-329`, `safety/testdata/canaries/cases.json:67-105`, `safety/README.md:65-77`, `safety/CLAUDE.md:53-55`

**Issue:** 修复把部分字段接到 validator，但信任仍是基于词法 denylist。`IsPublicID` / `isSafePublicID` 只拒绝已知 marker/prefix；`synthetic-run-alice`、`alice`、无已知前缀的 opaque credential 或稳定主机标识都能作为 `run_id` / `suite_id` 通过。更广泛地，`validateStringField` 的 `default` 分支会接受任意**未注册字段名**中的 public-ID-shaped 字符串，而 `scanValue` 对 number/bool/null 不做字段分类，因此类似 `display_name: "alice"` 或未注册的 numeric identity 也能通过 `privacy.Gate` 和 renderer。Artifact producer 又接受 caller-supplied run metadata，未把这些 ID 绑定到可信生成器或固定 registry。

新增 canary 仍只使用包含 `secret`、`username`、`provider`、`token` 等提示词的值，恰好命中 denylist；它没有测试中性真实身份、无已知前缀的 credential、稳定机器 ID，也没有证明未知 field name fail closed。文档“未注册自由字符串默认拒绝、不能伪装真实身份”的陈述与实现相反。

**Fix:** command-result 必须按具体输出类型使用 closed schema/field registry，不得存在接受任意 key 的 default public-ID 分支；未知字段和值类型一律拒绝。Artifact 的 run/suite/operation ID 应由可信 builder 生成或从固定 public registry 选择，不能仅凭可打印字符就认定为公开。为每一个允许的 string/number/bool 字段建立 positive contract，并加入不含敏感关键词的 identity、opaque credential、stable machine ID、unknown key 和 numeric UID canary；验证 construction、renderer、CLI 与 `Store.Write` 均在输出/落盘前失败且不反射值。

### CR-03: real manager-tree adapter 接受内部 symlink escape，外部目标变化可获得相同的 complete token

**Files:** `safety/internal/sentinel/real.go:1027-1121`, `safety/internal/sentinel/real.go:1181-1235`, `safety/internal/sentinel/real_test.go:163-228`, `safety/internal/sentinel/verdict.go:248-285`

**Issue:** `fingerprintExactTree` 遇到树内 symlink 时调用 `readExactSymlink`，但该函数只对 symlink 自身做 rooted `Lstat`/`Readlink` 和 identity recheck；它从不解析 target 并确认 target 位于 manager root。canonical fact 只包含 link target 的 digest，不包含外部 target 内容。因此在 `managerRoot/current -> /outside/state` 这种树内 escape 上，before/after 都会得到 `ObservationComplete`；只修改 `/outside/state` 内容而保持 link 字符串不变，opaque token 不变。所有 required surface 都如此 complete/equal 时，`Evaluate` 可以返回 passed，`RunRealEnvelope` 可以消费 one-shot capability 并产生 scoped claim。

现有负例只检查“整个 tree root 是 escaping symlink”，并没有创建“tree 内部 symlink 指向 root 外”的情况；相对的 synthetic tree 实现会把 symlink escape 标为 incomplete。Plan 01-05 的 T-04 和验收文字也明确要求 symlink escape → indeterminate。

**Fix:** tree adapter 对每个 symlink 解析绝对 target；任何 target 不在 exact manager root 时立即返回 `ReasonSymlinkEscape`，不要生成 token。若未来确需允许外部 target，必须像 named-file adapter 一样使用 manifest-bound allowed root 并同时 fingerprint target content。加入内部 relative escape、absolute escape、symlink-chain escape，以及“外部 target 内容在 window 中变化”的负例，断言始终 incomplete/无 token/无 claim。

## Warnings

### WR-01: 原 WR-01 只修复了无 marker/完整 marker；partial marker 写失败仍会遗留 fresh child

**Files:** `safety/internal/fixture/root.go:145-199`, `safety/internal/fixture/root.go:273-325`, `safety/internal/fixture/fixture_test.go:27-78`

**Issue:** deferred rollback 已建立，但 `rollbackFreshFixture` 在 `readMarker` 失败且 marker path 仍存在时拒绝删除。默认 `writeMarker` 先 `O_EXCL` 创建文件，再 write/close；磁盘满、部分写或 close failure 可返回错误并留下不完整 marker。defer 随后把这种 fresh、same-inode、same-UID direct child 判为“marker rejected”，最终返回 `fixture initialization rollback failed`，目录仍在。测试中的 marker-write failure hook 在写任何 byte 前直接返回，没有模拟 partial marker，因此原 finding 的“半写 marker”分支仍未关闭。

**Fix:** rollback capability 应在 `Mkdir` 后立即绑定 fresh directory inode/UID/nonce，并区分“本次 writer 创建的 marker path”与外部替换。对本次创建且 directory identity 未变化的 partial marker，应安全删除 exact child；或先把 marker 写入同目录临时文件、fsync/close 后以 no-replace 原子发布，失败时 root 仍保持 unmarked rollback 状态。加入 hook 写入截断 JSON 后返回错误的负例，并验证只删除本次 child、保留 sibling/base。

### WR-02: caller 可伪造 `YAMC_RUNNER_WATCHDOG_PID` 直接绕过 15/47/305 秒入口 watchdog

**Files:** `safety/scripts/test.sh:4-83`, `safety/internal/e2e/phase_e2e_test.go:381-458`

**Issue:** 是否安装 watchdog 只取决于 ambient `YAMC_RUNNER_WATCHDOG_PID == PPID`。顶层调用者可以在环境中把该值设为自己的 PID；脚本便直接跳到 body，所有 setup/docs/build/test/child-dispatch 都失去 hard deadline。测试还会主动删除此变量再启动，因此没有覆盖伪造或 stale inherited guard。这个变量本来只用于 parent watchdog 与其 direct child 的内部握手，但当前没有不可伪造 token/FD 或其他 parent proof。

**Fix:** 不信任 caller-provided ambient guard。用 watchdog 创建并传递的 private inherited FD 加随机 nonce、或只在 watchdog fork 后 exec 一个不公开的 internal mode 并校验 parent/session identity；外部同名环境变量必须被忽略/覆盖。增加 forged guard canary，确保 setup/docs/child block 仍以唯一 envelope + 124 结束且无 orphan/root。

### WR-03: real sentinel 的“per-run key”仍由 caller 提供，freshness 与清零都没有被实现保证

**Files:** `safety/internal/sentinel/real.go:397-448`, `safety/internal/sentinel/real.go:478-532`, `safety/internal/sentinel/real.go:1321-1326`, `safety/internal/sentinel/real_test.go:270-340`, `safety/README.md:103-110`

**Issue:** `RunRealEnvelope` 只检查 `len(options.Key) >= 32`，随后 before/after 复用 caller slice；函数既不生成 fresh key，也不在返回前清零。调用方可跨 run 重用同一 key，使相同 surface 的 HMAC token 在多个 claimed report 中稳定，从而违反文档的 per-run privacy boundary。测试固定使用重复 byte key，只证明单次比较，不证明跨 run freshness 或销毁。

**Fix:** production envelope 内部生成 key，并只向 snapshot stage 暂时传递；defer 清零所有内部副本。测试通过注入受限 RNG/secret factory 获得确定性，而不是暴露 caller-owned key。加入连续两次相同 surface 的 token 必须不同、before/after 同 run 必须可比，以及失败/claim-consumer rejection 后 key buffer 已清零的测试。

### WR-04: `validateTrackedInput` 只验证“位于仓库内”，没有证明输入被 Git 跟踪

**Files:** `safety/internal/workflow/synthetic.go:347-395`, `safety/internal/e2e/walking_skeleton_test.go:50-99`, `safety/internal/e2e/phase_e2e_test.go:107-200`

**Issue:** preflight 和 phase report 都依赖名为 `validateTrackedInput` 的 helper，但它仅做 `EvalSymlinks`、regular-file 与 repository containment；任何 ignored/untracked file 也会被接受为 blueprint/surface input。Phase report 的 suite/expected path虽固定，`fixture run` 的 blueprint/surfaces 是 caller path。这样 synthetic desired state 可以来自未进入 Git source-of-truth 的本机文件，且与 CR-02 组合时可能把本机身份伪装成 public ID 后持久化。现有测试只确认文件在 repo 中，不查询 tracked identity，也没有 untracked/ignored negative。

**Fix:** 使用不触发网络或 hooks 的 exact Git plumbing/预计算 tracked manifest，把 allowed logical repo inputs绑定到 tracked blob/digest；Git 不可用、不是 worktree、查询失败、文件未跟踪或 tracked set 为空时 fail closed。至少在 Phase 1 可直接限制为固定 tracked blueprint/surface 路径及 checked-in digest。加入 repo 内 untracked、ignored、symlinked-untracked 和 index/worktree substitution 负例。

## Revalidated Fixes and Known Boundaries

- 原 CR-01 的 standalone report overclaim 已关闭：`BuildPhaseReport` 只输出 `synthetic-report-claim-ineligible`；checked-in expectation 不含 verdict/claim/token；`ConsumeClaim` 只在 process-local real binding 有效时消费一次，序列化 evidence 不能重建 capability。
- 原 WR-02 的 regular-file bounded reader 已采用 size precheck、`remaining+1` streaming、deadline 与 before/after identity recheck；超大和增长负例存在。
- 原 WR-03 的 snapshot lifetime 已在 write/reopen/read/delete 绑定 store clock 与两分钟正向 skew。
- 原 WR-04 的 runner 已把 watchdog 放到 setup 前，并有 setup/docs/pre-child 的 exit 124/no-helper/no-temp-root 行为测试；WR-02 是该修复中新发现的 ambient guard bypass，而不是否定已覆盖的正常路径。
- tracked service proof 缺失时，current-host CLI 仍返回 `manual-required` / `32` / `indeterminate`，并在 adapter/workload 前 zero-call；这保持为安全边界，不是 finding。
- full offline phase 与 docs gate 均通过，但不会自动消除上述未覆盖路径。

---

_Reviewer: gsd-code-reviewer_
_Depth: standard_
