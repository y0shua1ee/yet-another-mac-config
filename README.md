# yet-another-mac-config

My Mac config

## 使用说明

1. 赋予脚本执行权限：`chmod +x setup_mac.sh`
2. 执行脚本：`./setup_mac.sh`
3. 根据提示输入目标 macOS 用户名，脚本会逐个遍历仓库 `.config` 下的子目录，并在 `/Users/<username>/.config` 中为每个项目单独创建软链接；若某个目标项已存在，会先确认是否覆盖，默认则跳过。

后续其他初始化操作也会陆续添加到 `setup_mac.sh` 中。
