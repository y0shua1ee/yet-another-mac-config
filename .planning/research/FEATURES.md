# Feature Research

**Domain:** 公开 Git 同步的个人 macOS 配置、开发工具链治理与恢复就绪系统
**Researched:** 2026-07-10
**Confidence:** HIGH（官方工具能力）；MEDIUM（复杂度、优先级与 v1 边界属于基于本仓库现状的产品判断）

## Research Frame

本项目不是从零设计一个通用 dotfiles 产品，而是在已有 Nix/nix-darwin/Home Manager、Homebrew inventory、应用原生配置和交互式软链接脚本之上补齐两层能力：先统一六类开发生态的所有权与项目契约，再建立安全的多 Mac 恢复就绪流程。

这里的核心价值不是“让每台机器字节完全相同”，而是让 Git 中公开、可声明的期望状态能够被检查、预览、确认、应用和验证，同时把秘密、登录状态、TCC 权限和不可移植应用数据留在安全边界之外。下列复杂度均指在当前 brownfield 仓库中集成、迁移、验证和写文档的综合成本，不只是编写一个脚本的成本。

## Feature Landscape

### Table Stakes (Users Expect These)

缺少这些能力时，这个仓库只能算“配置收藏”，还不能算安全的恢复来源。

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Canonical desired-state inventory | 恢复系统必须能回答“哪些公开状态由仓库负责” | MEDIUM | 覆盖 Nix、Home Manager、Homebrew formulae/casks/fonts/services、Shell、受跟踪 app config、稳定 defaults、CLI、符号链接与六类工具链；每项记录 owner、scope、check、apply、verify、rollback 和允许差异。 |
| 六生态所有权矩阵 | 多个 manager 同时提供同一 runtime 时，PATH 与项目行为不可预测 | HIGH | 对 Node/npm/pnpm/Corepack、Go、Python/uv、Rust/rustup、Deno/Bun、JVM/Maven/Gradle逐项规定唯一主 owner、项目契约、全局 fallback、系统库边界及冲突判定。mise 可以管理适合的 runtime，但不能取代生态 lockfile、wrapper 或专用 manager。 |
| 项目级版本与依赖契约 | 同一台 Mac 上的不同项目必须能选择不同版本并可复现 | HIGH | 采用各生态的原生文件：例如 `mise.toml`/`.tool-versions`、`packageManager` 与 lockfile、`go.mod`/`toolchain`、`pyproject.toml`/`uv.lock`/`.python-version`、`rust-toolchain.toml`、`deno.json`/`deno.lock`、`bun.lock`、Maven/Gradle Wrapper 与 JVM toolchain。 |
| 可解释的解析与冲突检查 | 仅看到一个版本号不足以证明命令来自预期 owner | MEDIUM | 报告契约来源、配置优先级、最终 executable path、版本、重复候选和 PATH shadowing；`mise config`/`mise ls --current` 一类能力证明解析链应可见。检查不能触发 install。 |
| 明确的项目环境加载边界 | 进入目录时自动切换很方便，但隐式执行不可信配置很危险 | MEDIUM | direnv 必须保留显式 `allow` 安全门；mise/direnv/Nix devShell 的职责不能重叠。系统库和复杂编译依赖可由 `nix develop` 提供，语言依赖仍由项目契约管理。 |
| 共享基线 + 隐私安全主机覆盖层 | 多 Mac 既有公共配置，也必然存在架构、角色和硬件差异 | HIGH | 首版使用逻辑 profile 标识，不把真实用户名、hostname、序列号或硬件指纹固化为公共选择键；主机本地 selector 可 ignored，公共 profile 只表达非敏感差异。chezmoi 与 Home Manager 的官方模型都证明多机共享加局部差异是基础需求。 |
| 非秘密 manifest 与 local overlay | 公共仓库必须知道“需要什么”，但不能保存“秘密值是什么” | MEDIUM | Git 只记录秘密项 ID、用途、是否必需、目标消费方、存在性检查和抽象 provider reference；真实值留在 ignored file、Keychain 或密码管理器。所有检查和报告默认只输出 present/missing/invalid-metadata，不读取或打印值。 |
| 只读 `check` / drift discovery | 在当前唯一一台工作 Mac 上，第一步必须不会改变任何状态 | HIGH | 检查声明、文件类型/链接目标、版本/路径、Homebrew 满足情况、服务/权限/登录/overlay 缺口和 extra state。检查命令要排除 mise 自动安装、uv exact sync、wrapper 下载和任何 `switch`/`defaults write`/service mutation。 |
| 完整、稳定、机器可读的 `plan` | 用户必须在改变机器前看清新增、替换、保留、潜在删除和人工步骤 | HIGH | 汇总两条现有 activation plane，而不是只显示 Nix diff；每个 action 含 owner、风险、前置条件、是否可逆、backup、rollback、预计外部下载和人工确认。plan 应可保存或计算摘要，防止确认后状态悄悄变化。 |
| 明确 `confirm` gate | `apply` 可能安装 app、修改 defaults、启动服务或替换链接 | MEDIUM | 默认无 apply；确认必须针对具体 plan，而不是模糊的“继续吗”。非交互模式也需要显式 apply flag/plan identity，不能把 CI 或测试中的成功检查升级成真实激活。 |
| 分组件、可停止的 `apply` | 恢复工具最终需要能落地公开状态 | HIGH | v1 不承诺跨 Nix、Homebrew、Home Manager、defaults 与文件链接的全局原子事务；按组件顺序执行、在失败点停止、记录已完成 action，并把危险删除/cleanup 留在单独确认步骤。 |
| 备份、回滚与幂等性说明 | 已有配置不能因一次失败或误选永久丢失 | HIGH | 文件冲突优先 rename/copy backup，禁止 `rm -rf` 式替换；Nix generation、Home Manager 文件备份、Homebrew 外部副作用和手工权限分别记录真实回滚边界。重复执行默认应收敛或明确报告为何不能重入。 |
| `verify` 与 machine-readiness report | “命令退出 0”不等于工作站已经可用 | HIGH | 分类为 declared+verified、declared+not-applied、drift、missing-secret-overlay、missing-login/permission、manual-action、expected-divergence、excluded。必须分别验证 Nix plane、symlink plane、Shell 路径和 app/runtime 后置条件。 |
| 隔离 fixtures 与统一验证入口 | 没有可重复测试就无法安全演进六生态契约和恢复状态机 | HIGH | fixture 可覆盖六生态及失败路径，但必须使用临时 HOME、XDG、mise data/cache、uv cache、Cargo/Rustup、Deno/Bun、Maven、Gradle 等目录；不得把真实项目或真实全局状态当 fixture。不能隔离的检查降级为只读报告或人工验证。 |
| Fail-closed 的受跟踪配置发现 | 公共仓库不能因为 Git 不可用就枚举、链接 ignored/untracked 私有状态 | MEDIUM | 配置来源必须来自明确 manifest 或成功的 tracked-file 查询；查询失败或为空就停止并说明，不得回退到物理目录扫描。 |
| 文档、来源与隐私质量门 | 恢复仓库本身必须可审查、可维护、可公开 | MEDIUM | 每次配置变更同步 README 与局部指导，记录第三方来源/锁定方式，执行 diff hygiene、个人标识检查和 secret scan；测试日志与 plan 同样需要 redaction。 |

