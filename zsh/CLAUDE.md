# Zsh configuration guidance

## Structure
- `.zshrc` is the main Zsh configuration file, symlinked to `~/.zshrc`.
- `shared.zsh` holds the public, cross-machine shell logic shared by both `.zshrc` and `nix/modules/zsh.nix`.
- Only general, non-sensitive settings belong here (theme, plugins, completions, prompt).
- Machine-specific or private content (API keys, project variables, local paths) must go in `~/.zshrc.local`, which is sourced at the end of `.zshrc` and is NOT tracked by git.

## PATH priority
- Custom PATH must be set **after** `eval "$(brew shellenv)"`, otherwise Homebrew will override it.
- Current public shell PATH logic now lives primarily in `shared.zsh`, then gets loaded by both `zsh/.zshrc` and `nix/modules/zsh.nix`.
- Current order: `$HOME/.bun/bin` (bun) > `~/.local/bin` (uv tools) > `/opt/homebrew/bin` (Homebrew).
- Use `typeset -U PATH` (uppercase) for deduplication. Note: lowercase `typeset -U path` only works on array operations, not on `export PATH=` string assignments.

## Bun
- `BUN_INSTALL` and bun completions are currently configured in `shared.zsh`, so they are loaded both by the symlinked `zsh/.zshrc` path and by the Home Manager `nix/modules/zsh.nix` path, and still run before the `~/.zshrc.local` source line.
- bun is installed via its official installer, not via Homebrew.

## Python / uv
- Python versions are managed by `uv` (installed via Homebrew). Do not install Python through Homebrew directly.
- The macOS system Python (`/usr/bin/python3`) is left untouched.

## Environment variables
- `EDITOR=nvim`: sets Neovim as default editor; used by yazi, git, etc. In the current Phase 2D+ layout, the declarative source of truth is primarily `nix/home/shell-env.nix`, while the symlinked `zsh/.zshrc` path still acts as the legacy fallback path.

## Editing rules
- Never put secrets, tokens, or machine-specific paths into `.zshrc` or `shared.zsh`.
- When adding new environment variables, decide: public → `.zshrc` / `shared.zsh`; private → remind the user to add it to `~/.zshrc.local`.
- Keep `.zshrc` concise. Prefer putting reusable shell fragments into `shared.zsh`, but keep Oh My Zsh bootstrap, completion wiring, and other caller-specific boot logic in the caller.
- Keep the file concise — avoid large blocks of commented-out boilerplate.
