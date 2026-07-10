# Project Research Summary

**Project:** Yet Another Mac Config
**Domain:** 公开 Git 同步的 macOS 开发环境治理与非破坏性恢复系统
**Researched:** 2026-07-10
**Confidence:** MEDIUM（安全边界与工具所有权为 HIGH；尚未实现验证的多主机绑定、跨 plane apply 与干净主机恢复拉低整体置信度）

## Executive Summary

这是一个 brownfield 的个人 Mac 配置系统，不是要再引入一个“总管”式 dotfiles 工具。现有 Nix、nix-darwin、Home Manager、Homebrew inventory 与应用配置软链接已经覆盖主要 desired state；真正缺失的是统一的所有权规则、只读发现、隔离验证、跨 activation plane 的计划与证据模型。推荐先建立不可变的安全底座和六生态 ownership inspector，再按 Node、Go、Python、Rust、Deno/Bun、JVM 逐个迁移。每个生态只允许一个主 owner，保留原生 lockfile、wrapper 与专用 manager 的职责，不为了形式统一把所有内容交给 mise。

恢复系统应当是一个薄控制面，而不是第二个包管理器。它必须严格区分五种状态：Git 中的 desired state、只读探测得到的 observed state、绑定摘要与前置条件的 immutable plan、真实执行产生的 applied receipt，以及重新探测得到的 verification evidence。Nix/Home Manager/Homebrew activation 与应用软链接是两个不同的写入 plane；它们可以排序和协调，但不能被宣传为一个原子事务。任何真实变更都遵循 `check → plan → confirm → apply → verify`，并以组件级停止点、备份和真实回滚边界替代“一键同步成功”的假象。

首要风险是唯一工作 Mac 被测试或“检查”意外修改。默认测试只能做静态解析、fake-adapter contract 和完整隔离的 fixture；不能证明只读的命令必须返回 `unknown` 或 `manual-required`。尤其不能把会自动安装的 mise/uv/Go/Gradle 操作或可能执行 Brewfile `system` 条目的 `brew bundle check` 当作安全 probe。多主机与秘密层也必须保持边界：公共 Git 只存逻辑 profile 和 secret obligation，真实 identity/provider binding 留在 Git 外；在 VM 或第二台干净 Mac 完成证据链以前，最高声明只能是 `recovery-ready-on-current-host`。

## Key Findings

### Recommended Stack

详细结论见 [STACK.md](./STACK.md)。保留现有 Nix/nix-darwin/Home Manager/Homebrew 组合；新增的是统一治理和恢复控制面，而不是新的全权配置工具。版本号只是 2026-07-10 的研究快照，不是升级指令；当前仓库 pin 和全局 fallback 必须保持不动，直到相应生态完成独立计划、fixture、只读核对与显式 apply。

**Core technologies:**

- **nix-darwin + Home Manager：** 管理机器层、用户层、manager binary 与主 Shell 集成；build/evaluation 与 switch 必须分开记录。
- **Homebrew（由 nix-darwin 声明）：** 管理 macOS GUI 应用、formula、cask、字体与有限服务；继续保持不自动 update、upgrade 或 cleanup。
- **mise：** 管理适合跨项目切换的 Node、Go、Deno、Bun 与项目 JDK；项目使用精确版本和锁定数据，全局 fallback 保持最小。
- **uv：** 独占 Python interpreter、项目环境、依赖锁定和 Python tool；Home Manager 只提供 uv/uvx binary。
- **rustup：** 独占 Rust toolchain、component、target 与 Cargo proxy；Home Manager 只提供 rustup binary。
- **Gradle Wrapper / Maven Wrapper：** 独占项目 build-tool 版本；不增加全局 Gradle 或 Maven。
- **Nix devShell：** 作为 opt-in 项目环境。它对自己声明的每个 executable 是唯一 owner；如果只提供不重叠的系统库，可与语言原生 contract 共存；如果声明语言 runtime/build tool，同一项目不得再由 mise、uv、rustup 或全局 profile 提供同一 executable。
- **direnv + nix-direnv：** 只负责经显式授权的目录环境加载，不拥有 package/runtime，也不自动批准 `.envrc`。

