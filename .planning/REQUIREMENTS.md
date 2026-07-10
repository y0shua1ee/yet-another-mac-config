# Requirements: Yet Another Mac Config

**Defined:** 2026-07-10
**Core Value:** 在不泄露私密信息、也不破坏任何已有可用环境的前提下，让受支持的 Mac 能从本仓库恢复到可验证、尽可能一致的开发与软件配置状态。

## v1 Requirements

### 安全、隐私与测试隔离

- [x] **SAFE-01**：操作者可以分别识别并校验 desired state、observed state、generated plan、applied receipt、verification evidence 和 readiness report；任何一种 artifact 都不能冒充另一种完成证据。
- [x] **SAFE-02**：所有可持久化 artifact 只使用逻辑路径和隐私安全标识，不包含真实用户名、真实 hostname、序列号、稳定硬件指纹或绝对 HOME 路径。
- [x] **SAFE-03**：秘密值、登录数据、私有网络信息、具体 provider item reference、完整环境变量 dump 和未经约束的原始命令输出在 stdout、stderr 或 artifact 落盘前即被拒绝或结构化清除。
- [x] **SAFE-04**：操作者可以在隔离 fixture 中验证配置；fixture 使用独立 HOME、XDG 和工具专用配置、数据、缓存、trust 及运行时根目录，不读取真实项目或真实全局状态。
- [ ] **SAFE-05**：自动测试默认关闭网络、自动安装、自动下载、自动修复和 trust mutation；只有明确 opt-in 的隔离 integration test 才能在临时根目录内产生状态。
- [ ] **SAFE-06**：live probe 只有在当前官方语义和隔离负路径测试都证明其不会写入、安装、下载或执行任意配置后才能进入 allowlist；否则不执行并返回 `unknown` 或 `manual-required`。
- [ ] **SAFE-07**：测试 harness 在执行前后使用 sentinel 证明真实 HOME、全局工具状态、worktree、服务和仓库外状态未发生未授权变化。
- [ ] **SAFE-08**：默认策略只报告 extra 或 unmanaged state，不自动执行 Homebrew cleanup/uninstall/zap、runtime 删除或其他 destructive convergence。

### 工具所有权与执行来源检查

- [ ] **OWN-01**：操作者可以按 executable 和 execution context 指定唯一主 owner，并区分 manager binary、runtime、package manager、project dependency、system library 和 environment loader 的职责。
- [ ] **OWN-02**：只读 inspector 分别报告工具的 `declared`、`installed`、`selected` 和 `executed` 状态，不用“已安装”替代“实际执行来源”。
- [ ] **OWN-03**：只读 inspector 报告 executable path、版本、架构、配置来源和优先级、override 来源及 auto-download 状态。
- [ ] **OWN-04**：只读 inspector 能发现重复候选、PATH shadowing、隐式下载、未纳管 executable 和 trust-required 状态，但不自动安装、切换、删除或修复。
- [ ] **OWN-05**：操作者可以比较 interactive、login、non-interactive、备用 Shell 和 hook/IDE 等关键执行上下文；无法安全验证的上下文被明确标记为 `unsupported` 或 `unknown`。
- [ ] **OWN-06**：操作者可以按生态维护 migration ledger，依次记录迁移前 owner、目标 owner、项目契约、fixture 证据、只读本机证据、回滚路径和 cleanup approval；证据不完整时旧入口保持不变。

### Node、npm、pnpm 与 Corepack

- [ ] **NODE-01**：Node 项目可以声明精确 Node 版本，并证明 mise 项目契约优先于最小全局 fallback；无项目契约时 fallback 仍可用。
- [ ] **NODE-02**：npm 随所选 Node 提供，项目能够证明不存在第二个 npm 主 owner。
- [ ] **NODE-03**：Node 项目默认由 mise 提供精确 pnpm，并使用 `packageManager`、唯一 lockfile 和 frozen/locked 行为表达包管理契约。
- [ ] **NODE-04**：明确需要 Corepack 的项目可以选择精确 standalone Corepack compatibility variant；启用后同一项目不得再由 mise 同时拥有 pnpm。
- [ ] **NODE-05**：隔离 fixture 可以验证正确版本、fallback 优先级、lockfile、错误 package manager 和 dual-owner 负路径，而不访问真实项目、真实缓存或现有 fallback。

