# Determinate Nix + Home Manager 管理说明

本目录是这份 Mac 配置仓库的声明式主控制面。目标是在 Apple Silicon Mac 上安装基础前置后，clone 同一仓库、选择对应主机 profile，并通过一条受控命令同步机器级与用户级配置。

## 管理模型

- **Determinate Nix**：安装并维护 Nix、Nix daemon 与 `/etc/nix` 的受支持配置边界。
- **Determinate nix-darwin module**：通过 `determinateNix.enable = true` 协调 Determinate Nix 与 nix-darwin，防止两者同时管理 Nix。
- **nix-darwin**：组合 macOS 系统设置、Homebrew inventory、服务试点与 Home Manager 激活。
- **Home Manager**：管理用户 packages、shell、环境变量、工具链管理器入口，以及仓库内配置到 home 目录的链接。
- **Homebrew / mise / uv / rustup**：作为明确委托的 payload owner；它们的下游状态不会因为存在 Nix module 就自动变成 Nix store 内容。

仓库保存期望状态，当前 Mac 是激活目标。`flake.lock` 必须跟随 Git 提交；正常 clone / build 使用已锁定版本，不会先更新依赖。

## 目录结构

```text
nix/
├── hosts/default.nix      # 主机 registry：system、username、repoPath
├── darwin/                # 所有 Mac 共用的系统层配置
│   ├── default.nix
│   ├── defaults.nix
│   └── homebrew.nix
├── home/                  # Home Manager 用户层
│   ├── default.nix
│   ├── dotfiles.nix       # 受审计的配置链接 allowlist
│   ├── packages.nix
│   ├── shell-env.nix
│   └── dev-toolchains.nix
└── modules/zsh.nix        # Home Manager zsh 配置
```

根目录的 `sync_mac.sh` 是统一 build / switch 入口；`setup_mac.sh` 只保留为非 Nix 回退工具。

## 当前主机与多机添加

`nix/hosts/default.nix` 中每个属性都会生成一个 `darwinConfigurations.<LocalHostName>`。名称必须与目标机器的以下命令一致：

```bash
scutil --get LocalHostName
```

新增另一台 Apple Silicon Mac 时，先复制一个 profile：

```nix
{
  "another-mac" = {
    system = "aarch64-darwin";
    username = "yourname";
    repoPath = "/Users/yourname/Documents/dev/config/yet-another-mac-config";
  };
}
```

若某台机器需要独有配置，可在 profile 中增加 `modules = [ ./another-mac.nix ];`，把差异留在 host module，共享配置继续留在 `darwin/` 与 `home/`。

当前正式验收范围是 Apple Silicon。nix-darwin 可以描述 Intel Mac，但截至 2026-07-14，Determinate Installer 的公开稳定支持矩阵没有给 Intel macOS 与 Apple Silicon 相同的承诺；不要在未核对最新官方支持并实机验证前把 `x86_64-darwin` 当作已支持恢复路径。

## Home Manager 管理的仓库配置

`home/dotfiles.nix` 使用显式 allowlist，把下列目录以 out-of-store symlink 链接到 `~/.config`：

- AeroSpace、borders、btop
- GitHub CLI 共享偏好、Ghostty、mise、mpv
- Neovim、tmux、Typora、Yazi
- `.hammerspoon` 链接到 `~/.hammerspoon`

使用 out-of-store symlink 是为了继续直接编辑 Git 工作区并让 app 立即看到变化。代价是仓库必须保留在 profile 的 `repoPath`；移动仓库后要先更新 profile，再重新 switch。

allowlist 不做目录自动发现。Alma、账号登录态、聊天/媒体、缓存、凭据和其他本机状态不会因为出现在 `.config` 下就自动进入 Home Manager。

## 全新 Mac 首次同步

### 1. 安装 Determinate Nix

