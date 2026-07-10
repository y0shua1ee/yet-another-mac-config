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

task、wave、phase 的 hard deadline 分别是 15、47、305 秒；完整公式是 `6 * 47 + 15 + 8 = 305`。timeout 必须原样返回 `124`。`docs-and-phase-gate` 只能做固定文档/symlink/结构检查，不得调用 phase；`phase-integration` 只能依次启动 `phase-e2e` 与 `docs-and-phase-gate`，不得重复 phase。完整 phase 必须单独运行，并保持六个固定组件 wave 后接 `phase-e2e` 的顺序。

缺少本地 Go 时返回 `manual-required` / `32`，绝不 bootstrap、安装或联网。测试环境必须使用 fresh 仓库外 root/cache、空白 allowlist 环境、`GOTOOLCHAIN=local`、`GOPROXY=off`，不得继承真实 HOME/XDG/manager state。长时间测试要保留同一执行会话并轮询到真实退出码。

新增 runner route 必须是完整 unsplit literal case label；不允许变量、glob、alternation、命令替换、任意 package/pattern、通用 shell dispatcher 或调用方提供的结构检查路径。每次 owner task 都要对 parent diff 验证只增加计划允许的 label。

## 隐私、输出与 claim

- 只持久化 closed schema 允许的 logical ref、canonical digest、opaque HMAC token 与 bounded status。
- `run_id`、`suite_id`、`operation_ids`、enum、digest、HMAC token 与 timestamp 必须走各自 field-specific validator；未注册自由字符串默认拒绝。canary 还必须把 secret/identity/provider/path-like 值放入看似合法的字段，验证 artifact、CLI renderer 与 Store 在输出/写入前失败且不回显。
- 禁止物理路径、真实 root/home、用户名、UID、host identity、raw output、resolver mapping、HMAC key、API key、token、密码、cookie、私钥、登录态或客户数据进入 artifact、报告、文档、fixture 样本或 Git。
- 当前 service adapter 的 tracked proof 缺失；current-host 必须在 adapter/workload 前 `manual-required`，不能写成 current-host passed。
- `synthetic-sentinel-passed` 只是内层 fixture 结果；standalone/replay report 必须是 `synthetic-report-claim-ineligible`，checked-in expectation 不得保存 passed、claim 或 surface token。唯一允许的 scoped claim 是 `covered-surfaces-unchanged-for-run`，且必须在同一次 `RunRealEnvelope` 中以 one-shot process capability 从 actual Evidence + Evaluation + `RequestClaim` 生成，并绑定 evidence/suite/manifest/window/surface evidence；正向路径只用 proof-valid isolated private doubles。禁止整机、当前 host readiness、多机或 fresh-install claim。
- `extra` 与 `unmanaged-present` 仅 report-only；`operations` 保持为空，不得生成 apply/cleanup authority。

## 文档与提交检查表

任何配置或控制面变化完成前必须同步检查：

1. 更新 `safety/README.md` 的真实 CLI、artifact、tier、verdict、deadline、privacy 与 claim 行为。
2. 更新本文件；确保 `safety/AGENTS.md` 仍是目标恰为 `CLAUDE.md` 的相对 symlink。
3. 检查根 `README.md` 的配置表、测试方法与仓库外 local-state 边界；只有全局约定变化时才修改更高层 guidance。
4. 审计 `.gitignore` 与 `.gitleaks.toml`，不得用 broad exception 隐藏 fixture、artifact 或 secret。
5. 对 scripts 的新增/修改注释使用中文，命名保持简单且 closed。
6. 只 stage 计划声明的精确文件；运行 cached diff check、精确路径 privacy scan 与 staged Gitleaks。
7. 使用英文、单一逻辑的 atomic commit；不得自动 push。
