# Nix 配置指南（面向 agents）

## 当前架构

- 本仓库是 Mac 期望状态的事实源，不再把 Nix 描述为可选附加路线。
- Determinate Nix 负责 Nix 分发、daemon 与受支持的 `/etc/nix` 配置边界。
- `inputs.determinate.darwinModules.default` + `determinateNix.enable = true` 负责与 nix-darwin 协调；不要重新启用 nix-darwin 的 Nix 管理，也不要同时写 `nix.enable = false` 复制同一职责。
- nix-darwin 负责机器级组合、Homebrew inventory、少量稳定 defaults 和 Home Manager 激活。
- Home Manager 负责用户 packages、zsh、环境变量、工具管理器入口与仓库配置链接。
- Homebrew、mise、uv、rustup 可以继续拥有下游 payload；避免在同一 scope 为同一 executable 声明多个 owner。

## 目录与主机模型

```text
nix/
├── hosts/default.nix      # 每台 Mac 的 system / username / repoPath / 可选 modules
├── darwin/                # 所有主机共用的系统层
├── home/                  # Home Manager 用户层，含 dotfiles.nix
└── modules/               # 可复用 Home Manager 模块
```

- `darwinConfigurations` 必须来自 `hosts/default.nix`；不要恢复含义模糊的 `default` 主机别名。
- profile 名称应与 `scutil --get LocalHostName` 一致，激活时始终显式选择 host。
- `nixpkgs.hostPlatform` 来自 profile 的 `system`，共享 darwin module 不得再次硬编码架构。
- 当前正式范围是 `aarch64-darwin`。新增 Intel profile 前必须重新核对 Determinate 官方支持并实机验证。
- 真正的机器差异放到 host profile 的 `modules`；共享行为继续放在 `darwin/` 或 `home/`。

## Dotfiles 所有权

- `home/dotfiles.nix` 使用 `mkOutOfStoreSymlink` 指向 profile 的 `repoPath`，让仓库编辑立即生效。
- allowlist 必须显式维护，禁止扫描 `.config` 自动生成条目；这会把本地聊天、媒体、凭据或缓存误纳入同步。
- 新增目录前先确认它是 Git 跟踪的静态配置，并同步根 README 的配置表与本地状态说明。
- Alma、登录态、凭据、聊天/媒体、缓存与 app runtime state 不得加入 allowlist。
- 移动仓库必须先修改 `repoPath` 并重新 build / switch。
- `setup_mac.sh` 已退役；不要恢复与 Home Manager 并行拥有同一 target 的手工链接流程。
- tmux 是叶子文件接管的例外：`home/tmux.nix` 管理锁定的上游 `tmux.conf` 和仓库 `tmux.conf.local`，但必须让 `~/.config/tmux` 与 `plugins/` 保持本机可写目录。

## 现有边界

- Homebrew 保持 `autoUpdate = false`、`upgrade = false`、`cleanup = "none"`；未经单独评审不要改成删除型收敛。
- `brew services` 仅接管 `borders` / `nginx`，并保持 `start_service = true`，不要改成每次 switch 都重启。
- 只纳入仓库配置明确依赖的字体；当前为 Ghostty 使用的 `font-maple-mono-nf`。
- `system.defaults` 只保留当前已验证的 Finder、Dock 与键盘小集合。
- Home Manager zsh 继续使用 `programs.zsh.initContent`，并显式用 `programs.zsh.dotDir = config.home.homeDirectory` 锁定 `~/.zshrc`；`~/.zshrc.local` 必须保留为私密/机器相关入口。
- `mise activate zsh` 保持在 `~/.zshrc.local` 之后。Node / Go fallback 由仓库 `.config/mise/config.toml` 声明，runtime payload 由 mise 管理。
- 不要恢复针对 mise `2026.6.11` 的 `doCheck = false` overlay；当前 nixpkgs 已升级到 mise `2026.7.5`，应优先使用上游可缓存 derivation。
- `home.stateVersion = "24.11"` 是兼容边界，不随 input 更新。
- Hammerspoon Accessibility、其他 TCC 权限、账号登录态与 secrets 仍需人工处理。
- `/etc/nix/nix.custom.conf` 的 `knownSha256Hashes` 只能列入已审核的 Determinate Installer 空模板；真实自定义设置必须经 `determinateNix.customSettings` 声明，禁止用扩大哈希白名单绕过评审。

## 修改流程

1. 修改前读官方 Determinate Nix、nix-darwin、Home Manager 或相关 app 文档。
2. 做小而聚焦的变更；系统层、Homebrew、dotfile 接管最好分别评审。
3. 同步根 `README.md`、本文件和需要的上层说明。
4. 不触碰根目录现有的用户 `CLAUDE.md` dirty change，也不暂存 `.ai/` 或本机私密状态。
5. 精确 stage 文件；禁止 `git add .` / `git add -A`。
6. 提交前检查 diff、运行凭据/隐私扫描并创建英文原子 commit；不自动 push。

## 验证命令

```bash
# 语法与 flake 求值
bash -n sync_mac.sh
nix flake check

# 构建当前主机，不激活
./sync_mac.sh --build-only

# 或显式构建某个 output
nix build --no-link \
  .#darwinConfigurations.AresdeMacBook-Air.config.system.build.toplevel

# 只有在 build、diff、备份与回滚点都确认后才激活
./sync_mac.sh
```

更新 input 必须是独立、显式的维护动作：

```bash
nix flake update nixpkgs home-manager nix-darwin determinate
nix flake check
./sync_mac.sh --build-only
```

oh-my-tmux 独立升级并单独评审：

```bash
nix flake update oh-my-tmux
nix flake check
./sync_mac.sh --build-only
```

正常新机和日常同步使用已提交的 `flake.lock`，不要先运行 update。首次引导使用 flake 暴露的锁定 `darwin-rebuild` package/app，不要临时拉取 `nix-darwin/master`。