### Six-Ecosystem Contract Expectations

这张表描述“功能必须支持什么”，不是在 FEATURES 研究中提前锁死全部实现细节。最终 ownership 仍应在相应阶段结合现有项目和官方行为确认。

| Ecosystem | Project Contract Expected in v1 | Ownership/Validation Nuance | Confidence |
|-----------|---------------------------------|-----------------------------|------------|
| Node/npm/pnpm/Corepack | Node version、唯一 package manager、`packageManager`/lockfile、frozen install/check、global fallback | Corepack 的分发方式随 Node major 不同；Node 20 文档描述 bundled Corepack，而 Node 25 文档引导既有 bundled 用户改用 userland module，因此不能把“Node 自带 Corepack”写成长期不变假设。 | HIGH |
| Go | Go version、`go.mod` 的 `go`/`toolchain` 语义、module verification、global fallback | Go 自身也会进行 toolchain selection；若 mise 是 runtime owner，治理层必须检测并明确 Go 自动下载策略，避免第二个隐式 owner。 | HIGH |
| Python/uv | `requires-python`、`.python-version`、`pyproject.toml`、`uv.lock`、locked check、项目 venv | uv 可自行发现/下载 Python，且 `uv sync` 默认 exact，会移除 lockfile 外包；只能在项目隔离环境运行，不能指向 system Python 或真实共享环境做 fixture。 | HIGH |
| Rust/rustup | `rust-toolchain.toml`、components/targets、Cargo lockfile、default fallback | rustup 有明确 override 优先级；检查必须报告环境变量、directory override、toolchain file 与 default 中哪个生效。 | HIGH |
| Deno/Bun | runtime version、`deno.json`/`deno.lock` 或 `package.json`、`bun.lock`、frozen/CI 模式 | Deno config 与 lockfile、Bun lockfile都属于项目 owner；生成 lockfile仍可能写 cache，测试必须重定向 cache/data。一个项目若同时兼容 Node/Deno/Bun，必须显式选主执行路径。 | HIGH |
| JVM/Maven/Gradle | JDK vendor/version policy、Maven/Gradle Wrapper、wrapper checksum、Java toolchain | Wrapper 固定 build-tool 版本但会下载分发包；JDK 需求与本机安装路径是另一层。fixture 需隔离 `MAVEN_USER_HOME`/`GRADLE_USER_HOME`，项目优先调用 `./mvnw`/`./gradlew`。 | HIGH |

