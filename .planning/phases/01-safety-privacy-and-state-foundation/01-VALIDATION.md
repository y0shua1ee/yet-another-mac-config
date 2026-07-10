---
phase: 1
slug: safety-privacy-and-state-foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-10
---

# Phase 1 — Validation Strategy

> Phase 1 的反馈采样合同。所有命令只允许读取仓库跟踪输入，并在仓库外的全新 synthetic fixture root 中写入；不得安装、下载、激活、修复、清理或探测真实 Mac 状态。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard library `testing` + repository-owned Bash runner |
| **Config file** | `safety/go.mod`；Phase 1 不引入需要联网取得的第三方 test dependency |
| **Quick run command** | `./safety/scripts/test.sh task <task-suite>` |
| **Full suite command** | `./safety/scripts/test.sh phase` |
| **Estimated runtime** | quick suite 目标小于 10 秒；offline phase suite 目标小于 60 秒 |

Runner 必须固定 `GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`、`GOENV=off`、`GOWORK=off`、`CGO_ENABLED=0`，并把 `HOME`、全部 `XDG_*`、`TMPDIR`、`GOCACHE`、`GOMODCACHE`、manager roots 与 artifact store 指向仓库外的 fresh fixture root。缺少本地 Go toolchain 时返回 `manual-required`，不得调用 Nix、Homebrew、mise 或其他 manager 自动补齐。

---

## Sampling Rate

- **After every task commit:** 运行该任务对应的 `./safety/scripts/test.sh task <task-suite>`，并确认 sentinel verdict 为 `passed`。
- **After every plan wave:** Wave 1 运行 `./safety/scripts/test.sh wave contracts`；Wave 2 及后续运行 `./safety/scripts/test.sh wave isolated-harness`。
- **Before `$gsd-verify-work`:** 运行 `./safety/scripts/test.sh phase`；只有完整 offline synthetic/isolated suite 为绿色且 claim 精确等于 `covered-surfaces-unchanged-for-run` 才可继续。
- **Max feedback latency:** 单任务 10 秒，单 wave 30 秒，完整 phase gate 60 秒；超限返回 non-zero，不能转 live-check、重试到通过或把 required sentinel 降级为 optional。

---

## Per-Task Verification Map

下表是 research-informed provisional mapping。Planner 必须让最终 PLAN task ID 与本表一致，或在提交计划前同步本表；不得留下 `TBD` 或把多个 requirement 只映射到一次末尾 smoke test。

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 | 1 | SAFE-01 | T-01 kind-confusion / stale-lineage | 六种 kind 封闭校验，exact digest lineage 拒绝替换、错序与 `latest` discovery | unit + negative | `./safety/scripts/test.sh task artifact-contracts` | ❌ W0 | ⬜ pending |
| 01-01-02 | 01 | 1 | SAFE-02, SAFE-03 | T-02 identity-leak / raw-output-leak | logical refs、safe diagnostics、bounded capture 与 store/stdout/stderr 共用 pre-output privacy gate | unit + canary | `./safety/scripts/test.sh task privacy-boundary` | ❌ W0 | ⬜ pending |
| 01-01-03 | 01 | 1 | SAFE-08 | T-03 duplicate-owner / destructive-default | typed control-plane contract 保留分层 owner，extra state 只报告且没有真实 apply/cleanup route | unit + structural | `./safety/scripts/test.sh task controlplane-contract` | ❌ W0 | ⬜ pending |
| 01-02-01 | 02 | 2 | SAFE-04 | T-04 fixture-escape / ambient-env | fresh external root、空白 allowlisted env、marker/TTL/ownership containment 和默认删除 | integration + negative | `./safety/scripts/test.sh task fixture-lifecycle` | ❌ W0 | ⬜ pending |
| 01-02-02 | 02 | 2 | SAFE-05, SAFE-06 | T-05 implicit-network / privilege-escalation | 默认 offline、tier 不自动升级、exact test-ID network manifest，live-check 只验证 deny/allowlist contract | unit + integration | `./safety/scripts/test.sh task fixture-lifecycle` | ❌ W0 | ⬜ pending |
| 01-02-03 | 02 | 2 | SAFE-07 | T-06 incomplete-observation / overclaim | protected manifest、before/after evidence、四态 verdict 与 exact scoped claim | unit + integration | `./safety/scripts/test.sh task sentinel-verdicts` | ❌ W0 | ⬜ pending |
| 01-03-01 | 03 | 2 | SAFE-01–SAFE-08 | T-01–T-06 | CLI 在 external fixture 中完成 synthetic desired→observed→plan→receipt→evidence→report 的真实 read/write 纵切，fixture 外零写入 | end-to-end | `./safety/scripts/test.sh task walking-skeleton` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `safety/go.mod` — stdlib-only Go module；不得包含会触发网络下载的第三方依赖。
- [ ] `safety/scripts/test.sh` — stable runner，先构造 external fixture 与 before sentinels，再运行 suite，最后无条件取得 after sentinels并判定 verdict。
- [ ] `safety/testdata/blueprints/walking-skeleton/` — 完全 synthetic 的 tracked blueprint；真实 worktree 只作为只读 tracked input。
- [ ] `safety/testdata/artifacts/` — 六种 valid/invalid artifact golden fixtures，包括 wrong kind/version/provenance/lineage cases。
- [ ] `safety/testdata/canaries/` 与 `safety/testdata/raw/` — 仅 synthetic 的 secret/path/identity/raw-output negative fixtures。
- [ ] `safety/manifests/protected-surfaces.v1.json` — synthetic suite 的 required/optional/excluded surfaces；不得包含真实路径、用户名或 hostname。
- [ ] `safety/manifests/network-tests.v1.json` — schema/deny fixtures，只使用 non-routable synthetic entries，Phase 1 validation 不真实联网。

Wave 0 只建立可离线运行的最小 test infrastructure；不安装 Go、不执行 live probe，也不把 configuration apply 引入 dependency graph。

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 当前 Mac 的真实 protected surfaces 在未来 allowlisted live-check 下可安全观察 | SAFE-06, SAFE-07 | Phase 1 没有足够 isolated negative evidence 来授权真实 probe；本阶段成功不得依赖该证据 | 不在 Phase 1 执行。待后续 phase 针对每个 probe 核对当前官方只读语义、建立 isolated negative fixture，再以独立 `live-check` tier 明确授权。 |
| clean Mac / VM 的 fresh-install recovery | — | 当前没有 clean Mac/VM，且不属于 Phase 1 | 明确 deferred；不得从 synthetic fixture 或当前主机推断 `fresh-install-verified`。 |

---

## Validation Sign-Off

- [ ] 所有最终 PLAN task 都有与上表一致的 `<automated>`/`<verify>` 命令，或明确依赖 Wave 0。
- [ ] Sampling continuity：不存在连续三个 task 没有 automated verify 的情况。
- [ ] Wave 0 覆盖所有标记为 `❌ W0` 的文件和 runner dependency。
- [ ] 所有命令无 watch mode、无 network、无 manager bootstrap、无 live HOME/worktree write。
- [ ] 每条 task/wave/phase 命令都由 sentinel 包住完整 observation window；non-pass verdict 一律 non-zero。
- [ ] Full phase gate 的 feedback latency 小于 60 秒，且不会通过 retry、optional downgrade 或 live escalation 变绿。
- [ ] 实现与计划检查通过后将 frontmatter 的 `nyquist_compliant` 与 `wave_0_complete` 更新为真实状态；规划阶段不得预先伪标绿色。

**Approval:** pending — 由 plan checker 验证 task IDs、requirements、threat refs 与 command continuity 后批准。
