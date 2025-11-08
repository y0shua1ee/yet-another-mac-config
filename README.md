# yet-another-mac-config

My Mac config

## 使用说明

1. 赋予脚本执行权限：`chmod +x setup_mac.sh`
2. 执行脚本：`./setup_mac.sh`
3. 根据提示输入目标 macOS 用户名，脚本会在 `/Users/<username>` 下创建指向仓库 `.config` 的软链接；若目标已存在会先询问是否覆盖。

后续其他初始化操作也会陆续添加到 `setup_mac.sh` 中。
