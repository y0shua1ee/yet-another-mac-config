# Nix 配置指南（面向 agents）

## 目的 / Phase 1 范围
- 这是渐进式 Nix 迁移的第一阶段骨架，底层运行时是 [Determinate Nix](https://docs.determinate.systems/)，系统层用 [nix-darwin](https://github.com/nix-darwin/nix-darwin)，用户层用 [Home Manager](https://nix-community.github.io/home-manager/)。
- 此阶段故意保守：**不接管任何现有 dotfile**，不触碰 Homebrew、系统默认值、服务等。
- 这个仓库仍然是「事实源」，Nix 只是又一种可选的激活方式。

## 目录结构
```
nix/
├── CLAUDE.md          # 本文件
├── AGENTS.md          # 软链接 -> CLAUDE.md（二者内容保持一致）
├── darwin/
│   └── default.nix    # nix-darwin 系统层（最小）
├── home/
│   └── default.nix    # Home Manager 用户层入口（默认不启用 zsh 模块）
└── modules/
    └── zsh.nix        # 首版 zsh Home Manager 模块（安全/核心子集）
```

根目录的 `flake.nix` 通过 `darwinConfigurations.AresdeMacBook-Air` 把上述三层装配起来。

## 重要约束
- `darwin/default.nix` 里 `nix.enable = false;` —— Determinate Nix 自己管理 nix 守护进程，nix-darwin **不得**再接管，否则会互相覆盖。
- `home/default.nix` 默认**不 import** `modules/zsh.nix`。Phase 1 禁止直接启用 zsh 模块，否则 Home Manager 会改写 `~/.zshrc`，与仓库中 `zsh/.zshrc → ~/.zshrc` 软链接冲突。
- 启用 zsh 模块前必须先：
  1. 移除 `~/.zshrc` 软链接（或依赖 `home-manager.backupFileExtension = "hm-backup"` 自动备份）。
  2. 在 `nix/home/default.nix` 的 `imports` 中取消注释 `../modules/zsh.nix`。
  3. 先 `darwin-rebuild build --flake .#...` 检查再 `switch`。

## 修改风格
- 保持最小改动。任何向系统层下沉（如启用 `homebrew`、`system.defaults`）的变更都应单独成一个 commit，并更新根 README 的「渐进式 Nix 迁移」章节。
- 新增 Home Manager 模块放在 `modules/`，并在 `home/default.nix` 中显式 import。
- Nix 代码中的注释使用中文（与仓库整体风格一致）。

## 常用命令（需已安装 Determinate Nix）

首次激活 nix-darwin 时系统里还没有 `darwin-rebuild`，必须用 `nix run` 引导；之后才能使用 `darwin-rebuild`。

```bash
# 静态校验（不执行激活）
nix flake check

# —— 首次激活（全新机器上）——
# build 预检：
nix run github:nix-darwin/nix-darwin/master#darwin-rebuild -- \
  build --flake .#AresdeMacBook-Air
# 正式切换：
nix run github:nix-darwin/nix-darwin/master#darwin-rebuild -- \
  switch --flake .#AresdeMacBook-Air

# —— 第二次以后 darwin-rebuild 已在 PATH ——
darwin-rebuild build  --flake .#AresdeMacBook-Air
darwin-rebuild switch --flake .#AresdeMacBook-Air
```

首次 `nix flake lock` / `nix flake check` 会生成 `flake.lock`。它应当纳入版本控制以保证可重现。
