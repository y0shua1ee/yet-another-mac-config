{ username, ... }:
{
  # =============================================================================
  # Phase 3A：Homebrew 声明式清单（保守首版）
  # =============================================================================
  #
  # 设计要点：
  #   - 只纳入本机已使用、长期稳定、低风险的 formula / cask。
  #   - 不做自动 upgrade、autoUpdate、cleanup；未声明的本机 brew 包不会被动掉。
  #   - 不接管 `brew services`、字体、`.hammerspoon` 相关 cask；不含 secrets / 账号态工具。
  #   - 这是 Phase 3A 的起点，不是对现有 Homebrew 安装项的全面接管。
  #
  # 参考：https://github.com/nix-darwin/nix-darwin/blob/master/modules/homebrew.nix
  homebrew = {
    enable = true;

    # 指定执行 brew 命令所用的用户（非 root），与 flake.nix 中 username 保持一致
    user = username;

    # 激活策略：保守模式
    onActivation = {
      # 切换时不自动 `brew update`
      autoUpdate = false;
      # 切换时不自动 `brew upgrade` 已声明的包
      upgrade = false;
      # 不清理未声明的本机 brew 包（保留 "none"；未来可视情况再考虑 "check"）
      cleanup = "none";
    };

    # 第三方 tap：仅声明当前清单中实际依赖的
    taps = [
      # AeroSpace cask 的来源
      "nikitabobko/tap"
    ];

    # brew formulae：聚焦于稳定 CLI 工具
    # 刻意不包含的：
    #   - ripgrep / fd / jq / tree / bat：已由 Home Manager `home.packages` 管理
    #   - 服务类（borders / nginx / unbound / colima / clouddrive2 / ollama 等）：
    #     与 brew services 绑定，按 Phase 3A 约束不纳入
    #   - 版本管理器 / 多语言运行时（nvm / pnpm / uv / deno / python@* / go / rust / llvm 等）：
    #     状态管理更复杂，延后评估
    #   - 媒体 / 构建依赖（ffmpeg / imagemagick 等）：多数是其它包的传递依赖，不需要显式声明
    brews = [
      "ast-grep"        # 结构化代码搜索，长期使用
      "btop"            # 系统监控（配置已由 .config/btop 管理）
      "fastfetch"       # 系统信息展示
      "fzf"             # 模糊查找器，zsh 交互依赖
      "gh"              # GitHub CLI
      "git"             # 显式声明 brew 版 git，覆盖 macOS 自带版本
      "lazygit"         # Git TUI
      "neovim"          # 主力编辑器（配置已由 .config/nvim 管理）
      "starship"        # 跨 shell 提示符，shell 共享逻辑中已引用
      # Phase 3B：tmux 运行时声明化。
      # 选择继续放在 Homebrew 而不是 Home Manager `home.packages`，理由：
      #   - 本机 tmux 已经由 Homebrew 安装在 /opt/homebrew/bin/tmux，长期稳定
      #   - 配置体系（oh-my-tmux + ~/.config/tmux/tmux.conf.local）不变，本阶段只接管运行时
      #   - 与 neovim / starship / git 的口径一致，避免双份 tmux 二进制在 PATH 里互相覆盖
      "tmux"            # 终端复用器（配置由 .config/tmux 管理；oh-my-tmux 仍按本地克隆方式使用）
      "wget"            # 通用下载工具
      "yazi"            # 文件管理器（配置已由 .config/yazi 管理）
      "yt-dlp"          # 媒体下载 CLI
      "zsh-completions" # zsh 第三方补全集合，zsh 配置已引用
    ];

    # cask GUI：只选“长期保留 + 仓库已管理其配置”的条目
    # 刻意不包含的：
    #   - font-*：字体按约定不在 Phase 3A 接管
    #   - hammerspoon：`.hammerspoon` 相关目前不动
    #   - 含账号态 / 登录态的工具（如 1password-cli、各 IM / 云盘 / 登录类 app）
    #   - raycast：带大量本地扩展与账号态
    casks = [
      "aerospace"          # 窗口管理器（配置由 .config/aerospace 管理）
      "ghostty"            # 主力终端（配置由 .config/ghostty 管理）
      "typora"             # Markdown 编辑器（主题由 .config/typora 管理）
      "visual-studio-code" # 编辑器（项目级设置由 .vscode 管理）
    ];
  };
}
