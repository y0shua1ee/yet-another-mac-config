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
  # 边界（Phase 5A 范围内不做的事）：
  # - 不迁移当前活跃 Node：NVM 与 ~/.nvm 体系保持原状
  # - 不删除或替换 Homebrew 中的 go / rust / nvm / pnpm / uv / deno / llvm 等
  # - 不在 nix/darwin/homebrew.nix 里追加任何语言运行时
  # - 不启用 `mise activate zsh` 等 shell 集成，留给 Phase 5B 单独评估
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
  # 注意：Phase 5A 仍不启用 `mise activate zsh`，避免抢走当前 NVM 管理的 Node。
  programs.direnv = {
    enable = true;
    nix-direnv.enable = true;
  };
}