### Resolved Design Questions

| Question | Recommendation | Boundary that must remain explicit |
|---|---|---|
| pnpm 由 mise 还是 Corepack 管理 | **默认由 mise 直接安装并固定 exact pnpm**；npm 随所选 Node；`packageManager` 可作为项目期望元数据 | Corepack 只作为项目明确要求的 compatibility variant；使用时必须有 exact standalone Corepack、关闭隐式更新，并移除该项目的 mise-pnpm ownership |
| Nix devShell 是否可以和 mise/uv/rustup 混用 | **按 executable 排他，而不是按工具品牌排他** | Nix 可只提供不重叠的系统库；一旦 devShell 声明某 runtime/build tool，Nix 就是该 executable 的唯一项目 owner，必须验证 PATH 进入和退出行为 |
| ignored host selector 如何进入 Git flake | **不要从公共 Git flake 导入 checkout 内的 ignored `.nix` 文件，也不要把 `--impure` 设为日常路径** | 先做独立 design spike；候选方案是 Git-tracked public flake + Git 外、identity-only 的本地 wrapper/binder，并以 synthetic host 和 clean Git source 验证；secret binding 永不进入 Nix |
| 哪些 probe 可以称为 read-only | **只有官方语义和隔离测试都证明不写入、不安装、不下载的 allowlisted probe** | 通用 `brew bundle check` 不是默认安全 probe，因为 Brewfile `system` 条目会在 check 时执行；优先直接查询命名 inventory，或先静态证明生成 Brewfile 只含安全 declarative entry |

### Expected Features

详细结论见 [FEATURES.md](./FEATURES.md)。

**Must have (table stakes):**

- **Canonical public desired-state inventory** — 每项记录 owner、scope、source、check、apply、verify、rollback 与允许差异，不复制 native source 中已经存在的版本值。
- **六生态所有权矩阵与项目 contract** — 覆盖版本、lockfile/wrapper、fallback、环境加载、自动下载策略、冲突判定和回滚。
- **Read-only ownership inspector** — 同时显示 declared、installed、selected、executed、配置来源、PATH shadowing 与 execution context；绝不自动修复。
- **隔离 fixture harness** — 静态/schema、fake-adapter、opt-in isolated integration 三层；真实 HOME、真实项目、manager state、trust、服务与缓存均不可作为 fixture。
- **共享 baseline + 隐私安全 logical host profile** — 公共层不含真实用户名、hostname、序列号或设备指纹；本地绑定与公共 policy 分离。
- **Provider-neutral secret obligation manifest** — Git 只存稳定 ID、用途、required/optional、accepted provider 和 presence rule；不解析或打印值。
- **Fail-closed tracked discovery** — 缺少 Git/worktree/manifest 时停止，不扫描物理 `.config` 作为 fallback。
- **稳定的 plan-bound confirmation** — plan 是 data-only typed operations，包含 digest、前置条件、风险、backup、rollback；apply 不重新发现或添加 operation。
- **组件化 apply、receipt 与 verify** — Nix、Home Manager、Homebrew、service、defaults、link 和 toolchain 分开记录结果与回滚边界。
- **分级 readiness report** — 至少区分 `verified`、`declared-unapplied`、`drift`、`private-missing`、`manual-required`、`excluded`、`unknown` 和 `unverified-platform`。
- **安全文档与隐私质量门** — diff hygiene、generic identifier scan、Gitleaks、结构化 redaction 和原子提交共同构成发布门。

**Should have (competitive):**

- **跨 manager ownership conflict detector** — 给出证据与推荐 owner，但不自动删除重复安装。
- **跨 activation-plane 统一 plan** — 一次预览全部 plane，同时保留每个 writer 的独立语义。
- **渐进式 migration ledger** — 每生态保存 before owner、target owner、fixture evidence、只读 live evidence、rollback 与 cleanup approval。
- **Shell route equivalence checks** — 比对主 Shell、fallback、login、non-interactive 与 hook/IDE 场景，或明确标记不支持的场景。
- **Cleanup advisory mode** — extras 只报告为 drift/unmanaged-present；删除始终是未来、逐项、额外确认的动作。
- **Plan-to-verify traceability** — operation ID 从 plan 贯穿 receipt 和 fresh verification evidence。
- **Safe claim levels** — 让项目完成度由证据升级，而不是由一个 ready boolean 决定。

