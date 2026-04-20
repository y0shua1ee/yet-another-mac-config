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
  # - Phase 2C：`../modules/zsh.nix` 已 import 进 home/default.nix，配置图 ready，
  #   但真正接管 ~/.zshrc 仍需一次 `sudo darwin-rebuild switch`。switch 之前
  #   这些变量不会在登录 shell 中生效，仍由 zsh/.zshrc 里的 `export EDITOR=nvim`
  #   承担运行时职责；switch 之后 Home Manager 版 zsh 会自动接管。
  # =============================================================================

  home.sessionVariables = {
    EDITOR = "nvim";
    VISUAL = "nvim";
    PAGER  = "less";
  };
}
