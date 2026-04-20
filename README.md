# yet-another-mac-config

My Mac config

## 包含的配置

| 目录/文件 | 说明 |
|-----------|------|
| `.config/aerospace` | AeroSpace 窗口管理器 |
| `.config/borders` | JankyBorders 窗口边框 |
| `.config/btop` | btop 系统监控 |
| `.config/ghostty` | Ghostty 终端 |
| `.config/nvim` | Neovim（基于 LazyVim） |
| `.config/tmux` | tmux（基于 oh-my-tmux） |
| `.config/typora` | Typora 自定义主题 |
| `.config/yazi` | Yazi 文件管理器及插件 |
| `.hammerspoon` | Hammerspoon 自动化 |
| `.vscode` | VS Code 项目级设置 |
| `zsh/.zshrc` | Zsh 通用配置（含 `EDITOR=nvim`、bun 等环境变量） |
| `flake.nix` + `nix/` | 渐进式 Nix 迁移配置（Phase 2D：Home Manager 已实际接管 `~/.zshrc`；Phase 3A 第一版：`nix/darwin/homebrew.nix` 已启用保守的 Homebrew 声明式清单；Phase 3B：tmux 运行时加入该清单，配置体系保持 oh-my-tmux + `tmux.conf.local` 不变；Phase 3C：`nix/darwin/defaults.nix` 已启用少量稳定的 `system.defaults.*` 试点，写入值与当前机器一致；Phase 4 最小版：`brew services` 仅接管 `borders` / `nginx`，并补上 `font-maple-mono-nf`、`hammerspoon` 两个 cask；详见 `nix/phase-3-plan.md` 与下文「渐进式 Nix 迁移」章节） |

## 使用说明

