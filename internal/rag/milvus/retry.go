package milvus

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Milvus Standalone 默认 gRPC 限流约 rate=0.1（每秒 0.1 次 ≈ 10 秒 1 次写入）。
// 批量导入或 kb add 过快时会报：
//
//	rate limit exceeded[rate=0.1]
//
// 这里统一做重试 + 退避，所有 Add 路径都受益。
const (
	milvusRetryMaxAttempts = 6
	milvusRetryBaseWait    = 10 * time.Second
)

func isMilvusRateLimitErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "rate limit") || strings.Contains(s, "ratelimiter")
}

// withMilvusRetry 执行 fn，遇到 Milvus 限流则等待后重试。
func withMilvusRetry(ctx context.Context, action string, fn func() error) error {
	var err error
	for attempt := 1; attempt <= milvusRetryMaxAttempts; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}
		if !isMilvusRateLimitErr(err) {
			return err
		}
		wait := time.Duration(attempt) * milvusRetryBaseWait
		fmt.Printf("⏳ Milvus 限流（%s），%v 后重试 (%d/%d)...\n", action, wait, attempt, milvusRetryMaxAttempts)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return fmt.Errorf("%s: %w", action, err)
}
