{ pkgs, lib, username, ... }:
{
  # =============================================================================
  # nix-darwin 系统层配置（Phase 1：刻意保持最小）
  # =============================================================================

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

  # Phase 1 暂不接管：Homebrew、系统默认值（system.defaults.*）、服务、字体等。
  # 后续阶段再按需要逐步打开。
}
