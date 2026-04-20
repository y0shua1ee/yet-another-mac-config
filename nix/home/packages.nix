{ pkgs, ... }:
{
  # =============================================================================
  # Home Manager 包（Phase 2A：只挑稳定、纯 CLI、与 Homebrew 可并存的低风险工具）
  # - 这些二进制会被装到 /etc/profiles/per-user/<user>/bin，由 nix-darwin 自动加到 PATH
  # - 与 Homebrew 版本共存时，PATH 顺序决定哪个胜出；此处不追求替换 Homebrew 安装
  # - 新增包请继续保持「纯 CLI、无 GUI、无 launchd 服务、不写用户 dotfile」的标准
  # =============================================================================

  home.packages = with pkgs; [
    ripgrep # rg：快速文本搜索
    fd      # find 的现代替代
    jq      # JSON 处理
    tree    # 目录树
    bat     # 带高亮的 cat
  ];
}
