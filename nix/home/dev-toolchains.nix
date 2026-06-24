{ pkgs, ... }:
{
  # =============================================================================
  # Phase 5A：语言/工具链管理器入口（仅入口，不迁移现有运行时）
  #
  # 设计意图：
  # - 仓库只负责安装稳定、跨机器通用的「管理器入口」
  # - 各项目的实际运行时版本应由项目内文件声明，不写进这份 Mac 配置仓库：
  #     * Node / Go / Deno / Bun     -> 项目内 .mise.toml
  #     * Python                     -> pyproject.toml + uv.lock
  #     * Rust                       -> rust-toolchain.toml
  #     * 需要系统库 / 编译器的项目  -> 项目内 flake.nix 的 devShell
  #
  # 边界：
  # - 不在 nix/darwin/homebrew.nix 里追加任何语言运行时
  # - 具体 Node / Go fallback 版本由仓库内 .config/mise/config.toml 管理
  # - zsh 激活由 nix/modules/zsh.nix 统一接入
  # =============================================================================

  home.packages = with pkgs; [
    # 多语言版本编排器：长期可作为 NVM 的替代候选，覆盖 Node / Go / Deno / Bun 等
    mise
    # Python 项目 / 包 / 虚拟环境管理器；保持全局可用
    uv
    # Rust 工具链管理器：相比固定的 rust 包更适合按项目切换 toolchain / target / component
    rustup
  ];

  # direnv + nix-direnv：为项目内 .envrc / Nix devShell 提供自动加载。
  # Home Manager 会为已启用的 zsh 安装 direnv hook；项目仍需显式 `direnv allow` 才会生效。
  programs.direnv = {
    enable = true;
    nix-direnv.enable = true;
  };
}