**Defer (v1.x/v2+):**

- **1Password/Keychain 等 secret-provider adapter** — provider-neutral manifest 稳定后再作为可选能力，不能成为 v1 前置条件。
- **可恢复的 apply resume/journal UX** — v1 先记录可靠 receipt；跨中断 resume 待 operation model 稳定后再开放。
- **可安全导出的 app extension/theme state** — 逐 app 做隐私与可移植性审计后再考虑。
- **干净 macOS VM 或第二台 Mac E2E** — 条件具备后执行；它是升级 fresh-install claim 的必要证据。
- **Intel/Rosetta 支持、fleet dashboard、GUI 与批量迁移真实项目** — 没有真实需求与授权前不提前抽象。
- **默认 destructive convergence** — 不实施；即使未来提供，也只能是 opt-in、逐项、可预览操作。

### Architecture Approach

详细结论见 [ARCHITECTURE.md](./ARCHITECTURE.md)。推荐以薄恢复控制面编排现有 writers：公共 Git 保存 policy 与 logical profile；本地 privacy boundary 提供 identity-only binder 和 secret/provider binding；只读 adapters 产生 normalized observed facts；pure planner 生成 content-addressed typed plan；confirmed apply 调用 allowlisted component adapters；fresh verifier 再次探测并生成 sanitized report。read 与 write 实现应物理分离，`check`、`plan`、`verify` 不得导入或 dispatch apply layer。

**State artifacts must not collapse into each other:**

| Artifact | Answers | Evidence rule |
|---|---|---|
| Desired state | 仓库期望什么 | 只来自 Git-tracked Nix/native contract/manifest；无 secret value、真实 identity 或重复版本 source |
| Observed state | 当前机器可安全证明什么 | 只读 named-target probe 后规范化；不持久化 raw stdout、完整路径或环境 dump |
| Generated plan | 为收敛差异准备执行什么 | data-only、immutable、content-addressed；无任意 shell、secret/provider value 或绝对 home path |
| Applied receipt | exact plan 中哪些 operation 实际执行了 | append-only per run；记录 outcome、checkpoint、backup/compensation anchor，不把退出 0 当作 verify |
| Verification evidence | apply 后现在能重新证明什么 | fresh observation + expected postcondition comparison；不能复用旧 observed snapshot 或 receipt 代替 |

**Major components:**

1. **Shared baseline and logical profiles** — 组合公共 Nix/HM/Homebrew/Shell policy 与非敏感角色差异。
2. **Local binder and private bindings** — 在 Git 外提供真实 identity 与 provider mapping；二者分开，secret 永不传入 Nix。
3. **Canonical manifests and desired projector** — ownership、link topology 与 secret obligation 各自只声明其负责的维度；其余值从 native source 投影。
4. **Read-only inventory adapters** — 只查询 allowlisted target，把 path、version、source、service/manual state 规范化。
5. **Policy validator, semantic diff and typed plan compiler** — 检查 duplicate owner、非法字段、unsupported action、rollback metadata 与 stale precondition。
6. **Confirmation gate and apply coordinator** — approval 绑定 plan digest；apply 只执行 plan 中的 enum operation，发现新 drift 即停止。
7. **Declarative, symlink and ecosystem adapters** — 每个 plane 独立 build/checkpoint/apply/receipt/rollback，不伪装成跨组件 transaction。
8. **Fresh verifier and report renderer** — 重新观察后输出 JSON/Markdown evidence 与受约束 claim level。
9. **Fixture pyramid** — static/schema、fake adapter、隔离 integration；默认无网络、无自动 install、无真实 state。

### Critical Pitfalls

详细结论见 [PITFALLS.md](./PITFALLS.md)。