### Differentiators (Competitive Advantage)

这些不是一般 dotfiles 工具天然提供的能力，但它们直接服务本项目“不泄密、不伤现有环境、仍能恢复”的核心价值。

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| 跨 manager ownership conflict detector | 在迁移前发现 Homebrew、Home Manager、mise、Nix devShell、rustup/uv 和系统路径的重复所有权 | HIGH | 不自动修复；给出证据、影响、推荐 owner 和可选清理动作。对当前已能工作的 Mac，比直接重装更安全。 |
| 跨 activation-plane 统一 plan | 用户一次看到 Nix/Homebrew/defaults 与 app symlink/Hammerspoon/fallback shell 的完整影响 | HIGH | 复用各工具的 read-only 能力，标准化 action schema；不声称底层工具形成一个事务。 |
| 证据分级 readiness report | 把“仓库声明了”“本机应用了”“运行验证过”“只缺人工权限”区分开 | HIGH | 每项带 timestamp、check kind、evidence summary、scope 和 claim level；报告不包含秘密值或稳定设备标识。 |
| 安全声明等级 | 避免把单机演练包装成全新安装或多机验证 | LOW | v1 最高只标记 `recovery-ready on existing Apple Silicon Mac`；只有 VM/第二台干净 Mac 通过后才能升级 claim。 |
| 渐进式六生态迁移 ledger | 每个生态先盘点、契约、fixture、只读验证、回滚，再允许移除旧入口 | HIGH | 记录迁移前 owner、候选 owner、验证证据、rollback 和 cleanup 是否获批；每生态独立提交和停止点。 |
| 删除/cleanup 的 advisory mode | 保留 Homebrew extra package 和旧 runtime，同时能清楚看到差异及未来回收成本 | MEDIUM | 默认只报告；只有新路径验证通过、备份/回滚齐备且单独确认后才生成精确删除命令。 |
| Shell 双路径等价验证 | 避免 Home Manager 主 Shell 可用而 fallback Shell 在恢复或回滚时缺少 mise/direnv/路径 | MEDIUM | 在隔离 HOME 比对关键 PATH 顺序、runtime resolution、local overlay ordering 和 alias/function contract。 |
| 隐私安全的 host selection | 公共 Git 可表达多机角色又不暴露真实 hostname/用户名 | MEDIUM | 公共 logical profile + ignored local selector；报告只输出 logical ID 和必要的平台类别。 |
| 安全 fixture matrix | 在没有第二台 Mac 时仍能验证六生态配置解析、冲突检测和状态机负路径 | HIGH | 以最小项目文件为 fixture，禁止访问真实 HOME/项目；对会下载或写 cache 的工具显式重定向目录或只做静态解析。 |
| Manual-step-as-data | 将 TCC、App Store/Apple ID 登录、Keychain/secret overlay、重启等不可自动化步骤变成可检查状态 | MEDIUM | 提供说明、入口和 completed/blocked 状态，但不绕过系统安全，也不收集登录内容。 |
| Plan-to-verify traceability | 应用后能逐条证明计划中的 action 是否达到预期 | HIGH | action ID 从 plan 贯穿 apply log 和 verify result；失败时只建议下一安全动作，不自愈真实机器。 |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Silent cleanup / “make exact” | 看起来能最快消除漂移 | Homebrew 官方说明 `brew bundle cleanup --force` 会卸载 Brewfile 外的依赖，nix-darwin 的 `uninstall`/`zap` cleanup 也会删除额外包；这会把用户有意安装但尚未声明的工具当垃圾。uv exact sync 同样会移除环境中的额外包。 | 默认报告 extras；cleanup 进入独立、逐项、可预览且有回滚的确认步骤。 |
| 把秘密值同步进 Git（即使仓库暂时 private） | 新机看似可以一步恢复所有凭据 | 凭据会进入 Git 历史、clone、fork、日志和备份面；GitHub 明确将 hardcoded credentials 视为可被滥用的泄露风险。加密文件还引入密钥恢复和误解密输出风险，不符合首版最小依赖。 | Git 只存 manifest、模板、provider reference 与 presence check；值留在 ignored local storage、Keychain 或以后可选的 password-manager provider。 |
| Byte-for-byte Mac cloning | “完全一致”听起来比“尽可能一致”更可靠 | 会混入登录会话、应用数据库、聊天/浏览器历史、缓存、硬件相关配置和设备身份；不同架构/角色也需要合法差异。即便 Home Manager能复现它负责的 home 内容，也不等于复制整台 Mac。 | 对公开、声明式 scope 做语义等价验证，维护 expected-divergence 与 excluded 清单。 |
| Unconfirmed apply / `init --apply` 式一步执行 | 新机操作最少 | 将网络下载、defaults、服务、文件替换和 package install 合并成不可审查动作；当前唯一一台工作 Mac 无法承受误判。 | 强制 `check → plan → confirm(plan identity) → apply → verify`，测试永远止于 plan 或隔离环境。 |
| One-shot migration of all six ecosystems | 一次清理所有旧 manager，表面进度快 | PATH、hooks、CI、编辑器、真实项目和 shell fallback 之间存在隐性依赖；一个生态失败会扩大故障面并削弱回滚。 | 每生态独立盘点、契约、fixture、验证和回滚，稳定后再处理下一个。 |
| 强迫所有工具都由 mise 管理 | 单一命令和配置文件看起来简单 | uv、rustup、Go toolchain selection、Deno/Bun lockfile、Maven/Gradle Wrapper 都有不可替代的项目语义；强制统一会产生第二 owner 或丢失官方工作流。 | 统一治理 schema 和冲突规则，不统一成一个实现工具；每项只有一个明确主 owner。 |
| 自动授予/绕过 TCC、Accessibility、Full Disk Access | 减少人工步骤 | Apple 的安全模型要求用户透明、同意与控制，Accessibility/Automation 需要用户授权；绕过它会削弱系统保护，也不能可靠跨机复现。 | readiness 报告缺失权限，并指向人工 System Settings 步骤。 |
| Check/verify 自动修复真实机器 | 让所有检查“自动变绿” | 只读命令可能因此触发 mise auto-install、uv sync、wrapper download、service restart 或 config rewrite；测试结果不再能证明无副作用。 | check 只采证；fix 只能形成 plan，真实 apply 必须重新确认。 |
| 把多组件 apply 宣称为原子事务 | 用户希望一个失败能全部回滚 | Nix generation 无法自动撤销 Homebrew 服务、cask 副作用、TCC 或手工登录；软链接替换也有独立失败面。 | 分组件 action log、备份、停止点和真实 rollback contract，明确哪些步骤只支持 forward repair。 |
| 失败时回退扫描物理 `.config` 目录 | 在没有 Git metadata 时仍能“继续工作” | 可能枚举并链接 ignored/untracked 的凭据、缓存和应用状态，破坏 version-control 隐私边界。 | manifest/tracked discovery 失败即 fail closed；要求恢复正确 clone/worktree 后重试。 |
| 运行未锁定的远程 installer/script | Bootstrap 命令更短 | moving branch 或远程内容在审查后仍可变化，且常在仓库 lockfile 接管之前以高权限执行。 | 固定、校验、下载后审查；在 plan 中显示 URL、revision/hash 和权限要求。 |
| 无干净主机证据却标记 fresh-install verified | 让项目显得完成度更高 | 当前本机已经具备依赖与历史状态，可能掩盖 bootstrap 缺口、顺序依赖和登录/权限问题。 | v1 只声明 recovery-ready；VM 或第二台干净 Mac 通过端到端验收后再升级 claim。 |
| 首版强依赖 1Password 或任一 secret provider | 自动注入秘密体验更顺滑 | 引入额外 app、CLI、账号登录与 provider 可用性，反而会阻塞基本公开状态恢复。 | v1 使用 provider-neutral manifest；v1.x/v2 再添加可选 adapter。 |

