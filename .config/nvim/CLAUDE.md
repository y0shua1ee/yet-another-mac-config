# Neovim / AstroNvim 配置指南（面向 agents）

## Introduction and Structure
- This folder contains Yoshua's Neovim configuration, based on the official AstroNvim template.
- `init.lua` bootstraps `lazy.nvim`, then loads `lua/lazy_setup.lua` and `lua/polish.lua`.
- `lua/community.lua` stores AstroCommunity imports.
- `lua/plugins/` stores local AstroNvim plugin specs such as `astrocore.lua`, `astrolsp.lua`, `astroui.lua`, `mason.lua`, `none-ls.lua`, `treesitter.lua`, and `user.lua`.
- `README.md` records the local setup and recovery notes for this configuration.

## Workflow
- Consult the AstroNvim documentation and the relevant plugin documentation before changing plugin specs.
- Prefer small, targeted changes under `lua/plugins/`, `lua/community.lua`, or `lua/polish.lua`.
- Keep machine-specific state, caches, sessions, and plugin downloads out of this folder and out of Git. Runtime state belongs in `~/.local/share/nvim`, `~/.local/state/nvim`, and `~/.cache/nvim`.
- When adding a dependency that must exist outside Mason, update the root README, `nix/darwin/homebrew.nix`, and `nix/README.md` as needed.
- After config changes, run a headless Neovim check and repo checks before committing.

## Verification

```bash
export PATH="/opt/homebrew/bin:/opt/homebrew/sbin:$PATH"
nvim --headless "+Lazy! sync" +qa
nvim --headless "+checkhealth vim.treesitter" +qa
nvim --headless +qa
git diff --check
```

## Enabled AstroCommunity Packs (lua/community.lua)
- **Languages:** lua, typescript-all-in-one, json, markdown, python, rust, go, bash
- **Web/formatting:** tailwindcss, eslint, prettier

## Migration Notes
- The previous LazyVim starter config was replaced by the AstroNvim template.
- Before replacement, the old config and runtime folders were backed up under `~/.hermes/backups/nvim-astronvim-*`.
