# Phase 3 计划（具体版）

## 背景

Phase 1 到 Phase 2D 已经完成了 Nix 骨架落地、低风险 Home Manager 扩展，以及 `~/.zshrc` 的实际 takeover。下一步不再是“证明这套骨架能跑”，而是把**更多可复现、但风险仍可控**的内容继续纳入声明式管理。

这一阶段继续坚持同一条原则：**优先接管低风险、可验证、易回滚的部分，不做大爆炸迁移。**

## Phase 3 的目标

1. 提高这台机器的可重建性，尤其是包清单与少量稳定系统偏好。
2. 继续以仓库为事实源，但不强行重写已经稳定工作的本地流。
3. 保持 `setup_mac.sh` 仍然可用，Nix 只是额外的、更可复现的激活通道。
4. 每一步都能独立 build、switch、验证、回滚。

## 明确不在本阶段处理的内容

以下内容继续延期，不放进 Phase 3：

- secrets、登录态、设备私有凭据
- GUI 自动化与 `.hammerspoon` 全量迁移
- `brew services` 的声明式接管
- 字体
- 大范围 `system.defaults.*` 扫荡式迁移
- 把现有 tmux 配置整体改写成 Home Manager `programs.tmux` 风格

## Phase 3 的拆分

Phase 3 拆成三个连续子阶段，范围从低风险到中低风险递进。

### Phase 3A：Homebrew 清单声明化（首要 — 第一版已落地）

**目标**

把当前“依赖 Homebrew，但主要靠人工记忆或散落文档维护”的状态，推进到“由 nix-darwin 统一声明 Homebrew inventory，但暂不做破坏性清理”。

**当前进度（第一版已完成）**

- 已新增 `nix/darwin/homebrew.nix`，并在 `nix/darwin/default.nix` 中通过 `imports` 接入。
- 激活策略保守：`enable = true`、`user = username`、`onActivation.autoUpdate = false`、`onActivation.upgrade = false`、`onActivation.cleanup = "none"`。
- 已声明的首批 inventory（仍属第一版，不是全面接管）：
  - **taps**：`nikitabobko/tap`（供 `aerospace` cask）
  - **brews**：`ast-grep`、`btop`、`fastfetch`、`fzf`、`gh`、`git`、`lazygit`、`neovim`、`starship`、`wget`、`yazi`、`yt-dlp`、`zsh-completions`
  - **casks**：`aerospace`、`ghostty`、`typora`、`visual-studio-code`
- 刻意未纳入第一版：
  - 与 Home Manager 已重合的 CLI（`ripgrep` / `fd` / `jq` / `tree` / `bat`）
  - `tmux`（归属由 Phase 3B 决定）
  - 服务类 formula（`borders` / `nginx` / `unbound` / `colima` / `clouddrive2` / `ollama` 等，与 `brew services` 绑定）
  - 版本管理器 / 运行时（`nvm` / `pnpm` / `uv` / `deno` / `python@*` / `go` / `rust` / `llvm` 等）
  - 媒体 / 构建传递依赖（`ffmpeg` / `imagemagick` 等）
  - 字体 cask、`hammerspoon` cask、含账号态 / 登录态的 cask（`1password-cli`、`raycast`、各 IM / 云盘等）
- 验证：`nix flake check` 通过、`darwin-rebuild build --flake .#AresdeMacBook-Air` 通过；`sudo darwin-rebuild switch` 仍需人工执行。

**刻意不做**

- 不接管 `brew services`
- 不做全量盘点后的一次性大迁移
- 不追求把所有本机 brew 安装项一次录完
- 不试图让 Nix 立刻替代所有 Homebrew 用途

**完成标准**

- [x] `nix flake check` 通过
- [x] `darwin-rebuild build --flake .#AresdeMacBook-Air` 通过
- [x] `sudo darwin-rebuild switch --flake .#AresdeMacBook-Air` 已在当前机器人工执行并验证
- [x] 未声明的本机包不会因为这一步被自动清掉（`onActivation.cleanup = "none"`）

### Phase 3B：tmux 进入“声明式运行时”，但不重写配置体系（已完成）

**目标**

先把 tmux 从“配置在 repo、运行时依赖靠手工准备”推进到“运行时安装路径更可复现”，但不破坏当前 oh-my-tmux + `tmux.conf.local` 的工作流。

**当前进度（已完成）**

- 决策：tmux 二进制**继续留在 Homebrew**，不转入 Home Manager `home.packages`。
  - 理由一：本机 tmux 已由 Homebrew 安装在 `/opt/homebrew/bin/tmux`（3.6a），长期稳定，迁移到 nixpkgs 会把 PATH 里的 provider 换掉并产生过渡成本。
  - 理由二：与已声明在 `nix/darwin/homebrew.nix` 中的 `neovim` / `starship` / `git` 口径保持一致，避免两个 provider 的 tmux 二进制在 PATH 里互相抢占。
  - 理由三：Phase 3B 的目标只是“让运行时更可复现”，只需声明一行，而不是换底层。
- 落地：`nix/darwin/homebrew.nix` 的 `brews` 列表中新增 `"tmux"`，并在同文件注释里记录 Phase 3B 的决策与边界。
- tmux 事实源边界明确保持不变：
  - `~/.config/tmux/tmux.conf.local` 继续是用户自定义入口，跟踪在 git 中。
  - `~/.config/tmux/tmux.conf` 继续是指向 `~/.local/share/tmux/oh-my-tmux/.tmux.conf` 的本地软链接（机器相关、gitignore）。
  - 插件（`tmux-resurrect` / `tmux-continuum`）仍按 oh-my-tmux 的原有方式管理。
