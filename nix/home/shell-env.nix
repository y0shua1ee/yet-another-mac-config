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
  # - Phase 2A 还没启用 programs.zsh，所以这些变量**当前还不会在登录 shell 中真正生效**
  # - 目的是先把「事实源」搬到 Nix，等 Phase 2B 打开 zsh 模块时它们会自动接管
  # - 这段期间，zsh/.zshrc 中已有的 `export EDITOR=nvim` 继续承担运行时职责，避免行为回退
  # =============================================================================

  home.sessionVariables = {
    EDITOR = "nvim";
    VISUAL = "nvim";
    PAGER  = "less";
  };
}
