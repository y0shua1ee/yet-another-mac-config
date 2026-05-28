# Nix 配置指南（面向 agents）

## 目的 / 当前阶段
- 这是渐进式 Nix 迁移，底层运行时是 [Determinate Nix](https://docs.determinate.systems/)，系统层用 [nix-darwin](https://github.com/nix-darwin/nix-darwin)，用户层用 [Home Manager](https://nix-community.github.io/home-manager/)。
- **Phase 2D 现状：Home Manager 已接管 `~/.zshrc`。`nix flake check`、`darwin-rebuild build --flake .#AresdeMacBook-Air` 与 `sudo darwin-rebuild switch --flake .#AresdeMacBook-Air` 都已在当前机器跑通。**
- **Phase 3A 现状：已引入保守的 Homebrew 声明式清单（`nix/darwin/homebrew.nix`）。`nix flake check` 与 `darwin-rebuild build --flake .#AresdeMacBook-Air` 均通过；`switch` 已由人工执行成功。onActivation 全部设为 `false` / `"none"`，不自动 update / upgrade / cleanup，也不会卸载任何本机已有的 brew 包。**
- **Phase 3B 现状：tmux 运行时已纳入 `nix/darwin/homebrew.nix` 的 `brews`，继续使用 Homebrew 提供的 `/opt/homebrew/bin/tmux`，没有转入 Home Manager `home.packages`。现有 oh-my-tmux + `~/.config/tmux/tmux.conf.local` 工作流保持不变；`tmux.conf` 仍是指向本地 oh-my-tmux 克隆的软链接（机器相关，不纳入仓库）。`nix flake check` 与 `darwin-rebuild build --flake .#AresdeMacBook-Air` 均通过。**
- **Phase 3C 现状：新增 `nix/darwin/defaults.nix`，以保守首版接管少量 `system.defaults.*`。所有写入值均与当前机器 `defaults read` 结果一致（Finder 四项 + Dock `mru-spaces` + NSGlobalDomain 键盘重复两项），首次 switch 预期无可感知行为变化。`nix flake check` 与 `darwin-rebuild build --flake .#AresdeMacBook-Air` 均通过；`sudo darwin-rebuild switch --flake .#AresdeMacBook-Air` 已随后续 Phase 4B / 5A 的 switch 在当前机器一并生效。**
- **Phase 4 最小版现状（已落地）：严格三件事——(1) `brew services` 试点：在 `nix/darwin/homebrew.nix` 的 `brews` 里以 `{ name; start_service = true; }` 的形式声明 `borders` 与 `nginx`；`start_service = true` 只在服务未运行时启动，不会重启或停止已运行服务，对本机状态零扰动。(2) 补上 Ghostty 依赖字体 `font-maple-mono-nf`。(3) 把 `hammerspoon` 纳入 `casks`。`nix flake check` 与 `darwin-rebuild build --flake .#AresdeMacBook-Air` 均通过；`sudo darwin-rebuild switch --flake .#AresdeMacBook-Air` 已随后续 Phase 4B / 5A 的 switch 在当前机器一并生效。**
- **Phase 4B 现状（已 switch，post-check 完成）：在 Phase 4 最小版基础上做一次小步扩张，仅追加“仓库工作流已经长期使用”的条目，激活策略保持不变（`autoUpdate = false` / `upgrade = false` / `cleanup = "none"`，`brew services` 仍只接管 `borders` / `nginx`）。新增 taps：`felixkratz/formulae`、`antoniorodr/memo`、`steipete/tap`、`xdevplatform/tap`。新增 brews：容器组 `colima` / `docker` / `docker-compose`（`colima` **不**走 `brew services`），Yazi / 媒体 / 文档 helper `sevenzip` / `imagemagick` / `mpv` / `poppler` / `zoxide` / `media-info` / `exiftool`，Neovim / AstroNvim helper `tree-sitter-cli`，Email / 助手 CLI `himalaya` / `antoniorodr/memo/memo` / `steipete/tap/remindctl`。新增 casks：`claude-code@latest` / `codex` / `cc-switch` / `xurl`（凭据仍由本机 `~/.xurl` 管理）。**故意不纳入**：多语言运行时（`go` / `rust` / `nvm` / `pnpm` / `uv` / `deno` / `python@*` / `llvm` 等）和账号态 / 登录态重的 GUI app（`raycast` / `telegram` / `discord` / `feishu` / `google-drive` / `tailscale` / `notion` / `spotify` / `zotero` / `jetbrains-toolbox` / `termius` 等）。`nix flake check` 与 `darwin-rebuild build --flake .#AresdeMacBook-Air` 均通过；`sudo darwin-rebuild switch --flake .#AresdeMacBook-Air` 已在当前机器执行成功。Post-check：登录 zsh 中 `claude` 解析为 `/opt/homebrew/bin/claude`（v2.1.120）、`codex` 解析为 `/opt/homebrew/bin/codex`（v0.125.0），与新增 cask 经由 Homebrew 提供命令的预期一致。**
- **Phase 5A 现状（已 switch，入口已生效，post-check 完成）：新增 `nix/home/dev-toolchains.nix` 并由 `nix/home/default.nix` import。通过 `home.packages` 引入语言/工具链管理器：`mise` / `uv` / `rustup`；通过 `programs.direnv.enable = true` + `programs.direnv.nix-direnv.enable = true` 启用 direnv。该阶段严格只加入口，不迁移运行时；`sudo darwin-rebuild switch --flake .#AresdeMacBook-Air` 已在当前机器执行成功。Post-check：`mise` / `rustup` / `direnv` 均解析到 `/etc/profiles/per-user/areslee/bin/`，`node` 仍由 NVM 提供，证明 Phase 5A 没有抢占现有 Node。**
- **Phase 5B 现状（待 switch）：开始把 Node 从 NVM 迁到 mise。`.config/mise/config.toml` 已固定全局 `node = "24.11.0"`；`mise install` 与 pilot 项目 `~/Documents/mise-node-pilot` 均验证通过，`mise exec -- node -v` 为 `v24.11.0`、`mise exec -- npm -v` 为 `11.6.1`，最小 Claude Code 任务在 mise Node 下成功返回 `OK`，说明依赖 PATH 中 `node` 的 Claude/GSD hooks 可用。`nix/modules/zsh.nix` 已准备在 `~/.zshrc.local` 之后启用 `mise activate zsh`；下一次 switch 后默认 Node/NPM 应由 mise 提供。NVM、Homebrew `nvm` 与 `~/.nvm` 暂不删除，作为 fallback 留到 Phase 5C 清理。**
- **Phase 3 计划文档：`nix/phase-3-plan.md`。Phase 3C 已落地，Phase 3 可视为完成；Phase 4 / 4B 后续如何继续扩张范围仍按“谨慎、可回退”原则逐项评估，不跳跃式迁移。语言运行时（多版本管理器）方向单独留作未来 Phase 评估，可能会落到 Home Manager / devshell / `mise` 等方案，不直接走 Homebrew。**
- 这个仓库仍然是「事实源」，Nix 只是又一种可选的激活方式。当前机器上的旧仓库软链接版 zsh 仍保留在 `~/.zshrc.pre-hm-switch-backup`，便于人工回退。

## 目录结构
```
nix/
├── CLAUDE.md          # 本文件
├── AGENTS.md          # 软链接 -> CLAUDE.md（二者内容保持一致）
├── darwin/
│   ├── default.nix    # nix-darwin 系统层入口
│   ├── defaults.nix   # Phase 3C：少量稳定的 system.defaults 试点
│   └── homebrew.nix   # Homebrew 声明式清单（Phase 3A 首版；3B 加 tmux；Phase 4 最小版加 borders/nginx 服务、font-maple-mono-nf、hammerspoon；Phase 4B 小幅扩张容器/Yazi/Neovim/邮件 helper、AI cask 与 xurl）
├── home/
│   ├── default.nix       # Home Manager 用户层入口（已 import ../modules/zsh.nix 并实际生效；Phase 5A 起额外 import dev-toolchains.nix）
│   ├── packages.nix      # Phase 2A：低风险纯 CLI 工具
│   ├── shell-env.nix     # Phase 2A：通用非私密环境变量（当前机器已随 Home Manager zsh 生效）
│   └── dev-toolchains.nix # Phase 5A：语言 / 工具链管理器入口（mise / uv / rustup + direnv；Phase 5B 起 Node 默认版本由 .config/mise/config.toml 管理）
└── modules/
    └── zsh.nix        # zsh Home Manager 模块（当前机器已 takeover 生效；Phase 5B 起启用 mise activate zsh）
```

另有仓库侧共享脚本：`zsh/shared.zsh`。它被 `zsh/.zshrc` 与 `nix/modules/zsh.nix` 共同复用，用来承载公开、跨机器通用的 shell 逻辑。

根目录的 `flake.nix` 通过 `darwinConfigurations.AresdeMacBook-Air` 把上述三层装配起来，并开启 `home-manager.backupFileExtension = "hm-backup"`，用于 switch 时备份会冲突的 home-manager 目标文件。当前机器上 `~/.zshenv` 已自动备份为 `~/.zshenv.hm-backup`；`~/.zshrc` 则因本地软链接冲突，实际采用了手动先挪到 `~/.zshrc.pre-hm-switch-backup` 再 switch 的方式完成 takeover。

## 重要约束
- `darwin/default.nix` 里 `nix.enable = false;` —— Determinate Nix 自己管理 nix 守护进程，nix-darwin **不得**再接管，否则会互相覆盖。
- `home/default.nix` 现在**确实 import** 了 `modules/zsh.nix`，而且本机 switch 已经完成，所以 Home Manager 正在直接生成并管理 `~/.zshrc`。
- 当前机器的注意事项：
  1. 以后继续迭代 zsh 相关 Nix 变更时，先 `nix flake check` 与 `darwin-rebuild build --flake .#AresdeMacBook-Air`，再 `sudo darwin-rebuild switch`。
  2. 旧仓库软链接版 zsh 目前保留在 `~/.zshrc.pre-hm-switch-backup`；若想回到旧路径，可人工还原。
  3. 机器相关或绝对路径的 shell 片段（例如 OpenClaw completion）不应进仓库共享区，而应写入 `~/.zshrc.local`；Home Manager 版 zsh 通过 `initContent` 在末尾自动 `source` 它。
  4. 需要回滚时：`sudo darwin-rebuild switch --rollback`，并按需把 `~/.zshrc.pre-hm-switch-backup` 还原为 `~/.zshrc`。
- 当前阶段仍**不** 触碰：`~/.zshrc.local`、secrets / 登录态，以及大范围 `system.defaults.*` 迁移（Phase 3C 仅接管极小子集）、大范围 `brew services` 接管（Phase 4 最小版只接管 `borders` / `nginx`）、字体的全面纳管（本轮只补 Ghostty 依赖的 `font-maple-mono-nf`，`font-hack-nerd-font` 仍未纳入）、`.hammerspoon` 脚本本身的迁移（Phase 4 最小版只把 `hammerspoon` cask 纳入 Homebrew 清单，不动 `init.lua`）。这些继续按原方式管理。
- Phase 5A / 5B 的语言栈边界：
  - `nix/home/dev-toolchains.nix` 只声明工具链**管理器入口**：`mise` / `uv` / `rustup`，以及 `programs.direnv` + `nix-direnv`；不把具体语言运行时写进 Homebrew 清单。
  - Phase 5B 起 Node 默认版本由 `.config/mise/config.toml` 固定为 `24.11.0`，并通过 `nix/modules/zsh.nix` 里的 `mise activate zsh` 在 shell 中生效。
  - `mise activate zsh` 必须保留在 source `~/.zshrc.local` 之后：迁移期 NVM 先加载作为 fallback，mise 再接管默认 `node` / `npm`。不要在未验证的情况下把顺序提前或写入 `zsh/shared.zsh`。
  - 暂不删除 `~/.nvm`、不卸载 Homebrew `nvm`、不清理 NVM 旧 Node 版本；这些留给 Phase 5C 在稳定后单独处理。
  - 其它项目级版本仍由项目内文件承担：Node / Go / Deno / Bun → `.mise.toml`，Python → `pyproject.toml + uv.lock`，Rust → `rust-toolchain.toml`，需要系统库 / 编译器 → 项目 `flake.nix` 的 devShell；这些不进本仓库。
- Phase 3A 的 Homebrew 模块是“保守首版”：只纳入长期稳定、已在日常使用的少量 formula / cask，未声明的本机 brew 包不会被自动卸载；要追加新条目时，按 `nix/darwin/homebrew.nix` 里的分类说明追加即可，不要开启 `cleanup = "check"` 或 `autoUpdate / upgrade`。
- Phase 4 最小版的 `brew services` 试点严格限定在以下边界：
-  - 仅接管 `borders` 与 `nginx`；`clouddrive2` / `colima` / `unbound` 继续按现有 `brew services` 命令人工管理，不纳入。
  - 策略只用 `start_service = true`（仅未运行时启动，不重启已运行服务），**不要**改用 `restart_service`，否则每次 `darwin-rebuild switch` 都会重启服务，造成不必要中断。
  - 新增服务条目前先确认：(1) 本机已长期稳定以 `brew services` 运行；(2) 重启代价低；(3) 无账号态或本地数据风险。不满足就继续人工管理。
- Phase 4 最小版的字体纳管规则：只允许纳入「仓库配置明确引用、缺失会直接影响既有行为」的字体。当前只有 Ghostty 明确依赖 `Maple Mono Normal NF CN`，所以只纳入 `font-maple-mono-nf`。`font-hack-nerd-font` 在本机虽已安装但未被任何仓库配置引用，**不**纳入。
- Phase 4 最小版的 `hammerspoon` 纳管只是把 cask 加入 Homebrew 清单，配置事实源仍然是仓库根目录的 `.hammerspoon/`。新机器除了 `darwin-rebuild switch` 让 cask 安装到位、用 `setup_mac.sh` 同步 `.hammerspoon` 之外，**还必须人工**在「系统设置 → 隐私与安全性 → 辅助功能（Accessibility）」中勾选 Hammerspoon；同时 `Ctrl+Alt+T` 快捷键依赖 Ghostty cask 已在（同一清单已声明）。详见根 README 的「Hammerspoon 激活说明」。
- Phase 3B 只接管 tmux **运行时**，不重写配置：不要把 `.config/tmux/tmux.conf.local` 迁到 Home Manager `programs.tmux.extraConfig`，也不要替换 oh-my-tmux 或插件体系。tmux 二进制继续由 Homebrew 提供（`/opt/homebrew/bin/tmux`），配置事实源是仓库中的 `.config/tmux/tmux.conf.local` 与本地 oh-my-tmux 软链接。
- Phase 3C 的 `nix/darwin/defaults.nix` 也是“保守首版”，严格遵守以下边界：
  - 已纳入：`finder.AppleShowAllExtensions` / `finder.ShowPathbar` / `finder.ShowStatusBar` / `finder.FXPreferredViewStyle`（"Nlsv"）、`dock.mru-spaces`、`NSGlobalDomain.KeyRepeat`、`NSGlobalDomain.InitialKeyRepeat`。
  - 每一项写入值都与当前机器 `defaults read` 结果一致，首次 switch 预期无可感知行为变化。
  - 刻意未纳入：`ApplePressAndHoldEnabled`、自动替换 / 自动引号 / 拼写相关（当前全部 unset，不做主动置位）、Dock `autohide` / `persistent-apps` / `tilesize` / `orientation`、触控板、通知中心、窗口动画、loginwindow、软件更新策略、输入法等。
  - 追加新项前先确认三条：(1) 当前机器已经稳定使用该值；(2) 是长期几乎不改的设置；(3) 改错后易人工恢复。不满足就继续延后，不要“能配就都配”。
- 当前已收掉 Home Manager zsh 的 `initExtraFirst` / `initExtra` deprecated 警告：`nix/modules/zsh.nix` 现改为 `programs.zsh.initContent` + `lib.mkMerge` / `lib.mkBefore`。若之后再改 zsh 初始化顺序，优先继续沿这套写法扩展。
- 目前剩余的已知非阻断警告主要是 nix-darwin 文档构建阶段的 `builtins.derivation -> options.json` 提示。它来自上游文档/选项 JSON 生成链路，不是当前仓库业务配置出错；除非明确要为了“零 warning”牺牲文档构建，否则先不要用关闭 documentation 的方式去压它。
- 若要继续推进 Phase 3，请先阅读 `nix/phase-3-plan.md`，不要跳过其中的范围边界与回滚原则。

## 修改风格
- 保持最小改动。任何向系统层下沉（如启用 `homebrew`、`system.defaults`）的变更都应单独成一个 commit，并更新根 README 的「渐进式 Nix 迁移」章节。
- 新增 Home Manager 模块放在 `modules/`，并在 `home/default.nix` 中显式 import。
- 如果某段 zsh 逻辑既要被当前软链接版 `.zshrc` 使用、又要被未来 Home Manager zsh 使用，优先考虑放进 `../zsh/shared.zsh`，不要在 `.zshrc` 与 `modules/zsh.nix` 里各写一份。
- Nix 代码中的注释使用中文（与仓库整体风格一致）。

## 常用命令（需已安装 Determinate Nix）

首次激活 nix-darwin 时系统里还没有 `darwin-rebuild`，必须用 `nix run` 引导；之后才能使用 `darwin-rebuild`。`switch` 会写入 `/run/current-system`、`/etc/static/*` 等系统路径，必须加 `sudo`；`build` 只在 nix store 里构建，不需要 sudo。

```bash
# 静态校验（不执行激活，无需 sudo）
nix flake check

# —— 首次激活（全新机器上）——
# build 预检（无需 sudo）：
nix run github:nix-darwin/nix-darwin/master#darwin-rebuild -- \
  build --flake .#AresdeMacBook-Air
# 正式切换（必须 sudo）：
sudo nix run github:nix-darwin/nix-darwin/master#darwin-rebuild -- \
  switch --flake .#AresdeMacBook-Air

# —— 第二次以后 darwin-rebuild 已在 PATH ——
darwin-rebuild build --flake .#AresdeMacBook-Air
sudo darwin-rebuild switch --flake .#AresdeMacBook-Air
```

首次 `nix flake lock` / `nix flake check` 会生成 `flake.lock`。它应当纳入版本控制以保证可重现。
