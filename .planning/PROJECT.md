# Yet Another Mac Config

## What This Is

`yet-another-mac-config` 是个人 Mac 配置的唯一事实来源，通过 GitHub 同步适合公开、复用和审查的系统、开发环境与应用配置。仓库当前以 Nix、nix-darwin、Home Manager、Homebrew inventory 和交互式软链接脚本共同管理一台 Apple Silicon Mac，并将演进为支持“共享基线 + 主机覆盖层”的多 Mac 安全恢复系统。

恢复与迁移必须可预览、可验证、可回滚：不把秘密或登录状态同步到 Git，也不以破坏已有可用环境为代价追求机器一致性。

## Core Value

在不泄露私密信息、也不破坏任何已有可用环境的前提下，让受支持的 Mac 能从本仓库恢复到可验证、尽可能一致的开发与软件配置状态。

## Requirements

### Validated

- ✓ Git 仓库已经保存并同步多种公开的 macOS 系统、Shell、应用原生配置与自动化配置，同时通过 ignore 规则和本地覆盖约定把凭据、登录状态、缓存和设备状态留在本机 — existing
- ✓ 已有两条配置激活路径：Nix、nix-darwin 与 Home Manager 负责声明式系统和用户层，`setup_mac.sh` 负责交互式部署应用配置、Hammerspoon 和备用 Shell 入口 — existing
- ✓ Nix 输入已有 lockfile；现有流程将 evaluation、非激活式 build、人工审查、授权 switch 与 generation rollback 分开 — existing
- ✓ nix-darwin 已声明一批 Homebrew formulae、casks、字体和有限服务，并刻意关闭自动 update、upgrade 与 cleanup，减少已有机器被意外改变的风险 — existing
- ✓ Home Manager 已提供 mise、uv、rustup、direnv 和 nix-direnv 等开发工具管理入口；Node/npm 与 Go 已有仓库管理的 mise 全局 fallback — existing
- ✓ Home Manager 主 Shell 与非 Home Manager 备用 Shell 共享公共 `zsh/shared.zsh`，并都允许从 Git 之外加载私有本地覆盖 — existing
- ✓ 仓库已管理终端、编辑器、窗口管理、文件管理、tmux、媒体工具和 macOS 自动化等应用原生配置，部分依赖已有 lockfile、固定 revision 或 vendored provenance — existing
- ✓ 已有 Nix evaluation/build、部分应用原生 validator、人工 post-check、隐私 diff review 与 Gitleaks 配置等验证基础 — existing
- ✓ 配置变更已有文档同步、隐私审查、原子英文提交和禁止自动 push 的维护纪律 — existing

### Active

- [ ] 将本仓库明确为所有未来 Mac、编程环境与软件配置变更的 canonical source：先在仓库表达期望状态，再经受控流程应用到电脑，避免只改机器而不回写配置。
- [ ] 将当前单主机结构演进为“共享基线 + 主机覆盖层”；首版使用不暴露真实用户名、hostname、序列号或其他设备指纹的逻辑主机标识，并为未来不同 Mac、角色与架构保留扩展点。
- [ ] 为六类开发生态建立无重叠的所有权矩阵：Node/npm/pnpm/Corepack、Go、Python/uv、Rust/rustup、Deno/Bun、Java/JVM/Maven/Gradle。
- [ ] 采用统一治理策略而非强迫所有工具由 mise 管理：mise 管理适合的运行时版本，uv 和 rustup 保留各自生态职责，项目包管理器遵守项目清单与 lockfile，Nix devShell 可承载系统库或复杂编译依赖，direnv 只负责显式加载项目环境。
- [ ] 为每类生态定义项目级版本契约、全局 fallback 边界、进入项目时的加载方式、缺失与冲突检查、迁移步骤、验证标准和回滚路径；同一运行时在任一层只能有一个明确主所有者。
- [ ] 按生态逐步迁移：盘点现状、声明所有权、加入项目契约、隔离验证、对真实环境执行只读检查、保留回滚，最后才允许移除重复入口或旧安装；每个生态使用独立的原子提交。
- [ ] 可以补充当前仓库缺失的测试、受版本控制的最小 fixtures，或在临时目录实例化示例项目；它们必须脱离真实 HOME、真实项目和真实全局工具状态运行。
- [ ] 任何测试或验证都不得破坏当前 Mac 已有的工具链、项目、Shell、服务、配置或登录状态；无法安全隔离的操作只能生成检查结果和待执行计划，不得为了让测试通过而自动修正本机。
- [ ] 建立固定的恢复工作流：`check → plan → confirm → apply → verify`。`check` 默认只读发现状态与缺失项，`plan` 完整列出变化、风险和回滚方式，只有明确确认后才能进入 `apply`。
- [ ] 恢复范围覆盖适合公开和声明的整机状态：Nix、nix-darwin、Home Manager、Homebrew formulae/casks、字体、受控服务、六类工具链、Shell、Git 跟踪的应用配置、稳定的 macOS defaults、CLI 和符号链接。
- [ ] 建立非秘密 manifest 与 ignored local overlay：Git 只保存秘密项的名称、用途、模板、校验规则和抽象 provider 引用；真实值只保留在忽略的本地文件、macOS Keychain 或密码管理器中。
- [ ] 恢复检查只能报告秘密、权限、登录或人工步骤是否缺失，不读取、打印、复制或提交真实秘密及登录数据。
- [ ] 生成统一的 machine-readiness report，区分已声明且验证、已声明但未应用、机器漂移、权限或登录缺失、私密覆盖缺失、人工步骤和明确排除项。
- [ ] 在当前 Mac 上完成非破坏性的恢复演练和 readiness 验证；在真正干净的 Mac 或 macOS 虚拟机验证前，状态只能称为“恢复就绪”，不能称为“已验证全新安装恢复”。
- [ ] 为每个恢复步骤记录幂等性边界、异常停止点、回滚方式、人工确认点，以及多台 Mac 之间允许存在的预期差异。