官方优先提供 macOS 安装包，也保留 CLI installer。先按 [Determinate 官方文档](https://docs.determinate.systems/) 完成安装，再确认：

```bash
nix --version
```

输出应明确包含 `Determinate Nix`。flake 中的 Determinate module 负责配置协调，不负责在完全没有 Nix 的机器上安装 Nix。

### 2. 安装 Homebrew

nix-darwin 的 `homebrew` module 只管理 formula / cask / service 清单，不安装 Homebrew 本身。按 [Homebrew 官方文档](https://brew.sh/) 安装后确认：

```bash
brew --version
```

### 3. Clone 到 profile 声明的路径

```bash
mkdir -p ~/Documents/dev/config
git clone <repository-url> ~/Documents/dev/config/yet-another-mac-config
cd ~/Documents/dev/config/yet-another-mac-config
```

如果用户名、路径或 LocalHostName 不同，先修改 `nix/hosts/default.nix` 并提交，不要靠临时命令隐藏主机差异。

### 4. 先构建，再激活

```bash
./sync_mac.sh --build-only
./sync_mac.sh
```

脚本会：

1. 检查 macOS、Determinate Nix、Homebrew、主机 profile、当前用户名与仓库物理路径。
2. 从仓库锁定的 nix-darwin input 构建 `darwin-rebuild`，不临时运行未锁定的 `master`。
3. 执行 `darwin-rebuild build`。
4. 只有在构建通过并得到确认后才执行 `sudo darwin-rebuild switch`。

无人值守地接受 switch 可用 `./sync_mac.sh --yes`；日常默认仍建议保留确认。

## 日常同步与依赖升级

同步已提交配置：

```bash
git pull --ff-only
./sync_mac.sh
```

只检查当前提交：

```bash
nix flake check
./sync_mac.sh --build-only
```

依赖升级是独立维护动作，不属于普通 clone / sync：

```bash
nix flake update nixpkgs home-manager nix-darwin determinate
nix flake check
./sync_mac.sh --build-only
```

这四个 input 共享兼容边界，应在同一次维护中验证；只更新 nix-darwin 而保留过旧 nixpkgs 可能让文档构建工具接口不匹配。评审 `flake.lock`、构建结果和运行时验证后再 switch。`home.stateVersion = "24.11"` 是兼容边界，不随 Home Manager 版本一起升级。

## 当前边界

- Homebrew 激活保持保守：`autoUpdate = false`、`upgrade = false`、`cleanup = "none"`。它会补齐声明项，但不会删除机器上额外安装的软件，也不保证所有 Mac 的 Homebrew payload 版本完全相同。
- `borders` 与 `nginx` 是仅有的 `brew services` 试点，使用 `start_service = true`；不要未经评审扩大到账号态或本地数据较重的服务。
- Node / Go 的全局 fallback 由 `.config/mise/config.toml` 声明，mise 负责实际 runtime payload。项目版本仍优先使用项目内 `.mise.toml`、`pyproject.toml + uv.lock`、`rust-toolchain.toml` 或 devShell。
- secrets、`~/.zshrc.local`、登录态、TCC / Accessibility 权限、云盘数据与聊天/媒体不纳入仓库。
- Hammerspoon app 与配置可以声明和链接，但 Accessibility 权限仍需在系统设置中人工授予。
- tmux 的仓库配置继续配合本地 oh-my-tmux；`.config/tmux/tmux.conf` 是机器相关软链接，不进入 Git。

## 回滚与故障处理

- build 失败不会切换当前 generation。
- Home Manager 遇到已有目标时使用 `hm-backup` 后缀备份；首次接管前应检查是否有同名备份。
- 回滚上一代：`sudo darwin-rebuild switch --rollback`。
- 仓库被移动后，先修正 host profile 的 `repoPath`，否则 out-of-store symlink 会失效。
- `existing file would be overwritten` 表示目标所有权仍有冲突；先检查目标和 `*.hm-backup`，不要直接删除用户数据。

维护约束与验证命令见 [`CLAUDE.md`](./CLAUDE.md)。
