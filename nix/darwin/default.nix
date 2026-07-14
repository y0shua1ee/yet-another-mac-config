{ system, username, ... }:
{
  # =============================================================================
  # nix-darwin 系统层配置
  #   - Phase 1：骨架
  #   - Phase 3A：引入保守的 Homebrew 声明式清单（见 ./homebrew.nix）
  #   - Phase 3C：引入少量稳定的 system.defaults 试点（见 ./defaults.nix）
  # =============================================================================

  imports = [
    ./homebrew.nix
    ./defaults.nix
  ];

  # 目标平台来自 nix/hosts 中的主机 profile，允许多台 Apple Silicon Mac
  # 共享模块；Intel 机器需在 Determinate 官方支持与实机验证后再加入。
  nixpkgs.hostPlatform = system;
  nixpkgs.config.allowUnfree = true;

  # 用户基础信息；实际 shell、home 配置由 Home Manager 接管
  users.users.${username} = {
    name = username;
    home = "/Users/${username}";
  };

  # 近期 nix-darwin 要求声明主用户
  system.primaryUser = username;

  # nix-darwin state version。升级前请阅读 release notes
  system.stateVersion = 5;

  # Homebrew 清单、两个低风险服务试点与少量稳定 system.defaults 已分别
  # 在子模块中声明。账号态、TCC 权限和大范围 app state 仍保留为本机状态。
}
