# Neovim / LazyVim 配置指南（面向 agents）

## Introduction and Structure

- This folder contains Yoshua's Neovim configuration, based on the official LazyVim starter.
- `init.lua` bootstraps `lazy.nvim`, LazyVim, and local plugin specs through `lua/config/lazy.lua`.
- `lua/config/` stores core LazyVim bootstrap, options, keymaps, and autocmds.
- `lua/plugins/` stores local plugin specs. LazyVim extras are imported in `lua/config/lazy.lua` before local specs.
- `README.md` records setup, maintenance, and verification for this configuration.

## Workflow

- Consult the LazyVim documentation and relevant plugin documentation before changing plugin specs.
- Prefer small, targeted changes under `lua/plugins/`, `lua/config/options.lua`, `lua/config/keymaps.lua`, or `lua/config/autocmds.lua`.
- Keep machine-specific state, caches, sessions, and plugin downloads out of this folder and out of Git. Runtime state belongs in `~/.local/share/nvim`, `~/.local/state/nvim`, and `~/.cache/nvim`.
- When adding a dependency that must exist outside Mason, update the root README, `nix/darwin/homebrew.nix`, and `nix/README.md` as needed.
- After config changes, run a headless Neovim check and repo checks before committing.

## Verification

```bash
export PATH="/opt/homebrew/bin:/opt/homebrew/sbin:$PATH"
nvim --headless "+Lazy! sync" +qa
nvim --headless "+checkhealth lazy" +qa
nvim --headless "+checkhealth vim.treesitter" +qa
nvim --headless +qa
git diff --check
```

## Enabled LazyVim Extras (`lua/config/lazy.lua`)

- **Languages:** TypeScript, JSON, Markdown, Python, Rust, Go, Tailwind CSS
- **Tooling:** ESLint, Prettier
