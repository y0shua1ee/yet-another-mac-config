# Roadmap: Yet Another Mac Config

## Overview

v1 先建立可以独立证明“不泄露、不越界、不修改真实环境”的安全基础，再提供只读的工具所有权解释能力，并按 Node、Go、Python、Rust、Deno/Bun、JVM 六个生态逐一交付完整治理契约。工具链边界稳定后，项目才进入多主机组合、恢复状态观察、符号链接安全和受确认的恢复控制面，最后在当前唯一一台 Mac 上完成非破坏性演练。每个阶段都是可单独验收的 Vertical MVP；当前里程碑不包含 VM 或第二台实体 Mac 的 clean-host 验证，也不以任何当前主机变更作为规划完成条件。

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions added after planning

- [ ] **Phase 1: Safety, Privacy, and State Foundation** - 提供可验证的 artifact 边界、隔离测试入口和真实状态不变证明。
- [ ] **Phase 2: Read-only Ownership Inspector** - 无副作用地解释六生态工具的唯一 owner、实际执行来源与冲突。
- [ ] **Phase 3: Node, npm, pnpm, and Corepack Governance** - 交付 Node 生态的精确项目契约、兼容例外和安全验证。
- [ ] **Phase 4: Go Governance** - 交付不会隐式下载或污染全局状态的 Go 工具链契约。
- [ ] **Phase 5: Python and uv Governance** - 交付由 uv 独占项目 Python 状态的可复现契约。
- [ ] **Phase 6: Rust and rustup Governance** - 交付由 rustup 独占工具链状态的精确 Rust 契约。
- [ ] **Phase 7: Deno and Bun Governance** - 交付按需启用、可识别多 runtime 冲突的 Deno/Bun 契约。
- [ ] **Phase 8: JVM, Maven, and Gradle Governance** - 交付 JDK 与 Wrapper 职责分离且不触发下载的 JVM 契约。
- [ ] **Phase 9: Multi-host Composition and Binder Spike** - 证明隐私安全的 shared/role/profile 组合与本机身份绑定边界。
- [ ] **Phase 10: Recovery Observation and Readiness** - 无副作用地投影期望状态、观察命名目标并生成安全 readiness 报告。
- [ ] **Phase 11: Fail-closed Symlink Safety** - 用显式 manifest、data-only plan 和可恢复替换消除链接部署风险。
- [ ] **Phase 12: Integrated Recovery Engine** - 交付受 exact-plan 确认约束的 `check → plan → confirm → apply → verify` 控制面。
- [ ] **Phase 13: Current-host Non-destructive Readiness Drill** - 在当前 Mac 上生成诚实、可审计且不越权的恢复就绪证据。

## Phase Details

### Phase 1: Safety, Privacy, and State Foundation
**Goal:** 操作者拥有一套可验证的安全基础，可以区分恢复 artifacts、隔离测试并证明真实 Mac 未被改变。
**Mode:** mvp
**Depends on:** Nothing (first phase)
**Requirements:** SAFE-01, SAFE-02, SAFE-03, SAFE-04, SAFE-05, SAFE-06, SAFE-07, SAFE-08
**Success Criteria** (what must be TRUE):
  1. 操作者可以分别校验 desired state、observed state、generated plan、applied receipt、verification evidence 和 readiness report，并看到错误类型或伪装 artifact 被拒绝。
  2. 操作者运行默认测试入口后，可以从 sentinel 结果确认真实 HOME、worktree、全局工具状态、服务和仓库外状态均未发生未授权变化。
  3. 持久化输出只显示逻辑路径与隐私安全标识；秘密 canary、真实机器身份、绝对 HOME 和未经约束的原始输出会在落盘或显示前被拒绝或清除。
  4. 默认测试在隔离根目录且关闭网络、下载、安装、修复和 trust mutation；未证明安全的 probe 返回 `unknown` 或 `manual-required`，extra state 只报告而不清理。
