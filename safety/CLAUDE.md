# Safety 子系统维护指南

## 本地维护规则

`safety/` 维护本仓库的离线安全验证控制面，包括：

- `cmd/yamc-safety/`：closed CLI 与有界 JSON 输出。
- `internal/artifact/`：六种 artifact、storage/lifecycle 与 lineage。
- `internal/privacy/`：logical ref、surface domain、resolver 与输出 gate。
- `internal/fixture/`：仓库外 marker-owned fixture、retention 与 network policy。
- `internal/sentinel/`：protected surfaces、real proof gate、before/after 与 verdict。
- `internal/workflow/`：synthetic skeleton、phase report 与整合流程。
- `manifests/`、`schemas/`、`testdata/`：仓库拥有的固定契约与 synthetic 输入。
- `scripts/test.sh`：唯一受支持的测试入口；只允许 literal task/wave/phase dispatch。

本仓库是期望状态的 source of truth；当前 Mac 只是在用户明确确认后才能使用的 activation target。Safety 代码只能观察受控输入、验证契约、创建仓库外 fixture/store 并生成有界证据。不要把 current-host 观察反写成新的声明状态，也不要用 machine-only 临时修改绕过仓库。

## 禁止的操作

禁止执行真实激活、安装、更新或清理命令。尤其不得在本子系统开发/测试中执行：

- `nix build`、`nix flake update`、`darwin-rebuild switch`、`home-manager switch` 或 Nix store/profile 清理；
- `brew install`、`brew upgrade`、`brew services start/stop/restart`；
- `mise install/use`、`uv sync/python install`、`rustup toolchain install/update`；
- `launchctl`、`defaults write/delete`、真实服务控制、真实链接改写、trust store 改写；
- 网络下载、真实 egress、ambient proxy/credential 继承；
- 对仓库、真实 Home、manager root 或当前 host 做 apply、restore、prune、rollback、repair 或 convergence cleanup。

唯一允许的删除是：主 verdict 已冻结后，重新验证 marker、effective UID、nonce、非 symlink、直接子级 containment 与 TTL，再删除**本次创建的仓库外 fixture 子目录**。默认删除；只有运行前显式 keep 才可保留最多 24 小时。不得把这个 teardown 例外扩展成真实环境清理权限。

外部 artifact store 必须用 store 自己的时钟约束 24 小时 snapshot；只允许明确的 2 分钟正向 clock skew，并在 write、reopen、read、delete 全部 fail closed。测试必须注入时钟，不能依赖固定日期或通过未来 `created_at` 延长生命周期。

## 测试契约

修改前后只通过下面的固定入口验证：

```bash
/bin/bash -n safety/scripts/test.sh
./safety/scripts/test.sh task phase-e2e
./safety/scripts/test.sh task docs-and-phase-gate
./safety/scripts/test.sh wave phase-integration
./safety/scripts/test.sh phase
```

task、wave、phase 的 hard deadline 分别是 15、47、305 秒；完整公式是 `6 * 47 + 15 + 8 = 305`。每次 public `test.sh` 调用都必须无条件建立一个 watchdog，并让 setup、固定 docs checks、build/list/test、child dispatch 与 marker-owned cleanup 全部计入预算。受监控 body 只能由 watchdog 从同一文件的固定 marker 后读取，大小有上限，并在独立 process group 内执行；禁止重新引入任何 caller-selectable internal/re-exec 参数，或使用 ambient environment、PID、继承 FD、nonce 来关闭 watchdog。父聚合层只能原样传播自限时 child 的唯一 timeout envelope，不得再叠加命令级 process-group wrapper。timeout 必须只输出一个 `runner-deadline-exceeded` envelope 并原样返回 `124`。`docs-and-phase-gate` 只能做固定文档/symlink/结构检查，不得调用 phase；`phase-integration` 只能依次启动 `phase-e2e` 与 `docs-and-phase-gate`，不得重复 phase。完整 phase 必须单独运行，并保持六个固定组件 wave 后接 `phase-e2e` 的顺序。

缺少本地 Go 时返回 `manual-required` / `32`，绝不 bootstrap、安装或联网。测试环境必须使用 fresh 仓库外 root/cache、空白 allowlist 环境、`GOTOOLCHAIN=local`、`GOPROXY=off`，不得继承真实 HOME/XDG/manager state。长时间测试要保留同一执行会话并轮询到真实退出码。

新增 runner route 必须是完整 unsplit literal case label；不允许变量、glob、alternation、命令替换、任意 package/pattern、通用 shell dispatcher 或调用方提供的结构检查路径。每次 owner task 都要对 parent diff 验证只增加计划允许的 label。

