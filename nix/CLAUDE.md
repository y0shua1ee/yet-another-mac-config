# Nix 配置指南（面向 agents）

## 目的 / 当前阶段
- 这是渐进式 Nix 迁移，底层运行时是 [Determinate Nix](https://docs.determinate.systems/)，系统层用 [nix-darwin](https://github.com/nix-darwin/nix-darwin)，用户层用 [Home Manager](https://nix-community.github.io/home-manager/)。
- **Phase 2C 现状：zsh 模块已纳入 Home Manager 配置图，`nix flake check` 与 `darwin-rebuild build --flake .#AresdeMacBook-Air` 均通过；真正接管 `~/.zshrc` 仍要等下一次 `sudo darwin-rebuild switch`。**
- 这个仓库仍然是「事实源」，Nix 只是又一种可选的激活方式。在 switch 发生之前（或 switch 之后 rollback），`~/.zshrc` 仍由仓库 `zsh/.zshrc` 软链接承担。

## 目录结构
```
nix/
├── CLAUDE.md          # 本文件
├── AGENTS.md          # 软链接 -> CLAUDE.md（二者内容保持一致）
├── darwin/
│   └── default.nix    # nix-darwin 系统层（最小）
├── home/
│   ├── default.nix    # Home Manager 用户层入口（已 import ../modules/zsh.nix）
│   ├── packages.nix   # Phase 2A：低风险纯 CLI 工具
│   └── shell-env.nix  # Phase 2A：通用非私密环境变量（switch 后 zsh 模块会令其生效）
└── modules/
    └── zsh.nix        # zsh Home Manager 模块（Phase 2C 已 ready，待 switch 生效）
```

另有仓库侧共享脚本：`zsh/shared.zsh`。它被 `zsh/.zshrc` 与 `nix/modules/zsh.nix` 共同复用，用来承载公开、跨机器通用的 shell 逻辑。

根目录的 `flake.nix` 通过 `darwinConfigurations.AresdeMacBook-Air` 把上述三层装配起来，并开启 `home-manager.backupFileExtension = "hm-backup"`，用于首次 switch 时把已有的 `~/.zshrc` 自动备份为 `~/.zshrc.hm-backup`，避免覆盖手写内容。

## 重要约束
- `darwin/default.nix` 里 `nix.enable = false;` —— Determinate Nix 自己管理 nix 守护进程，nix-darwin **不得**再接管，否则会互相覆盖。
- `home/default.nix` 现在**确实 import** 了 `modules/zsh.nix`。也就是说，下次执行 `sudo darwin-rebuild switch` 时，Home Manager 会直接生成自己的 `~/.zshrc`，从而接管这份 dotfile。在 switch 之前保持日常使用即可，仓库里的 `zsh/.zshrc` 软链接不会被自动触碰。
- 触发 switch 前后的注意事项：
  1. 先 `nix flake check` 与 `darwin-rebuild build --flake .#AresdeMacBook-Air`，再 `sudo darwin-rebuild switch`。
  2. 首次 switch 时，`~/.zshrc` 软链接会被 `home-manager.backupFileExtension` 重命名为 `~/.zshrc.hm-backup`；新 `~/.zshrc` 由 `modules/zsh.nix` 基于 `../../zsh/shared.zsh` 生成。
  3. 机器相关或绝对路径的 shell 片段（例如仓库里 `zsh/.zshrc` 末尾追加的 OpenClaw completion）不应进仓库共享区，而应写入 `~/.zshrc.local`；Home Manager 版 zsh 的 `initExtra` 会在末尾自动 `source` 它。
  4. 需要回滚时：`sudo darwin-rebuild switch --rollback`，并按需把 `~/.zshrc.hm-backup` 还原为 `~/.zshrc`。
- 当前阶段 **不** 触碰：`~/.zshrc`（takeover 动作本身）、`~/.zshrc.local`、Homebrew casks / 服务、`system.defaults.*`、字体、`.hammerspoon`。这些仍按原方式管理。

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