1. **把 Nix build/switch 当成整机原子事务** — 分 plane 计划、停止、receipt、postcondition 与 rollback；Nix generation 不能代表 Homebrew、service 或 link 已回滚。
2. **用 cleanup/zap 制造多机“完全一致”** — 保留 `cleanup = "none"`；extra package/cask 只报告，不生成默认删除动作。
3. **把 declared version 当成 executed version** — 每生态验证 binary provenance、merged config、override、auto-download、Shell/hook/daemon context 后才能清理旧 owner。
4. **所谓隔离测试仍继承真实 state** — 临时 HOME 不够；必须重定向 XDG 与全部 manager-specific root、限制 config discovery、关闭 auto-install/network，并用 sentinel 证明真实状态未变。
5. **Git 不含秘密但 Nix/store/log/plan 泄露秘密或身份** — 值从不进入 desired/plan/report；只输出 presence status；在落盘前结构化 redaction，而不是事后遮罩 raw log。
6. **`.gitignore` 被误当 deployment allowlist** — 只接受显式 tracked manifest；Git 查询失败/空结果必须 fail closed；link replace 只能 backup-and-restore，禁止递归删除。
7. **ignored host overlay 与 Git flake purity 冲突** — 多主机阶段先做 source-boundary spike；不采用 ignored import、routine impurity 或包含秘密的 local input。
8. **“read-only”命令实际执行或下载** — 对 Homebrew 使用直接 inventory；对 mise/uv/Go/wrapper 显式关闭自动行为；无法证明时报告 unknown/manual。
9. **确认后 plan 已过期** — apply 校验 repo/lock/profile/target/inventory fingerprint；任何前置条件变化都零写入并要求重新 plan。
10. **把当前 Mac 演练称作 fresh-install verified** — 报告证据等级；只有 disposable VM/第二台 Mac 从已记录起点完成 E2E 才能升级声明。

## Implications for Roadmap

研究建议采用 13 个当前里程碑阶段，另保留 1 个未来证据阶段。阶段拆得较细是为了让每个生态都有独立 rollback 与原子提交，也避免在唯一工作 Mac 上扩大故障面。

### Phase 1: Safety, Privacy, and State Foundation

**Rationale:** 所有后续 live probe、fixture 和 apply 都依赖先证明“不碰真实环境”。
**Delivers:** desired/observed/plan/receipt/evidence/report schema；forbidden-field/redaction policy；logical path namespace；fixture root isolation；real-state sentinel；provenance matrix；非破坏性测试入口。
**Addresses:** canonical inventory schema、secret obligation boundary、safe claim vocabulary、fail-closed discovery contract。
**Avoids:** PF-02、PF-05、PF-06、PF-07、PF-09、PF-11、PF-13、PF-14、PF-16。
**Gate:** 没有 live apply；任何测试输出不得包含真实 home/host/secret canary，且真实状态和 worktree sentinel 不变。

### Phase 2: Read-only Ownership Inspector

**Rationale:** 删除或迁移前必须知道实际执行的是谁，而不是只看配置文件。
**Delivers:** 六生态 owner schema、PATH/provenance/config-source inspector、Shell context matrix、service intent/runtime 分离、duplicate/unmanaged evidence；只报告不清理。
**Addresses:** ownership matrix、effective-resolution explanation、conflict detector。
**Avoids:** PF-03、PF-04、PF-15。
**Gate:** declared、installed、selected、executed 可区分；不能证明只读的 adapter 返回 `unknown`。

### Phase 3: Node, npm, pnpm, and Corepack Governance

**Rationale:** 当前已有 Node fallback，且 pnpm/Corepack ownership 是最容易发生双重路由的边界。
**Delivers:** mise-owned exact Node/pnpm project contract、Node-bundled npm policy、lock/frozen fixture、Shell/non-interactive validation、Corepack compatibility exception、rollback ledger。
**Addresses:** project version contract、global fallback priority、package-manager owner。
**Avoids:** Corepack 隐式更新、Node major 分发差异、真实项目 install、提前移除 fallback。
**Gate:** 保持当前 live fallback 不变，直到 isolated fixture 与只读 live evidence 都通过；研究快照不触发升级。

### Phase 4: Go Governance