1. 赋予脚本执行权限：`chmod +x setup_mac.sh`
2. 执行脚本：`./setup_mac.sh`
3. 根据提示输入目标 macOS 用户名，脚本会逐个遍历仓库中已跟踪的 `.config` 一级配置目录，并在 `/Users/<username>/.config` 中创建软链接；若某个目标项已存在，会先确认是否覆盖，默认则跳过。
4. 如果当前工作区里本地存在 `.codex/config.toml`，脚本会额外询问是否同步到 `~/.codex/config.toml`；该文件默认只保留在本地，不会提交到仓库。
5. 脚本会询问是否将 `zsh/.zshrc` 软链接到 `~/.zshrc`。通用配置（主题、插件、补全等）存放在此文件中；API 密钥、项目变量等隐私内容应写入 `~/.zshrc.local`（不纳入版本控制），会在 `.zshrc` 末尾自动加载。
6. 脚本会检测 `.config/tmux` 是否缺少 `tmux.conf`，如果缺少则提示安装 [oh-my-tmux](https://github.com/gpakosz/.tmux)，自动克隆到 `~/.local/share/tmux/oh-my-tmux` 并创建软链接。
7. 脚本会检测仓库根目录下的 `.hammerspoon`，提示是否同步到 `~/.hammerspoon`。在此之前请先安装 Hammerspoon：走 Nix 路线时由 `nix/darwin/homebrew.nix` 声明自动安装（Phase 4 最小版已纳入），不走 Nix 时用 `brew install --cask hammerspoon`。同步后仍需手动在「系统设置 → 隐私与安全性 → 辅助功能」授予 Hammerspoon 权限，否则 `init.lua` 里的事件 tap 与快捷键不会生效；`Ctrl+Alt+T` 快捷键还依赖 Ghostty cask，已一并在 Homebrew 清单中声明。完整激活流程见下文「Hammerspoon 激活说明」。

## Yazi 插件同步

`install_yazi_plugins.sh` 用来在新环境里批量安装/更新 `package.toml` 中锁定的所有 Yazi 插件，并按需设置部分环境变量（比如 `LG_CONFIG_FILE`，确保 `lazygit.yazi` 能工作）。使用方式：

1. 确认 `ya` CLI 已安装：`brew install yazi`。
2. 可选：指定配置目录，例如 `./install_yazi_plugins.sh --config-dir "$HOME/.config/yazi"`；若不传参数脚本会优先使用 `XDG_CONFIG_HOME/yazi`，否则回退到仓库内 `.config/yazi`。
3. 等待脚本自动执行 `ya pkg install`，输出当前生效的插件列表，并提示缺失的依赖工具（如 `git`、`starship`、`lazygit`、`7zz`、`magick` 等）。

脚本可安全重复执行，方便在多台机器间保持插件一致。

## 容器运行环境

使用 [Colima](https://github.com/abiosoft/colima) 作为 Docker Desktop 的轻量替代方案，搭配 Homebrew 安装的 `docker` CLI 和 `docker-compose` 插件：

```bash
brew install colima docker docker-compose
```

首次启动：

```bash
colima start            # 启动 Colima VM（默认 2 CPU / 2 GB 内存 / 20 GB 磁盘）
docker run hello-world  # 验证 Docker 是否正常
```

`docker-compose` 作为 Docker CLI 插件加载，需在 `~/.docker/config.json` 中添加以下键（如果文件已存在，请合并到现有 JSON 中，不要覆盖整个文件）：

```jsonc
// 合并到 ~/.docker/config.json，保留已有的 auths 等字段
"cliPluginsExtraDirs": [
  "/opt/homebrew/lib/docker/cli-plugins"
]
```

> **注意：** `~/.docker/config.json` 可能包含 registry 登录凭据等敏感信息，不纳入版本控制。

常用 Colima 命令：

```bash
colima start            # 启动（也可通过 brew services start colima 开机自启）
colima stop             # 停止
colima status           # 查看状态
colima delete           # 删除 VM（释放磁盘空间）
```

## 后台服务管理

以下服务通过 `brew services` 管理：

| 服务 | 说明 | 开机自启 | Nix 声明化 |
|------|------|----------|------------|
| borders | JankyBorders 窗口边框 | 是 | ✅ Phase 4 最小版（`start_service = true`） |
| nginx | HTTP 服务器（默认端口 8080） | 是 | ✅ Phase 4 最小版（`start_service = true`） |
| clouddrive2 | CloudDrive2 云盘挂载 | 是 | ❌ 未纳入（账号态 / 本地数据） |
| ollama | 本地 LLM 运行时 | 是 | ❌ 未纳入（本地模型数据） |
| unbound | DNS resolver | 否 | ❌ 未纳入（非默认开机自启） |
| colima | Colima 容器运行时（可选） | 否 | ❌ 未纳入（非默认开机自启） |

「Nix 声明化」列说明这台机器如果走 Nix 路线激活时，哪些服务会被 `nix/darwin/homebrew.nix` 自动处理；未纳入的项继续按本节下方的 `brew services` 命令人工管理。

Phase 4 最小版的声明策略是 `start_service = true`：nix-darwin 在 `brew bundle` 阶段**只在服务未运行时**调用 `brew services start`，不会重启或停止已运行服务，因此在当前机器上是幂等 no-op；只有新机器首次 switch 时才会实际把两个服务启动并登记为 login item。

常用命令：

```bash
brew services list              # 查看当前运行状态
brew services start <name>      # 启动服务（开机自启）
brew services stop <name>       # 停止服务（取消开机自启）
brew services restart <name>    # 重启服务
```

> **注意：** nginx 的配置路径为 `/opt/homebrew/etc/nginx/`。
> **注意：** 即使 `borders` / `nginx` 已由 Nix 声明化，日常重启、停服、查状态仍然使用 `brew services …` 命令；`darwin-rebuild switch` 不会自动重启这两个服务。

## 本地文件同步约定

以下内容仅保留在本地环境，不会同步到仓库：

- `.codex/`：Codex 本地配置。
- `.claude/`：Claude Code 的项目级状态（worktrees、settings.local.json 等）。
- `.config/op/`：1Password CLI 的本地设备信息。
- `.config/linearmouse/`：鼠标与触控板的本机硬件配置。
- `.config/mole/`：清理工具运行日志与本地运行状态。
- `.config/raycast/`：Raycast 本地扩展与缓存数据。
- `.config/jgit/`：Jujutsu / Git 相关本地配置。
- `.config/tmux/tmux.conf`：oh-my-tmux 主配置的软链接，指向本地克隆路径，机器相关。
- `.config/ghostty/*.bak`：Ghostty 配置备份文件。
- `.DS_Store`：macOS 自动生成的目录元数据文件。

`setup_mac.sh` 只会处理 Git 已跟踪的 `.config` 目录，因此这些本地忽略目录不会出现在同步提示里。

如果后续新增只适用于当前机器的配置或缓存文件，建议继续补充到 `.gitignore`，避免误提交到仓库。

## 渐进式 Nix 迁移

本仓库正在以「小步前进、不破坏现状」的方式引入 Nix。底层运行时选用 [Determinate Nix](https://docs.determinate.systems/)，系统层使用 [nix-darwin](https://github.com/nix-darwin/nix-darwin)，用户层使用 [Home Manager](https://nix-community.github.io/home-manager/)。详细结构与约束见 [`nix/CLAUDE.md`](nix/CLAUDE.md)。

### Phase 1 现已管理（骨架）

- `flake.nix`：入口，定义 `darwinConfigurations.AresdeMacBook-Air`，并开启 `home-manager.backupFileExtension = "hm-backup"`。
- `nix/darwin/default.nix`：**最小**系统层配置；关键点：`nix.enable = false;` 让 Determinate Nix 自行管理守护进程，避免与 nix-darwin 冲突。
- `nix/home/default.nix`：Home Manager 用户层入口。
- `nix/modules/zsh.nix`：zsh Home Manager 模块，使用 `initContent` 复用 `zsh/shared.zsh`，并在末尾 `source ~/.zshrc.local`。

### 仍按原方式管理（未下沉到 Nix）

- `brew services`、系统默认值（`system.defaults.*`）、字体、`.hammerspoon`、GUI 自动化等一律保持现状。
- Phase 3A 已开启保守的 Homebrew 声明式清单（见下节），但并非全面接管 —— 未声明的本机 brew 包不会被动卸载，仍可按原流程使用。
- `setup_mac.sh` 仍是首选的初始化方式；Nix 只是额外可选通道。

### 安全激活步骤

> **鸡生蛋问题**：全新机器上没有 `darwin-rebuild` 命令，它必须等 nix-darwin 第一次激活后才会被装进 PATH。因此首次激活用 `nix run` 触发 nix-darwin 自带的安装/切换入口，之后再切回 `darwin-rebuild`。

**Step 0 — 安装 Determinate Nix**（仅首次）

```bash
# 官方安装器（Determinate Nix 发行版；不要混用旧版 nix-installer）
curl -fsSL https://install.determinate.systems/nix | sh -s -- install
# 安装完毕后在新 shell 里验证
nix --version
```

**Step 1 — 生成 / 锁定依赖**（仓库根目录内）

```bash
# 首次会创建 flake.lock；生成后请 git add 并提交，保持可重现
nix flake lock
# 可选：静态检查
nix flake check
```

**Step 2 — 首次 build（不改系统，只下载依赖并评估，无需 sudo）**

```bash
# 不用 darwin-rebuild（还没装），用 nix run 拉起 nix-darwin
# build 不写入系统路径，因此保持普通用户身份即可
nix run github:nix-darwin/nix-darwin/master#darwin-rebuild -- \
  build --flake .#AresdeMacBook-Air
```

**Step 3 — 首次激活（真正写入系统状态，必须 sudo）**

```bash
# switch 会写入 /run/current-system、/etc/static/* 等，需要 root
# 首次激活 darwin-rebuild 还没装，用 sudo + nix run 引导
sudo nix run github:nix-darwin/nix-darwin/master#darwin-rebuild -- \
  switch --flake .#AresdeMacBook-Air
```

**Step 4 — 之后的迭代**

```bash
# 从第二次起 darwin-rebuild 已在 PATH；build 无需 sudo，switch 仍要 sudo
darwin-rebuild build --flake .#AresdeMacBook-Air
sudo darwin-rebuild switch --flake .#AresdeMacBook-Air
```

> **谨慎原则**：
> - 不要用 `--flake .` 直接运行（会猜测 hostname），显式指定 `.#AresdeMacBook-Air`。
> - 切换到新机器时请先改 `flake.nix` 中的 `hostname` / `username` / `system`，再执行上述步骤。
> - 激活过程若遇到「existing file would be overwritten」类冲突，`home-manager.backupFileExtension = "hm-backup"` 会把目标文件备份为 `*.hm-backup`。当前这台机器完成 zsh takeover 时，`~/.zshenv` 已自动备份为 `~/.zshenv.hm-backup`；`~/.zshrc` 因本地软链接冲突，最终采用了手动先挪开的方式，旧软链接现保留在 `~/.zshrc.pre-hm-switch-backup`。
> - 本骨架不执行任何卸载/清理类操作；如果激活失败或想回退，可选：(a) `sudo darwin-rebuild switch --rollback` 回滚系统代际；(b) 如要回到旧软链接版 zsh，再按需把 `~/.zshrc.pre-hm-switch-backup` 还原为 `~/.zshrc`。

### Phase 2A 现已管理（低风险 Home Manager 扩展）

- `nix/home/packages.nix`：把若干稳定纯 CLI 工具（`ripgrep`、`fd`、`jq`、`tree`、`bat`）交给 Home Manager 的 `home.packages`。激活后会装到 `/etc/profiles/per-user/<user>/bin`，与 Homebrew 版本共存、互不覆盖。
- `nix/home/shell-env.nix`：声明通用非私密变量 `EDITOR=nvim` / `VISUAL=nvim` / `PAGER=less`。随着 Home Manager 版 zsh 已接管 `~/.zshrc`，这些变量现在已经在登录 shell 中生效。

### Phase 2B 预备重构（已完成）

- `zsh/shared.zsh`：承载 `zsh/.zshrc` 与 `nix/modules/zsh.nix` 共同需要的公开、跨机器通用逻辑，消除双份漂移。
- 历史上的软链接版 `zsh/.zshrc` 负责 Oh My Zsh、Homebrew completion 与本机 OpenClaw completion 的调用方职责；切到 Home Manager 版 zsh 后，`nix/modules/zsh.nix` 直接复用同一份 `zsh/shared.zsh`。
- `~/.zshrc.local` 仍是私有覆盖入口，始终不纳入版本控制，也不被 Nix 接管。

### Phase 2C / 2D 现状：Home Manager 已接管 zsh

- `nix/home/default.nix` 已 `import ../modules/zsh.nix`，配置图里 zsh 模块**真实存在**。
- 已通过 `nix flake check`、`darwin-rebuild build --flake .#AresdeMacBook-Air`，并最终完成了 `sudo darwin-rebuild switch --flake .#AresdeMacBook-Air`。
- 当前 `~/.zshrc` 已由 Home Manager 生成，旧仓库软链接已手动挪到 `~/.zshrc.pre-hm-switch-backup` 作为回退通道。
- `nix/home/shell-env.nix` 里的 `EDITOR=nvim` / `VISUAL=nvim` / `PAGER=less` 已随 Home Manager 版 zsh 生效。
- 机器相关片段（例如 OpenClaw completion）已迁到 `~/.zshrc.local`，Home Manager 版 zsh 的 `initContent` 会在末尾自动 `source` 它。

### Phase 3A 第一版：Homebrew 声明式清单（保守模式，不等于全面接管）

- `nix/darwin/homebrew.nix` 已引入，并在 `nix/darwin/default.nix` 中 import。
- 激活参数刻意保守：`onActivation.autoUpdate = false`、`onActivation.upgrade = false`、`onActivation.cleanup = "none"`。也就是说，`darwin-rebuild switch` **不会**自动 `brew update`、`brew upgrade`，也**不会**清理未声明的本机 brew 包。
- 已声明的 inventory（CLI 优先、GUI 只挑仓库已管理其配置的；`tmux` 在 Phase 3B 纳入）：
  - **taps**：`nikitabobko/tap`
  - **brews**：`ast-grep`、`btop`、`fastfetch`、`fzf`、`gh`、`git`、`lazygit`、`neovim`、`starship`、`tmux`、`wget`、`yazi`、`yt-dlp`、`zsh-completions`
  - **casks**：`aerospace`、`ghostty`、`typora`、`visual-studio-code`
- 目前仍不纳入：服务类 formula（`brew services` 相关，如 `borders` / `nginx` / `unbound` / `colima` / `clouddrive2` / `ollama`）、版本管理器与多语言运行时（`nvm` / `pnpm` / `uv` / `deno` / `python@*` / `go` / `rust` / `llvm` 等）、字体 cask、`hammerspoon` cask、以及含账号态 / 登录态的工具（`1password-cli`、`raycast`、各 IM / 云盘类 app 等）。
- 要追加条目，请直接编辑 `nix/darwin/homebrew.nix`，遵循同样的保守口径；不要贸然启用 `cleanup = "check"` 或 `autoUpdate / upgrade`。

### Phase 3B：tmux 运行时声明化（仅运行时，不重写配置体系）

- `tmux` 已加入 `nix/darwin/homebrew.nix` 的 `brews` 列表。选择 Homebrew 而不是 Home Manager `home.packages` 的原因：
  - 本机 tmux 一直由 Homebrew 安装在 `/opt/homebrew/bin/tmux`，已长期稳定；仅做清单声明化而不换 provider，零行为变化。
  - 与 `neovim` / `starship` / `git` 的口径一致，避免双份 tmux 二进制在 PATH 里互相覆盖。
  - 新机器走 Nix 路线时，`darwin-rebuild switch` 会自动补上 tmux 运行时，可复现性提升。
- 明确保持不变的边界（**本阶段不做**）：
  - 不把 `.config/tmux/tmux.conf.local` 改写为 Home Manager `programs.tmux.extraConfig`。
  - 不替换 oh-my-tmux；`~/.config/tmux/tmux.conf` 继续是指向 `~/.local/share/tmux/oh-my-tmux/.tmux.conf` 的本地软链接（机器相关，按约定 gitignore）。
  - 不引入 tmux 插件系统重构；现有 `tmux-resurrect` / `tmux-continuum` 的使用方式不变。
- 验证：`nix flake check`、`darwin-rebuild build --flake .#AresdeMacBook-Air` 均通过；`sudo darwin-rebuild switch` 需人工执行。

### Phase 3C：少量稳定 `system.defaults.*` 试点（保守首版）

- `nix/darwin/defaults.nix` 已引入，并在 `nix/darwin/default.nix` 中 import。
- 选型原则：**所有写入值与当前机器 `defaults read` 结果一致**，首次 `switch` 预期无可感知行为变化；只接管「长期几乎不改、改错易人工恢复、不牵涉隐私 / 账号态 / 外设差异」的默认项。
- **已纳入**：
  - `system.defaults.finder.AppleShowAllExtensions = true`：始终显示文件扩展名。
  - `system.defaults.finder.ShowPathbar = true`：Finder 显示路径栏。
  - `system.defaults.finder.ShowStatusBar = true`：Finder 显示状态栏。
  - `system.defaults.finder.FXPreferredViewStyle = "Nlsv"`：Finder 默认使用列表视图。
  - `system.defaults.dock.mru-spaces = false`：关闭 Mission Control 空间按最近使用自动重排。
  - `system.defaults.NSGlobalDomain.KeyRepeat = 2`、`InitialKeyRepeat = 30`：与当前机器一致的键盘重复速率（tick 单位）。
- **刻意未纳入**（保留给后续阶段单独评估）：
  - `NSGlobalDomain.ApplePressAndHoldEnabled`：当前 unset，不做主动置位。
  - 自动替换 / 自动引号 / 自动破折号 / 拼写纠正等 NSGlobalDomain 开关：当前 unset，不做主动置位。
  - Dock：`autohide`、`persistent-apps`、`tilesize`、`orientation` 等偏好漂移项。
  - Finder：`_FXShowPosixPathInTitle`、`ShowHardDrivesOnDesktop` 等非本机常用项。
  - 触控板 / trackpad、窗口动画、通知中心、loginwindow、软件更新策略、输入法等整类偏好。
- 追加新项前请在 `nix/darwin/defaults.nix` 注释中先确认三条：(1) 当前机器已经稳定使用该值；(2) 是长期几乎不改的设置；(3) 改错后易人工恢复。不满足就继续延后，不要“能配就都配”。
- 回滚：`sudo darwin-rebuild switch --rollback` 回退到前一代系统；或手动 `defaults write` 覆盖个别 key。
- 验证：`nix flake check`、`darwin-rebuild build --flake .#AresdeMacBook-Air` 均通过；`sudo darwin-rebuild switch` 需人工执行（预期是空变化）。

### Phase 4 最小版：三件事（已落地）

Phase 4 最小版故意把范围收得很紧，只做三件独立、可独立回退的事：

1. **`brew services` 试点，仅接管 `borders` 与 `nginx`**
   - `nix/darwin/homebrew.nix` 的 `brews` 里以 `{ name = "..."; start_service = true; }` 形式声明。
   - `start_service = true` 的语义：nix-darwin 在 `brew bundle` 阶段调用 `brew services start`，**仅在服务未运行时启动并登记为 login item**，不会重启或停止已运行服务。
   - 本机当前两者均已 `started`，首次 switch 是幂等 no-op；新机器首次 switch 会自动补上。
   - 刻意未用 `restart_service`：避免每次 `darwin-rebuild switch` 都打断长期运行的服务。
   - 仍未纳入：`clouddrive2`（账号态 / 本地数据）、`ollama`（本地模型数据）、`unbound` / `colima`（非默认开机自启），继续按 `brew services` 命令人工管理。

2. **补上 Ghostty 依赖字体 `font-maple-mono-nf`**
   - `.config/ghostty/config` 中 `font-family = Maple Mono Normal NF CN` 明确依赖该字体；新机器缺少它会直接 fallback。
   - 刻意未纳入 `font-hack-nerd-font`：本机虽已安装但未被任何仓库配置引用；本轮字体只补「仓库明确依赖的」，避免“能配就都配”。

3. **把 `hammerspoon` cask 纳入清单**
   - cask 来源与 `ghostty` 相同（Homebrew 主干）。
   - 仅纳入 cask 安装层，**不动** `.hammerspoon/init.lua` 自身；配置事实源仍是仓库根目录的 `.hammerspoon/`，通过 `setup_mac.sh` 软链接到 `~/.hammerspoon`。
   - 新机器激活步骤见下文「Hammerspoon 激活说明」。

验证：`nix flake check`、`darwin-rebuild build --flake .#AresdeMacBook-Air` 均通过；`sudo darwin-rebuild switch` 需人工执行。

### Hammerspoon 激活说明（新机器）

`hammerspoon` cask 虽已纳入 Nix 清单，但 macOS 的 Accessibility 权限无法由 Nix 或任何脚本自动授予。在一台新 Mac 上完整启用本仓库的 Hammerspoon 自动化，顺序如下：

1. **安装 Hammerspoon app**
   - 走 Nix 路线：`sudo darwin-rebuild switch --flake .#AresdeMacBook-Air`（Phase 4 最小版起，`hammerspoon` 由 `nix/darwin/homebrew.nix` 自动安装）。
   - 不走 Nix 路线：`brew install --cask hammerspoon`。
2. **同步仓库配置**：运行 `./setup_mac.sh`，同意把 `.hammerspoon` 软链接到 `~/.hammerspoon`。
3. **（关键）授予 Accessibility 权限**
   - 首次启动 Hammerspoon 时 macOS 会弹窗请求「辅助功能」权限，点击允许；
   - 如果弹窗被忽略或已拒绝，请手动打开「系统设置 → 隐私与安全性 → 辅助功能」，找到 Hammerspoon 并勾选；
   - 未授权时，`init.lua` 里的事件 tap（双击 Cmd+W/Q、右 Cmd → F19 等）与大部分热键都会静默失效，且**没有**任何兜底提示。
4. **确认 Ghostty 已安装**：`Ctrl+Alt+T` 会调用 `hs.application.get("Ghostty")`；Ghostty 本身已在 `nix/darwin/homebrew.nix` 的 `casks` 中声明，走 Nix 路线会被自动安装。
5. **验证**：触发 `Ctrl+Alt+Cmd+R` 重载 Hammerspoon；`Ctrl+Alt+T` 应该能前台 Ghostty 或新建窗口；双击 `Cmd+W` / `Cmd+Q` 应该能正常工作。

若 Hammerspoon 在 Accessibility 列表里已勾选但仍不生效，常见原因是升级后条目失效：在列表里先**删除** Hammerspoon，再重新添加并重启 Hammerspoon。

### 后续阶段的路线图

- **Phase 4（最小版已完成）**：后续是否扩大范围（如更多 `brew services`、更多字体、其它 GUI 自动化 app）继续按「谨慎、可回退、逐项评估」的原则推进，不做扫荡式接管。
- **Phase 5**：再讨论更大范围的系统默认项、以及更大范围的本机自动化迁移。

每个阶段都应单独一次提交，并更新本章节。
