{ lib, config, pkgs, ... }:
{
  # =============================================================================
  # Zsh 的首版 Home Manager 模块
  # - 仅涵盖现有 zsh/.zshrc 中的「安全/核心」行为
  # - 保留 ~/.zshrc.local 作为私有配置入口（API key、项目变量等）
  # - 启用此模块后，Home Manager 会生成自己的 ~/.zshrc，不再兼容仓库里的软链接
  # Phase 1 默认不在 nix/home/default.nix 里 import 本模块
  # =============================================================================

  programs.zsh = {
    enable = true;

    # Home Manager 原生实现，等价于现有的 oh-my-zsh 两个插件
    enableCompletion = true;
    autosuggestion.enable = true;
    syntaxHighlighting.enable = true;

    # autosuggestion 颜色，保持与现有主题一致
    autosuggestion.strategy = [ "history" "completion" ];

    # 先于其它初始化：对 PATH 去重
    initExtraFirst = ''
      typeset -U PATH
    '';

    sessionVariables = {
      EDITOR = "nvim";
    };

    # 注意：这里只保留脚本中「非机密、跨机器通用」的片段
    # 任何机密/机器相关内容请继续写入 ~/.zshrc.local
    initExtra = ''
      # Homebrew 环境（与现有 zsh/.zshrc 保持一致的加载顺序）
      if [ -x /opt/homebrew/bin/brew ]; then
        eval "$(/opt/homebrew/bin/brew shellenv)"
      fi

      # uv 工具路径置于 Homebrew 之前
      export PATH="$HOME/.local/bin:$PATH"

      # bun
      export BUN_INSTALL="$HOME/.bun"
      export PATH="$BUN_INSTALL/bin:$PATH"
      [ -s "$BUN_INSTALL/_bun" ] && source "$BUN_INSTALL/_bun"

      # Starship 提示符（可选；未安装时静默跳过）
      if command -v starship >/dev/null 2>&1; then
        eval "$(starship init zsh)"
      fi

      # yazi：退出时可同步切换 shell 当前目录
      # 注：Nix 缩进字符串中 ''' 是 '' 的转义，会渲染成字面量 ''
      function y() {
        local tmp="$(mktemp -t "yazi-cwd.XXXXXX")" cwd
        command yazi "$@" --cwd-file="$tmp"
        IFS= read -r -d ''' cwd < "$tmp"
        [ "$cwd" != "$PWD" ] && [ -d "$cwd" ] && builtin cd -- "$cwd"
        rm -f -- "$tmp"
      }

      # Claude Code 快捷别名
      alias c='CLAUDE_CODE_AUTO_COMPACT_WINDOW=400000 claude --dangerously-skip-permissions'

      # 本地隐私配置（API key、项目变量等）——必须最后 source，以便覆盖前面所有设置
      [[ -f "$HOME/.zshrc.local" ]] && source "$HOME/.zshrc.local"
    '';
  };
}
