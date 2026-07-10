# Phase 1: Safety, Privacy, and State Foundation - Context

**Gathered:** 2026-07-10
**Status:** Ready for planning

<domain>
## Phase Boundary

本阶段交付一个可复用、可机械验证的安全基础，使后续所有 observation、ownership、toolchain、recovery 与 link 工作都必须通过同一套 artifact 身份、隐私、隔离测试和 sentinel 证明边界。它覆盖 SAFE-01 至 SAFE-08：区分六类 artifact、阻止隐私数据进入输出、提供 offline synthetic 与显式隔离 integration harness，并对 exact run 的受保护真实表面生成有范围的未变化证据。

本阶段不实现 Phase 2 的工具所有权 inspector，不迁移六类语言生态，不重构多主机 composition，不执行 Homebrew/Nix/Home Manager/toolchain apply，不替换真实链接，也不授予 `recovery-ready-on-current-host` 或 `fresh-install-verified` 声明。任何真实机器修复、安装、下载、cleanup、trust mutation、service mutation、defaults write 或 switch 都超出本阶段自动执行范围。

</domain>

<decisions>
## Implementation Decisions

### Artifact Identity and Lifecycle

- **D-01:** 六类 artifact 使用公共 envelope 加 kind-specific payload schema。Envelope 负责明确的 artifact kind、schema version、run metadata 与 provenance；payload 由对应类型 schema 独立校验。字段相似不允许一种 artifact 冒充另一种完成证据。
- **D-02:** 生命周期按 artifact kind 定义。真实运行生成物默认只进入 Git 忽略的本地状态根；可重建 snapshot 保持短期，exact plan 保持 immutable，receipt 与 verification evidence 按其证据语义保留。只有完全 synthetic、经过隐私校验的 fixture 或 golden artifact 可以进入 Git。
- **D-03:** kind、schema、version 或 provenance 任一校验失败时，整份 artifact 被拒绝并停止当前数据流，返回非零状态。不得生成 partial/degraded success artifact，也不得把原始无效 artifact 保存到 quarantine；只允许有界、结构化、非敏感诊断。
- **D-04:** Artifact 关系使用 schema-defined digest lineage graph。Plan 绑定 exact desired 与 observed digests；receipt 绑定 exact plan digest；verification evidence 绑定预期 postconditions 与 fresh observation；report 绑定它实际汇总的 exact evidence。无 apply 的 read-only 路径使用独立允许的 lineage。`run_id` 只供检索，不能替代完整性证明；禁止按目录或 latest-file 自动拼接。

### Privacy Violation Handling

- **D-05:** 只允许预先注册、可证明无歧义的结构化转换。秘密值、private provider reference、未知绝对路径、完整环境、未经约束的原始字段或无法分类的数据必须在 stdout、stderr 与落盘前使当前流程硬失败。通用 masking 不能把违规内容转成有效 artifact。
- **D-06:** Privacy rejection 只允许输出 schema-validated error envelope：稳定 error code、artifact kind、adapter ID、逻辑字段 pointer、violation category 与安全 remediation。禁止原值、截断值、前后缀、长度、content hash、真实路径 basename、provider 名称或其他内容派生指纹。
- **D-07:** Allowlisted 外部命令的 stdout/stderr 必须通过 pipe 有界捕获，设置时间与大小上限，在内存中由严格 adapter 解析，并在成功规范化后才允许输出。Parse failure、overflow 或未知字段返回 `unknown` 或 privacy error；原始字节立即丢弃。只有 synthetic fixture 可以保留 synthetic raw sample。
- **D-08:** 持久化路径与机器引用只使用显式注册的逻辑 namespace，例如 `repo:`, `home:`, `fixture:`, `local-state:`, `nix-output:` 与公开 logical profile。真实 root 和 identity 只在本地 resolver 当前进程中短暂存在。未知绝对引用直接拒绝；不使用 basename 裁剪或真实值 hash。

### Fixture and Opt-in Integration Experience

