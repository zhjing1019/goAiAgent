#!/usr/bin/env bash
# 安装 GVM 并切换到项目要求的 Go 版本（.go-version）
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_VERSION="$(tr -d '[:space:]' < "$ROOT/.go-version")"

echo "==> 项目 Go 版本: ${GO_VERSION}"

if [[ ! -d "$HOME/.gvm" ]]; then
  echo "==> 安装 GVM..."
  if ! bash < <(curl -s -S -L --connect-timeout 15 https://raw.githubusercontent.com/moontide/gvm/master/binscripts/gvm-installer); then
    echo ""
    echo "❌ 无法从 GitHub 下载 GVM 安装脚本（网络问题）。"
    echo "   可手动安装:"
    echo "   git clone https://github.com/moontide/gvm.git ~/.gvm"
    echo "   cd ~/.gvm && ./install"
    echo "   然后重新运行: bash scripts/install-gvm.sh"
    exit 1
  fi
else
  echo "==> GVM 已安装: $HOME/.gvm"
fi

# shellcheck disable=SC1091
source "$HOME/.gvm/scripts/gvm"

if ! gvm list | grep -q "${GO_VERSION}"; then
  echo "==> 安装 Go ${GO_VERSION}（可能需要几分钟）..."
  if ! gvm install "${GO_VERSION}" -s; then
    echo "==> 源码安装失败，尝试二进制安装..."
    gvm install "${GO_VERSION}" -b
  fi
fi

gvm use "${GO_VERSION}" --default
hash -r

echo ""
echo "✅ GVM 就绪"
echo "   Go: $(go version)"
echo ""
echo "请将以下内容加入 ~/.zshrc（若尚未添加）:"
echo '  [[ -s "$HOME/.gvm/scripts/gvm" ]] && source "$HOME/.gvm/scripts/gvm"'
echo ""
echo "进入项目后执行:"
echo "  source scripts/gvm-use.sh"
