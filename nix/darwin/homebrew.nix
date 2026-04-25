{ username, ... }:
{
  # =============================================================================
  # Homebrew 声明式清单
  #   - Phase 3A：保守首版（CLI 工具 + 已管理配置的 GUI cask）
  #   - Phase 3B：新增 tmux 运行时
  #   - Phase 4 最小版：新增 `brew services` 试点（仅 `borders` / `nginx`），
  #     并补上 Ghostty 所需字体 `font-maple-mono-nf` 与 `hammerspoon` cask。
  #   - Phase 4B 小幅扩张：补齐已被仓库工作流长期使用的少量 CLI helper，
  #     以及三款已稳定使用的 AI / 助手类 cask。仍刻意不引入语言运行时与
  #     账号态重的 GUI app（详见下方“刻意不纳入”小节）。
  # =============================================================================
  #
  # 设计要点：
  #   - 只纳入本机已使用、长期稳定、低风险的 formula / cask。
  #   - 不做自动 upgrade、autoUpdate、cleanup；未声明的本机 brew 包不会被动掉。
  #   - `brew services` 目前只接管 `borders` 与 `nginx`，策略选 `start_service = true`
  #     （只在服务未运行时启动，不会重启已运行服务），对现状零扰动。
  #     Phase 4B **不**扩张服务接管范围：colima / clouddrive2 / ollama / unbound
  #     仍走人工 `brew services` 流程。
  #   - 仍未纳入的字体：`font-hack-nerd-font`（本机当前虽已安装，但未被仓库配置引用）。
  #     本轮字体只补 Ghostty 明确依赖的一项，避免“能配就都配”。
  #
  # 参考：https://github.com/nix-darwin/nix-darwin/blob/master/modules/homebrew.nix
  homebrew = {
    enable = true;

    # 指定执行 brew 命令所用的用户（非 root），与 flake.nix 中 username 保持一致
    user = username;

    # 激活策略：保守模式（Phase 4B 维持不变）
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
      # JankyBorders（`borders` formula）的来源
      "felixkratz/formulae"
      # Apple 助手 CLI（`memo`）的来源
      "antoniorodr/memo"
      # Apple 助手 CLI（`remindctl`）的来源
      "steipete/tap"
    ];

    # brew formulae：聚焦于稳定 CLI 工具 + 少量保守接管的服务类
    # 刻意不包含的：
    #   - ripgrep / fd / jq / tree / bat：已由 Home Manager `home.packages` 管理
    #   - 其它服务类（clouddrive2 / ollama / unbound 等）：
    #     clouddrive2 / ollama 牵涉账号态与后台数据；unbound 当前不是默认开机自启，
    #     本轮仍不纳入服务接管，留给后续阶段单独评估
    #   - 版本管理器 / 多语言运行时（nvm / pnpm / uv / deno / python@* / go / rust / llvm 等）：
    #     状态管理更复杂，且与未来 Home Manager / devshell / mise 等方案耦合度高，
    #     Phase 4B **故意延后**评估，不在本轮纳入
    brews = [
      "ast-grep"        # 结构化代码搜索，长期使用
      "btop"            # 系统监控（配置已由 .config/btop 管理）
      "fastfetch"       # 系统信息展示
      "fzf"             # 模糊查找器，zsh 交互依赖
      "gh"              # GitHub CLI
      "gitleaks"        # secrets 扫描，适合提交前/手动审计做轻量防线
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

      # -----------------------------------------------------------------
      # Phase 4B：容器运行时（仅声明 CLI / 运行时，不接管 `brew services`）
      # -----------------------------------------------------------------
      # 说明：
      #   - colima 提供轻量 Linux VM，作为 Docker Desktop 的替代
      #   - docker / docker-compose 作为客户端 CLI，长期使用，已在根 README 描述
      #   - 这里**不**把 colima 纳入 `brew services`：本机一直按需 `colima start`，
      #     不希望切换 generation 时自动拉起后台 VM
      "colima"          # 轻量容器虚拟机（手动 `colima start` / `stop` 管理）
      "docker"          # docker CLI
      "docker-compose"  # docker compose 插件（需在 ~/.docker/config.json 中登记 cliPluginsExtraDirs）

      # -----------------------------------------------------------------
      # Phase 4B：Yazi / 媒体 / 文档 helper（多数是 Yazi 插件依赖）
      # -----------------------------------------------------------------
      # 说明：以下条目都是 install_yazi_plugins.sh 末尾提示过的依赖，或长期被 Yazi
      # 预览 / 提取 / 跳转流程使用。统一声明能避免新机器上预览缺失依赖、
      # 安静失败的问题。
      "sevenzip"        # 7zz 压缩 / 解压（Yazi 预览归档常用）
      "imagemagick"     # `magick`，多个 Yazi 预览 / 转换流程依赖
      "mpv"             # 视频播放与预览，Yazi 视频打开依赖
      "poppler"         # `pdftoppm` 等，PDF 预览依赖
      "zoxide"          # 智能目录跳转（Yazi 跳转键位依赖）
      "media-info"      # `mediainfo`，Yazi 媒体元信息预览
      "exiftool"        # 图像 / 媒体元数据查看

      # -----------------------------------------------------------------
      # Phase 4B：Email / 助手类 CLI
      # -----------------------------------------------------------------
      "himalaya"                       # 终端邮件客户端（Biya 工作流使用）
      "antoniorodr/memo/memo"          # Apple Notes 命令行助手（来自 antoniorodr/memo tap）
      "steipete/tap/remindctl"         # Apple Reminders 命令行助手（来自 steipete/tap）

      # -----------------------------------------------------------------
      # Phase 4 最小版：`brew services` 试点（仅这两项）
      # -----------------------------------------------------------------
      # 策略说明：
      #   - 使用 `start_service = true`：nix-darwin 会在 brew bundle 阶段调用
      #     `brew services start`，**仅在服务未运行时启动**，不会重启或停止
      #     已运行的服务，对当前状态零扰动。
      #   - 本机当前两者均已 `started`，首次 switch 预期是幂等 no-op。
      #   - 新机器走 Nix 路线 switch 后，会自动安装并登记为 login item。
      #   - 刻意未使用 `restart_service`：任何 `darwin-rebuild switch` 都不应
      #     重启这些长期运行的服务；仍沿用现有 `brew services restart <name>` 人工流程。
      #   - Phase 4B **不**扩张此名单：colima / clouddrive2 / ollama / unbound
      #     继续按 README 中的人工 `brew services` 流程管理。
      { name = "borders"; start_service = true; }   # JankyBorders 窗口边框（配置：.config/borders）
      { name = "nginx"; start_service = true; }     # 本地 HTTP 服务器（配置路径：/opt/homebrew/etc/nginx/）
    ];

    # cask GUI：只选“长期保留 + 仓库已管理其配置”或“仓库自动化依赖”的条目
    # 刻意不包含的（Phase 4B 也继续不纳入）：
    #   - font-hack-nerd-font：本机已安装但未被仓库配置引用
    #   - 含较重账号态 / 登录态、或与个人云端数据强耦合的 app：
    #       raycast / telegram / discord / feishu / google-drive / tailscale /
    #       notion / spotify / zotero / jetbrains-toolbox / termius 等
    #     这些 app 的本地数据 / 登录态远比 cask 安装本身更关键，不适合声明式接管
    casks = [
      "aerospace"          # 窗口管理器（配置由 .config/aerospace 管理）
      "ghostty"            # 主力终端（配置由 .config/ghostty 管理）
      "typora"             # Markdown 编辑器（主题由 .config/typora 管理）
      "visual-studio-code" # 编辑器（项目级设置由 .vscode 管理）

      # -----------------------------------------------------------------
      # Phase 4 最小版：补上 Ghostty 依赖字体 + Hammerspoon
      # -----------------------------------------------------------------
      # font-maple-mono-nf：`.config/ghostty/config` 中 `font-family = Maple Mono Normal NF CN`
      #   明确依赖此字体；新机器缺少它会直接 fallback，终端外观会不一致。
      # hammerspoon：仓库根目录 `.hammerspoon/` 是仓库事实源；新机器除了同步配置外，
      #   还需额外在「系统设置 → 隐私与安全性 → 辅助功能」里给 Hammerspoon 授权，
      #   详见根 README 的「Hammerspoon 激活说明」。
      "font-maple-mono-nf" # Ghostty 默认字体（Maple Mono NF）
      "hammerspoon"        # 自动化与快捷键（配置由 .hammerspoon 管理；需人工授予 Accessibility 权限）

      # -----------------------------------------------------------------
      # Phase 4B：AI / 助手类 cask
      # -----------------------------------------------------------------
      # 这三款 app 长期被本机日常使用，不依赖账号导入 / 同步即可起步，
      # 因此适合纳入声明式清单，方便新机器一键安装到位：
      #   - claude-code@latest：Claude Code CLI（沿用 Homebrew 上游的 @latest 版本通道）
      #   - codex：Codex CLI
      #   - cc-switch：Claude Code 模型 / 账号切换辅助
      "claude-code@latest" # Claude Code CLI（@latest 版本通道）
      "codex"              # Codex CLI
      "cc-switch"          # Claude Code 切换辅助
    ];
  };
}
