{ lib, pkgs, username, ... }:
{
  # =============================================================================
  # Home Manager 用户层配置（Phase 1：骨架，默认不接管已有 dotfile）
  # =============================================================================

  imports = [
    # Phase 2A：低风险的包与通用 shell 环境变量
    ./packages.nix
    ./shell-env.nix

    # zsh 模块仍然故意不启用：启用会让 Home Manager 接管 ~/.zshrc，
    # 而当前 ~/.zshrc 仍是仓库里 zsh/.zshrc 的软链接。
    # Phase 2B 激活前请先：
    #   1) rm ~/.zshrc（或允许 home-manager 用 *.hm-backup 后缀备份）
    #   2) 在下一行取消注释
    # ../modules/zsh.nix
  ];

  home.username = username;
  home.homeDirectory = "/Users/${username}";

  # home-manager 状态版本，首次设置后尽量不要改
  home.stateVersion = "24.11";

  # 管理 home-manager 自身
  programs.home-manager.enable = true;

  # Phase 2A 起：少量稳定 CLI 交由 home.packages（见 ./packages.nix）；
  # 其它大多数软件（尤其是 cask / GUI / 带服务的工具）仍由 Homebrew 管理。
}
