# Guidance for agents

## Introduction and Structure
- This directory stores tmux configuration, managed by [oh-my-tmux](https://github.com/gpakosz/.tmux).
- `~/.config/tmux/tmux.conf` 由 `nix/home/tmux.nix` 链接到 `flake.lock` 固定的 oh-my-tmux source；仓库内同名路径不再是事实源。
- `tmux.conf.local` 是用户自定义配置，纳入 Git；Home Manager 以 out-of-store symlink 暴露它。所有自定义只改此文件。
- `~/.config/tmux/plugins` 由 oh-my-tmux 内建 TPM 管理，是本机可变状态，不进入 Git 或 Nix store。
- tmux 运行时（二进制 `/opt/homebrew/bin/tmux`）由 Homebrew 提供，并纳入 `nix/darwin/homebrew.nix` 的声明式清单。不要改写成 Home Manager `programs.tmux.extraConfig`，也不要再手工 clone oh-my-tmux。

## Current Customizations
- Mouse support enabled, Vi mode for status keys and copy mode.
- New windows and panes retain the current working directory.
- Copy mode copies to macOS system clipboard.
- History limit set to 50000 lines.
- Status bar at top; right side shows battery + time + date (no hostname/username since this is a local-only setup).
- Plugins: tmux-resurrect and tmux-continuum for session persistence and auto-restore.

## Workflow
- To customize tmux, edit `tmux.conf.local` only. Never edit `tmux.conf` directly — it is managed by oh-my-tmux upstream.
- After editing, reload with `tmux source ~/.config/tmux/tmux.conf` or prefix + r inside tmux.
- 升级 oh-my-tmux 时运行 `nix flake update oh-my-tmux`，评审 `flake.lock` 后完成 Nix build 与隔离 tmux smoke test；不要在普通同步时隐式跟随 upstream。