### Out of Scope

- 真实 API key、token、密码、私钥、证书私钥、恢复码和其他秘密值 — GitHub 同步边界禁止保存这些内容。
- Apple ID、App Store、浏览器、聊天软件和其他账号的自动登录 — 这些属于敏感登录状态，不适合声明式同步。
- 浏览器历史、聊天记录、应用数据库、缓存、媒体历史和完整用户数据镜像 — 它们可能泄露隐私，也不是本仓库要恢复的配置源。
- 自动绕过 macOS TCC、Accessibility、Full Disk Access 或类似权限 — 系统安全授权必须由用户人工完成，仓库只负责报告缺失。
- 无人值守的 `sudo` 激活、静默覆盖、自动删除或无法预览的清理 — 这些行为与 confirm gate 和不破坏原则冲突。
- 为追求形式统一而把全部语言生态强制交给 mise — 目标是统一所有权政策，不是统一成一个实现工具。
- 字节级复制多台 Mac，或保证所有软件内部状态完全一致 — 目标是让公开、可声明状态尽可能一致，并显式报告合理差异。
- 未经单独授权批量修改仓库之外的真实项目 — 本仓库可以定义契约与 fixtures，实际项目按遇到和需要逐个迁移。
- 在没有证据时宣称已经完成多机或全新安装恢复 — 当前只有一台实体 Mac 可用于非破坏性验证。
- 首版强制依赖 1Password CLI — 它可在以后作为可选 secret provider 引入，但不能成为首版恢复的前置条件。
- 当前里程碑执行 macOS 虚拟机或第二台实体 Mac 的端到端恢复 — 先完成工具链治理与恢复就绪基线，具备条件后再做干净环境验证。
- 首版同步可导出的应用状态、扩展列表或插件账户状态 — 如以后需要，可在隐私和可移植性审计后单独评估。

## Context

- 当前只有一台 Apple Silicon Mac；Nix composition 仍直接表达单一用户、主机和架构，尚未形成共享基线与主机覆盖层。
- 仓库有两个相互关联但不等价的 activation plane：Nix/nix-darwin/Home Manager 负责系统和用户声明，`setup_mac.sh` 负责应用配置与符号链接。Nix 成功 build 或 switch 不代表应用配置已经部署，也不代表整台工作站已可用。
- 当前恢复能力有意只覆盖工作站的一部分。秘密、登录状态、Accessibility 等权限、部分服务和应用状态继续依赖本地或人工步骤，但目前缺少统一的可检查完成标准。
- Homebrew inventory 的保守策略不会清理额外软件或强制升级，因此相同声明不等于机器实时状态完全相同；未来应通过 drift/readiness report 表达差异，而不是静默清理。
- Node/npm 与 Go 已由 mise 提供全局 fallback。uv、rustup、direnv 等管理入口已经存在，但 Python、Rust、Deno/Bun、JVM 和 Node 包管理器等尚无完整、统一且可检查的项目治理契约。
- Home Manager 主 Shell 和备用 Shell 当前并不完全等价；备用路径没有同等的 mise 激活行为。部分现有工具或 hook 还依赖机器相关路径，需要在多机恢复中消除或显式处理。
- `setup_mac.sh` 目前没有完整 dry-run、机器可读 plan、非交互 profile 或可恢复状态。其 tracked-only discovery 依赖 Git、有效 worktree 和非空的 tracked config 查询；条件不满足时的物理目录 fallback 可能枚举 ignored/untracked 本地状态。
- 当前没有仓库级测试 runner、统一验证命令、CI gate 或隔离 fixture 体系；验证主要分散在 Nix 和各应用的维护文档中。
- 现有文档还保留少量语言栈迁移期描述，与当前 Node/Go/mise 终态存在漂移；后续实现应以受跟踪配置、当前项目规划和实际只读检查为准。
- 旧的 Hermes NVM-to-mise 文件只作为历史执行证据，不再是当前计划或下一步指令；本项目的 `.planning` 文档与受跟踪配置是后续规划事实源。
- 当前没有干净 Mac 可用于测试。首轮顺序是先完成项目工具链统一治理，再完成恢复就绪基线；未来有条件时再使用 macOS VM 或第二台实体 Mac 验证真正的全新恢复。

