{ lib, config, pkgs, ... }:
{
  # =============================================================================
  # Zsh 的首版 Home Manager 模块
  # - 与仓库中的 zsh/.zshrc 共享同一份 ../../zsh/shared.zsh，避免双份漂移
  # - 仅涵盖「安全/核心、跨机器通用」行为
  # - 保留 ~/.zshrc.local 作为私有配置入口（API key、项目变量等）
  # - 启用此模块后，Home Manager 会生成自己的 ~/.zshrc，不再兼容仓库里的软链接
  # - 当前仓库在 Phase 2D 以后已由 nix/home/default.nix 正式 import 本模块并完成 switch
  # =============================================================================

  programs.zsh = {
    enable = true;

    # Home Manager 原生实现：补全、自动建议、语法高亮
    enableCompletion = true;
    autosuggestion.enable = true;
    syntaxHighlighting.enable = true;

    # autosuggestion 建议策略：同时使用历史记录与补全候选
    autosuggestion.strategy = [ "history" "completion" ];

    # 使用 initContent + mkMerge / mkBefore，避免继续依赖已废弃的 initExtra* 选项。
    initContent = lib.mkMerge [
      # 先于其它初始化：对 PATH 去重
      (lib.mkBefore ''
        typeset -U PATH
      '')

      # 共享核心逻辑来自 ../../zsh/shared.zsh；
      # 本地隐私配置仍在这里额外 source，保持与当前 ~/.zshrc 的约定一致。
      # 注意：
      #   - 机器相关片段（如 OpenClaw 的绝对路径 completion）不在此处展开
      #   - EDITOR / VISUAL / PAGER 由 nix/home/shell-env.nix 统一声明，这里不再重复
      ((builtins.readFile ../../zsh/shared.zsh) + ''

        [[ -f "$HOME/.zshrc.local" ]] && source "$HOME/.zshrc.local"

        # Phase 5B+：启用 mise 的 zsh 激活
        # - 故意放在 ~/.zshrc.local 之后：本机私有 PATH / token / completion 先加载，
        #   mise 再根据 .config/mise/config.toml 提供全局 Node / Go fallback。
        # - command -v 守卫：mise 由 Home Manager 提供，但保留守卫以便最小环境
        #   或回滚场景下静默跳过。
        if command -v mise >/dev/null 2>&1; then
          eval "$(mise activate zsh)"
        fi
      '')
    ];
  };
}
