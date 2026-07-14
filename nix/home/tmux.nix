{ config, inputs, repoPath, ... }:
let
  repoSymlink = relativePath:
    config.lib.file.mkOutOfStoreSymlink "${repoPath}/${relativePath}";
in
{
  # 上游主配置由 flake.lock 固定，避免首次启动临时 clone 未锁定的 master。
  xdg.configFile."tmux/tmux.conf".source = inputs.oh-my-tmux + "/.tmux.conf";

  # 用户自定义继续直接指向仓库，编辑后无需重建 Nix Store 内容。
  xdg.configFile."tmux/tmux.conf.local".source =
    repoSymlink ".config/tmux/tmux.conf.local";

  # `~/.config/tmux` 故意保持真实可写目录；oh-my-tmux 内建 TPM 会在
  # `plugins/` 下安装、更新和修补插件，该可变状态不由 Home Manager 接管。
}
