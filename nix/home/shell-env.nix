{ ... }:
{
  # =============================================================================
  # 通用 shell 环境变量（Phase 2A）
  # - 只放「非私密、跨机器通用」的变量
  # - 私密/机器相关变量仍然写入 ~/.zshrc.local（不在版本控制里）
  #
  # 关于何时生效：
  # - home.sessionVariables 会被写入 ~/.nix-profile/etc/profile.d/hm-session-vars.sh
  # - 该文件只有在 Home Manager 生成的 shell 配置里才会被 source
  # - 当前机器已完成 Home Manager 版 zsh 的 switch，所以下列变量已经在登录 shell 中
  #   生效；仓库 zsh/.zshrc 中保留的 `export EDITOR=nvim` 只作为旧软链接回退路径。
  # =============================================================================

  home.sessionVariables = {
    EDITOR = "nvim";
    VISUAL = "nvim";
    PAGER  = "less";
  };
}
