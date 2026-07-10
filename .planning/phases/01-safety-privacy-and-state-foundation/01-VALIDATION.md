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
| **Runner hard ceilings** | task 15 秒；wave 47 秒；offline phase 305 秒 |

Runner 必须固定 `GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`、`GOENV=off`、`GOWORK=off`、`CGO_ENABLED=0`，并把 `HOME`、全部 `XDG_*`、`TMPDIR`、`GOCACHE`、`GOMODCACHE`、manager roots 与 artifact store 指向仓库外的 fresh fixture root。缺少本地 Go toolchain 时返回 bounded `manual-required`，不得调用 Nix、Homebrew、mise、uv、rustup 或其他 manager 自动补齐。

Task hard ceiling 为 15 秒。Wave 最多串行聚合三个固定 task child，每个 child 使用独立 fresh external root/store/cache/key 并拥有自己的 15 秒 hard deadline；wave 不在 child 外再创建 nested process-group deadline wrapper。Wave 的 47 秒 hard ceiling 按 `3 * 15 + 2` 组成：启动 child 前若剩余时间少于完整 15 秒，立即输出 bounded `runner-deadline-exceeded` 并 exit 124，不能启动该 child；child 完成后再次校验总 elapsed。任何 list、behavior、multi-package/composite、wave 或 phase 层观察到 124，都必须原样传播为 bounded JSON reason `runner-deadline-exceeded` + exit 124，不能改写成 selection failure、expected RED 或 contract failure。

---

## Sampling Rate

- **After every task commit:** 运行该任务的 `./safety/scripts/test.sh task <task-suite>`；除首个 RED-contract wrapper 外，suite 必须为 green。Wave 1 synthetic sentinel 的应用层成功状态必须精确为 `synthetic-sentinel-passed`，不得发出真实表面 claim；RED wrapper 仅在观察到预期失败原因时返回 0。
- **After every plan wave:** 按下表运行对应 wave suite；每个 task 使用新的 external root/store/sentinel key/cache，不复用 fixture ID、run ID 或 `latest` selection。Wave 只串行调用 fixed child，child 自带 15 秒 hard deadline，父 wave 只执行 47 秒 pre-start reservation 与 post-child elapsed gate。
- **Before `$gsd-verify-work`:** 运行 `./safety/scripts/test.sh phase`；只有完整 offline isolated suite 与五个 required 外层真实 before/after sentinels 为绿色且 claim 精确等于 `covered-surfaces-unchanged-for-run` 才可继续。
- **Hard feedback ceilings:** task 15 秒、wave 47 秒、phase 305 秒；超限统一为 bounded `runner-deadline-exceeded` + exit 124，不能被 selection/behavior/composite 层吞掉或改写，也不能转 live-check、重试到通过或把 required sentinel 降级为 optional。

### Incremental Runner Route Ownership

- 每个 owning task 必须把 `safety/scripts/test.sh` 纳入自己的 `<files>`、exact staged whitelist、cached diff-check、targeted privacy scan 与 staged Gitleaks；先用 `/bin/bash -n` 验证语法，再运行该 task route。提交前必须从相对 task parent 的 staged diff 中提取所有新增 case-label 行，并与下表该 task 的 exact set 完全相等；任何额外 task/wave/phase label 都阻断提交。
- Task route 只在对应测试文件与 exact runner-owned package/pattern 同一 task 落地时注册；计划最后一个 task 才注册该 wave route。`phase:phase)` 由 01-07-01 在 full phase E2E 落地时注册，01-07-02 才注册 docs gate 与 final wave。每个 case label 必须是单一、完整、未拆分的 literal；禁止引号拼接、变量、glob、alternation、命令替换或通用 dispatcher。
- Owner-time no-preregistration 是当前 task staged/parent diff 的 temporal commit evidence。长期 Go regression 不得把任何 planned future route name 写成 absence array；后续 owner task 落地后，该名称可以按下表合法成为 literal case label。
- Lifetime route regressions 只使用永久保留的 `never-registered-task`、`never-registered-wave`、`never-registered-scope` 与 malformed `phase unexpected-argument`。Task/wave probes 必须 bounded non-zero `harness-error/unsupported-suite`，scope/phase probes必须 bounded non-zero dispatch/usage rejection，且均不得含或满足 expected RED。每个 declared package/pattern pair 仍须精确选中一个 top-level test；零选中、同一 pair 多选中、wrong package 以及任何从用户 suite 派生 command/package/pattern 都是 non-zero `harness-error`。
- Wave/phase aggregation 只引用已经落地的 exact handlers，并按下表为 child handler 分配 fresh external root/store/key/cache；失败不得取得 network、live-check、manager、shell 或 arbitrary-command 能力，`unsupported-suite`、selection failure、runner deadline 或 harness setup failure 不得充当 TDD RED。Wave 与 phase 均不在已经拥有 hard deadline 的 child 外再创建 nested process group；启动 child 前必须为其保留完整 hard budget，完成后再校验父 scope elapsed。
- `phase:phase)` 的 exact child set 与顺序固定为 `wave skeleton` → `wave artifact-contracts` → `wave privacy` → `wave fixture-policy` → `wave sentinels` → `wave controlplane` → `task phase-e2e`。它不得调用 `task docs-and-phase-gate` 或 `wave phase-integration`。六个 component wave 各占 47 秒、最后 task 占 15 秒、phase composition overhead 为 8 秒，因此 phase hard ceiling 精确为 `6 * 47 + 15 + 8 = 305` 秒；启动 component wave 前 remaining 少于 47 秒或启动 `phase-e2e` 前 remaining 少于 15 秒时，立即以 `runner-deadline-exceeded` + 124 停止且不启动 child。
- `task docs-and-phase-gate` 只运行固定 docs/symlink/structural checks，不内嵌 `phase`。`wave phase-integration` 只串行聚合 `task phase-e2e` 与 `task docs-and-phase-gate`，两者各用 fresh root/cache 与 15 秒 hard deadline；完整 phase 由独立的 `./safety/scripts/test.sh phase` command 验证，避免重复执行 phase。

