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
| `.config/tmux` | tmux（基于 oh-my-tmux） |
| `.config/typora` | Typora 自定义主题 |
| `.config/yazi` | Yazi 文件管理器及插件 |
| `.hammerspoon` | Hammerspoon 自动化（含 `hs.ipc` CLI 控制） |
| `safety/` | 离线安全验证控制面（仓库外 fixture/store、隐私 gate、sentinel 与 scoped readiness report） |
| `zsh/.zshrc` | Zsh 备用软链接入口（当前主路径由 Home Manager 的 `nix/modules/zsh.nix` 接管；二者共享 `zsh/shared.zsh`） |
| `flake.nix` + `nix/` | 可选的渐进式 Nix 路线，用来在新机上更快补齐部分运行时与系统偏好；覆盖范围与启用方式见 [`nix/README.md`](nix/README.md) |

## 前置依赖

- 需要先安装 [Homebrew](https://brew.sh/)；本仓库的各类 app、CLI、后台服务均通过 `brew` 安装。
- `setup_mac.sh` 只负责在新机器上同步本仓库的配置（建立软链接等），**不会**安装 Homebrew 本身，也不会替你安装具体的 app。

## 使用说明

1. 赋予脚本执行权限：`chmod +x setup_mac.sh`
2. 执行脚本：`./setup_mac.sh`
3. 根据提示输入目标 macOS 用户名，脚本会逐个遍历仓库中已跟踪的 `.config` 一级配置目录，并在 `/Users/<username>/.config` 中创建软链接；若某个目标项已存在，会先确认是否覆盖，默认则跳过。
4. 如果当前工作区里本地存在 `.codex/config.toml`，脚本会额外询问是否同步到 `~/.codex/config.toml`；该文件默认只保留在本地，不会提交到仓库。
5. 脚本会询问是否将 `zsh/.zshrc` 软链接到 `~/.zshrc`。这是非 Nix / Home Manager 场景的备用入口；当前 Nix 路线会由 Home Manager 生成 `~/.zshrc`。API 密钥、项目变量等隐私内容应写入 `~/.zshrc.local`（不纳入版本控制），会在 zsh 初始化末尾自动加载。
6. 脚本会检测 `.config/tmux` 是否缺少 `tmux.conf`，如果缺少则提示安装 [oh-my-tmux](https://github.com/gpakosz/.tmux)，自动克隆到 `~/.local/share/tmux/oh-my-tmux` 并创建软链接。
7. 脚本会检测仓库根目录下的 `.hammerspoon`，提示是否同步到 `~/.hammerspoon`。同步前先用 `brew install --cask hammerspoon` 安装 app，同步后仍需在「系统设置 → 隐私与安全性 → 辅助功能」中授予 Hammerspoon 权限，否则 `init.lua` 里的事件 tap 与快捷键不会生效；`Ctrl+Alt+T` 快捷键还依赖 Ghostty。当前配置会加载 `hs.ipc`，Hammerspoon 启动后可用 `/Applications/Hammerspoon.app/Contents/Frameworks/hs/hs -c 'hs.reload()'` 远程刷新配置。Nix 路线下 cask 与字体已声明化，完整激活流程见 [`nix/README.md`](nix/README.md)。

## 安全验证控制面（Phase 1）

`safety/` 用来离线验证仓库声明、synthetic fixture、artifact lineage、隐私边界与有限 readiness claim。它不会安装、更新、激活、修复或清理当前 Mac；本仓库始终是期望状态的 source of truth，真实机器只是以后经用户确认的 activation target。

需要本机已经有可用的 Go；缺失时 runner 会返回 `manual-required` / `32`，不会联网下载或自动安装工具链。常用命令：

```bash
./safety/scripts/test.sh task phase-e2e
./safety/scripts/test.sh task docs-and-phase-gate
./safety/scripts/test.sh wave phase-integration
./safety/scripts/test.sh phase
```

前三条分别是完整 E2E task、纯结构文档 task 和聚合这两个 task 的最终 wave；完整 phase 必须用第四条单独运行。runner 使用空白 allowlist 环境和仓库外的新建临时根，网络默认 denied，生成的 fixture、Go cache、manager roots 与 artifact store 都不写入仓库。公开 `fixture run` 只接受仓库外 base 与 logical fixture ID，并由 ownership 状态机建立 fresh direct child；它不接受已有物理 fixture/store root。公共结果只接受 closed field/type schema；run identity 是 digest-derived opaque ID，suite/operation identity 来自固定 registry。15 / 47 / 305 秒预算由入口 watchdog 覆盖 setup、固定检查、测试、child dispatch 和 marker-owned cleanup；超时只返回一个有界 envelope 与退出码 `124`。

当前 service surface 所需的 `launchctl print` isolated negative proof 尚未纳入 tracked manifest，因此 current-host 路径会在任何真实 adapter 或 workload 之前停止为 `manual-required` / `indeterminate`。standalone `report` 只输出 claim-ineligible 的 synthetic/replay 状态；完整 outer sequence 与 `covered-surfaces-unchanged-for-run` 只能在同一次受控 real envelope 的 one-shot capability 内产生，且正向路径只由 proof-valid isolated private doubles 验证。这不表示当前 Mac 已通过，也不表示整机、多机或重装恢复已经验证。详细契约见 [`safety/README.md`](safety/README.md)。

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

下表中的 app / 服务需要你自己用 Homebrew 安装（例如 `brew install nginx`、`brew install --cask clouddrive`），`setup_mac.sh` 只负责同步相关配置，不会替你安装这些 app；装好之后再用 `brew services` 接管运行状态。

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
- `.config/gh/hosts.yml`：GitHub CLI 登录 / 凭据状态，部分环境会写入 OAuth token。
- `.config/himalaya/`：Himalaya 邮箱账号配置，包含邮箱地址与 app password。
- `.config/mpv/watch_later/`：mpv 本地观看进度，可能暴露媒体历史。
- `.config/op/`：1Password CLI 的本地设备信息。
- `.config/linearmouse/`：鼠标与触控板的本机硬件配置。
- `.config/mole/`：清理工具运行日志与本地运行状态。
- `.config/raycast/`：Raycast 本地扩展与缓存数据。
- `.config/jgit/`：Jujutsu / Git 相关本地配置。
- `.config/tmux/tmux.conf`：oh-my-tmux 主配置的软链接，指向本地克隆路径，机器相关。
- `.config/ghostty/*.bak`：Ghostty 配置备份文件。
- `.DS_Store`：macOS 自动生成的目录元数据文件。

Safety runner 的 fixture、artifact store、隔离 HOME/XDG、Go cache 与 manager roots 也只存在于仓库外的运行时解析目录。它们默认在 verdict 冻结后按 ownership marker 删除；若运行前显式保留，也仍是本机临时状态，不应复制进仓库或用 broad `.gitignore` / Gitleaks 例外掩盖。

`setup_mac.sh` 只会处理 Git 已跟踪的 `.config` 目录，因此这些本地忽略目录不会出现在同步提示里。

如果后续新增只适用于当前机器的配置或缓存文件，建议继续补充到 `.gitignore`，避免误提交到仓库。

## 可选：使用 Nix 激活这份配置

如果你希望在新机器上用 Nix 来补齐这份仓库的部分运行时与系统层配置，可以走仓库内的渐进式 Nix 路线。当前定位是**帮助新 Mac 更快恢复到可用状态**，并不追求 100% 声明式接管：secrets、登录态、大范围 app state 与琐碎系统偏好仍然默认人工处理。

当前 Nix 路线除 Home Manager zsh、少量稳定 CLI、保守 Homebrew inventory、`borders` / `nginx` 服务试点与少量 `system.defaults` 外，也已补入 Phase 4B 的小范围 Homebrew 扩张：容器 CLI、Yazi / 媒体 / 文档 helper、Neovim / Treesitter 运行时 helper、Biya/Hermes 常用的 Apple 辅助 CLI、X/Twitter 工具 `xurl`，以及 Claude Code / Codex / CC Switch。账号态较重的 GUI app 仍刻意留待后续单独评估。

Phase 5A 起，Home Manager 还会装好语言 / 工具链管理器**入口**：`mise` / `uv` / `rustup`，并启用 `direnv` + `nix-direnv`；实际运行时版本优先由项目本地的 `.mise.toml` / `pyproject.toml + uv.lock` / `rust-toolchain.toml` / 项目 `flake.nix` devShell 管理；仓库内 `.config/mise/config.toml` 只保存少量全局 fallback。

Phase 5B–5D 已完成 switch 与 post-check：默认 Node / npm / Go 已迁到 mise，仓库内的 `.config/mise/config.toml` 固定全局 Node `24.11.0` 与 Go `1.26.3`；登录 zsh 中 `node` / `npm` / `go` 会解析到 `~/.local/share/mise/installs/...` 下的版本。Homebrew `nvm` 与 `~/.nvm` 已清理，Node / Go 完全由 mise 管理。2026-06-24 `nixpkgs` refresh 将 Home Manager 的 mise 目标版本从 `2026.4.6` 升至 `2026.6.11`；该版本在 Darwin 上跳过一个 OCI metadata 单测，并通过 runtime post-check 验证。

完整的覆盖范围、激活步骤与回滚方式见：

- 面向使用者：[`nix/README.md`](nix/README.md)
- 面向后续维护 / 约束：[`nix/CLAUDE.md`](nix/CLAUDE.md)