### Go

- [ ] **GO-01**：Go 项目可以声明精确的 mise-owned Go runtime，并解释 `go.mod` 或 `go.work` 中 `go` 与 `toolchain` 指令的最终解析结果。
- [ ] **GO-02**：只读检查可以报告 `GOTOOLCHAIN`、最终 Go executable 和实际 compiler，不触发 toolchain 自动下载，也不执行 `go env -w`。
- [ ] **GO-03**：Go fixture 隔离 GOPATH、module cache、build cache 和 toolchain state，默认离线且不读取或污染真实 Go 状态。
- [ ] **GO-04**：项目 Go 契约优先于全局 fallback；当前 fallback 在 fixture、本机只读证据和回滚路径通过前不得被删除或升级。

### Python 与 uv

- [ ] **PY-01**：Home Manager 只负责提供 uv/uvx binary，uv 独占项目 Python interpreter、虚拟环境、依赖锁定和 Python tool。
- [ ] **PY-02**：Python 项目可以一致校验 `requires-python`、精确 `.python-version`、`pyproject.toml`、`uv.lock` 和 locked/frozen 行为。
- [ ] **PY-03**：只读检查报告实际 Python provenance 和项目环境来源，不下载 Python、不执行 sync，也不把 system Python 或共享虚拟环境当作项目 owner。
- [ ] **PY-04**：Python fixture 隔离 uv cache、managed Python、tool 和项目环境，不在真实项目中创建 `.venv` 或修改真实 Python 状态。

### Rust 与 rustup

- [ ] **RUST-01**：Home Manager 只负责提供 rustup binary，rustup 独占 Rust toolchain、component、target 和 Cargo proxy。
- [ ] **RUST-02**：Rust 项目使用精确 `rust-toolchain.toml`、所需 component/target 和 Cargo lockfile 表达可复现契约，不以浮动 `stable` 代替精确版本。
- [ ] **RUST-03**：只读检查解释 command、environment、directory override、toolchain file 和 default 的优先级，并证明实际 `rustc`/`cargo` 来自预期 rustup proxy。
- [ ] **RUST-04**：Rust fixture 隔离 `RUSTUP_HOME`、`CARGO_HOME` 和相关缓存，默认离线，不运行 self-update 或修改真实 global default。

### Deno 与 Bun

- [ ] **DBUN-01**：项目可以按需声明由 mise 管理的精确 Deno 或 Bun 版本，且无需为未使用的 runtime 建立默认全局 fallback。
- [ ] **DBUN-02**：Deno/Bun 项目能够使用原生 config 和 lockfile 校验 frozen/CI 契约，而不改写真实 lockfile。
- [ ] **DBUN-03**：同时包含 Node、Deno 或 Bun 元数据的项目必须明确主执行路径；不明确或存在多个隐式 owner 时 inspector 报告冲突。
- [ ] **DBUN-04**：Deno/Bun fixture 隔离 cache、data、install 和 lock state，默认离线，不执行 upgrade 或自动迁移其他 package-manager 的真实 lockfile。
- [ ] **DBUN-05**：inspector 能识别 Homebrew、mise 和 official/direct Bun path 的 shadowing；现有 direct Bun 入口在新路径证据和回滚完成前保持不变。

### JVM、Maven 与 Gradle

- [ ] **JVM-01**：项目可以通过 mise 声明精确 JDK vendor 和版本；真实项目 inventory 完成前不引入未经证明的通用全局 JDK fallback。
- [ ] **JVM-02**：Gradle Wrapper 独占项目 Gradle 版本并校验 distribution checksum，v1 不增加全局 Gradle owner。
- [ ] **JVM-03**：Maven Wrapper 独占项目 Maven 版本并校验 distribution checksum，v1 不增加全局 Maven owner。
- [ ] **JVM-04**：只读检查分别报告 Java executable、`JAVA_HOME`、Gradle daemon/toolchain 和 Maven toolchain 的实际选择及冲突。
- [ ] **JVM-05**：check 不触发 Wrapper distribution 或 JDK 下载；integration fixture 使用隔离 Maven/Gradle user home，并把 multi-JDK 场景记录为显式例外。

