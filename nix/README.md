# 渐进式 Nix 迁移说明

这份文档记录 `yet-another-mac-config` 里 Nix 路线的定位、当前覆盖范围、激活方式，以及后续迁移边界。

如果你只是想在新 Mac 上把仓库同步起来，优先看根目录的 [`README.md`](../README.md)。这里只讲 Nix 这条可选路径。

另见：

- 面向维护约束的 agent 文档：[`CLAUDE.md`](./CLAUDE.md)
- Phase 3 规划草案：[`phase-3-plan.md`](./phase-3-plan.md)

## 目标与边界

本仓库采用的是**渐进式** Nix 迁移，而不是“一次性全部声明式接管”。

底层运行时选用 [Determinate Nix](https://docs.determinate.systems/)，系统层使用 [nix-darwin](https://github.com/nix-darwin/nix-darwin)，用户层使用 [Home Manager](https://nix-community.github.io/home-manager/)。

当前目标应理解为：

- 帮助一台新 Mac 更快恢复到大约 70% 到 85% 的可用状态
- 保持迁移过程小步、可回退、不破坏现状
- 不追求 100% 自动化或 100% 声明式接管

当前明确**不**追求全面接管的内容包括：

- secrets / 登录态
- 大范围 app state
- 很多主观、易漂移、或机器差异较大的系统偏好
- 不值得长期维护的零碎本地状态

## 当前已纳入的内容

### Phase 1：骨架

- `flake.nix` 作为入口，定义 `darwinConfigurations.AresdeMacBook-Air`
- `nix/darwin/default.nix` 作为最小系统层配置
- `nix/home/default.nix` 作为 Home Manager 用户层入口
- `nix/modules/zsh.nix` 作为 zsh 模块

其中一个关键约束是：`nix.enable = false;`，让 Determinate Nix 自己管理守护进程，避免与 nix-darwin 冲突。

### Phase 2：Home Manager 扩展与 zsh 接管

已纳入：

- 少量稳定 CLI 工具（如 `ripgrep`、`fd`、`jq`、`tree`、`bat`）
- 通用非私密 shell 环境变量（如 `EDITOR=nvim`）
- Home Manager 版 zsh
- `zsh/shared.zsh` 作为仓库共享 shell 逻辑事实源

当前状态：Home Manager 已实际接管 `~/.zshrc`；机器相关或私密片段继续放在 `~/.zshrc.local`。

### Phase 3A：保守 Homebrew inventory

已引入 `nix/darwin/homebrew.nix`，并采用保守激活参数：

- `onActivation.autoUpdate = false`
- `onActivation.upgrade = false`
- `onActivation.cleanup = "none"`

这意味着：

- `darwin-rebuild switch` 不会自动 `brew update` / `brew upgrade`
- 不会清理未声明的本机 brew 包
- 仍允许平时继续直接 `brew install` / `brew install --cask`

当前已纳入的 inventory 以稳定、低风险条目为主，包括：

- taps：`nikitabobko/tap`
- brews：`ast-grep`、`btop`、`fastfetch`、`fzf`、`gh`、`gitleaks`、`git`、`lazygit`、`neovim`、`starship`、`tmux`、`wget`、`yazi`、`yt-dlp`、`zsh-completions`
- casks：`aerospace`、`ghostty`、`typora`、`visual-studio-code`、`hammerspoon`、`font-maple-mono-nf`

### Phase 3B：tmux 运行时声明化

这里只接管 `tmux` **运行时**，不重写配置体系。

保持不变的边界：

- `.config/tmux/tmux.conf.local` 仍是配置事实源
- `~/.config/tmux/tmux.conf` 仍是指向本地 oh-my-tmux 的软链接
- 不迁到 Home Manager `programs.tmux.extraConfig`
- 不重构 tmux 插件体系

### Phase 3C：少量稳定 `system.defaults.*`

当前只纳入极少数长期稳定、且已与当前机器现状一致的默认项：

- Finder 扩展名显示
- Finder 路径栏
- Finder 状态栏
- Finder 列表视图
- Dock `mru-spaces = false`
- 键盘重复速率两项

刻意未纳入：输入法、触控板、通知、窗口动画、Dock 大量偏好项、自动替换 / 拼写修正等更主观或更易漂移的设置。

### Phase 4 最小版

目前只做三件事：

1. 把 `borders` / `nginx` 作为 `brew services` 试点纳入声明
2. 补上 Ghostty 明确依赖的字体 `font-maple-mono-nf`
3. 把 `hammerspoon` cask 纳入清单

这里对 `brew services` 的策略是 `start_service = true`，也就是：**只在服务未运行时启动，不会重启或停止已运行服务**。

当前仍未纳入：

- `clouddrive2`（账号态 / 本地数据）
- `ollama`（本地模型数据）
- `unbound`、`colima`（非默认开机自启）
- 更大范围字体或 GUI 自动化 app

## 安全激活步骤

> 全新机器上还没有 `darwin-rebuild` 命令，因此首次激活需要用 `nix run` 引导；之后再切回 `darwin-rebuild`。

### Step 0：安装 Determinate Nix（仅首次）

```bash
curl -fsSL https://install.determinate.systems/nix | sh -s -- install
nix --version
```

### Step 1：生成 / 锁定依赖

```bash
nix flake lock
nix flake check
```

### Step 2：首次 build（无需 sudo）

```bash
nix run github:nix-darwin/nix-darwin/master#darwin-rebuild -- \
  build --flake .#AresdeMacBook-Air
```

### Step 3：首次 switch（需要 sudo）

```bash
sudo nix run github:nix-darwin/nix-darwin/master#darwin-rebuild -- \
  switch --flake .#AresdeMacBook-Air
```

### Step 4：之后的迭代

```bash
darwin-rebuild build --flake .#AresdeMacBook-Air
sudo darwin-rebuild switch --flake .#AresdeMacBook-Air
```

## 激活时的注意事项

- 不要直接用 `--flake .`，显式指定 `.#AresdeMacBook-Air`
- 切到新机器前，先确认 `flake.nix` 里的 `hostname` / `username` / `system` 是否需要调整
- 若遇到 `existing file would be overwritten` 一类冲突，Home Manager 会用 `hm-backup` 备份目标文件
- 本仓库不依赖 Nix 去做卸载或大扫除；回退优先用 `sudo darwin-rebuild switch --rollback`

## Hammerspoon 补充说明

虽然 `hammerspoon` cask 已纳入 Homebrew 清单，但 macOS 的 Accessibility 权限无法由 Nix 自动授予。

在新机器上完整启用它，一般需要：

1. `darwin-rebuild switch` 或 `brew install --cask hammerspoon` 安装 app
2. 运行仓库根目录的 `./setup_mac.sh`，把 `.hammerspoon` 同步到 `~/.hammerspoon`
3. 在「系统设置 → 隐私与安全性 → 辅助功能」里给 Hammerspoon 授权
4. 确认 Ghostty 已安装，因为当前 `Ctrl+Alt+T` 快捷键依赖它

如果权限列表里已勾选但仍不生效，常见处理是：先删除 Hammerspoon 条目，再重新添加并重启 Hammerspoon。

## 日常使用建议

当前推荐的节奏不是“所有东西必须先声明式化”，而是：

1. 平时照常用 `brew install` / `brew install --cask` / `brew services ...`
2. 某个软件、服务或配置被验证为长期保留、值得迁移后
3. 再把它吸收到 `nix/` 或仓库的声明式管理里

由于当前 Homebrew 模块处于保守模式，这种“先手动安装，后按价值纳管”的工作流是被允许且适合这份仓库的。

## 后续方向

- 继续保持谨慎、可回退、逐项评估
- 不做扫荡式迁移
- 是否扩大 `brew services`、字体、更多 GUI 自动化 app 或更多系统默认项，后续再单独判断

如果需要实现细节、编辑约束或 agent 侧注意事项，再看 [`nix/CLAUDE.md`](./CLAUDE.md)。
