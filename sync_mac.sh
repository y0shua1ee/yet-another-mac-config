#!/usr/bin/env bash
# 使用当前仓库锁定的 Determinate Nix / nix-darwin / Home Manager 配置同步 Mac。
# 默认根据 LocalHostName 选择主机 profile；先 build，确认后才 switch。
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
host="$(scutil --get LocalHostName 2>/dev/null || true)"
build_only=false
assume_yes=false

usage() {
  cat <<'EOF'
用法：./sync_mac.sh [--host <名称>] [--build-only] [--yes]

  --host <名称>   覆盖 scutil 检测到的 LocalHostName
  --build-only    只构建，不激活当前 Mac
  --yes           构建通过后不再询问，直接执行 sudo switch
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --host)
      [[ $# -ge 2 ]] || { echo "--host 缺少主机名称" >&2; exit 2; }
      host="$2"
      shift 2
      ;;
    --build-only)
      build_only=true
      shift
      ;;
    --yes)
      assume_yes=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "未知参数：$1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "此脚本只支持 macOS。" >&2
  exit 1
fi

if [[ -z "$host" ]]; then
  echo "无法读取 LocalHostName，请使用 --host 显式指定。" >&2
  exit 1
fi

if ! command -v nix >/dev/null 2>&1 || ! nix --version | grep -q "Determinate Nix"; then
  echo "未检测到 Determinate Nix。请先按官方文档完成安装，再重新运行。" >&2
  echo "https://docs.determinate.systems/" >&2
  exit 1
fi

if ! command -v brew >/dev/null 2>&1; then
  echo "未检测到 Homebrew。nix-darwin 的 homebrew module 不负责安装 Homebrew。" >&2
  echo "请先按 https://brew.sh/ 的官方步骤安装，再重新运行。" >&2
  exit 1
fi

cd "$repo_dir"

echo "正在检查主机 profile：$host"
if ! nix eval --raw \
  ".#darwinConfigurations.${host}.config.system.build.toplevel.drvPath" \
  >/dev/null; then
  echo "没有可用的 darwinConfigurations.${host}，请先更新 nix/hosts/default.nix。" >&2
  exit 1
fi

configured_user="$(nix eval --raw \
  ".#darwinConfigurations.${host}.config.system.primaryUser")"
configured_repo="$(nix eval --raw \
  ".#darwinConfigurations.${host}.config.home-manager.extraSpecialArgs.repoPath")"

if [[ "$(id -un)" != "$configured_user" ]]; then
  echo "当前用户 $(id -un) 与 profile 用户 $configured_user 不一致。" >&2
  exit 1
fi

if [[ "$repo_dir" != "$configured_repo" ]]; then
  echo "当前仓库路径与 profile 不一致。" >&2
  echo "当前：$repo_dir" >&2
  echo "声明：$configured_repo" >&2
  echo "请先更新 nix/hosts/default.nix，再重新运行。" >&2
  exit 1
fi

echo "正在从 flake.lock 锁定的 nix-darwin 构建 darwin-rebuild 入口……"
rebuild_package="$(nix build --no-link --print-out-paths .#darwin-rebuild)"
darwin_rebuild="$rebuild_package/bin/darwin-rebuild"

echo "正在构建主机配置：$host"
"$darwin_rebuild" build --flake ".#${host}"

if [[ "$build_only" == true ]]; then
  echo "构建通过；按 --build-only 要求未激活当前 Mac。"
  exit 0
fi

# 旧版仓库曾把整个 ~/.config 链到 checkout。Home Manager 现在只拥有显式
# 白名单入口；若父目录仍是链接，switch 可能透过它改写仓库，必须先人工迁移。
if [[ -L "$HOME/.config" ]]; then
  echo "拒绝激活：$HOME/.config 仍是旧版整体符号链接。" >&2
  echo "请先按 nix/README.md 的旧版迁移步骤将它转换为真实目录。" >&2
  exit 1
fi

if [[ "$assume_yes" != true ]]; then
  read -r -p "构建通过，是否 sudo 激活 .#${host}？[y/N] " answer
  if [[ ! "$answer" =~ ^[Yy]$ ]]; then
    echo "已保留构建结果，未激活当前 Mac。"
    exit 0
  fi
fi

sudo "$darwin_rebuild" switch --flake ".#${host}"
echo "同步完成：$(readlink /run/current-system 2>/dev/null || echo /run/current-system)"
