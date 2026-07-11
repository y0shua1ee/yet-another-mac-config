# Safety 控制面

`safety/` 是本仓库的离线安全验证控制面。它验证“仓库中声明的目标状态、受控 fixture、隐私边界与有限 readiness claim”之间的契约；它不是安装器、激活器、修复器或整机备份工具。仓库仍是期望状态的 source of truth，真实 Mac 只是以后经用户确认的 activation target。

当前 Phase 1 只允许读取仓库输入，并把所有生成状态写入仓库外的新建临时根或显式外部 store。它不会安装或更新 Nix、Homebrew、mise、uv、rustup，不会执行 `darwin-rebuild switch`、`home-manager switch`、`brew services`、`launchctl`、`defaults write`、真实链接/信任变更，也不会尝试“收敛”当前电脑。

这里的“仓库输入”不是“路径碰巧位于仓库内”。每次 workflow 在读取第一个输入前只捕获一次 HEAD commit OID 与一份大小受限、完整的 stage-0 index snapshot；所有 blueprint、surface、raw sample、suite、expected report 和 manifest 都使用这同一 frozen view，结束前还会再次核对 HEAD、完整 index bytes 与 repository root identity，任何漂移都会使整个 workflow fail closed。实际 worktree bytes 从 retained repository-root handle 开始逐组件读取：每层目录都先 `Lstat` 拒绝 symlink，再 `OpenRoot`，并用 opened/named `SameFile` 复核后保留 handle；最终文件用 no-follow、nonblocking 打开并做 before/opened/after identity recheck。每个输入还必须通过固定 `/usr/bin/git` 的离线 plumbing 证明：repository root 是 exact worktree top-level；frozen index 中存在唯一 stage-0 entry；index mode/blob 与 frozen HEAD tree 完全一致；只有 owner execute bit `0100` 会把 worktree mode 映射为 `100755`，例如只有 group/other execute 的 `0655` 仍映射为 `100644`；`cat-file` 读取的 HEAD blob 与本次实际消费的 bounded worktree bytes 完全一致。Git 调用使用空白配置、禁用 hooks/fsmonitor/replace object、optional lock、lazy fetch、prompt 与 protocol，因此不会联网或执行仓库 hook。Git 不可用、不是 worktree、查询失败、未跟踪、被 ignore、任何路径组件为 symlink、目录或文件替换、staged/index 替换、跨输入 HEAD/index 混用、bytes 替换或 chmod-only mode 漂移都会在 fixture/store 创建前 fail closed。

## 操作者入口

运行测试需要本机已有可用的 Go。runner 固定使用 `GOTOOLCHAIN=local`、`GOPROXY=off` 和空白 allowlist 环境；若 Go 不存在，会以 `manual-required`、退出码 `32` 停止，不会联网下载或自动 bootstrap 工具链。

```bash
# 单个完整 Phase E2E task；15 秒 hard deadline
./safety/scripts/test.sh task phase-e2e

# 仅检查本文档、local guidance、根 README 与 AGENTS symlink；15 秒 hard deadline
./safety/scripts/test.sh task docs-and-phase-gate

# 依次运行上面两个 task；47 秒 hard deadline，不包含完整 phase
./safety/scripts/test.sh wave phase-integration

# 独立的完整 phase gate；305 秒 hard deadline
./safety/scripts/test.sh phase
```

固定预算是 **15 / 47 / 305** 秒：每个 task 15 秒，每个 wave 47 秒，完整 phase 只按顺序运行六个组件 wave 和最后的 `phase-e2e`，预算公式为 `6 * 47 + 15 + 8 = 305`。每次 public runner 调用都会无条件建立一个 watchdog；watchdog 从同一脚本读取固定、大小受限的 embedded body，并为它建立本次 public invocation 唯一的 PGID supervisor。wave/phase 聚合只使用同一受监控 body 内的 fixed internal task/wave dispatcher：每个 child 仍创建 fresh 临时根和 cache，但绝不递归 public `test.sh`、不再建立 watchdog/process group。不存在调用者可选择的 internal/re-exec mode，也不接受 PID、环境变量、继承 FD 或 nonce 作为跳过 watchdog 的认证材料。预算覆盖参数处理、临时根 setup、固定文档检查、build/list/test、内部 child dispatch 与 marker-owned cleanup；最外层 deadline 直接终止同一 PGID 中的 nested body/helper，不依赖子 watcher 转发。deadline 一律只输出一个 `runner-deadline-exceeded` envelope 并保持退出码 `124`，不会被改写为 expected RED 或普通契约失败。`docs-and-phase-gate` 只做固定结构检查；`phase-integration` 只聚合两个 Phase 7 task；两者都不会递归运行 `phase`。