**Rationale:** Go 已有 mise fallback，但 `GOTOOLCHAIN=auto` 可能引入第二个隐式 toolchain owner。
**Delivers:** exact mise version contract、`go.mod`/`toolchain` interpretation、tracked `GOTOOLCHAIN=local` policy、隔离 module/cache fixture、rollback ledger。
**Addresses:** effective toolchain、project minimum/selected version、no-auto-download check。
**Avoids:** `go env -w`、自动 toolchain 下载和真实 GOPATH/cache 污染。

### Phase 5: Python and uv Governance

**Rationale:** uv 同时能选 Python、下载 interpreter、建环境和 exact sync，必须先划清 mutation boundary。
**Delivers:** Home Manager-owned uv binary + uv-owned Python/project state；`requires-python`、exact `.python-version`、`uv.lock`；manual download policy；isolated frozen fixture。
**Addresses:** interpreter provenance、project venv、dependency lock、无默认 global Python fallback。
**Avoids:** mise/Homebrew/pyenv 重复 owner、system Python、真实项目 exact sync 或全局 pip。

### Phase 6: Rust and rustup Governance

**Rationale:** rustup 有多层 override，安装了 rustup 并不证明项目使用预期 toolchain。
**Delivers:** Home Manager-owned rustup binary、rustup-owned exact toolchain/component/target contract、`rust-toolchain.toml`、Cargo locked fixture、override inspector。
**Addresses:** proxy provenance、project minimum version 与 selected toolchain 分离。
**Avoids:** floating `stable`、`rustup self update`、无证据的 global default 和真实 Cargo/Rustup state 污染。

### Phase 7: Deno and Bun Governance

**Rationale:** 两者都能产生 lock/cache/install state；当前 Bun 仍有 transitional direct path，不能直接删除。
**Delivers:** on-demand mise ownership、exact project contract、Deno/Bun frozen fixture、Node/Deno/Bun 主执行路径规则、direct Bun migration/rollback ledger。
**Addresses:** 无默认 global fallback、cache isolation、multi-runtime conflict evidence。
**Avoids:** `deno upgrade`、`bun upgrade`、Bun 自动迁移其他 package-manager lockfile，以及访问现有 Bun state。

### Phase 8: JVM, Maven, and Gradle Governance

**Rationale:** JDK、Gradle daemon/toolchain 与 Maven toolchain 可以各自选择不同 Java，必须先盘点兼容性。
**Delivers:** mise-owned exact vendor/patch JDK project policy；Gradle/Maven Wrapper ownership；wrapper checksum；isolated user homes；单 JDK 与 multi-JDK exception。
**Addresses:** build-tool reproducibility、`JAVA_HOME`/daemon/toolchain evidence、无全局 Gradle/Maven。
**Avoids:** 过早设定 universal JDK、wrapper 自动下载冒充 check、提交机器绝对路径。

### Phase 9: Multi-host Composition and Binder Spike

**Rationale:** ownership contract 稳定后才能安全把单主机 Nix composition 拆成 shared baseline 与 profile；Git flake source boundary 必须先证明。
**Delivers:** public common/role/logical-host composition；synthetic hosts；identity-only local binder design；clean Git-source evaluation；privacy/store-leak tests；迁移回滚。
**Addresses:** shared baseline、logical profile、architecture capability、expected divergence。
**Avoids:** ignored `.nix` import、routine `--impure`、真实 identifier 入 Git/report、secret 进入 Nix store。
**Gate:** 在 synthetic/clean source 验证前不替换当前可工作的 concrete host composition，也不宣称其他架构受支持。

### Phase 10: Recovery Observation and Readiness Model

**Rationale:** 在增加真实写入前，先让系统完整、无副作用地描述 desired 与 observed。
**Delivers:** component、link、toolchain、secret-obligation manifests；desired projector；Nix/Homebrew/HM/defaults/service/link/private/manual adapters；sanitized JSON/Markdown readiness report。
**Addresses:** canonical inventory、machine-readable check、evidence-level report、manual/excluded state。
**Avoids:** duplicated desired values、raw command persistence、whole-home scan、`brew bundle check` system execution。
**Gate:** 仍无 apply；每个 probe 有官方语义与 fixture evidence，否则输出 unknown/manual。

### Phase 11: Fail-closed Symlink Planning and Safe Apply

