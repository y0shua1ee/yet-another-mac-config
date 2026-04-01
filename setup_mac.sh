#!/usr/bin/env bash
# yet-another-mac-config 初始化脚本。
# 1) 读取目标用户名
# 2) 仅同步仓库中已跟踪的配置目录
# 3) 后续可在此脚本中扩展更多初始化步骤
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
configs_dir="$repo_dir/.config"
codex_dir="$repo_dir/.codex"

if [[ ! -d "$configs_dir" ]]; then
  echo "未找到 $configs_dir"
  exit 1
fi

collect_config_names() {
  local tracked_names

  # 优先读取 Git 已跟踪的目录，避免把本地忽略目录也拿来创建软链接。
  if command -v git >/dev/null 2>&1 && git -C "$repo_dir" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    tracked_names="$(git -C "$repo_dir" ls-files -- .config | awk -F/ 'NF >= 3 {print $2}' | sort -u)"
    if [[ -n "$tracked_names" ]]; then
      printf '%s\n' "$tracked_names"
      return
    fi
  fi

  # 如果当前目录不是 Git 工作区，则回退到现有目录扫描逻辑。
  find "$configs_dir" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort
}

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
while IFS= read -r config_name; do
  [[ -n "$config_name" ]] || continue

  config_source="$configs_dir/$config_name"
  if [[ ! -d "$config_source" ]]; then
    continue
  fi

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
done < <(collect_config_names)

codex_config_source="$codex_dir/config.toml"
if [[ -f "$codex_config_source" ]]; then
  codex_target_dir="$target_dir/.codex"
  codex_target_path="$codex_target_dir/config.toml"

  read -rp "是否将 Codex 配置链接到 $codex_target_path? [y/N] " codex_answer
  if [[ "$codex_answer" =~ ^[Yy]$ ]]; then
    mkdir -p "$codex_target_dir"
    if [[ -e "$codex_target_path" || -L "$codex_target_path" ]]; then
      read -rp "目标 $codex_target_path 已存在，是否覆盖? [y/N] " codex_replace
      if [[ ! "$codex_replace" =~ ^[Yy]$ ]]; then
        echo "保留现有：$codex_target_path"
      else
        rm -rf "$codex_target_path"
        ln -s "$codex_config_source" "$codex_target_path"
        echo "已创建: $codex_target_path -> $codex_config_source"
        created_any=true
      fi
    else
      ln -s "$codex_config_source" "$codex_target_path"
      echo "已创建: $codex_target_path -> $codex_config_source"
      created_any=true
    fi
  else
    echo "跳过 Codex 配置"
  fi
fi

# 额外处理 .zshrc，软链接到用户主目录
zshrc_source="$repo_dir/zsh/.zshrc"
if [[ -f "$zshrc_source" ]]; then
  zshrc_target="$target_dir/.zshrc"
  read -rp "是否将 .zshrc 链接到 $zshrc_target? [y/N] " zsh_answer
  if [[ "$zsh_answer" =~ ^[Yy]$ ]]; then
    if [[ -e "$zshrc_target" || -L "$zshrc_target" ]]; then
      read -rp "目标 $zshrc_target 已存在，是否覆盖? [y/N] " zsh_replace
      if [[ ! "$zsh_replace" =~ ^[Yy]$ ]]; then
        echo "保留现有：$zshrc_target"
      else
        rm -rf "$zshrc_target"
        ln -s "$zshrc_source" "$zshrc_target"
        echo "已创建: $zshrc_target -> $zshrc_source"
        created_any=true
      fi
    else
      ln -s "$zshrc_source" "$zshrc_target"
      echo "已创建: $zshrc_target -> $zshrc_source"
      created_any=true
    fi
  else
    echo "跳过 .zshrc"
  fi
fi

# 额外处理 Hammerspoon 配置目录，便于统一软链接
hammerspoon_source="$repo_dir/.hammerspoon"
if [[ -d "$hammerspoon_source" ]]; then
  hammerspoon_target="$target_dir/.hammerspoon"
  read -rp "是否将 Hammerspoon 配置链接到 $hammerspoon_target? [y/N] " hs_answer
  if [[ "$hs_answer" =~ ^[Yy]$ ]]; then
    if [[ -e "$hammerspoon_target" || -L "$hammerspoon_target" ]]; then
      read -rp "目标 $hammerspoon_target 已存在，是否覆盖? [y/N] " hs_replace
      if [[ ! "$hs_replace" =~ ^[Yy]$ ]]; then
        echo "保留现有：$hammerspoon_target"
      else
        rm -rf "$hammerspoon_target"
        ln -s "$hammerspoon_source" "$hammerspoon_target"
        echo "已创建: $hammerspoon_target -> $hammerspoon_source"
        created_any=true
      fi
    else
      ln -s "$hammerspoon_source" "$hammerspoon_target"
      echo "已创建: $hammerspoon_target -> $hammerspoon_source"
      created_any=true
    fi
  else
    echo "跳过 Hammerspoon 配置"
  fi
fi

if [[ "$created_any" != true ]]; then
  echo "本次未创建任何链接。"
fi