实现还提供五个稳定的控制面命令：

1. `validate`：解析并验证固定 schema、logical ref、控制面或 policy 契约。
2. `store`：把已经通过隐私 gate 和 lineage 校验的 artifact graph 写入显式外部 store。
3. `fixture run`：只接受显式仓库外 `--fixture-base` 与 logical `--fixture-id`，由生命周期状态机原子创建 fresh direct child，并在其中运行 marker-owned fixture。旧式 `--fixture-root` / `--store-root` 公开入口已删除；调用方不能把已有目录、HOME 形状目录或任意 store 当作 fixture。
4. `sentinel verify`：验证 synthetic snapshot，或先对 real adapter registry 执行 proof gate。
5. `report`：反向装载并校验 suite、只含结构期望的 expected report 与七个 artifact instance，再输出 `synthetic-report-claim-ineligible` 有界报告；standalone/replay 命令不会恢复 outer claim。

`test-policy` 与 `sentinel evaluate` 是上述验证流程的固定辅助路由，不增加网络或 live 权限。所有参数均由 CLI 的 closed parser 校验；没有任意 shell、任意 package/pattern 或通用命令 dispatcher。

## Artifact 契约与 lineage

六种 artifact kind 是：

| Kind | 生命周期 |
|------|----------|
| `desired-state` | 24 小时 snapshot |
| `observed-state` | 24 小时 snapshot；apply lineage 中会分别保存 before 与 fresh-after，因此有两个 instance |
| `generated-plan` | append-only；只允许从 `nonterminal` 终结为 `applied` 或 `abandoned` |
| `applied-receipt` | append-only evidence bundle |
| `verification-evidence` | append-only evidence bundle，并固定引用上游 digest |
| `readiness-report` | append-only evidence bundle，并固定引用 evidence digest |

外部 store 以自己的可信时钟校验 snapshot 生命周期：`created_at` 最多只允许比 store 时钟快 2 分钟，`expires_at` 也不得超过 store 当前时间加 24 小时与同一 2 分钟偏差。该检查覆盖 write、只读 reopen 与 read，调用方不能用未来时间延长 retention。到期只会使未被 pin 的 snapshot 不再可读；Phase 1 的 `Delete` 始终拒绝物理删除。

mutable store 只能独占创建一个此前不存在的 fresh root；同一路径的第二个 writer 与 caller 预建目录都会拒绝。创建时会发布 private capability marker，并在 store lifetime 内保留 parent → root → `sha256` / `transitions` 的完整 rooted handle 与 inode 链。existing store 只能通过只读 reopen 校验和读取，不能重新取得 write/transition authority。

object 与 transition 采用 append-only 发布：在 exact fresh child capability 内创建不可预测 staging file，完成 bounded write 与 `fsync` 后以 no-replace hard link 发布 final name。成功或失败都不按 pathname 删除 staging/final 文件，graph 失败也只撤销内存可见性，绝不 rollback/unlink 已发布名字；`Delete` 不物理删除 object 或 transition。若 same-UID 并发方移动目录，retained handle 仍只代表最初选择的 fresh capability，named chain 漂移会使结果 non-zero，但不会删除替换对象或把现有任意目录变成 cleanup target。所有物理回收统一交给 verdict 冻结后的整棵 marker-owned fixture teardown。artifact/transition read 仍要求 no-follow、nonblocking regular file，并在 bounded `limit+1` 读取前后核对 named/opened identity、mode、size 与 mtime；symlink、FIFO、device、socket、oversize 与替换都会有界失败。

apply 路径因此是六种 kind、七个 instance：`desired -> observed-before -> plan -> receipt -> observed-fresh -> evidence -> report`。更精确地说，plan 绑定 desired、before observation 与 expected postconditions；receipt 绑定 plan；fresh observation 绑定 receipt；evidence 绑定 plan、receipt、desired、fresh observation、expected postconditions 以及 sentinel before/after；report 绑定 evidence。read-only 与 apply evidence 的 compact fresh descriptor 都必须把 scope 和 state 绑定到 exact observed artifact 中真实存在的 facts，不能只依赖合法的 logical-ref 语法。所有边使用 canonical content digest 反向验证，evidence/receipt/report 的上游会被递归 pin。Phase 1 没有破坏性 prune；snapshot 到期也不授权删除任意现有用户状态。