**Rationale:** 现有 link plane 是独立 writer，且 physical fallback 与 destructive replacement 是当前最具体的恢复风险。
**Delivers:** explicit tracked link manifest；target containment/owner classification；data-only plan；backup-before-replace；idempotency/interruption/restore tests；兼容入口安全委派。
**Addresses:** fail-closed deployment、跨 Shell owner、link receipt/rollback。
**Avoids:** physical `.config` discovery、Home Manager ownership race、`rm -rf`、路径逃逸。
**Gate:** 默认只 plan；任何真实 link apply 均需 exact plan confirmation，且不得成为自动测试的一部分。

### Phase 12: Integrated Recovery Engine

**Rationale:** 所有 owner、adapter、report 和安全 primitive 稳定后，控制面才能协调真实 plane。
**Delivers:** immutable plan digest、precondition recheck、grouped confirmation、non-activation Nix build、privileged switch checkpoint、component receipts、stop-on-failure、component compensation 与 fresh verify。
**Addresses:** 完整 `check → plan → confirm → apply → verify`、plan-to-verify traceability、真实 rollback boundary。
**Avoids:** stale plan、executable plan、全局 sudo、一键 sync、假原子事务、Homebrew cleanup。
**Gate:** apply 不能重新发现 work；未知 operation/adapter/source change 零写入退出；Nix build、switch、Homebrew、service、link 分别有证据。

### Phase 13: Current-host Non-destructive Readiness Drill

**Rationale:** 当前只有一台实体 Mac，必须验证流程但不能把它当 disposable clean host。
**Delivers:** static + isolated + read-only current-host evidence；first-run/manual checklist；missing private/login/TCC state；drift/expected-divergence；最终 current-host readiness report。
**Addresses:** v1 Definition of Done 与诚实的 claim level。
**Avoids:** warm-state masking、自动修复、TCC/login overreach、fresh-install false claim。
**Gate:** 未经单独确认不执行 switch、Homebrew mutation、defaults write、service change、真实 link replace 或 runtime cleanup；最高状态为 `recovery-ready-on-current-host`。

### Future Phase 14: Clean-host Validation

**Rationale:** 只有 macOS VM 或第二台实体 Mac 能发现首次 bootstrap、冷 cache、permission/login 和真实 recovery ordering 缺口。
**Delivers:** documented starting state、fresh binder、staged apply receipts、postcondition report、rollback drill 和 evidence artifact。
**Addresses:** `fresh-install-verified` claim。
**Defer reason:** 当前没有 disposable clean Mac；它不是 v1 recovery-ready 的阻塞项，但没有它绝不能升级声明。

### Phase Ordering Rationale

- Safety Foundation 必须先于任何 ecosystem migration；否则 fixture、probe 或 cleanup 会把唯一工作 Mac 变成实验对象。
- Ownership Inspector 必须先于 removal；重复安装首先是要分类的 evidence，不是自动清理目标。
- 六生态逐个完成 contract → isolated fixture → read-only live evidence → rollback → optional cleanup，保证每次变更有独立停止点和原子提交。
- Multi-host composition 放在生态 ownership 稳定之后，但放在 Recovery Engine 之前；否则恢复层会把现有硬编码 identity 当作长期 API。
- Recovery 先建立 read-only observation/report，再修复最危险的 link plane，最后才引入跨 plane confirmed apply。
- 当前 Mac 演练只证明 current-host readiness；clean-host E2E 作为未来证据阶段，不用虚假自动化填补物理测试缺口。

### Research Flags

Phases likely needing deeper research during planning:

- **Phase 3 — Node/pnpm/Corepack:** 在计划时刷新 Node major、pnpm 与 standalone Corepack 官方兼容矩阵；只有真实项目明确需要时才设计 compatibility variant。
- **Phase 7 — Deno/Bun:** 两者 lockfile/CI/cache 行为变化快，实施前刷新官方文档并验证不会迁移其他 package-manager state。
- **Phase 8 — JVM:** 需要先盘点项目 wrapper 与 JDK vendor/major，再核对 Gradle runtime/toolchain 和 Maven toolchain 兼容矩阵。
- **Phase 9 — Multi-host:** 必须做 Git-flake/local-binder source-boundary spike；候选架构目前是 MEDIUM confidence，不应直接固化。
- **Phase 10 — Recovery observation:** 每个 probe 需逐一核对当前官方副作用语义；Homebrew Brewfile `system` 条目尤其需要专门 guard。
- **Phase 12 — Recovery engine:** content-addressed plan schema、privilege boundary、Homebrew/service compensation 与 interruption behavior 需要 threat-model 和 fixture iteration。
- **Future Phase 14 — Clean-host:** VM 产品、macOS 安装起点与首次 bootstrap 限制届时再研究，不能用当前机器假设替代。