| Owning task | Exact literal case-label lines first allowed after this task | Fixed package/pattern or structural target |
|-------------|----------------------------------------------------------|--------------------------------------------|
| `01-02-01` | `{task:artifact-kinds)}` | `./internal/artifact` + `^TestArtifactKinds$` |
| `01-02-02` | `{task:artifact-lineage), wave:artifact-contracts)}` | `./internal/e2e` + `^TestArtifactLineage$`; wave aggregates completed Phase 2 handlers |
| `01-03-01` | `{task:privacy-boundary)}` | `./internal/privacy` + `^TestPrivacyBoundary$` |
| `01-03-02` | `{task:bounded-capture), wave:privacy)}` | `./internal/privacy` + `^TestBoundedCapture$`; `./internal/e2e` + `^TestPrivacyCLI$` |
| `01-04-01` | `{task:fixture-lifecycle)}` | `./internal/fixture` + `^TestFixtureLifecycle$` |
| `01-04-02` | `{task:tier-network-policy), wave:fixture-policy)}` | `./internal/fixture` + `^TestTierNetworkPolicy$`; `./internal/e2e` + `^TestTierCLI$`; reserved generic/injection task/wave/scope and malformed phase deny |
| `01-05-01` | `{task:sentinel-manifest)}` | `./internal/sentinel` + `^TestSentinelManifest$` |
| `01-05-02` | `{task:sentinel-verdicts)}` | `./internal/sentinel` + `^TestSentinelVerdicts$`; `./internal/e2e` + `^TestSentinelCLI$` |
| `01-05-03` | `{task:real-sentinel-envelope), wave:sentinels)}` | `./internal/sentinel` + `^TestRealSentinelEnvelope$`; `./internal/e2e` + `^TestRealSentinelCLI$` |
| `01-06-01` | `{task:controlplane-contract)}` | `./internal/contract` + `^TestControlPlaneContract$`; `./internal/e2e` + `^TestControlPlaneCLI$` |
| `01-06-02` | `{task:no-destructive-defaults), wave:controlplane)}` | `./internal/contract` + `^TestNoDestructiveDefaults$`; `./internal/e2e` + `^TestNoCleanupCLI$` |
| `01-07-01` | `{task:phase-e2e), phase:phase)}` | `./internal/e2e` + `^TestPhaseE2E$`; phase uses the exact six-wave-then-`phase-e2e` child set with a 305-second hard ceiling |
| `01-07-02` | `{task:docs-and-phase-gate), wave:phase-integration)}` | fixed docs/symlink/structural checks only; final wave aggregates exactly `phase-e2e` and `docs-and-phase-gate` without invoking phase |

