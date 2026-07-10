---
phase: 1
slug: safety-privacy-and-state-foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-10
---

# Phase 1 — Validation Strategy

> Phase 1 的反馈采样合同。写入只能发生在仓库外 fresh synthetic fixture/local-state root；默认 phase gate 额外允许五个 exact-manifest、已证明、有界的真实只读外层 sentinels。禁止安装、下载、激活、修复、联网、functional discovery、live-check 或主机 mutation。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard library `testing` + repository-owned strict Bash runner |
| **Config file** | `safety/go.mod`；Phase 1 不引入第三方依赖 |
| **Quick run command** | `./safety/scripts/test.sh task <task-suite>` |
| **Wave run command** | `./safety/scripts/test.sh wave <wave-suite>` |
| **Full suite command** | `./safety/scripts/test.sh phase` |
| **Estimated runtime** | task < 10 秒；wave < 30 秒；offline phase < 60 秒 |

Runner 必须固定 `GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`、`GOENV=off`、`GOWORK=off`、`CGO_ENABLED=0`，并把 `HOME`、全部 `XDG_*`、`TMPDIR`、`GOCACHE`、`GOMODCACHE`、manager roots 与 artifact store 指向仓库外的 fresh fixture root。缺少本地 Go toolchain 时返回 bounded `manual-required`，不得调用 Nix、Homebrew、mise、uv、rustup 或其他 manager 自动补齐。

---

## Sampling Rate

- **After every task commit:** 运行该任务的 `./safety/scripts/test.sh task <task-suite>`；除首个 RED-contract wrapper 外，suite 必须为 green。Wave 1 synthetic sentinel 的应用层成功状态必须精确为 `synthetic-sentinel-passed`，不得发出真实表面 claim；RED wrapper 仅在观察到预期失败原因时返回 0。
- **After every plan wave:** 按下表运行对应 wave suite；每个 task 使用新的 external root/store/sentinel key，不复用 cache、fixture ID、run ID 或 `latest` selection。
- **Before `$gsd-verify-work`:** 运行 `./safety/scripts/test.sh phase`；只有完整 offline isolated suite 与五个 required 外层真实 before/after sentinels 为绿色且 claim 精确等于 `covered-surfaces-unchanged-for-run` 才可继续。
- **Max feedback latency:** task 10 秒、wave 30 秒、phase 60 秒；超限 non-zero，不能转 live-check、重试到通过或把 required sentinel 降级为 optional。