- 验证：`nix flake check` 通过；`darwin-rebuild build --flake .#AresdeMacBook-Air` 通过（生成的 `Brewfile` 包含 `brew "tmux"`）；`sudo darwin-rebuild switch` 需在单独一次操作中由人工执行。

**刻意不做**

- 不把 `.config/tmux/tmux.conf.local` 改写为 Home Manager `programs.tmux.extraConfig`
- 不替换 oh-my-tmux
- 不引入 tmux 插件系统重构

**完成标准**

- [x] 新机器按 README + Nix 路线能更稳定地得到 tmux 运行时（由 `nix/darwin/homebrew.nix` 声明）
- [x] 现有 tmux 行为与快捷键不因 Phase 3B 改变（provider 未变）
- [x] `.config/tmux/CLAUDE.md` 与根 README 文档口径一致

### Phase 3C：少量稳定 `system.defaults.*` 试点（已完成）

**目标**

只选少量“长期固定、容易验证、回滚成本低”的 macOS 默认项做试点，把系统偏好接管从 0 推进到 1。

**当前进度（已完成）**

- 已新增 `nix/darwin/defaults.nix`，并在 `nix/darwin/default.nix` 中通过 `imports` 接入。
- 已纳入（所有写入值与当前机器 `defaults read` 结果一致，首次 switch 预期无可感知行为变化）：
  - `system.defaults.finder.AppleShowAllExtensions = true`
  - `system.defaults.finder.ShowPathbar = true`
  - `system.defaults.finder.ShowStatusBar = true`
  - `system.defaults.finder.FXPreferredViewStyle = "Nlsv"`（list view）
  - `system.defaults.dock.mru-spaces = false`
  - `system.defaults.NSGlobalDomain.KeyRepeat = 2`
  - `system.defaults.NSGlobalDomain.InitialKeyRepeat = 30`
- 刻意未纳入（保留给后续阶段单独评估）：
  - `NSGlobalDomain.ApplePressAndHoldEnabled`：当前 unset，不做主动置位。
  - 自动替换 / 自动引号 / 自动破折号 / 拼写纠正（`NSAutomaticSpellingCorrectionEnabled` 等）：当前 unset。
  - Dock：`autohide`、`persistent-apps`、`tilesize`、`orientation` 等偏好漂移项。
  - Finder：`_FXShowPosixPathInTitle`、`ShowHardDrivesOnDesktop` 等非本机常用项。
  - 触控板 / trackpad、窗口动画、通知中心、loginwindow、软件更新策略、输入法等整类偏好。
- 验证：`nix flake check` 通过、`darwin-rebuild build --flake .#AresdeMacBook-Air` 通过；`sudo darwin-rebuild switch` 仍需人工执行（预期是空变化）。

**刻意不做**

- 不把 Finder、Dock、输入法、触控板、通知、窗口动画等大量偏好一口气塞进来
- 不做“看到能配就都配”的收集式迁移
- 不主动声明当前处于 unset 状态的 key，避免把未定义行为固化为强意见

**完成标准**

- [x] 只引入少量、经过人工验证的默认项
- [x] `nix flake check` 通过
- [x] `darwin-rebuild build --flake .#AresdeMacBook-Air` 通过
- [x] 写入值与当前机器现状一致，`switch` 后无可感知行为变化
- [x] 已纳入 / 刻意未纳入边界已在 README 与 `nix/CLAUDE.md` 中写清楚

## 建议执行顺序

1. **Phase 3A: Homebrew 清单声明化**
2. **Phase 3B: tmux 运行时声明化**
3. **Phase 3C: 少量 stable defaults 试点**

如果 3A 做完后发现 Homebrew inventory 的边界仍需大量梳理，可以把 3B 提前，3C 顺延。也就是说，顺序允许微调，但**3A 应始终是默认起点**。

## 建议文件落点

若按本计划推进，预计会新增或更新这些文件：

- `nix/darwin/homebrew.nix`
- `nix/darwin/defaults.nix`
- `nix/darwin/default.nix`
- `nix/home/default.nix`（仅当 tmux 最终放入 Home Manager）
- `README.md`
- `nix/CLAUDE.md`
- `.config/tmux/CLAUDE.md`（如 tmux 阶段需要补充说明）

## 每个子阶段都应遵守的验证流程

```bash
nix flake check
darwin-rebuild build --flake .#AresdeMacBook-Air
sudo darwin-rebuild switch --flake .#AresdeMacBook-Air
```

必要时再补人工验证，例如：

- `brew list` / 目标 app 是否可启动
- `tmux -V`、tmux session 是否正常
- 相关 macOS 默认项是否真的生效

## 回滚原则

- Nix 系统层回退优先用：`sudo darwin-rebuild switch --rollback`
- 不在 Phase 3 引入“自动清理未声明 Homebrew 项”这类难回退操作
- 保留现有 repo 结构与本地软链接流作为兜底，不做不可逆迁移

## Phase 3 完成后的状态定义

当以下条件同时满足时，可认为 Phase 3 完成：

- Homebrew 已进入保守的声明式清单管理
- tmux 运行时的安装/准备路径比现在更可复现
- 至少有一小组稳定的 `system.defaults.*` 已纳入管理
- `brew services`、字体、GUI 自动化、secrets 仍明确留在后续阶段

届时再开启下一轮 Phase 4，讨论更大范围的 Homebrew、services、GUI 或系统偏好接管。