**Plans:** 0/7 plans executed

### Phase 2: Read-only Ownership Inspector
**Goal:** 操作者可以无副作用地解释开发工具由谁声明、安装、选择并实际执行，以及迁移是否已具备安全证据。
**Mode:** mvp
**Depends on:** Phase 1
**Requirements:** OWN-01, OWN-02, OWN-03, OWN-04, OWN-05, OWN-06
**Success Criteria** (what must be TRUE):
  1. 操作者可以按 executable 与 execution context 查看唯一主 owner，并区分 manager、runtime、package manager、project dependency、system library 和 environment loader 的职责。
  2. Inspector 分开显示 `declared`、`installed`、`selected`、`executed`，并报告实际 path、版本、架构、配置优先级、override 来源和 auto-download 状态。
  3. 操作者可以看到重复候选、PATH shadowing、隐式下载、未纳管 executable 与 trust-required 状态，而检查过程不会安装、切换、删除或修复任何工具。
  4. 操作者可以比较 interactive、login、non-interactive、备用 Shell 与可安全观察的 hook/IDE 上下文，并通过 migration ledger 判断某生态应保留旧入口、允许迁移还是仍缺回滚证据。
**Plans:** TBD

### Phase 3: Node, npm, pnpm, and Corepack Governance
**Goal:** Node 项目具有唯一、精确且可隔离验证的 runtime 与 package-manager 契约，同时保留经过保护的全局 fallback。
**Mode:** mvp
**Depends on:** Phase 2
**Requirements:** NODE-01, NODE-02, NODE-03, NODE-04, NODE-05
**Success Criteria** (what must be TRUE):
  1. 操作者进入带项目契约的 fixture 时获得精确的 mise-owned Node 和随 Node 提供的 npm；离开项目后仍能使用未被修改的最小全局 fallback。
  2. 标准 Node fixture 使用精确 mise-owned pnpm、`packageManager`、唯一 lockfile 和 locked/frozen 行为，并能拒绝错误 package manager。
  3. 明确选择 Corepack compatibility variant 的 fixture 只使用精确 standalone Corepack 路径，不会同时把 pnpm 交给 mise 形成双 owner。
  4. 正常与负路径验证全部使用隔离 HOME、缓存和示例项目；结果能证明 fallback 优先级、lockfile 与 dual-owner 规则，且不访问或修改真实 Node 项目和现有 fallback。
**Plans:** TBD

### Phase 4: Go Governance
**Goal:** Go 项目可以获得可解释、不会隐式下载且不污染真实缓存的精确工具链选择。
**Mode:** mvp
**Depends on:** Phase 3
**Requirements:** GO-01, GO-02, GO-03, GO-04
**Success Criteria** (what must be TRUE):
  1. 操作者可以在 fixture 中声明精确的 mise-owned Go，并看到 `go.mod` 或 `go.work` 的 `go`、`toolchain` 与项目契约如何共同决定最终 compiler。
  2. 只读检查会报告 `GOTOOLCHAIN`、Go executable 与实际 compiler，且不会触发 toolchain 下载或执行 `go env -w`。
  3. 默认离线 fixture 使用独立 GOPATH、module cache、build cache 和 toolchain state，不读取或污染真实 Go 状态。
  4. 项目契约优先于未被修改的全局 fallback；在 fixture、本机只读证据和回滚路径完成前，现有 fallback 不会被删除或升级。
**Plans:** TBD

### Phase 5: Python and uv Governance
**Goal:** Python 项目由 uv 唯一管理 interpreter、环境与依赖状态，并能在完全隔离的条件下证明契约。
**Mode:** mvp
**Depends on:** Phase 4
**Requirements:** PY-01, PY-02, PY-03, PY-04
**Success Criteria** (what must be TRUE):
  1. 操作者可以确认 Home Manager 只提供 uv/uvx binary，而项目 interpreter、虚拟环境、依赖锁定和 Python tool 均由 uv 唯一拥有。
  2. Python fixture 能一致校验 `requires-python`、精确 `.python-version`、`pyproject.toml`、`uv.lock` 与 locked/frozen 行为。
  3. 只读检查会报告实际 Python provenance 与项目环境来源，但不会下载 Python、执行 sync，或把 system Python/共享虚拟环境当作项目 owner。
  4. Fixture 使用独立 uv cache、managed Python、tool 与项目环境，验证过程不会在真实项目中创建 `.venv` 或改变真实 Python 状态。