## Feature Dependencies

```text
[隐私/身份边界]
    ├──requires-before──> [共享基线 + logical host overlays]
    └──requires-before──> [非秘密 manifest + redacted reports]

[六生态只读现状盘点]
    └──requires──> [所有权矩阵]
                       └──requires──> [项目契约]
                                          └──requires──> [隔离 fixture matrix]
                                                               └──requires──> [逐生态迁移]
                                                                                    └──enables──> [经确认的重复入口清理]

[Canonical inventory] + [Fail-closed source discovery] + [Read-only probes]
    └──requires──> [check]
                       └──requires──> [stable machine-readable plan]
                                          └──requires──> [confirm exact plan]
                                                             └──requires──> [component apply]
                                                                                └──requires──> [verify + readiness report]

[backup/rollback contracts] ──required-before──> [component apply]
[ownership matrix] ──required-before──> [recovery inventory for toolchains]
[clean VM / second Mac] ──enhances──> [claim level: fresh-install verified]

[silent cleanup] ──conflicts──> [existing-environment safety]
[secret values in Git] ──conflicts──> [public source-of-truth]
[unconfirmed apply] ──conflicts──> [check → plan → confirm → apply → verify]
[byte-for-byte cloning] ──conflicts──> [privacy boundary + expected divergence]
```

### Dependency Notes

