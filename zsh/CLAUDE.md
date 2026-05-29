# Zsh configuration guidance

## Structure
- Current host state: Home Manager manages `~/.zshrc` through `nix/modules/zsh.nix`.
- `zsh/.zshrc` is the fallback entry for machines that use this repo through a direct symlink.
- `shared.zsh` holds public, cross-machine shell logic shared by the fallback entry and the Home Manager module.
- General, non-sensitive settings belong here: PATH logic, aliases, prompt setup, helper functions, and guarded plugin wiring.
- Machine-specific or private content belongs in `~/.zshrc.local`, sourced near the end of both zsh routes and left out of git.

## PATH priority
- Custom PATH additions should run after `eval "$(brew shellenv)"`, because Homebrew may rewrite PATH.
- Current public shell PATH logic lives primarily in `shared.zsh`, then gets loaded by both `zsh/.zshrc` and `nix/modules/zsh.nix`.
- Current order: `$HOME/.bun/bin` (bun) > `~/.local/bin` (uv tools) > `/opt/homebrew/bin` (Homebrew).
- Use `typeset -U PATH` for string-style PATH deduplication and `typeset -U path fpath` for array deduplication.

## Plugin model
- The active route uses Home Manager native support for completion, autosuggestions, and syntax highlighting.
- The fallback `zsh/.zshrc` uses guarded Homebrew plugin files when present.
- Keep the fallback route free of framework bootstraps; avoid adding shell frameworks or plugin managers.
- Load `zsh-syntax-highlighting` near the end of the fallback file so widget hooks see earlier shell setup.

## Bun
- `BUN_INSTALL` and bun completions are configured in `shared.zsh`, so both zsh routes load them before local private overrides.
- bun is installed via its official installer.

## Python / uv
- Python versions are managed by `uv`.
- The macOS system Python (`/usr/bin/python3`) is left untouched.

## Environment variables
- `EDITOR=nvim` is declared through `nix/home/shell-env.nix` on the Home Manager route.
- The fallback `zsh/.zshrc` keeps an `EDITOR=nvim` guard as a compatibility fallback.

## Editing rules
- Keep secrets, tokens, and machine-specific paths in `~/.zshrc.local`.
- Add public shell logic to `shared.zsh` when both routes need it.
- Add route-specific boot logic to `zsh/.zshrc` or `nix/modules/zsh.nix`.
- Keep files concise; avoid large blocks of commented-out boilerplate.
