# =============================================================================
# 通用 Zsh 配置（公开，纳入 dotfiles 版本控制）
# 机器相关或隐私内容请写在 ~/.zshrc.local 中
# =============================================================================

# PATH / fpath 去重：保留首次出现，移除重复项
typeset -U PATH path fpath

# 备用软链接路线的补全初始化；Home Manager 路线由 nix/modules/zsh.nix 接管。
if [ -x /opt/homebrew/bin/brew ]; then
  _brew_prefix="/opt/homebrew"
elif [ -x /usr/local/bin/brew ]; then
  _brew_prefix="/usr/local"
else
  _brew_prefix=""
fi

if [ -n "$_brew_prefix" ]; then
  fpath=("$_brew_prefix/share/zsh/site-functions" "$_brew_prefix/share/zsh-completions" $fpath)
fi
unset _brew_prefix

autoload -Uz compinit && compinit

# 与 Home Manager 共享的公开、跨机器通用逻辑
_zshrc_source="${(%):-%N}"
_zshrc_dir="${_zshrc_source:A:h}"
source "$_zshrc_dir/shared.zsh"
unset _zshrc_source _zshrc_dir

# zsh-autosuggestions：备用路线优先读取 Homebrew 安装，当前主路线由 Home Manager 提供。
for _autosuggest_file in \
  /opt/homebrew/share/zsh-autosuggestions/zsh-autosuggestions.zsh \
  /usr/local/share/zsh-autosuggestions/zsh-autosuggestions.zsh; do
  if [ -r "$_autosuggest_file" ]; then
    source "$_autosuggest_file"
    break
  fi
done
unset _autosuggest_file

# 默认编辑器（声明式路线由 nix/home/shell-env.nix 提供，这里作为备用路线兜底）
command -v nvim >/dev/null 2>&1 && export EDITOR=nvim

# 加载本地隐私配置（API 密钥、项目变量等）——必须放在最后，以便覆盖上方任何设置
[[ -f "$HOME/.zshrc.local" ]] && source "$HOME/.zshrc.local"

# zsh-syntax-highlighting 需要尽量靠后加载，确保覆盖前面定义的 widgets。
for _syntax_file in \
  /opt/homebrew/share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh \
  /usr/local/share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh; do
  if [ -r "$_syntax_file" ]; then
    source "$_syntax_file"
    break
  fi
done
unset _syntax_file