- **工具链治理必须先于恢复 readiness：** 如果同一 runtime 仍有多个 owner，恢复层无法判断应安装、保留还是报告哪一个版本，plan 也无法可信。
- **只读盘点必须先于 ownership 变更：** 当前 Mac 是唯一工作环境；未先记录 executable path、版本、配置来源和 hooks 就删除旧入口，会失去回滚基线。
- **项目契约必须先于 fixture：** fixture 应证明明确的规则，而不是把当前偶然 PATH 行为固化为测试。
- **fixture 必须先于真实迁移：** 六生态均存在自动下载、cache 写入或 exact sync 行为；隔离负路径能够先验证冲突和停止条件。
- **隐私边界必须先于多主机结构：** 否则会继续把真实用户名/hostname 当公共配置 key，后续再清理 Git 历史成本更高。
- **inventory、fail-closed discovery 和 read-only probes 共同构成 check：** 只有 manifest 没有 live evidence 不能识别漂移；只有 live scan 没有 manifest 又会扩大到私有状态。
- **plan 必须先于 confirm：** 确认必须绑定明确 actions、风险和 rollback；不能确认一个尚未计算的动态流程。
- **backup/rollback contract 必须先于 apply：** 对无法自动回滚的 Homebrew、服务、TCC 等步骤，要在 plan 中明确 forward-repair 或人工恢复，而不是承诺虚假的原子性。
- **verify 依赖 action trace：** 每个计划 action 要有稳定 ID，才能区分“未执行”“执行失败”“执行成功但后置条件失败”。
- **VM/第二台 Mac 不是 v1 recovery-ready 的前置条件：** v1 可在现有 Mac 完成静态、隔离和非破坏演练；但它是升级到 fresh-install verified 声明的必要证据。

## MVP Definition

### Launch With (v1)

v1 的顺序不可交换：先完成工具链治理，再完成恢复就绪。即使同一里程碑交付，也应拆成独立阶段和原子提交。

#### A. Toolchain Governance First

- [ ] **统一 ownership schema 与六生态矩阵** — 每个 runtime/tool/package manager 只有一个主 owner，并记录 global fallback、project override、system-library 和 native-manager 边界。
- [ ] **六类项目契约** — 每类都有版本文件/lockfile/wrapper 规则、进入目录行为、只读检查、验证命令、迁移步骤和回滚说明。
- [ ] **Read-only conflict inventory** — 报告 PATH、版本、来源和重复候选；不得安装、切换或删除。
- [ ] **隔离 fixture matrix** — 六类生态至少各有一个正常 fixture 和关键负路径；使用临时 HOME/XDG/tool-specific data/cache，禁止真实机器 mutation。
- [ ] **逐生态迁移 ledger** — 按“盘点→owner→契约→fixture→只读本机检查→回滚→可选 cleanup”推进；一个生态稳定后再进入下一个。
- [ ] **全局 fallback 与项目优先级验证** — 保证无项目契约时有可用基线，有项目契约时项目配置胜出；验证主 Shell 与 fallback Shell 的关键行为。

#### B. Recovery Readiness Second

- [ ] **共享基线 + logical host overlay** — 当前只有一个公共逻辑 profile 和一个 ignored local selector，但结构可扩展到不同角色/架构。
- [ ] **Canonical public inventory + non-secret manifest** — 列清仓库负责、人工负责和明确排除的状态；秘密只记录 metadata/presence。
- [ ] **Fail-closed discovery** — 只从受跟踪/显式 manifest 来源生成配置动作，消除物理目录 fallback。
- [ ] **统一 `check` 与机器可读 `plan`** — 聚合 Nix/Homebrew/Home Manager、symlink、Shell、defaults、服务、工具链和人工步骤；默认绝不 apply。
- [ ] **Plan-bound confirmation 与组件化 apply** — 仅显式授权后执行；文件替换先备份；删除与 cleanup 需要额外确认；失败时停止并保留 action log。
- [ ] **Post-apply verify + readiness report** — 输出证据分类、漂移、秘密/权限/登录缺口、人工动作、允许差异和排除项，不包含秘密或真实设备标识。
- [ ] **当前 Mac 非破坏演练** — 验证 check/plan/fixture/报告与可安全隔离的 apply 路径；不执行未经用户授权的 switch、Homebrew mutation、defaults write、服务变更或真实链接替换。
- [ ] **诚实的 v1 claim** — 只发布“当前 Apple Silicon Mac 上 recovery-ready”；明确尚未在干净 VM/第二台 Mac 验证。

### v1 Definition of Done Boundary

v1 可以证明：仓库能在当前 Mac 上无副作用地发现和解释期望状态与漂移；六生态契约能在隔离 fixture 中运行；真实变更只能来自经确认的稳定 plan；应用后有逐项验证和真实回滚说明。

v1 不需要证明：全新 macOS 从零自动安装、Intel Mac 兼容、所有 GUI 登录、所有 TCC 权限、秘密值恢复、浏览器/聊天/应用数据库恢复、跨组件原子回滚或真实业务项目的批量迁移。

### Add After Validation (v1.x)

