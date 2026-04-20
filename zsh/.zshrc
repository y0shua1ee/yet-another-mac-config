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

# 默认编辑器（需先安装 neovim：brew install neovim）
command -v nvim >/dev/null 2>&1 && export EDITOR=nvim

# 与 Home Manager 共享的公开、跨机器通用逻辑
source "$(dirname "$(realpath "$HOME/.zshrc")")/shared.zsh"

# Homebrew 补全
if type brew &>/dev/null; then
  FPATH=$(brew --prefix)/share/zsh-completions:$FPATH

  autoload -Uz compinit
  compinit
fi

# 加载本地隐私配置（API 密钥、项目变量等）——必须放在最后，以便覆盖上方任何设置
[[ -f "$HOME/.zshrc.local" ]] && source "$HOME/.zshrc.local"
