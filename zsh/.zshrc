# =============================================================================
# 通用 Zsh 配置（公开，纳入 dotfiles 版本控制）
# 机器相关或隐私内容请写在 ~/.zshrc.local 中
# =============================================================================

# PATH 去重：对 export PATH= 字符串赋值生效，保留首次出现、移除重复
typeset -U PATH

# Oh My Zsh 基础路径
export ZSH="$HOME/.oh-my-zsh"

# 主题
ZSH_THEME="robbyrussell"

# 插件
plugins=(git zsh-syntax-highlighting zsh-autosuggestions)

source $ZSH/oh-my-zsh.sh

# 自动建议颜色（适配 Catppuccin Mocha + 半透明背景，提高可读性）
ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE='fg=#7f849c'

# Homebrew 环境
eval "$(/opt/homebrew/bin/brew shellenv)"
# nanobrew 和 uv 路径置于 Homebrew 之前，确保升级后的包和 uv 工具优先生效
export PATH="$HOME/.local/bin:/opt/nanobrew/prefix/bin:$PATH"

# 默认编辑器（需先安装 neovim：nb install neovim）
command -v nvim >/dev/null 2>&1 && export EDITOR=nvim

# Starship 提示符
eval "$(starship init zsh)"

# Homebrew 补全
if type brew &>/dev/null; then
  FPATH=$(brew --prefix)/share/zsh-completions:$FPATH

  autoload -Uz compinit
  compinit
fi

# nanobrew 补全（使用 nb 内置的完整补全脚本，不依赖 brew）
if type nb &>/dev/null; then
  eval "$(nb completions zsh)"
fi

# yazi
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

# 加载本地隐私配置（API 密钥、项目变量等）——必须放在最后，以便覆盖上方任何设置
[[ -f "$HOME/.zshrc.local" ]] && source "$HOME/.zshrc.local"

