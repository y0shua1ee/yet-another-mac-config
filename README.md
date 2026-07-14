# yet-another-mac-config

My Mac config

## 包含的配置

| 目录/文件 | 说明 |
|-----------|------|
| `.config/aerospace` | AeroSpace 窗口管理器 |
| `.config/borders` | JankyBorders 窗口边框 |
| `.config/btop` | btop 系统监控 |
| `.config/ghostty` | Ghostty 终端（含 custom shader collection） |
| `.config/gh` | GitHub CLI 共享偏好（凭据状态保留在本机 `hosts.yml`） |
| `.config/mise` | mise 全局工具链配置（当前固定 Node `24.11.0` 与 Go `1.26.3` 作为全局 fallback） |
| `.config/mpv` | mpv 播放器主配置（播放进度状态保留在本机） |
| `.config/nvim` | Neovim（基于 LazyVim） |
| `.config/tmux` | tmux 自定义配置；oh-my-tmux 主配置由 `flake.lock` 固定并通过 Home Manager 链接 |
| `.config/typora` | Typora 自定义主题 |
| `.config/yazi` | Yazi 文件管理器及插件 |
| `.hammerspoon` | Hammerspoon 自动化（含 `hs.ipc` CLI 控制） |
| `zsh/.zshrc` | Zsh 备用软链接入口（当前主路径由 Home Manager 的 `nix/modules/zsh.nix` 接管；二者共享 `zsh/shared.zsh`） |
| `flake.nix` + `nix/` | Determinate Nix + nix-darwin + Home Manager 主控制面；包含多主机 profile、软件清单、系统偏好与配置链接，详见 [`nix/README.md`](nix/README.md) |
| `sync_mac.sh` | 根据当前 Mac 的 LocalHostName 选择 profile，先构建、确认后再激活 |

## 主同步流程

