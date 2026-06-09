package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// AppEnv 返回当前运行环境，默认 development。
//
// 通过 APP_ENV 指定：development | staging | production
// 别名：dev → development，stage → staging，prod → production
func AppEnv() string {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	switch raw {
	case "", "development", "dev":
		return "development"
	case "staging", "stage":
		return "staging"
	case "production", "prod":
		return "production"
	default:
		return raw
	}
}

// LoadEnv 按企业级优先级加载环境变量文件。
//
// 加载顺序：
//  1. .env
//  2. .env.{APP_ENV}
//  3. .env.{APP_ENV}.local
//
// 规则：
//   - 后加载的文件覆盖先加载的同名字段（实现 dev/staging 隔离）
//   - 进程启动前已存在的系统环境变量优先级最高，不会被 .env 覆盖
//   - production 不加载任何 .env 文件，仅使用系统/容器注入的变量
func LoadEnv() {
	if AppEnv() == "production" {
		return
	}

	osEnv := snapshotEnvKeys()
	applyEnvFile(".env", osEnv)
	applyEnvFile(fmt.Sprintf(".env.%s", AppEnv()), osEnv)
	applyEnvFile(fmt.Sprintf(".env.%s.local", AppEnv()), osEnv)
}

// EnvSummary 启动时打印的环境摘要（不含密钥）。
func EnvSummary() string {
	return fmt.Sprintf("APP_ENV=%s", AppEnv())
}

func snapshotEnvKeys() map[string]bool {
	out := make(map[string]bool)
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		out[k] = true
	}
	return out
}

func applyEnvFile(path string, osEnv map[string]bool) {
	m, err := godotenv.Read(path)
	if err != nil {
		return
	}
	for k, v := range m {
		if osEnv[k] {
			continue
		}
		os.Setenv(k, v)
	}
}
