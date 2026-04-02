# Guidance for agents

## Introduction and Structure
- This directory stores tmux configuration, managed by [oh-my-tmux](https://github.com/gpakosz/.tmux).
- `tmux.conf` is a symlink to the local oh-my-tmux clone (`~/.local/share/tmux/oh-my-tmux/.tmux.conf`). It is gitignored because the absolute path is machine-specific. The `setup_mac.sh` script creates it automatically.
- `tmux.conf.local` is the user's personal configuration file, tracked by git. All customizations should go here.

## Workflow
- To customize tmux, edit `tmux.conf.local` only. Never edit `tmux.conf` directly — it is managed by oh-my-tmux upstream.
- After editing, reload with `tmux source ~/.config/tmux/tmux.conf` or prefix + r inside tmux.
