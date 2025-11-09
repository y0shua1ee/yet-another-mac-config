#!/usr/bin/env bash
# yet-another-mac-config bootstrap script.
# 1) 读取目标用户名
# 2) 在目标用户的 ~/.config 中按项目创建软链接
# 后续可在此脚本中扩展更多初始化步骤。
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
configs_dir="$repo_dir/.config"

if [[ ! -d "$configs_dir" ]]; then
  echo "未找到 $configs_dir"
  exit 1
fi

# 请求目标用户名；允许同一脚本用于多台机器。
read -rp "请输入 macOS 用户名: " username
if [[ -z "$username" ]]; then
  echo "用户名不能为空"
  exit 1
fi

target_dir="/Users/$username"
target_config_dir="$target_dir/.config"

if [[ ! -d "$target_dir" ]]; then
  echo "用户目录不存在: $target_dir"
  exit 1
fi

mkdir -p "$target_config_dir"

created_any=false
while IFS= read -r -d '' config_source; do
  config_name="$(basename "$config_source")"
  target_path="$target_config_dir/$config_name"

  read -rp "是否为 $config_name 创建软链接到 $target_path? [y/N] " answer
  if [[ ! "$answer" =~ ^[Yy]$ ]]; then
    echo "跳过 $config_name"
    continue
  fi

  if [[ -e "$target_path" || -L "$target_path" ]]; then
    read -rp "目标 $target_path 已存在，是否覆盖? [y/N] " replace
    if [[ ! "$replace" =~ ^[Yy]$ ]]; then
      echo "保留现有：$target_path"
      continue
    fi
    rm -rf "$target_path"
  fi

  ln -s "$config_source" "$target_path"
  echo "已创建: $target_path -> $config_source"
  created_any=true
done < <(find "$configs_dir" -mindepth 1 -maxdepth 1 -print0)

if [[ "$created_any" != true ]]; then
  echo "本次未创建任何链接。"
fi
