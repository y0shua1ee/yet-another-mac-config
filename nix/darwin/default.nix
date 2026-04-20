{ pkgs, lib, username, ... }:
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

  # 目标平台
  nixpkgs.hostPlatform = "aarch64-darwin";
  nixpkgs.config.allowUnfree = true;

  # Determinate Nix 自行管理 nix 守护进程与安装，禁止 nix-darwin 再插手
  # 参考：https://determinate.systems/posts/nix-darwin-on-determinate/
  nix.enable = false;

  # 用户基础信息；实际 shell、home 配置由 Home Manager 接管
  users.users.${username} = {
    name = username;
    home = "/Users/${username}";
  };

  # 近期 nix-darwin 要求声明主用户
  system.primaryUser = username;

  # nix-darwin state version。升级前请阅读 release notes
  system.stateVersion = 5;

  # Phase 3A 已开始接管：Homebrew 清单（保守模式，见 ./homebrew.nix）。
  # Phase 3C 已开始接管：少量稳定的 system.defaults 项（见 ./defaults.nix）。
  # 仍未接管：`brew services`、大范围 `system.defaults.*`、字体、GUI 自动化等，按后续阶段推进。
}