### 多主机组合与身份绑定

- [ ] **HOST-01**：公共配置可以组合 shared baseline、非敏感 role 和 logical host profile；v1 至少提供当前 Mac 对应的一个逻辑 profile。
- [ ] **HOST-02**：公共 profile、Git 历史、plan 和 report 不使用真实用户名、hostname、序列号、硬件指纹或私有 endpoint 作为选择键或输出。
- [ ] **HOST-03**：Git 外的 identity-only binder 只把必要的本机身份映射到公共 logical profile，并与 secret/provider binding 完全分离。
- [ ] **HOST-04**：日常配置不通过 ignored Nix import、包含秘密的 local flake input 或 routine `--impure` 绕过 Git-flake source boundary。
- [ ] **HOST-05**：替换当前可工作的单主机 composition 前，至少两个 synthetic logical profile 能从 clean Git source evaluation，并通过 Nix store/privacy 检查；验证失败时保留当前 composition。
- [ ] **HOST-06**：logical profile 可以声明 expected divergence 和受支持平台；v1 只把当前 Apple Silicon host class 标为已验证，其他架构保持 `unverified-platform`。

### 非秘密义务与人工私密状态

- [ ] **SECR-01**：provider-neutral secret obligation manifest 只保存稳定 ID、用途、required/optional、消费方、允许的 provider 类型和安全 presence rule。
- [ ] **SECR-02**：v1 的秘密检查只返回 `present`、`missing`、`manual` 或 `unknown`，不读取或打印值，不触发 provider 登录、Keychain prompt 或秘密注入。
- [ ] **SECR-03**：TCC、Accessibility、Full Disk Access、Apple ID/App Store、其他账号登录、重启和类似状态只以非敏感 manual-step data 表达，不尝试自动授权、复制数据库或伪造完成。

### Desired/Observed 发现与 readiness

- [ ] **OBS-01**：操作者可以查看 canonical public inventory，覆盖 Nix/nix-darwin、Home Manager、Homebrew formula/cask/font、service、Shell、六生态工具链、受跟踪应用配置、稳定 defaults、CLI、符号链接和人工状态。
- [ ] **OBS-02**：inventory 条目可以标明 owner、scope、canonical source、check、apply、verify、rollback 或 forward-repair，以及 expected divergence、manual 和 excluded 边界。
- [ ] **OBS-03**：desired state 只投影 Git-tracked native source 和显式 manifest，不复制已有版本 source，也不把 observed machine state 反写成唯一 desired source。
- [ ] **OBS-04**：desired-state discovery 在 Git 缺失、非 worktree、tracked query 失败或为空、manifest 缺失或无效时 fail closed，不回退扫描物理 `.config`、HOME 或 ignored/untracked 目录。
- [ ] **OBS-05**：observed state 只来自 allowlisted named-target probe，并保存规范化事实；通用 `brew bundle check` 只有在 Brewfile 不含可执行 `system` 条目且通过隔离负路径验证后才能使用。
- [ ] **OBS-06**：sanitized JSON 和 Markdown readiness report 至少区分 `verified`、`declared-unapplied`、`drift`、`private-missing`、`manual-required`、`excluded`、`unknown`、`expected-divergence` 和 `unverified-platform`。

### 符号链接安全

