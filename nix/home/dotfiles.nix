{ config, lib, repoPath, ... }:
let
  # 只链接明确审核过、由 Git 跟踪的静态配置目录。
  # Alma、登录态、缓存和凭据目录不得通过自动目录扫描进入这里。
  managedConfigDirectories = [
    "aerospace"
    "borders"
    "btop"
    "gh"
    "ghostty"
    "mise"
    "mpv"
    "nvim"
    "typora"
    "yazi"
  ];

  repoSymlink = relativePath:
    config.lib.file.mkOutOfStoreSymlink "${repoPath}/${relativePath}";
in
{
  xdg.enable = true;

  # 使用 out-of-store symlink，让 app 与编辑器继续直接操作仓库工作区；
  # Git 和 .gitignore 负责区分可同步配置与本机状态。
  xdg.configFile = lib.genAttrs managedConfigDirectories (name: {
    source = repoSymlink ".config/${name}";
  });

  home.file.".hammerspoon".source = repoSymlink ".hammerspoon";
}
