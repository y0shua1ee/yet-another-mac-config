# Zsh configuration guidance

## Structure
- `.zshrc` is the main Zsh configuration file, symlinked to `~/.zshrc`.
- Only general, non-sensitive settings belong here (theme, plugins, completions, prompt).
- Machine-specific or private content (API keys, project variables, local paths) must go in `~/.zshrc.local`, which is sourced at the end of `.zshrc` and is NOT tracked by git.

## PATH priority
- Custom PATH must be set **after** `eval "$(brew shellenv)"`, otherwise Homebrew will override it.
- Current order: `~/.local/bin` (uv tools) > `/opt/nanobrew/prefix/bin` (nanobrew packages) > `/opt/homebrew/bin` (Homebrew migration leftovers).
- Use `typeset -U PATH` (uppercase) for deduplication. Note: lowercase `typeset -U path` only works on array operations, not on `export PATH=` string assignments.

## Python / uv
- Python versions are managed by `uv` (installed via nanobrew). Do not install Python through nanobrew or Homebrew.
- The macOS system Python (`/usr/bin/python3`) is left untouched.

## Environment variables
- `EDITOR=nvim`: sets Neovim as default editor; used by yazi, git, etc. Requires `nb install neovim`; if nvim is not installed, the line is a no-op and tools fall back to their own defaults.

## Editing rules
- Never put secrets, tokens, or machine-specific paths into `.zshrc`.
- When adding new environment variables, decide: public → `.zshrc`; private → remind the user to add it to `~/.zshrc.local`.
- Keep the file concise — avoid large blocks of commented-out boilerplate.