- [ ] **LINK-01**：所有符号链接入口都只使用受跟踪的显式 manifest 和同一套 fail-closed planner，不保留绕过 manifest 的物理目录 fallback。
- [ ] **LINK-02**：planner 在生成动作前解析 source、destination、目标 home 和现有 target owner，并拒绝路径逃逸、循环、跨 owner 冲突或覆盖 Home Manager 等其他 writer 的目标。
- [ ] **LINK-03**：符号链接操作默认只生成 data-only plan；真实替换必须确认 exact plan，且不能成为自动测试的一部分。
- [ ] **LINK-04**：已有 destination 在替换前必须创建可恢复 backup/restore anchor，正常替换不得使用 `rm -rf` 或递归删除。
- [ ] **LINK-05**：fake-home fixtures 覆盖首次创建、重复执行、冲突、Git/manifest 失败、执行中断和 restore，证明操作幂等或能停止在可恢复点。

### 恢复控制流程

- [ ] **RCVR-01**：generated plan 是 immutable、machine-readable、data-only 的 allowlisted typed operation 列表；每项具有稳定 operation ID、owner、目标、风险、前置条件、postcondition、预期下载、backup 和真实 rollback/forward-repair metadata。
- [ ] **RCVR-02**：plan digest 绑定 repository revision、相关 lockfile、logical profile、platform、adapter/schema 版本、exact operation list 和相关 non-secret observed fingerprint。
- [ ] **RCVR-03**：confirmation 只对 exact plan digest 生效；默认行为停在 plan，模糊的通用确认不能授权未来重新计算的动作。
- [ ] **RCVR-04**：apply 只消费已确认 plan，不重新 discovery、重新 diff、追加或重排 operation；发现未计划工作时零写入停止并要求重新 plan。
- [ ] **RCVR-05**：每个 write boundary 前重新校验相关 precondition；repo、lock、profile、platform、target、inventory 或 before-state 变化时 stale plan 必须在第一笔写入前失败。
- [ ] **RCVR-06**：Nix build、privileged switch、Home Manager、Homebrew package/cask、service、defaults、link 和 toolchain operation 保持独立 confirmation/checkpoint/receipt，不能伪装成单一事务。
- [ ] **RCVR-07**：operation 失败时停止后续写入并保留 component receipt；每项明确标记 `reversible`、`compensatable`、`forward-repair-only` 或 `manual`，只承诺自身组件的恢复边界。
- [ ] **RCVR-08**：verify 使用 fresh observation 按 operation ID 比较 postcondition；apply 前 snapshot、退出码或 receipt 不能替代 verification evidence。

### 当前 Mac 验收证据与声明等级

- [ ] **EVID-01**：当前 Mac 的 v1 drill 默认只运行静态检查、隔离 fixtures、已证明安全的 read-only probes 和单独审查的 non-activating build；未经额外 plan 与确认不执行真实 mutation。
- [ ] **EVID-02**：current-host evidence 覆盖主 Shell、备用 Shell、login、non-interactive 和可安全观察的 hook/IDE 场景，并报告 first-run、private、login、TCC、manual、drift 和 expected-divergence 项。
- [ ] **EVID-03**：仅凭当前 Mac evidence，claim validator 最高只能输出 `recovery-ready-on-current-host`，不能输出多机或 `fresh-install-verified`。
- [ ] **EVID-04**：`fresh-install-verified` 必须由未来 clean VM 或第二台 Mac 的 artifact 支持，该 artifact 包含记录的起点、exact plan、component receipts、fresh verification、manual/excluded 项和 rollback drill。

## v1.x Requirements

### 可选秘密 provider

- **SECR-04**：操作者可以选择 1Password、Keychain 或其他 adapter 满足 provider-neutral secret obligation，且 adapter 不把秘密写入 Git、Nix store、plan、receipt、report 或日志。

### 可安全导出的应用状态

- **APPS-01**：经逐应用隐私与可移植性审计后，操作者可以同步公开的扩展/插件列表、主题和偏好，不包含账号、会话或应用数据库。

### Apply journal 与恢复

- **RCVR-09**：operation model 稳定后，操作者可以从明确 receipt/checkpoint 恢复中断的 apply，而不是盲目重放整个流程。

### 原生 validator 聚合

- **VAL-01**：操作者可以从统一入口运行 Ghostty、Neovim、AeroSpace、Yazi、Hammerspoon 等受控应用的原生 validator，并保留 GUI/TCC 的人工门。