**Plans:** TBD

### Phase 6: Rust and rustup Governance
**Goal:** Rust 项目由 rustup 唯一解析并提供精确 toolchain、component 与 target，且验证不触碰真实 Rust 状态。
**Mode:** mvp
**Depends on:** Phase 5
**Requirements:** RUST-01, RUST-02, RUST-03, RUST-04
**Success Criteria** (what must be TRUE):
  1. 操作者可以确认 Home Manager 只提供 rustup binary，而 Rust toolchain、component、target 和 Cargo proxy 由 rustup 唯一拥有。
  2. Rust fixture 使用精确 `rust-toolchain.toml`、所需 component/target 与 Cargo lockfile，不接受浮动 `stable` 作为可复现项目契约。
  3. 只读检查会解释 command、environment、directory override、toolchain file 与 default 的优先级，并证明实际 `rustc`/`cargo` 来自预期 rustup proxy。
  4. 默认离线 fixture 隔离 `RUSTUP_HOME`、`CARGO_HOME` 与缓存，不运行 self-update，也不修改真实 global default。
**Plans:** TBD

### Phase 7: Deno and Bun Governance
**Goal:** Deno 与 Bun 项目可以按需获得精确 runtime，并在不迁移真实状态的前提下识别 lock 与多 runtime 冲突。
**Mode:** mvp
**Depends on:** Phase 6
**Requirements:** DBUN-01, DBUN-02, DBUN-03, DBUN-04, DBUN-05
**Success Criteria** (what must be TRUE):
  1. 操作者可以只为需要的项目声明精确 mise-owned Deno 或 Bun，未使用的 runtime 不会因此获得默认全局 fallback。
  2. Deno/Bun fixture 使用各自原生 config、lockfile 与 frozen/CI 规则；验证不会改写真实 lockfile、执行 upgrade 或自动迁移其他 package-manager 状态。
  3. 同时出现 Node、Deno 或 Bun 元数据时，Inspector 会显示明确的主执行路径；多个隐式 owner 或未声明路径会作为冲突报告。
  4. Fixture 隔离 cache、data、install 与 lock state；Inspector 能识别 Homebrew、mise 和 official/direct Bun shadowing，并在新路径证据与回滚完成前保留现有 direct Bun 入口。
**Plans:** TBD

### Phase 8: JVM, Maven, and Gradle Governance
**Goal:** JVM 项目可以明确区分 JDK 与 build-tool owner，并在无隐式下载的条件下验证 Wrapper 和 toolchain 选择。
**Mode:** mvp
**Depends on:** Phase 7
**Requirements:** JVM-01, JVM-02, JVM-03, JVM-04, JVM-05
**Success Criteria** (what must be TRUE):
  1. 操作者可以为项目声明精确 JDK vendor/version，且在真实项目 inventory 完成前不会被加入未经证明的通用全局 JDK fallback。
  2. Gradle 与 Maven fixture 分别只使用带 distribution checksum 的项目 Wrapper，不引入全局 Gradle 或 Maven owner。
  3. 只读检查会分别显示 Java executable、`JAVA_HOME`、Gradle daemon/toolchain 与 Maven toolchain 的实际选择和冲突。
  4. Check 不会下载 Wrapper distribution 或 JDK；integration fixture 隔离 Maven/Gradle user home，并把 multi-JDK 作为明确例外而非隐式行为。
**Plans:** TBD