| Wave | Plan | Wave suite | Purpose |
|------|------|------------|---------|
| 1 | 01-01 | `./safety/scripts/test.sh wave skeleton` | RED→GREEN external walking skeleton；synthetic-only status, no real claim |
| 2 | 01-02 | `./safety/scripts/test.sh wave artifact-contracts` | closed six-kind schemas, canonical digest and lineage |
| 3 | 01-03 | `./safety/scripts/test.sh wave privacy` | logical refs, safe errors and bounded capture |
| 4 | 01-04 | `./safety/scripts/test.sh wave fixture-policy` | fixture lifecycle, tiers and network/live deny policy |
| 5 | 01-05 | `./safety/scripts/test.sh wave sentinels` | protected manifests, snapshots, verdicts and claim |
| 6 | 01-06 | `./safety/scripts/test.sh wave controlplane` | layered ownership and report-only extra state |
| 7 | 01-07 | `./safety/scripts/test.sh wave phase-integration` | exact full phase integration and docs gate |

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 | 1 | SAFE-01, SAFE-04 | T-01, T-04, T-06 | 外部 E2E 先 RED；GREEN 只接受 `synthetic-sentinel-passed` 并拒绝真实表面/whole-Mac/current-host claim | E2E RED contract | `./safety/scripts/test.sh task walking-skeleton-red` | `safety/internal/e2e/walking_skeleton_test.go` ❌ W0 | ⬜ pending |
| 01-01-02 | 01 | 1 | SAFE-01, SAFE-04 | T-01, T-04, T-06 | CLI 在 synthetic sentinels 内写入六种 artifact，成功状态只是 `synthetic-sentinel-passed` | E2E integration | `./safety/scripts/test.sh task walking-skeleton` | `safety/internal/workflow/synthetic.go` ❌ W0 | ⬜ pending |
| 01-02-01 | 02 | 2 | SAFE-01 | T-01 | 六 kind + closed class-kind-retention；wrong-class/unsupported-retention 整体拒绝 | unit + negative | `./safety/scripts/test.sh task artifact-kinds` | `safety/internal/artifact/validate_test.go` ❌ W0 | ⬜ pending |
| 01-02-02 | 02 | 2 | SAFE-01 | T-01, T-06 | exact lineage、24h snapshots、plan terminal state、transitive pins；拒绝 stale/overwrite/premature delete | unit + CLI negative | `./safety/scripts/test.sh task artifact-lineage` | `safety/internal/artifact/lineage.go` ❌ W0 | ⬜ pending |
| 01-03-01 | 03 | 3 | SAFE-02, SAFE-03 | T-02, T-04 | 六 namespace + closed surface_domain compatibility 在 store/stdout/stderr 前共用 gate；wrong-domain/resolver escape hard-fail | unit + canary | `./safety/scripts/test.sh task privacy-boundary` | `safety/internal/privacy/gate_test.go` ❌ W0 | ⬜ pending |
| 01-03-02 | 03 | 3 | SAFE-03 | T-02, T-05 | fake adapter raw output 有界、内存内、strict parse/discard；无 shell/raw retention | unit + integration | `./safety/scripts/test.sh task bounded-capture` | `safety/internal/privacy/capture_test.go` ❌ W0 | ⬜ pending |
| 01-04-01 | 04 | 4 | SAFE-04 | T-04 | fresh external root、空白 allowlisted env、marker/TTL/ownership、默认删除 | integration + negative | `./safety/scripts/test.sh task fixture-lifecycle` | `safety/internal/fixture/fixture_test.go` ❌ W0 | ⬜ pending |
| 01-04-02 | 04 | 4 | SAFE-05, SAFE-06 | T-05 | tier 不升级、exact network ID deny/manual contract、phase gate 无 live-check | unit + CLI negative | `./safety/scripts/test.sh task tier-network-policy` | `safety/internal/fixture/network_test.go` ❌ W0 | ⬜ pending |
| 01-05-01 | 05 | 5 | SAFE-07 | T-02, T-04, T-06 | 五 domain manifest 必须使用六 namespace compatibility table；synthetic adapters 仅为 test doubles | unit + integration | `./safety/scripts/test.sh task sentinel-manifest` | `safety/internal/sentinel/sentinel_test.go` ❌ W0 | ⬜ pending |
| 01-05-02 | 05 | 5 | SAFE-07 | T-06 | 四态 non-pass exits；synthetic-only evidence 不能 claim | unit + E2E | `./safety/scripts/test.sh task sentinel-verdicts` | `safety/internal/sentinel/verdict_test.go` ❌ W0 | ⬜ pending |
| 01-05-03 | 05 | 5 | SAFE-06, SAFE-07 | T-02, T-03, T-04, T-05, T-06 | 已证明 exact 只读 adapters 对五 domain 做外层 envelope；只输出 compatible public refs/opaque tokens | integration + E2E | `./safety/scripts/test.sh task real-sentinel-envelope` | `safety/internal/e2e/real_sentinel_cli_test.go` ❌ W0 | ⬜ pending |
| 01-06-01 | 06 | 6 | SAFE-08 | T-03 | Determinate/nix-darwin/HM 分层与 one-owner typed contract，无 live inspection | unit + CLI | `./safety/scripts/test.sh task controlplane-contract` | `safety/internal/contract/controlplane_test.go` ❌ W0 | ⬜ pending |
| 01-06-02 | 06 | 6 | SAFE-08 | T-03 | extra/unmanaged report-only；cleanup/uninstall/zap/runtime-delete/apply route 拒绝 | structural + E2E | `./safety/scripts/test.sh task no-destructive-defaults` | `safety/internal/e2e/no_cleanup_cli_test.go` ❌ W0 | ⬜ pending |
| 01-07-01 | 07 | 7 | SAFE-01–SAFE-08 | T-01–T-06 | exact suite 串起所有 contracts，fresh external full phase 与 cross-cut negative matrix | full E2E | `./safety/scripts/test.sh task phase-e2e` | `safety/internal/e2e/phase_e2e_test.go` ❌ W0 | ⬜ pending |
| 01-07-02 | 07 | 7 | SAFE-01–SAFE-08 | T-01–T-06 | 文档/实际 symlink/phase gate/隐私审查一致，无 broad ignore/allowlist | structural + full phase | `./safety/scripts/test.sh task docs-and-phase-gate` | `safety/AGENTS.md` ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `safety/go.mod` — stdlib-only Go module；不得包含第三方依赖或网络 bootstrap。
- [ ] `safety/scripts/test.sh` — stable task/wave/phase runner；external roots、before/after sentinels、after-on-failure、fixed offline Go env 与 bounded manual-required。
- [ ] `safety/testdata/blueprints/walking-skeleton/` — 完全 synthetic 的 tracked blueprint 与 expected report；真实 worktree 只作为只读 tracked input。
- [ ] `safety/testdata/artifacts/` — 六 kind valid/invalid、wrong kind/version/provenance/digest/lineage/latest cases。
- [ ] `safety/testdata/canaries/` 与 `safety/testdata/raw/` — 仅 synthetic secret/path/identity/provider/env/raw negative samples。
- [ ] `safety/manifests/protected-surfaces.v1.json` — 五个真实 domain 的 required/optional/excluded scope；refs 仅使用 `repo:`/`home:`/`profile:` compatibility shapes，无物理路径/身份，synthetic blueprint 另存且不可 claim。
- [ ] `safety/manifests/real-adapters.v1.json` — 五个 exact adapter/version 的官方只读语义、review/expiry、隔离负证据 digest 与 bounds；只含 public logical metadata。
- [ ] `safety/manifests/network-tests.v1.json` — exact-ID schema/deny fixtures，仅 `example.invalid`，不真实联网。
- [ ] `safety/manifests/offline-suite.v1.json` — exact task/manifest/tier/claim bindings for full phase.

