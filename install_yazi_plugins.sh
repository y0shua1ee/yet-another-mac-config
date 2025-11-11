#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: install_yazi_plugins.sh [--config-dir PATH]

Sync the Yazi plugins tracked in this repository onto a new system.

Options:
  -c, --config-dir PATH  Yazi configuration directory to operate on.
                         Defaults to $XDG_CONFIG_HOME/yazi if it exists,
                         otherwise falls back to the repo’s .config/yazi.
  -h, --help             Show this message.
EOF
}

CONFIG_DIR_OVERRIDE=""

## 解析命令行参数，方便对接其他初始化脚本
while [[ $# -gt 0 ]]; do
  case "$1" in
    -c|--config-dir)
      CONFIG_DIR_OVERRIDE="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
tracked_config="$repo_dir/.config/yazi"
default_xdg="${XDG_CONFIG_HOME:-$HOME/.config}/yazi"

# 优先使用用户传入的配置，其次是现有 XDG 目录，最后回退到仓库自身
if [[ -n "$CONFIG_DIR_OVERRIDE" ]]; then
  config_dir="$CONFIG_DIR_OVERRIDE"
elif [[ -d "$default_xdg" ]]; then
  config_dir="$default_xdg"
else
  config_dir="$tracked_config"
fi

if [[ ! -d "$config_dir" ]]; then
  echo "Yazi config directory not found: $config_dir" >&2
  exit 1
fi

package_file="$config_dir/package.toml"
if [[ ! -f "$package_file" ]]; then
  echo "package.toml not found inside $config_dir" >&2
  exit 1
fi

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

warn_if_missing() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Warning: $2" >&2
  fi
}

require_cmd ya
require_cmd git

warn_if_missing starship "starship is needed for the starship.yazi prompt integration."
warn_if_missing lazygit "lazygit is needed for the lazygit.yazi binding."
warn_if_missing 7zz "7-Zip CLI (7zz) unlocks password-protected archives in compress.yazi."
# zoom.yazi 依赖 ImageMagick，提前提醒缺失的 magick 命令
warn_if_missing magick "ImageMagick (magick) is required for zoom.yazi to scale previews."

if [[ "$OSTYPE" == darwin* ]]; then
  default_lg_config="$HOME/Library/Application Support/lazygit/config.yml"
else
  default_lg_config="${XDG_CONFIG_HOME:-$HOME/.config}/lazygit/config.yml"
fi

# 确保 LG_CONFIG_FILE 指向一个实际存在的目录，方便 lazygit 写入缓存
: "${LG_CONFIG_FILE:=$default_lg_config}"
mkdir -p "$(dirname "$LG_CONFIG_FILE")"
export LG_CONFIG_FILE

# 指定 YAZI_CONFIG_HOME，让 ya 针对正确的 package.toml 工作
export YAZI_CONFIG_HOME="$config_dir"

plugins=(
  "yazi-rs/plugins:smart-enter"
  "yazi-rs/plugins:git"
  "Rolv-Apneseth/starship"
  "KKV9/compress"
  "Lil-Dank/lazygit"
  "yazi-rs/plugins:full-border"
  "yazi-rs/plugins:zoom"
)

echo "YAZI_CONFIG_HOME set to: $YAZI_CONFIG_HOME"
echo "LG_CONFIG_FILE set to: $LG_CONFIG_FILE"
echo "Ensuring ${#plugins[@]} plugins from package.toml are installed…"

# 使用 package.toml 锁定的版本安装插件，保证多机器一致
ya pkg install >/dev/null

echo "Installed plugins:"
ya pkg list

cat <<'EOF'
Done! Restart Yazi (and your shell, if needed) to load the refreshed plugin stack.
EOF