### Phase 9: Multi-host Composition and Binder Spike
**Goal:** 操作者可以用隐私安全的 logical profile 组合共享配置，并通过 spike 证明本机身份绑定不会绕过 Git/Nix 边界。
**Mode:** mvp
**Depends on:** Phase 8
**Requirements:** HOST-01, HOST-02, HOST-03, HOST-04, HOST-05, HOST-06
**Success Criteria** (what must be TRUE):
  1. 公共配置可以组合 shared baseline、非敏感 role 与 logical host profile，并为当前 Mac 提供一个不暴露真实身份的逻辑 profile。
  2. 操作者检查公共 profile、Git source、plan、report 与 Nix store 后，不会发现真实用户名、hostname、序列号、稳定硬件指纹或私有 endpoint。
  3. Identity-only local binder 只把必要本机身份映射到公共 profile，并与秘密/provider binding 分离；日常路径不依赖 ignored Nix import、秘密 local flake input 或 routine `--impure`。
  4. 至少两个 synthetic logical profile 能从 clean Git source evaluation 并通过 privacy/store 检查；若 spike 失败，当前可工作的单主机 composition 保持不变。
  5. Logical profile 能表达 expected divergence 与支持平台；只有当前 Apple Silicon host class 被标为已验证，其他架构明确显示 `unverified-platform`。
**Plans:** TBD

### Phase 10: Recovery Observation and Readiness
**Goal:** 操作者可以无副作用地查看完整公共期望状态、规范化本机观察、私密缺口与诚实的 readiness 结果。
**Mode:** mvp
**Depends on:** Phase 9
**Requirements:** SECR-01, SECR-02, SECR-03, OBS-01, OBS-02, OBS-03, OBS-04, OBS-05, OBS-06
**Success Criteria** (what must be TRUE):
  1. 操作者可以查看覆盖 Nix/nix-darwin、Home Manager、Homebrew、service、Shell、六生态工具链、受跟踪应用配置、稳定 defaults、CLI、链接和人工状态的 canonical public inventory，并看到每项 owner、scope 与生命周期边界。
  2. Desired state 只来自 Git-tracked native source 与显式 manifest；Git、worktree、tracked query 或 manifest 无效时 discovery 会 fail closed，而不会扫描物理 `.config`、HOME 或 ignored/untracked 内容。
  3. Observed state 只运行已证明安全的 named-target probe 并保存规范化事实；不安全或可能执行 Brewfile `system` 的通用检查不会被当作只读证据。
  4. 操作者可以查看 provider-neutral secret obligations 和 TCC、登录、重启等人工步骤的安全状态，但检查只返回 `present`、`missing`、`manual` 或 `unknown`，不会读取秘密值、触发登录/Keychain prompt 或伪造完成。
  5. 操作者可以同时获得 sanitized JSON 与 Markdown readiness report，并区分 `verified`、`declared-unapplied`、`drift`、`private-missing`、`manual-required`、`excluded`、`unknown`、`expected-divergence` 和 `unverified-platform`。
**Plans:** TBD

### Phase 11: Fail-closed Symlink Safety
**Goal:** 操作者可以在不触碰真实 HOME 的情况下预览并验证受跟踪链接部署，且每个允许的替换都有恢复锚点。
**Mode:** mvp
**Depends on:** Phase 10
**Requirements:** LINK-01, LINK-02, LINK-03, LINK-04, LINK-05
**Success Criteria** (what must be TRUE):
  1. 所有链接入口只从受跟踪的显式 manifest 生成同一套 fail-closed 结果；Git/manifest 失败时不会回退扫描物理目录。
  2. 操作者在 data-only plan 中可以看到 source、destination、target home 与现有 owner；路径逃逸、循环、跨 owner 冲突或覆盖 Home Manager 目标会在任何写入前被拒绝。
  3. 真实链接替换必须绑定 exact plan confirmation；已有 destination 会先获得可恢复 backup/restore anchor，正常路径不使用 `rm -rf` 或递归删除。
  4. Fake-home fixture 能证明首次创建、重复运行、冲突、Git/manifest 失败、中断与 restore 的幂等或可恢复行为，且自动测试不会替换真实链接。
