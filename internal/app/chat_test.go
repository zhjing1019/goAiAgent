package app

import (
	"context"
	"strings"
	"testing"
)

func TestRunChatEmptyMessage(t *testing.T) {
	a := &App{agent: nil}
	_, err := a.RunChat(context.Background(), "", "")
	if err == nil || !strings.Contains(err.Error(), "message") {
		t.Fatalf("expected message error, got %v", err)
	}
}

func TestRunChatSessionIDWithoutMySQL(t *testing.T) {
	a := &App{}
	_, err := a.RunChat(context.Background(), "some-id", "hello")
	if err == nil || !strings.Contains(err.Error(), "MySQL") {
		t.Fatalf("expected mysql error, got %v", err)
	}
}
