package cached

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zhjing1019/goAiAgent/internal/llm"
	"github.com/zhjing1019/goAiAgent/internal/store"
)

type memStore struct {
	msgs      map[string][]llm.Message
	sessions  map[string]bool
	loadCalls map[string]int
}

func (m *memStore) Migrate(context.Context) error { return nil }
func (m *memStore) CreateSession(_ context.Context, _ string) (string, error) {
	return "s1", nil
}
func (m *memStore) LoadMessages(_ context.Context, id string) ([]llm.Message, error) {
	if m.loadCalls != nil {
		m.loadCalls[id]++
	}
	return m.msgs[id], nil
}
func (m *memStore) AppendMessage(_ context.Context, id string, msg llm.Message) error {
	m.msgs[id] = append(m.msgs[id], msg)
	return nil
}
func (m *memStore) DeleteSession(_ context.Context, id string) error {
	delete(m.msgs, id)
	delete(m.sessions, id)
	return nil
}
func (m *memStore) ListSessions(context.Context, int) ([]store.Session, error) {
	return nil, nil
}
func (m *memStore) SessionExists(_ context.Context, id string) (bool, error) {
	return m.sessions[id], nil
}

func TestCachedLoadMessages(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	underlying := &memStore{
		msgs:     map[string][]llm.Message{"s1": {llm.NewUserMessage("hello")}},
		sessions: map[string]bool{"s1": true},
	}
	cached := New(underlying, rdb, time.Hour)

	ctx := context.Background()
	msgs, err := cached.LoadMessages(ctx, "s1")
	if err != nil || len(msgs) != 1 {
		t.Fatalf("first load: %v %v", msgs, err)
	}

	underlying.msgs["s1"] = nil
	msgs2, err := cached.LoadMessages(ctx, "s1")
	if err != nil || len(msgs2) != 1 {
		t.Fatalf("cache hit: %v %v", msgs2, err)
	}
}

func TestCachedInvalidateOnAppend(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	underlying := &memStore{
		msgs:     map[string][]llm.Message{"s1": {llm.NewUserMessage("v1")}},
		sessions: map[string]bool{"s1": true},
	}
	cached := New(underlying, rdb, time.Hour)
	ctx := context.Background()

	_, _ = cached.LoadMessages(ctx, "s1")
	if err := cached.AppendMessage(ctx, "s1", llm.NewUserMessage("v2")); err != nil {
		t.Fatal(err)
	}
	msgs, err := cached.LoadMessages(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 msgs after append, got %d", len(msgs))
	}
}

func TestNullCachePenetration(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	underlying := &memStore{
		msgs:      map[string][]llm.Message{},
		sessions:  map[string]bool{},
		loadCalls: map[string]int{},
	}
	cached := New(underlying, rdb, time.Hour)
	ctx := context.Background()
	fakeID := "00000000-0000-0000-0000-000000000000"

	_, err := cached.LoadMessages(ctx, fakeID)
	if err == nil {
		t.Fatal("expected session not found")
	}

	_, err = cached.LoadMessages(ctx, fakeID)
	if err == nil {
		t.Fatal("expected session not found on second call")
	}
	if underlying.loadCalls[fakeID] != 1 {
		t.Fatalf("expected 1 mysql load, got %d", underlying.loadCalls[fakeID])
	}

	val, err := rdb.Get(ctx, cached.msgKey(fakeID)).Result()
	if err != nil || val != nullCacheMarker {
		t.Fatalf("null cache: val=%q err=%v", val, err)
	}
}

func TestCacheTTLJitter(t *testing.T) {
	s := &Store{ttl: time.Hour}
	seen := make(map[time.Duration]bool)
	for i := 0; i < 50; i++ {
		seen[s.cacheTTL()] = true
	}
	if len(seen) < 2 {
		t.Fatal("expected TTL jitter to produce varied durations")
	}
}
