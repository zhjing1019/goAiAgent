package redisx

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func TestRateLimiter(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	lim := NewRateLimiter(rdb, 2)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		ok, err := lim.Allow(ctx, "127.0.0.1")
		if err != nil || !ok {
			t.Fatalf("request %d should pass: ok=%v err=%v", i+1, ok, err)
		}
	}
	ok, err := lim.Allow(ctx, "127.0.0.1")
	if err != nil || ok {
		t.Fatalf("third request should be blocked: ok=%v err=%v", ok, err)
	}
}
