# Nix 配置指南（面向 agents）

## 目的 / 当前阶段
- 这是渐进式 Nix 迁移，底层运行时是 [Determinate Nix](https://docs.determinate.systems/)，系统层用 [nix-darwin](https://github.com/nix-darwin/nix-darwin)，用户层用 [Home Manager](https://nix-community.github.io/home-manager/)。
- **Phase 2D 现状：Home Manager 已接管 `~/.zshrc`。`nix flake check`、`darwin-rebuild build --flake .#AresdeMacBook-Air` 与 `sudo darwin-rebuild switch --flake .#AresdeMacBook-Air` 都已在当前机器跑通。**
- **Phase 3A 现状：已引入保守的 Homebrew 声明式清单（`nix/darwin/homebrew.nix`）。`nix flake check` 与 `darwin-rebuild build --flake .#AresdeMacBook-Air` 均通过；`switch` 需由人工执行。onActivation 全部设为 `false` / `"none"`，不自动 update / upgrade / cleanup，也不会卸载任何本机已有的 brew 包。**
- **Phase 3 计划文档：`nix/phase-3-plan.md`。默认顺序是：先做 Homebrew 清单声明化，再做 tmux 运行时声明化，最后只试点少量稳定 `system.defaults.*`。**
- 这个仓库仍然是「事实源」，Nix 只是又一种可选的激活方式。当前机器上的旧仓库软链接版 zsh 仍保留在 `~/.zshrc.pre-hm-switch-backup`，便于人工回退。

## 目录结构
```
nix/
├── CLAUDE.md          # 本文件
├── AGENTS.md          # 软链接 -> CLAUDE.md（二者内容保持一致）
├── darwin/
│   ├── default.nix    # nix-darwin 系统层入口
│   └── homebrew.nix   # Phase 3A：保守的 Homebrew 声明式清单
├── home/
│   ├── default.nix    # Home Manager 用户层入口（已 import ../modules/zsh.nix 并实际生效）
│   ├── packages.nix   # Phase 2A：低风险纯 CLI 工具
│   └── shell-env.nix  # Phase 2A：通用非私密环境变量（当前机器已随 Home Manager zsh 生效）
└── modules/
    └── zsh.nix        # zsh Home Manager 模块（当前机器已 takeover 生效）
```

另有仓库侧共享脚本：`zsh/shared.zsh`。它被 `zsh/.zshrc` 与 `nix/modules/zsh.nix` 共同复用，用来承载公开、跨机器通用的 shell 逻辑。

根目录的 `flake.nix` 通过 `darwinConfigurations.AresdeMacBook-Air` 把上述三层装配起来，并开启 `home-manager.backupFileExtension = "hm-backup"`，用于 switch 时备份会冲突的 home-manager 目标文件。当前机器上 `~/.zshenv` 已自动备份为 `~/.zshenv.hm-backup`；`~/.zshrc` 则因本地软链接冲突，实际采用了手动先挪到 `~/.zshrc.pre-hm-switch-backup` 再 switch 的方式完成 takeover。

## 重要约束
- `darwin/default.nix` 里 `nix.enable = false;` —— Determinate Nix 自己管理 nix 守护进程，nix-darwin **不得**再接管，否则会互相覆盖。
- `home/default.nix` 现在**确实 import** 了 `modules/zsh.nix`，而且本机 switch 已经完成，所以 Home Manager 正在直接生成并管理 `~/.zshrc`。
- 当前机器的注意事项：
  1. 以后继续迭代 zsh 相关 Nix 变更时，先 `nix flake check` 与 `darwin-rebuild build --flake .#AresdeMacBook-Air`，再 `sudo darwin-rebuild switch`。
  2. 旧仓库软链接版 zsh 目前保留在 `~/.zshrc.pre-hm-switch-backup`；若想回到旧路径，可人工还原。
  3. 机器相关或绝对路径的 shell 片段（例如 OpenClaw completion）不应进仓库共享区，而应写入 `~/.zshrc.local`；Home Manager 版 zsh 的 `initExtra` 会在末尾自动 `source` 它。
  4. 需要回滚时：`sudo darwin-rebuild switch --rollback`，并按需把 `~/.zshrc.pre-hm-switch-backup` 还原为 `~/.zshrc`。
- 当前阶段仍**不** 触碰：`~/.zshrc.local`、`brew services`、字体、`.hammerspoon`、secrets / 登录态，以及大范围 `system.defaults.*` 迁移。这些继续按原方式管理。
- Phase 3A 的 Homebrew 模块是“保守首版”：只纳入长期稳定、已在日常使用的少量 formula / cask，未声明的本机 brew 包不会被自动卸载；要追加新条目时，按 `nix/darwin/homebrew.nix` 里的分类说明（服务类 / 字体 / 账号态工具暂不纳入）追加即可，不要开启 `cleanup = "check"` 或 `autoUpdate / upgrade`。
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
