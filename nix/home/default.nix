{ lib, pkgs, username, ... }:
{
  # =============================================================================
  # Home Manager 用户层配置
  # Phase 2C：zsh 模块已纳入配置图，但真正接管 ~/.zshrc 仍要等下一次 switch
  # =============================================================================

  imports = [
    # Phase 2A：低风险的包与通用 shell 环境变量
    ./packages.nix
    ./shell-env.nix

    # Phase 2C：zsh 模块现已进入配置图（flake check / build 可见），
    # 但真正生效仍需一次 `sudo darwin-rebuild switch`。
    # switch 发生时：
    #   - 仓库 `zsh/.zshrc → ~/.zshrc` 的软链接会被 home-manager 视为冲突，
    #     因 flake.nix 里设了 `backupFileExtension = "hm-backup"`，原软链接
    #     会被重命名为 `~/.zshrc.hm-backup`，随后 home-manager 生成自己的
    #     `~/.zshrc`（内容见 ../modules/zsh.nix + ../../zsh/shared.zsh）。
    #   - 机器相关片段（如 OpenClaw 绝对路径 completion）应放到 `~/.zshrc.local`，
    #     它会在 home-manager 版 zsh 末尾被自动 source，不进仓库。
    ../modules/zsh.nix
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
