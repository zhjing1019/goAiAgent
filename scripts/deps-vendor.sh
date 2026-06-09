#!/usr/bin/env bash
# 将依赖复制到 vendor/，实现与全局 GOPATH/GOMODCACHE 完全解耦的构建
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

# shellcheck disable=SC1091
source "${ROOT}/scripts/go-env.sh"

echo "==> 生成 vendor/ (go mod vendor)"
go mod vendor

export GOFLAGS="-mod=vendor"
echo "==> 验证 vendor 模式构建"
go build -o /dev/null ./cmd/agent-demo

echo ""
echo "✅ vendor/ 已生成"
echo "   后续 go build / go run 将只使用 vendor/ 中的依赖"
echo "   更新依赖后请重新执行: bash scripts/deps-vendor.sh"
