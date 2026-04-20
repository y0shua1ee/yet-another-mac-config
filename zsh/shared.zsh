# =============================================================================
# 跨实现共享的 Zsh 片段
# 同时被 zsh/.zshrc（软链接版）与 nix/modules/zsh.nix（Home Manager 版）复用，
# 目的是让两份配置不再漂移。
#
# 取舍原则：
# - 只放「公开、非私密、跨机器通用」的逻辑
# - 不在这里启用 Oh My Zsh 或 Home Manager 的补全/插件系统（两条路径机制不同，
#   分别由各自的调用方用各自的原生方式完成）
# - 不放 `typeset -U PATH`：它必须在一切 PATH 修改之前执行，由调用方在更靠前
#   的位置保证（zshrc 放文件最上方，Home Manager 放 initContent + lib.mkBefore）
# - 不放 EDITOR / VISUAL / PAGER：运行时与声明式来源目前分阶段处理，避免与
#   nix/home/shell-env.nix 重复漂移
# - 机器相关 / 私密内容仍然写入 ~/.zshrc.local 或由调用方在本文件之后追加
# =============================================================================

# 自动建议颜色（适配 Catppuccin Mocha + 半透明背景，提高可读性）
# 同时适用于 oh-my-zsh 的 zsh-autosuggestions 插件与 Home Manager 原生 autosuggestion
ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE='fg=#7f849c'

# Homebrew 环境（加存在性守卫，方便未装 Homebrew 的最小环境静默跳过）
if [ -x /opt/homebrew/bin/brew ]; then
  eval "$(/opt/homebrew/bin/brew shellenv)"
fi

# uv 工具路径置于 Homebrew 之前
export PATH="$HOME/.local/bin:$PATH"

# Starship 提示符（未安装时静默跳过）
if command -v starship >/dev/null 2>&1; then
  eval "$(starship init zsh)"
fi

# yazi：退出时可同步切换 shell 当前目录
function y() {
  local tmp="$(mktemp -t "yazi-cwd.XXXXXX")" cwd
  command yazi "$@" --cwd-file="$tmp"
  IFS= read -r -d '' cwd < "$tmp"
  [ "$cwd" != "$PWD" ] && [ -d "$cwd" ] && builtin cd -- "$cwd"
  rm -f -- "$tmp"
}

# bun
export BUN_INSTALL="$HOME/.bun"
export PATH="$BUN_INSTALL/bin:$PATH"
[ -s "$BUN_INSTALL/_bun" ] && source "$BUN_INSTALL/_bun"

# Claude Code 快捷别名
alias c='CLAUDE_CODE_AUTO_COMPACT_WINDOW=400000 claude --dangerously-skip-permissions'