持久化前只接受六个 logical-ref namespace，物理路径永远不进入持久化或公共输出：

| Namespace | 用途 |
|-----------|------|
| `repo:` | 仓库内受控输入 |
| `home:` | 具名 home surface，不等于任意 `$HOME` 路径 |
| `fixture:` | 本次外部 fixture 内的逻辑对象 |
| `local-state:` | 外部 artifact store 中的内容寻址对象 |
| `nix-output:` | 公开、规范化的 Nix 输出标识 |
| `profile:` | 公开 profile/surface 标识 |

字段和值都采用 closed contract：artifact `run_id` 只能由可信 builder 把结构输入变成 digest-derived opaque ID，`suite_id` 与 `operation_ids` 只能来自固定 registry（Phase 1 operation 还必须使用 `fixture.` 前缀）；blueprint 不接受 caller-supplied `run_id`。command-result 也使用封闭 field/type registry，未知 key、未知值类型与 numeric identity 一律在 renderer 前拒绝。state、status、reason、tier、mode 与 verdict 只能取已注册值或其明确的 logical-ref 变体；digest、HMAC token 与 timestamp 分别使用独立 validator。词法上看似中性的 identity、opaque credential、stable machine ID、secret、provider item 或路径都不能借合法字段名进入输出或 artifact。walking-skeleton blueprint 因此只保存固定 registry 中的 `suite_id` / `operation_id` 与只供 policy 使用的 `fixture:` logical `operation_target`。

namespace 与 `surface_domain` 是两张不同的闭合表。当前五个 domain 与六个允许映射为：

| `surface_domain` | 允许的 public logical ref |
|------------------|----------------------------|
| `worktree` | `repo:sentinel/worktree/tracked`、`repo:sentinel/worktree/index` |
| `named-home` | `home:.zshrc` |
| `manager-root` | `home:sentinel/manager/mise-data` |
| `service` | `profile:sentinel/service/homebrew-mxcl-nginx` |
| `named-target` | `profile:sentinel/named-target/system-shells` |

未知 namespace/domain、跨域搭配、绝对 suffix、`..` traversal、resolver escape、raw output、真实 home/root、UID、host identity、HMAC key 与未规范化路径都 fail closed。manager-tree 会解析树内每个 symlink 的最终 target；relative、absolute 或 chain target 只要离开 exact manager root，就固定为 `symlink-escape` / incomplete，既不产生 token 也不能产生 claim。公共 surface token 只使用 opaque HMAC 结果。

## Fixture、tier 与网络边界

公开 `fixture run` 不接受物理 child/store 路径。它只在经过 containment 检查的外部 base 下，以不可预测 nonce 原子建立本次 direct child；HOME/XDG、manager roots、fake PATH 与 sentinel scratch 由该 child 派生，artifact store 路径则保留到首次 writer 独占创建。existing/non-empty/HOME-shaped child、fixture/store overlap 和预先存在的 witness 不能通过参数成为写入目标；初始化失败的 rollback 与运行后 finalize 都只处理本次 marker-owned child。

fixture base 和 artifact store 必须位于仓库外。每次运行只在显式 base 的直接子级创建一个新目录；ownership marker 先写入同目录 temp、完成 `fsync`/close，再以 no-replace hard-link 原子发布，内容包含 logical ID、创建/过期时间、effective UID 与随机 nonce。若初始化期间发生无 marker、partial marker 或目录创建失败，rollback capability 只绑定本次 fresh child 的 directory inode、UID、nonce 与 direct-child containment；它会保留 base/sibling。运行后删除前仍会重新验证完整 marker、UID、nonce、非 symlink、直接子级 containment 和最长 24 小时 TTL。

默认行为是在主 verdict 冻结后删除本次 marker-owned 外部 fixture 子目录。只有运行前明确选择 keep 才保留，且最多 24 小时；到期删除仍需相同 ownership 校验。这个“verdict 冻结后的 marker-owned fixture teardown”是唯一清理例外。禁止对真实 Mac、仓库、Home、manager root、Nix store、Homebrew、服务、defaults、链接或 trust state 做破坏性/收敛式 cleanup。

三层测试能力不会自动升级：