- **D-09:** 测试分为三个显式 tier，且绝不自动升级副作用：默认 offline static/schema/pure/synthetic adapter tests；显式选择的 isolated integration tests；独立的 allowlisted `live-check`。缺少能力或失败不会让 runner 自动转入更高权限层。
- **D-10:** Git 只保存 synthetic fixture blueprint。每次运行在新的系统临时根实例化，使用最小 allowlisted environment，隔离 `HOME`、全部 XDG roots、`TMPDIR`、`PATH`、trust、cache、runtime 与 manager-specific directories。真实 worktree 只作为 tracked input source，不作为可写 sandbox。
- **D-11:** 成功和失败 fixture 默认都删除。只有运行前显式选择 `--keep-fixture` 才能保留 synthetic 或 isolated integration root；保留项位于专用本地状态根，具有 logical fixture ID、created-at、TTL 与 validated ownership marker。Cleanup 只能删除该根内具有有效 marker 的内容；live raw output 永不保留。
- **D-12:** Integration 默认仍 offline。需要网络的 test 必须在 tracked manifest 中声明 exact test/adapter、download purpose、integrity check、maximum bytes、timeout 与 isolated cache，并由操作者仅授权该 exact test ID。禁止继承真实 credentials、Keychain、proxy variables 或 generic token。无法限制 egress 或验证完整性时不运行，返回 `manual-required`；不提供全局网络开关，也不允许 cache miss 自动联网。

### Sentinel Proof and Final Verdict

- **D-13:** Sentinel 以显式 protected-surface manifest 为范围。Worktree/index、命名的真实 HOME 配置入口、global manager state roots、受控 services 与仓库外 named targets 分别使用 privacy-safe adapter。每个 test 必须声明理论上可能接触的真实 surface。不得递归读取整个 HOME、秘密文件或任意 app database，也不得声称覆盖 manifest 外范围。
- **D-14:** Verdict 为严格四态：所有 required sentinel 都具有完整、可解析且一致的 before/after evidence 才是 `passed`；发现变化为 `violation`；缺少观察或证据为 `indeterminate`；sentinel schema/内部失败为 `harness-error`。后三者均返回非零状态。只有明确 optional 的 sentinel 可以 warning 而不阻止通过。
- **D-15:** Required protected surface 在观察窗口内出现任何差异即判当前 run 为 `violation`，但报告只陈述 `change-detected-during-window`，不能擅自归因给 test。不得 auto-restore、auto-ignore 或 auto-retry。高噪声 surface 必须在运行前缩小、标为 optional 或明确 excluded，不能事后忽略。
- **D-16:** Phase 1 最强声明仅为 `covered-surfaces-unchanged-for-run`。Evidence 必须绑定 exact suite、test tier、protected-surface manifest digest、observation window、before/after snapshot digests 与 excluded/optional list。不得宣称整个 Mac 永久未变化；`recovery-ready-on-current-host` 保留给 Phase 13，`fresh-install-verified` 仍需未来 clean VM 或第二台 Mac。

### Determinate Nix and Home Manager Control Plane

- **D-17:** Determinate Nix、nix-darwin 与 Home Manager 构成项目的 primary declarative control plane：Determinate Nix 负责 Nix distribution/daemon/support boundary，nix-darwin 负责 machine composition/activation，Home Manager 负责 user configuration、Nix-built manager entrypoints、config files 与 shell integration。
- **D-18:** Nix/Home Manager module 的存在不自动把 mutable downstream payload 转移到 Nix store。Homebrew、mise、uv、rustup、project wrappers 与 exclusive Nix devShell 可以保留明确的 payload/executable ownership。系统必须分别记录 declaration owner、manager binary owner、managed payload owner、selected executable 与 activation context；每个 `(scope, executable)` 只允许一个 primary owner。
- **D-19:** Module 中会触发 download、upgrade、prune、trust 或 deletion 的 activation 仍是独立 mutable write boundary，必须具有 plan、expected downloads、receipt、fresh verification 与真实 rollback/forward-repair 语义。在相关生态 phase 通过隔离验证和 exact confirmation 前不得启用。Nix 可在明确选定的 scope 通过 Nix package 或 exclusive devShell 直接拥有 executable；该 scope 不得再由 delegated manager 同时拥有。

### Agent's Discretion

