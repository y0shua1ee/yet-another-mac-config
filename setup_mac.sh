#!/usr/bin/env bash
# yet-another-mac-config bootstrap script.
# 1) 读取目标用户名
# 2) 将仓库中的 .config 以软链接方式放到用户目录
# 后续可在此脚本中扩展更多初始化步骤。
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source_path="$repo_dir/.config"

if [[ ! -d "$source_path" ]]; then
  echo "未找到 $source_path"
  exit 1
fi

# 请求目标用户名；允许同一脚本用于多台机器。
read -rp "请输入 macOS 用户名: " username
if [[ -z "$username" ]]; then
  echo "用户名不能为空"
  exit 1
fi

target_dir="/Users/$username"
target_path="$target_dir/.config"

if [[ ! -d "$target_dir" ]]; then
  echo "用户目录不存在: $target_dir"
  exit 1
fi

# 如果已有 .config，询问是否覆盖，避免意外删除。
if [[ -e "$target_path" || -L "$target_path" ]]; then
  read -rp "目标 $target_path 已存在，是否覆盖? [y/N] " answer
  if [[ ! "$answer" =~ ^[Yy]$ ]]; then
    echo "已取消"
    exit 0
  fi
  rm -rf "$target_path"
fi

ln -s "$source_path" "$target_dir/"
echo "已创建: $target_path -> $source_path"
