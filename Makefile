.PHONY: help gvm-install gvm-use env-init env-init-all deps-init deps-vendor deps-verify run-dev run-staging run-prod run-agent test build

GO       := bash scripts/with-go-env.sh go
GO_VERSION := $(shell cat .go-version 2>/dev/null || echo go1.26.3)
APP_ENV ?= development

help:
	@echo "Go AI Agent — 环境与依赖命令"
	@echo ""
	@echo "依赖隔离:"
	@echo "  make deps-init       初始化项目本地 Go 缓存 + 下载依赖"
	@echo "  make deps-vendor     生成 vendor/（完全离线可构建）"
	@echo "  make deps-verify     校验 go.sum 完整性"
	@echo ""
	@echo "Go 版本 (GVM):"
	@echo "  make gvm-install     安装 GVM + 项目 Go 版本 ($(GO_VERSION))"
	@echo "  make gvm-use         输出 GVM 激活命令（需 source）"
	@echo ""
	@echo "应用环境 (APP_ENV):"
	@echo "  make env-init        初始化 .env.development"
	@echo "  make env-init-all    初始化 dev/staging 环境文件"
	@echo "  make run-dev         APP_ENV=development 运行 agent-demo"
	@echo "  make run-staging     APP_ENV=staging 运行 agent-demo"
	@echo "  make run-prod        APP_ENV=production 运行 agent-demo"
	@echo ""
	@echo "构建与测试:"
	@echo "  make build           编译 agent-demo"
	@echo "  make test            运行全部测试"

deps-init:
	bash scripts/deps-init.sh

deps-vendor:
	bash scripts/deps-vendor.sh

deps-verify:
	$(GO) mod verify

gvm-install:
	bash scripts/install-gvm.sh

gvm-use:
	@echo 'source scripts/gvm-use.sh'

env-init:
	bash scripts/init-env.sh development

env-init-all:
	bash scripts/init-env.sh development
	bash scripts/init-env.sh staging

run-agent:
	APP_ENV=$(APP_ENV) $(GO) run ./cmd/agent-demo

run-dev:
	$(MAKE) run-agent APP_ENV=development

run-staging:
	$(MAKE) run-agent APP_ENV=staging

run-prod:
	$(MAKE) run-agent APP_ENV=production

build:
	$(GO) build -o bin/agent-demo ./cmd/agent-demo

test:
	$(GO) test ./...