- 选择实现 harness、schema validator 与 adapter contract 的具体语言和最小依赖，但必须能离线运行、提供确定性输出，并保持 check/test code 与任何 apply executor 的物理依赖边界。
- 选择 digest 算法、envelope 的精确字段名、schema 文件拆分方式、stable error code 命名与本地状态目录名，只要满足上述 identity、lineage、privacy 与 lifecycle 决定。
- 选择 fixture TTL 默认值、并行运行目录布局、大小/时间上限与 synthetic fake-binary 实现方式；默认必须保守且不能接触真实 HOME/global manager state。
- 选择首批 protected-surface adapter 的最小集合及其 privacy-safe fingerprint，只要 Phase 1 success criteria 覆盖 worktree、命名 HOME 入口、global tool state、services 与仓库外 named state，且不以全 HOME 扫描伪造覆盖。
- 选择 CLI 的最终命令名称和人类可读渲染格式；三个 tier、非零 verdict 语义、exact authorization 与机器可读 artifact contract 不得改变。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Scope and Phase Contract

- `.planning/PROJECT.md` — Core value、non-destructive constraints、canonical-source policy 与当前 brownfield boundary。
- `.planning/REQUIREMENTS.md` — SAFE-01 至 SAFE-08 的 normative v1 requirements，以及后续 phase 边界。
- `.planning/ROADMAP.md` — Phase 1 goal、success criteria、dependencies 与明确不属于本阶段的后续能力。
- `.planning/todos/pending/2026-07-10-clarify-determinate-nix-and-home-manager-authority.md` — 已确认的 primary control-plane 分层及仍需在项目文档中统一措辞的后续任务。

### Architecture and Risk Research

- `.planning/research/ARCHITECTURE.md` — Separate state kinds、privacy boundary、typed plan、isolated fixture pyramid、logical references 与 report semantics。
- `.planning/research/PITFALLS.md` — Raw output、ambient HOME、implicit downloads、false verification、cross-plane rollback 与 destructive convergence 风险。
- `.planning/research/STACK.md` — One owner per `(scope, executable)`、Home Manager manager-entrypoint role、delegated runtime ownership 与 Nix devShell exclusivity。

### Existing Codebase and Safety Conventions

- `CLAUDE.md` — Repository-wide documentation、privacy review、atomic commit、no-auto-push 与 official-doc requirements。
- `.planning/codebase/TESTING.md` — 当前没有统一 test runner/fixture system；现有 Nix、native validator 与 manual verification patterns。
- `.planning/codebase/CONCERNS.md` — `setup_mac.sh` fallback、destructive replacement、host identity、mixed activation 与 privacy coverage gaps。
- `.planning/codebase/CONVENTIONS.md` — Shell strict-mode、guard clauses、naming、documentation placement 与 local guidance conventions。
- `.gitleaks.toml` — 现有 secret scan policy；可作为一道 gate，但不能替代 structured privacy validation。
- `setup_mac.sh` — 现有 interactive/symlink path 与已知 physical-directory fallback 风险；Phase 1 tests 不得调用其真实 mutation branches。
- `install_yazi_plugins.sh` — 现有 configurable target 与 guard-clause pattern，可参考其隔离入口思路但不得运行 network-backed install 作为默认 test。

### Nix and Toolchain Integration Boundaries

- `flake.nix` — Determinate Nix + nix-darwin + Home Manager composition、locked inputs 与 build/switch entrypoints。
- `nix/CLAUDE.md` — Evaluation/build/review/switch/post-check separation，以及 activation rollback boundary。
- `nix/darwin/default.nix` — Determinate Nix compatibility boundary (`nix.enable = false`) 与 system-level imports。
- `nix/darwin/homebrew.nix` — Conservative Homebrew inventory/activation policy；Homebrew remains a mutable component boundary。
- `nix/home/default.nix` — Home Manager user composition and imported modules。
- `nix/home/dev-toolchains.nix` — Current manager binaries installed through Home Manager and delegated per-project runtime intent。
- `nix/modules/zsh.nix` — Current mise activation path and shell integration point that later ecosystem phases may migrate carefully。

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `.gitleaks.toml`: 可复用为提交前与 fixture artifact 的一层 leak scan，但需要新增结构化 forbidden-field/path validation 才能满足 SAFE-02/03。
- `flake.nix` 与 `nix/CLAUDE.md`: 已有 evaluation → non-activating build → review → authorized switch 的分层，可直接作为 test tier 与 activation separation 的先例。
- `setup_mac.sh` / `install_yazi_plugins.sh`: 已有 Bash strict mode、quoted expansions、precondition guards、stderr diagnostics 与 configurable target 的局部模式；只复用安全模式，不复用 physical-directory fallback 或真实 replacement branch。
- 应用原生 validators（记录于 `.planning/codebase/TESTING.md`）: 后续可由统一 harness 适配，但 Phase 1 首先建立 adapter contract 与 synthetic fixture，不把 live app reload 纳入默认 test。

