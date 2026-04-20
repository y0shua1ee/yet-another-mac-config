{ lib, config, pkgs, ... }:
{
  # =============================================================================
  # Zsh 的首版 Home Manager 模块
  # - 与仓库中的 zsh/.zshrc 共享同一份 ../../zsh/shared.zsh，避免双份漂移
  # - 仅涵盖「安全/核心、跨机器通用」行为
  # - 保留 ~/.zshrc.local 作为私有配置入口（API key、项目变量等）
  # - 启用此模块后，Home Manager 会生成自己的 ~/.zshrc，不再兼容仓库里的软链接
  # Phase 1 / 2A 默认不在 nix/home/default.nix 里 import 本模块
  # =============================================================================

  programs.zsh = {
    enable = true;

    # Home Manager 原生实现，等价于现有 oh-my-zsh 两个插件 + compinit 流程
    enableCompletion = true;
    autosuggestion.enable = true;
    syntaxHighlighting.enable = true;

    # autosuggestion 建议策略，保持与常见 oh-my-zsh 默认一致
    autosuggestion.strategy = [ "history" "completion" ];

    # 先于其它初始化：对 PATH 去重
    initExtraFirst = ''
      typeset -U PATH
    '';

    # 共享核心逻辑来自 ../../zsh/shared.zsh；
    # 本地隐私配置仍在这里额外 source，保持与当前 ~/.zshrc 的约定一致。
    # 注意：
    #   - 机器相关片段（如 OpenClaw 的绝对路径 completion）不在此处展开
    #   - EDITOR / VISUAL / PAGER 由 nix/home/shell-env.nix 统一声明，这里不再重复
    initExtra = (builtins.readFile ../../zsh/shared.zsh) + ''

      [[ -f "$HOME/.zshrc.local" ]] && source "$HOME/.zshrc.local"
    '';
  };
}
