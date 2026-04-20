# Guidance for agents

## Introduction and Structure
- This directory stores tmux configuration, managed by [oh-my-tmux](https://github.com/gpakosz/.tmux).
- `tmux.conf` is a symlink to the local oh-my-tmux clone (`~/.local/share/tmux/oh-my-tmux/.tmux.conf`). It is gitignored because the absolute path is machine-specific. The `setup_mac.sh` script creates it automatically.
- `tmux.conf.local` is the user's personal configuration file, tracked by git. All customizations should go here.
- tmux 运行时（二进制 `/opt/homebrew/bin/tmux`）由 Homebrew 提供，并已在 Phase 3B 纳入 `nix/darwin/homebrew.nix` 的声明式清单。切勿把 `tmux.conf.local` 改写成 Home Manager `programs.tmux.extraConfig`，也不要替换 oh-my-tmux —— 声明式只覆盖运行时，配置事实源仍是本目录。

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
