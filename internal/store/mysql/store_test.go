package mysql

import (
	"context"
	"os"
	"testing"

	"github.com/zhjing1019/goAiAgent/internal/llm"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		t.Skip("MYSQL_DSN not set, skip integration test")
	}
	s, err := Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return s
}

func TestStoreSessionRoundTrip(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	id, err := s.CreateSession(ctx, "测试会话")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DeleteSession(ctx, id) })

	msgs := []llm.Message{
		llm.NewUserMessage("1+1=?"),
		llm.NewAssistantMessage("2"),
	}
	for _, m := range msgs {
		if err := s.AppendMessage(ctx, id, m); err != nil {
			t.Fatal(err)
		}
	}

	loaded, err := s.LoadMessages(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(loaded))
	}
	if loaded[0].Content != "1+1=?" || loaded[1].Content != "2" {
		t.Fatalf("unexpected messages: %+v", loaded)
	}

	list, err := s.ListSessions(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, sess := range list {
		if sess.ID == id {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("session not in list")
	}
}
