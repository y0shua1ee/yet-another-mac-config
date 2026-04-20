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
| `flake.nix` + `nix/` | 渐进式 Nix 迁移骨架（Phase 2A：低风险 CLI 包 + 通用环境变量；`~/.zshrc` 仍由仓库软链接托管） |

## 使用说明

1. 赋予脚本执行权限：`chmod +x setup_mac.sh`
2. 执行脚本：`./setup_mac.sh`
3. 根据提示输入目标 macOS 用户名，脚本会逐个遍历仓库中已跟踪的 `.config` 一级配置目录，并在 `/Users/<username>/.config` 中创建软链接；若某个目标项已存在，会先确认是否覆盖，默认则跳过。
4. 如果当前工作区里本地存在 `.codex/config.toml`，脚本会额外询问是否同步到 `~/.codex/config.toml`；该文件默认只保留在本地，不会提交到仓库。
5. 脚本会询问是否将 `zsh/.zshrc` 软链接到 `~/.zshrc`。通用配置（主题、插件、补全等）存放在此文件中；API 密钥、项目变量等隐私内容应写入 `~/.zshrc.local`（不纳入版本控制），会在 `.zshrc` 末尾自动加载。
6. 脚本会检测 `.config/tmux` 是否缺少 `tmux.conf`，如果缺少则提示安装 [oh-my-tmux](https://github.com/gpakosz/.tmux)，自动克隆到 `~/.local/share/tmux/oh-my-tmux` 并创建软链接。
7. 脚本会检测仓库根目录下的 `.hammerspoon`，提示是否同步到 `~/.hammerspoon`；在此之前请先通过 `brew install --cask hammerspoon` 安装好 Hammerspoon，并根据需要安装 `Ghostty`（例如 `brew install --cask ghostty`）以使用 `Ctrl+Alt+T` 新开 Ghostty 窗口的快捷方式。

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

- `flake.nix`：入口，定义 `darwinConfigurations.AresdeMacBook-Air`。
- `nix/darwin/default.nix`：**最小**系统层配置；关键点：`nix.enable = false;` 让 Determinate Nix 自行管理守护进程，避免与 nix-darwin 冲突。
- `nix/home/default.nix`：Home Manager 用户层入口；默认**不 import** zsh 模块。
- `nix/modules/zsh.nix`：首版 zsh Home Manager 模块，仅搬入 `zsh/.zshrc` 中「安全/通用」子集（`EDITOR=nvim`、bun、`PATH` 去重、yazi 辅助函数、Claude Code 别名、`~/.zshrc.local` 入口），Phase 1 **故意不启用**。

### Phase 1 暂不管理（仍按原方式）

- `~/.zshrc` 仍是仓库里 `zsh/.zshrc` 的软链接；现有 Oh My Zsh 主题与插件继续沿用现有脚本。
- Homebrew 软件包、`brew services`、系统默认值（`system.defaults.*`）、字体、应用等一律保持现状。
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
> - 激活过程若遇到「existing file would be overwritten」类冲突，Phase 1 的 `home-manager.backupFileExtension = "hm-backup"` 会把目标文件备份为 `*.hm-backup`；Phase 1 默认不启用 zsh 模块，所以不会碰 `~/.zshrc`。
> - 本骨架不执行任何卸载/清理类操作；如果激活失败，回退方式就是不再运行 `switch`，原有 dotfile 保持不变。

### Phase 2A 现已管理（低风险 Home Manager 扩展）

- `nix/home/packages.nix`：把若干稳定纯 CLI 工具（`ripgrep`、`fd`、`jq`、`tree`、`bat`）交给 Home Manager 的 `home.packages`。激活后会装到 `/etc/profiles/per-user/<user>/bin`，与 Homebrew 版本共存、互不覆盖。
- `nix/home/shell-env.nix`：声明通用非私密变量 `EDITOR=nvim` / `VISUAL=nvim` / `PAGER=less`。目前 `programs.zsh` 仍未启用，这些变量**暂不在登录 shell 中生效**，由 `zsh/.zshrc` 里的 `export EDITOR=nvim` 继续承担运行时职责；等 Phase 2B 打开 zsh 模块后它们会自动接管。

Phase 2A **仍然不接管** `~/.zshrc`、`~/.zshrc.local`、Homebrew casks、GUI 应用、`.hammerspoon`、`system.defaults.*`、`brew services`、字体等。

### Phase 2B 预备重构（已完成）

- `zsh/shared.zsh`：新增一份共享 shell 片段，承载 `zsh/.zshrc` 与 `nix/modules/zsh.nix` 共同需要的公开、跨机器通用逻辑，减少两边继续漂移。
- 当前软链接版 `zsh/.zshrc` 仍保留 Oh My Zsh、Homebrew completion 与本机 OpenClaw completion 的调用方职责；未来如果启用 Home Manager 的 `programs.zsh`，`nix/modules/zsh.nix` 会直接复用同一份 `zsh/shared.zsh`。
- `~/.zshrc.local` 仍保留为私有覆盖入口，未被接管。

### 后续阶段的路线图（暂定）

- **Phase 2B**：真正启用 `nix/modules/zsh.nix`，替换 `~/.zshrc` 软链接；同步把 Oh My Zsh 主题/插件迁到 Home Manager 原生能力或 `programs.zsh.oh-my-zsh`。届时 `shell-env.nix` 里的 sessionVariables 会生效，可从 `zsh/.zshrc` 中移除对应 `export`。
- **Phase 3**：将更多 Homebrew 软件包迁到 `nix-darwin` 的 `homebrew` 模块（声明式）或继续扩充 Home Manager `home.packages`。
- **Phase 4**：逐步把 `system.defaults.*`、字体、服务纳入管理。

每个阶段都应单独一次提交，并更新本章节。
