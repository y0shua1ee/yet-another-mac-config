# AstroNvim

This directory stores Yoshua's Neovim configuration based on the official [AstroNvim template](https://github.com/AstroNvim/template).

## 本机依赖

- Neovim 0.11+（当前由 Homebrew `neovim` 提供）
- `tree-sitter-cli`（AstroNvim / Treesitter 运行时依赖，当前由 Homebrew 提供）
- Nerd Font（Ghostty 当前字体使用 Maple Mono NF）
- 可选但常用：`ripgrep`、`lazygit`、Python、Node、C compiler

## 常用命令

```bash
nvim
```

在 Neovim 内：

```vim
:Lazy
:AstroUpdate
:LspInstall <server>
:TSInstall <language>
:DapInstall <debugger>
```

## 维护约定

- 本目录只保存可版本化配置。
- 插件下载、Mason 包、会话、缓存等运行时状态放在 `~/.local/share/nvim`、`~/.local/state/nvim` 和 `~/.cache/nvim`。
- 本配置来自 AstroNvim template；本地扩展优先放在 `lua/plugins/`、`lua/community.lua` 或 `lua/polish.lua`。
