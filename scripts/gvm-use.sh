#!/usr/bin/env bash
# 在当前 shell 启用 GVM 并切换到项目 Go 版本
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_VERSION="$(tr -d '[:space:]' < "$ROOT/.go-version")"

if [[ ! -s "$HOME/.gvm/scripts/gvm" ]]; then
  echo "GVM 未安装，请先运行: bash scripts/install-gvm.sh"
  return 1 2>/dev/null || exit 1
fi

# shellcheck disable=SC1091
source "$HOME/.gvm/scripts/gvm"
gvm use "${GO_VERSION}" >/dev/null

export GOROOT="$GVMROOT/gos/${GO_VERSION}"
export PATH="$GOROOT/bin:$PATH"

echo "Go $(go version | awk '{print $3}') @ ${GOROOT}"
