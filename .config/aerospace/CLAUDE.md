# AeroSpace configuration guidance

## Structure
- `aerospace.toml` is the single config file, read from `~/.config/aerospace/aerospace.toml` (symlinked from this repo via `setup_mac.sh`).
- Major sections: global flags、`[[on-window-detected]]` 窗口规则、`[workspace-to-monitor-force-assignment]` 显示器映射、`[mode.main.binding]` 主快捷键、`[mode.*]` 其他绑定模式。

## Workflow
- 修改前先查 [AeroSpace 官方文档](https://nikitabobko.github.io/AeroSpace/guide)，尤其是 commands/guide 两页。
- 保持 `start-at-login = true`（已在仓库中启用）。
- 新增 `[[on-window-detected]]` 规则时，`if.app-id` / `if.window-title-regex-substring` 的取值必须通过 AeroSpace 官方内置 inspection 命令获得，不要依赖 OS 元数据（如 `mdls`、`osascript`）——AeroSpace 看到的标识可能与 macOS 系统 API 不同。
  - 常用查询：`aerospace list-windows --all`、`aerospace list-apps`。
- 改动快捷键时注意 `alt-` 前缀在 macOS 上等价于 Option；避免与系统/其他 WM 冲突。
- 避免一次性大规模重排配置，按需追加/调整即可。

## Reload
- 执行 `aerospace reload-config` 应用改动；保存文件后 AeroSpace 不会自动重载。