| Wave | Plan | Wave suite | Purpose |
|------|------|------------|---------|
| 1 | 01-01 | `./safety/scripts/test.sh wave skeleton` | RED→GREEN external walking skeleton；synthetic-only status, no real claim |
| 2 | 01-02 | `./safety/scripts/test.sh wave artifact-contracts` | closed six-kind schemas, canonical digest and lineage |
| 3 | 01-03 | `./safety/scripts/test.sh wave privacy` | logical refs, safe errors and bounded capture |
| 4 | 01-04 | `./safety/scripts/test.sh wave fixture-policy` | fixture lifecycle, tiers and network/live deny policy |
| 5 | 01-05 | `./safety/scripts/test.sh wave sentinels` | protected manifests, snapshots, verdicts and claim |
| 6 | 01-06 | `./safety/scripts/test.sh wave controlplane` | layered ownership and report-only extra state |
| 7 | 01-07 | `./safety/scripts/test.sh wave phase-integration` | Phase 7 E2E task plus structural docs gate；full phase runs separately |

---

## Per-Task Verification Map

Every Automated Command below inherits the 15-second task hard ceiling. A deadline is a runner-level `harness-error` with bounded reason `runner-deadline-exceeded` and exit 124; it leaves the corresponding Nyquist row unverified and can never satisfy expected RED, selection, behavior, sentinel-verdict, or contract-failure acceptance. Wave rows inherit the 47-second hard ceiling and the same unchanged 124 propagation; the separate full-phase gate inherits the exact 305-second composition above.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 | 1 | SAFE-01, SAFE-04 | T-01, T-04, T-06 | 外部 E2E 先 RED；GREEN 只接受 `synthetic-sentinel-passed` 并拒绝真实表面/whole-Mac/current-host claim | E2E RED contract | `./safety/scripts/test.sh task walking-skeleton-red` | `safety/internal/e2e/walking_skeleton_test.go` ❌ W0 | ⬜ pending |
| 01-01-02 | 01 | 1 | SAFE-01, SAFE-04 | T-01, T-04, T-06 | CLI 在 synthetic sentinels 内写入六种 artifact，成功状态只是 `synthetic-sentinel-passed` | E2E integration | `./safety/scripts/test.sh task walking-skeleton` | `safety/internal/workflow/synthetic.go` ❌ W0 | ⬜ pending |
| 01-02-01 | 02 | 2 | SAFE-01 | T-01 | 六 kind + closed class-kind-retention；wrong-class/unsupported-retention 整体拒绝 | unit + negative | `/bin/bash -n safety/scripts/test.sh && ./safety/scripts/test.sh task artifact-kinds` | `safety/internal/artifact/validate_test.go` ❌ W0 | ⬜ pending |
| 01-02-02 | 02 | 2 | SAFE-01 | T-01, T-06 | exact lineage、24h snapshots、plan terminal state、transitive pins；拒绝 stale/overwrite/premature delete | unit + CLI negative | `/bin/bash -n safety/scripts/test.sh && ./safety/scripts/test.sh task artifact-lineage` | `safety/internal/artifact/lineage.go` ❌ W0 | ⬜ pending |
| 01-03-01 | 03 | 3 | SAFE-02, SAFE-03 | T-02, T-04 | 六 namespace + closed surface_domain compatibility 在 store/stdout/stderr 前共用 gate；wrong-domain/resolver escape hard-fail | unit + canary | `/bin/bash -n safety/scripts/test.sh && ./safety/scripts/test.sh task privacy-boundary` | `safety/internal/privacy/gate_test.go` ❌ W0 | ⬜ pending |
| 01-03-02 | 03 | 3 | SAFE-03 | T-02, T-05 | fake adapter raw output 有界、内存内、strict parse/discard；无 shell/raw retention | unit + integration | `/bin/bash -n safety/scripts/test.sh && ./safety/scripts/test.sh task bounded-capture` | `safety/internal/privacy/capture_test.go` ❌ W0 | ⬜ pending |
| 01-04-01 | 04 | 4 | SAFE-04 | T-04 | fresh external root、空白 allowlisted env、marker/TTL/ownership、默认删除 | integration + negative | `/bin/bash -n safety/scripts/test.sh && ./safety/scripts/test.sh task fixture-lifecycle` | `safety/internal/fixture/fixture_test.go` ❌ W0 | ⬜ pending |
| 01-04-02 | 04 | 4 | SAFE-05, SAFE-06 | T-05 | tier 不升级、exact network ID deny/manual；reserved generic/injection task/wave/scope、malformed phase、zero/multiple selection 均不能取得 live-check | unit + CLI negative | `/bin/bash -n safety/scripts/test.sh && ./safety/scripts/test.sh task tier-network-policy` | `safety/internal/fixture/network_test.go` ❌ W0 | ⬜ pending |
| 01-05-01 | 05 | 5 | SAFE-07 | T-02, T-04, T-06 | 五 domain manifest 必须使用六 namespace compatibility table；synthetic adapters 仅为 test doubles | unit + integration | `/bin/bash -n safety/scripts/test.sh && ./safety/scripts/test.sh task sentinel-manifest` | `safety/internal/sentinel/sentinel_test.go` ❌ W0 | ⬜ pending |
| 01-05-02 | 05 | 5 | SAFE-07 | T-06 | 四态 non-pass exits；synthetic-only evidence 不能 claim | unit + E2E | `/bin/bash -n safety/scripts/test.sh && ./safety/scripts/test.sh task sentinel-verdicts` | `safety/internal/sentinel/verdict_test.go` ❌ W0 | ⬜ pending |
| 01-05-03 | 05 | 5 | SAFE-06, SAFE-07 | T-02, T-03, T-04, T-05, T-06 | 已证明 exact 只读 adapters 对五 domain 做外层 envelope；只输出 compatible public refs/opaque tokens | integration + E2E | `/bin/bash -n safety/scripts/test.sh && ./safety/scripts/test.sh task real-sentinel-envelope` | `safety/internal/e2e/real_sentinel_cli_test.go` ❌ W0 | ⬜ pending |
| 01-06-01 | 06 | 6 | SAFE-08 | T-03 | Determinate/nix-darwin/HM 分层与 one-owner typed contract，无 live inspection | unit + CLI | `/bin/bash -n safety/scripts/test.sh && ./safety/scripts/test.sh task controlplane-contract` | `safety/internal/contract/controlplane_test.go` ❌ W0 | ⬜ pending |
| 01-06-02 | 06 | 6 | SAFE-08 | T-03 | extra/unmanaged report-only；cleanup/uninstall/zap/runtime-delete/apply route 拒绝 | structural + E2E | `/bin/bash -n safety/scripts/test.sh && ./safety/scripts/test.sh task no-destructive-defaults` | `safety/internal/e2e/no_cleanup_cli_test.go` ❌ W0 | ⬜ pending |
| 01-07-01 | 07 | 7 | SAFE-01–SAFE-08 | T-01–T-06 | exact suite 串起所有 contracts，fresh external full phase 与 cross-cut negative matrix；phase 固定六 wave 后接 `phase-e2e`，hard ceiling 305 秒 | full E2E | `/bin/bash -n safety/scripts/test.sh && ./safety/scripts/test.sh task phase-e2e` | `safety/internal/e2e/phase_e2e_test.go` ❌ W0 | ⬜ pending |
| 01-07-02 | 07 | 7 | SAFE-01–SAFE-08 | T-01–T-06 | docs/symlink/structural checks 不内嵌 phase；final wave 仅聚合两个 Phase 7 task，完整 phase 独立运行 | structural | `/bin/bash -n safety/scripts/test.sh && ./safety/scripts/test.sh task docs-and-phase-gate` | `safety/AGENTS.md` ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `safety/go.mod` — stdlib-only Go module；不得包含第三方依赖或网络 bootstrap。
- [ ] `safety/scripts/test.sh` — incrementally owned task/wave/phase runner；每个 owning task 只注册本 task exact route，计划末 task 才注册 wave，01-07-01 注册 phase，01-07-02 注册 docs/final-wave；package/pattern/structural commands 全为 runner-owned literals，并保留 external roots、before/after sentinels、after-on-failure、fixed offline Go env 与 bounded manual-required。Task/wave/phase hard ceilings 分别为 15/47/305 秒；fixed child 自带 deadline，父 aggregation 不叠加 nested process group，pre-start reservation 与 post-child elapsed gate 都 fail closed，timeout 统一 bounded `runner-deadline-exceeded` + exit 124。
- [ ] `safety/testdata/blueprints/walking-skeleton/` — 完全 synthetic 的 tracked blueprint 与 expected report；真实 worktree 只作为只读 tracked input。
- [ ] `safety/testdata/artifacts/` — 六 kind valid/invalid、wrong kind/version/provenance/digest/lineage/latest cases。
- [ ] `safety/testdata/canaries/` 与 `safety/testdata/raw/` — 仅 synthetic secret/path/identity/provider/env/raw negative samples。
- [ ] `safety/manifests/protected-surfaces.v1.json` — 五个真实 domain 的 required/optional/excluded scope；refs 仅使用 `repo:`/`home:`/`profile:` compatibility shapes，无物理路径/身份，synthetic blueprint 另存且不可 claim。
- [ ] `safety/manifests/real-adapters.v1.json` — 五个 exact adapter/version 的官方只读语义、review/expiry、隔离负证据 digest 与 bounds；只含 public logical metadata。
- [ ] `safety/manifests/network-tests.v1.json` — exact-ID schema/deny fixtures，仅 `example.invalid`，不真实联网。
- [ ] `safety/manifests/offline-suite.v1.json` — exact task/manifest/tier/claim bindings for full phase.

