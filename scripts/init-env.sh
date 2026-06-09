#!/usr/bin/env bash
# 初始化指定环境的 .env 文件（从模板复制，不覆盖已存在文件）
set -euo pipefail

ENV_NAME="${1:-development}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMPLATE="$ROOT/config/env/${ENV_NAME}.env.example"
TARGET="$ROOT/.env.${ENV_NAME}"

if [[ ! -f "$TEMPLATE" ]]; then
  echo "未知环境: ${ENV_NAME}"
  echo "可选: development | staging | production"
  exit 1
fi

if [[ -f "$TARGET" ]]; then
  echo "已存在，跳过: ${TARGET}"
else
  cp "$TEMPLATE" "$TARGET"
  echo "已创建: ${TARGET}"
  echo "请编辑其中的密钥和 DSN"
fi

# 基础 .env（共享默认值）
if [[ ! -f "$ROOT/.env" ]]; then
  cp "$ROOT/.env.example" "$ROOT/.env"
  echo "已创建: $ROOT/.env（来自 .env.example）"
fi

echo ""
echo "启动 ${ENV_NAME} 环境:"
echo "  APP_ENV=${ENV_NAME} go run ./cmd/agent-demo"
echo "或:"
case "$ENV_NAME" in
  development) echo "  make run-dev" ;;
  staging)     echo "  make run-staging" ;;
  production)  echo "  make run-prod" ;;
  *)           echo "  make run-agent APP_ENV=${ENV_NAME}" ;;
esac