## 隐私、输出与 claim

- 只持久化 closed schema 允许的 logical ref、canonical digest、opaque HMAC token 与 bounded status。
- `run_id` 只能由可信 builder 生成 digest-derived opaque ID；`suite_id` / `operation_ids` 只能来自固定 registry。command-result 必须走 closed field/type registry，未知 key、numeric identity 或任意 caller public-ID 不能依赖词法 denylist 放行。canary 必须覆盖无敏感关键词 identity、opaque credential、stable machine ID、unknown key、numeric UID，并验证 construction、CLI renderer 与 Store 在输出/写入前失败且不回显。
- `fixture run` 只能接受仓库外 base 与 logical fixture ID，并由 `fixture.Create` 创建 fresh direct child；禁止重新公开 `--fixture-root` / `--store-root`，也禁止把已有目录或真实 HOME 当作测试 sandbox。
- ownership marker 必须通过同目录 temp + fsync/close + no-replace publish；初始化 rollback 仅可依靠本次 fresh directory identity/UID/nonce capability 删除 exact child，即使 marker 截断也不得遗留 child 或触碰 sibling/base。
- 所有 repository input 必须先以 no-follow、nonblocking、before/opened/after identity recheck 读取，再通过固定 `/usr/bin/git` 的无网络、无 hook exact plumbing：唯一 stage-0 index entry、index 与 frozen HEAD mode/blob 一致、实际 worktree executable bit 映射的 `100644` / `100755` 与 frozen mode 一致、HEAD blob 与实际消费的 bounded worktree bytes 一致。Git 缺失/失败、非 worktree、untracked、ignored、symlink、index substitution、bytes substitution 或 chmod-only mode drift 必须在创建 fixture/store 前 fail closed。
- 禁止物理路径、真实 root/home、用户名、UID、host identity、raw output、resolver mapping、HMAC key、API key、token、密码、cookie、私钥、登录态或客户数据进入 artifact、报告、文档、fixture 样本或 Git。
- 当前 service adapter 的 tracked proof 缺失；current-host 必须在 adapter/workload 前 `manual-required`，不能写成 current-host passed。
- manager-tree 的内部 symlink 必须解析最终 target 并限制在 exact manager root；relative/absolute/chain escape 一律 incomplete、无 token、无 claim，不能通过只 hash link text 绕过。
- `RunRealEnvelope` 必须在内部为每个 run 生成 fresh 32-byte key，同 run before/after 只临时共享该 key，所有返回路径都清零 buffer；公共 options 禁止接受 caller key。确定性测试只能使用 package-private secret factory，并必须证明跨 run token 不同以及 workload/entropy/claim-consumer 失败后 key 已清零。
- `synthetic-sentinel-passed` 只是内层 fixture 结果；standalone/replay report 必须是 `synthetic-report-claim-ineligible`，checked-in expectation 不得保存 passed、claim 或 surface token。唯一允许的 scoped claim 是 `covered-surfaces-unchanged-for-run`，且必须在同一次 `RunRealEnvelope` 中以 one-shot process capability 从 actual Evidence + Evaluation + `RequestClaim` 生成，并绑定 evidence/suite/manifest/window/surface evidence；正向路径只用 proof-valid isolated private doubles。禁止整机、当前 host readiness、多机或 fresh-install claim。
- `extra` 与 `unmanaged-present` 仅 report-only；`operations` 保持为空，不得生成 apply/cleanup authority。
- read-only 与 apply lineage 都必须证明 `FreshObserved.Scope` 和 `FreshObserved.State` 实际存在于 exact observed artifact 的 typed facts；合法 logical ref、digest 或 scope 一致本身不能替代这项语义绑定。

## 文档与提交检查表

任何配置或控制面变化完成前必须同步检查：

1. 更新 `safety/README.md` 的真实 CLI、artifact、tier、verdict、deadline、privacy 与 claim 行为。
2. 更新本文件；确保 `safety/AGENTS.md` 仍是目标恰为 `CLAUDE.md` 的相对 symlink。
3. 检查根 `README.md` 的配置表、测试方法与仓库外 local-state 边界；只有全局约定变化时才修改更高层 guidance。
4. 审计 `.gitignore` 与 `.gitleaks.toml`，不得用 broad exception 隐藏 fixture、artifact 或 secret。
5. 对 scripts 的新增/修改注释使用中文，命名保持简单且 closed。
6. 只 stage 计划声明的精确文件；运行 cached diff check、精确路径 privacy scan 与 staged Gitleaks。
7. 使用英文、单一逻辑的 atomic commit；不得自动 push。
