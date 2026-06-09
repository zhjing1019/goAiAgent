#!/usr/bin/env bash
# 在隔离环境中执行任意命令
# 用法: scripts/with-go-env.sh go test ./...
set -eo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${ROOT}/scripts/go-env.sh"

# 若已安装 GVM，优先使用项目 Go 版本（GVM 内部可能返回非 0，忽略即可）
if [[ -s "${HOME}/.gvm/scripts/gvm" ]]; then
  set +e
  # shellcheck disable=SC1091
  source "${HOME}/.gvm/scripts/gvm"
  if [[ -f "${ROOT}/.go-version" ]]; then
    ver="$(tr -d '[:space:]' < "${ROOT}/.go-version")"
    gvm use "${ver}" >/dev/null 2>&1
  fi
  set -e
fi

exec "$@"
