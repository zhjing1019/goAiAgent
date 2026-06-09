#!/usr/bin/env bash
# 项目级 Go 依赖隔离环境变量
# 用法: source scripts/go-env.sh   （bash / zsh 均可）
# 效果: 本项目的模块缓存、编译缓存、GOPATH 全部落在 .cache/go/ 下，与系统/其他项目完全隔离

# 兼容 bash 与 zsh（zsh 没有 BASH_SOURCE）
if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
  _GO_ENV_SCRIPT="${BASH_SOURCE[0]}"
elif [[ -n "${ZSH_VERSION:-}" ]]; then
  _GO_ENV_SCRIPT="${(%):-%x}"
else
  _GO_ENV_SCRIPT="$0"
fi
_GO_ENV_ROOT="$(cd "$(dirname "${_GO_ENV_SCRIPT}")/.." && pwd)"

export GO111MODULE=on
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"
export GOPRIVATE="${GOPRIVATE:-}"

export GOMODCACHE="${_GO_ENV_ROOT}/.cache/go/pkg/mod"
export GOCACHE="${_GO_ENV_ROOT}/.cache/go/build"
export GOPATH="${_GO_ENV_ROOT}/.cache/go/gopath"

mkdir -p "${GOMODCACHE}" "${GOCACHE}" "${GOPATH}/bin"

# vendor/ 存在时强制只用项目内依赖，离线也能构建
if [[ -d "${_GO_ENV_ROOT}/vendor" ]]; then
  export GOFLAGS="${GOFLAGS:--mod=vendor}"
fi

export GO_ENV_ROOT="${_GO_ENV_ROOT}"
