#!/usr/bin/env bash
# 初始化项目隔离依赖环境（首次或换机器后执行一次）
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

# shellcheck disable=SC1091
source "${ROOT}/scripts/go-env.sh"

echo "==> Go 依赖隔离目录"
echo "    GOMODCACHE=${GOMODCACHE}"
echo "    GOCACHE=${GOCACHE}"
echo "    GOPATH=${GOPATH}"
echo ""

echo "==> Go 版本"
go version
echo ""

echo "==> 下载依赖到项目本地缓存 (go mod download)"
go mod download

echo "==> 校验 go.sum (go mod verify)"
go mod verify

echo "==> 整理 go.mod (go mod tidy)"
go mod tidy

echo ""
echo "✅ 依赖隔离环境就绪"
echo ""
echo "下一步（推荐，完全离线可构建）:"
echo "  bash scripts/deps-vendor.sh"
echo ""
echo "日常开发:"
echo "  source scripts/go-env.sh   # 或 direnv allow"
echo "  make run-dev"