- [ ] **可选 secret-provider adapter** — 仅当 provider-neutral manifest 稳定且用户希望减少手工步骤时，增加 1Password/Keychain 等适配器；仍禁止日志输出秘密。
- [ ] **安全可导出的 app state inventory** — 在逐 app 隐私审计后，考虑扩展/插件列表、主题和公开偏好，不同步账号或数据库。
- [ ] **可恢复的 apply journal/resume** — 当 v1 action log 已证明足够稳定，再支持从明确失败点继续，而不是重新执行整个流程。
- [ ] **更完整的 native validator 聚合** — 将 Ghostty、Neovim、AeroSpace、Yazi、Hammerspoon 等现有分散检查纳入统一验证入口；GUI/TCC 项保留人工门。
- [ ] **持续集成的静态/隔离门** — 在不接触真实 Mac 状态的 runner 上执行 schema、fixture、shell syntax、Nix eval 和 privacy checks；不能把 CI 当作整机 activation 证据。
- [ ] **更多 logical profiles** — 实际出现第二台 Mac 或明确角色差异时再增加 profile，避免为假想主机过度抽象。

### Future Consideration (v2+)

- [ ] **干净 macOS VM 端到端演练** — 具备可用虚拟化条件后验证 bootstrap 顺序、首次下载、登录/权限清单和回滚；通过后才提升 claim。
- [ ] **第二台实体 Mac 与不同架构验证** — Apple Silicon 第二机优先；若确有 Intel Mac，再加入并验证 `x86_64-darwin`，不提前承诺。
- [ ] **多机/fleet dashboard** — 个人使用且只有一台机器时价值低；等 profile 与 readiness report 在真实多机上稳定后再评估。
- [ ] **可视化 plan/recovery UI** — CLI/机器可读 schema 应先成熟；否则 UI 会掩盖底层 action 和安全边界。
- [ ] **真实项目批量迁移工具** — 本仓库只提供规则和 fixture；只有拥有明确授权、项目 inventory 与回滚策略后才可跨仓库批量修改。
- [ ] **选择性 destructive convergence** — 只在多机实践证明 extra-state policy 足够精确后，才考虑 opt-in cleanup profile；永远不设为默认。

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| 六生态 ownership matrix | HIGH | HIGH | P1 |
| 六生态项目契约 | HIGH | HIGH | P1 |
| Read-only ownership/path conflict inventory | HIGH | MEDIUM | P1 |
| 隔离 fixture matrix | HIGH | HIGH | P1 |
| 渐进迁移与 rollback ledger | HIGH | HIGH | P1 |
| Canonical public inventory | HIGH | MEDIUM | P1 |
| Shared baseline + logical host overlay | HIGH | HIGH | P1 |
| Non-secret manifest + redaction | HIGH | MEDIUM | P1 |
| Fail-closed config discovery | HIGH | MEDIUM | P1 |
| Unified check + machine-readable plan | HIGH | HIGH | P1 |
| Plan-bound confirm + component apply | HIGH | HIGH | P1 |
| Backup/rollback contracts | HIGH | HIGH | P1 |
| Verify + readiness report | HIGH | HIGH | P1 |
| Safe claim levels | MEDIUM | LOW | P1 |
| Shell-route equivalence checks | HIGH | MEDIUM | P1 |
| Optional secret-provider integration | MEDIUM | MEDIUM | P2 |
| Apply journal/resume | MEDIUM | HIGH | P2 |
| App validator aggregation | MEDIUM | MEDIUM | P2 |
| Exportable app state | MEDIUM | HIGH | P2 |
| Clean VM E2E | HIGH | HIGH | P2（条件具备后） |
| Intel Mac profile | LOW（当前） | HIGH | P3 |
| Fleet dashboard / GUI | LOW（当前） | HIGH | P3 |
| Default destructive convergence | NEGATIVE | HIGH | 不实施 |

**Priority key:**

- **P1:** v1 必须具备，否则不能安全地把仓库称为恢复来源。
- **P2:** 核心流程验证后加入，或依赖外部条件。
- **P3:** 当前只有一台 Mac 时价值不足，避免提前复杂化。

## Competitor / Comparable-System Feature Analysis

本节只比较可借鉴的官方能力，不建议 v1 为了“更像竞品”整体迁移到 chezmoi。当前仓库已经深度使用 Nix/Home Manager/Homebrew 与原生 app config；新增另一个全权 dotfile manager 会制造所有权重叠。更合理的做法是借鉴 chezmoi 的 `status/diff/apply/verify` 交互语义，在现有架构上统一 plan 和证据。