### 安全 CI

- **CI-01**：CI 可以运行 schema、fixture、Shell syntax、Nix evaluation 和隐私检查，但不能把 CI 结果当作真实 Mac activation 或 readiness 证据。

### 更多逻辑 profile

- **HOST-07**：出现真实第二台 Mac 或明确角色差异后，操作者可以新增经过验证的 logical profile，而无需复制整个 shared baseline。

## v2 Requirements

### 干净主机验证

- **E2E-01**：操作者可以在 disposable macOS VM 上从记录的干净起点完成 bootstrap、staged apply、verify 和 rollback drill；证据链通过后才能提升为 `fresh-install-verified`。
- **E2E-02**：操作者可以在第二台实体 Mac 上复现同一流程，并比较 logical profile 的 expected divergence。

### 平台扩展

- **PLAT-01**：只有实际出现 Intel/Rosetta 需求后，系统才增加并验证 `x86_64-darwin` profile 和相应兼容性规则。

### 多机与界面

- **FLEET-01**：多个真实 profile 稳定后，操作者可以查看跨主机 readiness 汇总。
- **UX-01**：CLI 与 schema 稳定后，操作者可以通过可视化界面审查 plan、确认 operation 和查看 verification evidence。

### 跨项目迁移

- **BULK-01**：具备明确项目 inventory、逐仓库授权和回滚策略后，操作者可以批量迁移真实项目的工具链契约。

### 选择性收敛

- **CONV-01**：extra-state policy 经多机验证后，操作者可以 opt-in 生成逐项 destructive convergence plan；该能力永不成为默认行为。

## Out of Scope

