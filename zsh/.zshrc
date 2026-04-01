# =============================================================================
# 通用 Zsh 配置（公开，纳入 dotfiles 版本控制）
# 机器相关或隐私内容请写在 ~/.zshrc.local 中
# =============================================================================

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

# Starship 提示符
eval "$(starship init zsh)"

# Homebrew 补全
if type brew &>/dev/null; then
  FPATH=$(brew --prefix)/share/zsh-completions:$FPATH

  autoload -Uz compinit
  compinit
fi

# 加载本地隐私配置（API 密钥、项目变量等）
[[ -f "$HOME/.zshrc.local" ]] && source "$HOME/.zshrc.local"
