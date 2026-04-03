# Zsh configuration guidance

## Structure
- `.zshrc` is the main Zsh configuration file, symlinked to `~/.zshrc`.
- Only general, non-sensitive settings belong here (theme, plugins, completions, prompt).
- Machine-specific or private content (API keys, project variables, local paths) must go in `~/.zshrc.local`, which is sourced at the end of `.zshrc` and is NOT tracked by git.

## PATH 优先级
- 自定义 PATH 必须在 `eval "$(brew shellenv)"` **之后**设置，否则会被 homebrew 覆盖。
- 当前顺序：`~/.local/bin`（uv 工具）> `/opt/nanobrew/prefix/bin`（nanobrew 包）> `/opt/homebrew/bin`（homebrew 迁移残留）。
- 使用 `typeset -U PATH`（大写）去重。注意：小写 `typeset -U path` 只对数组操作生效，对 `export PATH=` 字符串赋值无效。

## Python / uv
- Python 版本由 `uv` 统一管理（通过 nanobrew 安装），不再使用 nanobrew 或 Homebrew 安装 Python。
- macOS 系统自带 Python (`/usr/bin/python3`) 保留不动。

## Editing rules
- Never put secrets, tokens, or machine-specific paths into `.zshrc`.
- When adding new environment variables, decide: public → `.zshrc`; private → remind the user to add it to `~/.zshrc.local`.
- Keep the file concise — avoid large blocks of commented-out boilerplate.