| Feature | Nix/nix-darwin + Home Manager + Homebrew Bundle | chezmoi | Our Approach |
|---------|--------------------------------------------------|---------|--------------|
| 系统/package 声明 | nix-darwin 提供 macOS 声明模块；Homebrew module 生成 Brewfile，Homebrew Bundle 能 `check/install/cleanup` | 主要管理目标文件；package 安装通常经 scripts/外部工具 | 保留现有 Nix/Homebrew owner，不让 dotfile layer重复管理 packages。 |
| Home/config 文件 | Home Manager 可声明 programs、环境变量与任意文件并跨 host 复现 | source state、templates、ignore 和 target state 是核心 | Home Manager 管主 Shell/低风险 user config，现有 symlink plane 管 app-native tree；统一 inventory 而非强行合并。 |
| 多机差异 | Nix module composition 可参数化 host/user/system；当前仓库尚未拆层 | 官方支持 template data、OS/arch/hostname 条件与 machine-local config | 公共 logical profile + ignored selector；避免真实 hostname 进入公共 Git。 |
| 差异预览 | Nix build/eval 与 Homebrew `bundle check` 能覆盖各自范围，但没有跨 plane 总 plan | `status` 总览、`diff` 显示 target 与 destination 差异、`apply --dry-run --verbose` | 聚合每个 owner 的 read-only evidence，生成一份 machine-readable plan。 |
| 应用 | `darwin-rebuild switch`/Home Manager activation/Homebrew Bundle 有真实副作用 | `apply` 使 target 达到目标状态，目标被用户修改时会询问覆盖 | 明确 confirm 后按组件执行；文件冲突备份，危险 cleanup 独立确认。 |
| 验证 | Nix/应用各有 build 或 post-check，但仓库当前分散 | `verify` 以 exit 0/1 表示所有 target 是否匹配 | 在文件匹配之外增加 runtime path、service、permission/manual-step 和双 activation plane readiness。 |
| Secrets | Nix 可接外部 secret systems；本仓库当前使用 ignored local fragments | 官方支持 password-manager template functions 与 encryption | v1 只做 provider-neutral metadata/presence；不要求 1Password，也不把值写进 report。 |
| Cleanup | nix-darwin/Homebrew 可选择 none/check/uninstall/zap；uninstall/zap 真实删除 extras | 可管理 remove state，但仍是显式 target operation | 默认只 report extras；cleanup 永不作为“恢复成功”的必需条件。 |
| Rollback | Nix generations覆盖 Nix-owned部分；Homebrew/TCC/外部文件不自动跟随回滚 | 目标文件可重写，但不是整机事务 | 每个 action 记录独立 rollback/forward-repair，不声称跨组件原子。 |
| 项目 runtime | Nix devShell 能提供构建环境；Homebrew不负责项目版本切换 | 不以语言 runtime ownership 为核心 | mise/uv/rustup/native files/wrappers 按 ownership matrix 合作，并对冲突做一等检查。 |
| 恢复声明 | 各工具只保证自己管理的 scope | 主要保证 managed target state | 明确 `recovery-ready` 与 `fresh-install verified` 两种证据等级。 |

## Product Conclusions

1. **不要引入第三个总管。** 当前组合已经能覆盖 system、user、package 和 app-native config；v1 的缺口是统一 governance、plan、safety 和 evidence，而不是缺少另一个会写入 HOME 的工具。
2. **统一 schema，不统一实现。** 六生态应共享 owner/contract/check/verify/rollback 字段，但保留 uv、rustup、lockfile、Wrapper、Nix devShell 等官方职责。
3. **把自动行为视为 mutation。** mise auto-install、uv sync、Deno/Bun lock/cache、Maven/Gradle Wrapper 下载都会写状态；check 与 fixture 设计必须显式阻断或隔离这些路径。
4. **把“不会删除”作为 v1 正向能力。** Homebrew/nix-darwin 已提供 destructive cleanup 选项，但本项目的核心差异化是先报告 extra state，再由用户逐项决定，而不是默认追求 exact machine state。
5. **让证据决定产品声明。** 当前 Mac 可以验证 recovery readiness；没有干净 VM/第二台 Mac，就不能把 bootstrap 和首次恢复标记为已验证。

## Sources

全部技术能力仅使用当前官方文档或官方仓库；访问日期均为 2026-07-10。

### macOS state, packages, and comparable recovery UX