**Plans:** TBD

### Phase 12: Integrated Recovery Engine
**Goal:** 操作者可以生成、精确确认并按组件执行不可变恢复计划，再用新鲜观察验证结果，而不会把多个 writer 伪装成一个事务。
**Mode:** mvp
**Depends on:** Phase 11
**Requirements:** RCVR-01, RCVR-02, RCVR-03, RCVR-04, RCVR-05, RCVR-06, RCVR-07, RCVR-08
**Success Criteria** (what must be TRUE):
  1. 操作者得到的 plan 是 machine-readable、data-only、allowlisted typed operations；digest 绑定 repo revision、lockfile、logical profile、platform、schema/adapter、exact operations 与非秘密 observed fingerprint，确认只对该 digest 生效。
  2. Apply 只消费已确认 plan，不重新 discovery、追加或重排动作；repo、lock、profile、platform、target、inventory 或 before-state 变化会使 stale plan 在第一笔写入前零写入退出。
  3. Nix build/switch、Home Manager、Homebrew、service、defaults、link 与 toolchain 分别具有 confirmation/checkpoint/receipt；失败会停止后续写入，并按操作如实显示 `reversible`、`compensatable`、`forward-repair-only` 或 `manual`。
  4. Verify 会进行 fresh observation 并按 operation ID 检查 postcondition；apply 前 snapshot、退出码和 receipt 不会被当作完成验证。
**Plans:** TBD

### Phase 13: Current-host Non-destructive Readiness Drill
**Goal:** 操作者可以在当前唯一一台 Mac 上完成非破坏性演练，并获得与证据等级匹配的恢复就绪结论。
**Mode:** mvp
**Depends on:** Phase 12
**Requirements:** EVID-01, EVID-02, EVID-03, EVID-04
**Success Criteria** (what must be TRUE):
  1. 当前主机 drill 默认只执行静态检查、隔离 fixture、已证明安全的 read-only probe 与单独审查的 non-activating build；没有额外 plan 与确认时不会发生真实 mutation。
  2. 操作者可以查看主 Shell、备用 Shell、login、non-interactive 与可安全观察的 hook/IDE 证据，以及 first-run、private、login、TCC、manual、drift 和 expected-divergence 缺口。
  3. Claim validator 在只有当前 Mac 证据时最高只输出 `recovery-ready-on-current-host`，并拒绝多机或 `fresh-install-verified` 声明。
  4. 在未来 clean VM 或第二台 Mac artifact 尚未包含记录起点、exact plan、组件 receipts、fresh verification、manual/excluded 项和 rollback drill 时，系统会明确说明缺失证据，而不会提升声明等级。
**Plans:** TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → 11 → 12 → 13

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Safety, Privacy, and State Foundation | 0/7 | Planned    |  |
| 2. Read-only Ownership Inspector | 0/TBD | Not started | - |
| 3. Node, npm, pnpm, and Corepack Governance | 0/TBD | Not started | - |
| 4. Go Governance | 0/TBD | Not started | - |
| 5. Python and uv Governance | 0/TBD | Not started | - |
| 6. Rust and rustup Governance | 0/TBD | Not started | - |
| 7. Deno and Bun Governance | 0/TBD | Not started | - |
| 8. JVM, Maven, and Gradle Governance | 0/TBD | Not started | - |
| 9. Multi-host Composition and Binder Spike | 0/TBD | Not started | - |
| 10. Recovery Observation and Readiness | 0/TBD | Not started | - |
| 11. Fail-closed Symlink Safety | 0/TBD | Not started | - |
| 12. Integrated Recovery Engine | 0/TBD | Not started | - |
| 13. Current-host Non-destructive Readiness Drill | 0/TBD | Not started | - |