| Feature | Reason |
|---------|--------|
| 将 API key、token、密码、私钥、恢复码、证书私钥、具体 secret binding 或私有 endpoint 写入 Git | Git 历史、clone、日志和备份会扩大泄露面；未来 provider adapter 也不能改变此边界 |
| 自动同步 Apple ID、App Store、浏览器、聊天软件或其他账号登录 | 登录会话属于敏感本机状态，不适合声明式复制 |
| 同步浏览器历史、聊天记录、应用数据库、缓存、媒体历史或完整用户数据 | 超出配置恢复范围且存在显著隐私风险 |
| 自动授予或绕过 TCC、Accessibility、Full Disk Access、Automation 等权限 | 必须保留 macOS 的用户知情与授权边界 |
| 无人值守 sudo、模糊确认或一步式 check-and-apply | 与 exact-plan confirmation 和现有环境安全冲突 |
| 默认静默 cleanup、zap、递归覆盖或“让机器完全一致” | 会把用户有意保留的 extra state 当作垃圾并破坏唯一工作 Mac |
| 强迫所有语言、runtime 与 build tool 都归 mise | 目标是统一治理和唯一 owner，不是抹掉 uv、rustup、lockfile、Wrapper 或 Nix devShell 的职责 |
| 引入另一个全权 dotfiles manager | 会与现有 Nix/Home Manager/Homebrew/symlink writer 形成新的重叠所有权 |
| 字节级克隆多台 Mac | 会混入身份、会话、数据库、缓存和硬件相关状态；项目只保证公开、可声明范围的语义一致 |
| 宣称跨 Nix、Homebrew、Home Manager、service、defaults 和 link 的全局原子事务或统一 rollback | 各 writer 的副作用与恢复能力不同，只能提供组件级停止、补偿和 forward repair |
| Git/tracked discovery 失败后扫描 ignored/untracked 物理目录 | 会绕过公共 source-of-truth 和隐私边界 |
| 运行未固定 revision/hash、未经审查的远程 installer | Moving remote content 不能形成可审查、可复现的恢复计划 |
| check/verify 为了变绿而自动安装、下载、切换、重启或修复真实机器 | 会破坏只读证据语义；修复只能先形成新的 plan |
| 未经逐项目授权批量修改仓库之外的真实项目 | 本仓库定义治理契约与 fixture；外部项目需要独立范围和回滚 |
| 没有 clean-host evidence 就声称 fresh-install 或多机已验证 | 当前 warm host 无法证明 bootstrap、冷缓存、首次登录和权限顺序 |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| SAFE-01 | Phase 1 | Complete |
| SAFE-02 | Phase 1 | Complete |
| SAFE-03 | Phase 1 | Complete |
| SAFE-04 | Phase 1 | Complete |
| SAFE-05 | Phase 1 | Pending |
| SAFE-06 | Phase 1 | Pending |
| SAFE-07 | Phase 1 | Pending |
| SAFE-08 | Phase 1 | Pending |
| OWN-01 | Phase 2 | Pending |
| OWN-02 | Phase 2 | Pending |
| OWN-03 | Phase 2 | Pending |
| OWN-04 | Phase 2 | Pending |
| OWN-05 | Phase 2 | Pending |
| OWN-06 | Phase 2 | Pending |
| NODE-01 | Phase 3 | Pending |
| NODE-02 | Phase 3 | Pending |
| NODE-03 | Phase 3 | Pending |
| NODE-04 | Phase 3 | Pending |
| NODE-05 | Phase 3 | Pending |
| GO-01 | Phase 4 | Pending |
| GO-02 | Phase 4 | Pending |
| GO-03 | Phase 4 | Pending |
| GO-04 | Phase 4 | Pending |
| PY-01 | Phase 5 | Pending |
| PY-02 | Phase 5 | Pending |
| PY-03 | Phase 5 | Pending |
| PY-04 | Phase 5 | Pending |
| RUST-01 | Phase 6 | Pending |
| RUST-02 | Phase 6 | Pending |
| RUST-03 | Phase 6 | Pending |
| RUST-04 | Phase 6 | Pending |
| DBUN-01 | Phase 7 | Pending |
| DBUN-02 | Phase 7 | Pending |
| DBUN-03 | Phase 7 | Pending |
| DBUN-04 | Phase 7 | Pending |
| DBUN-05 | Phase 7 | Pending |
| JVM-01 | Phase 8 | Pending |
| JVM-02 | Phase 8 | Pending |
| JVM-03 | Phase 8 | Pending |
| JVM-04 | Phase 8 | Pending |
| JVM-05 | Phase 8 | Pending |
| HOST-01 | Phase 9 | Pending |
| HOST-02 | Phase 9 | Pending |
| HOST-03 | Phase 9 | Pending |
| HOST-04 | Phase 9 | Pending |
| HOST-05 | Phase 9 | Pending |
| HOST-06 | Phase 9 | Pending |
| SECR-01 | Phase 10 | Pending |
| SECR-02 | Phase 10 | Pending |
| SECR-03 | Phase 10 | Pending |
| OBS-01 | Phase 10 | Pending |
| OBS-02 | Phase 10 | Pending |
| OBS-03 | Phase 10 | Pending |
| OBS-04 | Phase 10 | Pending |
| OBS-05 | Phase 10 | Pending |
| OBS-06 | Phase 10 | Pending |
| LINK-01 | Phase 11 | Pending |
| LINK-02 | Phase 11 | Pending |
| LINK-03 | Phase 11 | Pending |
| LINK-04 | Phase 11 | Pending |
| LINK-05 | Phase 11 | Pending |
| RCVR-01 | Phase 12 | Pending |
| RCVR-02 | Phase 12 | Pending |
| RCVR-03 | Phase 12 | Pending |
| RCVR-04 | Phase 12 | Pending |
| RCVR-05 | Phase 12 | Pending |
| RCVR-06 | Phase 12 | Pending |
| RCVR-07 | Phase 12 | Pending |
| RCVR-08 | Phase 12 | Pending |
| EVID-01 | Phase 13 | Pending |
| EVID-02 | Phase 13 | Pending |
| EVID-03 | Phase 13 | Pending |
| EVID-04 | Phase 13 | Pending |

**Coverage:**
- v1 requirements: 73 total
- Mapped to phases: 73
- Unmapped: 0

---
*Requirements defined: 2026-07-10*
*Last updated: 2026-07-10 after initial definition*
