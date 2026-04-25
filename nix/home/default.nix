{ lib, pkgs, username, ... }:
{
  # =============================================================================
  # Home Manager 用户层配置
  # Phase 2D：Home Manager 已接管 ~/.zshrc
  # =============================================================================

  imports = [
    # Phase 2A：低风险的包与通用 shell 环境变量
    ./packages.nix
    ./shell-env.nix

    # Phase 5A：语言 / 工具链管理器入口（mise / uv / rustup + direnv）
    # - 只负责安装管理器本身，不迁移现有 NVM / Homebrew 语言运行时
    # - 实际运行时版本由项目内 .mise.toml / uv.lock / rust-toolchain.toml / devShell 声明
    ./dev-toolchains.nix

    # Phase 2D：zsh 模块已完成 switch 并实际接管 ~/.zshrc。
    # 当前机器上：
    #   - Home Manager 正在生成 `~/.zshrc`（内容见 ../modules/zsh.nix + ../../zsh/shared.zsh）。
    #   - 旧仓库软链接已手动保留为 `~/.zshrc.pre-hm-switch-backup`，便于人工回退。
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
