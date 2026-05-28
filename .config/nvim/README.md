# Neovim / LazyVim

This directory contains Yoshua's Neovim configuration based on the official [LazyVim starter](https://github.com/LazyVim/starter).

## Layout

- `init.lua` bootstraps `lua/config/lazy.lua`.
- `lua/config/` stores core LazyVim bootstrap, options, keymaps, and autocmds.
- `lua/plugins/` stores local plugin specs and LazyVim extras imports.
- Runtime data lives in the standard XDG Neovim directories: `~/.local/share/nvim`, `~/.local/state/nvim`, and `~/.cache/nvim`.

## Prerequisites

```bash
brew install neovim tree-sitter-cli
```

Useful optional CLI tools already expected in Yoshua's environment:

```bash
brew install ripgrep fd fzf lazygit
```

## First run / maintenance

```bash
nvim
```

Inside Neovim, common commands are:

- `:Lazy` — plugin manager UI
- `:LazyExtras` — enable or inspect LazyVim extras
- `:checkhealth lazy` — lazy.nvim health
- `:Mason` — language server/tool installer UI
- `:checkhealth` — Neovim health report

Headless verification:

```bash
export PATH="/opt/homebrew/bin:/opt/homebrew/sbin:$PATH"
nvim --headless "+Lazy! sync" +qa
nvim --headless "+checkhealth lazy" +qa
nvim --headless "+checkhealth vim.treesitter" +qa
nvim --headless +qa
git diff --check
```

## Enabled LazyVim Extras

Local extras are declared in `lua/plugins/extras.lua`:

- Languages: TypeScript, JSON, Markdown, Python, Rust, Go, Tailwind CSS
- Tooling: ESLint, Prettier