### Established Patterns

- Repository changes require synchronized documentation, privacy review, focused English commits, and no automatic push。
- Nix evaluation/build 与 activation 已分离；通过 build 不等同于 machine apply 或 verification。
- Homebrew `autoUpdate = false`, `upgrade = false`, `cleanup = "none"` 体现 conservative convergence，符合 SAFE-08。
- Home Manager 当前通过 `home.packages` 提供 mise/uv/rustup manager entrypoints；runtime payload 分别由项目 contract 与 delegated manager 管理。
- 当前没有 repository-wide runner、committed tests directory、assertion library、fixture factory 或 CI gate，planner 需要从最小 vertical slice 建立而不能假设已有框架。

### Integration Points

- 新 schema、privacy validator、fixture harness 与 sentinel adapters 应放在新建且职责清晰的 repository-owned safety/test boundary 中；具体目录由 planner 决定，不能把 apply executor 引入默认 check/test dependency graph。
- Root-level test entrypoint 需要与 `flake.nix` non-activating checks、`.gitleaks.toml` 和 future adapter validators 组合，但默认不得调用 `darwin-rebuild switch`、Homebrew mutation、manager install 或 live link replacement。
- Synthetic fixtures 从 Git-tracked blueprint 实例化到 external temp root；真实 `.config`, `nix/`, Shell 与 app files 只作为受跟踪输入读取，不作为 writable fixture。
- Protected-surface sentinel 与 artifact store 使用 logical namespaces/local resolver，不把 real username、hostname、HOME 或 absolute path 写入 committed/generatable evidence。

</code_context>

<specifics>
## Specific Ideas

- Artifact envelope 至少需要明确 kind、schema version、run identity 与 provenance，但 exact field naming 由 planner 决定。
- Logical references 以 `repo:`, `home:`, `fixture:`, `local-state:`, `nix-output:` 与 public logical profile 为基础；unknown absolute reference fail closed。
- Privacy diagnostics 应适合自动测试：stable error code + logical pointer，不包含可关联真实内容的 sample/hash。
- Test UX 明确显示当前 tier，默认 offline；network authorization、fixture retention 与 live-check 都是逐项显式 opt-in。
- Sentinel pass label 使用 `covered-surfaces-unchanged-for-run`，避免用自然语言夸大成“整个 Mac 未变化”。
- Determinate Nix/nix-darwin/Home Manager 是主控制平面；module 负责声明/manager entrypoint 不代表 downstream payload 已转入 Nix store。

</specifics>

<deferred>
## Deferred Ideas

- Phase 2：实现 read-only ownership inspector，消费本阶段 schema/diagnostic/sentinel contracts。
- Phases 3–8：逐生态评估 `programs.mise`, `programs.uv`, rustup、project wrappers 与 exclusive Nix devShell；任何 writer/owner 迁移都需 isolated evidence 与 rollback。
- Phase 9 或相关 recovery planning：评估 optional Determinate nix-darwin module 与 `nix-homebrew`，不得在 Phase 1 自动迁移现有 Nix/Homebrew installation。
- Phase 10：把 secret obligations 与完整 desired/observed readiness taxonomy 建立在本阶段 privacy/artifact foundations 上。
- Phase 11：修复 `setup_mac.sh` physical fallback 与真实 link replacement；Phase 1 仅禁止默认 test 进入这些 branch。
- Phase 12：实现 exact-plan confirmation、component apply/receipt/verification；Phase 1 只定义可复用 artifact contract。
- Phase 13：运行 current-host non-destructive readiness drill，并在证据充分时产生 `recovery-ready-on-current-host`。
- Future milestone：使用 clean macOS VM 或第二台 Mac 证明 `fresh-install-verified`。

</deferred>

---

*Phase: 1-Safety, Privacy, and State Foundation*
*Context gathered: 2026-07-10*
