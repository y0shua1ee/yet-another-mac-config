# Walking Skeleton — Yet Another Mac Config Safety Control Plane

**Phase:** 1
**Generated:** 2026-07-10

## Capability Proven End-to-End

> 操作者可以在 exact real-before/real-after 只读安全 envelope 内，于仓库外的全新 synthetic fixture root 中完成 desired state → observed state → generated plan → fake applied receipt → fresh verification evidence → bounded readiness report；只有外层真实 required evidence 完整一致时才获得 `covered-surfaces-unchanged-for-run`。

## Architectural Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Interaction surface | Repository-owned local operator CLI：`validate`、`store`、`fixture run`、`sentinel verify`、`report` | 这是本地配置仓库，不需要 Web UI、HTTP API 或 daemon；子命令边界可机械测试且不会扩大 mutation authority。 |
| Framework | Go standard library only，入口为 `safety/cmd/yamc-safety`，测试使用 `testing`；外层使用 strict Bash runner | `encoding/json`、`crypto/sha256`、`context`、`os/exec` 与 `io/fs` 足以离线完成 strict parsing、digest、bounded capture 和 filesystem containment；无第三方依赖或网络 bootstrap。 |
| Typed data plane | Common envelope + six kind-specific payload contracts；canonical JSON 经 SHA-256 寻址；tracked Git 只有 public desired sources/validated synthetic golden，所有 runtime artifacts 在 external-local-state | Closed class-kind lifecycle：snapshots 为 unpinned 24h，plan append-only 直到 explicit applied/abandoned，evidence bundles 递归 pin ancestors 且 Phase 1 不自动 prune。 |
| Logical surface identity | Persistent namespaces 固定为 `repo:`、`home:`、`fixture:`、`local-state:`、`nix-output:`、`profile:`；另用 closed `surface_domain` compatibility table | Worktree/index 使用 `repo:`，named HOME/manager root 使用 `home:`，service/named target 使用 public `profile:sentinel/...`；物理路径、UID、identity 与 resolver mapping 仅在进程内存。 |
| Auth and credentials | Not applicable；credentials、provider values、login state 与 secret injection 一律禁止 | Phase 1 没有远程服务或身份认证需求；任何 credential handling 都会违反 SAFE-03 与本阶段边界。 |
| Local isolated run | `./safety/scripts/test.sh phase` 用 exact logical manifest 读取五类真实 surface before/after，中间在 fresh external root 运行 offline synthetic stack | 只有当前官方只读语义+隔离负证据通过的精确 adapters；这是安全 sentinel，不是 functional discovery/live-check，不部署或 mutation。 |
| Directory layout | `safety/cmd` 为 interaction surface，`safety/internal/{artifact,privacy,fixture,sentinel,contract,workflow}` 为 typed boundaries，`safety/{manifests,testdata,scripts}` 为 tracked contracts | 读取、写入、隐私、fixture、sentinel 与控制平面职责物理分离；未来真实 apply executor 不得进入该 module 的 dependency graph。 |
| Claim ceiling | `covered-surfaces-unchanged-for-run`，并绑定 exact suite、tier、manifest、window 与 before/after evidence | 只证明本次运行中 manifest 覆盖的 surface；不推断整个 Mac、current-host readiness、多主机或 fresh install。 |

## Stack Touched in Phase 1

- [ ] Interaction scaffold — stdlib-only Go module、CLI routing、strict Bash runner 与 deterministic exit semantics
- [ ] Interaction route — 至少一个真实 `fixture run` 子命令以及 `validate`、`store`、`sentinel verify`、`report` operator routes
- [ ] Typed data plane — 六类 artifact 的 canonical content-addressed read AND write
- [ ] Local isolated run — fake adapter 在 fixture namespace 内产生 synthetic receipt/fresh evidence，Wave 1 唯一成功状态为 `synthetic-sentinel-passed`
- [ ] Sentinel boundary — 外层 exact real before/after + 内层 synthetic test doubles、四态 verdict 与 exact scoped claim
- [ ] Documentation — root README、`safety/README.md`、`safety/CLAUDE.md` 与真实 `safety/AGENTS.md -> CLAUDE.md` symlink