## Constraints

- **Canonical source**：所有未来 Mac、编程环境和软件配置变更必须先从本仓库入手，再应用到电脑；不允许机器状态成为唯一未回写的事实源。
- **Existing environment safety**：任何测试、探测、迁移或恢复操作都不得破坏当前 Mac 已有的工具链、项目、Shell、服务、配置、权限或登录状态；安全优先于自动化深度和表面一致性。
- **Test isolation**：默认验证只能做静态检查、配置解析、依赖关系验证、只读状态探测和经审查的非激活式构建。需要执行工具链时，必须使用临时目录以及隔离的 HOME、XDG 路径和工具专用数据/缓存目录，不得把真实用户目录或真实项目当作 fixture。
- **No implicit mutation**：自动测试不得执行 `darwin-rebuild switch`、Homebrew install/upgrade/uninstall/cleanup、全局 mise 或 rustup 切换、`defaults write`、服务启停、真实符号链接替换或真实 Shell 配置改写。无法隔离的验证只能进入报告和 plan。
- **Build versus activation**：允许明确标注且经审查的 Nix evaluation 或非激活式 build；它们不得改变当前激活 generation。任何 switch 始终属于用户确认后的 apply。
- **Migration safety**：每个生态独立迁移；先证明新路径可用并准备回滚，再删除旧入口或重复安装。不得一次性重置全部开发工具链，也不得为了让测试通过而修正本机。
- **Ownership**：同一运行时或工具在 Homebrew、Home Manager、mise、Nix devShell 和语言原生管理器之间只能有一个主所有者；其他层只能承担明确且不冲突的职责。
- **Recovery gate**：所有真实机器变更遵循 `check → plan → confirm → apply → verify`，并确保 plan 与 apply 在界面、命令和日志中清晰分离。
- **Secrets and privacy**：Git、日志、报告、测试输出、handoff 和规划文档均不得包含秘密值、登录数据、私有网络信息、真实机器身份或不必要的个人标识；共享主机配置使用逻辑标识。
- **Compatibility**：首个已支持并可现场验证的目标是当前 Apple Silicon Mac。多主机结构必须可扩展，但不得假装已验证其他 Mac 或架构。
- **Package ownership**：macOS-native 应用、formulae、casks 和字体遵循仓库现有 Homebrew inventory；Nix/Home Manager 管理声明式系统/用户层和适合的工具入口；项目运行时遵循各生态契约。改变安装或所有权前先阅读当前官方文档。
- **Conservative activation**：除非用户以后明确授权，保留 Homebrew 不自动 update、upgrade 或 cleanup 的非破坏策略，并让 drift report 代替静默清理。
- **Documentation**：配置变更必须同步根 README、相关目录的 `CLAUDE.md` 与相邻 `AGENTS.md` symlink，以及必要的上级文档；脚本注释使用中文。
- **Change discipline**：实施保持小范围、原子、可回滚；提交信息使用英文；提交前进行 diff 与隐私扫描；永不自动 push。

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 使用共享基线与主机覆盖层 | 多台 Mac 共享公共期望状态，同时允许硬件、角色和兼容性差异 | — Pending |
| 主机层使用隐私安全的逻辑标识 | 避免在公共 Git 中继续固化真实用户名、hostname 或设备指纹 | — Pending |
| 恢复采用 `check → plan → confirm → apply → verify` | 保证每次机器变更可见、可控、可验证且有明确停止点 | — Pending |
| 先完成项目工具链治理，再建设恢复就绪基线，最后考虑 VM | 工具所有权与项目契约是可靠恢复的前置条件，当前又没有干净 Mac | — Pending |
| 统一治理但不强制单一工具 | 尊重各生态的官方工具与 lockfile，同时避免重复所有权 | — Pending |
| 首轮覆盖全部六类开发生态 | 建立完整的跨项目版本治理边界，而不是只解决 Node/Go | — Pending |
| 使用非秘密 manifest 与 ignored local overlay | 能检查恢复缺口，同时不把秘密或登录状态提交到 GitHub | — Pending |
| 1Password CLI 延后为可选 provider | 首版不增加额外账号、登录或工具依赖 | — Pending |
| 按生态渐进迁移并保留回滚 | 降低对现有项目和全局环境的中断风险 | — Pending |
| 恢复范围覆盖全部公开且适合声明的机器配置 | 目标是可工作的工作站状态，而不只是语言运行时 | — Pending |
| 测试可以补充，但不得修改或破坏真实环境 | 当前只有一台工作 Mac，环境安全高于测试自动化程度 | — Pending |
| 无干净环境证据前只标记“恢复就绪” | 避免把当前 Mac 的本机演练误称为全新安装验证 | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `$gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `$gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-07-10 after initialization*
