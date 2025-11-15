# yet-another-mac-config

My Mac config

## 使用说明

1. 赋予脚本执行权限：`chmod +x setup_mac.sh`
2. 执行脚本：`./setup_mac.sh`
3. 根据提示输入目标 macOS 用户名，脚本会逐个遍历仓库 `.config` 下的子目录，并在 `/Users/<username>/.config` 中创建软链接；若某个目标项已存在，会先确认是否覆盖，默认则跳过。
4. 同一脚本也会检测仓库根目录下的 `.hammerspoon`，提示是否同步到 `~/.hammerspoon`，这样 Hammerspoon 配置可与仓库保持一致；在此之前请先通过 `brew install --cask hammerspoon` 安装好 Hammerspoon，并根据需要安装 `Ghostty`（例如 `brew install --cask ghostty`）以使用 `Ctrl+Alt+T` 新开 Ghostty 窗口的快捷方式。

## Yazi 插件同步

`install_yazi_plugins.sh` 用来在新环境里批量安装/更新 `package.toml` 中锁定的所有 Yazi 插件，并按需设置部分环境变量（比如 `LG_CONFIG_FILE`，确保 `lazygit.yazi` 能工作）。使用方式：

1. 确认 `ya` CLI 已安装：`brew install yazi`。
2. 可选：指定配置目录，例如 `./install_yazi_plugins.sh --config-dir "$HOME/.config/yazi"`；若不传参数脚本会优先使用 `XDG_CONFIG_HOME/yazi`，否则回退到仓库内 `.config/yazi`。
3. 等待脚本自动执行 `ya pkg install`，输出当前生效的插件列表，并提示缺失的依赖工具（如 `starship`、`lazygit`、`7zz` 等）。

脚本可安全重复执行，方便在多台机器间保持插件一致。

后续其他初始化操作也会陆续添加到 `setup_mac.sh` 中。