Phases with established patterns; focused official-doc refresh is enough:

- **Phase 1 — Safety foundation:** schema validation、fake adapters、temporary roots、sentinel 和 pre-write privacy scan 都是清晰模式。
- **Phase 2 — Ownership inspector:** 主要是 read-only normalized evidence 与 source precedence；不涉及写入。
- **Phase 4 — Go:** 官方 `go`/`toolchain`/`GOTOOLCHAIN` 语义明确；只需刷新具体版本和验证 flags。
- **Phase 5 — Python/uv:** uv ownership、managed Python 与 frozen project contract 有完整官方文档；重点是隔离测试。
- **Phase 6 — Rust/rustup:** override precedence、exact toolchain 与 state roots 明确。
- **Phase 11 — Symlink safety:** manifest allowlist、canonical containment、backup/restore 与 fail-closed 原则成熟，但真实 apply 仍需人工 gate。
- **Phase 13 — Readiness drill:** evidence grading 与 manual/excluded 状态已经明确，重点是执行纪律而非新技术选择。

## Confidence Assessment

| Area | Confidence | Notes |
|---|---|---|
| Stack | HIGH for ownership; MEDIUM for versions/integration | 官方资料支持 manager 职责、lock/wrapper、Corepack 分发变化与 activation 语义；具体版本会漂移，mise + nix-direnv PATH 行为尚需 fixture |
| Features | HIGH for required safety capabilities; MEDIUM for prioritization | table stakes 来自现有仓库缺口与官方副作用；v1/v1.x 边界仍是产品判断 |
| Architecture | HIGH for state/plane separation; MEDIUM for binder/plan implementation | dual-plane、read/write separation 和 Nix source/privacy边界证据强；local binder UX 与 exact plan schema 尚未实现 |
| Pitfalls | HIGH | 绝大多数风险直接来自官方命令语义或现有仓库边界；未来 Intel/VM 与 cross-plane compensation 尚缺实测 |

**Overall confidence:** MEDIUM

安全约束、ownership direction 和 roadmap ordering 可以直接进入 requirements；多主机 binder、具体 control-plane schema、跨组件 rollback 与 clean-host 结论必须通过后续 spike/fixture/实机证据升级，不能仅凭研究写成已验证能力。

### Gaps to Address

- **Control-plane implementation language与 schema 字段：** 研究只确定边界，不应在 requirements 阶段凭偏好锁死；Phase 1 以最小 prototype、schema test 和 threat model 选择。
- **Local host binder 的 Nix source/store 行为：** Phase 9 用 synthetic identity、clean Git source 和 store/privacy scan 验证；失败时保留当前单主机 pure composition。
- **Corepack compatibility 的真实需求：** 当前只定义 exception，不假设任何项目必须使用；遇到真实项目时再核对 Node/Corepack engine 与 package-manager contract。
- **Nix devShell mixed mode：** 必须证明每个 declared executable 只有一个 owner，以及进入/退出 shell 后 PATH 恢复；未通过前优先显式 `nix develop`。
- **JDK inventory：** 尚无足够证据选择 universal JDK；先记录真实项目 wrapper/vendor/major 需求，不添加全局 Java/Gradle/Maven fallback。
- **Probe side effects：** 每个 adapter 都要有 allowlist 和 negative fixture；generic Brewfile check、wrapper download、manager auto-install 不得默认执行。
- **Component rollback：** Nix generation、Homebrew package/cask/service、Home Manager file 和 link backup 的补偿能力不同；Phase 12 必须逐 operation 证明，而不是提供全局 rollback flag。
- **真实项目覆盖：** 仓库 fixture 只能证明 contract；未经单独授权不批量修改外部项目，遇到项目时逐个验证和迁移。
- **Architecture/clean-host evidence：** 当前只支持并可现场核对现有 Apple Silicon host class；Intel、Rosetta、VM 和 fresh install 均保持 unverified。
- **版本时效：** 所有 exact upstream version 都是研究日期快照；每个生态计划开始前重新读取官方 release/compatibility 文档，不自动升级当前 live fallback。