- 先安装 [Determinate Nix](https://docs.determinate.systems/) 与 [Homebrew](https://brew.sh/)。Determinate Nix 提供 Nix 与 daemon；nix-darwin 的 Homebrew module 只管理清单，不负责安装 Homebrew 本身。
- 把仓库 clone 到 `nix/hosts/default.nix` 中当前主机声明的 `repoPath`。
- 正常同步运行 `./sync_mac.sh`。脚本会使用 `scutil --get LocalHostName` 选择 `darwinConfigurations.<host>`，先执行 build，只有确认后才执行需要 `sudo` 的 switch。
- 只验证、不激活时运行 `./sync_mac.sh --build-only`。完整的首次安装、多主机添加与回滚流程见 [`nix/README.md`](nix/README.md)。

## Ghostty

`.config/ghostty` 管理 Ghostty 主配置与 custom shader collection。当前 shader collection 来自 [`0xhckr/ghostty-shaders`](https://github.com/0xhckr/ghostty-shaders)，已把上游顶层 `*.glsl` 文件 vendored 到 `.config/ghostty/shaders/`。

当前默认启用轻量光标拖尾：

```ini
custom-shader = ~/.config/ghostty/shaders/cursor_blaze.glsl
custom-shader-animation = true
```

切换效果时只需要改 `custom-shader` 指向的文件名，然后运行：

```bash
/Applications/Ghostty.app/Contents/MacOS/ghostty +validate-config --config-file="$HOME/.config/ghostty/config"
```

## Neovim / LazyVim

`.config/nvim` 基于 [LazyVim starter](https://github.com/LazyVim/starter)。新机器建议先补齐运行时依赖：

```bash
brew install neovim tree-sitter-cli
```

首次启动 `nvim` 会自动 bootstrap `lazy.nvim` 与 LazyVim 插件。常用维护命令：`:Lazy`、`:LazyExtras`、`:Mason`、`:checkhealth lazy`、`:checkhealth`。

## Yazi 插件同步

`install_yazi_plugins.sh` 用来在新环境里批量安装/更新 `package.toml` 中锁定的所有 Yazi 插件，并按需设置部分环境变量（比如 `LG_CONFIG_FILE`，确保 `lazygit.yazi` 能工作）。使用方式：

当前 Yazi 配置使用 v26+ 的 opener 占位符（如 `%s` / `%s1` / `%d1`）。`l` 键由 `smart-enter.yazi` 接管：目录会进入，文件会走 `[opener].edit`，默认用 `$EDITOR`（当前为 `nvim`）打开选中文件。

1. 确认 `ya` CLI 已安装：`brew install yazi`。
2. 可选：指定配置目录，例如 `./install_yazi_plugins.sh --config-dir "$HOME/.config/yazi"`；若不传参数脚本会优先使用 `XDG_CONFIG_HOME/yazi`，否则回退到仓库内 `.config/yazi`。
3. 等待脚本自动执行 `ya pkg install`，输出当前生效的插件列表，并提示缺失的依赖工具（如 `git`、`starship`、`lazygit`、`7zz`、`magick` 等）。

脚本可安全重复执行，方便在多台机器间保持插件一致。

## 容器运行环境（可选）

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

下表中已进入 `nix/darwin/homebrew.nix` 的 app / 服务会在 switch 时由 nix-darwin 补齐；未进入清单的条目仍需按需使用 Homebrew 安装与管理。

以下服务通过 `brew services` 管理：

| 服务 | 说明 | 开机自启 |
|------|------|----------|
| borders | JankyBorders 窗口边框 | 是 |
| nginx | HTTP 服务器（默认端口 8080） | 是 |
| clouddrive2 | CloudDrive2 云盘挂载 | 是 |
| unbound | DNS resolver | 否 |
| colima | Colima 容器运行时（可选） | 否 |

常用命令：

```bash
brew services list              # 查看当前运行状态
brew services start <name>      # 启动服务（开机自启）
brew services stop <name>       # 停止服务（取消开机自启）
brew services restart <name>    # 重启服务
```

> **注意：** nginx 的配置路径为 `/opt/homebrew/etc/nginx/`。
> **注意：** 走 Nix 路线时，`borders` / `nginx` 的开机自启会在首次 switch 时由 nix-darwin 帮忙拉起（日常 start/stop/restart 仍用上面的 `brew services` 命令）；其余服务继续按本节命令人工管理。详见 [`nix/README.md`](nix/README.md)。

## 本地文件同步约定

以下内容仅保留在本地环境，不会同步到仓库：

- `.codex/`：Codex 本地配置。
- `.claude/`：Claude Code 的项目级状态（worktrees、settings.local.json 等）。
- `.config/alma/`：Alma 的聊天、媒体、任务、认证桥接与运行状态；整个目录都属于本机私密数据，不纳入版本控制。
- `.config/gh/hosts.yml`：GitHub CLI 登录 / 凭据状态，部分环境会写入 OAuth token。
- `.config/himalaya/`：Himalaya 邮箱账号配置，包含邮箱地址与 app password。
- `.config/mpv/watch_later/`：mpv 本地观看进度，可能暴露媒体历史。
- `.config/op/`：1Password CLI 的本地设备信息。
- `.config/linearmouse/`：鼠标与触控板的本机硬件配置。
- `.config/mole/`：清理工具运行日志与本地运行状态。
- `.config/raycast/`：Raycast 本地扩展与缓存数据。
- `.config/jgit/`：Jujutsu / Git 相关本地配置。
- `.config/tmux/plugins`：TPM 安装、更新的可变插件树，实际位于 `~/.config/tmux/plugins`；仓库同名路径如存在，只是旧版兼容链接。
- `.config/ghostty/*.bak`：Ghostty 配置备份文件。
- `.DS_Store`：macOS 自动生成的目录元数据文件。

Home Manager 的配置链接使用显式 allowlist；本地忽略目录不会被自动同步。旧版 `setup_mac.sh` 已退役，避免它与 Home Manager 同时拥有相同目标。

如果后续新增只适用于当前机器的配置或缓存文件，建议继续补充到 `.gitignore`，避免误提交到仓库。

## Determinate Nix + Home Manager 管理边界

这个仓库是 Mac 期望状态的事实源：Determinate Nix 管理 Nix 与 daemon，nix-darwin 负责机器级组合和激活，Home Manager 负责用户配置、shell、工具入口和仓库配置链接。Homebrew、mise、uv、rustup 继续作为明确委托的安装或运行时 owner。

Home Manager 当前会把受跟踪的 AeroSpace、borders、btop、GitHub CLI 共享偏好、Ghostty、mise、mpv、Neovim、Typora、Yazi 与 Hammerspoon 配置链接到 home 目录。tmux 单独按文件接管：主配置来自 `flake.lock` 固定的 oh-my-tmux source，`tmux.conf.local` 指向仓库，TPM 插件留在本机可写目录。仓库链接目标依赖当前工作区，因此仓库必须保留在主机 profile 声明的 `repoPath`。

这不等于同步所有本机状态。secrets、账号登录态、聊天和媒体、TCC / Accessibility 权限、缓存、设备专属数据与大范围 app state 仍然留在本机，并通过 `.gitignore` 与显式 allowlist 排除。

当前 Nix 路线除 Home Manager zsh、少量稳定 CLI、保守 Homebrew inventory、`borders` / `nginx` 服务试点与少量 `system.defaults` 外，也已补入 Phase 4B 的小范围 Homebrew 扩张：容器 CLI、Yazi / 媒体 / 文档 helper、Neovim / Treesitter 运行时 helper、Biya/Hermes 常用的 Apple 辅助 CLI、X/Twitter 工具 `xurl`，以及 Claude Code / Codex / CC Switch。账号态较重的 GUI app 仍刻意留待后续单独评估。

Phase 5A 起，Home Manager 还会装好语言 / 工具链管理器**入口**：`mise` / `uv` / `rustup`，并启用 `direnv` + `nix-direnv`；实际运行时版本优先由项目本地的 `.mise.toml` / `pyproject.toml + uv.lock` / `rust-toolchain.toml` / 项目 `flake.nix` devShell 管理；仓库内 `.config/mise/config.toml` 只保存少量全局 fallback。

Phase 5B–5D 已完成 switch 与 post-check：默认 Node / npm / Go 已迁到 mise，仓库内的 `.config/mise/config.toml` 固定全局 Node `24.11.0` 与 Go `1.26.3`；登录 zsh 中 `node` / `npm` / `go` 会解析到 `~/.local/share/mise/installs/...` 下的版本。Homebrew `nvm` 与 `~/.nvm` 已清理，Node / Go 完全由 mise 管理。2026-07-14 更新后的 nixpkgs 提供 mise `2026.7.5`，旧版 Darwin 单测 workaround 已移除，恢复使用上游可缓存 derivation。

完整的覆盖范围、多主机 profile、激活步骤与回滚方式见：

- 面向使用者：[`nix/README.md`](nix/README.md)
- 面向后续维护 / 约束：[`nix/CLAUDE.md`](nix/CLAUDE.md)