Wave 0 由 01-01-01 建立 runner/RED contract，并由后续 owning tasks 在各自 vertical capability 中增量加入 exact literal route 与所需 fixture/manifest。某个 planned route 在 owner task 之前未注册，只由该 task parent→staged/commit diff 的 exact added-label set 证明；不得把其名称写进永久 absence regression。Lifetime `unsupported-suite` 证据只来自 reserved generic unknown task/wave probes。规划阶段仍保持 `wave_0_complete: false`；只有实现文件存在、route 精确选中合同测试且对应 task suites 实际通过后才能改为 true。

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
- [ ] 01-02～01-07 的 13 个 owning task 都在 frontmatter/task `<files>`、exact staged whitelist、cached diff-check、targeted privacy scan 与 staged Gitleaks 中声明 `safety/scripts/test.sh`；每份 plan 的 frontmatter file set、task file union 与 plan-level diff-check 相等。
- [ ] 每个 task route 只在 owner task 注册；每个 wave route 只在该 plan 最后 task 注册；`phase:phase)` 只在 01-07-01 注册，docs/final wave 只在 01-07-02 注册。11 个 01-03～01-07 owner task 的 staged/parent diff 新增 case-label set 与上表完全相等；所有 label 都是单一完整 literal，且无引号拼接、变量、glob、alternation、命令替换或通用 dispatcher。长期测试不含 planned future-name absence array。
- [ ] Reserved generic unknown task/wave probes bounded non-zero `harness-error/unsupported-suite`，reserved scope 与 malformed phase bounded non-zero dispatch/usage rejection；zero-selected、wrong-package、multiple-match 均 non-zero。Runner timeout 在 list、behavior、multi-package/composite、wave 与 phase 层都保持 bounded `runner-deadline-exceeded` + exit 124；任何这些失败都不得满足 RED 或被改写成 contract failure。
- [ ] 所有 Wave 0 files 由明确 task 创建且在实现前保持 `❌ W0`。
- [ ] 所有命令无 watch、network、manager bootstrap、live HOME/worktree write、Nix/Homebrew/app/defaults/link/trust 或 destructive/convergence cleanup；只有 exact outer read-only sentinels 与 post-verdict marker-owned fixture teardown 例外。
- [ ] 每个 task child 使用 fresh root/store/key/cache；task/wave/phase hard ceilings 为 15/47/305 秒。Wave/phase 无 nested process-group wrapper，启动 child 前保留完整 15/47 秒 child budget并在完成后校验 elapsed；non-pass sentinel 与 deadline 一律 non-zero。
- [ ] 01-04-02 只使用 reserved generic/injection-shaped task/wave/scope 与 malformed phase probes，不使用任何未来会合法的 planned route name，并证明 suite 字符串无法选择 `live-check`、任意 package/pattern、shell 或 arbitrary command。
- [ ] Full phase 的 exact child set 为六个固定 component wave 后接 `task phase-e2e`，不包含 docs gate/final wave；hard ceiling 为 `6 * 47 + 15 + 8 = 305` 秒，且不会通过 retry、optional downgrade、live escalation 或 cleanup 变绿。
- [ ] 外层顺序固定为 real-before → isolated workload/inner synthetic → freeze primary verdict → marker teardown unless pre-run keep → real-after → monotonic combine；teardown/after 失败不能把 non-pass 变 pass。
- [ ] Wave 1 精确输出 `synthetic-sentinel-passed` 并拒绝任何真实表面 claim；只有完整外层真实 evidence 的 full phase 可输出 `covered-surfaces-unchanged-for-run`，whole-Mac/current-host/fresh-install overclaim 负测试通过。
- [ ] 实现与 plan checker 通过后，才把 `nyquist_compliant` 与 `wave_0_complete` 更新为真实状态；规划阶段不得预先标绿。

**Approval:** pending — 由 plan checker 验证 task IDs、requirements、D-01..D-19、T-01..T-06、commands 与 source-audit continuity 后批准。