## Out of Scope (Deferred to Later Slices)

- Phase 2 的 read-only ownership inspector 与真实 executable provenance。
- Phases 3–8 的 Node、Go、Python、Rust、Deno/Bun 与 JVM 生态契约或任何 runtime hydration/migration。
- Phase 9 的 shared/role/logical-host composition 与 identity-only binder。
- Phase 10 的完整 desired/observed inventory、secret obligations 与 machine-readiness taxonomy。
- Phase 11 的真实 symlink planning、backup、replace 与 restore。
- Phase 12 的 exact-plan confirmation、真实 apply executor、component receipts 与 rollback/forward-repair。
- Phase 13 的 current-host non-destructive readiness drill 与 `recovery-ready-on-current-host`。
- Future milestone 的 clean VM/second-Mac evidence 与 `fresh-install-verified`。
- 任何 `flake.nix`、`nix/**`、`.config/mise/**`、Shell activation、Homebrew inventory、setup script、service/defaults/link 的 functional discovery/mutation 或 live Mac mutation；仅允许 exact-manifest、已证明、有界的只读外层安全 sentinels。

## Subsequent Slice Plan

后续 phase 在不重新协商本 skeleton 的 interaction、typed data、offline isolation 与 claim ceiling 的前提下增加纵向能力：

- Phase 2: Read-only Ownership Inspector
- Phase 3: Node, npm, pnpm, and Corepack Governance
- Phase 4: Go Governance
- Phase 5: Python and uv Governance
- Phase 6: Rust and rustup Governance
- Phase 7: Deno and Bun Governance
- Phase 8: JVM, Maven, and Gradle Governance
- Phase 9: Multi-host Composition and Binder Spike
- Phase 10: Recovery Observation and Readiness
- Phase 11: Fail-closed Symlink Safety
- Phase 12: Integrated Recovery Engine
- Phase 13: Current-host Non-destructive Readiness Drill

## Local Run Contract

- Runner 固定 `GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`、`GOENV=off`、`GOWORK=off`、`CGO_ENABLED=0`，并将 `HOME`、全部 `XDG_*`、`TMPDIR`、`GOCACHE`、`GOMODCACHE` 与 manager roots 指向 fresh external root。
- 本地 Go toolchain 缺失时输出 bounded `manual-required`，不得安装、下载或调用 Nix、Homebrew、mise、uv、rustup 等 manager。
- 默认 phase run 先验证 exact adapter/version 的官方只读语义新鲜度与隔离负证据；任一缺失即 `indeterminate`/`manual-required` non-zero，不使用 synthetic fallback。公开 artifacts 只含 logical IDs/opaque tokens，physical roots、UID、identity 与 HMAC key 只在进程内存。
- 五个 surface domain 必须通过闭合 compatibility table：`worktree` → `repo:sentinel/worktree/{tracked,index}`，`named-home` → `home:.zshrc`，`manager-root` → `home:sentinel/manager/mise-data`，`service` → `profile:sentinel/service/homebrew-mxcl-nginx`，`named-target` → `profile:sentinel/named-target/system-shells`。
- 固定顺序为 `real-before → isolated workload + inner synthetic sentinels → freeze primary verdict → marker-owned fixture teardown unless pre-run keep → real-after → monotonic final combine`。Synthetic 只能输出内部测试状态，不能产生真实表面 claim。
- 禁止 restore/live/destructive convergence cleanup。唯一允许的 teardown 是 primary verdict 冻结后删除 marker/UID/nonce/containment-verified external fixture child；不能删除 retention base/任意/live state，且 teardown/after 失败只能保留或恶化 non-pass。