Wave 0 由 01-01-01 建立 runner/RED contract，并由后续 tasks 在各自 vertical capability 中补齐所需 fixture/manifest。规划阶段仍保持 `wave_0_complete: false`；只有实现文件存在且对应 task suites 实际通过后才能改为 true。

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 外层真实 sentinel 缺少当前证明 | SAFE-06, SAFE-07 | 缺失/过期官方只读语义或 exact isolated negative evidence 时不能自动 claim | 默认 phase gate 必须 non-zero `indeterminate`/`manual-required`；不降级为 synthetic，不改用 live-check。 |
| clean Mac / VM fresh-install recovery | — | 当前没有 clean Mac/VM，且属于 future milestone | 不执行；不得从 synthetic fixture 或 current host 推断 `fresh-install-verified`。 |

---

## Validation Sign-Off

- [ ] 所有 15 个 PLAN task ID、plan/wave、requirement、threat ref、suite command 与本表一致。
- [ ] 每个 task 有 `<automated>`；不存在连续 task sampling gap。
- [ ] 所有 Wave 0 files 由明确 task 创建且在实现前保持 `❌ W0`。
- [ ] 所有命令无 watch、network、manager bootstrap、live HOME/worktree write、Nix/Homebrew/app/defaults/link/trust 或 destructive/convergence cleanup；只有 exact outer read-only sentinels 与 post-verdict marker-owned fixture teardown 例外。
- [ ] 每个 task/wave/phase command 使用 fresh root；non-pass sentinel 一律 non-zero。
- [ ] Full phase 小于 60 秒，且不会通过 retry、optional downgrade、live escalation 或 cleanup 变绿。
- [ ] 外层顺序固定为 real-before → isolated workload/inner synthetic → freeze primary verdict → marker teardown unless pre-run keep → real-after → monotonic combine；teardown/after 失败不能把 non-pass 变 pass。
- [ ] Wave 1 精确输出 `synthetic-sentinel-passed` 并拒绝任何真实表面 claim；只有完整外层真实 evidence 的 full phase 可输出 `covered-surfaces-unchanged-for-run`，whole-Mac/current-host/fresh-install overclaim 负测试通过。
- [ ] 实现与 plan checker 通过后，才把 `nyquist_compliant` 与 `wave_0_complete` 更新为真实状态；规划阶段不得预先标绿。

**Approval:** pending — 由 plan checker 验证 task IDs、requirements、D-01..D-19、T-01..T-06、commands 与 source-audit continuity 后批准。