| Tier | Phase 1 行为 |
|------|--------------|
| `offline-static` | 默认层；网络恒为 denied，只处理仓库输入与 synthetic fixture |
| `isolated-integration` | 仍然 offline；可以校验仓库拥有的 exact network test ID、host、digest、大小和 timeout 契约，但执行保持 `manual-required` |
| `live-check` | 只接受将来经过 official semantics、时效证明与 isolated negative evidence 共同批准的固定 probe；当前未批准 probe 返回 bounded unknown，不执行 live 命令 |

精确网络契约是“默认拒绝、不得读取 ambient proxy/credential、不得跟随未声明重定向、不得发送认证信息、不得因 tier 名称自动放权”。Phase 1 的 allowlisted network manifest 仅可被验证，不能触发真实 egress。认证、API key、token、密码、cookie、私钥、登录态和客户数据禁止进入 manifest、fixture、artifact、报告或 Git。

## Sentinel、verdict 与 claim ceiling

Wave 1 的内层 synthetic skeleton 只允许输出 `synthetic-sentinel-passed`。它证明 fixture 内的闭环和写入边界，不代表真实 Mac 未变化，也不能单独生成 readiness claim。

外层 sentinel 的强制顺序是：

1. `real-before`
2. `isolated-workload`
3. `freeze-primary`
4. `fixture-finalize`
5. `real-after`
6. `monotonic-combine`

四种 verdict 与固定退出码为 `passed` / `0`、`violation` / `20`、`indeterminate` / `21`、`harness-error` / `22`。缺少本地 Go 或当前 host 所需 proof 时使用 `manual-required` / `32`；runner timeout 使用 `124`。teardown 失败只能保持或降低结果，不能把失败提升为 passed。

`RunRealEnvelope` 不接受 caller-owned HMAC key。它在 proof gate 通过后为每次 run 内部生成 32-byte 随机 key，同一次 run 的 before/after snapshot 共用该 key 以保持可比；函数返回前无论成功、workload 失败、entropy 失败或 claim consumer 拒绝都会清零内部 buffer。不同 run 不共享 key，因此同一 surface 的 opaque token 不能被当作稳定跨 run 标识。测试需要确定性时只能通过 package-private secret factory 注入，公共调用方不能提供或复用 key。

只有完整、fresh、proof-bound 的外层 real evidence 在所有 required surface 上完成 before/after，且主工作负载、fixture teardown 与 sentinel 均通过时，才可输出唯一的 scoped claim：`covered-surfaces-unchanged-for-run`。claim 必须在同一次 `RunRealEnvelope` 内由 one-shot process capability 调用 `RequestClaim` 生成；claimed report 同时绑定 actual evidence digest、suite digest、manifest digest、window ID/digest 与逐 surface evidence。capability 在消费后清除，序列化 evidence、standalone `report` 与 checked-in expectation 都不能重建 claim。正向 claim 仅在 proof-valid isolated private doubles 中测试。real adapter registry 还必须绑定 exact manifest/source digest、通过 isolated negative evidence，并处于声明的 freshness window 内；缺一项就不能启动 adapter。`extra` 与 `unmanaged-present` 只作为 report-only 状态呈现，`operations` 必须为空，不能据此生成 apply/cleanup 动作。

当前 `launchctl print` service adapter 的 tracked isolated negative proof 缺失。因此 current-host 路径必须在任何 adapter 或 workload 调用前返回 `manual-required` / `32`、`indeterminate`，且 claim ineligible。完整 outer sequence 与 scoped claim 目前只能由离线、proof-valid 的 isolated doubles 验证；这**不是** current-host pass。

明确拒绝 `whole-Mac-unchanged`、`recovery-ready-on-current-host`、`multi-host-verified`、`fresh-install-verified` 以及任何等价表述。本阶段既没有证明整台 Mac，也没有证明重装恢复、多机一致性或当前 host readiness。

## 输出与本地状态

持久化输出只包含 schema 允许的 public ID、logical ref、canonical digest、opaque token、timestamp、bounded enum 和 scoped claim。不得输出或提交真实物理根、用户名、UID、主机身份、原始命令输出、resolver mapping、密钥或 credential；canary 必须同时覆盖敏感值藏在允许字段内的情况，而不只检查敏感字段名。

runner 的临时 home、XDG、Go cache、manager roots、fixture 和 artifact store 全部在仓库外的新建根中，默认在本次结束时按 ownership marker 删除。若为了诊断预先选择 keep，保留内容仍是本机临时状态，不属于 source of truth，不得加入 Git，也不需要为它扩大 `.gitignore` 或 `.gitleaks.toml` 例外。
