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
| `zsh/.zshrc` | Zsh 通用配置入口（与 `zsh/shared.zsh` 共同提供公开 shell 逻辑；兼容旧软链接回退路径） |
| `flake.nix` + `nix/` | 渐进式 Nix 迁移配置（当前已完成：Home Manager 接管 zsh、保守 Homebrew 清单、tmux 运行时、少量稳定 `system.defaults.*`，以及 Phase 4 最小版的 `borders` / `nginx` 服务、Ghostty 字体、Hammerspoon；细节见 [`nix/README.md`](nix/README.md)） |

## 使用说明

1. 赋予脚本执行权限：`chmod +x setup_mac.sh`
2. 执行脚本：`./setup_mac.sh`
3. 根据提示输入目标 macOS 用户名，脚本会逐个遍历仓库中已跟踪的 `.config` 一级配置目录，并在 `/Users/<username>/.config` 中创建软链接；若某个目标项已存在，会先确认是否覆盖，默认则跳过。
4. 如果当前工作区里本地存在 `.codex/config.toml`，脚本会额外询问是否同步到 `~/.codex/config.toml`；该文件默认只保留在本地，不会提交到仓库。
5. 脚本会询问是否将 `zsh/.zshrc` 软链接到 `~/.zshrc`。通用配置（主题、插件、补全等）存放在此文件中；API 密钥、项目变量等隐私内容应写入 `~/.zshrc.local`（不纳入版本控制），会在 `.zshrc` 末尾自动加载。
6. 脚本会检测 `.config/tmux` 是否缺少 `tmux.conf`，如果缺少则提示安装 [oh-my-tmux](https://github.com/gpakosz/.tmux)，自动克隆到 `~/.local/share/tmux/oh-my-tmux` 并创建软链接。
7. 脚本会检测仓库根目录下的 `.hammerspoon`，提示是否同步到 `~/.hammerspoon`。在此之前请先安装 Hammerspoon：走 Nix 路线时由 `nix/darwin/homebrew.nix` 声明自动安装（Phase 4 最小版已纳入），不走 Nix 时用 `brew install --cask hammerspoon`。同步后仍需手动在「系统设置 → 隐私与安全性 → 辅助功能」授予 Hammerspoon 权限，否则 `init.lua` 里的事件 tap 与快捷键不会生效；`Ctrl+Alt+T` 快捷键还依赖 Ghostty cask，已一并在 Homebrew 清单中声明。完整激活流程见 [`nix/README.md`](nix/README.md) 的相关说明。

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

## 可选：使用 Nix 激活这份配置

如果你希望在新机器上用 Nix 来补齐这份仓库的部分运行时与系统层配置，可以走仓库内的渐进式 Nix 路线。

- 面向使用者的说明：[`nix/README.md`](nix/README.md)
- 面向后续维护 / 约束的说明：[`nix/CLAUDE.md`](nix/CLAUDE.md)

当前这条 Nix 路线的定位是：**帮助新 Mac 更快恢复到可用状态**，但不追求 100% 声明式接管。像 secrets、登录态、较大范围 app state，以及很多琐碎系统偏好，仍然默认保留人工处理。

### 当前会通过 Nix 补齐什么

- Home Manager 接管的 zsh 入口与少量通用 shell 环境
- 一份保守的 Homebrew inventory（不会自动 cleanup 未声明条目）
- `tmux` 运行时
- 少量稳定的 `system.defaults.*`
- 最小版 `brew services` 试点：`borders` / `nginx`
- `Ghostty` 依赖字体 `font-maple-mono-nf`
- `hammerspoon` cask（但辅助功能权限仍需手动授予）

### 最短激活路径

```bash
# 1) 安装 Determinate Nix（仅首次）
curl -fsSL https://install.determinate.systems/nix | sh -s -- install

# 2) 进入仓库，生成 / 检查 flake
nix flake lock
nix flake check

# 3) 首次 build（无需 sudo）
nix run github:nix-darwin/nix-darwin/master#darwin-rebuild -- \
  build --flake .#AresdeMacBook-Air

# 4) 首次 switch（需要 sudo）
sudo nix run github:nix-darwin/nix-darwin/master#darwin-rebuild -- \
  switch --flake .#AresdeMacBook-Air
```

第二次以后可直接用：

```bash
darwin-rebuild build --flake .#AresdeMacBook-Air
sudo darwin-rebuild switch --flake .#AresdeMacBook-Air
```

> 详细边界、阶段说明、回滚方式与 Hammerspoon / services 的补充说明，见 [`nix/README.md`](nix/README.md)。
