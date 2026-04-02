# Zsh configuration guidance

## Structure
- `.zshrc` is the main Zsh configuration file, symlinked to `~/.zshrc`.
- Only general, non-sensitive settings belong here (theme, plugins, completions, prompt).
- Machine-specific or private content (API keys, project variables, local paths) must go in `~/.zshrc.local`, which is sourced at the end of `.zshrc` and is NOT tracked by git.

## Editing rules
- Never put secrets, tokens, or machine-specific paths into `.zshrc`.
- When adding new environment variables, decide: public → `.zshrc`; private → remind the user to add it to `~/.zshrc.local`.
- Keep the file concise — avoid large blocks of commented-out boilerplate.