## Sources

四份详细研究均以官方文档或官方仓库为主要技术依据：

- [Stack research](./STACK.md) — tool ownership、project contract、activation 与版本快照。
- [Feature research](./FEATURES.md) — table stakes、differentiator、anti-feature 与 v1 边界。
- [Architecture research](./ARCHITECTURE.md) — state model、dual-plane control、binder、typed plan 与 report。
- [Pitfalls research](./PITFALLS.md) — destructive behavior、privacy、trust、test isolation 与 phase gates。

### Primary (HIGH confidence)

- [Nix flakes and lock model](https://nix.dev/manual/nix/2.26/command-ref/new-cli/nix3-flake.html) — Git/source purity、locking 与 evaluation boundary。
- [Nix store secrets guidance](https://nix.dev/manual/nix/2.34/store/secrets) — secret value 不应进入 derivation/store。
- [nix-darwin configuration options](https://nix-darwin.github.io/nix-darwin/manual/) — Homebrew activation、cleanup mode、system/Home Manager integration。
- [Home Manager manual](https://nix-community.github.io/home-manager/) — user configuration composition、build/switch 与 activation ownership。
- [Homebrew Bundle documentation](https://docs.brew.sh/Brew-Bundle-and-Brewfile) — inventory、cleanup 与 Brewfile `system` 在 check 时执行的边界。
- [mise configuration](https://mise.jdx.dev/configuration.html), [lockfiles](https://mise.jdx.dev/dev-tools/mise-lock.html), and [trust](https://mise.jdx.dev/cli/trust.html) — precedence、pinning、auto behavior 与 project trust。
- [uv Python management](https://docs.astral.sh/uv/concepts/python-versions/) and [project sync](https://docs.astral.sh/uv/concepts/projects/sync/) — interpreter discovery/download 与 exact/frozen sync。
- [rustup override precedence](https://rust-lang.github.io/rustup/overrides.html) — project/default/toolchain resolution。
- [Go toolchain selection](https://go.dev/doc/toolchain) — `go`/`toolchain` directives 与 automatic download。
- [Node Corepack documentation](https://nodejs.org/download/release/v25.8.0/docs/api/corepack.html) and [Corepack upstream](https://github.com/nodejs/corepack) — Corepack distribution/compatibility and explicit project contract。
- [Gradle Wrapper](https://docs.gradle.org/current/userguide/gradle_wrapper.html), [Gradle JVM toolchains](https://docs.gradle.org/current/userguide/toolchains.html), [Maven Wrapper](https://maven.apache.org/tools/wrapper.html), and [Maven toolchains](https://maven.apache.org/guides/mini/guide-using-toolchains) — build-tool and JDK ownership separation。
- [direnv manual](https://direnv.net/man/direnv.1.html) — explicit allow/deny security boundary。
- [Apple Privacy & Security settings](https://support.apple.com/guide/mac-help/change-privacy-security-settings-on-mac-mchl211c911f/mac) — TCC and permissions remain user-controlled manual state。
- [GitHub sensitive-data removal](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository) — rotation/history remediation when a value reaches Git。

### Secondary (MEDIUM confidence)

- [chezmoi command model](https://www.chezmoi.io/user-guide/command-overview/) — only a UX/design precedent for separating status/diff/apply/verify; it is not recommended as a new owner in this repository。
- Repository codebase maps and current configuration — reliable for the observed brownfield boundary at research time, but live machine state and external versions must be refreshed before implementation。

### Tertiary (LOW confidence)

- None. Unverified implementation choices are recorded as gaps/research flags rather than promoted to technical conclusions.

---
*Research completed: 2026-07-10*
*Ready for requirements: yes*
*Ready for roadmap: after requirements confirmation*