- [nix-darwin official repository](https://github.com/nix-darwin/nix-darwin) — declarative macOS modules and switch workflow. **Confidence: HIGH**
- [nix-darwin configuration options](https://nix-darwin.github.io/nix-darwin/manual/) — Homebrew integration, idempotent defaults, and `none/check/uninstall/zap` cleanup behavior. **Confidence: HIGH**
- [Home Manager introduction](https://nix-community.github.io/home-manager/introduction.html) — reproducible programs, environment variables, files, and cross-host home configuration. **Confidence: HIGH**
- [Homebrew Bundle and Brewfile](https://docs.brew.sh/Brew-Bundle-and-Brewfile) — declarative package state, `bundle check`, dump, install, services, and destructive cleanup semantics. **Confidence: HIGH**
- [Homebrew manpage](https://docs.brew.sh/Manpage) — exact command options and dry-run/cleanup behavior. **Confidence: HIGH**
- [chezmoi command overview](https://www.chezmoi.io/user-guide/command-overview/) — `doctor/status/diff/apply` separation and multi-machine workflow. **Confidence: HIGH**
- [chezmoi apply](https://www.chezmoi.io/reference/commands/apply/) — target application, overwrite prompt, and `--dry-run --verbose`. **Confidence: HIGH**
- [chezmoi diff](https://www.chezmoi.io/reference/commands/diff/) — target-state versus destination-state preview. **Confidence: HIGH**
- [chezmoi verify](https://www.chezmoi.io/reference/commands/verify/) — post-apply state verification with meaningful exit status. **Confidence: HIGH**
- [chezmoi machine-to-machine differences](https://www.chezmoi.io/user-guide/manage-machine-to-machine-differences/) — templates, machine-local data, OS/arch differences, and ignore rules. **Confidence: HIGH**
- [chezmoi password-manager integration](https://www.chezmoi.io/user-guide/password-managers/) — public source repository with external secret retrieval; used only as a future comparison, not a v1 dependency. **Confidence: HIGH**

### Toolchain and project contracts

- [mise configuration](https://mise.jdx.dev/configuration.html) — hierarchical project/global/local configuration, precedence, `mise.local.toml`, schema, and current resolution behavior. **Confidence: HIGH**
- [mise dev tools](https://mise.jdx.dev/dev-tools/) — per-directory version switching, `mise exec`, installation versus activation, and auto-install mechanisms. **Confidence: HIGH**
- [uv project layout](https://docs.astral.sh/uv/concepts/projects/layout/) — `pyproject.toml`, universal `uv.lock`, project environment, and version-control guidance. **Confidence: HIGH**
- [uv locking and syncing](https://docs.astral.sh/uv/concepts/projects/sync/) — locked checks, exact sync, and removal of extraneous packages. **Confidence: HIGH**
- [uv Python versions](https://docs.astral.sh/uv/concepts/python-versions/) — `.python-version`, discovery, managed/system preferences, and automatic downloads. **Confidence: HIGH**
- [rustup overrides](https://rust-lang.github.io/rustup/overrides.html) — command/env/directory/`rust-toolchain.toml`/default precedence. **Confidence: HIGH**
- [Go toolchains](https://go.dev/doc/toolchain) and [go.mod reference](https://go.dev/doc/modules/gomod-ref) — `go`/`toolchain` selection and project module contract. **Confidence: HIGH**
- [Node.js 20 Corepack docs](https://nodejs.org/download/release/latest-v20.x/docs/api/corepack.html) and [Node.js 25 Corepack docs](https://nodejs.org/download/release/v25.8.0/docs/api/corepack.html) — evidence that Corepack packaging/ownership cannot be assumed stable across Node major versions. **Confidence: HIGH**
- [Deno configuration](https://docs.deno.com/runtime/reference/deno_json/) and [Deno dependency management](https://docs.deno.com/runtime/packages/) — project config, lockfile, frozen/CI behavior, permissions, and lifecycle-script boundaries. **Confidence: HIGH**
- [Bun lockfile](https://bun.com/docs/pm/lockfile) — `bun.lock`, lockfile-only behavior, migration, and cache side effects. **Confidence: HIGH**
- [Maven Wrapper](https://maven.apache.org/tools/wrapper/index.html) and [Maven Toolchains](https://maven.apache.org/guides/mini/guide-using-toolchains) — build-tool version wrapper, checksums, and JDK selection. **Confidence: HIGH**
- [Gradle Wrapper](https://docs.gradle.org/current/userguide/gradle_wrapper.html) and [Gradle JVM Toolchains](https://docs.gradle.org/current/userguide/toolchains.html) — recommended wrapper workflow, distribution checksum, and JDK toolchain separation. **Confidence: HIGH**
- [Nix `develop`](https://nix.dev/manual/nix/2.26/command-ref/new-cli/nix3-develop) — project build environment for system/compiler dependencies. **Confidence: HIGH**
- [direnv official documentation](https://direnv.net/) and [direnv manual](https://direnv.net/man/direnv.1.html) — per-directory environment loading and explicit authorization security boundary. **Confidence: HIGH**

### Privacy and manual security boundaries

- [GitHub secret scanning concepts](https://docs.github.com/en/code-security/concepts/secret-security/secret-scanning) — hardcoded credential exposure and repository-history scanning risk. **Confidence: HIGH**
- [Apple Platform Security: controlling app access to files](https://support.apple.com/guide/security/controlling-app-access-to-files-secddd1d86a6/web) — user transparency, consent, control, and explicit Accessibility/Automation permission. **Confidence: HIGH**
- [Apple macOS Privacy & Security settings](https://support.apple.com/guide/mac-help/change-privacy-security-settings-on-mac-mchl211c911f/mac) — user-managed Full Disk Access, Accessibility, Automation, Input Monitoring and related permissions. **Confidence: HIGH**

---
*Feature research for: Yet Another Mac Config — toolchain governance first, recovery readiness second*
*Researched: 2026-07-10*
